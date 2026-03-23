package state_test

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetLog_Add_And_Entries(t *testing.T) {
	nl := state.NewNetLog()

	nl.Add(state.NetLogEntry{
		Timestamp:  time.Now(),
		Method:     "GET",
		Path:       "/v1/me/player",
		StatusCode: 200,
		DurationMs: 42,
	})
	nl.Add(state.NetLogEntry{
		Timestamp:  time.Now(),
		Method:     "GET",
		Path:       "/v1/me/player/queue",
		StatusCode: 429,
		DurationMs: 5,
	})

	entries := nl.Entries()
	require.Len(t, entries, 2)
	assert.Equal(t, "/v1/me/player", entries[0].Path)
	assert.Equal(t, "/v1/me/player/queue", entries[1].Path)
	assert.Equal(t, 429, entries[1].StatusCode)
}

func TestNetLog_Len(t *testing.T) {
	nl := state.NewNetLog()
	assert.Equal(t, 0, nl.Len())

	nl.Add(state.NetLogEntry{Method: "GET", Path: "/test"})
	assert.Equal(t, 1, nl.Len())
}

func TestNetLog_RingBuffer_Overflow(t *testing.T) {
	nl := state.NewNetLog()

	// Fill beyond capacity (200).
	for i := 0; i < 210; i++ {
		nl.Add(state.NetLogEntry{
			Method:     "GET",
			Path:       "/v1/test",
			StatusCode: 200,
			DurationMs: int64(i),
		})
	}

	assert.Equal(t, 200, nl.Len())

	entries := nl.Entries()
	require.Len(t, entries, 200)
	// Oldest entry should be the 11th added (index 10, DurationMs=10).
	assert.Equal(t, int64(10), entries[0].DurationMs)
	// Newest should be the last added (DurationMs=209).
	assert.Equal(t, int64(209), entries[199].DurationMs)
}

func TestNetLog_Empty_Entries(t *testing.T) {
	nl := state.NewNetLog()
	entries := nl.Entries()
	assert.Nil(t, entries)
}

func TestStore_RecordNetCall(t *testing.T) {
	s := state.New()
	s.RecordNetCall("GET", "/v1/me/player", 200, 50)
	s.RecordNetCall("PUT", "/v1/me/player/play", 204, 120)

	entries := s.NetLogEntries()
	require.Len(t, entries, 2)
	assert.Equal(t, "GET", entries[0].Method)
	assert.Equal(t, "PUT", entries[1].Method)
	assert.Equal(t, 204, entries[1].StatusCode)
}
