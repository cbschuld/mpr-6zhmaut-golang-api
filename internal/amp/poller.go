package amp

import (
	"context"
	"log/slog"
	"time"

	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/model"
	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/serial"
)

// Poller periodically queries all zones and updates the cache.
type Poller struct {
	controller *Controller
	queue      *serial.Queue
	state      *StateMachine
	cache      *ZoneCache
	logger     *slog.Logger
	interval   time.Duration
	cmdTimeout time.Duration
	ampCount   int
}

// NewPoller creates a background zone poller.
func NewPoller(controller *Controller, queue *serial.Queue, state *StateMachine, cache *ZoneCache, interval, cmdTimeout time.Duration, ampCount int, logger *slog.Logger) *Poller {
	return &Poller{
		controller: controller,
		queue:      queue,
		state:      state,
		cache:      cache,
		logger:     logger,
		interval:   interval,
		cmdTimeout: cmdTimeout,
		ampCount:   ampCount,
	}
}

// Start begins the polling loop.
func (p *Poller) Start(ctx context.Context) {
	go p.loop(ctx)
}

func (p *Poller) loop(ctx context.Context) {
	// Do an initial poll immediately when ready
	if p.state.IsReady() {
		p.poll(ctx)
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !p.state.IsReady() {
				continue
			}
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	start := time.Now()
	var allZones []model.Zone

	for i := 1; i <= p.ampCount; i++ {
		pollCtx, cancel := context.WithTimeout(ctx, p.cmdTimeout)
		cmd := serial.QueryCommand(i)
		result, err := p.queue.Enqueue(pollCtx, cmd, 6, p.cmdTimeout)
		cancel()

		if err != nil {
			p.logger.Warn("poll failed", "amp", i, "error", err)
			p.controller.RecordCommandError(ctx, err)
			return
		}
		p.controller.RecordCommandSuccess()
		allZones = append(allZones, result.Zones...)
	}

	changes := p.cache.Update(allZones)
	elapsed := time.Since(start)

	p.logger.Debug("poll cycle", "zones_updated", len(allZones), "duration_ms", elapsed.Milliseconds())

	for _, ch := range changes {
		p.logger.Info("keypad change detected",
			"zone", ch.ZoneID,
			"attr", ch.Attr,
			"old_value", ch.OldValue,
			"new_value", ch.NewValue,
		)
	}
}
