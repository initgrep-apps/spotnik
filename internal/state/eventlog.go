package state

import (
	"sync"

	"github.com/initgrep-apps/spotnik/internal/domain"
)

const defaultEventLogCapacity = 500

// GatewayEventLog is a fixed-size ring buffer of GatewayEvent values with
// cursor-based reads. Multiple consumers can independently track their
// position using monotonically increasing sequence numbers.
//
// Thread-safe: Add() takes a write lock, ReadFrom() takes a read lock.
type GatewayEventLog struct {
	mu       sync.RWMutex
	entries  []domain.GatewayEvent
	head     int    // next write position in the ring
	count    int    // entries stored (max capacity)
	sequence uint64 // monotonically increasing, incremented on each Add()
}

// NewGatewayEventLog creates an event log with the given capacity.
// If capacity is <= 0 it defaults to defaultEventLogCapacity (500).
func NewGatewayEventLog(capacity int) *GatewayEventLog {
	if capacity <= 0 {
		capacity = defaultEventLogCapacity
	}
	return &GatewayEventLog{
		entries: make([]domain.GatewayEvent, capacity),
	}
}

// Add appends an event to the ring buffer, overwriting the oldest if full.
func (l *GatewayEventLog) Add(event domain.GatewayEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries[l.head] = event
	l.head = (l.head + 1) % len(l.entries)
	if l.count < len(l.entries) {
		l.count++
	}
	l.sequence++
}

// ReadFrom returns events added since the given cursor position.
// Returns the new cursor and the slice of new events.
// First call should use cursor=0.
//
// If the cursor is older than the oldest retained event (due to ring buffer
// wraparound), all currently stored events are returned.
func (l *GatewayEventLog) ReadFrom(cursor uint64) (uint64, []domain.GatewayEvent) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.count == 0 || cursor >= l.sequence {
		return l.sequence, nil
	}

	// How many events have been added since the cursor?
	behind := l.sequence - cursor
	if behind > uint64(l.count) {
		// Cursor is too old — some events were overwritten. Return all stored.
		behind = uint64(l.count)
	}

	result := make([]domain.GatewayEvent, behind)
	// Start position in the ring: head points to the next write slot,
	// so the oldest stored entry is at (head - count) mod capacity.
	// The first event we want is at (head - behind) mod capacity.
	ringCap := len(l.entries)
	start := (l.head - int(behind) + ringCap) % ringCap
	for i := range result {
		result[i] = l.entries[(start+i)%ringCap]
	}

	return l.sequence, result
}

// Len returns the number of events currently stored.
func (l *GatewayEventLog) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.count
}
