---
title: "Typed Errors & TokenProvider"
feature: 10-error-resilience
status: done
---

## Background
Two fragile patterns exist in the codebase: (1) String-based error matching -- app.go uses `strings.Contains(errMsg, "429")` and `strings.Contains(errMsg, "403")` to detect rate limits and premium requirements. Any change to error message format breaks this silently. (2) Static token -- all API clients receive `accessToken string` at construction time. If the token expires between polls, all requests fail with 401 until the app restarts. The architecture spec calls for a TokenProvider interface with per-request resolution and proactive refresh.

## Design

### Task 1: Define typed API errors
Create typed error types in `internal/api/errors.go`:

```go
type RateLimitError struct {
    RetryAfter int // seconds to wait before retrying
}

type ForbiddenError struct {
    Message string
}

type UnauthorizedError struct{}
```

Update the shared HTTP response checking code to return these typed errors instead of generic `fmt.Errorf` strings when encountering 401, 403, or 429 status codes. Parse the `Retry-After` header in the 429 case.

### Task 2: Replace string matching with errors.As
Replace `strings.Contains(errMsg, "403")` with `errors.As(err, &forbiddenErr)` and `strings.Contains(msg, "429")` with `errors.As(err, &rateLimitErr)`. This requires that `build*Cmd` functions propagate the actual error value through result messages.

### Task 3: Define TokenProvider interface
```go
type TokenProvider interface {
    AccessToken(ctx context.Context) (string, error)
}

type StaticTokenProvider struct {
    Token string
}
```

Update all 6 client constructors to accept TokenProvider instead of string. Update the newRequest / HTTP helper in each client to call `tp.AccessToken(ctx)` per-request.

## Acceptance Criteria
- [ ] `RateLimitError`, `ForbiddenError`, `UnauthorizedError` types exist in `api/errors.go`
- [ ] API clients return typed errors for 401, 403, 429 responses
- [ ] `app.go` uses `errors.As` instead of `strings.Contains` for error matching
- [ ] `TokenProvider` interface exists in `api/token.go`
- [ ] All 6 API clients accept `TokenProvider` and call it per-request
- [ ] `StaticTokenProvider` provides backward compatibility
- [ ] All existing tests updated and passing
- [ ] `make ci` passes

## Tasks
- [ ] Define typed API errors in internal/api/errors.go and update all clients
      - test: API client returns *RateLimitError on 429 with correct RetryAfter
      - test: API client returns *ForbiddenError on 403
      - test: API client returns *UnauthorizedError on 401
- [ ] Replace string matching with errors.As in app.go
      - test: 429 error triggers backoff with correct RetryAfter value
      - test: 403 error shows "Spotify Premium required" message
- [ ] Define TokenProvider interface in internal/api/token.go and update all client constructors
      - test: client uses TokenProvider per-request (mock provider that tracks calls)
      - test: StaticTokenProvider returns the fixed token
      - test: existing tests updated to pass StaticTokenProvider
