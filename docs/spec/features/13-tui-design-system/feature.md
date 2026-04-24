---
title: "TUI Design System"
status: open
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
canonical step-by-step TDD guide for every story. Each story below cross-references the
matching `## Task N (SN): ...` section in the plan.

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

**Supersedes:** fragments of `docs/DESIGN.md` that describe pane chrome, header, status
bar, overlay rendering, and onboarding panel internals (S19). `DESIGN.md` retains
layout, grid, page, and preset mechanics.

**Peer feature:** `12-cli-output`. The glyph catalogue (§5 of the design record) and
emphasis-role vocabulary (§6) are shared between `internal/cliout` and
`internal/uikit`; glyph swaps propagate to both packages in the same PR (S14).

**Depends on:** nothing — every story merges green against `make ci` independently.
Phase 1 (S1 + S2) must ship before any Phase 2+ story. Phases 2–4 parallelise
internally. Phase 5 depends on everything.

## Acceptance Criteria

- [ ] `internal/uikit/` package exists with 18 primitive files (`pane_chrome.go`,
      `overlay_chrome.go`, `panel.go`, `table_chrome.go`, `list_row.go`,
      `section_label.go`, `empty_state.go`, `url_box.go`, `header_bar.go`,
      `status_bar.go`, `key_bar.go`, `chip.go`, `form_field.go`, `toast.go`,
      `status_glyph.go`, `progress_bar.go`, `spinner.go`) plus scaffold files
      (`doc.go`, `glyph.go`, `role.go`, `config.go`, `capture.go`)
- [ ] `internal/uikit` package coverage = 100% (`go test -cover ./internal/uikit/...`)
- [ ] `internal/uikit.Capture(fn)` captures ANSI-stripped line arrays for structural
      assertions without rendering side-effects
- [ ] Every primitive ships with a unicode snapshot test **and** an ascii snapshot test
- [ ] Every primitive has a role-token assertion (field → theme token) matching §6.2 of
      the design record
- [ ] `config.Config` has a `UI` section with `Glyphs string` field; default `"auto"`;
      accepted values `auto` | `unicode` | `ascii`; rejected values fail validation
- [ ] `uikit.ActiveMode()` returns `GlyphUnicode` when `LANG`/`LC_ALL` matches `UTF-8`,
      else `GlyphASCII`; honours the config override
- [ ] `docs/TUI-DESIGN-SYSTEM.md` exists under `docs/` with every primitive documented
      via the six-block contract template (Purpose, Fields, Rendering, Roles, Glyphs,
      Lifecycle, Tests)
- [ ] `docs/DESIGN.md` no longer contains the primitive-rendering sections; retains
      layout, grid, page, preset, and pane-toggle mechanics; points at
      `docs/TUI-DESIGN-SYSTEM.md`
- [ ] `docs/PANE-TEMPLATE.md` Step 2 scaffold uses `uikit.PaneChrome` + uikit primitives
- [ ] `CLAUDE.md` Reading Order references `docs/TUI-DESIGN-SYSTEM.md`
- [ ] `CLAUDE.md` "What Agents Must NEVER Do" has an entry requiring glyph/primitive
      additions to update the design-system doc in the same commit
- [ ] `internal/ui/layout/border.go` no longer emits `ᐅ`; filter mode uses the notch
      format `filtering: "query" ─╮ Esc close ╭`; supports ascii mode via `GlyphFor`
- [ ] `internal/cliout/message.go` `StatusWarning` glyph is `◬`, not `⚠`; ascii form `!`
- [ ] All 5 `renderWith*Overlay` call sites in `internal/app/render.go` go through
      `uikit.OverlayChrome`
- [ ] `renderOnboarding*`, `renderAuthPanel`, `renderTooSmall`, `renderSplash` go through
      `uikit.Panel`
- [ ] `renderHeader`, `renderProfileChip`, device-chip builder go through
      `uikit.HeaderBar` + `uikit.Chip`
- [ ] `renderStatusBar` goes through `uikit.StatusBar` + `uikit.KeyBar`
- [ ] `components/table.go` rendering goes through `uikit.TableChrome`
- [ ] Theme overlay, profile overlay, playlist read-only rows use `uikit.ListRow` /
      `uikit.LockedRow`
- [ ] Page B sub-section labels (GATEWAY, APP, GATEWAY LOG, SPOTIFY, AUTO-TRAFFIC) go
      through `uikit.SectionLabel`
- [ ] Empty queue, empty search results, loading playlist tracks use `uikit.EmptyState`
- [ ] Onboarding redirect-URI block uses `uikit.URLBox`
- [ ] Every `a.alerts.NewAlertCmd(type, msg)` call site migrated to `a.toasts.Cmd(Toast{...})`
      with intent-typed `ToastIntent`
- [ ] Seek bar and volume bar in `components/controls.go` go through `uikit.ProgressBar`
      with partial-block rendering unicode and `=`/`-`/`.` ascii fallback
- [ ] OAuth wait in onboarding uses `uikit.Spinner`; Done/Fail/Cancel states emit
      `SpinnerDoneMsg` / `SpinnerFailMsg` / `SpinnerCancelledMsg`
- [ ] Onboarding client-ID input uses `uikit.FormField` with intrinsic validation and
      error slot beneath
- [ ] `make ci` passes after every story
- [ ] No feature flag (`ui.design_system = on/off`) — each primitive is migrated at its
      call sites with visual diff in code review

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
