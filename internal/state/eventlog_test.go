package state_test

import (
	"sync"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeEvent returns a GatewayEvent with the given kind for use in tests.
func makeEvent(kind domain.EventKind) domain.GatewayEvent {
	return domain.GatewayEvent{
		Timestamp: time.Now(),
		Kind:      kind,
		RequestID: uint64(kind) + 1,
		Path:      "/test",
	}
}

func TestGatewayEventLog_Add_IncrementsCounts(t *testing.T) {
	log := state.NewGatewayEventLog(10)
	assert.Equal(t, 0, log.Len())

	log.Add(makeEvent(domain.EventRequestEntered))
	log.Add(makeEvent(domain.EventTokenConsumed))
	log.Add(makeEvent(domain.EventHttpCompleted))

	assert.Equal(t, 3, log.Len())
}

func TestGatewayEventLog_Add_RingWraparound(t *testing.T) {
	capacity := 5
	log := state.NewGatewayEventLog(capacity)

	// Add more entries than capacity.
	for i := 0; i < 8; i++ {
		log.Add(makeEvent(domain.EventKind(i % 13)))
	}

	// Len caps at capacity.
	assert.Equal(t, capacity, log.Len())
}

func TestGatewayEventLog_ReadFrom_FirstCall(t *testing.T) {
	log := state.NewGatewayEventLog(10)
	log.Add(makeEvent(domain.EventRequestEntered))
	log.Add(makeEvent(domain.EventTokenConsumed))
	log.Add(makeEvent(domain.EventSemaphoreAcquired))

	cursor, events := log.ReadFrom(0)
	assert.Equal(t, uint64(3), cursor)
	require.Len(t, events, 3)
}

func TestGatewayEventLog_ReadFrom_IncrementalReads(t *testing.T) {
	log := state.NewGatewayEventLog(10)
	log.Add(makeEvent(domain.EventRequestEntered))
	log.Add(makeEvent(domain.EventTokenConsumed))
	log.Add(makeEvent(domain.EventHttpCompleted))

	// First read: cursor=0 returns all 3 events.
	cursor, events := log.ReadFrom(0)
	assert.Len(t, events, 3)
	assert.Equal(t, uint64(3), cursor)

	// Add 2 more events.
	log.Add(makeEvent(domain.EventSemaphoreAcquired))
	log.Add(makeEvent(domain.EventSemaphoreReleased))

	// Second read: from previous cursor returns only the 2 new events.
	cursor2, events2 := log.ReadFrom(cursor)
	assert.Len(t, events2, 2)
	assert.Equal(t, uint64(5), cursor2)
}

func TestGatewayEventLog_ReadFrom_CursorUpToDate(t *testing.T) {
	log := state.NewGatewayEventLog(10)
	log.Add(makeEvent(domain.EventRequestEntered))

	cursor, _ := log.ReadFrom(0)

	// Cursor is up to date — nothing new.
	cursor2, events := log.ReadFrom(cursor)
	assert.Nil(t, events)
	assert.Equal(t, cursor, cursor2)
}

func TestGatewayEventLog_ReadFrom_CursorTooOld(t *testing.T) {
	capacity := 5
	log := state.NewGatewayEventLog(capacity)

	// Fill beyond capacity so some events are overwritten.
	for i := 0; i < 8; i++ {
		log.Add(makeEvent(domain.EventKind(i % 13)))
	}

	// A cursor of 0 is older than the oldest retained event (sequence is 8, capacity is 5).
	// Graceful recovery: return all stored events.
	_, events := log.ReadFrom(0)
	assert.Len(t, events, capacity, "stale cursor should return all stored events")
}

func TestGatewayEventLog_ReadFrom_EventOrdering(t *testing.T) {
	log := state.NewGatewayEventLog(10)

	kinds := []domain.EventKind{
		domain.EventRequestEntered,
		domain.EventTokenConsumed,
		domain.EventSemaphoreAcquired,
		domain.EventHttpCompleted,
	}
	for _, k := range kinds {
		log.Add(makeEvent(k))
	}

	_, events := log.ReadFrom(0)
	require.Len(t, events, len(kinds))
	for i, ev := range events {
		assert.Equal(t, kinds[i], ev.Kind, "event at index %d has wrong kind", i)
	}
}

func TestGatewayEventLog_ReadFrom_IndependentCursors(t *testing.T) {
	log := state.NewGatewayEventLog(20)

	// Consumer A reads first 3.
	log.Add(makeEvent(domain.EventRequestEntered))
	log.Add(makeEvent(domain.EventTokenConsumed))
	log.Add(makeEvent(domain.EventHttpCompleted))
	cursorA, eventsA := log.ReadFrom(0)
	assert.Len(t, eventsA, 3)

	// 2 more events added.
	log.Add(makeEvent(domain.EventSemaphoreAcquired))
	log.Add(makeEvent(domain.EventSemaphoreReleased))

	// Consumer B starts from scratch — reads all 5.
	_, eventsB := log.ReadFrom(0)
	assert.Len(t, eventsB, 5)

	// Consumer A advances from its cursor — reads only the 2 new events.
	_, eventsA2 := log.ReadFrom(cursorA)
	assert.Len(t, eventsA2, 2)
}

func TestGatewayEventLog_Add_ZeroCapacity(t *testing.T) {
	// capacity=0 must default to 500.
	log := state.NewGatewayEventLog(0)

	// Add 501 events — Len should cap at 500, not panic.
	for i := 0; i < 501; i++ {
		log.Add(makeEvent(domain.EventKind(i % 13)))
	}
	assert.Equal(t, 500, log.Len())
}

func TestGatewayEventLog_ConcurrentAccess(t *testing.T) {
	log := state.NewGatewayEventLog(100)
	var wg sync.WaitGroup

	// 10 writers.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				log.Add(makeEvent(domain.EventRequestEntered))
			}
		}()
	}

	// 5 readers.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var cursor uint64
			for j := 0; j < 10; j++ {
				cursor, _ = log.ReadFrom(cursor)
			}
		}()
	}

	wg.Wait()
	// If we got here without a race or panic, the test passes.
	assert.LessOrEqual(t, log.Len(), 100)
}
