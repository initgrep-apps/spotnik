package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --------------------------------------------------------------------------
// DebounceTracker — HandleKey
// --------------------------------------------------------------------------

func TestDebounceTracker_HandleKey_UpdatesCurrent(t *testing.T) {
	var d DebounceTracker
	seq := d.HandleKey(5, 10, 0, 100)
	assert.Equal(t, 15, d.Current(), "current should be confirmed+delta")
	assert.True(t, d.HasPending(), "should have pending after HandleKey")
	assert.Equal(t, 1, seq, "first HandleKey should return seq=1")
}

func TestDebounceTracker_HandleKey_ClampsAtMax(t *testing.T) {
	var d DebounceTracker
	seq := d.HandleKey(50, 90, 0, 100)
	assert.Equal(t, 100, d.Current(), "should clamp to max")
	assert.Equal(t, 1, seq)
}

func TestDebounceTracker_HandleKey_ClampsAtMin(t *testing.T) {
	var d DebounceTracker
	seq := d.HandleKey(-50, 10, 0, 100)
	assert.Equal(t, 0, d.Current(), "should clamp to min")
	assert.Equal(t, 1, seq)
}

func TestDebounceTracker_HandleKey_AccumulatesFromPending(t *testing.T) {
	var d DebounceTracker
	d.HandleKey(5, 50, 0, 100)        // seq=1, current=55, pending
	seq := d.HandleKey(5, 50, 0, 100) // must use current=55 as base, not confirmed=50
	assert.Equal(t, 60, d.Current(), "should accumulate from pending value")
	assert.Equal(t, 2, seq, "second HandleKey should return seq=2")
}

func TestDebounceTracker_HandleKey_NoOpWhenMinGeMax(t *testing.T) {
	var d DebounceTracker
	seq := d.HandleKey(5, 50, 100, 50) // min > max → no-op
	assert.Equal(t, -1, seq, "should return -1 when min >= max")
	assert.Equal(t, 0, d.Current(), "should not change current")
	assert.False(t, d.HasPending(), "should not set pending")
}

func TestDebounceTracker_HandleKey_NoOpWhenMinMaxEqual(t *testing.T) {
	var d DebounceTracker
	seq := d.HandleKey(5, 50, 100, 100) // min == max → no-op
	assert.Equal(t, -1, seq)
	assert.Equal(t, 0, d.Current())
	assert.False(t, d.HasPending())
}

func TestDebounceTracker_HandleKey_SeqIncrements(t *testing.T) {
	var d DebounceTracker
	seq1 := d.HandleKey(1, 50, 0, 100)
	seq2 := d.HandleKey(1, 50, 0, 100)
	seq3 := d.HandleKey(1, 50, 0, 100)
	assert.Equal(t, 1, seq1)
	assert.Equal(t, 2, seq2)
	assert.Equal(t, 3, seq3)
}

// --------------------------------------------------------------------------
// DebounceTracker — HandleDebounce
// --------------------------------------------------------------------------

func TestDebounceTracker_HandleDebounce_CurrentAccepted(t *testing.T) {
	var d DebounceTracker
	d.HandleKey(5, 50, 0, 100) // seq=1, current=55
	matched, target, tickSeq := d.HandleDebounce(1)
	assert.True(t, matched, "current seq should match")
	assert.Equal(t, 55, target)
	assert.Equal(t, 1, tickSeq, "HandleDebounce returns the matched tick seq")
}

func TestDebounceTracker_HandleDebounce_StaleRejected(t *testing.T) {
	var d DebounceTracker
	d.HandleKey(5, 50, 0, 100) // seq=1
	d.HandleKey(5, 50, 0, 100) // seq=2 — supersedes seq=1
	matched, _, _ := d.HandleDebounce(1)
	assert.False(t, matched, "stale seq must be discarded")
}

func TestDebounceTracker_HandleDebounce_DoubleFireGuard(t *testing.T) {
	var d DebounceTracker
	d.HandleKey(5, 50, 0, 100) // seq=1
	matched, _, _ := d.HandleDebounce(1)
	assert.True(t, matched, "first call should match")
	// Second call with same seq should be stale (double-fire guard)
	matched2, _, _ := d.HandleDebounce(1)
	assert.False(t, matched2, "second call with same seq must be stale")
}

// --------------------------------------------------------------------------
// DebounceTracker — SetConfirmed
// --------------------------------------------------------------------------

func TestDebounceTracker_SetConfirmed_UpdatesWhenNoPending(t *testing.T) {
	var d DebounceTracker
	d.SetConfirmed(50)
	assert.Equal(t, 50, d.Current())
	assert.False(t, d.HasPending())
}

func TestDebounceTracker_SetConfirmed_BlocksWhenPendingMismatch(t *testing.T) {
	var d DebounceTracker
	d.HandleKey(5, 50, 0, 100) // current=55, pending
	d.SetConfirmed(30)         // should be ignored (pending, value != current)
	assert.Equal(t, 55, d.Current(), "should keep pending value")
	assert.True(t, d.HasPending(), "should keep pending flag")
}

func TestDebounceTracker_SetConfirmed_ClearsPendingOnMatch(t *testing.T) {
	var d DebounceTracker
	d.HandleKey(5, 50, 0, 100) // current=55, pending
	d.SetConfirmed(55)         // value matches current → clear pending
	assert.Equal(t, 55, d.Current())
	assert.False(t, d.HasPending(), "should clear pending when value matches current")
}

// --------------------------------------------------------------------------
// DebounceTracker — ConfirmFromAPI
// --------------------------------------------------------------------------

func TestDebounceTracker_ConfirmFromAPI_SeqMatch(t *testing.T) {
	var d DebounceTracker
	d.HandleKey(5, 50, 0, 100)                 // seq=1, current=55, pending
	matched, _, tickSeq := d.HandleDebounce(1) // d.seq→2, tickSeq=1
	requireTrue(t, matched)
	d.ConfirmFromAPI(tickSeq, 55) // d.seq (2) == tickSeq+1 (2) → match
	assert.Equal(t, 55, d.Current(), "should update to API-confirmed value")
	assert.True(t, d.HasPending(), "hasPending stays true until SetConfirmed matches or ClearPendingOnProximity clears")
}

func TestDebounceTracker_ConfirmFromAPI_SeqMismatch(t *testing.T) {
	var d DebounceTracker
	d.HandleKey(5, 50, 0, 100)                 // seq=1
	matched, _, tickSeq := d.HandleDebounce(1) // d.seq→2, tickSeq=1
	requireTrue(t, matched)
	d.HandleKey(5, 50, 0, 100)    // seq=3 — new burst supersedes
	d.ConfirmFromAPI(tickSeq, 55) // d.seq (4) != tickSeq+1 (2) → no-op
	assert.True(t, d.HasPending(), "hasPending should stay true on seq mismatch")
}

// --------------------------------------------------------------------------
// DebounceTracker — ClearPendingOnForwardProximity
// --------------------------------------------------------------------------

func TestDebounceTracker_ClearPendingOnForwardProximity_ClearsWhenAtOrPast(t *testing.T) {
	// Seek bar scenario: after ConfirmFromAPI, hasPending stays true.
	// A poll arriving at or past the confirmed position clears pending.
	var d DebounceTracker
	d.HandleKey(5000, 30000, 0, 180000) // current=35000, pending, confirmed=30000 (forward seek)
	d.ConfirmFromAPI(1, 35000)          // hasPending stays true (volume-style)

	// Poll at the exact confirmed position → clears pending
	d.ClearPendingOnForwardProximity(35000)
	assert.False(t, d.HasPending(), "should clear pending when poll reaches confirmed position")

	// Reset and test poll past the confirmed position
	d = DebounceTracker{}
	d.HandleKey(5000, 30000, 0, 180000) // current=35000, pending, confirmed=30000 (forward seek)
	d.ConfirmFromAPI(1, 35000)          // hasPending stays true

	// Poll past the confirmed position (playback advanced 3s) → clears pending
	d.ClearPendingOnForwardProximity(38000)
	assert.False(t, d.HasPending(), "should clear pending when poll passes confirmed position")
}

func TestDebounceTracker_ClearPendingOnForwardProximity_NoOpBelowTarget(t *testing.T) {
	// Poll has not yet reached the confirmed position → pending stays
	var d DebounceTracker
	d.HandleKey(5000, 30000, 0, 180000) // current=35000, pending, confirmed=30000 (forward seek)
	d.ConfirmFromAPI(1, 35000)          // hasPending stays true

	// Stale poll with old position → does NOT clear pending
	d.ClearPendingOnForwardProximity(30000)
	assert.True(t, d.HasPending(), "stale poll below confirmed position must not clear pending")
	assert.Equal(t, 35000, d.Current(), "current should stay at confirmed position")
}

func TestDebounceTracker_ClearPendingOnForwardProximity_NoOpWhenNotPending(t *testing.T) {
	// When not pending, ClearPendingOnForwardProximity is a no-op
	var d DebounceTracker
	d.SetConfirmed(50)
	assert.False(t, d.HasPending())
	d.ClearPendingOnForwardProximity(50)
	assert.Equal(t, 50, d.Current(), "should not change current when not pending")
	assert.False(t, d.HasPending())
}

func TestDebounceTracker_ClearPendingOnForwardProximity_BackwardSeekStaleForwardPoll(t *testing.T) {
	// CRITICAL: Backward seek must NOT be cleared by a stale poll at the old
	// forward position. User at 60s seeks back to 55s; stale poll returns 60123.
	// 60123 >= 55000 (current) but 60123 >= 60000 (confirmed) → must NOT clear.
	var d DebounceTracker
	d.HandleKey(-5000, 60000, 0, 180000) // current=55000, confirmed=60000 (backward seek)
	d.ConfirmFromAPI(1, 55000)           // hasPending stays true

	// Stale poll at old forward position → must NOT clear pending
	d.ClearPendingOnForwardProximity(60123)
	assert.True(t, d.HasPending(), "stale forward poll must not clear pending on backward seek")
	assert.Equal(t, 55000, d.Current(), "current should stay at backward-seek target")
}

func TestDebounceTracker_ClearPendingOnForwardProximity_BackwardSeekValidPoll(t *testing.T) {
	// After a backward seek, a poll at or near the target (but below the old
	// baseline) should clear pending. User at 60s seeks back to 55s; Spotify
	// processes the seek and the next poll returns 55234.
	var d DebounceTracker
	d.HandleKey(-5000, 60000, 0, 180000) // current=55000, confirmed=60000 (backward seek)
	d.ConfirmFromAPI(1, 55000)           // hasPending stays true

	// Poll at target position, below old baseline → clears pending
	d.ClearPendingOnForwardProximity(55234)
	assert.False(t, d.HasPending(), "valid poll at backward-seek target should clear pending")

	// Also test exact match at target
	d2 := DebounceTracker{}
	d2.HandleKey(-5000, 60000, 0, 180000) // current=55000, confirmed=60000
	d2.ConfirmFromAPI(1, 55000)

	d2.ClearPendingOnForwardProximity(55000)
	assert.False(t, d2.HasPending(), "exact poll at backward-seek target should clear pending")
}

func TestDebounceTracker_ClearPendingOnForwardProximity_BackwardSeekPollAtOldBaseline(t *testing.T) {
	// Edge case: poll returning exactly the old baseline (confirmed) value
	// must NOT clear pending — it is at the boundary and indicates a stale poll
	// that hasn't yet reflected the backward seek.
	var d DebounceTracker
	d.HandleKey(-5000, 60000, 0, 180000) // current=55000, confirmed=60000 (backward seek)
	d.ConfirmFromAPI(1, 55000)

	// Poll at exactly the old baseline → must NOT clear (val < confirmed is strict)
	d.ClearPendingOnForwardProximity(60000)
	assert.True(t, d.HasPending(), "poll at old baseline must not clear pending on backward seek")
}

// --------------------------------------------------------------------------
// DebounceTracker — CancelPending
// --------------------------------------------------------------------------

func TestDebounceTracker_CancelPending_SeqMatch(t *testing.T) {
	var d DebounceTracker
	d.HandleKey(5, 50, 0, 100)                 // seq=1, current=55
	matched, _, tickSeq := d.HandleDebounce(1) // d.seq→2, tickSeq=1
	requireTrue(t, matched)
	d.CancelPending(tickSeq, 50) // d.seq (2) == tickSeq+1 (2) → match, revert to confirmed store value
	assert.Equal(t, 50, d.Current(), "should revert to confirmed value")
	assert.False(t, d.HasPending(), "should clear pending")
}

func TestDebounceTracker_CancelPending_SeqMismatch(t *testing.T) {
	var d DebounceTracker
	d.HandleKey(5, 50, 0, 100)                 // seq=1
	matched, _, tickSeq := d.HandleDebounce(1) // d.seq→2, tickSeq=1
	requireTrue(t, matched)
	d.HandleKey(5, 50, 0, 100)   // seq=3 — new burst
	d.CancelPending(tickSeq, 50) // d.seq (4) != tickSeq+1 (2) → no-op
	assert.True(t, d.HasPending(), "should keep pending on seq mismatch")
}

// --------------------------------------------------------------------------
// DebounceTracker — Current / HasPending accessors
// --------------------------------------------------------------------------

func TestDebounceTracker_Current_InitialValue(t *testing.T) {
	var d DebounceTracker
	assert.Equal(t, 0, d.Current(), "initial current should be 0")
	assert.False(t, d.HasPending(), "initial hasPending should be false")
}

// requireTrue is a test helper that asserts a condition is true.
func requireTrue(t *testing.T, cond bool) {
	t.Helper()
	if !cond {
		t.Fatal("expected condition to be true")
	}
}
