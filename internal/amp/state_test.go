package amp

import (
	"log/slog"
	"os"
	"testing"
)

func newTestSM() *StateMachine {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	events := NewEventLog(10)
	return NewStateMachine(logger, events)
}

func TestValidTransitions(t *testing.T) {
	tests := []struct {
		from State
		to   State
		ok   bool
	}{
		{Disconnected, Probing, true},
		{Disconnected, Ready, false},
		{Probing, Negotiating, true},
		{Probing, Disconnected, true},
		{Probing, Ready, true},
		{Negotiating, Ready, true},
		{Negotiating, Probing, true},
		{Negotiating, Disconnected, false},
		{Ready, Recovering, true},
		{Ready, Disconnected, false},
		{Ready, Probing, false},
		{Recovering, Probing, true},
		{Recovering, Ready, false},
	}
	for _, tt := range tests {
		if got := validTransition(tt.from, tt.to); got != tt.ok {
			t.Errorf("validTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.ok)
		}
	}
}

func TestStateMachineTransition(t *testing.T) {
	sm := newTestSM()

	if sm.Current() != Disconnected {
		t.Fatalf("initial state = %s, want DISCONNECTED", sm.Current())
	}

	if err := sm.Transition(Probing, "startup"); err != nil {
		t.Fatalf("Disconnected -> Probing: %v", err)
	}
	if sm.Current() != Probing {
		t.Fatalf("state = %s, want PROBING", sm.Current())
	}

	if err := sm.Transition(Negotiating, "found at 9600"); err != nil {
		t.Fatalf("Probing -> Negotiating: %v", err)
	}

	if err := sm.Transition(Ready, "reached 115200"); err != nil {
		t.Fatalf("Negotiating -> Ready: %v", err)
	}
	if !sm.IsReady() {
		t.Fatal("expected IsReady() = true")
	}

	if err := sm.Transition(Recovering, "timeout"); err != nil {
		t.Fatalf("Ready -> Recovering: %v", err)
	}

	if err := sm.Transition(Probing, "retry"); err != nil {
		t.Fatalf("Recovering -> Probing: %v", err)
	}
}

func TestInvalidTransition(t *testing.T) {
	sm := newTestSM()
	if err := sm.Transition(Ready, "skip"); err == nil {
		t.Error("expected error for Disconnected -> Ready")
	}
}

func TestOnReadyCallback(t *testing.T) {
	sm := newTestSM()
	called := false
	sm.OnReady(func() { called = true })

	sm.Transition(Probing, "start")
	sm.Transition(Negotiating, "found")
	sm.Transition(Ready, "done")

	if !called {
		t.Error("OnReady callback not called")
	}
}

func TestStateString(t *testing.T) {
	if s := Disconnected.String(); s != "DISCONNECTED" {
		t.Errorf("Disconnected.String() = %q", s)
	}
	if s := Ready.String(); s != "READY" {
		t.Errorf("Ready.String() = %q", s)
	}
}
