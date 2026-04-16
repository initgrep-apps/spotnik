---
title: "Gateway Rate Protection"
status: done
---

## Context

Feature 26 fixed two gateway correctness bugs:

1. **Story 124** — Interactive GETs now skip the inflight dedup map, so a
   reconcile fetch after a user command always fires a fresh HTTP call and
   never shares a pre-command Background poll's stale response.

2. **Story 125** — The 100ms `interactiveDebounce` was removed from the gateway
   because it silently dropped semantically independent playback commands (e.g.
   three volume presses at 60ms intervals → only the last one fired).

Both fixes were correct. But removing the debounce without adding any
compensating rate protection exposed two new failure modes observed in
production immediately after the feature shipped.

---

## Description

Fixes two interactive-priority bugs in the gateway's backoff and token-bucket
policies that were unmasked by the F26 debounce removal.

**Bug 1 — Interactive requests wait on backoff instead of being rejected.**
When the gateway receives a 429, `backoffUntil` is set. Interactive requests
entering `Do()` during that window call `waitForBackoff()`, which *blocks* the
goroutine until the backoff expires. Holding the volume key for a few seconds
queues many goroutines sleeping in `waitForBackoff`. When the backoff expires
all goroutines wake simultaneously, burst-fire against Spotify (bypassing the
token bucket — see Bug 2), trigger a fresh 429, and the cycle restarts. Requests
remain "waiting" for minutes rather than the expected 10 seconds.

**Bug 2 — Interactive requests bypass the token bucket.**
The `bucket.wait` call in `Do()` is inside the `else` branch that only executes
for Background requests. Interactive requests skip it entirely. A user holding
the `+` or `-` key generates OS key-repeat events at ~10–30 per second. Each
event fires a goroutine that immediately attempts an HTTP PUT with no local
rate gate. This is the root cause of the 429: Spotify's own rate limit is hit
because the app applies none of its own.

---

## Acceptance Criteria

- Interactive requests arriving during an active 429 backoff return a
  `RateLimitError` immediately — they do NOT wait in `waitForBackoff`.
- Background requests during backoff continue to be rejected immediately
  (existing behaviour, unchanged).
- Interactive requests consume a token from the bucket before proceeding,
  exactly as Background requests do.
- A single interactive request with a warm bucket (tokens available) still
  feels instant — no perceptible latency added.
- Holding the volume key at OS key-repeat rate (~15 events/s) does not trigger
  a 429 — the token bucket caps throughput at 10 req/s.
- `waitForBackoff` is deleted — it is dead code after this feature.
- The `waited` field and `EventRequestWaited` emission path in `Do()` are
  removed — the event is no longer emitted by the gateway.
- The `PlaybackCmdSentMsg` handler emits a distinct "Rate limited" toast (not
  a raw error string) when the error is a `RateLimitError`.
- `make ci` passes (lint + tests + 80% coverage).
