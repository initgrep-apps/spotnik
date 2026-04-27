---
name: project_spotnik_feature37_complete
description: Feature 37 (Gateway Hardening): atomic.Pointer, timer leaks, nil guard, doNoContent error, 429 dedup, parseRetryAfter, issues.md
type: project
---

## Feature 37 — Gateway Hardening

**What was built:**
- Changed `BaseClient.gateway` from `*Gateway` to `atomic.Pointer[Gateway]` in base.go. Reads use `.Load()`, writes `.Store()`. Import `"sync/atomic"` (no separate atomic pkg).
- Replaced `time.After` w/ `time.NewTimer` + explicit `Stop()` in `tokenBucket.wait()` and `waitForBackoff()`. Pattern: create timer pre-select, call `timer.Stop()` in `ctx.Done()` case, NOT `defer`.
- Added nil guard post `fn()` in `Gateway.Do()`: `if resp == nil && err == nil { err = fmt.Errorf("HTTP transport returned nil response") }`.
- Fixed `doNoContent` to check `io.ReadAll` err: `body, readErr := io.ReadAll(resp.Body); if readErr != nil { return fmt.Errorf("reading response body: %w", readErr) }`.
- Extracted `parseRetryAfter(resp *http.Response) int` shared helper in gateway.go. Added `const defaultRetryAfterSecs = 5`. Updated errors.go to use it (dropped inline dup parsing + `strconv` import). Body now always cloned for ALL responses (not just non-429) so dedup waiters get readable body.
- `parseRetryAfter` docs intentional behavior: non-integer Retry-After (HTTP-date per RFC 7231) falls through to default w/ comment.
- Marked all 7 PR #35 issues resolved in `docs/issues.md`.

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/api/base.go` — atomic.Pointer field, SetGateway Store, doJSON/doJSONOptional/doNoContent Load, doNoContent readErr fix
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/api/gateway.go` — time.NewTimer in wait() and waitForBackoff(), nil guard, always-clone body, parseRetryAfter helper, defaultRetryAfterSecs const
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/api/errors.go` — uses parseRetryAfter, dropped strconv import
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/api/gateway_hardening_test.go` — 13 new tests, all 6 tasks

**Patterns established:**
- `atomic.Pointer[T]` for fields needing concurrent read/write safety sans mutex (e.g., gateway ptr).
- Timer leak prevent: `timer := time.NewTimer(d); select { case <-ctx.Done(): timer.Stop(); return err; case <-timer.C: ... }` — explicit Stop() in cancel branch, NOT defer in for-loop.
- Shared helper pattern: extract parseRetryAfter so gateway.go + errors.go share one impl.
- Always-clone body: clone resp.Body for ALL gateway responses (not just non-429) so consumers + dedup waiters get fresh reader.

**Gotchas:**
- `defer timer.Stop()` in for-loop not idiomatic even when safe (fn returns pre-next iter). Review caught — use explicit `timer.Stop()` in cancel case.
- `sync/atomic` is correct import for `atomic.Pointer[T]` Go 1.19+. No separate `atomic` pkg.
- Test comment mismatch: fn renamed `DedupWaiterCanReadBody` → `PrimaryAndWaiterConsistentError` but comment not updated. PR review caught.
- `TestDoNoContent_ReadAllError` hardcodes port 19999 — safe, `fixedResponseTransport` intercepts pre-network. Could use `http://test-only.invalid` for clearer intent, not blocking.
- `"sending request:"` wrapper in `doJSON` wraps `RateLimitError` from gateway — pre-existing, `errors.As` works thru `%w` wrap so app.go handlers unaffected.

**Testing notes:**
- 13 tests in `gateway_hardening_test.go`
- Run w/ `-race` to verify Task 1: `go test -race ./internal/api/...`
- Coverage: 84.7% `internal/api`, 83.1% total
- PR: https://github.com/initgrep-apps/spotnik/pull/42