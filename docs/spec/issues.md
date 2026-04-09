
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

---

## Story 110: nowplaying_test.go type-assertion pattern against StateReader
**Found:** 2026-04-09 | **Source:** PR #141 Review
**Feature:** 22-developer-foundations

`nowplaying_test.go` (lines 217, 234, 447, 470) uses `pane.store.(*state.Store).SetPlaybackState(...)` to set up test state. This works because `newTestNowPlayingPane` always passes `*state.Store`, but if a non-`*Store` `StateReader` is ever injected it will panic. A follow-up should introduce a test helper (e.g. `type testStateWriter struct{ *state.Store }`) that exposes write methods for test setup without requiring a type assertion back through the interface.

---

## Story 110: StateReader interface includes 5 staleness methods unused by panes
**Found:** 2026-04-09 | **Source:** PR #141 Review
**Feature:** 22-developer-foundations

`PlaylistsStale`, `AlbumsStale`, `LikedTracksStale`, `RecentlyPlayedStale`, and `DevicesStale` are included in `StateReader` (as spec-required) but are only called by `handlers.go` via the concrete `*Store`, not through the interface. Only `StatsStale` is called through `StateReader` by panes. A future cleanup could remove the unused five from the interface to keep it minimal and self-documenting.

---

## Story 111: BasePane.HasActiveFilter() default never exercised through interface routing
**Found:** 2026-04-09 | **Source:** PR #142 Review
**Feature:** 22-developer-foundations

`BasePane.HasActiveFilter()` returns `false` by default but is never called through the interface — every pane that embeds `BasePane` overrides it. The method has 0% coverage in production routing. A `base_pane_test.go` test asserting `var b BasePane; b.HasActiveFilter() == false` would document the intentional default and prevent regressions if a pane forgets to override.

---

## Story 109: DESIGN.md §26 accessibility bullet implies ? overlay is live
**Found:** 2026-04-09 | **Source:** PR #144 Review
**Feature:** 22-developer-foundations

`docs/DESIGN.md` line 1424 in the Accessibility section reads `` - `?` help always available ``. This implies the help overlay is operational, inconsistent with the §17 `*(PLANNED — not yet implemented)*` annotation added in PR #144. When the help overlay is implemented (story 108), the implementer should update this line and apply the three-location sync rule (§17, `docs/keybinding.md`, `help_overlay.go`).

---

## Story 111: postTokenRequest nil-client produces undescriptive panic
**Found:** 2026-04-09 | **Source:** PR #142 Review
**Feature:** 22-developer-foundations

`postTokenRequest` now accepts `*http.Client` but has no nil guard. A nil client panics at `httpClient.Do(req)` with no attribution. All current callers pass `http.DefaultClient` so this is not a current bug, but the injection pattern makes nil a realistic future mistake. A simple guard (`if httpClient == nil { return TokenPair{}, errors.New(...) }`) would give a clear error message.
