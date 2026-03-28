package amp

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/serial"
)

// HealthMonitor periodically checks that the amp connection is alive.
type HealthMonitor struct {
	controller *Controller
	queue      *serial.Queue
	state      *StateMachine
	logger     *slog.Logger
	interval   time.Duration
	cmdTimeout time.Duration

	lastCheck   atomic.Value // time.Time
	lastCheckOk atomic.Bool
	mu          sync.Mutex
}

// NewHealthMonitor creates a health monitor.
func NewHealthMonitor(controller *Controller, queue *serial.Queue, state *StateMachine, interval, cmdTimeout time.Duration, logger *slog.Logger) *HealthMonitor {
	hm := &HealthMonitor{
		controller: controller,
		queue:      queue,
		state:      state,
		logger:     logger,
		interval:   interval,
		cmdTimeout: cmdTimeout,
	}
	hm.lastCheck.Store(time.Time{})
	hm.lastCheckOk.Store(true)
	return hm
}

// Start begins periodic health checks.
func (hm *HealthMonitor) Start(ctx context.Context) {
	go hm.loop(ctx)
}

// LastCheck returns the time and result of the last health check.
func (hm *HealthMonitor) LastCheck() (time.Time, bool) {
	t, _ := hm.lastCheck.Load().(time.Time)
	return t, hm.lastCheckOk.Load()
}

// Interval returns the health check interval.
func (hm *HealthMonitor) Interval() time.Duration {
	return hm.interval
}

func (hm *HealthMonitor) loop(ctx context.Context) {
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !hm.state.IsReady() {
				continue
			}
			hm.check(ctx)
		}
	}
}

func (hm *HealthMonitor) check(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, hm.cmdTimeout)
	defer cancel()

	cmd := serial.QueryCommand(1)
	_, err := hm.queue.Enqueue(checkCtx, cmd, 6, hm.cmdTimeout)

	hm.lastCheck.Store(time.Now())

	if err != nil {
		hm.lastCheckOk.Store(false)
		hm.logger.Warn("health check failed", "error", err)
		hm.controller.TriggerRecovery(ctx, "health check failed: "+err.Error())
		return
	}

	hm.lastCheckOk.Store(true)
}
