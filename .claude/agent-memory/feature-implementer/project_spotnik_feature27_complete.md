---
name: Feature 27 (Error Resilience)
description: What was built: 429/403/401 error handling extended to all API calls
type: project
---

Feature 27 added comprehensive error resilience across all Spotify API calls.

**What was built:**

- `internal/app/commands.go`:
  - Added `isUnauthorizedError(err)` helper (mirrors `parse429RetryAfter` pattern)
  - Added `buildRefreshTokenCmd(store, clientID, tokenBaseURL)` — calls `api.Refresh`, returns `tokenRefreshedMsg`
  - All 8 `build*Cmd` + `fetchPlaybackStateCmd` + `fetchQueueCmd` now check: 429 → `RateLimitedMsg`, 401 → `unauthorizedMsg{}`
  - Added `keychain` import to commands.go

- `internal/app/app.go`:
  - Added `unauthorizedMsg{}` and `tokenRefreshedMsg{newToken, err}` internal message types
  - Added `tokenBaseURL string` field and `AppOptions.TokenBaseURL` field (for test overrides)
  - `unauthorizedMsg` handler dispatches `buildRefreshTokenCmd`
  - `tokenRefreshedMsg` handler: on success calls `initAPIClients(newToken)`; on failure shows "Session expired. Run: spotnik auth"
  - `AddToQueueResultMsg` error handler now checks `*api.ForbiddenError` and shows `forbiddenErr.Message` directly

**Test file:** `internal/app/error_resilience_test.go` — 13 new tests

**Key patterns:**
- The "401 refresh" is NOT a `RefreshableTokenProvider` type — it's a message/command chain: `unauthorizedMsg{}` → `buildRefreshTokenCmd` → `tokenRefreshedMsg` → `initAPIClients(newToken)`
- `initAPIClients` is reused for re-init after refresh (same as post-auth-flow path)
- All Bubble Tea cmd chains require manually executing each step in tests: cmd() → Update(msg) → cmd() → Update(msg)

**Why:** Feature 24 added typed errors; Feature 25 made BaseClient.checkResponseStatus return them. Feature 27 wires the app layer to use those typed errors across all API paths, not just playback.
