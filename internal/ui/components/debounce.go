// Package components — DebounceTracker is a reusable state machine for
// debouncing rapid keypresses (volume, seek) into a single API call.
// Both GradientVolumeBar and GradientSeekBar embed it to share the same
// seq-based stale-tick rejection logic.
package components

// DebounceTracker manages the debounce state for a value that can be
// adjusted interactively (volume, seek position). It tracks:
//   - current: the displayed value (pending or last confirmed)
//   - hasPending: true while a debounce tick is in flight
//   - seq: monotonically increasing counter for stale-tick rejection
//
// The typical flow is:
//  1. HandleKey(delta, confirmed, min, max) → updates current, sets hasPending,
//     increments seq, returns the new seq (or -1 if min ≥ max)
//  2. A 300ms tea.Tick fires with the seq → HandleDebounce checks seq
//  3. On match: emit an intent message; on mismatch: discard (stale)
//  4. ConfirmFromAPI sets current to the API-confirmed value; hasPending stays
//     true until SetConfirmed receives a matching poll. For volume (integer
//     0-100) this protects against stale polls snapping the bar back. For seek
//     (milliseconds), the call site uses ClearPendingOnProximity to clear based
//     on playback advancement rather than exact match.
//  5. CancelPending reverts to the store value on error
//  6. SetConfirmed: when hasPending is true, only clears on exact match (volume)
//     or proximity (seek via ClearPendingOnProximity). When false, updates current.
type DebounceTracker struct {
	current    int
	hasPending bool
	seq        int
}

// Current returns the displayed value (pending or last confirmed).
func (d *DebounceTracker) Current() int { return d.current }

// HasPending returns true while a debounce tick is in flight.
func (d *DebounceTracker) HasPending() bool { return d.hasPending }

// HandleKey computes the new pending value, updates current immediately so
// the bar renders the new value on the next frame, increments seq to
// invalidate any in-flight debounce tick, and returns the new seq number.
// Returns -1 (and no state change) when min ≥ max (no valid range).
//
// The caller is responsible for wrapping the returned seq into a
// tea.Tick command that carries both the target value and the seq.
func (d *DebounceTracker) HandleKey(delta, confirmed, min, max int) int {
	if min >= max {
		return -1
	}
	base := confirmed
	if d.hasPending {
		base = d.current
	}
	newVal := base + delta
	if newVal > max {
		newVal = max
	}
	if newVal < min {
		newVal = min
	}
	d.current = newVal
	d.hasPending = true
	d.seq++
	return d.seq
}

// HandleDebounce checks whether the debounce tick is current.
// Returns (true, targetVal, tickSeq) when matched — the caller must forward
// tickSeq through an intent message so ConfirmFromAPI/CancelPending can guard
// against concurrent bursts. tickSeq is the seq value that matched, and the
// bar's internal seq is now tickSeq+1, so the guard condition seq==tickSeq+1 works.
// Returns (false, 0, 0) when the tick is stale.
func (d *DebounceTracker) HandleDebounce(tickSeq int) (matched bool, targetVal int, seq int) {
	if tickSeq != d.seq {
		return false, 0, 0
	}
	d.seq++ // double-fire guard: any future tick with this same seq is now stale
	return true, d.current, tickSeq
}

// ConfirmFromAPI sets current to the API-confirmed value but keeps hasPending true.
// hasPending is only cleared by SetConfirmed when a subsequent poll returns the same
// value (for volume, exact match) or by ClearPendingOnProximity for seek positions
// where exact match is unlikely.
// Guards on seq == intentSeq+1 so concurrent bursts don't clobber each other.
func (d *DebounceTracker) ConfirmFromAPI(intentSeq, val int) {
	if d.seq == intentSeq+1 {
		d.current = val
	}
}

// CancelPending reverts current to the last confirmed store value and clears
// hasPending, but only when no newer burst has started. Call this on API error
// so the bar immediately snaps back to the real server-side value.
func (d *DebounceTracker) CancelPending(intentSeq, confirmed int) {
	if d.seq == intentSeq+1 {
		d.hasPending = false
		d.current = confirmed
	}
}

// SetConfirmed updates current from the authoritative poll value.
// When hasPending is true, it only clears hasPending if the poll matches current,
// preventing stale in-flight polls from snapping the bar back during a debounce burst.
// When hasPending is false (normal polling), it updates current directly.
func (d *DebounceTracker) SetConfirmed(val int) {
	if d.hasPending {
		if val == d.current {
			d.hasPending = false
		}
		return
	}
	d.current = val
}

// ClearPendingOnProximity clears hasPending when the poll value has reached or
// passed the current pending value. This is used by seek bars where the confirmed
// position rarely matches exactly — playback naturally advances past the seek target,
// so any poll value at or beyond the target means the seek has been processed.
// When hasPending is false, this is a no-op.
func (d *DebounceTracker) ClearPendingOnProximity(val int) {
	if d.hasPending && val >= d.current {
		d.hasPending = false
	}
}
