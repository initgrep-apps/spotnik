# Feature 27 — Error Resilience

> **Feature:** Implement 401 token refresh with retry-once, extend 429 backoff to all
> API calls, and extend 403 handling beyond playback.

## Context

The architecture spec (`docs/ARCHITECTURE.md`) defines error handling strategies that
are only partially implemented:

1. **401 handling:** Spec says "refresh token immediately, retry once." Not implemented
   at all — 401 errors are shown as generic errors.
2. **429 handling:** Only playback state polling handles rate limits with backoff.
   Library, search, and device API calls don't handle 429.
3. **403 handling:** Only playback commands check for 403 (Premium required). Library
   and other API calls that require Premium don't surface this.

**Dependency:** This feature should be implemented AFTER Feature 24 (typed errors +
TokenProvider), since it relies on `RateLimitError`, `ForbiddenError`, and
`UnauthorizedError` types and the `TokenProvider` interface for token refresh.

---

## Task 1: Implement 401 token refresh + retry-once

**Problem:** When a token expires mid-session, all API calls fail with 401. The user
must restart the app to re-authenticate.

**Fix:**

1. Create a `RefreshableTokenProvider` in `internal/api/token.go` that wraps the
   keychain-based token storage:
   ```go
   type RefreshableTokenProvider struct {
       tokenStore  keychain.TokenStore
       clientID    string
       mu          sync.Mutex
   }

   func (r *RefreshableTokenProvider) AccessToken(ctx context.Context) (string, error) {
       token, expiry, err := r.tokenStore.GetToken()
       if err != nil {
           return "", err
       }
       // Proactive refresh: 5 minutes before expiry
       if time.Until(expiry) < 5*time.Minute {
           if err := r.refresh(ctx); err != nil {
               // If refresh fails but token isn't expired yet, use it
               if time.Now().Before(expiry) {
                   return token, nil
               }
               return "", fmt.Errorf("token expired and refresh failed: %w", err)
           }
           token, _, _ = r.tokenStore.GetToken()
       }
       return token, nil
   }
   ```

2. In `app.go`, handle `UnauthorizedError` in the result message handlers:
   - On 401 from any API call, attempt token refresh via the token store
   - If refresh succeeds, retry the failed command once
   - If refresh fails, show "Session expired. Run: spotnik auth" in status bar

3. Use `cmd/root.go` to construct `RefreshableTokenProvider` instead of
   `StaticTokenProvider` for production use.

**Files:**
- `internal/api/token.go` — Add `RefreshableTokenProvider`
- `internal/app/app.go` — Handle UnauthorizedError with refresh + retry
- `cmd/root.go` — Use RefreshableTokenProvider

**Tests:**
- Unit test: RefreshableTokenProvider refreshes when token near expiry
- Unit test: RefreshableTokenProvider uses cached token when valid
- Unit test: app handles 401 → refresh → retry flow
- Unit test: app shows error when refresh fails
- Mock the token store for tests

---

## Task 2: Extend 429 backoff to all API calls

**Problem:** Only `fetchPlaybackStateCmd` handles 429 rate limits. If a library fetch,
search, or device fetch hits 429, the error is shown but no backoff occurs, leading
to repeated rate-limited requests.

**Current pattern (playback only):**
```go
case RateLimitedMsg:
    a.backoffTicks = msg.Seconds * ticksPerSecond
    a.store.SetError("Too many requests. Retrying...")
```

**Fix:**
1. Make the `RateLimitedMsg` handling in `Update()` universal — it already applies
   to the tick loop, but verify that ALL `build*Cmd` functions that receive typed
   `RateLimitError` emit `RateLimitedMsg`
2. In each `build*Cmd` function, check for `*api.RateLimitError` and return
   `RateLimitedMsg{Seconds: err.RetryAfter}` instead of a generic error message
3. The existing backoff mechanism (skip fetches while `backoffTicks > 0`) handles
   the rest — it just needs all API calls to feed into it

**Functions to update:**
- `buildFetchPlaylistsCmd`
- `buildFetchAlbumsCmd`
- `buildFetchLikedTracksCmd`
- `buildFetchRecentlyPlayedCmd`
- `buildSearchCmd`
- `buildFetchDevicesCmd`
- `buildFetchStatsCmd`
- `buildFetchPlaylistTracksCmd`

**Files:**
- `internal/app/commands.go` (or `app.go`) — Add RateLimitError checking to all build*Cmd

**Tests:**
- Unit test: library fetch 429 → RateLimitedMsg emitted
- Unit test: search 429 → RateLimitedMsg emitted
- Unit test: backoff applies after any 429, not just playback

---

## Task 3: Extend 403 handling beyond playback

**Problem:** Only playback commands show "Spotify Premium required" on 403. Other API
calls that may require Premium (e.g., adding to queue) show generic errors.

**Fix:**
1. In all `build*Cmd` functions, check for `*api.ForbiddenError`
2. For playback-related 403: show "Spotify Premium required for playback"
3. For other 403 (e.g., private playlist access): show the error message from the
   `ForbiddenError.Message` field
4. Use the status bar for display (existing pattern)

**Files:**
- `internal/app/commands.go` (or `app.go`) — Add ForbiddenError checking

**Tests:**
- Unit test: queue add 403 → appropriate error message
- Unit test: playlist operation 403 → appropriate error message

---

## Acceptance Criteria

- [ ] `RefreshableTokenProvider` exists with proactive 5-minute refresh
- [ ] 401 errors trigger token refresh + single retry
- [ ] Failed refresh shows "Session expired. Run: spotnik auth"
- [ ] All `build*Cmd` functions handle `RateLimitError` → `RateLimitedMsg`
- [ ] 429 from any API call activates the backoff mechanism
- [ ] 403 errors show context-appropriate messages for all features
- [ ] All existing tests pass
- [ ] `make ci` passes
