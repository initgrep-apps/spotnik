# Feature 24 — Typed Errors & TokenProvider

> **Refactoring:** Replace string-based HTTP error matching with typed errors, and
> introduce a TokenProvider interface for per-request token resolution.

## Context

Two fragile patterns exist in the codebase:

1. **String-based error matching:** `app.go` uses `strings.Contains(errMsg, "429")` and
   `strings.Contains(errMsg, "403")` to detect rate limits and premium requirements.
   Any change to error message format breaks this silently.

2. **Static token:** All API clients receive `accessToken string` at construction time.
   If the token expires between polls, all requests fail with 401 until the app restarts.
   The architecture spec calls for a `TokenProvider` interface with per-request resolution
   and proactive refresh.

---

## Task 1: Define typed API errors

**Fix:** Create typed error types in `internal/api/errors.go`:

```go
// internal/api/errors.go

// RateLimitError is returned when the Spotify API responds with 429.
type RateLimitError struct {
    RetryAfter int // seconds to wait before retrying
}

func (e *RateLimitError) Error() string {
    return fmt.Sprintf("rate limited: retry after %ds", e.RetryAfter)
}

// ForbiddenError is returned when the Spotify API responds with 403.
type ForbiddenError struct {
    Message string
}

func (e *ForbiddenError) Error() string {
    return fmt.Sprintf("forbidden: %s", e.Message)
}

// UnauthorizedError is returned when the Spotify API responds with 401.
type UnauthorizedError struct{}

func (e *UnauthorizedError) Error() string {
    return "unauthorized: token expired or invalid"
}
```

Update the shared HTTP response checking code (each client has a response checker
or `doJSON`/`doNoContent` helper) to return these typed errors instead of generic
`fmt.Errorf` strings when encountering 401, 403, or 429 status codes.

Parse the `Retry-After` header in the 429 case and embed it in `RateLimitError`.

**Files:**
- `internal/api/errors.go` (new) — Typed error definitions
- `internal/api/player.go` — Update response checking to return typed errors
- `internal/api/library.go` — Same
- `internal/api/search.go` — Same
- `internal/api/devices.go` — Same
- `internal/api/user.go` — Same
- `internal/api/playlists.go` — Same

**Tests:**
- Unit test: API client returns `*RateLimitError` on 429 response with correct RetryAfter
- Unit test: API client returns `*ForbiddenError` on 403 response
- Unit test: API client returns `*UnauthorizedError` on 401 response
- Use `httptest.NewServer` returning the appropriate status codes

---

## Task 2: Replace string matching with errors.As in app.go

**Fix:** Replace the two string-matching patterns in `app.go`:

**Before (line ~667):**
```go
if strings.Contains(errMsg, "403") {
```

**After:**
```go
var forbiddenErr *api.ForbiddenError
if errors.As(err, &forbiddenErr) {
```

**Before (line ~1275, parse429RetryAfter):**
```go
if !strings.Contains(msg, "429") {
```

**After:**
```go
var rateLimitErr *api.RateLimitError
if errors.As(err, &rateLimitErr) {
    return rateLimitErr.RetryAfter
}
```

This requires that `build*Cmd` functions propagate the actual `error` value (not just
an error message string) through result messages. Check how errors flow from commands
to Update handlers and adjust the message structs if needed.

**Files:**
- `internal/app/app.go` (or `commands.go` after Feature 22) — Replace string matching
- Message structs may need `err error` fields instead of string fields

**Tests:**
- Unit test: 429 error triggers backoff with correct RetryAfter value
- Unit test: 403 error shows "Spotify Premium required" message
- Existing tests pass

---

## Task 3: Define TokenProvider interface

**Fix:** Define a `TokenProvider` interface that API clients use to get a fresh token
per-request:

```go
// internal/api/token.go (new)

// TokenProvider resolves an access token for each API request.
// Implementations handle caching, expiry checking, and refresh.
type TokenProvider interface {
    AccessToken(ctx context.Context) (string, error)
}
```

Create a simple implementation that wraps the existing static token for backward
compatibility:

```go
// StaticTokenProvider returns a fixed token. Used during initial construction
// and in tests.
type StaticTokenProvider struct {
    Token string
}

func (s *StaticTokenProvider) AccessToken(_ context.Context) (string, error) {
    return s.Token, nil
}
```

Update all 6 client constructors to accept `TokenProvider` instead of `string`:

```go
func NewPlayer(baseURL string, tp TokenProvider) *Player {
```

Update the `newRequest` / HTTP helper in each client to call `tp.AccessToken(ctx)`
per-request instead of using the stored string.

**Files:**
- `internal/api/token.go` (new) — TokenProvider interface + StaticTokenProvider
- `internal/api/player.go` — Accept TokenProvider, use per-request
- `internal/api/library.go` — Same
- `internal/api/search.go` — Same
- `internal/api/devices.go` — Same
- `internal/api/user.go` — Same
- `internal/api/playlists.go` — Same
- `cmd/root.go` — Construct `StaticTokenProvider` from token string

**Tests:**
- Unit test: client uses TokenProvider per-request (mock provider that tracks calls)
- Unit test: StaticTokenProvider returns the fixed token
- Existing tests updated to pass `StaticTokenProvider` or `&StaticTokenProvider{Token: "test"}`

---

## Acceptance Criteria

- [ ] `RateLimitError`, `ForbiddenError`, `UnauthorizedError` types exist in `api/errors.go`
- [ ] API clients return typed errors for 401, 403, 429 responses
- [ ] `app.go` uses `errors.As` instead of `strings.Contains` for error matching
- [ ] `TokenProvider` interface exists in `api/token.go`
- [ ] All 6 API clients accept `TokenProvider` and call it per-request
- [ ] `StaticTokenProvider` provides backward compatibility
- [ ] All existing tests updated and passing
- [ ] `make ci` passes
