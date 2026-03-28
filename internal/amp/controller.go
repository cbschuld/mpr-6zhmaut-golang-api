package amp

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/config"
	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/model"
	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/serial"
)

// Controller orchestrates the amp connection lifecycle.
type Controller struct {
	cfg    *config.Config
	port   *serial.Port
	queue  *serial.Queue
	state  *StateMachine
	cache  *ZoneCache
	events *EventLog
	logger *slog.Logger

	consecutiveErrors atomic.Int32
	totalRecoveries   atomic.Int64
	lastRecovery      atomic.Value // time.Time
	lastRecoveryMs    atomic.Int64
	lastRecoveryWhy   atomic.Value // string
}

// NewController creates a new amp controller.
func NewController(cfg *config.Config, port *serial.Port, queue *serial.Queue, state *StateMachine, cache *ZoneCache, events *EventLog, logger *slog.Logger) *Controller {
	c := &Controller{
		cfg:    cfg,
		port:   port,
		queue:  queue,
		state:  state,
		cache:  cache,
		events: events,
		logger: logger,
	}
	c.lastRecoveryWhy.Store("")
	c.lastRecovery.Store(time.Time{})
	return c
}

// Start initiates the connection lifecycle (probe -> negotiate -> ready).
func (c *Controller) Start(ctx context.Context) error {
	return c.connect(ctx)
}

// TriggerRecovery transitions to recovering and attempts to reconnect.
func (c *Controller) TriggerRecovery(ctx context.Context, reason string) {
	if c.state.Current() == Recovering || c.state.Current() == Probing {
		return
	}

	c.logger.Warn("recovery triggered", "reason", reason)
	c.events.Add("recovery_started", reason)

	if err := c.state.Transition(Recovering, reason); err != nil {
		c.logger.Error("failed to transition to recovering", "error", err)
		return
	}

	c.totalRecoveries.Add(1)
	c.lastRecoveryWhy.Store(reason)
	start := time.Now()

	// Stop the command queue so its goroutines don't compete with the probe
	// for serial port reads
	c.queue.Stop()

	if err := c.state.Transition(Probing, "recovery initiated"); err != nil {
		c.logger.Error("failed to transition to probing", "error", err)
		return
	}

	go func() {
		if err := c.connect(ctx); err != nil {
			c.logger.Error("recovery failed", "error", err)
			// Restart the queue even on failure so future attempts work
			c.queue.Start(ctx)
			return
		}
		// Restart the queue now that we're connected
		c.queue.Start(ctx)
		c.consecutiveErrors.Store(0)
		elapsed := time.Since(start)
		c.lastRecovery.Store(time.Now())
		c.lastRecoveryMs.Store(elapsed.Milliseconds())
		c.events.Add("recovery_complete", fmt.Sprintf("recovered in %dms", elapsed.Milliseconds()))
		c.logger.Info("recovery completed", "duration_ms", elapsed.Milliseconds())
	}()
}

// RecordCommandSuccess resets the consecutive error counter.
func (c *Controller) RecordCommandSuccess() {
	c.consecutiveErrors.Store(0)
}

// RecordCommandError increments the consecutive error counter
// and triggers recovery if threshold is reached.
func (c *Controller) RecordCommandError(ctx context.Context, err error) {
	count := c.consecutiveErrors.Add(1)
	if count >= 3 {
		c.TriggerRecovery(ctx, fmt.Sprintf("consecutive errors: %d, last: %v", count, err))
	}
}

// QueryAllZones queries all configured amplifiers and returns zone data.
func (c *Controller) QueryAllZones(ctx context.Context) ([]model.Zone, error) {
	if !c.state.IsReady() {
		return nil, serial.ErrRecovering
	}

	var allZones []model.Zone
	for i := 1; i <= c.cfg.AmpCount; i++ {
		cmd := serial.QueryCommand(i)
		result, err := c.queue.Enqueue(ctx, cmd, 6, c.cfg.CmdTimeout)
		if err != nil {
			c.RecordCommandError(ctx, err)
			return nil, err
		}
		c.RecordCommandSuccess()
		allZones = append(allZones, result.Zones...)
	}
	return allZones, nil
}

// QueryZone queries a single zone by ID.
func (c *Controller) QueryZone(ctx context.Context, zoneID string) (*model.Zone, error) {
	if !c.state.IsReady() {
		return nil, serial.ErrRecovering
	}

	ampID := serial.AmpIDForZone(zoneID)
	cmd := serial.QueryCommand(ampID)
	result, err := c.queue.Enqueue(ctx, cmd, 6, c.cfg.CmdTimeout)
	if err != nil {
		c.RecordCommandError(ctx, err)
		return nil, err
	}
	c.RecordCommandSuccess()

	for _, z := range result.Zones {
		if z.Zone == zoneID {
			return &z, nil
		}
	}
	return nil, fmt.Errorf("zone %s not found in response", zoneID)
}

// SetAttribute sets a zone attribute and returns the expected new state (optimistic).
func (c *Controller) SetAttribute(ctx context.Context, zoneID, attr, value string) (*model.Zone, error) {
	if !c.state.IsReady() {
		return nil, serial.ErrRecovering
	}

	cmd := serial.ControlCommand(zoneID, attr, value)
	_, err := c.queue.Enqueue(ctx, cmd, 1, c.cfg.CmdTimeout)
	if err != nil {
		c.RecordCommandError(ctx, err)
		return nil, err
	}
	c.RecordCommandSuccess()

	// Optimistic update
	c.cache.OptimisticSet(zoneID, attr, value)
	z, ok := c.cache.Get(zoneID)
	if !ok {
		return nil, fmt.Errorf("zone %s not in cache", zoneID)
	}
	return &z, nil
}

// RecoveryStats returns recovery-related statistics for the health endpoint.
type RecoveryStats struct {
	TotalRecoveries    int64     `json:"total_recoveries"`
	LastRecovery       time.Time `json:"last_recovery,omitempty"`
	LastRecoveryReason string    `json:"last_recovery_reason,omitempty"`
	LastRecoveryMs     int64     `json:"last_recovery_duration_ms,omitempty"`
	ConsecutiveErrors  int32     `json:"consecutive_errors"`
}

func (c *Controller) GetRecoveryStats() RecoveryStats {
	lr, _ := c.lastRecovery.Load().(time.Time)
	lrw, _ := c.lastRecoveryWhy.Load().(string)
	return RecoveryStats{
		TotalRecoveries:    c.totalRecoveries.Load(),
		LastRecovery:       lr,
		LastRecoveryReason: lrw,
		LastRecoveryMs:     c.lastRecoveryMs.Load(),
		ConsecutiveErrors:  c.consecutiveErrors.Load(),
	}
}

func (c *Controller) connect(ctx context.Context) error {
	currentBaud := c.port.BaudRate()
	if currentBaud == 0 {
		currentBaud = c.cfg.TargetBaudRate
	}

	// If already in Probing state (from recovery), skip the transition
	if c.state.Current() != Probing {
		if err := c.state.Transition(Probing, "starting connection"); err != nil {
			return err
		}
	}

	// Try to find the amp at some baud rate
	foundBaud, err := c.probe(ctx, currentBaud)
	if err != nil {
		c.state.Transition(Disconnected, fmt.Sprintf("probe failed: %v", err))
		return err
	}

	// If already at target, go straight to Ready
	if foundBaud == c.cfg.TargetBaudRate {
		return c.state.Transition(Ready, fmt.Sprintf("connected at target baud rate %d", foundBaud))
	}

	// Step up to target baud rate
	if err := c.state.Transition(Negotiating, fmt.Sprintf("found at %d, stepping up to %d", foundBaud, c.cfg.TargetBaudRate)); err != nil {
		return err
	}

	if err := c.stepUp(ctx, foundBaud); err != nil {
		c.logger.Error("baud step-up failed", "error", err)
		c.state.Transition(Probing, "step-up failed")
		return c.connect(ctx) // retry from probing
	}

	return c.state.Transition(Ready, fmt.Sprintf("reached target baud rate %d", c.cfg.TargetBaudRate))
}

func (c *Controller) probe(ctx context.Context, startBaud int) (int, error) {
	// Order: try target first, then 9600, then everything else
	tryOrder := []int{startBaud, 9600}
	for _, rate := range config.ValidBaudRates {
		if rate != startBaud && rate != 9600 {
			tryOrder = append(tryOrder, rate)
		}
	}
	// Deduplicate
	seen := make(map[int]bool)
	var unique []int
	for _, r := range tryOrder {
		if !seen[r] {
			seen[r] = true
			unique = append(unique, r)
		}
	}

	const maxAttempts = 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		for _, baud := range unique {
			c.logger.Info("probing baud rate", "baud_rate", baud, "attempt", attempt+1)
			c.events.Add("baud_probe", fmt.Sprintf("trying %d (attempt %d)", baud, attempt+1))

			if err := c.port.Open(baud); err != nil {
				c.logger.Warn("failed to open port", "baud_rate", baud, "error", err)
				continue
			}

			// Try querying amp 1
			_, cancel := context.WithTimeout(ctx, c.cfg.CmdTimeout)
			if err := c.port.Write(serial.QueryCommand(1)); err != nil {
				cancel()
				continue
			}

			// Read lines until we get a zone response or timeout
			success := false
			deadline := time.After(c.cfg.CmdTimeout)
		readLoop:
			for {
				lineCh := make(chan string, 1)
				errCh := make(chan error, 1)
				go func() {
					line, err := c.port.ReadLine()
					if err != nil {
						errCh <- err
						return
					}
					lineCh <- line
				}()

				select {
				case <-deadline:
					break readLoop
				case <-errCh:
					break readLoop
				case line := <-lineCh:
					if zone := serial.ParseZoneResponse(line); zone != nil {
						success = true
						break readLoop
					}
				}
			}
			cancel()

			if success {
				c.logger.Info("probe success", "baud_rate", baud)
				c.events.Add("baud_probe", fmt.Sprintf("success at %d", baud))
				return baud, nil
			}

			c.port.Close()
		}

		if attempt < maxAttempts-1 {
			c.logger.Info("probe cycle failed, retrying", "wait", "5s")
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}
	}

	return 0, fmt.Errorf("failed to find amp at any baud rate after %d attempts", maxAttempts)
}

func (c *Controller) stepUp(ctx context.Context, currentBaud int) error {
	steps := serial.BaudRateSteps(currentBaud, c.cfg.TargetBaudRate)
	totalSteps := len(steps)

	for i, nextBaud := range steps {
		c.logger.Info("baud step-up", "from_baud", currentBaud, "to_baud", nextBaud, "step_num", i+1, "total_steps", totalSteps)
		c.events.Add("baud_stepup", fmt.Sprintf("%d -> %d", currentBaud, nextBaud))

		// Send baud rate change command at current rate
		cmd := serial.BaudRateCommand(nextBaud)
		if err := c.port.Write(cmd); err != nil {
			return fmt.Errorf("write baud command: %w", err)
		}

		// Wait for the amp to apply the change
		select {
		case <-time.After(c.cfg.StepDelay):
		case <-ctx.Done():
			return ctx.Err()
		}

		// Switch our port to the new baud rate
		if err := c.port.SetBaudRate(nextBaud); err != nil {
			return fmt.Errorf("set baud rate to %d: %w", nextBaud, err)
		}

		// Verify we can still talk to the amp
		if err := c.port.Write(serial.QueryCommand(1)); err != nil {
			return fmt.Errorf("verify write at %d: %w", nextBaud, err)
		}

		verified := false
		deadline := time.After(c.cfg.CmdTimeout)
	verifyLoop:
		for {
			lineCh := make(chan string, 1)
			errCh := make(chan error, 1)
			go func() {
				line, err := c.port.ReadLine()
				if err != nil {
					errCh <- err
					return
				}
				lineCh <- line
			}()

			select {
			case <-deadline:
				break verifyLoop
			case <-errCh:
				break verifyLoop
			case line := <-lineCh:
				if zone := serial.ParseZoneResponse(line); zone != nil {
					verified = true
					break verifyLoop
				}
			}
		}

		if !verified {
			return fmt.Errorf("verification failed at %d baud", nextBaud)
		}

		currentBaud = nextBaud
	}

	return nil
}
