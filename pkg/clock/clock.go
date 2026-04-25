// Package clock — LogicalClock (Lamport-style) for deterministic message ordering.
// RL-022: All cross-cluster messages carry LogicalClock timestamps.
// No time.time(), no uuid.uuid4() — only tick-based timestamps.
package clock

import "sync"

// LogicalClock implements Lamport logical clocks for cross-cluster ordering.
// Every message increments the clock. Recipients max their clock before delivery.
// This guarantees: messages from the same causal chain arrive in identical order on all nodes.
type LogicalClock struct {
	tick    int64
	nodeID  string
	mu      sync.Mutex
}

func NewLogicalClock(nodeID string) *LogicalClock {
	return &LogicalClock{tick: 0, nodeID: nodeID}
}

// Tick returns the current logical tick.
func (c *LogicalClock) Tick() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tick
}

// Send returns a timestamp for a message being sent at the current tick.
func (c *LogicalClock) Send() (tick int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tick++
	return c.tick
}

// Recv advances the clock to max(current, msgTick)+1 and returns the new tick.
// Called upon message receipt. Guarantees causal ordering.
func (c *LogicalClock) Recv(msgTick int64) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if msgTick > c.tick {
		c.tick = msgTick
	}
	c.tick++
	return c.tick
}

// SetTick forces the tick (used for replay state restore).
func (c *LogicalClock) SetTick(t int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tick = t
}

// Compare returns -1 if a < b, 1 if a > b, 0 if equal.
func Compare(a, b ClockMessage) int {
	if a.Tick != b.Tick {
		if a.Tick < b.Tick {
			return -1
		}
		return 1
	}
	if a.NodeID != b.NodeID {
		if a.NodeID < b.NodeID {
			return -1
		}
		return 1
	}
	return 0
}

type ClockMessage struct {
	Tick   int64
	NodeID string
}
