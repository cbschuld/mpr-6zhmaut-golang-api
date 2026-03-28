package amp

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// State represents the connection lifecycle state.
type State int

const (
	Disconnected State = iota
	Probing
	Negotiating
	Ready
	Recovering
)

func (s State) String() string {
	switch s {
	case Disconnected:
		return "DISCONNECTED"
	case Probing:
		return "PROBING"
	case Negotiating:
		return "NEGOTIATING"
	case Ready:
		return "READY"
	case Recovering:
		return "RECOVERING"
	default:
		return "UNKNOWN"
	}
}

// StateMachine manages connection lifecycle transitions.
type StateMachine struct {
	mu             sync.RWMutex
	state          State
	lastTransition time.Time
	logger         *slog.Logger
	events         *EventLog
	onReady        func()
	onRecovering   func()
}

// NewStateMachine creates a state machine starting in Disconnected.
func NewStateMachine(logger *slog.Logger, events *EventLog) *StateMachine {
	return &StateMachine{
		state:          Disconnected,
		lastTransition: time.Now(),
		logger:         logger,
		events:         events,
	}
}

// OnReady sets a callback for when the state transitions to Ready.
func (sm *StateMachine) OnReady(fn func()) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onReady = fn
}

// OnRecovering sets a callback for when the state transitions to Recovering.
func (sm *StateMachine) OnRecovering(fn func()) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onRecovering = fn
}

// Transition attempts a state change. Returns an error if the transition is invalid.
func (sm *StateMachine) Transition(to State, reason string) error {
	sm.mu.Lock()
	from := sm.state
	if !validTransition(from, to) {
		sm.mu.Unlock()
		return fmt.Errorf("invalid transition %s -> %s", from, to)
	}
	sm.state = to
	sm.lastTransition = time.Now()
	onReady := sm.onReady
	onRecovering := sm.onRecovering
	sm.mu.Unlock()

	sm.logger.Info("state transition", "from", from.String(), "to", to.String(), "reason", reason)
	sm.events.Add("state_change", fmt.Sprintf("%s -> %s: %s", from, to, reason))

	if to == Ready && onReady != nil {
		onReady()
	}
	if to == Recovering && onRecovering != nil {
		onRecovering()
	}

	return nil
}

// Current returns the current state.
func (sm *StateMachine) Current() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// IsReady returns true if the state machine is in the Ready state.
func (sm *StateMachine) IsReady() bool {
	return sm.Current() == Ready
}

// LastTransition returns the time of the last state change.
func (sm *StateMachine) LastTransition() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.lastTransition
}

// TimeInState returns how long the state machine has been in the current state.
func (sm *StateMachine) TimeInState() time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return time.Since(sm.lastTransition)
}

func validTransition(from, to State) bool {
	switch from {
	case Disconnected:
		return to == Probing
	case Probing:
		return to == Negotiating || to == Disconnected
	case Negotiating:
		return to == Ready || to == Probing
	case Ready:
		return to == Recovering
	case Recovering:
		return to == Probing
	default:
		return false
	}
}
