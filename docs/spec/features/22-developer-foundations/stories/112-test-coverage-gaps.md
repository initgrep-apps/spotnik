---
title: "Test Coverage Gaps and Assertion Fixes"
feature: 22-developer-foundations
status: done
---

## Background

Three test quality gaps were identified during PR reviews of Stories 109–111:

1. **`fixtures.go` untested** — `internal/testhelpers/fixtures.go` exports
   `LoadFixture` and relies on `runtime.Caller(0)` to anchor `fixturesDir`.
   The panic branch is dead code and `LoadFixture` itself has no isolation test.
   A future contributor could break the helper silently.

2. **OR assertion in `TestGetDevices_InvalidJSON`** — `devices_test.go` line 165
   uses a disjunctive `strings.Contains(...) || strings.Contains(...)` assertion.
   Per CLAUDE.md error-wrapping rules, error assertions must use
   `assert.ErrorContains` with the most specific expected substring exclusively.
   The OR form allows a broader, less-informative message to pass the test.

3. **`BasePane.HasActiveFilter()` uncovered** — `base_pane.go` line 46 defines
   the shared default behaviour (`return false`) but no `base_pane_test.go`
   exists. The method has 0% coverage in production routing because every pane
   that embeds `BasePane` overrides it. A unit test documents the intentional
   default and prevents regressions.

All changes are test-only; no production code is modified.

**Source:** `docs/spec/issues.md` (PR review findings from Stories 109, 111)

**Depends on:** Story 111 (BasePane must exist before testing it)

---

## Design

### Task 1 — `internal/testhelpers/fixtures_test.go`

Create a new file in package `testhelpers_test` (external test package so it
exercises the exported API):

```go
package testhelpers_test

import (
    "testing"

    "github.com/initgrep-apps/spotnik/internal/testhelpers"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoadFixture_ReturnsNonEmptyContent(t *testing.T) {
    // Uses an existing fixture file that is guaranteed to exist.
    data := testhelpers.LoadFixture(t, "playback_state.json")
    require.NotEmpty(t, data, "fixture file should not be empty")
    assert.Contains(t, string(data), "{", "fixture content should be valid JSON")
}
```

Note: `LoadFixture` calls `require.NoError` which calls `t.Fatal` — the error
path cannot be tested in a normal test without a fake `*testing.T`. Document
the error branch in a comment rather than constructing a mock T. The test must
confirm a known fixture loads non-empty content.

Before writing: confirm which fixture file exists to use as the known input:
```bash
ls testdata/fixtures/ | head -5
```

Verify: `go test ./internal/testhelpers/... -race -count=1 -v` passes.

### Task 2 — Fix `TestGetDevices_InvalidJSON` assertion

**File:** `internal/api/devices_test.go`

Read the assertion at line 165 first to confirm current text, then replace:

```go
// Before:
assert.True(t, strings.Contains(err.Error(), "decoding") || strings.Contains(err.Error(), "getting devices"),
    "expected error to contain 'decoding' or 'getting devices', got: %s", err.Error())

// After:
assert.ErrorContains(t, err, "getting devices")
```

Use `"getting devices"` — it is the outer wrapping added by `GetDevices`,
which is always present regardless of the inner decode error message.
Remove the `strings` import from the file if it is no longer used by any
other assertion.

Verify: `go test ./internal/api/... -run TestGetDevices -race -count=1 -v` → PASS.

### Task 3 — `internal/ui/panes/base_pane_test.go`

Create a test file in `package panes` (internal package, needed to access
`BasePane` fields directly):

```go
package panes

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestBasePane_DefaultBehaviour(t *testing.T) {
    var b BasePane

    assert.False(t, b.IsFocused(), "new BasePane should not be focused")
    assert.False(t, b.HasActiveFilter(), "default HasActiveFilter should return false")
}

func TestBasePane_SetFocused(t *testing.T) {
    var b BasePane

    b.SetFocused(true)
    assert.True(t, b.IsFocused())

    b.SetFocused(false)
    assert.False(t, b.IsFocused())
}

func TestBasePane_SetSize(t *testing.T) {
    var b BasePane
    b.SetSize(80, 24)

    assert.Equal(t, 80, b.width)
    assert.Equal(t, 24, b.height)
}
```

Verify: `go test ./internal/ui/panes/... -run TestBasePane -race -count=1 -v` → PASS.

---

## Acceptance Criteria

- [ ] `internal/testhelpers/fixtures_test.go` exists and passes
- [ ] `TestLoadFixture_ReturnsNonEmptyContent` loads a real fixture and asserts non-empty
- [ ] `TestGetDevices_InvalidJSON` uses `assert.ErrorContains(t, err, "getting devices")`
      exclusively — no `strings.Contains` OR assertion
- [ ] `strings` import removed from `devices_test.go` if no longer needed
- [ ] `internal/ui/panes/base_pane_test.go` exists with at least 3 test functions covering
      default behaviour, `SetFocused`, and `SetSize`
- [ ] `go test ./internal/testhelpers/... ./internal/api/... ./internal/ui/panes/... -race -count=1` passes
- [ ] `make ci` passes

## Tasks

- [ ] Create `internal/testhelpers/fixtures_test.go` with `TestLoadFixture_ReturnsNonEmptyContent`
      - test: `go test ./internal/testhelpers/... -race -count=1 -v` → PASS
- [ ] Fix OR assertion in `TestGetDevices_InvalidJSON` to use `assert.ErrorContains`
      - test: `go test ./internal/api/... -run TestGetDevices -race -count=1 -v` → PASS
- [ ] Create `internal/ui/panes/base_pane_test.go` with default behaviour, SetFocused, SetSize tests
      - test: `go test ./internal/ui/panes/... -run TestBasePane -race -count=1 -v` → PASS
- [ ] `make ci` passes
