---
title: "Error Resilience"
feature: 10-error-resilience
status: done
---

## Background
The architecture spec (docs/ARCHITECTURE.md) defines error handling strategies that were only partially implemented. 401 errors were shown as generic errors with no token refresh. Only fetchPlaybackStateCmd handled 429 rate limits with backoff -- library, search, and device API calls did not. Only playback commands checked for 403 (Premium required), leaving other Premium-gated operations showing generic errors. This story implements all three error-handling strategies universally across the codebase.

The implementation depends on the typed error types (RateLimitError, ForbiddenError, UnauthorizedError) and the TokenProvider interface established in the typed-errors feature (spec 24).

## Design

### 1. 401 Token Refresh + Retry-Once
Create a `RefreshableTokenProvider` that wraps keychain-based token storage with proactive refresh, handle `UnauthorizedError` in app result message handlers with refresh + single retry, and fall back to "Session expired. Run: spotnik auth" on refresh failure.

- `RefreshableTokenProvider` struct holds `tokenStore keychain.TokenStore`, `clientID string`, `mu sync.Mutex`
- `AccessToken(ctx)` checks if token expires within 5 minutes and proactively refreshes; if refresh fails but token is not yet expired, returns the existing token

### 2. 429 Backoff Extension
Make all `build*Cmd` functions check for `*api.RateLimitError` and return `RateLimitedMsg{Seconds: err.RetryAfter}` instead of a generic error message. The existing backoff mechanism (skip fetches while `backoffTicks > 0`) handles the rest.

Functions to update: `buildFetchPlaylistsCmd`, `buildFetchAlbumsCmd`, `buildFetchLikedTracksCmd`, `buildFetchRecentlyPlayedCmd`, `buildSearchCmd`, `buildFetchDevicesCmd`, `buildFetchStatsCmd`, `buildFetchPlaylistTracksCmd`

Existing pattern (playback only): `case RateLimitedMsg: a.backoffTicks = msg.Seconds * ticksPerSecond; a.store.SetError("Too many requests. Retrying...")`

### 3. 403 Handling Extension
Add `*api.ForbiddenError` checking to all `build*Cmd` functions: playback-related 403 shows "Spotify Premium required for playback"; other 403 shows the error message from `ForbiddenError.Message`. Use the status bar for display (existing pattern).

## Acceptance Criteria
- [ ] `RefreshableTokenProvider` exists with proactive 5-minute refresh
- [ ] 401 errors trigger token refresh + single retry
- [ ] Failed refresh shows "Session expired. Run: spotnik auth"
- [ ] All `build*Cmd` functions handle `RateLimitError` -> `RateLimitedMsg`
- [ ] 429 from any API call activates the backoff mechanism
- [ ] 403 errors show context-appropriate messages for all features
- [ ] All existing tests pass
- [ ] `make ci` passes

## Tasks
- [ ] Implement 401 token refresh + retry-once -- Create RefreshableTokenProvider in internal/api/token.go, handle UnauthorizedError in app.go Update handlers, use RefreshableTokenProvider in cmd/root.go
      - test: RefreshableTokenProvider refreshes when token near expiry
      - test: RefreshableTokenProvider uses cached token when valid
      - test: app handles 401 -> refresh -> retry flow
      - test: app shows error when refresh fails
      - test: Mock the token store for tests
- [ ] Extend 429 backoff to all API calls -- Add RateLimitError checking to all build*Cmd functions in commands.go
      - test: library fetch 429 -> RateLimitedMsg emitted
      - test: search 429 -> RateLimitedMsg emitted
      - test: backoff applies after any 429, not just playback
- [ ] Extend 403 handling beyond playback -- Add ForbiddenError checking to all build*Cmd functions
      - test: queue add 403 -> appropriate error message
      - test: playlist operation 403 -> appropriate error message
