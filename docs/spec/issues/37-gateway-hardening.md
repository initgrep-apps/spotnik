# Feature 37 — Gateway Hardening

> **Feature:** Fix thread safety, resource leaks, panic conditions, and error handling
> inconsistencies in the API Gateway.

## Context

PR review of Feature 30 (API Gateway) identified 7 robustness issues ranging from
a data race in SetGateway to timer leaks and a nil-pointer panic path. These fixes
harden the gateway for production use.

**Source:** `docs/issues.md` — PR #35 issues 1-7

**Depends on:** Nothing (api/ package is independent)

---

## Task 1: Make SetGateway thread-safe with atomic.Pointer

**Problem:** `BaseClient.SetGateway` in `internal/api/base.go` (lines 57-59) writes to
`b.gateway` without synchronization. Concurrent `doJSON` calls read `b.gateway` at
line 86, creating a data race during token refresh.

**Fix:**

1. Change `gateway *Gateway` field in BaseClient to `gateway atomic.Pointer[Gateway]`
2. Update `SetGateway` to use `b.gateway.Store(gw)`
3. Update all reads of `b.gateway` to use `b.gateway.Load()`:
   - `doJSON` (line 86)
   - `doJSONOptional` (line ~130)
   - `doNoContent` (line ~165)

**Files:**
- Modify: `internal/api/base.go` — change field type, update all access

**Tests:**
- Unit: concurrent SetGateway + doJSON calls with `-race` flag
- Unit: verify gateway is nil-safe (Load() returns nil before SetGateway called)

**Commit:** `fix(api): make SetGateway thread-safe with atomic.Pointer`

---

## Task 2: Replace time.After with time.NewTimer to prevent leaks

**Problem:** `tokenBucket.wait()` (gateway.go lines 95-100) and `waitForBackoff()`
(gateway.go lines 322-327) use `time.After` which creates timers that leak when the
context is cancelled before the timer fires.

**Fix:**

1. In `tokenBucket.wait()`:
   ```go
   timer := time.NewTimer(waitFor)
   select {
   case <-ctx.Done():
       timer.Stop()
       return ctx.Err()
   case <-timer.C:
       // Loop back and try again.
   }
   ```

2. In `waitForBackoff()`:
   ```go
   timer := time.NewTimer(remaining)
   defer timer.Stop()
   select {
   case <-ctx.Done():
       return ctx.Err()
   case <-timer.C:
       return nil
   }
   ```

**Files:**
- Modify: `internal/api/gateway.go` — both wait functions

**Tests:**
- Unit: verify context cancellation returns immediately (no timer leak)
- Unit: verify normal wait completes after duration

**Commit:** `fix(api): replace time.After with time.NewTimer to prevent leaks`

---

## Task 3: Add nil response guard after fn()

**Problem:** `Gateway.Do()` (gateway.go line 266) assumes `resp != nil` when `err == nil`.
If the HTTP transport returns `(nil, nil)` (edge case), `resp.Body` causes a nil-pointer panic.

**Fix:**

1. Add nil check after `fn()` call:
   ```go
   resp, err := fn()
   if resp == nil && err == nil {
       err = fmt.Errorf("HTTP transport returned nil response")
   }
   ```

**Files:**
- Modify: `internal/api/gateway.go` — add nil guard

**Tests:**
- Unit: verify (nil, nil) from fn() returns error instead of panicking

**Commit:** `fix(api): guard against nil response from HTTP transport`

---

## Task 4: Handle io.ReadAll error in doNoContent

**Problem:** `doNoContent` in `internal/api/base.go` (line ~182) discards the `io.ReadAll`
error with `body, _ := io.ReadAll(resp.Body)`. If the read fails, `checkResponseStatus`
receives an empty or partial body.

**Fix:**

1. Check the error:
   ```go
   body, readErr := io.ReadAll(resp.Body)
   if readErr != nil {
       return fmt.Errorf("reading response body: %w", readErr)
   }
   return checkResponseStatus(resp, body)
   ```

**Files:**
- Modify: `internal/api/base.go` — handle readErr in doNoContent

**Tests:**
- Unit: verify doNoContent returns error when body read fails

**Commit:** `fix(api): handle io.ReadAll error in doNoContent`

---

## Task 5: Clean up double 429 parsing

**Problem:** Both `checkResponseStatus` in `errors.go` (lines 54-59) and `Gateway.Do()` in
`gateway.go` (lines 280-285) parse the `Retry-After` header independently. The gateway
creates a `RateLimitError`, which `doJSON` then wraps with `"sending request:"` prefix.
Dedup waiters get the unwrapped error, creating inconsistency.

**Fix:**

1. In `Gateway.Do()`, when 429 is detected, set the backoff duration but do NOT create
   a `RateLimitError`. Instead, let the response pass through to `checkResponseStatus`
   which already creates a proper `RateLimitError`:
   ```go
   if resp.StatusCode == http.StatusTooManyRequests {
       retryAfter := parseRetryAfter(resp)
       g.setBackoff(retryAfter)
       // Don't create error here — let checkResponseStatus handle it
       // so all callers (including dedup waiters) get consistent errors.
   }
   ```

2. Extract shared `parseRetryAfter(resp)` helper to avoid duplicate parsing logic.

3. Clone the response body for all responses (not just non-429), so dedup waiters
   always get a readable body regardless of status code. This also fixes issue #7
   (429 body not cloned for waiters).

**Files:**
- Modify: `internal/api/gateway.go` — simplify 429 handling, add parseRetryAfter
- Modify: `internal/api/errors.go` — use shared parseRetryAfter if needed

**Tests:**
- Unit: verify 429 response produces consistent RateLimitError for both primary caller and dedup waiter
- Unit: verify dedup waiters can read response body on 429

**Commit:** `fix(api): unify 429 handling between gateway and checkResponseStatus`

---

## Task 6: Log unparseable Retry-After header

**Problem:** Non-integer `Retry-After` values (e.g., HTTP-date format per RFC 7231) are
silently ignored with a 5s default in both `checkResponseStatus` (errors.go lines 54-59)
and `Gateway.Do()` (gateway.go lines 280-285).

**Fix:**

1. In the shared `parseRetryAfter` helper (or wherever the parsing lives after Task 5):
   ```go
   if ra := resp.Header.Get("Retry-After"); ra != "" {
       if v, err := strconv.Atoi(ra); err == nil {
           return v
       }
       // Non-integer Retry-After (possibly HTTP-date format per RFC 7231).
       // Fall through to default — we don't support date-based values.
   }
   return defaultRetryAfterSecs
   ```

   The comment documents the intentional behavior. Since the app doesn't use `log`
   package and all feedback goes through toasts, a code comment is the appropriate
   documentation (not a log.Warn call).

**Files:**
- Modify: `internal/api/gateway.go` or `internal/api/errors.go` — add comment

**Tests:**
- Unit: verify non-integer Retry-After uses default value

**Commit:** `docs(api): document intentional Retry-After default for non-integer values`

---

## Task 7: Update issues.md

**Fix:** Mark all PR #35 issues (1-7) as resolved.

**Files:**
- Modify: `docs/issues.md`

**Commit:** `docs: mark gateway hardening issues as resolved`

---

## Verification

```bash
# Thread safety
go test -race ./internal/api/...
# Expected: PASS

# No time.After in wait functions
grep -n 'time\.After' internal/api/gateway.go
# Expected: ZERO matches

# Nil response guard
grep -n 'nil response' internal/api/gateway.go
# Expected: 1 match

make ci
# Expected: Full pass
```

---

*Depends on: Nothing*
*Blocks: Nothing*
