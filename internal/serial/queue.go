package serial

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/model"
)

var (
	ErrTimeout    = errors.New("command timeout")
	ErrCmdError   = errors.New("command error from amp")
	ErrQueueFull  = errors.New("command queue full")
	ErrRecovering = errors.New("amp connection recovering")
	ErrShutdown   = errors.New("queue shutting down")
)

// CommandResult holds the response(s) from a serial command.
type CommandResult struct {
	Zones []model.Zone
	Raw   []string
}

type command struct {
	data              string
	expectedResponses int
	timeout           time.Duration
	resultCh          chan CommandResult
	errCh             chan error
}

// Stats tracks queue-level counters exposed via /health.
type Stats struct {
	TotalCommands int64
	TotalTimeouts int64
	TotalErrors   int64
	Pending       int
}

// Queue serializes access to the serial port.
type Queue struct {
	port      *Port
	logger    *slog.Logger
	cmdCh     chan command
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	stats     Stats
	statsLock sync.RWMutex

	totalCommands atomic.Int64
	totalTimeouts atomic.Int64
	totalErrors   atomic.Int64

	mu        sync.Mutex
	pendingMu sync.Mutex
	pending   int
}

// NewQueue creates a command queue backed by the given serial port.
func NewQueue(port *Port, logger *slog.Logger) *Queue {
	return &Queue{
		port:   port,
		logger: logger,
		cmdCh:  make(chan command, 64),
	}
}

// Start begins processing commands. Call Stop to shut down.
func (q *Queue) Start(ctx context.Context) {
	ctx, q.cancel = context.WithCancel(ctx)
	q.wg.Add(1)
	go q.processLoop(ctx)
}

// Stop gracefully shuts down the queue.
func (q *Queue) Stop() {
	if q.cancel != nil {
		q.cancel()
	}
	q.wg.Wait()
}

// Enqueue submits a command and waits for the result.
func (q *Queue) Enqueue(ctx context.Context, data string, expectedResponses int, timeout time.Duration) (CommandResult, error) {
	cmd := command{
		data:              data,
		expectedResponses: expectedResponses,
		timeout:           timeout,
		resultCh:          make(chan CommandResult, 1),
		errCh:             make(chan error, 1),
	}

	q.pendingMu.Lock()
	q.pending++
	q.pendingMu.Unlock()

	defer func() {
		q.pendingMu.Lock()
		q.pending--
		q.pendingMu.Unlock()
	}()

	select {
	case q.cmdCh <- cmd:
	case <-ctx.Done():
		return CommandResult{}, ctx.Err()
	}

	select {
	case result := <-cmd.resultCh:
		return result, nil
	case err := <-cmd.errCh:
		return CommandResult{}, err
	case <-ctx.Done():
		return CommandResult{}, ctx.Err()
	}
}

// GetStats returns current queue statistics.
func (q *Queue) GetStats() Stats {
	q.pendingMu.Lock()
	pending := q.pending
	q.pendingMu.Unlock()

	return Stats{
		TotalCommands: q.totalCommands.Load(),
		TotalTimeouts: q.totalTimeouts.Load(),
		TotalErrors:   q.totalErrors.Load(),
		Pending:       pending,
	}
}

func (q *Queue) processLoop(ctx context.Context) {
	defer q.wg.Done()

	for {
		select {
		case <-ctx.Done():
			q.drainQueue(ErrShutdown)
			return
		case cmd := <-q.cmdCh:
			q.executeCommand(ctx, cmd)
		}
	}
}

func (q *Queue) executeCommand(ctx context.Context, cmd command) {
	q.totalCommands.Add(1)
	sanitized := strings.TrimSpace(cmd.data)
	q.logger.Debug("command sent", "cmd", sanitized, "timeout", cmd.timeout)

	start := time.Now()

	if err := q.port.Write(cmd.data); err != nil {
		q.totalErrors.Add(1)
		q.logger.Error("serial write failed", "cmd", sanitized, "error", err)
		cmd.errCh <- fmt.Errorf("serial write: %w", err)
		return
	}

	var result CommandResult
	timer := time.NewTimer(cmd.timeout)
	defer timer.Stop()

	for len(result.Zones) < cmd.expectedResponses {
		lineCh := make(chan string, 1)
		errCh := make(chan error, 1)

		go func() {
			line, err := q.port.ReadLine()
			if err != nil {
				errCh <- err
				return
			}
			lineCh <- line
		}()

		select {
		case <-timer.C:
			q.totalTimeouts.Add(1)
			elapsed := time.Since(start)
			q.logger.Warn("command timeout", "cmd", sanitized, "timeout_ms", elapsed.Milliseconds())
			cmd.errCh <- ErrTimeout
			return
		case err := <-errCh:
			q.totalErrors.Add(1)
			q.logger.Error("serial read failed", "cmd", sanitized, "error", err)
			cmd.errCh <- fmt.Errorf("serial read: %w", err)
			return
		case line := <-lineCh:
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			result.Raw = append(result.Raw, line)

			if IsErrorResponse(line) {
				q.totalErrors.Add(1)
				q.logger.Error("command error from amp", "cmd", sanitized, "raw_response", line)
				cmd.errCh <- ErrCmdError
				return
			}

			if zone := ParseZoneResponse(line); zone != nil {
				result.Zones = append(result.Zones, *zone)
				elapsed := time.Since(start)
				q.logger.Debug("response received", "zone", zone.Zone, "response_time_ms", elapsed.Milliseconds())
			}
		case <-ctx.Done():
			cmd.errCh <- ctx.Err()
			return
		}
	}

	cmd.resultCh <- result
}

func (q *Queue) drainQueue(err error) {
	for {
		select {
		case cmd := <-q.cmdCh:
			cmd.errCh <- err
		default:
			return
		}
	}
}
