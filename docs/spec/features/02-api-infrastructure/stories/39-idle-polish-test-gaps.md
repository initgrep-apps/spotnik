---
title: "Idle Polish & Test Coverage Gaps"
feature: 11-api-gateway
status: done
---

## Background
PR reviews identified 3 idle backoff UX issues and 8 test coverage gaps spread across multiple features: WindowSizeMsg doesn't reset idle, returning from idle during backoff shows no feedback, nil PlaybackState duration is untracked, weak toast assertions, missing buildSearchCmd store isolation test, missing SearchResultsMsg error path test, and untested concurrent stats partial failure.

Source: `docs/issues.md` -- PR #38 issues 11-13; PR #36 issue 1; PR #34 issues 8-10. Depends on: Feature 38.

## Design

### WindowSizeMsg Idle Reset
Terminal resize implies user presence but does not reset idle polling. Add idle reset to the WindowSizeMsg handler.

### Backoff Toast on Idle Return
When user returns from idle during active 429 backoff, show a ratelimit toast explaining stale data.

### Nil PlaybackState Tracking
Add `nilPlaybackStateTicks` counter. Warn after 30 nil ticks (~30-90s depending on interval).

### Weak Toast Assertion Tests
Five tests only assert `cmd != nil`. Replace with two-pass pattern: execute command, feed alert to Update, check rendered view.

### Missing Test Coverage
- buildSearchCmd store isolation test
- SearchResultsMsg error and clear path tests
- Concurrent stats partial failure test

## Acceptance Criteria
- [ ] WindowSizeMsg resets lastInteraction and tickCount when idle
- [ ] Returning from idle during backoff emits ratelimit toast
- [ ] Warning after 30 consecutive nil PlaybackState ticks
- [ ] Five weak toast assertion tests strengthened with two-pass pattern
- [ ] buildSearchCmd store isolation test exists
- [ ] SearchResultsMsg error and clear path tests exist
- [ ] Concurrent stats partial failure test exists
- [ ] `make ci` passes

## Tasks
- [ ] Reset idle on tea.WindowSizeMsg in app.go
      - test: WindowSizeMsg resets lastInteraction; resets tickCount when previously idle
- [ ] Show info toast on idle-return during backoff in app.go KeyMsg handler
      - test: returning from idle during backoff emits ratelimit toast; without backoff does NOT emit
- [ ] Track nil PlaybackState duration -- warn after 30 nil ticks
      - test: no warning before 30 nil ticks; warning at 30th; counter resets on non-nil state
- [ ] Strengthen weak toast assertion tests in app_test.go
      - test: self-verifying -- the strengthened tests ARE the verification
- [ ] Add buildSearchCmd store isolation test in elm_purity_test.go
      - test: buildSearchCmd does not write to store
- [ ] Add SearchResultsMsg error path test in elm_purity_test.go
      - test: error Msg doesn't update store results; emits error toast; clear path clears state
- [ ] Add concurrent stats partial failure test
      - test: TopTracks succeeds but TopArtists fails -- verify partial data and error toast
- [ ] Update issues.md -- mark all remaining issues resolved
      - test: docs change only
