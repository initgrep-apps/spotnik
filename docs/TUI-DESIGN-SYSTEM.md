# TUI Design System — Spotnik

> Canonical reference for `internal/uikit` primitives, glyph catalogue, role matrix, and
> feedback surfaces. Peer to `docs/CLI-OUTPUT.md`. Layout mechanics (grid, pages, presets)
> live in `docs/DESIGN.md`.

---

## 1. Purpose

`internal/uikit` formalises every visible surface of the Spotnik TUI into 18 typed
primitives. Before it existed, every pane composed `lipgloss.NewStyle()` values inline —
duplicating spacing, colour logic, and border geometry. The package solves three problems:

- **Consistency** — one source of truth for border anatomy, glyph choices, and colour roles.
- **ASCII fallback** — every primitive renders correctly when `ui.glyphs = "ascii"`.
- **Testability** — `uikit.Capture` strips ANSI codes so tests assert structure, not escape
  sequences.

**What this document covers:** primitive contracts (struct fields, rendering, roles, glyphs,
lifecycle, tests), the frozen glyph catalogue, the role / colour matrix, and the six
non-overlapping feedback surfaces.

**What this document does NOT cover:** layout grid, pages, presets, pane toggling, or focus
rotation — those live in `docs/DESIGN.md`.

---

## 2. Hard Rules

### Do

- Use a primitive's constructor or struct literal — never compose raw `lipgloss.NewStyle()`
  at call sites.
- Set a **role** on every styled element — roles resolve to theme tokens (see §5).
- Provide an ASCII snapshot test for every primitive.
- Wrap action hints in the notch format (`╮ key label ╭`), never a prefix character.
- Use `◬` for warnings (unicode), `!` (ascii).

### Do Not

1. Compose primitives with raw `lipgloss.NewStyle()` at call sites — use the primitive.
2. Introduce a new glyph without updating §4 (Glyph catalogue) in the same PR **and**
   `docs/TUI-DESIGN-SYSTEM.md` in the same commit.
3. Use `ᐅ` (U+1405 Canadian Syllabics Pa) — banned. All action hints use notch format.
4. Use `⚠` — use `◬` (unicode) or `!` (ascii).
5. Use sharp corners `┌┐└┘` or double corners `╔╗╚╝` — banned.
6. Render action hints without the notch format.
7. Wrap both a key AND a URL in Accent inside the same call-to-action — one Accent per CTA.
8. Use `StatusGlyph` + text inline for things a `Toast` handles — toasts are for completion
   acknowledgements and async events; `StatusGlyph` is for persistent informational state.
9. Use `Hint` — it is not a primitive. Use `KeyBar`.
10. Render inline error boxes in pane `View()` methods. All API errors route through `Toast`.
11. Skip the ascii-mode test for a primitive.
12. Add a feature flag for the design-system migration.

---

## 3. Primitive Catalogue

Every primitive is documented with a six-block contract: **Purpose · Fields · Rendering ·
Roles · Glyphs · Lifecycle · Tests**.

### 3.1 PaneChrome

**Purpose:** Standard bordered pane with title, toggle-key superscript, and right-aligned
action notches. Every music and developer pane uses this. Wraps `layout.RenderPaneBorder`.

**Fields:**

```go
type PaneChrome struct {
    Width       int
    Height      int
    Title       string
    ToggleKey   int           // 0 = no key shown
    Actions     []layout.Action  // {Key, Label} pairs
    AccentColor lipgloss.Color   // per-pane border token via layout.PaneBorderColor
    Focused     bool
    FilterQuery string        // "" = normal mode
    Theme       theme.Theme
}
```

**Rendering (unicode, actions mode):**

```
╭─ ³Playlists────────────────╮ f filter ╭─╮ n new ╮
│  (content)                                      │
╰─────────────────────────────────────────────────╯
```

**Rendering (unicode, filter mode — notch format):**

```
╭─ ³Playlists────filtering: "rock" ─╮ Esc close ╮
│  (content)                                    │
╰───────────────────────────────────────────────╯
```

**Rendering (ascii, actions mode):**

```
+- 3 Playlists---------------+ f filter +-+ n new ++
|  (content)                                       |
+---------------------------------------------------+
```

**Roles:**

| Field | Role |
|---|---|
| Border (focused) | PaneBorder-<ID> |
| Border (unfocused) | Muted PaneBorder-<ID> |
| ToggleKey superscript | Accent |
| Title (focused) | Strong |
| Title (unfocused) | Plain |
| Action.Key | Accent |
| Action.Label | Muted |
| FilterPreamble label | Muted |
| FilterPreamble query | Accent |

**Glyphs:** corners `╭╮╰╯` / `+`, horizontal rule `─` / `-`, vertical rule `│` / `|`,
superscripts `¹…⁸` / `1…8`.

**Lifecycle:** stateless — all inputs via struct fields.

**Tests:** corner characters match mode; width/height match requested; focused title is
Strong; action notches render correctly for 1, 2, and 3 actions; filter mode emits
preamble + notch (no `ᐅ`); ascii mode swaps every glyph.

---

### 3.2 OverlayChrome

**Purpose:** Centered floating panel on a dimmed background. Consolidates the five
`renderWith*Overlay` functions in `internal/app/render.go`.

**Fields:**

```go
type OverlayChrome struct {
    Width   int
    Height  int
    Title   string
    Actions []Action   // type Action = layout.Action
    Theme   theme.Theme
}
```

**Rendering (unicode):**

```
╭─ Theme ─────────────────────────────────────╮
│  (overlay content)                          │
╰─────────────────────────────────────────────╯
```

**Rendering (ascii):**

```
+- Theme -----------------------------------------------+
|  (overlay content)                                    |
+-------------------------------------------------------+
```

**Roles:**

| Field | Role |
|---|---|
| Border | Accent |
| Title | Strong |
| Action.Key | Accent |
| Action.Label | Muted |

**Glyphs:** same structural set as PaneChrome; no toggle-key superscript.

**Lifecycle:** stateless.

**Tests:** border colour is Accent; title is Strong; action notches render correctly;
ascii mode; width/height match.

---

### 3.3 Panel

**Purpose:** Full-screen centered framed panel for transitional screens (onboarding,
auth, splash, too-small). The panel title IS the step header — no separate heading
element needed.

**Fields:**

```go
type Panel struct {
    Width   int
    Height  int
    Title   string
    Intent  PanelIntent   // PanelIntentDefault | PanelIntentError
    Theme   theme.Theme
}
```

**Rendering (unicode, default intent):**

```
╭─ Spotnik Setup ────────────────────────────────╮
│  (panel content)                               │
╰────────────────────────────────────────────────╯
```

**Rendering (unicode, error intent — Error role border):**

```
╭─ Authorization failed ─────────────────────────╮
│  (panel content)                               │
╰────────────────────────────────────────────────╯
```

**Rendering (ascii):**

```
+- Spotnik Setup -------------------------------------------+
|  (panel content)                                          |
+-----------------------------------------------------------+
```

**Roles:**

| Field | Role |
|---|---|
| Border (default) | Accent |
| Border (error) | Error |
| Title | Strong |

**Glyphs:** corners, horizontal / vertical rules — same set as PaneChrome.

**Lifecycle:** stateless.

**Tests:** default border is Accent; error border is Error; title is Strong; ascii mode.

---

### 3.4 TableChrome

**Purpose:** Standardises column construction — column tokens, header colour, playing-indicator
colour — so panes no longer build `TableConfig` literals inline.

**Note:** Implemented in `internal/ui/components/table_chrome.go` (alongside the `Table`
primitive it wraps), not in `internal/uikit/`. Call sites continue to call `NewTable`
directly; `TableChrome` is the canonical wrapping pattern for future migrations.

**Fields:**

```go
type TableChrome struct {
    Columns []ColumnDef    // column layout and per-column colour tokens
    Theme   theme.Theme
    // inner *Table — constructed on first Inner() call
}
```

**Key method:** `Inner() *Table` — returns the wrapped `*Table`, constructing it on first
call. The inner table owns all interactive state (scroll position, selection, etc.);
`TableChrome` is stateless from the caller's perspective.

**Rendering (unicode, header + rows):**

```
 #   Track                    Artist              Duration
 1   Lil Boo Thang            Paul Russell        3:12
▶2   Street Fighter           Kamasi Washington   5:44   ← playing row
```

**Rendering (ascii):** same layout; no braille or Unicode column-separator glyphs.

**Roles:**

| Field | Role |
|---|---|
| Header | `theme.TableHeader()` |
| Cell (index column) | Column-Index |
| Cell (primary column) | Column-Primary |
| Cell (secondary column) | Column-Secondary |
| Cell (tertiary column) | Column-Tertiary |
| Cell (selected row) | Selection |
| Playing indicator (`▶`) | `theme.PlayingIndicator()` |

**Glyphs:** `▶` / `>` playing indicator.

**Lifecycle:** owns-state via inner `*Table`.

**Tests:** header renders in TableHeader colour; selected row uses Selection role; playing
row shows `▶`; ascii mode.

---

### 3.5 ListRow

**Purpose:** Single-line row with optional glyph, label, and optional caption. Used in
theme overlay, profile overlay, and playlist read-only rows.

**Fields:**

```go
type ListRow struct {
    Glyph          GlyphRole             // empty = no glyph
    Label          string
    Caption        string                // "" = no caption
    Intent         Role
    Theme          theme.Theme
    RowBackground  lipgloss.TerminalColor // nil = default; set for cursor-highlight continuity
}
// Render(width int) string — width passed at call time, not stored
```

**Rendering (unicode, focused):**

```
◉  iPhone 14                [active]
```

**Rendering (ascii):**

```
(*)  iPhone 14              [active]
```

**Roles:**

| Field | Role |
|---|---|
| Glyph | matches row intent |
| Label | Plain |
| Caption | Muted |

**Glyphs:** intent-matched glyph from §4 (e.g., `◉` / `(*)` for active, `○` / `(o)` for
inactive).

**Lifecycle:** stateless.

**Tests:** glyph matches intent; label is Plain; caption is Muted; RowBackground is
propagated to all segments (no visible gaps); ascii mode.

---

### 3.6 LockedRow

**Purpose:** Disabled / inaccessible row variant — dim with leading `◌` glyph. Used for
Spotify-owned playlists and read-only items.

**Fields:**

```go
type LockedRow struct {
    Label  string
    Theme  theme.Theme
}
```

Width is not stored on the struct; `Render(width int)` and `PlainText(width int)` accept it
at call time, matching the `ListRow` pattern.

**Rendering (unicode):**

```
◌  Spotify Originals
```

**Rendering (ascii):**

```
(r)  Spotify Originals
```

**Roles:**

| Field | Role |
|---|---|
| Glyph (`◌`) | Muted |
| Label (entire row) | Muted |

**Glyphs:** `◌` / `(r)` locked/readonly.

**Lifecycle:** stateless.

**Tests:** glyph is Muted; label is Muted; ascii mode uses `(r)`.

---

### 3.7 SectionLabel

**Purpose:** Caps label marking a sub-section inside a pane, underlined by a `─` rule.
Used for Page B labels (GATEWAY, APP, SPOTIFY, POLLING, STORE).

**Fields:**

```go
type SectionLabel struct {
    Label       string
    Width       int
    AccentColor lipgloss.Color
    Theme       theme.Theme
}
```

**Rendering (unicode):**

```
Section
───────
```

**Rendering (ascii):**

```
Section
-------
```

**Roles:**

| Field | Role |
|---|---|
| Label | Parent pane's border token (via AccentColor) |
| Rule | Parent pane's border token (via AccentColor) |

**Glyphs:** horizontal rule `─` / `-`.

**Lifecycle:** stateless. Two lines tall.

**Tests:** label text is present; rule is full Width; ascii mode; AccentColor applies to label.

---

### 3.8 EmptyState

**Purpose:** Centred "no data" message with optional hint. Replaces hand-rolled empty
messages in queue, search results, and playlist loading.

**Fields:**

```go
type EmptyState struct {
    Text   string
    Hint   string   // "" = no hint
    Width  int
    Height int
    Theme  theme.Theme
}
```

**Rendering (unicode, with hint):**

```
(centred vertically and horizontally within Width × Height)

   Nothing in queue

   Press / to search
```

**Rendering (ascii):** identical layout; no Unicode-specific characters.

**Roles:**

| Field | Role |
|---|---|
| Text | Muted |
| Hint | Muted |

**Glyphs:** none.

**Lifecycle:** stateless. Output is exactly `Height` lines.

**Tests:** text is centred; hint is centred; output is exactly Height lines; ascii mode.

---

### 3.9 URLBox

**Purpose:** Muted-border rectangle wrapping accent-coloured URL or code content.
Wraps long URLs at `&` boundaries (matches existing `wrapURL` helper behaviour).

**Fields:**

```go
type URLBox struct {
    URL   string
    Width int
    Theme theme.Theme
}
```

**Rendering (unicode):**

```
╭──────────────────────────────────────────────────────╮
│  http://localhost:8888/callback?code=abc              │
│  &state=xyz&scope=user-read-playback-state            │
╰──────────────────────────────────────────────────────╯
```

**Rendering (ascii):**

```
+------------------------------------------------------+
|  http://localhost:8888/callback?code=abc              |
|  &state=xyz&scope=user-read-playback-state            |
+------------------------------------------------------+
```

**Roles:**

| Field | Role |
|---|---|
| Border | Muted |
| URL content | Accent |

**Glyphs:** corners, horizontal / vertical rules.

**Lifecycle:** stateless.

**Tests:** border is Muted; URL text is Accent; long URL wraps at `&`; ascii mode.

---

### 3.10 HeaderBar

**Purpose:** Top app bar: app name · page indicator · preset info · right-side chips.
Extracts `renderHeader` from `internal/app/render.go`. Shares `StatusBarBg()` with
`StatusBar` to visually bracket the grid.

**Fields:**

```go
type HeaderBar struct {
    Width      int
    AppName    string
    Page       string     // "A" or "B"
    Preset     int        // -1 hides preset segment (Page B); >= 0 shows preset N
    RightChips []string   // pre-rendered chip strings from Chip.Render()
    Theme      theme.Theme
}
```

**Rendering (unicode):**

```
 spotnik ─ Page A ─ preset 0 ──────────────────── ◉ iPhone
```

**Rendering (ascii):**

```
 spotnik - Page A - preset 0 -------------------------------- (*) iPhone
```

**Roles:**

| Field | Role |
|---|---|
| Background | `theme.StatusBarBg()` |
| AppName | Strong |
| Separator ` ─ ` | Muted |
| Page key (A/B) | Accent |
| PresetLabel | Muted |
| Right chips | Chip role (see §3.13) |

**Glyphs:** horizontal rule `─` / `-`; chip glyphs from §3.13.

**Lifecycle:** stateless. Single line tall.

**Tests:** AppName is Strong; separator is Muted; page key is Accent; width matches;
ascii mode.

---

### 3.11 StatusBar

**Purpose:** Bottom global key bar, three lines tall. Composition over `KeyBar` using
`bubbles/help` `ShortHelp()` / `FullHelp()`. Body background is terminal-default (no
`StatusBarBg()` applied); a muted-accent border distinguishes the bar from the grid.

**Fields:**

```go
type StatusBar struct {
    Width    int
    Bindings help.KeyMap
    Theme    theme.Theme
}
```

**Rendering (unicode):**

```
 /search   0 page   p preset   1-8 toggle   Tab pane   d devices   ? help   q quit
```

**Rendering (ascii):** same; no unicode-specific characters in bindings.

**Roles:**

| Field | Role |
|---|---|
| Body background | terminal default (no fill) |
| Key label | `theme.KeyHint()` |
| Description | Muted |

**Glyphs:** none beyond what `KeyBar` contributes.

**Lifecycle:** stateless.

**Tests:** key labels use KeyHint colour; descriptions use Muted; width matches; ascii mode.

---

### 3.12 KeyBar

**Purpose:** Reusable `key:desc · key:desc` strip. The underlying primitive for the
StatusBar body, overlay footers, and inline hints. Stateless.

**Fields:**

```go
type KeyBar struct {
    Bindings  []key.Binding
    Theme     theme.Theme
}
```

**Rendering (unicode):**

```
f filter · n new · Esc close
```

**Rendering (ascii):** identical; separator becomes ` | ` in ascii mode.

**Roles:**

| Field | Role |
|---|---|
| Key | `theme.KeyHint()` |
| Description | Muted |
| Separator ` · ` (unicode) / ` | ` (ascii) | Muted |

**Glyphs:** `·` (unicode) / `|` (ascii) separator.

**Lifecycle:** stateless.

**Tests:** key is KeyHint colour; separator is Muted; multiple bindings render with
separators; ascii mode.

---

### 3.13 Chip

**Purpose:** Inline pill with leading glyph + label on `StatusBarBg`. Used in the header
for device and profile chips.

**Fields:**

```go
type Chip struct {
    Glyph  GlyphRole
    Label  string
    Intent Role
    Theme  theme.Theme
}
```

**Rendering (unicode, active device):**

```
◉ iPhone 14
```

**Rendering (ascii):**

```
(*) iPhone 14
```

**Roles:**

| Field | Role |
|---|---|
| Glyph | intent role |
| Label | `theme.HeaderChipFg()` |
| Background | `theme.StatusBarBg()` |

**Glyphs:** intent-matched glyph (e.g., `◉` / `(*)` for active, `○` / `(o)` for inactive).

**Lifecycle:** stateless.

**Tests:** glyph matches intent; label uses HeaderChipFg; ascii mode.

---

### 3.14 FormField

**Purpose:** Labelled input with intrinsic validation and an error slot beneath.
Wraps `bubbles/textinput`. Used for the onboarding Client-ID input.

**Fields / constructors:**

```go
type FormFieldConfig struct {
    Label       string
    Placeholder string
    Validate    func(string) error
    Theme       theme.Theme
}

func NewFormField(cfg FormFieldConfig) *FormField
```

**Key methods:** `Focus()`, `Blur()`, `Update(tea.Msg) (*FormField, tea.Cmd)`,
`Render() string`, `Value() string`, `SetValue(string)`, `Validate() error`,
`ValidationError() string`, `InputTextStyle() lipgloss.Style`,
`InputCursorStyle() lipgloss.Style`.

**Rendering (unicode, validation error):**

```
Client ID
┌─────────────────────────────────────────────────────┐
│ abc123_invalid█                                     │
└─────────────────────────────────────────────────────┘
✗ Must be 32 characters
```

**Rendering (ascii):**

```
Client ID
+---------------------------------------------------------+
| abc123_invalid█                                         |
+---------------------------------------------------------+
x Must be 32 characters
```

**Roles:**

| Field | Role |
|---|---|
| Label | Muted |
| Input text | Plain |
| Cursor | Accent |
| ValidationError glyph | Error |
| ValidationError text | Plain |

**Glyphs:** `✗` / `x` failure (no success glyph rendered in current implementation).

**Lifecycle:** owns-state (wraps `textinput.Model`).

**Tests:** label is Muted; valid input shows no error slot; invalid input shows
`✗` + message; `Value()` returns current text; ascii mode.

---

### 3.15 Toast

**Purpose:** Typed notification surfaced via `bubbleup`. Replaces raw-string call sites:
`a.alerts.NewAlertCmd("error", msg)` becomes `uikit.Toast{Intent: ToastError, …}.Cmd(a.alerts)`.

**Fields:**

```go
type Toast struct {
    Intent ToastIntent     // ToastSuccess | ToastError | ToastWarning | ToastInfo | ToastRateLimit
    Title  string          // required, ≤ 48 runes, sentence case, no trailing "."
    Body   string          // optional, ≤ 160 runes, 1 sentence, trailing "."
    TTL    time.Duration   // 0 = default per intent
}
```

**Default TTLs:**

| Intent | TTL |
|---|---|
| Success | 4s |
| Info | 4s |
| Warning | 5s |
| Error | 6s |
| RateLimit | Retry-After seconds |

**Rendering (unicode, Error intent):**

```
✗ Spotify unreachable
  Retrying in 3s.
```

Note: bubbleup renders the combined `Title + "\n" + Body` string in a single intent foreground
colour. The bordered box style above is for illustration only; the current implementation does
not draw a per-toast border (future revision may add per-line styling).

**Rendering (ascii):** same layout; `✗` → `x`.

**Roles:**

| Field | Role |
|---|---|
| Glyph | intent role |
| Title + Body | intent role |

Note: Internal Title/Body split is for content-rule purposes (sentence case, length limits).
bubbleup renders the combined string in a single foreground colour — there is no Strong/Plain
split at render time.

**Glyphs by intent:** `✓`/`+` Success · `✗`/`x` Error · `◬`/`!` Warning · `→`/`>` Info ·
`⧖`/`~` RateLimit.

**Positioning:**

| View mode | Position |
|---|---|
| Grid view | viewport bottom-right |
| Panel view | panel bottom-right |
| Overlay view | viewport bottom-right |

**Content rules:** Title: past-participle verb phrase for completions, noun + state for
async events. Body: single sentence, trailing `.`, optional for Success/Info, required for
Error. Sentence case, no emoji.

**Lifecycle:** owns-state (bubbleup animates lifetime).

**Tests:** intent colour matches theme token; default TTL per intent; Title and Body
truncate at documented limits; ascii mode swaps glyph; RateLimit TTL honours Retry-After.

---

### 3.16 StatusGlyph

**Purpose:** Atomic glyph with intent colour and adjacent text. Replaces scattered
ad-hoc `✓` / `✗` / `◉` usages for persistent informational state (not async events —
use `Toast` for those).

**Fields:**

```go
type StatusGlyph struct {
    Role  Role
    Text  string
    Theme theme.Theme
    Gap   int   // extra spaces beyond the mandatory single separator; default 0 = 1 space, 1 = 2 spaces
}
```

**Rendering (unicode, Success):**

```
✓ Premium
```

**Rendering (ascii):**

```
+ Premium
```

**Roles:**

| Field | Role |
|---|---|
| Glyph | intent role |
| Text | intent role |

**Glyphs:** intent-matched (see §4.2).

**Lifecycle:** stateless.

**Tests:** glyph is intent colour; text is intent role; Gap inserts correct spaces; ascii mode.

---

### 3.17 ProgressBar

**Purpose:** Fill bar with unicode partial-block glyphs and ascii fallback. Used for the
seek bar and volume bar in the NowPlaying pane.

**Fields:**

```go
type ProgressBar struct {
    Width    int
    Progress float64   // 0.0 – 1.0
    Theme    theme.Theme
}
```

**Helper:** `PartialGlyph(remainder float64, m GlyphMode) string` — returns the
partial-block character for the fractional remainder (1/8 resolution).

**Rendering (unicode, 60% filled):**

```
████████████░░░░░░░░░
```

**Rendering (ascii, 60% filled):**

```
========-----.......
```

**Ascii fill characters:** `#` full · `=` 5/8–7/8 · `-` 3/8–1/2 · `.` 1/8–1/4 and empty.

**Roles:**

| Field | Role |
|---|---|
| Fill (`█` and partials) | `theme.Gradient1()` |
| Empty (`░`) | Muted |

**Glyphs:** `█ ▉ ▊ ▋ ▌ ▍ ▎ ▏ ░` / `# = - .` (see §4.7).

**Note:** Callers wanting per-position gradient compose via `uikit.PartialGlyph` +
`GlyphFor` directly (see `internal/ui/components/gradient.go`).

**Lifecycle:** stateless.

**Tests:** fill width matches Progress × Width; partial glyph at boundary; empty region
is Muted; ascii mode uses `=`/`-`/`.`; `PartialGlyph` returns correct character at 0, 0.25,
0.5, 0.75, 1.0.

---

### 3.18 Spinner

**Purpose:** Animated wait indicator with `Done` / `Fail` / `Cancel` terminal states.
Wires the onboarding OAuth wait. TUI peer to `cliout.Spinner`.

**Constructor and key methods:**

```go
func NewSpinner(text string, th theme.Theme) *Spinner
func (s *Spinner) Init() tea.Cmd
func (s *Spinner) Update(msg tea.Msg) (*Spinner, tea.Cmd)
func (s *Spinner) Done(text string) (*Spinner, tea.Cmd)
func (s *Spinner) Fail(text string) (*Spinner, tea.Cmd)
func (s *Spinner) Cancel() (*Spinner, tea.Cmd)
func (s *Spinner) View() string
```

**Terminal-state messages:** `SpinnerDoneMsg{Text string}`, `SpinnerFailMsg{Err string}`,
`SpinnerCancelledMsg{}`.

**Terminal states:**

| Call | Behaviour |
|---|---|
| `Done(text)` | Frame becomes `✓` (Success); held ~1.2s; emits `SpinnerDoneMsg`; clears. |
| `Fail(text)` | Frame becomes `✗` (Error); held ~2s; emits `SpinnerFailMsg`; clears. |
| `Cancel()` | Clears immediately, no final line; emits `SpinnerCancelledMsg`. |

**Rendering (unicode, animated):**

```
⣾ Waiting for authorization
```

**Rendering (unicode, Done resolution):**

```
✓ Authorized
```

**Rendering (unicode, Fail resolution):**

```
✗ Authorization failed
```

**Rendering (ascii):**

```
| Waiting for authorization   (frames rotate | / - \)
+ Authorized
x Authorization failed
```

**Roles:**

| Field | Role |
|---|---|
| Frame (animating) | Accent |
| Frame (Done) | Success |
| Frame (Fail) | Error |
| Text | Muted |

**Glyphs:** braille frames `⣾⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` / `|/-\`; `✓`/`+` done; `✗`/`x` fail.

**Lifecycle:** owns-state (frame index, resolution state).

**Tests:** frame advances on `SpinnerTickMsg`; `Done` replaces frame with `✓` and emits
`SpinnerDoneMsg` after TTL; `Fail` replaces frame with `✗` and emits `SpinnerFailMsg`
after TTL; `Cancel` clears without a final line; ascii mode uses rotating `|/-\` frames.

---

### 3.19 PlaybackControls

**Purpose:** Stateless transport-strip primitive. Owns the four transport icon
positions (shuffle / play-pause / queue / repeat) and resolves every glyph through
`GlyphFor` so both unicode and ASCII modes are handled automatically. The
`components.Controls` compatibility wrapper delegates entirely to this struct.

**Struct and constructor:**

```go
type RepeatMode int

const (
    RepeatOff RepeatMode = iota // ⟳ / ro — inactive colour
    RepeatAll                   // ↻ / rp — active colour
    RepeatOne                   // ↻¹ / rp1 — active colour
)

type PlaybackControls struct {
    Playing    bool
    Shuffle    bool
    RepeatMode RepeatMode
    Theme      theme.Theme
}

func (c PlaybackControls) Render() string
```

**Rendering (unicode, playing, shuffle off, repeat off):**

```
⇄  ⏸  ≡  ⟳
```

**Rendering (unicode, paused, shuffle on, repeat all):**

```
⇄  ▷  ≡  ↻
```

**Rendering (ascii, playing, shuffle off, repeat off):**

```
sh  ||  Q  ro
```

**Roles:**

| Position | Condition | Role |
|---|---|---|
| Shuffle | shuffle on | `PlayingIndicator` |
| Shuffle | shuffle off | `TextSecondary` |
| Play/Pause | playing | `PlayingIndicator` (shows ⏸/`||`) |
| Play/Pause | paused | `TextSecondary` (shows ▷/`|>`) |
| Queue | always | `TextSecondary` |
| Repeat | RepeatAll / RepeatOne | `PlayingIndicator` |
| Repeat | RepeatOff | `TextSecondary` |

**Glyphs:**

| Position | Unicode | ASCII | GlyphRole |
|---|---|---|---|
| Shuffle | `⇄` | `sh` | `GlyphShuffle` |
| Play (paused state) | `▷` | `\|>` | `GlyphPausedPB` |
| Pause (playing state) | `⏸` | `\|\|` | `GlyphPaused` |
| Queue | `≡` | `Q` | `GlyphQueue` |
| Repeat Off | `⟳` | `ro` | `GlyphRepeatOff` |
| Repeat All | `↻` | `rp` | `GlyphRepeatAll` |
| Repeat One | `↻¹` | `rp1` | `GlyphRepeatOne` |

**Visual note:** `GlyphRepeatOff` (`⟳`) is distinct from `GlyphRepeatAll` (`↻`).
The legacy `components.Controls` rendered `↻` for both off and all states (with
different colours). The primitive uses catalogue-intent glyphs exclusively.

**Lifecycle:** stateless value — call `Render()` directly from `View()`.

**Tests:** `TestPlaybackControls_RenderUnicode_Playing` — unicode output contains
`⏸`, `≡`, `⇄`, `⟳` (off-state); `TestPlaybackControls_RenderASCII_Playing` —
ascii output contains `||`, `Q`, `sh`, `ro`, no unicode glyphs;
`TestPlaybackControls_RepeatModes` — all three `RepeatMode` values render correct
glyph in both modes.

---

## 4. Glyph Catalogue

Every glyph the TUI and CLI use. Every row has a unicode form and an ascii fallback.
New glyphs require a PR that updates this table and `docs/TUI-DESIGN-SYSTEM.md` in the
same commit. Removed glyphs are flagged "banned".

### 4.1 Structural / borders

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

### 4.2 Intent / feedback

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

**Banned:** `⚠` (variation-selector sensitive, renders as emoji on many terminals);
`✅` `❌` `❗` (emoji).

### 4.3 State / availability

| Role | Unicode | ASCII | Where used |
|---|---|---|---|
| active / on | `◉` | `(*)` | Device chip active, playing indicator |
| inactive | `◎` | `( )` | Pending, dim state |
| available / free-tier | `○` | `(o)` | Profile free-tier, empty slot |
| filled dot | `●` | `(#)` | Count indicator, progress step done |
| empty square | `□` | `[ ]` | Checkbox off (future) |
| filled square | `■` | `[x]` | Checkbox on (future) |
| locked / readonly | `◌` | `(r)` | Inaccessible playlist row (Spotify-owned), read-only items |
| pinned / starred | `★` | `*` | Starred item, pinned playlist (future) |
| unpinned | `☆` | `-` | Optional counterpart |
| bullet | `•` | `*` | Prose lists |

### 4.4 Navigation / scroll

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

### 4.5 Playback controls

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

### 4.6 Domain / music / identity

| Role | Unicode | ASCII |
|---|---|---|
| music note | `♪` | `*` |
| double note | `♫` | `**` |
| premium badge | `♛` | `*P` |
| free-tier badge | `○` | `(o)` |
| cloud / remote device | `☁` | `(c)` |
| playlist badge | `▤` | `[=]` |

### 4.6a Generic separators

| Role | Unicode | ASCII |
|---|---|---|
| separator (bullet) | `·` | `\|` |

### 4.6b Device-type icons (devices pane)

| Role | Unicode | ASCII |
|---|---|---|
| computer | `⊡` | `[c]` |
| phone | `⊞` | `[p]` |
| speaker | `⊟` | `[s]` |
| TV | `⊠` | `[tv]` |

### 4.7 Graphical fills (ProgressBar, Visualizer)

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

### 4.8 Spinner frames

Exported via `uikit.SpinnerFrames(mode GlyphMode) []string`. Both `uikit.Spinner`
and `cliout.Spinner` source frames from this function — no inline arrays.

| Set | Unicode | ASCII |
|---|---|---|
| braille (10 frames) | `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` | `\|/-` (4 frames) |

### 4.9 Keyboard chords

Keyboard-chord glyphs are **text-first**. Only arrow keys, Enter, and Esc may use glyph
form; modifier keys (Ctrl, Alt, Shift, Cmd) always render as text for cross-platform
readability.

| Role | GlyphRole constant | Unicode | ASCII |
|---|---|---|---|
| enter | `GlyphEnter` | `⏎` | `Enter` |
| escape | `GlyphEscape` | `⎋` | `Esc` |
| tab | `GlyphTab` | `⇥` | `Tab` |
| backspace | `GlyphBackspace` | `⌫` | `BS` |
| space | `GlyphSpace` | `␣` | `Space` |
| shift | — | `Shift` (text only) | `Shift` |
| ctrl / alt / cmd | — | `Ctrl` / `Alt` / `Cmd` (text only) | `Ctrl` / `Alt` / `Cmd` |

### 4.10 Superscripts

Used in pane titles (toggle-key number) and repeat-one indicator.

| Role | GlyphRole constant | Unicode | ASCII |
|---|---|---|---|
| 0 | `GlyphSuperscript0` | `⁰` | `0` |
| 1 | `GlyphSuperscript1` | `¹` | `1` |
| 2 | `GlyphSuperscript2` | `²` | `2` |
| 3 | `GlyphSuperscript3` | `³` | `3` |
| 4 | `GlyphSuperscript4` | `⁴` | `4` |
| 5 | `GlyphSuperscript5` | `⁵` | `5` |
| 6 | `GlyphSuperscript6` | `⁶` | `6` |
| 7 | `GlyphSuperscript7` | `⁷` | `7` |
| 8 | `GlyphSuperscript8` | `⁸` | `8` |
| 9 | `GlyphSuperscript9` | `⁹` | `9` |
| + | `GlyphSuperscriptPlus` | `⁺` | `+` |
| − | `GlyphSuperscriptMinus` | `⁻` | `-` |

### 4.11 Glyph mode detection

Resolution order on first `uikit.Render` call (lazy, `sync.Once`):

1. `ui.glyphs` config (`"auto"`, `"unicode"`, `"ascii"`). Default `"auto"`.
2. If `"auto"`: `LC_ALL` or `LANG` contains `UTF-8` or `utf8` → unicode; else → ascii.
3. `NO_COLOR` is orthogonal — it strips colour, not glyphs.

---

## 5. Role / Colour Matrix

### 5.1 Roles

| Role | Default token | Intent |
|---|---|---|
| **Accent** | `theme.Accent()` | Interactive / call-to-action — keys, URLs, filter query, focus cues |
| **Strong** | `theme.TextPrimary()` + bold | Primary headlines — pane title, panel title, section caps |
| **Plain** | `theme.TextPrimary()` | Body content — track name, value, description |
| **Muted** | `theme.TextMuted()` | Labels, captions, secondary prose, action-key descriptions |
| **Success** | `theme.Success()` | Success toasts, premium badge, playing indicator |
| **Error** | `theme.Error()` | Error toasts, validation fail |
| **Warning** | `theme.Warning()` | Warning toasts, Premium-required line |
| **Info** | `theme.Info()` | Info toasts, hint arrows |
| **Selection** | `theme.SelectedFg()` | Focused row in list/table |
| **Column-Index** | `theme.ColumnIndex()` | Table index column |
| **Column-Primary** | `theme.ColumnPrimary()` | Table primary data column |
| **Column-Secondary** | `theme.ColumnSecondary()` | Table supporting context column |
| **Column-Tertiary** | `theme.ColumnTertiary()` | Table tertiary / metadata column |
| **PaneBorder-<ID>** | `theme.PaneBorderX()` per pane | Pane-chrome border, dims automatically when unfocused |

Call sites set a **role**, never a raw colour.

### 5.2 Field-role mapping

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
| `StatusBar.Bg` | terminal default (no fill) |
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
| `Toast.Title` | intent role |
| `Toast.Body` | intent role |
| `StatusGlyph` | intent role |
| `ProgressBar.Fill` | `theme.Gradient1()` |
| `ProgressBar.Empty` | Muted |
| `Spinner.Frame` | Accent |
| `Spinner.Text` | Muted |

### 5.3 Rules enforced by the matrix

- Only **Accent** signals "you can press this" — keys, URLs, interactive cues.
  Informational values are Plain.
- **Strong** is bold, not bright — contrast through weight.
- One Accent per call-to-action — an action key OR an action URL, never both wrapped
  into the same span.
- `HeaderBar` uses `StatusBarBg()` to bracket the grid; `StatusBar` uses a muted-accent border on a terminal-default body.

---

## 6. Feedback Channels

Six non-overlapping surfaces carry feedback. Each has a single reason to exist; use
the right one.

| Surface | Use when | Primitive |
|---|---|---|
| **Toast** | Async operation completed or failed (auto-dismisses) | `uikit.Toast` |
| **StatusGlyph** | Persistent informational state visible at a glance | `uikit.StatusGlyph` |
| **EmptyState** | No data in a pane — guide the user to the next action | `uikit.EmptyState` |
| **KeyBar** | Discoverable key hints embedded in a pane footer or overlay | `uikit.KeyBar` |
| **StatusBar** | Global key hints always visible at the bottom of the screen | `uikit.StatusBar` |
| **PaneChrome filter preamble** | Current filter query displayed inside the pane border | `uikit.PaneChrome` (FilterQuery field) |

**Rules:**

- `Toast` is for completion acknowledgements and async events — not for persistent state.
- `StatusGlyph` is for persistent informational state — not for completion events.
- `EmptyState` covers the entire pane content area — never a partial overlay.
- `KeyBar` and `StatusBar` both use `theme.KeyHint()` for key labels — visually consistent.
- The PaneChrome filter preamble replaces action notches when `FilterQuery != ""` — the
  two modes never coexist in the same border.

---

## 7. Relationship to Other Docs

| Document | Authority over |
|---|---|
| `docs/TUI-DESIGN-SYSTEM.md` (this file) | Primitive contracts, glyph catalogue, role matrix, feedback surfaces |
| `docs/DESIGN.md` | Layout mechanics: grid, pages, presets, pane toggling, focus rotation, min-terminal-size rule, keybindings |
| `docs/CLI-OUTPUT.md` | CLI message types, glyphs, palette, interactive prompts (`internal/cliout`) |

Where both `DESIGN.md` and `TUI-DESIGN-SYSTEM.md` apply — for example, pane borders —
`DESIGN.md` describes the pane identity (colour token, toggle key, pane ID); this document
describes the exact rendering contract (field roles, glyph choices, notch format).

The glyph catalogue (§4) and emphasis-role vocabulary (§5) are shared between
`internal/cliout` and `internal/uikit`. Changes to either propagate to both packages in
the same PR.
