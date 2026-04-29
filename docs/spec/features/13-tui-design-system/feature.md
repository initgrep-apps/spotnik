---
title: "TUI Design System"
status: in-progress
---

## Description

Formalise the Spotnik TUI's visible surface into a reusable `internal/uikit` package — 18
typed primitives (`PaneChrome`, `OverlayChrome`, `Panel`, `TableChrome`, `ListRow`,
`LockedRow`, `SectionLabel`, `EmptyState`, `URLBox`, `HeaderBar`, `StatusBar`, `KeyBar`,
`Chip`, `FormField`, `Toast`, `StatusGlyph`, `ProgressBar`, `Spinner`), a frozen glyph
catalogue with ASCII fallback, a role-to-theme-token emphasis matrix, and six
non-overlapping feedback surfaces. Migrates every existing pane, overlay, header, status
bar, onboarding panel, and notification call site onto the new primitives; removes the
`ᐅ` action prefix and swaps `⚠` for `◬` across both the TUI and the shipped `internal/cliout`
package.

**Design record:** `docs/superpowers/specs/2026-04-24-tui-design-system-design.md` — the
full rationale (problem, goals, glyph catalogue, role matrix, primitive contracts,
migration plan, risks).

**Implementation plan:** `docs/superpowers/plans/2026-04-24-tui-design-system.md` — the
canonical step-by-step TDD guide for stories 150–172. Each story below cross-references
the matching `## Task N (SN): ...` section in the plan.

**Glyph fallback follow-up (stories 183–192).** The post-merge audit
(`docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md`) found that while
`uikit` is ~97% compliant, the running app still leaks unicode glyphs end-to-end:
`renderGrid` builds `BorderConfig` inline without populating glyph fields (every grid
pane border stays unicode under `ui.glyphs = "ascii"`); `internal/cliout` maintains a
parallel hardcoded glyph set and ignores `ActiveMode()`; visualizer is unicode-only;
`controls.go` and `infobox.go` hand-roll their own borders. Stories 183–192 close the
gap end-to-end. Plan: `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

**What changes:**

- New package `internal/uikit` — 18 primitives, glyph registry (`GlyphFor(role, mode)`),
  role matrix (`ColourFor(role, theme)`), `GlyphMode` config (`auto` | `unicode` | `ascii`)
  with env-based auto-detect, `Capture` test helper for ANSI-stripped structural assertions.
- New reference doc `docs/TUI-DESIGN-SYSTEM.md` — canonical guideline for TUI primitive
  work, peer to `docs/CLI-OUTPUT.md`. Replaces the primitive-rendering sections of
  `docs/DESIGN.md`; `DESIGN.md` retains layout/grid/page/preset mechanics only.
- `docs/PANE-TEMPLATE.md` Step 2 scaffold rewritten to use `PaneChrome` + uikit primitives.
- New config field `[ui] glyphs = "auto" | "unicode" | "ascii"` (default `"auto"`).
- `⚠` → `◬` everywhere (TUI notifications, cliout `StatusWarning`, onboarding header).
- `ᐅ` action prefix removed from `layout/border.go`; filter mode uses notch format.
- All pane chrome, overlay chrome, header bar, status bar, onboarding screens, splash,
  auth panel, and too-small screen re-rendered through primitives (no more ad-hoc
  `lipgloss.NewStyle()` compositions at call sites).
- `bubbleup` alerts wrapped behind a typed `Toast` API (`Intent`, `Title`, `Body`, `TTL`);
  every `a.alerts.NewAlertCmd` call site migrated.
- OAuth wait in onboarding uses the new `Spinner` with `Done`/`Fail`/`Cancel` contract.
- **Glyph-fallback follow-up (183–192):** new `PlaybackControls` primitive, new
  `GlyphSeparator` / `GlyphPlaylist` / `GlyphDevice*` / `GlyphEnter` / `GlyphEscape` /
  `GlyphTab` / `GlyphBackspace` / `GlyphSpace` / `GlyphSuperscript0..9/Plus/Minus`
  catalogue rows, exported `uikit.SpinnerFrames(mode)`, `cliout` imports `uikit` for the
  shared catalogue, `renderGrid` + 5 direct callers migrated to `PaneChrome` /
  `OverlayChrome`, audio visualizer gains `AsciiBarsRenderer`, banned-glyph + catalogue
  -leak + direct-`RenderPaneBorder` CI guards, and `LANG=C` test matrix.

**Supersedes:** fragments of `docs/DESIGN.md` that describe pane chrome, header, status
bar, overlay rendering, and onboarding panel internals (S19). `DESIGN.md` retains
layout, grid, page, and preset mechanics.

**Peer feature:** `12-cli-output`. The glyph catalogue (§5 of the design record) and
emphasis-role vocabulary (§6) are shared between `internal/cliout` and
`internal/uikit`; glyph swaps propagate to both packages in the same PR (S14, S184).

**Depends on:** nothing — every story merges green against `make ci` independently.
Phase 1 (S1 + S2) must ship before any Phase 2+ story. Phases 2–4 parallelise
internally. Phase 5 depends on everything. Glyph-fallback stories form Phases 6–11
(see Story Order); story 183 gates 184–192; story 192 ships last.

## Acceptance Criteria

- [x] `internal/uikit/` package exists with 18 primitive files (`pane_chrome.go`,
      `overlay_chrome.go`, `panel.go`, `table_chrome.go`, `list_row.go`,
      `section_label.go`, `empty_state.go`, `url_box.go`, `header_bar.go`,
      `status_bar.go`, `key_bar.go`, `chip.go`, `form_field.go`, `toast.go`,
      `status_glyph.go`, `progress_bar.go`, `spinner.go`) plus scaffold files
      (`doc.go`, `glyph.go`, `role.go`, `config.go`, `capture.go`)
- [x] `internal/uikit` package coverage = 100% (`go test -cover ./internal/uikit/...`)
- [x] `internal/uikit.Capture(fn)` captures ANSI-stripped line arrays for structural
      assertions without rendering side-effects
- [x] Every primitive ships with a unicode snapshot test **and** an ascii snapshot test
- [x] Every primitive has a role-token assertion (field → theme token) matching §6.2 of
      the design record
- [x] `config.Config` has a `UI` section with `Glyphs string` field; default `"auto"`;
      accepted values `auto` | `unicode` | `ascii`; rejected values fail validation
- [x] `uikit.ActiveMode()` returns `GlyphUnicode` when `LANG`/`LC_ALL` matches `UTF-8`,
      else `GlyphASCII`; honours the config override
- [x] `docs/TUI-DESIGN-SYSTEM.md` exists under `docs/` with every primitive documented
      via the six-block contract template (Purpose, Fields, Rendering, Roles, Glyphs,
      Lifecycle, Tests)
- [x] `docs/DESIGN.md` no longer contains the primitive-rendering sections; retains
      layout, grid, page, preset, and pane-toggle mechanics; points at
      `docs/TUI-DESIGN-SYSTEM.md`
- [x] `docs/PANE-TEMPLATE.md` Step 2 scaffold uses `uikit.PaneChrome` + uikit primitives
- [x] `CLAUDE.md` Reading Order references `docs/TUI-DESIGN-SYSTEM.md`
- [x] `CLAUDE.md` "What Agents Must NEVER Do" has an entry requiring glyph/primitive
      additions to update the design-system doc in the same commit
- [x] `internal/ui/layout/border.go` no longer emits `ᐅ`; filter mode uses the notch
      format `filtering: "query" ─╮ Esc close ╭`; supports ascii mode via `GlyphFor`
- [x] `internal/cliout/message.go` `StatusWarning` glyph is `◬`, not `⚠`; ascii form `!`
- [x] All 5 `renderWith*Overlay` call sites in `internal/app/render.go` go through
      `uikit.OverlayChrome`
- [x] `renderOnboarding*`, `renderAuthPanel`, `renderTooSmall`, `renderSplash` go through
      `uikit.Panel`
- [x] `renderHeader`, `renderProfileChip`, device-chip builder go through
      `uikit.HeaderBar` + `uikit.Chip`
- [x] `renderStatusBar` goes through `uikit.StatusBar` + `uikit.KeyBar`
- [x] `components/table.go` rendering goes through `uikit.TableChrome`
- [x] Theme overlay, profile overlay, playlist read-only rows use `uikit.ListRow` /
      `uikit.LockedRow`
- [x] Page B sub-section labels (GATEWAY, APP, GATEWAY LOG, SPOTIFY, AUTO-TRAFFIC) go
      through `uikit.SectionLabel`
- [x] Empty queue, empty search results, loading playlist tracks use `uikit.EmptyState`
- [x] Onboarding redirect-URI block uses `uikit.URLBox`
- [x] Every `a.alerts.NewAlertCmd(type, msg)` call site migrated to `a.toasts.Cmd(Toast{...})`
      with intent-typed `ToastIntent`
- [x] Seek bar and volume bar in `components/controls.go` go through `uikit.ProgressBar`
      with partial-block rendering unicode and `=`/`-`/`.` ascii fallback
- [x] OAuth wait in onboarding uses `uikit.Spinner`; Done/Fail/Cancel states emit
      `SpinnerDoneMsg` / `SpinnerFailMsg` / `SpinnerCancelledMsg`
- [x] Onboarding client-ID input uses `uikit.FormField` with intrinsic validation and
      error slot beneath
- [x] No feature flag (`ui.design_system = on/off`) — each primitive is migrated at its
      call sites with visual diff in code review

### Glyph fallback (stories 183–192)

- [ ] `internal/uikit/glyph.go` exposes `GlyphSeparator`, `GlyphPlaylist`,
      `GlyphDeviceComputer`, `GlyphDevicePhone`, `GlyphDeviceSpeaker`, `GlyphDeviceTV`,
      `GlyphEnter`, `GlyphEscape`, `GlyphTab`, `GlyphBackspace`, `GlyphSpace`,
      `GlyphSuperscript0..9`, `GlyphSuperscriptPlus`, `GlyphSuperscriptMinus` with
      `glyphTable` rows and matching unicode + ascii forms (per audit §4.1)
- [ ] `uikit.SpinnerFrames(mode GlyphMode) []string` is exported and is the single
      source of braille (unicode) and `|/-\` (ascii) frame arrays for both
      `uikit.Spinner` and `cliout.Spinner`
- [ ] `internal/uikit/list_row.go`, `toast.go`, `header_bar.go`, `key_bar.go`,
      `status_bar.go` resolve every glyph and border field via `GlyphFor` —
      no hardcoded `…`, ` ─ `, ` · `, ` | `, or omitted `BorderConfig` glyph fields
- [ ] `EmptyState`, `URLBox`, `HeaderBar`, `FormField` ship ASCII-mode snapshot tests
- [ ] `internal/cliout` imports `internal/uikit` and resolves every glyph and spinner
      frame via `uikit.GlyphFor` / `uikit.SpinnerFrames`; `statusGlyph()` is a `Status`
      → `GlyphRole` map; `Hint` arrow uses `GlyphFor(GlyphInfo, mode)`;
      `cliout.SetTestMode(true)` additionally calls `uikit.SetModeForTest(GlyphASCII)`
- [ ] `internal/cliout` test suite asserts glyph **roles** via `uikit.GlyphFor`, not raw
      codepoints, and runs in both unicode and ASCII modes
- [ ] `internal/app/render.go:renderGrid` migrates from inline `BorderConfig` +
      `layout.RenderPaneBorder` to `uikit.PaneChrome.Render`; every grid pane border
      honours `ui.glyphs = "ascii"`
- [ ] `internal/ui/panes/themes.go` and `internal/ui/panes/profile.go` migrate from
      direct `RenderPaneBorder` to `uikit.PaneChrome.Render`
- [ ] `internal/ui/panes/devices.go`, `internal/ui/panes/help_overlay.go`, and the three
      `internal/ui/panes/search.go` overlay-render paths migrate to
      `uikit.OverlayChrome.Render`
- [ ] `internal/ui/components/infobox.go` migrates from hand-rolled border to
      `uikit.PaneChrome.Render`
- [ ] `internal/uikit/playback_controls.go` ships a `PlaybackControls` primitive owning
      the seven transport glyphs (shuffle / play / pause / queue / repeat-off /
      repeat-all / repeat-one); `internal/ui/components/controls.go` becomes a thin
      compatibility wrapper that translates legacy string repeat modes
- [ ] `uikit.RegisterBubbleupAlerts(theme)` resolves all five toast prefixes
      (`success`, `error`, `warning`, `info`, `ratelimit`) via `GlyphFor` at call time
- [ ] `internal/ui/panes/nowplaying.go:Title()` resolves `▶`, `⏸`, `─` via `GlyphFor`
- [ ] `internal/ui/components/viz/ascii_bars.go` ships an `AsciiBarsRenderer` (4-level
      `# = .` columns); `viz/engine.go` selects renderer based on `uikit.ActiveMode()`
      so the visualizer is present (degraded) in ascii mode
- [ ] Every Phase 5 inline glyph leak — `panes/devices.go` (`◉/○/⊡⊞⊟⊠`, "No devices found"),
      `panes/search.go` (`bubbles/spinner.Model`), `panes/search_delegate.go`
      (`categorySymbol`, separators), `panes/recentlyplayed_pane.go` and
      `panes/nowplaying.go` empty messages, `panes/networklog_pane.go` (`◷`/`⚡`),
      `panes/profile.go` (`…`), `panes/help_overlay.go` (`│`),
      `panes/gateway_health_pane.go` (custom dot-bar → `uikit.ProgressBar`),
      `app/render.go` (`♪`/`•`/`…`), `components/gradient.go` (`♪`),
      `components/table.go` (`playingSymbol`), `components/viz/block.go` (`█`) —
      resolves through `GlyphFor` or its owning primitive
- [ ] `make check-glyphs` exists, executable, and CI-wired; it runs:
      `scripts/check-banned-glyphs.sh` (no `⚠`, `ᐅ`, `┌┐└┘`, `╔╗╚╝`, `✅`, `❌`, `❗`),
      `scripts/check-catalogue-leaks.sh` (catalogue characters appear only in
      `internal/uikit/glyph.go` and canonical doc files), and
      `scripts/check-render-pane-border.sh` (no direct `layout.RenderPaneBorder`
      callers outside `internal/uikit/`)
- [ ] CI matrix runs the full test suite once with `LANG=en_US.UTF-8` and once with
      `LANG=C`; both pass
- [ ] Manual smoke test under `LANG=C` and `ui.glyphs = "ascii"` confirms every grid
      pane, every overlay, splash, onboarding, toast intent, and the visualizer
      render legibly with no mojibake and correct width
- [ ] `make ci` passes after every story

## Story Order

Phase 1 — Foundation (gate). Must merge before any Phase 2+ story.

1. `150-uikit-scaffold.md` — `internal/uikit` package: glyph catalogue, role matrix,
   `GlyphMode` config with env detection, `Capture` test helper; add `ui.glyphs` to
   `internal/config`; register feature row in `00-overview.md`.
2. `151-layout-border-notch-ascii.md` — remove `ᐅ` from `layout/border.go` filter mode;
   corners look up glyph via `uikit.ActiveMode()` so ascii mode swaps `╭╮╰╯─│` →
   `+++--|`; flip `border_test.go:742` assertion; add ascii snapshot tests.

Phase 2 — Chrome primitives (parallelisable after S1/S2).

3. `152-pane-chrome-primitive.md` — `PaneChrome` wraps `layout.RenderPaneBorder`.
4. `153-overlay-chrome-primitive.md` — `OverlayChrome` consolidates the 5
   `renderWith*Overlay` funcs in `render.go`.
5. `154-panel-primitive.md` — `Panel` replaces `renderOnboarding*`, `renderAuthPanel`,
   `renderTooSmall`, `renderSplash`; panel title slot absorbs the step-header role.
6. `155-header-bar-and-chip.md` — `HeaderBar` + `Chip` extract from
   `render.go:renderHeader`.
7. `156-status-bar-and-key-bar.md` — `StatusBar` composition over the reusable `KeyBar`
   atom; `bubbles/help` short/full help integration.

Phase 3 — Content primitives (parallelisable after S3 for `TableChrome`; S9–S12 after S1).

8. `157-table-chrome-primitive.md` — `TableChrome` wraps `components/table.go`.
9. `158-list-row-and-locked-row.md` — `ListRow` + `LockedRow`; migrate theme overlay,
   profile overlay, playlist read-only rows.
10. `159-section-label-primitive.md` — `SectionLabel` for Page B sub-sections.
11. `160-empty-state-primitive.md` — `EmptyState` replaces hand-rolled "no data"
    messages in queue, search, playlist loading.
12. `161-url-box-primitive.md` — `URLBox` wraps URL/code content; wraps long URLs at
    `&` boundaries (current `wrapURL` helper).

Phase 4 — Feedback (parallelisable after S1).

13. `162-toast-primitive-migration.md` — typed `Toast` wrapping `bubbleup`; migrate all
    `a.alerts.NewAlertCmd` call sites; position rules per view-mode.
14. `163-status-glyph-warning-swap.md` — atomic `StatusGlyph`; swap `⚠` → `◬` across
    `internal/cliout`, `internal/ui/components/notifications.go`, onboarding, and
    remaining scattered sites.
15. `164-progress-bar-primitive.md` — seek + volume through `ProgressBar`; partial-block
    unicode rendering, `=`/`-`/`.` ascii fallback.
16. `165-spinner-primitive-oauth-wiring.md` — `Spinner` with Done/Fail/Cancel contract;
    wire onboarding OAuth wait; emit `SpinnerDoneMsg` / `SpinnerFailMsg` /
    `SpinnerCancelledMsg`.

Phase 5 — Composites & cleanup (depends on Phases 2–4).

17. `166-form-field-onboarding-input.md` — `FormField` wraps `bubbles/textinput`;
    migrate onboarding client-ID input with intrinsic validation + error slot.
18. `167-onboarding-end-to-end-rewrite.md` — onboarding rewrite composing every
    primitive (`Panel`, `FormField`, `URLBox`, `Spinner`, `Toast`, `StatusGlyph`).
19. `168-design-system-docs-rewrite.md` — `docs/DESIGN.md` rewrite (strip primitives,
    retain layout/grid/page mechanics, pointer to `TUI-DESIGN-SYSTEM.md`);
    `docs/TUI-DESIGN-SYSTEM.md` created with 18 primitive contracts;
    `docs/PANE-TEMPLATE.md` Step 2 scaffold updated to use `PaneChrome`.

Phase 6 — Glyph-fallback foundation (gate for 184–192).

20. `183-glyph-catalogue-uikit-audit.md` — domain / chord / superscript catalogue rows
    with co-committed `docs/TUI-DESIGN-SYSTEM.md` updates; export
    `uikit.SpinnerFrames(mode)`; fix `list_row.go`, `toast.go`, `header_bar.go`,
    `key_bar.go`, `status_bar.go` self-audit gaps; ASCII snapshot tests for
    `EmptyState`, `URLBox`, `HeaderBar`, `FormField`.

Phase 7 — cliout integration (depends on 183).

21. `184-cliout-uikit-integration.md` — `internal/cliout` imports `internal/uikit`;
    `statusGlyph()` becomes `Status` → `GlyphRole` map; `Hint` arrow via
    `GlyphFor(GlyphInfo)`; spinner frames via `uikit.SpinnerFrames(ActiveMode())`;
    `cliout.SetTestMode` calls `uikit.SetModeForTest`; tests rebaselined to assert
    glyph roles, not codepoints, in both modes.

Phase 8 — Pane chrome migration (depends on 183).

22. `185-pane-chrome-migration.md` — `app/render.go:renderGrid`,
    `panes/themes.go`, `panes/profile.go`, `components/infobox.go` migrate to
    `uikit.PaneChrome.Render`. Single largest fallback gap.
23. `186-overlay-chrome-migration.md` — `panes/devices.go`, `panes/help_overlay.go`,
    and three `panes/search.go` overlay-render paths migrate to
    `uikit.OverlayChrome.Render`.

Phase 9 — Critical content fixes (depends on 183).

24. `187-playback-controls-primitive.md` — new `uikit.PlaybackControls` primitive owning
    the seven transport glyphs; `components/controls.go` becomes a thin compatibility
    wrapper translating legacy string repeat modes; `panes/nowplaying.go:Title()`
    routes `▶`/`⏸`/`─` via `GlyphFor`.
25. `188-toast-bubbleup-registration.md` — `uikit.RegisterBubbleupAlerts(theme)`
    resolves the five alert prefixes via `GlyphFor` at call time; `notifications.go`
    becomes a thin wrapper calling the helper.
26. `189-viz-ascii-renderer.md` — new `viz.AsciiBarsRenderer` (4-level `# = .` columns);
    `viz/engine.go` selects renderer based on `uikit.ActiveMode()`; visualizer stays
    present (degraded) in ascii mode rather than rendering mojibake.

Phase 10 — Pane content cleanup (depends on 183, 187).

27. `190-pane-content-search-devices.md` — `panes/devices.go` status glyphs / device
    icons / empty state; `panes/search.go` `bubbles/spinner.Model` → `uikit.Spinner`;
    `panes/search_delegate.go` `categorySymbol` and separators routed through
    `GlyphFor`.
28. `191-remaining-glyph-leaks.md` — empty-state migrations
    (`recentlyplayed_pane.go`, `nowplaying.go`); inline glyph leaks across
    `networklog_pane.go`, `profile.go`, `help_overlay.go`, `gradient.go`,
    `app/render.go` (banner / bullets / ellipsis), `components/table.go`
    `playingSymbol`, `viz/block.go` `█`; `gateway_health_pane.go` custom dot-bar →
    `uikit.ProgressBar`.

Phase 11 — Test parity, CI guards, smoke (depends on 184–191).

29. `192-glyph-fallback-ci-guards.md` — `cliout` ASCII test sweep; `make check-glyphs`
    target wrapping `scripts/check-banned-glyphs.sh`,
    `scripts/check-catalogue-leaks.sh`, and `scripts/check-render-pane-border.sh`;
    CI matrix runs the test suite under `LANG=en_US.UTF-8` and `LANG=C`; manual smoke
    test under `LANG=C` and `ui.glyphs = "ascii"` covering every surface.
