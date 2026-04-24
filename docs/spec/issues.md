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

---

## CLI auth UX — minor test and style gaps
**Found:** 2026-04-22 | **Source:** PR #189 Review
**Feature:** 09-auth-and-profile

Items to log:
1. `TestRunAuthFlow_writesURLToWriter` uses `time.Sleep(200ms)` to synchronize — fragile on slow CI; goroutine running RunAuthFlow is not terminated by the test. Consider a channel-based sync or accept as known pattern.
2. No structural test verifying `rootCmd.SilenceErrors = true` and `SilenceUsage: true` on all five auth subcommands. A regression here would re-introduce double error printing.
3. `fmt.Fprintln(os.Stderr, "Session expired. Please re-authenticate.")` in `EnsureAuthenticated` (cmd/root.go) is unstyled — inconsistent with the PR's styled CLI output system.
4. `authLogoutCmd` and `authForgetCmd` styled success lines (`✓ Signed out`, `✓ Session ended`) are not captured by any test — the store is wired unconditionally inside the handler, making injection difficult without refactoring.

---

## cliout type design: encapsulation and invariant gaps
**Found:** 2026-04-23 | **Source:** PR #191 Review (type-design-analyzer)
**Feature:** 12-cli-output

Items to log:
1. `cliout.Fixed` is an exported mutable `var` — any import can overwrite the canonical fallback palette. Consider changing to an unexported `fixed` var plus exported `func Fixed() Palette` returning a copy.
2. `Builder.Messages()` returns the live internal slice. External mutation silently corrupts builder state. Should return a copy (like `Recorder.Messages()` already does).
3. `isMessage()` unexported marker method is redundant — `render(Palette) string` being unexported already seals the interface. The marker adds no additional enforcement.
4. `Spinner` and `Prompt` satisfy `Message` but panic from `render()`. Any `[]Message` slice containing either becomes unsafe to pass to `Write`. Consider a narrower `RenderableMessage` interface or change stubs to return a sentinel string instead of panicking.

---

## cliout spinner/prompt: theoretical concurrency and UX gaps
**Found:** 2026-04-23 | **Source:** PR #194 Review (silent-failure-hunter, test-analyzer)
**Feature:** 12-cli-output

Items to log:
1. `sigCh` is written inside `sigOnce.Do` but read in `UninstallSIGINTHandler` without a synchronizing mutex. Sequential in practice (main goroutine), theoretical data race under concurrent use. Fix: protect `sigCh` reads/writes with a mutex or use `atomic.Pointer`.
2. `UninstallSIGINTHandler` calls `signal.Stop(sigCh)` but does not unblock the goroutine blocked on `<-ch`. One goroutine leaks per process invocation (acceptable for CLI, invisible to users but shows in pprof). Fix: add a quit channel to the goroutine select so `UninstallSIGINTHandler` can unblock it cleanly.
3. `validateClientID` uses `hex.DecodeString` which accepts uppercase A–F, but the doc comment says "32 lowercase hex chars". Spotify dashboard always emits lowercase but a mixed-case paste passes validation silently. Fix: add `strings.ToLower(s)` before the hex check (or document the permissive behavior).
4. `TestUninstallSIGINTHandler_nilSigCh` only covers the no-op path (sigCh == nil). The production `signal.Stop(sigCh)` branch is never exercised by tests. Fix: add a test that sets `sigCh` directly and calls `UninstallSIGINTHandler`, asserting no panic.

---

## Story 150 — uikit scaffold minor polish items
**Found:** 2026-04-24 | **Source:** PR #196 Review
**Feature:** 13-tui-design-system

Minor items surfaced in review that are safe to defer; none block the gate story.

Items to log:
1. `internal/uikit/config.go:ResolveMode` doc comment says "inspects LC_ALL then LANG" but the implementation actually checks `LC_ALL`, `LC_CTYPE`, then `LANG`. Align the comment with the code.
2. `PaneBorderFor` keys in `internal/uikit/role.go` use no-underscore pane IDs (`nowplaying`, `likedsongs`, `recentlyplayed`, `toptracks`, `topartists`, `requestflow`, `networklog`) whereas `internal/ui/theme/config_theme.go` uses snake_case TOML keys (`now_playing`, `liked_songs`, …). When S3 (PaneChrome primitive) adds the first caller, introduce a `PaneID.String()` helper (or similar) so both surfaces use the same convention. Today there are no callers so it is not a live bug.
3. `internal/uikit/capture.go` ANSI regex matches only simple CSI sequences (`\x1b\[[0-9;]*[a-zA-Z]`). Primitives that later emit OSC hyperlinks, cursor-visibility controls, 24-bit colon-colour, charset designators, or DCS will leak those bytes into `Capture` output. Broaden the regex (or switch to a dedicated stripansi import) when a primitive first exercises it.
4. `internal/uikit/config.go:ActiveMode()` zero-value defaults to `GlyphUnicode` if `Use()` was never called at startup. Safe today because no production code path calls `ActiveMode()` yet. When S3+ primitives land, either require `Use()` to have been called (panic guard) or reserve a `GlyphUnset` sentinel so an unwired call site is loud, not silent.
5. `internal/uikit/role.go:AllRoles()` is a hand-maintained slice parallel to the `Role*` constant block. Adding a new constant without updating `AllRoles()` silently skips every table-driven test. Consider deriving it from a single source (e.g. a package-private `var allRoles = []Role{...}` that the constants alias) or adding a reflection/count guard test.
6. `internal/uikit/config.go:SetModeForTest` mutates package-global state without a `sync.Mutex`; not race-safe once any uikit test opts into `t.Parallel()`. Also callable from production code (no `testing.TB` parameter). Add a `t testing.TB` argument so misuse is a compile error outside tests.
7. `internal/uikit/config.go:UIConfig.Validate` normalises with `strings.ToLower(strings.TrimSpace(...))` for the comparison but does not write the normalised value back to `c.Glyphs`. Anything that reads `cfg.UI.Glyphs` directly still sees the un-trimmed mixed-case value. Either write back or document "Glyphs is normalised at read-time only".
8. No production call site of `uikit.Use()` yet — `cmd/root.go` wiring lands with the first primitive (S3). Until then `ActiveMode()` always returns the zero-value default. Ensure S3 adds the wiring and a test proves non-UTF-8 envs flip to ASCII end-to-end.

---

## Story 152 — PaneChrome polish follow-ups
**Found:** 2026-04-24 | **Source:** PR #198 Review
**Feature:** 13-tui-design-system

Non-blocking items surfaced during review.

Items to log:
1. `internal/uikit/pane_chrome.go` `Render()` doc comment says content is "pre-sized to (Width-2, Height-2) by the caller". In reality `layout.RenderPaneBorder` pads short content and truncates long content internally. Rewrite the comment to: "Content is padded or truncated to fit the interior dimensions (Width-2, Height-2); callers may pass content of any size."
2. `internal/ui/layout/border.go` `buildRightSegment` rebinds `resolveGlyphs` results through intermediate variables `rsTL/rsTR/rsH` then reassigns to `trCorner/tlCorner/hRule`. Either use the results directly, or pass the already-resolved glyphs from `RenderPaneBorder` (resolved once at line 156) as parameters to `buildRightSegment` to avoid the second call.
3. Filter-preamble *query* (the text inside quotes) is rendered `Muted` instead of `Accent` as the design record role matrix prescribes (spec §7.3). This is pre-existing on `main` (introduced in commit `82610aa`, before feature 13). Triage and fix when the filter UX gets its next polish pass.
4. No PaneChrome test covers ASCII mode + `ToggleKey: 0`. Covered by statement via `WidthAndHeightMatch` but the "no numeric prefix" branch is not behaviourally asserted. Add when next touching the test.
5. `PaneChrome` struct mirrors `BorderConfig` (minus glyph override fields). As S4 (`OverlayChrome`) and later primitives accumulate, consider embedding `layout.BorderConfig` or a shared config type.

---

## Story 151 — border.go test fidelity follow-ups
**Found:** 2026-04-24 | **Source:** PR #197 Review
**Feature:** 13-tui-design-system

Minor polish surfaced in review; none block merge.

Items to log:
1. Filter-mode test in `internal/ui/layout/border_test.go` does not assert the `─` separator between preamble and notch (spec shows `filtering: "query" ─╮`). Add substring assertion for `─╮` when touching this test next.
2. No ASCII-mode test exercises `buildRightSegment`'s filter or actions branches. When a primitive first integrates ASCII mode + filter notch, extend the ASCII test.
3. No test asserts `Esc` key is rendered via `keyHintStyle` (Accent) vs `mutedStyle`. A future color-token swap could regress silently. Add structural ANSI-code assertion when visual-regression tooling lands.
4. Redundant `uikit.ActiveMode()` calls: once in `corners()` and once in `buildRightSegment`. Harmless today (atomic read), but could be deduplicated by passing `mode GlyphMode` into `buildRightSegment` next time the function is touched.
5. Stale test name `TestRenderPaneBorder_ActionPrefixCharacterWidth` in `border_test.go` references `ᐅ` as "action prefix"; the glyph is no longer rendered, so the test measures a banned-glyph width with no production tie. Rename or remove next cleanup pass.
