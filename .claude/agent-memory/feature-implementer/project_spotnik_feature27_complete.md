---
name: Feature 27 (Error Resilience)
description: What was built: 429/403/401 error handling extended to all API calls
type: project
---

Feature 27 add error resilience across all Spotify API calls.

**What was built:**

- `internal/app/commands.go`:
  - Add `isUnauthorizedError(err)` helper (mirrors `parse429RetryAfter` pattern)
  - Add `buildRefreshTokenCmd(store, clientID, tokenBaseURL)` — calls `api.Refresh`, returns `tokenRefreshedMsg`
  - All 8 `build*Cmd` + `fetchPlaybackStateCmd` + `fetchQueueCmd` check: 429 → `RateLimitedMsg`, 401 → `unauthorizedMsg{}`
  - Add `keychain` import to commands.go

- `internal/app/app.go`:
  - Add `unauthorizedMsg{}` + `tokenRefreshedMsg{newToken, err}` internal message types
  - Add `tokenBaseURL string` field + `AppOptions.TokenBaseURL` field (test overrides)
  - `unauthorizedMsg` handler dispatch `buildRefreshTokenCmd`
  - `tokenRefreshedMsg` handler: success → `initAPIClients(newToken)`; failure → show "Session expired. Run: spotnik auth"
  - `AddToQueueResultMsg` error handler check `*api.ForbiddenError`, show `forbiddenErr.Message` directly

**Test file:** `internal/app/error_resilience_test.go` — 13 new tests

**Key patterns:**
- 401 refresh NOT `RefreshableTokenProvider` type — message/command chain: `unauthorizedMsg{}` → `buildRefreshTokenCmd` → `tokenRefreshedMsg` → `initAPIClients(newToken)`
- `initAPIClients` reused for re-init after refresh (same as post-auth-flow path)
- Bubble Tea cmd chains require manual exec each step in tests: cmd() → Update(msg) → cmd() → Update(msg)

**Why:** Feature 24 add typed errors; Feature 25 make BaseClient.checkResponseStatus return them. Feature 27 wire app layer to use typed errors across all API paths, not just playback.