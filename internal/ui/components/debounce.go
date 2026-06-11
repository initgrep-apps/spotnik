// Package components — DebounceTracker is a reusable state machine for
// debouncing rapid keypresses (volume, seek) into a single API call.
// Both GradientVolumeBar and GradientSeekBar embed it to share the same
// seq-based stale-tick rejection logic.
package components

// DebounceTracker manages the debounce state for a value that can be
// adjusted interactively (volume, seek position). It tracks:
//   - current: the displayed value (pending or last confirmed)
//   - confirmed: the pre-seek baseline value (set by HandleKey's "confirmed" param)
//   - hasPending: true while a debounce tick is in flight
//   - seq: monotonically increasing counter for stale-tick rejection
//
// The typical flow is:
//  1. HandleKey(delta, confirmed, min, max) → updates current, stores confirmed as
//     the baseline, sets hasPending, increments seq, returns the new seq (or -1 if
//     min ≥ max)
//  2. A 300ms tea.Tick fires with the seq → HandleDebounce checks seq
//  3. On match: emit an intent message; on mismatch: discard (stale)
//  4. ConfirmFromAPI sets current to the API-confirmed value; hasPending stays
//     true until SetConfirmed receives a matching poll. For volume (integer
//     0-100) this protects against stale polls snapping the bar back. For seek
//     (milliseconds), the call site uses ClearPendingOnForwardProximity to clear
//     based on playback advancement rather than exact match.
//  5. CancelPending reverts to the store value on error
//  6. SetConfirmed: when hasPending is true, only clears on exact match (volume)
//     or proximity (seek via ClearPendingOnForwardProximity). When false, updates
//     current.
//
// The confirmed field enables direction-aware proximity clearing. After a forward
// seek (current >= confirmed), any poll at or past current clears pending. After
// a backward seek (current < confirmed), a poll clears pending only when it is at
// or past current AND still below the old baseline — this prevents stale forward
// polls (from before the backward seek) from incorrectly clearing pending.
type DebounceTracker struct {
	current    int
	confirmed  int
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
	} else {
		// Store the pre-seek baseline only on the first keypress of a burst.
		// This is the position before the user started adjusting, enabling
		// direction-aware proximity clearing in ClearPendingOnForwardProximity.
		d.confirmed = confirmed
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
// value (for volume, exact match) or by ClearPendingOnForwardProximity for seek
// positions where exact match is unlikely.
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

// ClearPendingOnForwardProximity clears hasPending when the poll value indicates
// the seek has been processed. This is used by seek bars where the confirmed
// position rarely matches exactly — playback naturally advances past the seek target,
// so any poll value at or beyond the target means the seek has been processed.
//
// Direction awareness prevents stale polls from incorrectly clearing pending after
// a backward seek:
//   - Forward seek (current >= confirmed): clear when val >= current
//   - Backward seek (current < confirmed): clear when val >= current AND val < confirmed
//
// The backward-seek guard prevents a stale poll at the old forward position from
// clearing pending. For example: user at 60s seeks back to 55s; a stale poll returning
// 60123 must NOT clear pending (60123 >= 55000 but 60123 >= 60000 = confirmed).
//
// When hasPending is false, this is a no-op. The "Forward" in the name signals that
// this method assumes monotonic forward playback advancement — it should not be used
// for controls where values can jump arbitrarily (volume bars use SetConfirmed instead).
func (d *DebounceTracker) ClearPendingOnForwardProximity(val int) {
	if !d.hasPending {
		return
	}
	if d.current >= d.confirmed {
		// Forward seek: any poll at or past the target clears pending.
		if val >= d.current {
			d.hasPending = false
		}
	} else {
		// Backward seek: clear only when poll is between target and old baseline.
		// A stale poll at or past the old baseline (before the backward seek)
		// must not clear pending.
		if val >= d.current && val < d.confirmed {
			d.hasPending = false
		}
	}
}
