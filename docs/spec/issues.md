## App-layer priority wiring untested — stale-reconcile regression risk
**Found:** 2026-04-15 | **Source:** PR #163 External Review (pr-test-analyzer)
**Feature:** 26-playback-correctness

No test verifies that the `PlaybackCmdSentMsg` success path dispatches `fetchPlaybackStateCmd`
with `api.Interactive` priority, or that `DeviceTransferredMsg` success does the same.
The two handlers are the only places the bug was fixed at the app layer — if either is
reverted to `api.Background` no CI failure occurs, silently reintroducing the stale-reconcile bug.

Fix: inject a mock `PlayerAPI` that captures the context passed to `PlaybackState()`, send
`PlaybackCmdSentMsg{Err: nil}` through `Update()`, execute the returned command, and assert
`api.PriorityFromContext(capturedCtx) == api.Interactive`. Repeat for `DeviceTransferredMsg`.

---

## Do() doc comment still describes Background dedup for both priorities
**Found:** 2026-04-15 | **Source:** PR #163 External Review (silent-failure-hunter)
**Feature:** 26-playback-correctness

The `Do()` doc comment "Both priorities" bullet still reads: "Check the in-flight map; if a
matching request is already running, wait for it and return the cached result." This is now
incorrect — Interactive requests skip the inflight map entirely. Future debuggers reading the
entry-point doc will form a wrong mental model. The inline Phase 2/Phase 4 code comments were
updated but the outer doc comment was missed.

Fix: split the "Both priorities" section into separate Background and Interactive bullets per
the suggestion in silent-failure-hunter report for #163.

---

## priority parameter and key.Priority can diverge in Do()
**Found:** 2026-04-15 | **Source:** PR #163 External Review (silent-failure-hunter)
**Feature:** 26-playback-correctness

`gateway.Do()` accepts `priority Priority` and `key RequestKey` as independent arguments.
The Phase 2/4 guards use the `priority` parameter; the inflight map uses `key` as the map key.
Both always agree today (both come from `PriorityFromContext` in `base.go`) but there is no
assertion or guard enforcing `priority == key.Priority`. A future caller or test that passes
mismatched values would create silent confusion. Lowest-risk fix: add a brief comment at each
`Do()` call site in `base.go` noting that `priority` and `key.Priority` are always identical.

---

## Stale "100ms debounce" comments in new dedup tests
**Found:** 2026-04-15 | **Source:** PR #163 External Review (pr-test-analyzer)
**Feature:** 26-playback-correctness

Three new tests added by PR #163 (`TestDedup_InteractiveDoesNotJoinBackground`,
`TestDedup_InteractiveDoesNotJoinInteractive`, `TestDedup_BackgroundJoinsBackground`) reference
the interactive debounce hold in comments and sleep timings ("100ms debounce + overhead",
"passes the 100ms debounce hold", etc.). The debounce was removed in PR #164. The sleep delays
still work as synchronization buffers but readers will look for debounce code that no longer exists.

Fix: update the comments and timing rationale in the three tests in `gateway_test.go` to describe
the delays as synchronization buffers, not debounce windows.

---

## errNilClient silent-drop comment is misleading
**Found:** 2026-04-15 | **Source:** PR #164 External Review (silent-failure-hunter)
**Feature:** 26-playback-correctness (pre-existing across handlers.go)

`handlers.go:140` (SearchPageLoadedMsg handler) has comment "errNilClient is a programming error,
not a network result — always surface it" but the code does `return a, nil` (silent drop, no toast).
The same pattern appears across every other `errNilClient` guard in the file. The intent is correct
(treat as expected pre-auth startup condition, not a runtime error) but the comment on the
SearchPageLoadedMsg case actively contradicts the code. All other guards have no comment or a
neutral one.

Fix: change the `SearchPageLoadedMsg` `errNilClient` comment to match the rest:
`// errNilClient is an expected pre-auth startup condition — drop silently`

---

## Stale test comments — fetchPlaybackStateCmd signature
**Found:** 2026-04-15 | **Source:** PR #163 Review
**Feature:** 26-playback-correctness

Two test comments reference the old `fetchPlaybackStateCmd` signature:
- `internal/app/command_safety_test.go:373` — comment says `fetchPlaybackStateCmd(nil)` (old 1-arg form); correct is `fetchPlaybackStateCmd(nil, api.Background)`
- `internal/app/app_test.go:458` — comment says `fetchPlaybackStateCmd(nil, store)` (wrong second arg); correct is `fetchPlaybackStateCmd(nil, api.Background)`

Both are comment-only (non-executable), harmless but misleading to future readers.

---

## Splash test assertions match substring "dev" in "developers"
**Found:** 2026-04-17 | **Source:** PR #170 Review
**Feature:** 11-cicd

The `assert.Contains(t, output, "dev")` assertions in `splash_test.go` and `app_test.go` (TestApp_SplashScreen_ShownOnStartup) can pass vacuously because the tagline "A terminal Spotify client for developers" contains "dev". Use a unique sentinel version string (e.g., "v-test-sentinel") for version rendering tests to avoid ambiguous assertions.

---

## No test for rootCmd.Version wiring in Execute()
**Found:** 2026-04-17 | **Source:** PR #170 Review
**Feature:** 11-cicd

`cmd.Execute(version)` sets `rootCmd.Version = version` to enable `spotnik --version`, but there is no test verifying this wiring. A refactor that drops the assignment would silently break `--version` output. Add a test to `cmd/root_test.go` that calls `Execute("v1.2.3")` and asserts `RootCommand().Version == "v1.2.3"`.

---

## Splash screen small-terminal test missing version assertion
**Found:** 2026-04-21 | **Source:** PR #186 Review
**Feature:** 09-auth-and-profile

`TestRenderSplashView_smallTerminal_noPanic` in `splash_test.go` dropped the `assert.Contains(t, view, "dev")` assertion from the old test. Version string is still present at 40×10 but not asserted.

Fix: Add `assert.Contains(t, view, "dev")` back to `TestRenderSplashView_smallTerminal_noPanic` in `internal/ui/panes/splash_test.go`

---

## Profile overlay: differentKeyAfterFirstPress test missing msg type assertion
**Found:** 2026-04-21 | **Source:** PR #187 Review
**Feature:** 09-auth-and-profile

`TestProfileOverlay_differentKeyAfterFirstPress_cancelsAndArmsNew` verifies `cmd != nil` after pressing `f` following `l`, but doesn't assert the cmd is `ProfileConfirmToastMsg` with "confirm forget" text, unlike its sibling tests.

Items to log:
1. Add type assertion `assert.IsType(t, panes.ProfileConfirmToastMsg{}, msg)` and text check to `TestProfileOverlay_differentKeyAfterFirstPress_cancelsAndArmsNew` in `internal/ui/panes/profile_test.go`

---

## Onboarding copy: onboardingCopied set even when clipboard fails
**Found:** 2026-04-21 | **Source:** PR #188 Review
**Feature:** 09-auth-and-profile

`_ = copyToClipboard(...)` silently discards clipboard errors but `onboardingCopied` is set to `true` regardless, showing "✓ Copied!" even if nothing was actually copied. URL is still visible so UX is acceptable, but accuracy could be improved.

Items to log:
1. In `internal/app/routing.go` (and `auth.go`) copy handler: only set `onboardingCopied = true` on successful clipboard copy, show a different hint on failure.
