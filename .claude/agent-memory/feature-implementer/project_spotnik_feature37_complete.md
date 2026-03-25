---
name: project_spotnik_feature37_complete
description: Feature 37 (Gateway Hardening): atomic.Pointer, timer leaks, nil guard, doNoContent error, 429 dedup, parseRetryAfter, issues.md
type: project
---

## Feature 37 — Gateway Hardening

**What was built:**
- Changed `BaseClient.gateway` from `*Gateway` to `atomic.Pointer[Gateway]` in base.go. All reads use `.Load()`, writes use `.Store()`. Import is `"sync/atomic"` (no separate atomic package needed).
- Replaced `time.After` with `time.NewTimer` + explicit `Stop()` in both `tokenBucket.wait()` and `waitForBackoff()`. Pattern: create timer before select, call `timer.Stop()` in the `ctx.Done()` case, NOT with `defer`.
- Added nil guard after `fn()` call in `Gateway.Do()`: `if resp == nil && err == nil { err = fmt.Errorf("HTTP transport returned nil response") }`.
- Fixed `doNoContent` to check `io.ReadAll` error: `body, readErr := io.ReadAll(resp.Body); if readErr != nil { return fmt.Errorf("reading response body: %w", readErr) }`.
- Extracted `parseRetryAfter(resp *http.Response) int` shared helper in gateway.go. Added `const defaultRetryAfterSecs = 5`. Updated errors.go to use it (removed inline duplicate parsing + `strconv` import). Body now always cloned for ALL responses (not just non-429) so dedup waiters always get readable body.
- `parseRetryAfter` documents intentional behavior: non-integer Retry-After (HTTP-date format per RFC 7231) falls through to default with a comment.
- Marked all 7 PR #35 issues as resolved in `docs/issues.md`.

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/api/base.go` — atomic.Pointer field, SetGateway Store, doJSON/doJSONOptional/doNoContent Load, doNoContent readErr fix
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/api/gateway.go` — time.NewTimer in wait() and waitForBackoff(), nil guard, always-clone body, parseRetryAfter helper, defaultRetryAfterSecs const
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/api/errors.go` — uses parseRetryAfter, removed strconv import
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/api/gateway_hardening_test.go` — 13 new tests covering all 6 tasks

**Patterns established:**
- `atomic.Pointer[T]` for fields that need concurrent read/write safety without a mutex (e.g., gateway pointer).
- Timer leak prevention: `timer := time.NewTimer(d); select { case <-ctx.Done(): timer.Stop(); return err; case <-timer.C: ... }` — explicit Stop() in cancel branch, NOT defer in for-loop.
- Shared helper function pattern: extract parseRetryAfter so gateway.go and errors.go share one implementation.
- Always-clone body pattern: clone resp.Body for ALL gateway responses (not just non-429) so downstream consumers and dedup waiters always get a fresh reader.

**Gotchas:**
- `defer timer.Stop()` inside a for-loop is not idiomatic even when safe (function returns before next iteration). Review caught this — use explicit `timer.Stop()` in the cancel case.
- `sync/atomic` is the correct import for `atomic.Pointer[T]` in Go 1.19+. No separate `atomic` package needed.
- Test comment mismatch: function was renamed from `DedupWaiterCanReadBody` to `PrimaryAndWaiterConsistentError` but comment text wasn't updated. PR review caught this.
- `TestDoNoContent_ReadAllError` hardcodes port 19999 but this is safe because `fixedResponseTransport` intercepts before network. Could use `http://test-only.invalid` instead to make intent clearer but not blocking.
- The `"sending request:"` wrapper in `doJSON` wraps `RateLimitError` from gateway — this is pre-existing and `errors.As` works through `%w` wrapping so app.go handlers are not affected.

**Testing notes:**
- 13 tests in `gateway_hardening_test.go`
- Must run with `-race` flag to verify Task 1: `go test -race ./internal/api/...`
- Coverage: 84.7% on `internal/api`, 83.1% total
- PR: https://github.com/initgrep-apps/spotnik/pull/42
