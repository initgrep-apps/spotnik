# TUI Design System — `internal/uikit`

**Date:** 2026-04-24
**Status:** early spec — subject to refinement during the writing-plans phase.
**Feature placeholder:** `NN-tui-design-system` (slot assigned when added to `docs/spec/00-overview.md`).

**Supersedes:** fragments of `docs/DESIGN.md` that describe pane chrome, header, status
bar, overlay rendering, and onboarding panel internals. `DESIGN.md` retains layout,
grid, page, and preset mechanics. The relationship is spelled out in §9.

**Sibling reference:** `docs/CLI-OUTPUT.md` — the CLI subcommand output spec. The TUI
design system described here is a peer, not a subset. The two packages share the glyph
catalogue (§3) and emphasis-role vocabulary (§4) but render through separate code paths.

---

## 1 — Problem

The TUI works today, but three systemic issues limit its consistency and longevity:

1. **No typed primitive vocabulary.** Pane borders, overlays, header chips, toasts,
   onboarding panels, and status bars are each hand-composed with lipgloss styles.
   Primitives leak into every pane file. Changing a rule (e.g. swap `⚠` for a better
   glyph, or add ascii fallback) requires touching dozens of call sites.
2. **No written rules for feedback.** Toasts, inline flashes, status-glyph lines, and
   hint bars overlap in purpose. "Copied!" uses a 2-second inline flash in onboarding
   but toasts elsewhere. No single source of truth tells implementers which surface to
   use for what.
3. **Unicode assumed everywhere.** Rounded borders, block-fill bars, braille dots, and
   variation-selector-sensitive glyphs (`⚠`) all assume a modern unicode terminal.
   There is no ascii fallback path, and no user control.

## 2 — Goals

1. **Formal TUI primitive taxonomy** — typed, documented, tested. 18 primitives cover
   every visible surface. No ad-hoc hand composition at call sites.
2. **Reusable Go package** — `internal/uikit`. Panes, overlays, and app-level render
   code import from here. No more ad-hoc `lipgloss.NewStyle()` chains for primitives.
3. **Canonical reference doc** — `docs/TUI-DESIGN-SYSTEM.md` (written as part of
   Phase 1 of the migration, derived from this spec). Treated as a living spec like
   `CLI-OUTPUT.md` / `keybinding.md`.
4. **Glyph catalogue frozen, with ascii fallback** — one table, never extended ad hoc.
   `ui.glyphs = "auto" | "unicode" | "ascii"` config mirrors `cli.palette`.
5. **Feedback rules explicit** — six surfaces (Toast, StatusGlyph, EmptyState, KeyBar,
   StatusBar, PaneChrome filter preamble) each have a single reason to exist.
6. **Consistency between CLI and TUI** — glyph swaps (notably `⚠` → `◬`) propagate to
   `internal/cliout` in the same PR.

## 3 — Non-goals

- CLI subcommand output — already governed by `docs/CLI-OUTPUT.md`.
- Layout / grid / page / preset mechanics — remain in `docs/DESIGN.md`.
- Cross-application UI framework — this is spotnik-specific.
- Theme palette changes — the existing Theme interface is rich enough. This spec maps
  primitives onto existing tokens; it does not add or remove tokens.
- A new preset, a new pane, a new keybinding — orthogonal work.

## 4 — Decisions (answers to brainstorming questions)

| # | Question | Decision |
|---|---|---|
| Q1 | Package shape | New package `internal/uikit` mirroring `internal/cliout`'s pattern. Does not touch cliout. May share glyph/role vocabulary via a shared micro-package later if useful; out of scope now. |
| Q2 | Glyph fallback | `ui.glyphs = "auto" \| "unicode" \| "ascii"`; auto inspects `LANG`/`LC_ALL` for UTF-8. **Full fallback** — every glyph, including progress bars and visualizer, has an ascii form. |
| Q3 | Catalogue size | 18 primitives (Large scope). Rationale: TUI has broader user-facing surface than CLI; typed coverage prevents future drift. |
| Q4 | Onboarding model | Onboarding is a Panel with a FormField — no bespoke onboarding primitive. All keybars use the same `key.Binding` API as the status bar. Copy confirmation uses `Toast`, not an inline flash. |

## 5 — Frozen glyph catalogue

Every glyph the TUI and CLI use. Every row has a unicode form and an ascii fallback.
New glyphs require a PR that updates this table. Removed glyphs are flagged "banned".

### 5.1 Structural / borders

| Role | Unicode | ASCII | Notes |
|---|---|---|---|
| corner rounded | `╭ ╮ ╰ ╯` | `+ + + +` | Default pane, overlay, panel chrome |
| corner sharp | `┌ ┐ └ ┘` | — | **Banned** (DESIGN.md rule) |
| corner double | `╔ ╗ ╚ ╝` | — | **Banned** |
| horizontal rule | `─` | `-` | |
| vertical rule | `│` | `\|` | |
| double horizontal | `═` | `=` | Reserved — section break inside prose (currently unused) |
| tee / cross | `├ ┤ ┬ ┴ ┼` | `+` | Table row separators (future) |
| overlay dismiss | `×` | `x` | Close glyph on overlays (if shown) |

### 5.2 Intent / feedback

| Role | Unicode | ASCII | Where used |
|---|---|---|---|
| success | `✓` | `+` | Toast success, validation pass, saved confirmations |
| failure | `✗` | `x` | Toast error, validation fail |
| warning | `◬` | `!` | Toast warning, Premium-required line, StatusGlyph warning |
| info / hint arrow | `→` | `>` | Toast info, inline hint arrow |
| rate-limit / wait | `⧖` | `~` | Rate-limit toast |
| running / bolt | `⚡` | `*` | Active auto-traffic indicator |
| deadline / clock | `◷` | `@` | Timeout, expiry (infobox future) |
| paused-state | `⏸` | `\|\|` | Non-playback pause (auto-traffic paused) |
| blocked / no-entry | `⊘` | `(-)` | Action refused — reserved for future "cannot" states |

**Banned:** `⚠` (variation-selector sensitive, renders as emoji on many terminals),
`✅` `❌` `❗` (emoji).

### 5.3 State / availability

| Role | Unicode | ASCII | Where used |
|---|---|---|---|
| active / on | `◉` | `(*)` | Device chip active, playing indicator |
| inactive | `◎` | `( )` | Pending, dim state |
| available / free-tier | `○` | `(o)` | Profile free-tier, empty slot |
| filled dot | `●` | `(#)` | Count indicator, progress step done |
| empty square | `□` | `[ ]` | Checkbox off (future) |
| filled square | `■` | `[x]` | Checkbox on (future) |
| locked / readonly | `◌` | `(r)` | **New primitive-backed role.** Inaccessible playlist row (Spotify-owned), read-only items |
| pinned / starred | `★` | `*` | Starred item, pinned playlist (future) |
| unpinned | `☆` | `-` | Optional counterpart |
| bullet | `•` | `*` | Prose lists |

### 5.4 Navigation / scroll

| Role | Unicode | ASCII | Where used |
|---|---|---|---|
| scroll down | `▼` | `v` | More content below |
| scroll up | `▲` | `^` | More content above |
| scroll right | `►` | `>` | Horizontal overflow |
| scroll left | `◄` | `<` | Horizontal overflow |
| sort asc | `▲` | `^` | Table column sort (future) |
| sort desc | `▼` | `v` | Table column sort (future) |
| ellipsis | `…` | `...` | Truncation |
| chevron R | `›` | `>` | Breadcrumbs, sub-views |
| chevron L | `‹` | `<` | Back |
| key arrow L / R / U / D | `← → ↑ ↓` | `<- -> ^ v` | Help overlay display |
| key arrow LR | `↔` | `<>` | |

**Banned:** `ᐅ` (U+1405 Canadian Syllabics Pa). Any action hint in any mode uses
**corner-notch format** (`╮ key label ╭`) — not a prefix character. Filter-mode hints
also use notch format; `filtering: "query"` renders as muted preamble before the notch.

### 5.5 Playback controls

| Role | Unicode | ASCII |
|---|---|---|
| playing | `▶` | `>` |
| paused | `▷` | `\|>` |
| stop | `■` | `[]` |
| next track | `⏭` | `>>` |
| prev track | `⏮` | `<<` |
| ffwd | `⏩` | `>>>` |
| rewind | `⏪` | `<<<` |
| shuffle | `⇄` | `sh` |
| repeat all | `↻` | `rp` |
| repeat one | `↻¹` | `rp1` |
| repeat off | `⟳` | `ro` |
| queue | `≡` | `Q` |
| eject / disconnect | `⏏` | `/E` |

### 5.6 Domain / music / identity

| Role | Unicode | ASCII |
|---|---|---|
| music note | `♪` | `*` |
| double note | `♫` | `**` |
| premium badge | `♛` | `*P` |
| free-tier badge | `○` | `(o)` |
| cloud / remote device | `☁` | `(c)` |

### 5.7 Graphical fills (ProgressBar, Visualizer)

| Role | Unicode | ASCII |
|---|---|---|
| bar full | `█` | `#` |
| bar 7/8 | `▉` | `#` |
| bar 3/4 | `▊` | `#` |
| bar 5/8 | `▋` | `=` |
| bar 1/2 | `▌` | `=` |
| bar 3/8 | `▍` | `-` |
| bar 1/4 | `▎` | `-` |
| bar 1/8 | `▏` | `.` |
| bar empty | `░` | `.` |
| bar medium | `▒` | `:` |
| bar heavy | `▓` | `#` |
| braille cells (256 combos) | `⠀⠁…⣿` | `.` / `#` collapsed by dot-density |

### 5.8 Spinner frames

| Set | Unicode | ASCII |
|---|---|---|
| braille (default) | `⣾⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` | `\|/-\|/-\` |

### 5.9 Keyboard chords

Keyboard-chord glyphs are **text-first**. Only arrow keys, Enter, and Esc may use
glyph form; modifier keys (Ctrl, Alt, Shift, Cmd) always render as text for
cross-platform readability.

| Role | Unicode | ASCII |
|---|---|---|
| enter | `⏎` / `↵` | `Enter` |
| escape | `⎋` | `Esc` |
| tab | `⇥` | `Tab` |
| shift | — | `Shift` |
| backspace | `⌫` | `BS` |
| space | `␣` | `Space` |
| ctrl / alt / cmd | — | `Ctrl` / `Alt` / `Cmd` |

### 5.10 Superscripts

Used in pane titles (toggle-key number) and repeat-one indicator. No ascii equivalent
beyond regular digits.

| Role | Unicode | ASCII |
|---|---|---|
| 1–8 | `¹ ² ³ ⁴ ⁵ ⁶ ⁷ ⁸` | `1 2 3 4 5 6 7 8` |
| 0, 9 | `⁰ ⁹` | `0 9` |
| +, − | `⁺ ⁻` | `+ -` |

### 5.11 Detection

Resolution order on first `uikit.Render` call (lazy, `sync.Once`):

1. **`ui.glyphs` config** (`"auto"`, `"unicode"`, `"ascii"`). Default `"auto"`.
2. If `"auto"`:
   - `LC_ALL` or `LANG` contains `UTF-8` or `utf8` → unicode.
   - Else → ascii.
3. `NO_COLOR` is **orthogonal** — it strips colour, not glyphs.

---

## 6 — Emphasis roles (colour matrix)

### 6.1 Roles

| Role | Default token | Intent |
|---|---|---|
| **Accent** | `theme.Accent()` (falls back to `theme.SeekBar()`) | Interactive / call-to-action — keys, URLs, filter query, focus cues |
| **Strong** | `theme.TextPrimary()` + bold | Primary headlines — pane title, panel title, section caps |
| **Plain** | `theme.TextPrimary()` | Body content — track name, value, description |
| **Muted** | `theme.TextMuted()` | Labels, captions, secondary prose, action-key descriptions |
| **Success** | `theme.Success()` | Success toasts, premium badge, playing indicator |
| **Error** | `theme.Error()` | Error toasts, validation fail |
| **Warning** | `theme.Warning()` | Warning toasts, Premium-required line |
| **Info** | `theme.Info()` | Info toasts, hint arrows |
| **Selection** | `theme.SelectedFg()` | Focused row in list/table |
| **Column-Index/Primary/Secondary/Tertiary** | `theme.ColumnIndex/Primary/Secondary/Tertiary()` | Table columns |
| **PaneBorder-<ID>** | `theme.PaneBorderX()` per pane | Pane-chrome border, dims automatically when unfocused |

Call sites set a **role**, never a raw colour.

### 6.2 Field-role mapping

| Primitive.Field | Role |
|---|---|
| `PaneChrome.Border (focused)` | PaneBorder-<ID> |
| `PaneChrome.Border (unfocused)` | Muted PaneBorder-<ID> |
| `PaneChrome.ToggleKey` (¹..⁸) | Accent |
| `PaneChrome.Title (focused)` | Strong |
| `PaneChrome.Title (unfocused)` | Plain |
| `PaneChrome.Action.Key` (notch) | Accent |
| `PaneChrome.Action.Label` (notch) | Muted |
| `PaneChrome.FilterPreamble` label | Muted |
| `PaneChrome.FilterPreamble` query | Accent |
| `OverlayChrome.Border` | Accent |
| `OverlayChrome.Title` | Strong |
| `OverlayChrome.Action.Key/Label` | Accent / Muted |
| `Panel.Border (default)` | Accent |
| `Panel.Border (error)` | Error |
| `Panel.Title` | Strong |
| `TableChrome.Header` | `theme.TableHeader()` |
| `TableChrome.Cell.Index` | Column-Index |
| `TableChrome.Cell.Primary` | Column-Primary |
| `TableChrome.Cell.Secondary` | Column-Secondary |
| `TableChrome.Cell.Tertiary` | Column-Tertiary |
| `TableChrome.Cell (selected)` | Selection |
| `TableChrome.Cell.PlayingIndicator` | `theme.PlayingIndicator()` |
| `ListRow.Glyph` | matches row intent |
| `ListRow.Label` | Plain |
| `ListRow.Caption` | Muted |
| `LockedRow.Glyph` (`◌`) | Muted |
| `LockedRow.Label` | Muted (entire row dim) |
| `SectionLabel` | Parent pane's border token |
| `EmptyState.Text` | Muted |
| `EmptyState.Hint` | Muted |
| `URLBox.Border` | Muted |
| `URLBox.Content` | Accent |
| `HeaderBar.Bg` | `theme.StatusBarBg()` |
| `HeaderBar.AppName` | Strong |
| `HeaderBar.Separator` | Muted |
| `HeaderBar.PageKey` (A/B) | Accent |
| `HeaderBar.PresetLabel` | Muted |
| `StatusBar.Bg` | `theme.StatusBarBg()` |
| `StatusBar.Key` | `theme.KeyHint()` |
| `StatusBar.Desc` | Muted |
| `KeyBar.Key` | `theme.KeyHint()` |
| `KeyBar.Desc` | Muted |
| `KeyBar.Separator` | Muted |
| `Chip.Glyph` | intent role |
| `Chip.Label` | `theme.HeaderChipFg()` |
| `Chip.Bg` | `theme.StatusBarBg()` |
| `FormField.Label` | Muted |
| `FormField.Input.Text` | Plain |
| `FormField.Input.Cursor` | Accent |
| `FormField.ValidationError` | Error glyph + Plain text |
| `Toast.Glyph` | intent role |
| `Toast.Title` | Strong |
| `Toast.Body` | Plain |
| `StatusGlyph` | intent role |
| `ProgressBar.Fill` | `theme.Gradient1/2/3()` per position |
| `ProgressBar.Empty` | Muted |
| `Spinner.Frame` | Accent |
| `Spinner.Text` | Muted |

### 6.3 Rules enforced by the matrix

- Only **Accent** signals "you can press this" — keys, URLs, interactive cues.
  Informational values are Plain.
- **Strong** is bold, not bright — contrast through weight.
- One Accent per call-to-action — an action key OR an action URL, never both wrapped
  into the same span.
- `HeaderBar` and `StatusBar` share `StatusBarBg()` to visually bracket the grid.

---

## 7 — Primitive catalogue

Eighteen primitives. Every one is documented in `docs/TUI-DESIGN-SYSTEM.md` using the
contract format in §7.2. This spec shows three worked examples (PaneChrome, Toast,
Spinner) and a summary table for the remaining fifteen. Each remaining primitive gets
its full contract in the implementation-plan story that introduces it.

### 7.1 Summary table

| # | Primitive | Role | Replaces / formalises |
|---|---|---|---|
| 1 | `PaneChrome` | Pane border with title, toggle key, right-side actions | `layout.RenderPaneBorder` (wrapped) |
| 2 | `OverlayChrome` | Floating panel on dimmed background | 5 `renderWith*Overlay` funcs in `render.go` |
| 3 | `Panel` | Full-screen centered framed panel | `renderOnboarding*`, `renderAuthPanel`, `renderTooSmall`, `renderSplash` |
| 4 | `TableChrome` | Column header + themed rows + playing indicator | Wraps `components/table.go` |
| 5 | `ListRow` | Single-line row with glyph + label + optional caption | Hand-composed strings in theme overlay, profile pane |
| 6 | `LockedRow` | Disabled/inaccessible variant (dim + `◌` glyph) | **Gap today** — user-flagged |
| 7 | `SectionLabel` | Caps label marking a sub-section inside a pane | Page B labels (GATEWAY, APP, …) |
| 8 | `EmptyState` | "No data" message with optional hint | Each pane hand-rolls its own |
| 9 | `URLBox` | Muted-border block wrapping code / URL content | `uriBoxStyle` / `urlBoxStyle` inline |
| 10 | `HeaderBar` | Top app bar: name · page · preset · right chips | `renderHeader` in `render.go` |
| 11 | `StatusBar` | Bottom global key bar with 2-column help layout | `renderStatusBar` in `render.go` |
| 12 | `KeyBar` | Reusable `key:label · key:label` strip | bubbles/help wrapper + hand-rolled search keybar |
| 13 | `Chip` | Inline pill: glyph + label on status-bar background | `renderProfileChip`, device chip builder |
| 14 | `FormField` | Labelled input with validation | Inline input in `renderOnboardingRegister` |
| 15 | `Toast` | Typed notification (intent, title, body) | `a.alerts.NewAlertCmd` raw-string calls |
| 16 | `StatusGlyph` | Atomic glyph with intent colour | Ad-hoc `✓` / `✗` / `◉` usages |
| 17 | `ProgressBar` | Fill bar with unicode + ascii modes | `controls.go` seek and volume |
| 18 | `Spinner` | Animated wait with Done/Fail/Cancel contract | `onboardingSpinner` (bubbles/spinner wrapper) |

Note: the 19-primitive draft collapsed `Hint` into `KeyBar` — there is one widget,
used in two contexts (app StatusBar composition vs. inline placements).

### 7.2 Contract template

Every primitive in `docs/TUI-DESIGN-SYSTEM.md` is documented with the same six
blocks:

```
### N.M <PrimitiveName>

Purpose: one-line reason for existence.

Fields:
  <Go struct literal>

Rendering:
  ascii art of unicode form
  ascii art of ascii form

Roles:
  field → role (from §6.2)

Glyphs used:
  list of roles from §5

Lifecycle:
  stateless | owns-state | animated

Tests:
  what must be asserted (unicode + ascii snapshot; role assertion)
```

### 7.3 Worked example — PaneChrome

```
Purpose:
  Standard bordered pane with title, toggle-key superscript, and right-aligned
  action hints. Every pane uses this. Wraps layout.RenderPaneBorder; the legacy
  function becomes the internal implementation.

Fields:
  type PaneChrome struct {
    Width, Height int
    Title         string
    ToggleKey     int       // 0 = no key shown
    Actions       []Action  // {Key, Label}
    AccentToken   ThemeFn   // per-pane border token
    Focused       bool
    FilterQuery   string    // "" = no filter mode
  }
  type Action struct { Key, Label string }

Rendering (unicode, actions mode):
  ╭─ ³Playlists────────────────╮ f filter ╭─╮ n new ╭╮
  │  (content)                                      │
  ╰─────────────────────────────────────────────────╯

Rendering (unicode, no actions):
  ╭─ ³Playlists─────────────────────────────────────╮
  │  (content)                                      │
  ╰─────────────────────────────────────────────────╯

Rendering (unicode, filter mode — notch format, no ᐅ):
  ╭─ ³Playlists────filtering: "rock" ─╮ Esc close ╭╮
  │  (content)                                    │
  ╰───────────────────────────────────────────────╯

Rendering (ascii, actions mode):
  +- 3 Playlists---------------+ f filter +-+ n new ++
  |  (content)                                       |
  +---------------------------------------------------+

Roles:
  Border (focused)       → PaneBorder-<ID>
  Border (unfocused)     → Muted PaneBorder-<ID>
  ToggleKey (superscript)→ Accent
  Title (focused)        → Strong
  Title (unfocused)      → Plain
  Action.Key             → Accent
  Action.Label           → Muted
  FilterPreamble label   → Muted
  FilterPreamble query   → Accent

Glyphs:
  corners ╭╮╰╯ / +, horizontal rule ─ / -, vertical rule │ / |,
  superscript ¹…⁸ / 1…8

Lifecycle: stateless — all inputs via struct.

Format rules (explicit):
  - Title is rendered immediately after "─ " with NO trailing space.
    Dashes flush against it.
  - Each action sits in a notch: "╮ <key><space><label> ╭".
  - Notches are joined by a single ─.
  - The last notch's ╭ is immediately followed by the top-right corner ╮,
    producing ╭╮. This is intentional.
  - Filter mode does not use ᐅ. Preamble "filtering: "<query>"" renders
    muted; the "Esc close" notch joins with a single ─.

Tests:
  - corner characters match mode
  - width matches requested
  - height matches requested
  - focused title is Strong
  - action notches render correctly for 1, 2, 3 actions
  - filter mode emits preamble + notch (no ᐅ)
  - ascii mode swaps every glyph row
```

### 7.4 Worked example — Toast

```
Purpose:
  Typed notification surfaced via bubbleup. Replaces raw-string call sites:
    a.alerts.NewAlertCmd("error", msg)
  becomes
    a.toasts.New(Toast{Intent: Error, Title: "Spotify unreachable",
                       Body: "Retrying in 3s."})

Fields:
  type Toast struct {
    Intent ToastIntent     // Success | Error | Warning | Info | RateLimit
    Title  string          // required, ≤ 48 runes, sentence case, no trailing "."
    Body   string          // optional, ≤ 160 runes, 1 sentence, trailing "."
    TTL    time.Duration   // 0 = default per intent
  }

Default TTLs:
  Success   4s
  Info      4s
  Warning   5s
  Error     6s
  RateLimit = Retry-After seconds

Rendering (unicode, Error intent):
  ╭──────────────────────────╮
  │ ✗  Spotify unreachable   │
  │    Retrying in 3s.       │
  ╰──────────────────────────╯

Rendering (ascii):
  +--------------------------+
  | x  Spotify unreachable   |
  |    Retrying in 3s.       |
  +--------------------------+

Roles:
  Border → intent role
  Glyph  → intent role
  Title  → Strong
  Body   → Plain

Glyphs by intent:
  Success ✓/+ · Error ✗/x · Warning ◬/! · Info →/> · RateLimit ⧖/~

Lifecycle: owns-state (bubbleup animates lifetime).

Positioning:
  grid view        → viewport bottom-right
  panel view       → panel bottom-right (onboarding, auth, splash, too-small)
  overlay view     → viewport bottom-right (overlay stays focused)

Content rules:
  - Title: past-participle verb phrase for completions ("Copied", "Saved"),
    noun + state for async events ("Device disconnected", "Rate-limited").
  - Body: single sentence, trailing ".", optional for Success/Info,
    required for Error (must name the next step).
  - Sentence case, no emoji.
  - Hard-truncate Title at 48, Body at 160 with "…".

Tests:
  - intent colour matches theme token
  - default TTL per intent
  - Title and Body are truncated at the documented limits
  - ascii mode swaps glyph
  - RateLimit TTL honours Retry-After
  - position adapts per view-mode
```

### 7.5 Worked example — Spinner

```
Purpose:
  Unbounded-wait indicator. TUI version; cliout.Spinner is the CLI peer.

Fields:
  type Spinner struct {
    Text   string
    frame  int  // internal — advanced by SpinnerTickMsg
    done   bool // internal — true after Done/Fail
    result *resolution
  }
  type resolution struct {
    glyph string // ✓ or ✗
    text  string
    intent ToastIntent // Success or Error
    ttl   time.Duration
  }

Terminal states (mirror cliout.Spinner):
  Done(text)    → frame becomes ✓ in Success; text replaces .Text;
                  held ~1.2s; emits SpinnerDoneMsg; then primitive clears.
  Fail(text)    → frame becomes ✗ in Error; text replaces .Text;
                  held ~2s; emits SpinnerFailMsg{Err}; then primitive clears.
  Cancel()      → clears immediately, no final line; emits SpinnerCancelledMsg.

Rendering (unicode, animated):
  ⣾ Waiting for authorization

Rendering (unicode, Done resolution):
  ✓ Authorized

Rendering (unicode, Fail resolution):
  ✗ Authorization failed

Rendering (ascii):
  | Waiting for authorization   (rotating | / - \)
  + Authorized
  x Authorization failed

Roles:
  Frame  → Accent  (during animation)
  Frame  → Success (during Done hold)
  Frame  → Error   (during Fail hold)
  Text   → Muted

Applied to onboarding OAuth wait:
  OAuth succeeds  → Spinner.Done("Authorized"), 1.2s, then grid view +
                    Toast(Success, "Signed in as <user>")
  OAuth fails     → Spinner.Fail("Authorization failed"), 2s, then error
                    panel + Toast(Error, <detail>, "Re-enter Client ID")
  User cancels q  → Spinner.Cancel(), quit

Tests:
  - frame advances on SpinnerTickMsg
  - Done replaces frame with ✓ and emits SpinnerDoneMsg after TTL
  - Fail replaces frame with ✗ and emits SpinnerFailMsg after TTL
  - Cancel clears without a final line
  - ascii mode uses rotating `|/-\` frames
```

### 7.6 Remaining primitives (stubs)

Fifteen primitives still to contract. Each will be documented in
`docs/TUI-DESIGN-SYSTEM.md` in the story that introduces it
(S4–S17 in §8.2). One-line stub per primitive follows; formal
contracts live in their implementation stories.

- **OverlayChrome** — centered floating panel on dimmed background; consolidates
  Search, Profile, Theme, Help, Device overlay rendering.
- **Panel** — full-screen bordered panel with title in the border top. Absorbs the
  step-header role (panel title IS the step header). Used by onboarding, auth,
  too-small, splash.
- **TableChrome** — wraps `components/table.go`; reads column tokens from
  `theme.Column*`; adds ascii fallback.
- **ListRow** — single-line item: optional glyph, label, optional caption.
- **LockedRow** — dim variant of `ListRow` with leading `◌`; entire row is Muted.
- **SectionLabel** — caps label in parent pane's accent token, underlined by a
  `─` rule; used for sub-sections inside Page B panes.
- **EmptyState** — Muted centered text + optional hint; replaces hand-rolled
  "no data" messages.
- **URLBox** — muted-border rectangle wrapping accent-coloured URL or code
  content; wraps long URLs at `&` boundaries (current `wrapURL` helper).
- **HeaderBar** — app name · page · preset · device chip · profile chip.
  Extracts `renderHeader` from `render.go`.
- **StatusBar** — bottom key bar composition over `KeyBar`; uses `bubbles/help`
  for ShortHelp/FullHelp.
- **KeyBar** — stateless strip taking `[]key.Binding`; renders `key:desc ·
  key:desc`. Single underlying primitive for status-bar body, overlay footers,
  and inline hints.
- **Chip** — inline pill with leading glyph + label on `StatusBarBg`.
- **FormField** — labelled input; intrinsic validation; error slot beneath.
  Wraps `bubbles/textinput`.
- **StatusGlyph** — atomic glyph with intent colour; replaces scattered `✓`/`✗`/`◉`.
- **ProgressBar** — seek/volume with unicode partial-block and ascii fallback
  (`=`/`-`/`.`).

---

## 8 — Migration plan

### 8.1 Shape

One feature: `NN-tui-design-system`. Nineteen stories across five phases. Phase 1
is a hard gate. Phases 2–4 have internal parallelism. Phase 5 depends on
everything.

### 8.2 Phases and stories

```
Phase 1 — Foundation (gate)
  S1  internal/uikit scaffold: package, ui.glyphs config, auto-detect,
      glyph registry, role registry, Capture test helper
  S2  layout/border.go: remove ᐅ, add ascii mode, rewrite DESIGN.md §5
      border anatomy section, flip border_test.go:742

Phase 2 — Chrome primitives (parallelizable)
  S3  PaneChrome
  S4  OverlayChrome — consolidate 5 renderWith*Overlay funcs
  S5  Panel — replaces renderOnboarding*, renderAuthPanel, renderTooSmall,
      renderSplash; panel title slot absorbs step header
  S6  HeaderBar + Chip — extract from render.go:renderHeader
  S7  StatusBar + KeyBar — migrate status bar; make KeyBar the reusable atom

Phase 3 — Content primitives
  S8  TableChrome — wraps components/table.go; no call-site changes
  S9  ListRow + LockedRow — migrate theme overlay, profile overlay, playlists
  S10 SectionLabel — Page B sub-sections
  S11 EmptyState — empty queue, empty search, loading playlist tracks
  S12 URLBox — onboarding

Phase 4 — Feedback
  S13 Toast — typed API wrapping bubbleup; position rules; migrate all
      a.alerts.NewAlertCmd call sites
  S14 StatusGlyph atom — swap ⚠ → ◬ across cliout, onboarding, remaining sites
  S15 ProgressBar — consolidate seek + volume; ascii fallback for partial blocks
  S16 Spinner — Done/Fail/Cancel contract; wire onboarding OAuth wait

Phase 5 — Composites & cleanup
  S17 FormField — migrate onboarding client-ID input
  S18 Onboarding end-to-end rewrite using all primitives
  S19 docs/DESIGN.md rewrite — strip primitive details, retain layout/grid/
      page mechanics; canonical pointer to docs/TUI-DESIGN-SYSTEM.md;
      update docs/PANE-TEMPLATE.md Step 2 scaffold to use PaneChrome
```

### 8.3 Sequencing

- **S1 and S2 must merge first.** Everything later depends on the uikit scaffold
  and the ᐅ-free border.
- **S3–S7** parallelise after S1/S2. No shared state between them.
- **S8–S12** parallelise after S3. S8 (TableChrome) is a no-op at call sites —
  safe to ship early.
- **S13–S16** parallelise after S1. Mostly independent of chrome primitives.
- **S17** blocks on S13 (Toast) and S15/S16 (feedback atoms).
- **S18** blocks on everything. Biggest visible PR.
- **S19** is the final PR. Doc-only.

### 8.4 Non-negotiables

- Existing test coverage (≥ 80%) never drops between PRs.
- Every primitive ships with unicode + ascii snapshot tests and role-token
  assertions.
- No story skips the ascii-mode test — the fallback is verified at merge time.
- `internal/uikit` coverage must be 100% — it's the shared vocabulary.
- No feature flag (`ui.design_system = on/off`). Each primitive is migrated
  at its call sites with visual diff in code review.

### 8.5 Risks and mitigations

| Risk | Mitigation |
|---|---|
| `bubbleup` toast position doesn't adapt per view-mode | `Toast.PositionHint` field; the caller passes current view-mode. Fallback: always bottom-right. |
| Ascii visualizer looks awful | Accepted. Users who care disable via `v` cycle (already exists) or set `ui.visualizer = "off"`. |
| `⚠` → `◬` breaks identity for existing users | `◬` is closer in shape than `!`. One-line release note. |
| 19 stories is large | Phase 5 is doc-heavy; Phases 2–4 parallelise; pipeline has handled similar sizes (Feature 11 ≈ 15 stories). |
| `FormField` design drifts from `bubbles/textinput` | S17 wraps `textinput.Model`, does not replace it. |

### 8.6 Per-story deliverables

Every story PR includes:

1. Primitive implementation (or migration).
2. Table-driven tests with unicode + ascii snapshots.
3. Primitive contract added to `docs/TUI-DESIGN-SYSTEM.md`.
4. Update to `docs/DESIGN.md` if touched code is referenced there.
5. Update to `docs/PANE-TEMPLATE.md` if the pane-authoring recipe changes.
6. `make ci` passes.

---

## 9 — Relationship to other docs

- **`docs/CLI-OUTPUT.md`** — peer. CLI subcommand output spec. Shares the glyph
  catalogue in §5 via the migration in S14 (`⚠` → `◬` propagates). The two
  packages render through separate code paths but agree on vocabulary.
- **`docs/DESIGN.md`** — narrower after this spec lands. DESIGN.md retains layout,
  grid, page, preset, and pane-toggle mechanics. Primitive rendering (border
  anatomy, header, status bar, overlay chrome, onboarding panel internals) moves
  to `docs/TUI-DESIGN-SYSTEM.md`. See S2 and S19.
- **`docs/PANE-TEMPLATE.md`** — updated by S19 to reference `PaneChrome` and the
  uikit primitives in Step 2 scaffold.
- **`docs/keybinding.md`** — unchanged. KeyBar consumes `key.Binding` which is
  what keybindings are defined with today.

---

## 10 — Rules summary (the "do not" list)

1. Do not compose primitives with raw `lipgloss.NewStyle()` at call sites — use
   the primitive's constructor.
2. Do not introduce a new glyph without updating §5 in the same PR.
3. Do not use `ᐅ` (U+1405) — banned.
4. Do not use `⚠` — use `◬`.
5. Do not use sharp corners `┌┐└┘` or double corners `╔╗╚╝` — banned.
6. Do not render action hints without the notch format.
7. Do not wrap both a key AND a URL in Accent inside the same call-to-action —
   one Accent per CTA.
8. Do not use StatusGlyph + text inline for things a Toast handles — toasts are
   for completion acknowledgements and async events; StatusGlyph is for
   persistent informational state.
9. Do not use `Hint` — it is not a primitive. Use `KeyBar`.
10. Do not render inline error boxes in pane `View()` methods. All API errors
    route through `Toast`.
11. Do not skip the ascii-mode test for a primitive.
12. Do not add a feature flag for the design system migration.

---

## 11 — Open questions

- **Shared glyph/role micro-package?** `internal/cliout` and `internal/uikit`
  share §5 and §6.1. A third package `internal/uitokens` could hold both. Cost
  of that is touching shipped `cliout`. Defer the decision until S14 lands and
  we see how painful the duplication is.
- **Keyboard-chord display on Windows terminals** — `Ctrl`/`Alt` as text works
  everywhere, but could we show `⌃`/`⌥` on macOS only via a config key? Deferred.
- **Nerd Fonts opt-in** — some users install Nerd Fonts and would welcome icons
  (`` for queue, ` ` for play, etc.). Out of scope; would be a fourth glyph
  mode alongside unicode/ascii. Deferred.
- **Animation in ascii mode** — spinner animates `|/-\`; visualizer doesn't
  animate meaningfully in ascii. Do we freeze visualizer in ascii mode, or
  alternate two frames? Deferred to S15.

---

## 12 — Change log

| Date | Change |
|---|---|
| 2026-04-24 | Initial spec. Brainstorming session between owner and assistant. |
