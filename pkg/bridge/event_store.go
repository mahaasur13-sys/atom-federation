package bridge

import (
	"sync"
)

// EventStore — DESC event log (append-only, immutable events).
type EventStore struct {
	mu     sync.RWMutex
	events []Event
	path   string
}

// Event — immutable DESC event record.
type Event struct {
	Tick    uint64  `json:"tick"`
	Type    string  `json:"type"`
	AgentID string  `json:"agent_id,omitempty"`
	Domain  string  `json:"domain,omitempty"`
	TaskID  string  `json:"task_id,omitempty"`
	Success bool    `json:"success,omitempty"`
	Reward  float64 `json:"reward,omitempty"`
	Mu      float64 `json:"mu,omitempty"`
	Seed    uint64  `json:"seed,omitempty"`
}

// NewEventStore — create event store (in-memory + optional file persistence).
func NewEventStore(path string) *EventStore {
	return &EventStore{path: path}
}

// Record — append immutable event.
func (es *EventStore) Record(e Event) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.events = append(es.events, e)
}

// All — return all events (for replay).
func (es *EventStore) All() []Event {
	es.mu.RLock()
	defer es.mu.RUnlock()
	out := make([]Event, len(es.events))
	copy(out, es.events)
	return out
}

// Len — event count.
func (es *EventStore) Len() int {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return len(es.events)
}
