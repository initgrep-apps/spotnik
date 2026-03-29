// Tests for net log and gateway call recording.
// This file intentionally uses deprecated domain.GatewayDecision types that are
// retained for Feature 68 migration compatibility.
//
//nolint:staticcheck // Deprecated: GatewayDecision types retained until Feature 68.
package state_test

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
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

func TestNetLogEntry_HasPriorityAndDecision(t *testing.T) {
	nl := state.NewNetLog()

	entry := state.NetLogEntry{
		Timestamp:       time.Now(),
		Method:          "GET",
		Path:            "/v1/me/player",
		StatusCode:      200,
		DurationMs:      50,
		Priority:        domain.PriorityInteractive,
		GatewayDecision: domain.DecisionAllowed,
	}
	nl.Add(entry)

	entries := nl.Entries()
	require.Len(t, entries, 1)
	assert.Equal(t, domain.PriorityInteractive, entries[0].Priority)
	assert.Equal(t, domain.DecisionAllowed, entries[0].GatewayDecision)
}

func TestNetLogEntry_ZeroValueDefaults(t *testing.T) {
	// Zero-valued NetLogEntry should have Background priority and DecisionAllowed.
	var e state.NetLogEntry
	assert.Equal(t, domain.PriorityBackground, e.Priority, "default priority is Background (zero)")
	assert.Equal(t, domain.DecisionAllowed, e.GatewayDecision, "default decision is Allowed (zero)")
}

func TestStore_RecordGatewayCall_PopulatesAllFields(t *testing.T) {
	s := state.New()
	s.RecordGatewayCall("GET", "/v1/me/player", 200, 45, domain.PriorityInteractive, domain.DecisionAllowed)
	s.RecordGatewayCall("GET", "/v1/me/player", 0, 0, domain.PriorityBackground, domain.DecisionBlocked)

	entries := s.NetLogEntries()
	require.Len(t, entries, 2)

	// First entry: allowed Interactive request.
	assert.Equal(t, "GET", entries[0].Method)
	assert.Equal(t, "/v1/me/player", entries[0].Path)
	assert.Equal(t, 200, entries[0].StatusCode)
	assert.Equal(t, int64(45), entries[0].DurationMs)
	assert.Equal(t, domain.PriorityInteractive, entries[0].Priority)
	assert.Equal(t, domain.DecisionAllowed, entries[0].GatewayDecision)
	assert.False(t, entries[0].Timestamp.IsZero(), "timestamp must be set")

	// Second entry: blocked Background request.
	assert.Equal(t, domain.PriorityBackground, entries[1].Priority)
	assert.Equal(t, domain.DecisionBlocked, entries[1].GatewayDecision)
	assert.Equal(t, 0, entries[1].StatusCode)
}

func TestStore_RecordNetCall_BackwardCompat(t *testing.T) {
	// RecordNetCall (old path) must still work and default to Background+Allowed.
	s := state.New()
	s.RecordNetCall("GET", "/v1/me/player", 200, 50)

	entries := s.NetLogEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, domain.PriorityBackground, entries[0].Priority)
	assert.Equal(t, domain.DecisionAllowed, entries[0].GatewayDecision)
}
