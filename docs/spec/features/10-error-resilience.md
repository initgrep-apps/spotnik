---
title: "Error Resilience"
description: "Automatic token refresh on 401, universal rate-limit backoff on 429, and context-aware 403 messaging across all API calls, so the app recovers gracefully from Spotify API errors without user intervention."
status: done
stories: [27]
---

# Error Resilience

## Background

Spotnik's architecture spec defines error handling strategies for HTTP 401, 429, and 403 responses from the Spotify API. However, these strategies were only partially implemented: 401 token refresh was missing entirely, 429 backoff only covered playback polling, and 403 Premium-required messaging was limited to playback commands.

This feature closes those gaps by implementing a `RefreshableTokenProvider` with proactive 5-minute-before-expiry refresh, extending the existing backoff mechanism so every `build*Cmd` function emits `RateLimitedMsg` on 429, and adding context-appropriate 403 error messages for non-playback API calls. Together these changes ensure the app recovers from transient Spotify API errors without requiring the user to restart or re-authenticate.

The implementation depends on the typed error types (`RateLimitError`, `ForbiddenError`, `UnauthorizedError`) and the `TokenProvider` interface established in the typed-errors feature (spec 24).

---

## Story: Error Resilience (spec 27)

### Background

The architecture spec (`docs/ARCHITECTURE.md`) defines error handling strategies that were only partially implemented. 401 errors were shown as generic errors with no token refresh. Only `fetchPlaybackStateCmd` handled 429 rate limits with backoff — library, search, and device API calls did not. Only playback commands checked for 403 (Premium required), leaving other Premium-gated operations showing generic errors. This story implements all three error-handling strategies universally across the codebase.

### Acceptance Criteria

- [ ] `RefreshableTokenProvider` exists with proactive 5-minute refresh
- [ ] 401 errors trigger token refresh + single retry
- [ ] Failed refresh shows "Session expired. Run: spotnik auth"
- [ ] All `build*Cmd` functions handle `RateLimitError` → `RateLimitedMsg`
- [ ] 429 from any API call activates the backoff mechanism
- [ ] 403 errors show context-appropriate messages for all features
- [ ] All existing tests pass
- [ ] `make ci` passes

### Tasks

1. **Implement 401 token refresh + retry-once** — When a token expires mid-session, all API calls fail with 401 and the user must restart to re-authenticate. Create a `RefreshableTokenProvider` that wraps keychain-based token storage with proactive refresh, handle `UnauthorizedError` in app result message handlers with refresh + single retry, and fall back to "Session expired. Run: spotnik auth" on refresh failure.
   - Files: `internal/api/token.go` — Add `RefreshableTokenProvider`
   - Files: `internal/app/app.go` — Handle `UnauthorizedError` with refresh + retry
   - Files: `cmd/root.go` — Use `RefreshableTokenProvider` instead of `StaticTokenProvider`
   - Implementation detail: `RefreshableTokenProvider` struct holds `tokenStore keychain.TokenStore`, `clientID string`, `mu sync.Mutex`. `AccessToken(ctx)` checks if token expires within 5 minutes and proactively refreshes; if refresh fails but token is not yet expired, returns the existing token.
   - Tests: Unit test — `RefreshableTokenProvider` refreshes when token near expiry
   - Tests: Unit test — `RefreshableTokenProvider` uses cached token when valid
   - Tests: Unit test — app handles 401 → refresh → retry flow
   - Tests: Unit test — app shows error when refresh fails
   - Tests: Mock the token store for tests

2. **Extend 429 backoff to all API calls** — Only `fetchPlaybackStateCmd` handles 429 rate limits. Other API calls that hit 429 show errors but do not activate the backoff mechanism, leading to repeated rate-limited requests. Make all `build*Cmd` functions check for `*api.RateLimitError` and return `RateLimitedMsg{Seconds: err.RetryAfter}` instead of a generic error message. The existing backoff mechanism (skip fetches while `backoffTicks > 0`) handles the rest.
   - Files: `internal/app/commands.go` (or `app.go`) — Add `RateLimitError` checking to all `build*Cmd` functions
   - Functions to update: `buildFetchPlaylistsCmd`, `buildFetchAlbumsCmd`, `buildFetchLikedTracksCmd`, `buildFetchRecentlyPlayedCmd`, `buildSearchCmd`, `buildFetchDevicesCmd`, `buildFetchStatsCmd`, `buildFetchPlaylistTracksCmd`
   - Existing pattern (playback only): `case RateLimitedMsg: a.backoffTicks = msg.Seconds * ticksPerSecond; a.store.SetError("Too many requests. Retrying...")`
   - Tests: Unit test — library fetch 429 → `RateLimitedMsg` emitted
   - Tests: Unit test — search 429 → `RateLimitedMsg` emitted
   - Tests: Unit test — backoff applies after any 429, not just playback

3. **Extend 403 handling beyond playback** — Only playback commands show "Spotify Premium required" on 403. Other API calls that may require Premium (e.g., adding to queue) or encounter 403 (e.g., private playlist access) show generic errors. Add `*api.ForbiddenError` checking to all `build*Cmd` functions: playback-related 403 shows "Spotify Premium required for playback"; other 403 shows the error message from `ForbiddenError.Message`. Use the status bar for display (existing pattern).
   - Files: `internal/app/commands.go` (or `app.go`) — Add `ForbiddenError` checking
   - Tests: Unit test — queue add 403 → appropriate error message
   - Tests: Unit test — playlist operation 403 → appropriate error message
