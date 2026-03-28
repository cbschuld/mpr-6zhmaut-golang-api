package amp

import (
	"sync"
	"time"
)

// Event represents a significant system event for the ring buffer.
type Event struct {
	Time   time.Time `json:"time"`
	Type   string    `json:"type"`
	Detail string    `json:"detail"`
}

// EventLog is a thread-safe ring buffer of recent events.
type EventLog struct {
	mu     sync.RWMutex
	events []Event
	size   int
	pos    int
	count  int
}

// NewEventLog creates a ring buffer that holds the last n events.
func NewEventLog(size int) *EventLog {
	return &EventLog{
		events: make([]Event, size),
		size:   size,
	}
}

// Add records a new event.
func (el *EventLog) Add(eventType, detail string) {
	el.mu.Lock()
	defer el.mu.Unlock()
	el.events[el.pos] = Event{
		Time:   time.Now(),
		Type:   eventType,
		Detail: detail,
	}
	el.pos = (el.pos + 1) % el.size
	if el.count < el.size {
		el.count++
	}
}

// All returns all events in chronological order.
func (el *EventLog) All() []Event {
	el.mu.RLock()
	defer el.mu.RUnlock()

	result := make([]Event, 0, el.count)
	if el.count < el.size {
		result = append(result, el.events[:el.count]...)
	} else {
		result = append(result, el.events[el.pos:]...)
		result = append(result, el.events[:el.pos]...)
	}
	return result
}
