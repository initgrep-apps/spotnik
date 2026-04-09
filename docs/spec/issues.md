
---

## Story 109: PANE-TEMPLATE.md coverage gap in testhelpers
**Found:** 2026-04-09 | **Source:** PR #140 Review
**Feature:** 22-developer-foundations

`internal/testhelpers/fixtures.go` has no accompanying test file. The package-level var `fixturesDir` uses `runtime.Caller(0)` which is reliable in practice but the panic branch is dead code and LoadFixture itself is untested in isolation. A future story should add `internal/testhelpers/fixtures_test.go` with a test that loads a known fixture and asserts non-empty content.

---

## Story 109: TestGetDevices_InvalidJSON uses OR error assertion
**Found:** 2026-04-09 | **Source:** PR #140 Review
**Feature:** 22-developer-foundations

`internal/api/devices_test.go` `TestGetDevices_InvalidJSON` uses `strings.Contains(err.Error(), "decoding") || strings.Contains(err.Error(), "getting devices")`. Per CLAUDE.md error wrapping rules, the assertion should be `assert.ErrorContains(t, err, "getting devices")` exclusively. Minor but worth fixing in a follow-up.
