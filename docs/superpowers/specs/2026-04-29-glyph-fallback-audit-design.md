# Glyph Fallback Audit & Remediation — Design

> **Date:** 2026-04-29
> **Author:** Audit by parallel research agents, synthesis owned by spec author.
> **Peer docs:** `docs/TUI-DESIGN-SYSTEM.md` (canonical primitive contract),
> `docs/CLI-OUTPUT.md` (canonical cliout contract).

---

## 1. Purpose

`internal/uikit` introduced 18 typed primitives, a frozen glyph catalogue, and a
config-driven mode resolver (`ui.glyphs = "auto" | "unicode" | "ascii"`). The
contract: every visible surface of the TUI **and** the CLI must render correctly
in ASCII mode without breaking layout, intent, or affordance.

This document is the comprehensive overview the next plan will execute against.
It captures:

- What the glyph system provides today, end to end.
- Every place the app bypasses it.
- What it would cost — in component changes, test additions, and missing
  catalogue entries — to reach full ASCII parity without destroying the UI.

## 2. What "full glyph fallback" means

The contract is not just "ASCII chars instead of unicode." It is:

1. **Resolution path is uniform.** A single source of truth (`uikit.GlyphFor`
   driven by `uikit.ActiveMode()`) decides every glyph the app renders. No call
   site picks unicode unconditionally. No package maintains a parallel table.
2. **Layout survives the swap.** A unicode glyph that renders 1 column wide may
   map to a 3-column ASCII fallback (e.g. `◉` → `(*)`, `⏭` → `>>`). Every call
   site that allocates space must use width-aware measurement (`lipgloss.Width`,
   not `len`).
3. **Both packages stay in sync.** `uikit` and `cliout` share the catalogue
   per design doc §7. Drift is a bug.
4. **Every primitive ships an ASCII test.** Snapshot or structural assertion
   under `SetModeForTest(GlyphASCII)`.
5. **Banned glyphs stay banned.** `⚠`, `ᐅ`, sharp/double corners, emoji.

## 3. Findings — gap inventory

The audit covered four areas in parallel: `internal/uikit`, `internal/ui/panes`,
`internal/ui/components` + `internal/app` + `internal/ui/layout`, and
`internal/cliout`. Findings classified by severity.

### 3.1 `internal/uikit` (self-audit)

The package is ~97 % compliant — every dynamic primitive routes through
`GlyphFor`. Remaining gaps:

| Severity | Location | Gap | Fix |
|---|---|---|---|
| Minor | `list_row.go:48` | Hardcodes `"…"` in `PadOrTruncate` | Route via `GlyphFor(GlyphEllipsis, mode)` |
| Minor | `header_bar.go:52` | Hardcodes `" ─ "` separator | Route via `GlyphFor(GlyphHRule, mode)` |
| Minor | `key_bar.go:31–34` | Hardcodes ` · ` / ` \| ` separator | Add `GlyphSeparator` role or branch on `ActiveMode()` |
| Minor | `status_bar.go` | Calls `layout.RenderPaneBorder` without populating `BorderConfig` glyph fields — relies on layout defaults | Populate fields via `GlyphFor` like `pane_chrome.go` does |
| Minor | `toast.go:112` | Raw rune `'…'` in truncation | Route via `GlyphFor(GlyphEllipsis, mode)` |
| Test gap | `empty_state_test.go` | No ASCII snapshot test | Add ASCII test (low priority — no glyphs rendered) |
| Test gap | `url_box_test.go` | No ASCII snapshot test | Add ASCII test (no glyphs, but border via lipgloss may differ) |
| Test gap | `header_bar_test.go` | No ASCII snapshot test | Add ASCII test once separator fix lands |
| Test gap | `form_field_test.go` | No ASCII test (despite using `GlyphError`) | Add ASCII test |
| **Catalogue** | `glyph.go` | §4.9 keyboard-chord glyphs not as `GlyphRole` constants (`⏎`, `⎋`, `⇥`, `⌫`, `␣`) | Add roles + table entries |
| **Catalogue** | `glyph.go` | §4.10 superscript glyphs not as `GlyphRole` constants (`¹…⁸`, `⁰`, `⁹`, `⁺`, `⁻`) | Add roles + table entries |

No banned glyphs found in `uikit`.

### 3.2 `internal/ui/panes`

Most panes that do data formatting via tables route through `GlyphFor` — good
adoption in `gateway_live_pane.go`, `playlists_pane.go`, `polling_traffic_pane.go`,
`profile.go`, `themes.go`, `gateway_health_pane.go`. Adoption is weakest for
playback-state glyphs in titles, custom empty messages, and a few fixed icons.

| Severity | File | Gap | Fix |
|---|---|---|---|
| Critical | `nowplaying.go:85,87` | `▶` / `⏸` hardcoded in `Title()` | Use `GlyphFor(GlyphPlaying, mode)` and `GlyphFor(GlyphPausedPB, mode)` |
| Critical | `nowplaying.go:91` | `─` hardcoded separator | Use `GlyphFor(GlyphHRule, mode)` |
| Moderate | `nowplaying.go:314–319` | Custom "Nothing playing" empty message | Migrate to `uikit.EmptyState` |
| Moderate | `devices.go:209,212,219,222` | Raw `◉` / `○` for active/available device | Use `uikit.StatusGlyph` with `GlyphActive` / `GlyphAvailable` |
| Moderate | `devices.go:258–264` | Custom device-type icons `⊡⊞⊟⊠` (smartphone/computer/speaker/tv) — not in catalogue, no ASCII fallback | Add `GlyphDeviceComputer`, `GlyphDevicePhone`, `GlyphDeviceSpeaker`, `GlyphDeviceTV` roles — see §4.3 |
| Moderate | `devices.go:166–175` | Calls `layout.RenderPaneBorder` directly for an overlay | Migrate to `uikit.OverlayChrome` |
| Moderate | `devices.go:145–147` | Custom "No devices found" message | Migrate to `uikit.EmptyState` |
| Moderate | `search_delegate.go:62–77` | `categorySymbol` hardcodes `♪`, `★`, `◎`, `▤`, `·` — `▤` (playlist) has no role | Add `GlyphPlaylist` role; route others via existing `GlyphMusicNote`, `GlyphPinned`, `GlyphInactive`, `GlyphBullet` |
| Moderate | `search_delegate.go:95,335` | Raw `│` / `·` separators | Use `GlyphFor(GlyphVRule, mode)` for the vertical rule, new `GlyphSeparator` for the bullet separator |
| Moderate | `search.go:214` | Uses `bubbles/spinner.Model` for first-page loading | Migrate to `uikit.Spinner` |
| Minor | `networklog_pane.go:275,277` | Raw `◷` / `⚡` for priority indicators | Use `GlyphFor(GlyphDeadline, mode)` / `GlyphFor(GlyphRunning, mode)` |
| Minor | `recentlyplayed_pane.go:134` | Custom "No recently played tracks" message | Migrate to `uikit.EmptyState` |
| Minor | `profile.go:228` | Raw `…` in truncation helper | Use `GlyphFor(GlyphEllipsis, mode)` |
| Minor | `gateway_health_pane.go:169–184` | Custom dot-bar for capacity | Migrate to `uikit.ProgressBar` |
| Minor | `help_overlay.go:140` | Raw `│` divider | Use `GlyphFor(GlyphVRule, mode)` |

No banned glyphs found in panes.

### 3.3 `internal/ui/components`, `internal/app`, `internal/ui/layout`

This is where the **critical** breakage lives. Two components render their own
borders / playback chrome with hardcoded unicode and no mode branch.

| Severity | File | Gap | Fix |
|---|---|---|---|
| **Critical** | `ui/components/controls.go:36–57` | 7 hardcoded playback glyphs (`⇄`, `⏸`, `▷`, `≡`, `↻¹`, `↻`) — no mode check, no fallback | Introduce `uikit.PlaybackControls` primitive that owns the 7 transport glyphs and resolves them via `GlyphFor`; migrate `controls.go` to use it |
| **Critical** | `ui/components/infobox.go:99,149,151,162,169,172,183` | 6 border glyphs (`╭`, `╮`, `╰`, `╯`, `─`, `│`) hardcoded — bypasses `layout.RenderPaneBorder` contract | Migrate to `uikit.PaneChrome` |
| Moderate | `ui/components/notifications.go:30,35,40,45,50` | 5 alert prefixes (`✓`, `✗`, `◬`, `→`, `⧖`) registered with bubbleup at construction — never re-resolved per mode | Move bubbleup-registration into `uikit.Toast` so prefixes resolve via `GlyphFor` after `uikit.Use` is called |
| Moderate | `ui/components/viz/braille.go` | Braille visualizer is unicode-only by construction (U+2800–U+28FF) — no ASCII path | Add `AsciiBarsRenderer` that draws columns using `█`/`#`/`=`/`.` per block; engine selects renderer based on `ActiveMode()` |
| Moderate | `ui/components/viz/block.go:45` | `█` hardcoded; East-Asian-width caveat in comment | Route via `GlyphFor(GlyphBarFull, mode)`; acceptable since ASCII mode is `#` |
| Moderate | `ui/components/table.go:13` | `const playingSymbol = "▶"` | Lazy-resolve at render time via `GlyphFor(GlyphPlaying, ActiveMode())` |
| Minor | `ui/components/gradient.go:197,200` | Raw `♪` icon on volume bar | Use `GlyphFor(GlyphMusicNote, mode)` |
| Minor | `app/render.go:132` | Raw `♪` in app banner | Use `GlyphFor(GlyphMusicNote, mode)` |
| Minor | `app/render.go:320–322` | Raw `•` × 3 in onboarding error bullets | Use `GlyphFor(GlyphBullet, mode)` |
| Minor | `app/render.go:526,551` | Raw `…` in name truncation | Use `GlyphFor(GlyphEllipsis, mode)` |
| Clean | `ui/layout/border.go:71–97` | Honors caller-provided glyph fields; falls back to unicode defaults if caller does not populate them — design intentional | No change. Fix is at the caller (InfoBox). |
| Clean | `app/splash.go` | Uses `uikit.StatusGlyph` correctly; banner is figlet prose | No change |
| Clean | `app/auth.go` | Uses lipgloss but no raw glyphs | No change |

### 3.4 `internal/cliout`

`cliout` does not import `uikit` and maintains a parallel hardcoded glyph set.
Today the values match by accident; tomorrow's drift is a bug. **More
importantly, cliout has no ASCII fallback path at all** — `pinASCII` only
strips colour, not glyphs.

| Severity | File | Gap | Fix |
|---|---|---|---|
| **Critical** | `cliout/message.go:36–53` | `statusGlyph()` returns hardcoded `◉ ◎ ✓ ✗ ◬ ◌` — never resolves by mode | Replace switch with `uikit.GlyphFor(role, uikit.ActiveMode())`; map `Status` → `GlyphRole` once |
| **Critical** | `cliout/message.go:176` | `Render("→")` for hint arrow | Route via `GlyphFor(GlyphInfo, mode)` |
| **Critical** | `cliout/spinner.go:16` | `spinnerFrames = ["⠋","⠙","⠹","⠸","⠼","⠴","⠦","⠧","⠇","⠏"]` — no ASCII fallback (TUI peer uses `|/-\` in ASCII) | Resolve frame set by `ActiveMode()`; expose unicode + ASCII frame arrays from `uikit` so both packages share them |
| **Critical** | `cliout/*` | No package consults `uikit.ActiveMode()` — `ui.glyphs = "ascii"` config is ignored entirely on the CLI side | Wire `uikit.Use` resolution into cliout's startup path; cliout should NOT duplicate `ResolveMode` |
| Test gap | `cliout/*_test.go` | All tests hardcode unicode expectations under `SetTestMode(true)` (which only pins colour to ASCII) — no test exercises ASCII glyph fallback | Add ASCII-mode test pass once `GlyphFor` integration lands |
| Clean | Status → glyph mapping | `Success/Failure/Warning` glyphs match `uikit.Toast` intents exactly | No semantic change |

No banned glyphs in cliout.

### 3.5 Pane chrome — the biggest gap

`internal/ui/layout/border.go` implements glyph fallback correctly: `BorderConfig`
exposes `CornerTL/TR/BL/BR/HRule/VRule/ToggleKeyStr` fields, and `resolveGlyphs`
falls back to hardcoded unicode (`╭╮╰╯─│`) when any field is empty. `uikit.PaneChrome.Render`
populates them via `GlyphFor`. The contract is sound.

**The callers do not fulfil it.** Almost every visible pane border on the running
app today is rendered by a direct `layout.RenderPaneBorder` call that never
populates the glyph fields, so every border falls through to the unicode defaults
even when `ui.glyphs = "ascii"`.

| Severity | Caller | Why it matters |
|---|---|---|
| **Critical** | `internal/app/render.go:417–426` `renderGrid` | Builds `BorderConfig` for **every grid pane** (10 panes × 2 pages) and calls `RenderPaneBorder` directly — without `CornerTL/…/HRule/VRule/ToggleKeyStr`. Result: every grid pane border is hardcoded unicode in ASCII mode. This is the single largest fallback gap in the app. |
| **Critical** | `internal/ui/panes/themes.go:160` | Direct `RenderPaneBorder` call, no glyph fields |
| **Critical** | `internal/ui/panes/profile.go:183` | Direct `RenderPaneBorder` call, no glyph fields |
| **Critical** | `internal/ui/panes/devices.go:166–175` | Direct `RenderPaneBorder` for the device-switch overlay, no glyph fields |
| **Critical** | `internal/ui/panes/help_overlay.go:152–161` | Direct `RenderPaneBorder` for the help overlay, no glyph fields |
| **Critical** | `internal/ui/panes/search.go:778, 880, 943` | Direct `RenderPaneBorder` for the search overlay frames, no glyph fields |
| Minor | `internal/uikit/status_bar.go` | Direct `RenderPaneBorder` call, no glyph fields (already noted in §3.1) |

`uikit.PaneChrome.Render` is the only correct caller in the codebase, and it
currently has **zero invocations from the running app** — `renderGrid` reimplements
the same `BorderConfig` build inline instead of delegating to `PaneChrome`.

**Decision:** Migrate `renderGrid` and the 5 direct pane callers to
`uikit.PaneChrome.Render` (and `uikit.OverlayChrome.Render` for overlays).
Single chrome surface across the app, automatic glyph routing, automatic
toggle-key superscript fallback. Removes the duplication that allowed this
gap to ship in the first place.

This is the highest-leverage critical fix and runs first in §6.

## 4. Cross-cutting findings

### 4.1 Missing `GlyphRole` constants

Six gaps in the catalogue surfaced during the audit. All require additions to
`uikit/glyph.go` and a corresponding row in `docs/TUI-DESIGN-SYSTEM.md` §4.

| Proposed role | Unicode | ASCII | Where used today |
|---|---|---|---|
| `GlyphSeparator` | `·` | `\|` | `KeyBar`, `search_delegate.go`, prose lists |
| `GlyphPlaylist` | `▤` | `[=]` | `search_delegate.go` (search results) |
| `GlyphDeviceComputer` | `⊡` | `[c]` | `devices.go` device-type icon |
| `GlyphDevicePhone` | `⊞` | `[p]` | `devices.go` |
| `GlyphDeviceSpeaker` | `⊟` | `[s]` | `devices.go` |
| `GlyphDeviceTV` | `⊠` | `[tv]` | `devices.go` |

Plus the §4.9 (keyboard chords) and §4.10 (superscripts) entries already documented in the design doc but not yet implemented in `glyph.go`:

| Section | Roles missing |
|---|---|
| Keyboard chords | `GlyphEnter ⏎/Enter`, `GlyphEscape ⎋/Esc`, `GlyphTab ⇥/Tab`, `GlyphBackspace ⌫/BS`, `GlyphSpace ␣/Space` |
| Superscripts | `GlyphSuperscript0–9`, `GlyphSuperscriptPlus`, `GlyphSuperscriptMinus` |

### 4.2 Missing primitives

Three patterns recur across multiple panes / components and would benefit from
a primitive:

1. **`PlaybackControls`** — currently hand-rolled in `controls.go`; would
   centralise mode resolution for all 7 transport glyphs and the active-state
   colour role.
2. **Wider `EmptyState` adoption** — primitive exists, but `nowplaying.go`,
   `devices.go`, and `recentlyplayed_pane.go` ship custom empty messages.
3. **Shared spinner-frame source** — both `uikit.Spinner` and `cliout.Spinner`
   need the same braille / ASCII frame arrays. Today `cliout` duplicates
   them and has no ASCII variant. Expose as exported `uikit.SpinnerFrames(mode)`.

### 4.3 Visualizer ASCII path

The audio visualizer (`internal/ui/components/viz/`) has two renderers —
braille and block — both unicode-only. There is no per-render-call mode switch.
In ASCII mode today the visualizer either renders mojibake (terminal cannot
draw braille) or a column of unicode `█` (single-glyph block).

**Decision: engine-level renderer switch.** Add a new `AsciiBarsRenderer` that
draws columns using `#` (filled), `=` (half), `.` (empty). Bar-height
resolution is approximated to 4 levels (vs. 8 for blocks, 256 for braille).
At engine init the visualizer selects between `BrailleRenderer`,
`BlockRenderer`, and `AsciiBarsRenderer` based on `uikit.ActiveMode()` plus
the existing `viz.style` config knob (which currently chooses braille vs.
block in unicode mode). The visualizer stays present in ASCII mode at reduced
resolution rather than disappearing.

### 4.4 cliout ↔ uikit dependency direction

`cliout` is structurally independent of `uikit` despite the design doc's
explicit "shared catalogue" rule (§7). It also has no glyph fallback of its
own — `pinASCII` only strips the colour profile (`lipgloss.SetColorProfile(termenv.Ascii)`),
glyphs remain hardcoded unicode regardless of mode.

The fix is not just adding an ASCII branch inside `cliout` — that would
re-introduce the same drift risk. **`cliout` must import `uikit` and call
`GlyphFor` directly**, so the shared-catalogue rule becomes compiler-enforced
instead of documentation-enforced.

`uikit` and `cliout` remain peer packages (different rendering models — Bubble
Tea components vs. fire-and-forget stdout writes). The dependency is
unidirectional: `cliout` → `uikit`. `uikit` never imports `cliout`.

Import surface from `uikit` into `cliout`:

- `GlyphRole` constants used by cliout (Success, Error, Warning, Info, Active,
  Inactive, Locked, RateLimit, Running)
- `GlyphFor(role, mode) string`
- `ActiveMode() GlyphMode`
- A new `SpinnerFrames(mode GlyphMode) []string` — single source of truth
  for both packages' spinners.

`cliout.SetTestMode` and `uikit.SetModeForTest` stay separate because they
serve different test harnesses, but `cliout.SetTestMode` should additionally
call `uikit.SetModeForTest(GlyphASCII)` so the two stay aligned.

### 4.5 Width-aware layout safety

Most multi-column ASCII fallbacks (`◉` → `(*)`, `⏭` → `>>`) live inside
`ListRow`, `LockedRow`, `Chip`, `StatusGlyph` — all of which use
`lipgloss.Width` for sizing. No critical width-safety bug found in `uikit`.

The risk lives at call sites that bypass the primitives. The two known
offenders (`controls.go`, `infobox.go`) are both folded into Phase 3 — the
new `uikit.PlaybackControls` primitive owns playback-strip width arithmetic;
`infobox.go` migrates to `uikit.PaneChrome` which already handles width.
After Phase 3 there are no remaining width-arithmetic risks specific to
glyph swapping.

## 5. Severity classification

| Severity | Definition |
|---|---|
| **Critical** | In ASCII mode the surface is broken: mojibake, blank, or visually corrupt. Must fix before declaring ASCII support. |
| **Moderate** | In ASCII mode the surface renders but bypasses the design system's contract; risks future drift and inconsistent feel. Fix as part of the rollout. |
| **Minor** | Single hardcoded glyph in an otherwise compliant file. Low impact, low effort. Bundle into the rollout. |
| **Test gap** | Code likely correct but unverified. Add tests during the rollout. |

Counts:

| Area | Critical | Moderate | Minor | Test gap |
|---|---|---|---|---|
| uikit | 0 | 0 | 5 | 4 |
| panes | 2 | 7 | 4 | 0 |
| components + app + layout | 2 | 4 | 4 | 0 |
| cliout | 4 | 0 | 0 | 1 (whole package) |
| pane chrome (§3.5) | 6 | 0 | 0 | 0 |
| **Total** | **14** | **11** | **13** | **5** |

Note: §3.5 entries overlap §3.2 (panes) and §3.3 (components + app) by file but
not by gap — §3.2/§3.3 listed glyph leaks inside pane content; §3.5 lists the
border-chrome gap on the same files. They are tracked separately because they
require different fixes (chrome migration vs. inline glyph routing).

## 6. Remediation plan (phased)

Six phases. Each phase has a single concern; later phases depend on earlier
ones.

### Phase 1 — catalogue & shared infrastructure

Foundation. Adds every catalogue entry the later phases will reference, fixes
the small uikit self-audit gaps, and exports the shared spinner-frame helper.

- Add the missing `GlyphRole` constants from §4.1 (`GlyphSeparator`,
  `GlyphPlaylist`, `GlyphDeviceComputer`, `GlyphDevicePhone`,
  `GlyphDeviceSpeaker`, `GlyphDeviceTV`) plus the §4.9 keyboard chords
  (`GlyphEnter`, `GlyphEscape`, `GlyphTab`, `GlyphBackspace`, `GlyphSpace`)
  and §4.10 superscripts (`GlyphSuperscript0–9`, `GlyphSuperscriptPlus`,
  `GlyphSuperscriptMinus`) to `uikit/glyph.go` and `glyphTable`.
- Update `docs/TUI-DESIGN-SYSTEM.md` §4 in the same commit.
- Export `uikit.SpinnerFrames(mode GlyphMode) []string` — single source of
  braille (unicode) / `|/-\` (ASCII) frame arrays.
- Fix uikit self-audit gaps: `list_row.go:48` and `toast.go:112` ellipsis,
  `header_bar.go:52` separator, `key_bar.go:31–34` separator, `status_bar.go`
  glyph-field population.
- Add ASCII snapshot tests for `EmptyState`, `URLBox`, `HeaderBar`,
  `FormField`.

### Phase 2 — cliout integration

cliout becomes a one-way consumer of uikit's catalogue and mode resolver.

- Add `import "github.com/initgrep-apps/spotnik/internal/uikit"` to cliout.
- Replace `statusGlyph()` (`message.go:36–53`) with a `Status → GlyphRole`
  map plus `uikit.GlyphFor(role, uikit.ActiveMode())`.
- Replace `Hint.render()` raw `→` (`message.go:176`) with
  `uikit.GlyphFor(uikit.GlyphInfo, uikit.ActiveMode())`.
- Replace `spinner.go:16` `spinnerFrames` with
  `uikit.SpinnerFrames(uikit.ActiveMode())` resolved once at spinner start.
- Wire `cmd/root.go` so its existing `uikit.Use(cfg.UI.Glyphs)` call covers
  cliout — cliout reads `uikit.ActiveMode()` and does not duplicate
  `ResolveMode`.
- Update `cliout.SetTestMode` to additionally call
  `uikit.SetModeForTest(GlyphASCII)`.
- Document `pinASCII` in-code as colour-only (it controls
  `lipgloss.SetColorProfile`, not glyphs).
- Rebaseline cliout tests: parameterise glyph assertions via the catalogue
  lookup so tests document the role, not the codepoint, and run in both
  unicode and ASCII modes.

### Phase 3 — pane chrome migration

The single largest fallback gap. Migrates every border-rendering call site to
the `uikit` chrome primitives so glyph fields are always populated.

- `internal/app/render.go` `renderGrid` (`:417–426`) → replace inline
  `BorderConfig` build + `layout.RenderPaneBorder` call with
  `uikit.PaneChrome.Render` per pane.
- Direct `RenderPaneBorder` callers in panes:
    - `panes/themes.go:160`, `panes/profile.go:183` → `uikit.PaneChrome.Render`.
    - `panes/devices.go:166–175`, `panes/help_overlay.go:152–161`,
      `panes/search.go:778, 880, 943` → `uikit.OverlayChrome.Render`.
- `ui/components/infobox.go` → migrate to `uikit.PaneChrome.Render`.

After Phase 3 the only two callers of `layout.RenderPaneBorder` are
`uikit.PaneChrome.Render` and `uikit.OverlayChrome.Render`. Add a lint or
test guard to enforce this.

### Phase 4 — critical content fixes

Remaining critical glyph leaks that don't belong to chrome.

- Introduce `uikit.PlaybackControls` primitive owning the 7 transport
  glyphs; migrate `ui/components/controls.go:36–57` to use it.
- Move bubbleup alert-definition registration into `uikit.Toast` so the
  prefixes in `ui/components/notifications.go:30–50` resolve via `GlyphFor`
  after `uikit.Use`.
- `panes/nowplaying.go:85,87,91` Title → `GlyphFor` for `▶`/`⏸`/`─`.
- Visualizer: add `viz.AsciiBarsRenderer`; engine selects renderer based on
  `uikit.ActiveMode()` (per §4.3).

### Phase 5 — pane content cleanup

Remaining moderate / minor inline glyph leaks. All independent of one another;
can ship as one PR or split.

- `panes/devices.go` → swap raw `◉/○` (`:209,212,219,222`) for
  `uikit.StatusGlyph` with `GlyphActive`/`GlyphAvailable`; swap custom
  `⊡⊞⊟⊠` device-type icons (`:258–264`) for `GlyphDeviceComputer`,
  `GlyphDevicePhone`, `GlyphDeviceSpeaker`, `GlyphDeviceTV`; swap empty
  message (`:145–147`) for `uikit.EmptyState`.
- `panes/search.go:214` → swap `bubbles/spinner.Model` for `uikit.Spinner`.
- `panes/search_delegate.go:62–77` → `categorySymbol` resolves via
  `GlyphFor` (`GlyphMusicNote`, `GlyphPinned`, `GlyphInactive`,
  `GlyphPlaylist`, `GlyphSeparator`); separators at `:95,335` via
  `GlyphVRule` / `GlyphSeparator`.
- `panes/recentlyplayed_pane.go:134`, `panes/nowplaying.go:314–319` empty
  messages → `uikit.EmptyState`.
- `panes/networklog_pane.go:275,277` → `GlyphFor` for `◷`/`⚡`.
- `panes/profile.go:228` → `GlyphFor(GlyphEllipsis, mode)`.
- `panes/help_overlay.go:140` → `GlyphFor(GlyphVRule, mode)`.
- `panes/gateway_health_pane.go:169–184` → migrate dot-bar to `uikit.ProgressBar`.
- `app/render.go:132` (`♪`), `:320–322` (`•`×3), `:526,551` (`…`) →
  `GlyphFor` for `GlyphMusicNote`, `GlyphBullet`, `GlyphEllipsis`.
- `ui/components/gradient.go:197,200` → `GlyphFor(GlyphMusicNote, mode)`.
- `ui/components/table.go:13` `playingSymbol` const → lazy-resolve via
  `GlyphFor(GlyphPlaying, ActiveMode())` at render time.
- `ui/components/viz/block.go:45` → `GlyphFor(GlyphBarFull, mode)`.

### Phase 6 — test parity & smoke

- ASCII test sweep for `cliout` (every output assertion runs in both modes).
- Smoke test under `LANG=C` and `ui.glyphs = "ascii"` covering every pane,
  every overlay, splash, onboarding, error states.
- CI matrix: full test suite once with `LANG=en_US.UTF-8`, once with `LANG=C`.
- Banned-glyph grep guard (`⚠`, `ᐅ`, `┌┐└┘`, `╔╗╚╝`, `✅`, `❌`, `❗`)
  added to CI.
- Catalogue grep guard: unicode catalogue characters appear only in
  `uikit/glyph.go` and the canonical doc files.

## 7. Acceptance criteria

A reviewer can declare full glyph fallback support when **all** of the
following are true:

1. Grep across `internal/{uikit,ui,app,cliout}` for the unicode catalogue
   characters returns matches **only** in `uikit/glyph.go` (the table) and
   the canonical doc files.
2. Every primitive in `uikit` has a snapshot test that runs under
   `SetModeForTest(GlyphASCII)`.
3. `cliout` imports `uikit` and resolves every glyph and spinner frame via
   `GlyphFor` / `SpinnerFrames(mode)`.
4. Setting `ui.glyphs = "ascii"` in `config.toml` and launching the app
   produces a fully readable TUI: no mojibake, no missing borders, no broken
   width arithmetic.
5. The visualizer renders columns in ASCII mode (per Phase 4 plan).
6. CI runs the full test suite once with `LANG=en_US.UTF-8` and once with
   `LANG=C` — both pass.
7. Banned-glyph grep (`⚠`, `ᐅ`, `┌┐└┘`, `╔╗╚╝`, `✅`, `❌`, `❗`) returns
   zero hits across the four packages.

## 8. Out of scope

- Theme system changes (already complete via Feature 16).
- Layout grid / preset / focus rotation — `docs/DESIGN.md`.
- New visual primitives beyond `PlaybackControls` and the device-icon
  helpers (e.g., breadcrumbs, sortable headers) — defer to feature work.
- East-Asian-width handling beyond what `lipgloss.Width` already covers.

---

## Appendix A — file-level audit summary

Total files scanned: 73 source files across `uikit` (16), panes (~30),
components (~10 + viz/), app (~15), layout (~5), cliout (9).

Files with zero gaps (sample): `queue.go`, `topartists_pane.go`,
`toptracks_pane.go`, `likedsongs_pane.go`, `albums_pane.go`, `gateway_live_pane.go`,
`polling_traffic_pane.go`, `splash.go`, `auth.go`, `filter.go`, `timeutil.go`,
`table_chrome.go`.

Files with critical gaps (must-fix): `app/render.go` (renderGrid),
`panes/themes.go`, `panes/profile.go`, `panes/devices.go`,
`panes/help_overlay.go`, `panes/search.go`, `controls.go`, `infobox.go`,
`nowplaying.go`, `viz/braille.go`, `viz/block.go`, `cliout/message.go`,
`cliout/spinner.go`.

## Appendix B — research provenance

This document was produced by four parallel read-only audit passes on
2026-04-29:

1. uikit primitive self-audit (16 files + tests)
2. panes audit (~30 files)
3. components + app + layout audit (~25 files)
4. cliout audit (9 files)

Findings synthesized into the gap inventory, severity matrix, and phased
remediation plan above. No code was modified during the audit — this document
is the input to the implementation plan that will follow.
