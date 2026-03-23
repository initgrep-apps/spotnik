package state

import (
	"sync"
	"time"
)

const netLogCapacity = 200

// NetLogEntry records a single API call for the network log panel.
type NetLogEntry struct {
	Timestamp  time.Time
	Method     string
	Path       string
	StatusCode int
	DurationMs int64
}

// NetLog is a fixed-size ring buffer of NetLogEntry values.
// It is safe for concurrent use.
type NetLog struct {
	mu      sync.RWMutex
	entries []NetLogEntry
	head    int // next write position
	count   int // number of entries stored (max netLogCapacity)
}

// NewNetLog creates a new NetLog with the default capacity.
func NewNetLog() *NetLog {
	return &NetLog{
		entries: make([]NetLogEntry, netLogCapacity),
	}
}

// Add appends an entry to the ring buffer, overwriting the oldest if full.
func (nl *NetLog) Add(entry NetLogEntry) {
	nl.mu.Lock()
	defer nl.mu.Unlock()
	nl.entries[nl.head] = entry
	nl.head = (nl.head + 1) % netLogCapacity
	if nl.count < netLogCapacity {
		nl.count++
	}
}

// Entries returns all stored entries in oldest-first order.
func (nl *NetLog) Entries() []NetLogEntry {
	nl.mu.RLock()
	defer nl.mu.RUnlock()

	if nl.count == 0 {
		return nil
	}

	result := make([]NetLogEntry, nl.count)
	if nl.count < netLogCapacity {
		// Buffer not full — entries start at 0.
		copy(result, nl.entries[:nl.count])
	} else {
		// Buffer full — oldest is at head, wrap around.
		n := copy(result, nl.entries[nl.head:])
		copy(result[n:], nl.entries[:nl.head])
	}
	return result
}

// Len returns the number of entries currently stored.
func (nl *NetLog) Len() int {
	nl.mu.RLock()
	defer nl.mu.RUnlock()
	return nl.count
}
