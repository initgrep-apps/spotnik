## Story 158 — LockedRow in bubble-table cells: per-row foreground not supported
**Found:** 2026-04-24 | **Source:** PR #204 Review
**Feature:** 13-tui-design-system

`LockedRow.Render()` embeds full ANSI styling (Muted foreground for glyph and label).
In bubble-table cell context, the per-column foreground pass (`applyRows`) overwrites ANSI
colour applied inside the cell string, so the Muted role is effectively discarded. `LockedRow`
therefore uses `PlainText()` for table-cell contexts, which emits no ANSI, letting bubble-table's
column colour apply normally. The locked glyph (`◌`) is still visible; the entire-row Muted
foreground styling cannot be applied without per-row foreground support in bubble-table.

Fix (future): If bubble-table adds per-row foreground (via `WithRowStyleFunc` keyed on cell data,
or a dedicated row metadata field), wire Spotify-owned playlist rows through that API and restore
the full Muted role. Until then, the glyph-only distinction is the documented constraint.

---

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

## Story 153 — OverlayChrome polish follow-ups
**Found:** 2026-04-24 | **Source:** PR #199 Review
**Feature:** 13-tui-design-system

Non-blocking items.

Items to log:
1. Two stale references to the deleted `renderWith{Theme,Device,Profile,Search,Help}Overlay` helper names remain in `internal/app/app_test.go` comments (~lines 1081, 1095). Comment-only, non-load-bearing; rename to `renderWithOverlayChrome` next time that file is touched.
2. `internal/uikit/overlay_chrome_test.go` uses `uikit.Capture` + `len([]rune(...))` to measure visible width while `pane_chrome_test.go` uses `lipgloss.Width(l)` directly. Both correct, but harmonise once a helper like `uikit.VisibleWidth` lands.
3. `TestRender_ThemeOverlay_Composited` assertion was adapted from `"Themes"` → `"Gruvbox"` because the default test-app height truncates the title row after overlay composition. The parallel `TestRenderWithOverlayChrome_Composited` restores the title assertion with a full-height background. Consolidate both when visual-regression tooling lands.
4. Spec §"Rendering" says "The individual overlay panes (SearchOverlay, DeviceOverlay, etc.) compose their own bodies and pass them into OverlayChrome.Render(content)." This PR creates the primitive + consolidates the render helpers but does not wire overlay panes to use OverlayChrome for their bodies. Handle in a later story when each overlay pane is migrated.

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

---

## Story 155 — HeaderBar polish follow-ups
**Found:** 2026-04-24 | **Source:** PR #201 Review
**Feature:** 13-tui-design-system

Non-blocking.

Items to log:
1. `internal/uikit/header_bar.go` applies bold via raw `lipgloss.NewStyle().Foreground(t.TextPrimary()).Bold(true)` rather than `uikit.Apply(RoleStrong, ...)`. Functionally equivalent but breaks the Role abstraction. Harmonise when touching next.
2. `TestRenderHeader_NoProfileChip_EmptyWhenNotLoaded` in `internal/app/render_test.go` uses `NotContains` for profile glyphs instead of the stronger `assert.Empty(chip)` the deleted direct test used. Tighten if a future regression sneaks another profile indicator in.
3. `TestHeaderBar_ZeroWidth_FallsBack` asserts `NotEmpty` + "spotnik" present but does not pin the exact `"  "` double-space fallback documented in the doc comment.
4. `renderHeader` doc comment no longer explains the role shift (page key now Accent, not KeyHint; device glyph now Info, not HeaderChipFg). Add a one-line note pointing at `header_bar.go` for future readers.

---

## Story 156 — StatusBar/KeyBar follow-ups
**Found:** 2026-04-24 | **Source:** PR #202 Review
**Feature:** 13-tui-design-system

Non-blocking.

Items to log:
1. `StatusBar` initially shipped with `theme.Info()` for key colour (confused with `KeyHint`) — caught in review round 1 and fixed. Add a uikit lint/test that asserts `Info()` is NEVER used for key-role styling in any primitive, so this class of mistake cannot recur.
2. `StatusBarBg` role from design-record §6 is intentionally NOT applied in `StatusBar.Render()` to preserve the pre-migration visual (terminal-default body, muted accent border). If/when the design record switches to require a background fill, apply it in `StatusBar.Render()` and add a structural test.
3. `TestStatusBar_RoleTokens` hex-to-RGB conversions are hardcoded in comments. Refactor to compute the ANSI escape from the theme hex at test time so theme TOML edits don't silently invalidate the anti-colour assertion.
4. `newStatusHelp()` dead code was removed in the same PR. Audit `internal/app/app.go` and `handlers.go` for other `help.Model` usages that the bubbles/help refactor may have orphaned.

---

---

## Story 159 — SectionLabel follow-ups
**Found:** 2026-04-25 | **Source:** PR #205 Review
**Feature:** 13-tui-design-system

Non-blocking.

Items to log:
1. Spec line 43 says SectionLabel inherits parent pane's border token. Implementation only uses `PaneBorderRequestFlow()` for GATEWAY and GATEWAY LOG; APP/AUTO-TRAFFIC keep `ColumnPrimary()`, SPOTIFY keeps `Success()`. Preserves pre-PR colour assignments. Update spec or refactor when call site colour rules are revisited.
2. `renderSectionColumn` content lines and the GATEWAY banner / AUTO-TRAFFIC strip both rely on `layout.TruncateOrPad` to enforce width. Consider lifting that pad logic into the SectionLabel primitive (`SectionLabel.RenderWithContent(lines []string)`) so callers don't repeat width handling.

---

## Story 158 — ListRow/LockedRow follow-ups
**Found:** 2026-04-25 | **Source:** PR #204 Review (3 rounds)
**Feature:** 13-tui-design-system

Non-blocking; surfaced during round-1/2/3 review.

Items to log:
1. `LockedRow.PlainText(width)` exists as a no-ANSI variant for bubble-table cell content because the table column foreground overwrites our ANSI. Document the constraint in the design record §6.2 + `docs/TUI-DESIGN-SYSTEM.md` (S168) so future readers know full-row Muted is reserved for non-table contexts.
2. `playlists_pane.refreshPlaylistRows` passes `p.width` (pane width) as `nameWidth` to `LockedRow.PlainText`, padding well beyond the actual column. Bubble-table truncates so it's benign, but pass the actual column width when convenient.
3. `profile.go renderActions` doc comment claims it matches the pre-PR layout exactly. Pre-PR started at column 2 (`"  l  Logout"`); the new layout starts at column 0. Visually fine but the comment overstates equivalence.
4. `RowBackground` exists on ListRow. Consider extending it to LockedRow for symmetry, in case a future call site renders LockedRow inside a hover/selection context.
5. `TestThemeOverlay_CursorRow_UsesSelectedBg` was tightened to assert bg appears in the SGR run immediately before the label. Consider extracting that assertion as a uikit test helper (`uikit.AssertBgImmediatelyBeforeText(t, raw, label, bg)`) so other primitive tests can reuse it.

---

## Story 160 — EmptyState follow-ups
**Found:** 2026-04-25 | **Source:** PR #206 Review
**Feature:** 13-tui-design-system

Non-blocking.

Items to log:
1. `EmptyState` hardcodes `e.Theme.TextMuted()` instead of going through `uikit.Apply(RoleMuted, theme)`. Other primitives use the Role abstraction. Harmonise when next touching this file.
2. Role assertions in `empty_state_test.go` only check ANSI presence, not the specific TextMuted colour. ListRow tests use a stronger pattern (extract foreground ANSI, compare to theme colour). Tighten when next extending these tests.
3. The PR added a Makefile fix to exclude `.gocache/` and `.gomodcache/` from `fmt-check`. Document in CONTRIBUTING or developer notes if not already covered.

---

## Story 161 — URLBox follow-ups
**Found:** 2026-04-25 | **Source:** PR #207 Review
**Feature:** 13-tui-design-system

Non-blocking.

Items to log:
1. Story spec line 40 says wrap at "the last `&` in the first half" — actual implementation (and the existing `wrapURL` helper at `internal/app/render.go:98-120`) cuts when `i > width/2` (the second half). The code is correct; the spec wording is a typo. Fix the spec text in the docs rewrite (S168) at `docs/superpowers/specs/2026-04-24-tui-design-system-design.md` §7.6 and the story spec.
2. `TestURLBox_AmpersandInFirstHalf` in `internal/uikit/url_box_test.go` has a self-contradictory inline comment ("i=3 <= 8 is true, but 3 > 8 is false"). Assertions are correct; only the comment is muddled. Tidy when next touching.
3. Three role/ANSI tests (`TestURLBox_AccentAnsiPresent`, `TestURLBox_RoleBorderMuted`, `TestURLBox_RoleContentAccent`) all assert the same `\x1b[` substring. Strengthen to assert exact `string(th.Accent())` / `string(th.TextMuted())` colour tokens — apply the same pattern across other primitives (EmptyState, Chip, etc.) when next extended.

---

## Story 162 — Toast follow-ups
**Found:** 2026-04-25 | **Source:** PR #208 Review
**Feature:** 13-tui-design-system

Non-blocking.

Items to log:
1. `internal/app/app_test.go:1993` retains lowercase `"resuming in"` in an `assert.NotContains`. Migrated body uses capital `"Resuming in"`. Substring is case-sensitive — even if the toast erroneously rendered the assertion would not catch it. Tighten or `strings.ToLower` the comparison next time the test is touched.
2. `internal/uikit/toast_test.go:209-213` hardcodes hex values (`#1db954` etc.) when constructing `bubbleup.AlertDefinition` for unit tests. Acceptable in test-only code but consider exposing a `uikit.IntentColor(intent, theme)` helper so the same colours come from the theme rather than literals.
3. `bubbleup` style mapping happens inside `ToastManager.Cmd`. When the bubbleup library exposes per-intent style hooks more directly, simplify the manager.

---

## Story 163 — StatusGlyph follow-ups
**Found:** 2026-04-25 | **Source:** PR #209 Review (2 rounds)
**Feature:** 13-tui-design-system

Non-blocking.

Items to log:
1. Initial PR shipped with single-space separator that broke alignment with adjacent `✓  ` rows in onboarding/splash. Fixed via `Gap` field in round 2. Watch for similar spacing assumptions when migrating future call sites — visual diffs > textual contains-checks.
2. `StatusGlyph.Render()` unknown-role fallback uses `GlyphInfo` for the glyph but the colour resolves via `ColourFor(role)` which falls back to `TextPrimary` (NOT `RoleInfo` colour). Doc comment says "Unknown roles fall back to RoleInfo" — semantically inconsistent. Either change the colour fallback to `RoleInfo` or update the doc comment to say "glyph falls back to GlyphInfo; colour falls back to TextPrimary".
3. `internal/uikit/main_test.go` `stripANSI` helper is now reused across multiple test files. Promote to a uikit-internal test util when next touched.

---

## Story 164 — ProgressBar follow-ups
**Found:** 2026-04-25 | **Source:** PR #210 Review
**Feature:** 13-tui-design-system

Non-blocking.

Items to log:
1. Story spec line 51-52, 72-74 reference `internal/ui/components/controls.go`. Actual seek/volume rendering lives in `internal/ui/components/gradient.go`. Code migrated `gradient.go` correctly. Fix spec text in S168 docs rewrite.
2. `TableChrome` (added by S157 in `internal/uikit/`) was relocated to `internal/ui/components/` to break a planned import cycle when `components → uikit` was introduced for ProgressBar. Had no external callers — no-op for users. Document the package boundary rule (`uikit` cannot import `components`; `components` may import `uikit`) in S168.
3. `ProgressBar.Render()` paints empty cells with `Theme.TextMuted()`. Existing `GradientSeekBar`/`GradientVolumeBar` paint empty cells with `Theme.Surface()`. Bars currently only borrow `PartialGlyph`/`GlyphFor` helpers, not `ProgressBar.Render()`, so no regression today. When a future migration swaps in `ProgressBar.Render()`, decide which colour wins and harmonise.
4. Volume-bar dead-zone (vol=0) was removed in this PR. Verify no UX regression once playback ships under non-trivial use.

---

## Story 165 — Spinner follow-ups
**Found:** 2026-04-25 | **Source:** PR #211 Review
**Feature:** 13-tui-design-system

Non-blocking.

Items to log:
1. Spec line 50 says wiring lives in `internal/app/auth.go`, but actual OAuth handlers live in `internal/app/handlers.go`. Implementation correctly follows the real handler location. Fix spec wording during S168 docs rewrite.
2. Spec Background lines 17-19 mention an error toast on failure; explicit Design block (lines 74-77) returns `a, nil` with no toast. Reconcile during S168 — pick one and update both.
3. Spinner success-path tests (`TestSpinner_Done_EmitsMsgAfterTTL` etc.) execute `tea.Tick` cmds directly, blocking ~6.7s total. Consider asserting cmd-type-with-payload via synthetic msgs for faster tests.
4. `Spinner.Fail("Authorization failed")` puts the hold-text into `SpinnerFailMsg.Err`, not an underlying error string. Field name suggests error. Either rename `Err` to `Text` or have `Fail` accept a separate error string.
5. `authSuccessMsg` resolves `onboardingSpinner.Done` unconditionally — adds a 1.2s hold to the legacy `viewAuth` path too (where the spinner isn't rendered). Toast still fires on `SpinnerDoneMsg`. Acceptable but worth documenting.

---

## Story 166 — FormField follow-ups
**Found:** 2026-04-25 | **Source:** PR #212 Review (2 rounds)
**Feature:** 13-tui-design-system

Non-blocking.

Items to log:
1. Initial PR painted the validation error message in `Error()` colour; round 1 fixed to glyph=Error, text=TextPrimary per spec. Add a uikit test helper that asserts a rendered field contains exactly N distinct foreground escapes — promotes the regression-detection pattern across primitives.
2. `InputTextStyle()` / `InputCursorStyle()` accessors were added to support testing role wiring without parsing ANSI. Consider whether other primitives (Chip, ListRow, StatusGlyph) need similar accessors so role contracts are unit-testable instead of asserted via raw bytes.
3. Behaviour change: pressing a key (other than Enter) no longer clears the cached validation error; only `SetValue` or `Validate` does. Spec is the source of truth, but a stale `✗` glyph can persist while the user backspaces. Consider clearing the cached error on every `Update()` keypress in a follow-up if it surfaces in QA.
4. Visible label changed from "Paste your Client ID here:" to "Client ID:". Border colour shifted from `ActiveBorder()` to `Accent()`. Both intentional; document the visual change in S168 onboarding screenshots.

---

## Story 167 — Onboarding rewrite follow-ups
**Found:** 2026-04-25 | **Source:** PR #213 Review
**Feature:** 13-tui-design-system

Non-blocking.

Items to log:
1. Initial assertion pattern for "Step 1 of 2" matched both old and new implementations. Round 1 strengthened to assert `╭─ Step 1 of 2` (border-glyph prefix) so tests detect Panel composition vs raw lipgloss border. Apply same pattern at any future Panel-based render assertions.
2. `wrapURL` helper still used by `auth.go:renderAuthPanel`. When the re-auth flow migrates to URLBox in a follow-up, drop `wrapURL`.
3. `panelInnerWidth=72` and `panelW=max(width-4, 80)` floors paired correctly. Document in S168 the rationale so future panel callers can derive widths consistently.
4. Manual visual smoke test (spec lines 71-76 — copy toast, spinner Done hold, error panel intent) should be performed before the next user-facing release.

---

## Story 168 — Design system docs rewrite follow-ups
**Found:** 2026-04-25 | **Source:** PR #214 Review (3 rounds)
**Feature:** 13-tui-design-system

Non-blocking; design + implementation gaps surfaced during review.

Items to log:
1. `internal/ui/components/notifications.go` registers bubbleup alerts with hardcoded unicode prefixes (`✓ ✗ ◬ → ⧖`) — they do NOT switch to ASCII when `uikit.ActiveMode() == GlyphASCII`. Toast.go's `ToastGlyph(intent, mode)` API does honour mode, so the inconsistency is at the bubbleup registration layer. Wire bubbleup prefixes via `ToastGlyph(intent, ActiveMode())` next time the notification model is touched.
2. `bubbleup` renders the toast as a plain prefix + colored text — there's no bordered toast box. The TUI-DESIGN-SYSTEM.md examples were softened in round 2 to acknowledge this. If a future revision adds a true bordered toast (per the original §7.4 design intent), update the doc again.
3. ProgressBar primitive uses Gradient1() only. `internal/ui/components/gradient.go` keeps its own per-position gradient via `PartialGlyph + GlyphFor`. Either expand ProgressBar to natively support per-position gradient, or keep the two surfaces as documented.
4. SectionLabel rule line uses AccentColor (parent pane's border). The role row in §3.7 was Muted; corrected in round 2. If/when SectionLabel gets a stand-alone use outside Page B sub-sections, confirm the AccentColor parameter still makes sense.
5. StatusBar intentionally omits `StatusBarBg()` — body is terminal-default. If a future visual revision wants a fill, apply it in `status_bar.go` AND update §3.11 / §5.2 / §5.3 in the same commit.

---

## Story 171 — PanelSize sizing helper follow-ups
**Found:** 2026-04-26 | **Source:** PR #218 Review
**Feature:** 13-tui-design-system

Non-blocking.

Items to log:
1. `docs/TUI-DESIGN-SYSTEM.md §3.3` Panel section could optionally cross-reference `PanelSize` as the canonical sizing policy for Panel callers, so future readers know where to find the 70%/65% policy.
2. No test in `render_test.go` asserts the rendered panel's actual width at a known terminal size. A future regression reverting to `a.width - 4` would not be caught by content-string assertions alone. Add a test asserting `lipgloss.Width(borderLine) == panelW` at a known terminal width (e.g. 200 → 140) when next touching render_test.go.

---

## Story 173 — Esc scroll-reset: filter-close tests missing in 5 panes
**Found:** 2026-04-26 | **Source:** PR #221 Review
**Feature:** 14-page-b-redesign

Items to log:
1. TopTracks, TopArtists, LikedSongs, RecentlyPlayed, and NetworkLog panes each lack an explicit `Filter_EscCloses` test asserting that pressing Esc while the filter is active closes the filter only and does NOT reset scroll. QueuePane has `TestQueuePane_Filter_EscCloses` covering the pattern. The implementation is correct (the `GotoTop()` call sits after the filter-active early-return branch), but an ordering regression (moving `GotoTop()` above the guard) would not be caught by CI in those 5 panes. Add one test per pane following the QueuePane pattern.

---

## Story 175 — Gateway/Polling pane threshold tests don't isolate color logic
**Found:** 2026-04-27 | **Source:** PR #223 Review (round 2)
**Feature:** 14-page-b-redesign

Items to log:
1. `TestGatewayHealthPane_Threshold_Tokens`, `_Threshold_Slots`, `_Threshold_Backoff`, `_FreshPane_NotWarningColor`, and `TestPollingTrafficPane_CacheStale_WarningVsError` use `assert.NotEqual` between two `View()` outputs whose visible content (bar-fill counts, text strings) differs in addition to color. Empirically verified: deleting the warning-threshold block in `gateway_health_pane.go` still leaves these tests passing. A regression that drops a warning color while leaving the data correct would slip through. Fix: extract a `resolveTokenColor`-style helper that returns the resolved color for a snapshot and unit-test that helper directly, OR assert specific ANSI escape sequences (e.g., `strings.Contains(view, warningANSI)` for the warning view but NOT the healthy view).
2. `TestGatewayHealthPane_Update_DrainsCursor` is misnamed — it verifies "the latest event's snapshot wins after each tick" (which users care about) but does not actually prove cursor advancement. Mutating `drainEvents` to read from cursor 0 every tick would not break this test. To actually catch a cursor bug, add a test that records 3 events between two ticks to verify no events are dropped and that re-reading at the same cursor yields no events.

---

## Story 176 — GatewayLivePane SetTheme has 0% test coverage
**Found:** 2026-04-27 | **Source:** PR #224 Review (round 2)
**Feature:** 14-page-b-redesign

Items to log:
1. `GatewayLivePane.SetTheme()` rebuilds the table, recreates the filter, and re-renders rows — non-trivial logic at 0% line coverage. Add a smoke test that calls `SetTheme(theme.Load("midnight"))` after seeding events and asserts the buffer count survives and `View()` still renders rows. Theme switching is a runtime feature (Feature 16) so this matters.

---

## Story 178 — Universal filter UX: test coverage gaps
**Found:** 2026-04-27 | **Source:** PR #226 Review
**Feature:** 14-page-b-redesign

Items to log:
1. The 8 new `TestXxxPane_Esc_ClearsCommittedFilter` tests assert only that `ActiveFilterQuery()` returns `""` after Esc — they do not verify that previously-filtered rows reappear in `View()`. If `refreshRows()` is dropped after `ClearQuery()`, the filter query clears but the table stays narrowed. Fix: add `assert.Contains(t, pane.View(), "Jazz")` (or equivalent non-matching row name) to each test, following the `GatewayLivePane` `TestGatewayLivePane_CommittedFilter_ClearedByEsc` pattern.
2. `TestFilter_ClearQuery_ResetsQueryWithoutDeactivating` does not assert the "without deactivating" contract in the active→ClearQuery direction. It only calls `ClearQuery()` when `IsActive()` is already false. Add a parallel assertion that calls `ClearQuery()` on an active filter (toggled but no Enter) and asserts `IsActive()` remains `true`.
3. `docs/DESIGN.md` pane walkthrough text (~line 1139 for NetworkLog) still says "Esc resets scroll to page 1" without mentioning the filter-clear step. The canonical keybinding tables (§17 and `docs/keybinding.md`) are correct; this is only the descriptive prose. Update when next touching the design doc.

---

## Story 179 — Page B toggle keys: minor test coverage gaps
**Found:** 2026-04-27 | **Source:** PR #227 Review
**Feature:** 14-page-b-redesign

Items to log:
1. Keys `'6'`-`'8'` on Page B intentionally map to nothing (not in `pageBToggleKeyMap`), but no routing-level test asserts this no-op. A future change that mistakenly extends `pageBToggleKeyMap` to `'6'`-`'8'` would not be caught at the routing layer (the preset-membership guard in `TogglePane` would catch it downstream). Add a test asserting `'6'` on Page B does not change any pane visibility.
2. `TestTogglePane_PageB_IgnoresPageAPanes` calls `TogglePane(PaneQueue)` on Page B then asserts `IsPaneVisible(PaneNowPlaying)` — an indirect check because `PaneQueue` is never visible on Page B regardless of toggles. Add a comment to the test explaining why the direct assertion is not expressible via the public `IsPaneVisible` API.

---

## Story 180 — Stacked Page B layout: RowSpan edge cases
**Found:** 2026-04-27 | **Source:** PR #228 Review (round 2)
**Feature:** 14-page-b-redesign

Items to log:
1. Focus order in spanner rows places the spanner (`GatewayLive`) before its left sibling (`GatewayHealth`) in row 1. The `focusOrder` doc comment says "row-by-row, left-to-right" but spanner cells are added at origin row in the loop before non-spanner cells. Harmless in practice (Tab cycles through all panes) but the ordering is slightly unintuitive. Reorder if the doc comment is ever enforced.
2. Step 4's non-spanner placement loop has a latent edge case: if a continuation row contains 2+ non-spanner cells that straddle a reserved spanner interval, the proportional `w = available * weight / totalWWeight` for a non-last cell could place it overlapping the reserved interval (the `nextFreeX` call is at the top of the loop, but the proportional `w` may extend into the reserved zone). `PresetNerdStatus` never exercises this (continuation rows have at most one non-spanner cell), but a future preset with multiple non-spanner cells in a continuation row should be tested explicitly.

> **Resolved by story 181 (PR #230, 2026-04-28):** RowSpan retired entirely; recompute simplified to a flat two-loop algorithm. Both items above are no longer applicable.

---

## Story 181 — TableBasedPane refactor: minor follow-ups

**Found:** 2026-04-28 | **Source:** PR #230 Review (round 1)
**Feature:** 14-page-b-redesign

Items to log:

1. `NewTableBasedPane` accepts `nil` for `table` or `filter` silently. The first call to any forwarded method (e.g. `HasActiveFilter()`) would nil-deref deep in `HandleFilterKey` rather than at the construction call site. Today's nine call sites all pass non-nil, but a constructor-level nil check would surface the failure at app startup. One-line fix; symmetric with the existing nil-hook fallback inside `HandleFilterKey`.

2. The pointer-embedding requirement (`*TableBasedPane` rather than `TableBasedPane` value) is convention only — Go has no syntax to enforce it. A future contributor switching to value embedding would silently break `SwapTableAndFilter` semantics on the `SetTheme` path. A reflection-based test that walks every registered pane and asserts the embed kind is `reflect.Ptr` would close the loop.

3. `BorderConfig.FilterQuery` doc comment in `internal/ui/layout/border.go:35-39` says "non-empty when filter mode is active." Slightly imprecise — filter mode can be active with an empty query (immediately after pressing `f`, before typing). The actual semantics: the field always reflects `Filter().Query()`; rendering keys on `FilterQuery != ""`. Reword to "FilterQuery is the live filter query string. When non-empty, ..."

4. `recompute` in `internal/ui/layout/layout.go:62-64` early-returns on invalid preset index without resetting `m.focusOrder`. The `len(liveRows) == 0` branch (line 92-95) does reset; the early-return path doesn't. Today no code path can hit this branch (preset indexes are always validated by `SetPreset`/`CyclePreset`), but the inconsistency is a defensive one-liner away. Either reset there too, or document the unreachability.

5. `GatewayLivePane.buildTableRows()` no-ops when `p.width == 0`. If a `TickMsg` arrives before the first `WindowSizeMsg`, the buffer grows correctly but the table never gets the rows; user sees a one-tick flicker once sizing is applied. Add `p.buildTableRows()` at the end of `SetSize` once `w > 0` to close the gap.

6. `Cell`, `Row`, `Preset` shapes (post-RowSpan retirement) lack a `Validate()` helper for invariants (positive `WidthWeight`/`HeightWeight`, unique `PaneID` in Grid, `Visible` map consistency with cells). A misconfigured preset literal would silently produce a blank screen. Cheap defensive helper; ideal place is a unit test that walks `PageAPresets` and `PageBPresets` once.

---

## Story 183 minor issues — uikit catalogue & SpinnerFrames hardening
**Found:** 2026-04-29 | **Source:** PR #235 Review
**Feature:** 13-tui-design-system

PR #235 (story 183) shipped clean after two review rounds. The following sub-threshold concerns were filed instead of blocking the merge:

1. `internal/uikit/spinner_frames.go` — `SpinnerFrames(mode)` returns the package-level slice (`spinnerFramesUnicode` / `spinnerFramesASCII`) directly. Doc comment says "must NOT be mutated by the caller", but the contract is comment-only. Both `uikit.Spinner` and `cliout.Spinner` share these slices, so an in-place mutation in one consumer corrupts the other. Either return a fresh `append([]string(nil), ...)` slice per call, or expose `SpinnerFrame(m, i)` / `SpinnerFrameCount(m)` accessors that never hand out the slice.

2. `internal/uikit/glyph.go` `GlyphFor` and `internal/uikit/spinner_frames.go` `SpinnerFrames` use a two-state `if mode == GlyphASCII { ... } return unicode` cascade. If a third `GlyphMode` is ever added (e.g. `GlyphNerdFont`), every call site silently keeps returning Unicode without a compile-time signal. Convert to a `switch mode { case GlyphASCII: …; case GlyphUnicode: …; default: … }` so future modes surface as missing cases.

3. `internal/uikit/toast.go` `truncateRunes` invariant ("max must be >= len(ellipsis runes)") is comment-only. Currently safe because `Normalize` is the only caller and uses 48 / 160. A future caller passing `max < 3` in ASCII mode would slice `runes[max-3:max]` and panic. Add a runtime guard or expose the function only via a typed helper that enforces the bound.

4. `internal/uikit/toast_test.go` — body truncation is shared with title via `Normalize`, so the existing `TestToast_TruncatedTitle_AsciiEllipsis` covers the code path. Adding one explicit body assertion (161-rune body in ASCII mode ends with `...`) would make the test surface match the public contract more directly.

5. `internal/uikit/spinner.go` — FPS changed from `time.Second / 10` to `time.Second / 12` during the refactor. Story spec didn't mandate a value, but the change is undocumented. Either revert or note the rationale in a doc comment.

6. `internal/uikit/spinner_test.go:170` — `TestSpinner_Running_Unicode_UsesSpinnerFrames` calls `SetModeForTest(GlyphUnicode)` and defers `SetModeForTest(GlyphUnicode)`. The defer is a no-op (Unicode is already the default). Harmless, but copying this test as a template would propagate the no-op pattern.

7. `internal/uikit/glyph.go` — `GlyphSeparator` value is `"sep.bullet"` but the Go identifier is generic. If a second separator ever lands (`sep.dash`, `sep.pipe`), the existing identifier becomes ambiguous. Consider renaming to `GlyphSepBullet` for parity with the rest of the bucketed naming.

---

## Story 184 minor issues — cliout/uikit integration polish
**Found:** 2026-04-29 | **Source:** PR #236 Review
**Feature:** 13-tui-design-system

PR #236 (story 184) shipped clean after one fix round. Sub-threshold concerns logged for future hardening:

1. `internal/cliout/spinner_test.go` `TestSpinnerFrames_AsciiSet` asserts only the ASCII frame set. A constant-mode bug like `func resolveSpinnerFrames() []string { return uikit.SpinnerFrames(GlyphASCII) }` would still pass because TestMain pins ASCII. Add a symmetric unicode-mode sub-test that toggles `uikit.SetModeForTest(GlyphUnicode)` and asserts `resolveSpinnerFrames()` returns `uikit.SpinnerFrames(GlyphUnicode)`.

2. `internal/cliout/tty.go` `SetTestMode(false)` does not restore the prior `uikit` mode (only `SetTestMode(true)` pins it). Currently benign because `cmd/root_test.go` and `cliout` `TestMain` pin ASCII once and never toggle mid-suite. If a future test toggles modes, snapshot the prior mode at `SetTestMode(true)` entry and restore on `false`.

---

## Story 185 minor issues — PaneChrome migration brittleness
**Found:** 2026-04-29 | **Source:** PR #237 Review
**Feature:** 13-tui-design-system

PR #237 (story 185) shipped clean after one fix round. One sub-threshold concern logged:

1. `TestRenderGrid_AsciiBorders` (`internal/app/render_test.go`) asserts `─` and `│` appear nowhere in the entire grid output, not just in border positions. `internal/ui/panes/playlists_pane.go` and `albums_pane.go` emit literal `──` separators in `Title()` when `inTrackView` is true. Default test-app state doesn't trigger that branch today, so the test passes. A future change that selects a playlist before snapshot, or a new pane that uses `─`/`│` as a content separator, will break this test even if border glyphs are correct. Consider scoping assertions to first/last lines of each pane rect, or asserting the four corners specifically.

---

## Story 186 minor issues — overlay content ASCII fallback gaps
**Found:** 2026-04-29 | **Source:** PR #238 Review
**Feature:** 13-tui-design-system

PR #238 (story 186) shipped clean after one fix round. Two latent ASCII-fallback content gaps logged — they live in inner content (not OverlayChrome border), so out of scope for this story:

1. `internal/ui/panes/search.go:803` hardcodes `strings.Repeat("─", innerWidth)` for the search tab-bar separator. Under `ui.glyphs = "ascii"` this still emits `─`. Story 190 (pane-content-search-devices) or 191 (remaining-glyph-leaks) should route this through `uikit.GlyphFor(GlyphHRule, ActiveMode())`.

2. `internal/uikit/components/table.go` (third-party `bubble-table@v0.19.2/table/border.go:78-79,93`) hardcodes `│` for column `Left`/`Right`/`InnerDivider`. The search Results overlay shows these column separators in ASCII mode. Either patch bubble-table via TableChrome glyph injection or document the limitation. Same category as item 1.

3. `internal/ui/panes/help_overlay.go:140` renders `│` as a content divider via `lipgloss.NewStyle()...Render("│")` — already a known carve-out (it's a vertical divider, not a border), but worth tracking alongside items 1-2 if a future story sweeps inline glyphs across the help surface.

---

## Story 187 minor issues — PlaybackControls test depth + visual diffs
**Found:** 2026-04-30 | **Source:** PR #239 Review
**Feature:** 13-tui-design-system

PR #239 (story 187) shipped clean after one fix round. Sub-threshold concerns logged:

1. `internal/uikit/playback_controls_test.go` lacks a direct uikit-level test for the paused branch (`Playing=false`) asserting `▷` (unicode) / `|>` (ASCII) glyphs appear. Currently covered transitively via `controls_test.go` legacy wrapper. A regression that swapped `GlyphPaused`/`GlyphPausedPB` in the primitive itself wouldn't be caught directly.

2. Unicode-mode tests don't assert ASCII fallbacks (`||`, `Q`, `sh`, `ro`, `|>`) are absent. `GlyphFor` itself has catalogue-level tests in `glyph_test.go`, so a regression that hardcoded ASCII strings while bypassing `GlyphFor` is unlikely but possible.

3. Visual diff (intentional, mention in PR description for downstream surprises): `RepeatOff` now renders `⟳` instead of legacy `↻`; `nowplaying.go:Title()` flipped from state-mode (`▶` when playing) to action-mode (`⏸` when playing) to match the controls strip semantics.

---

## Story 189 minor issues — viz ASCII renderer test depth
**Found:** 2026-04-30 | **Source:** PR #241 Review
**Feature:** 13-tui-design-system

PR #241 (story 189) shipped clean after one fix round. One sub-threshold concern logged:

1. `selectRenderer` is called inside `engine.go:generateFrames`, which only runs on `SetSize` / `CyclePattern` / `SetPattern`. If a future change toggles `uikit.ActiveMode()` at runtime without resize/cycle (e.g. live mode toggle hotkey), the cached frames stay stale until the next regen. No test covers this. Borderline — only matters if mode-toggle-at-runtime becomes a feature. Add a test that flips mode after `SetSize`, calls `CyclePattern` (forcing regen), and verifies the new renderer wins, or hook `selectRenderer` into a frame-time read if mode flips become user-facing.

2. Comment typo in `TestAsciiBars_FourLevelDensityRamp` at line ~195: row-glyph documentation is jumbled (test logic is correct, only the explanatory comment is wrong). Cleanup-only.

---

## Story 190 minor issues — pane content cleanup polish
**Found:** 2026-04-30 | **Source:** PR #242 Review
**Feature:** 13-tui-design-system

PR #242 (story 190) shipped after one fix round that addressed two CRITICAL production regressions (search spinner not animating, devices empty-state losing chrome). Sub-threshold cleanups logged:

1. `internal/ui/panes/devices.go:View()` computes `innerWidth` twice — once at lines 129-132 before the `len(d.devices) == 0` check, then again inside the empty-state path. Trivial dedup.

2. `TestSearchOverlay_Spinner_AsciiRunningFrames` positive assertion may pass on `OverlayChrome` border `|` characters (VRule) since they share the rotating-bar glyph in ASCII. The braille negative assertion is airtight, but the positive frame check could be tightened by isolating spinner output (e.g. assert frame appears in the same line as "Searching…" rather than anywhere in the view).

3. The default-branch `o.sp.Update(msg)` was intentionally removed in favour of explicit `searchSpinnerTickMsg` handling. If a future change adds another spinner-driven sub-component to the search overlay, the dispatcher needs revisiting (currently only the search overlay's own spinner uses `searchSpinnerTickMsg`).

---

## Story 191 minor issues — remaining glyph leaks
**Found:** 2026-04-30 | **Source:** PR #243 Review
**Feature:** 13-tui-design-system

PR #243 (story 191) shipped after one fix round that closed 6 ASCII test gaps (3 critical: profile-truncation gate, help_overlay `│` divider, table playingSymbol). Sub-threshold concerns:

1. `engine_test.go` `block.go` `'█'` rune assertions remain hardcoded. Will break under story 192's `LANG=C` CI matrix when `TestMain` calls `Use("auto")` and ASCII becomes the test default. Story 192 must update those assertions to use `uikit.GlyphFor(uikit.GlyphBarFull, mode)` for the expected value.

2. `render_test.go` `TestRender_AsciiInlineGlyphs` positive assertion for `*` is mildly weak — multiple roles map to `*` in ASCII (`GlyphMusicNote`, `GlyphBullet`, `GlyphPinned`, `GlyphRunning`). The `NotContains` checks for unicode forms enforce no leakage, but a stronger positive assertion would isolate which role is producing the `*` (e.g. assert `*` appears in the banner line specifically rather than anywhere).

3. Polish items: `networklog_pane.go:276` calls `uikit.ActiveMode()` per-row inside the loop (could hoist); `render.go:321-323` calls `uikit.GlyphFor(uikit.GlyphBullet, ...)` three times in the same function (could hoist as a local var, matching the existing `note` pattern); `recentlyplayed_pane.go:View()` has an `else` after a `return` (drop for idiomatic Go).
