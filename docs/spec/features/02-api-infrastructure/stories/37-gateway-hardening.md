---
title: "Gateway Hardening"
feature: 11-api-gateway
status: done
---

## Background
PR review of Feature 30 (API Gateway) identified 7 robustness issues ranging from a data race in SetGateway to timer leaks and a nil-pointer panic path. These fixes harden the gateway for production use.

Source: `docs/issues.md` -- PR #35 issues 1-7. Depends on: Nothing (api/ package is independent).

## Design

### Task 1: SetGateway Thread Safety
Change `gateway *Gateway` field in BaseClient to `gateway atomic.Pointer[Gateway]`. Update all reads to use `b.gateway.Load()`.

### Task 2: Timer Leak Prevention
Replace `time.After` with `time.NewTimer` in `tokenBucket.wait()` and `waitForBackoff()` to prevent timer leaks when context is cancelled.

### Task 3: Nil Response Guard
Add nil check after `fn()` call: `if resp == nil && err == nil { err = fmt.Errorf("HTTP transport returned nil response") }`.

### Task 4: io.ReadAll Error Handling
In `doNoContent`, check the `io.ReadAll` error instead of discarding it.

### Task 5: Double 429 Parsing
Unify 429 handling: Gateway.Do() sets backoff but lets checkResponseStatus create the RateLimitError. Extract shared `parseRetryAfter` helper. Clone response body for all responses so dedup waiters get readable bodies.

### Task 6: Retry-After Documentation
Document intentional default for non-integer Retry-After values (HTTP-date format per RFC 7231).

### Task 7: Issues.md Update
Mark all PR #35 issues (1-7) as resolved.

### Verification
```bash
go test -race ./internal/api/...
grep -n 'time\.After' internal/api/gateway.go  # ZERO matches
make ci
```

## Acceptance Criteria
- [ ] SetGateway is thread-safe with atomic.Pointer
- [ ] No time.After in wait functions (use time.NewTimer)
- [ ] Nil response from HTTP transport returns error instead of panic
- [ ] io.ReadAll error handled in doNoContent
- [ ] 429 handling unified between gateway and checkResponseStatus
- [ ] Non-integer Retry-After documented
- [ ] All PR #35 issues marked resolved
- [ ] `make ci` passes

## Tasks
- [ ] Make SetGateway thread-safe with atomic.Pointer in internal/api/base.go
      - test: concurrent SetGateway + doJSON with -race flag; nil-safe gateway Load()
- [ ] Replace time.After with time.NewTimer in internal/api/gateway.go
      - test: context cancellation returns immediately; normal wait completes
- [ ] Add nil response guard after fn() in Gateway.Do()
      - test: (nil, nil) from fn() returns error instead of panicking
- [ ] Handle io.ReadAll error in doNoContent
      - test: doNoContent returns error when body read fails
- [ ] Clean up double 429 parsing -- unify handling, extract parseRetryAfter
      - test: 429 produces consistent RateLimitError for both primary caller and dedup waiter
- [ ] Log unparseable Retry-After header -- add documentation comment
      - test: non-integer Retry-After uses default value
- [ ] Update issues.md -- mark PR #35 issues resolved
      - test: docs change only
