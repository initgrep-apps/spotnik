# DESIGN.md — UI/UX Specification (FROZEN)

> **This document is the design constitution for Spotnik.**
> No UI change, layout modification, color change, or keybinding shift is permitted
> without explicitly updating this document first and getting owner sign-off.
> Agents: treat every pixel of this spec as a hard constraint, not a suggestion.

---

## Design Philosophy

1. **Clarity over cleverness** — the interface communicates state instantly, no hunting
2. **Keyboard first, always** — every action reachable without a mouse
3. **Respect the terminal** — no colors that break in 256-color mode, no unicode that renders as boxes
4. **Motion has meaning** — animations exist only when they communicate state change
5. **Developer aesthetic** — feels like a tool, not a toy; dense but not cluttered

---

## Layout — The Three-Pane Model

This layout is **frozen**. The three-pane model is the identity of the app.

Alternative views (Stats via `2`, Playlist Manager via `3`) may temporarily replace the
three-pane layout. Pressing `1` always returns to it. The freeze means the three-pane layout
itself is never modified, not that it must be the only view.

```
╭──────────────────────────────────────────────────────────────────────────────╮
│  Spotnik                                              ◉ MacBook Pro Speakers │
├─────────────────────┬───────────────────────────────┬────────────────────────┤
│                     │                               │                        │
│  LIBRARY            │  NOW PLAYING                  │  QUEUE                 │
│  ─────────────────  │  ───────────────────────────  │  ────────────────────  │
│                     │                               │                        │
│  ▶ Playlists   (12) │  Blinding Lights              │  ▶ 1  Save Your Tears  │
│    Albums       (8) │  The Weeknd                   │     2  Starboy         │
│    Artists     (34) │  After Hours                  │     3  Can't Feel...   │
│    Liked Songs (287)│                               │     4  In Your Eyes    │
│    Podcasts     (3) │  ████████████████░░░░░░░░░░   │     5  Repeat After Me │
│                     │  2:34 ──────────────── 4:12   │                        │
│  ─────────────────  │                               │  ────────────────────  │
│  RECENTLY PLAYED    │  ⏮   ⏸   ⏭      🔀   🔁       │  5 tracks remaining    │
│  ─────────────────  │  ───────────────────────────  │                        │
│  Blinding Lights    │  VOL  ████████░░░░░░  65%     │                        │
│  Save Your Tears    │                               │                        │
│  Starboy            │                               │                        │
│  Levitating         │                               │                        │
│  Peaches            │                               │                        │
│                     │                               │                        │
├─────────────────────┴───────────────────────────────┴────────────────────────┤
│  /search   j/k move   Space play   Tab pane   d devices   ? help   q quit    │
╰──────────────────────────────────────────────────────────────────────────────╯
```

### Pane Proportions

| Pane | Width | Purpose |
|---|---|---|
| Library (left) | 22% | Navigation: playlists, albums, recently played |
| Player (center) | 50% | Now playing: track info, seek bar, controls |
| Queue (right) | 28% | What's coming up next |

Proportions are responsive — they scale to terminal width but maintain the same ratios. Minimum terminal width: **100 columns**. Minimum height: **24 rows**.

Below minimum size, show a message: `Terminal too small. Please resize to at least 100×24.`

---

## Layout — Search Overlay

Search opens as a **floating modal overlay** centered on screen. It does not replace any pane.

```
╭──────────────────────────────────────────────────────────────────────────────╮
│  Spotnik                                              ◉ MacBook Pro Speakers   │
├─────────────────────┬───────────────────────────────┬────────────────────────┤
│  (dimmed)           │       ╭─────────────────────────────────╮ (dimmed)    │
│                     │       │  🔍 Search                       │             │
│                     │       │  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄ ┄┄┄  ┄  │             │
│                     │       │  > blinding lig█                │             │
│                     │       │  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄  ┄┄┄  │             │
│                     │       │  ● TRACKS                       │             │
│                     │       │  ▶ Blinding Lights  · The Weeknd│             │
│                     │       │    Blinding Lights  · Sunday Se │             │
│                     │       │    Blinding Light   · Maroon 5  │             │
│                     │       │  ● ARTISTS                      │             │
│                     │       │    The Weeknd                   │             │
│                     │       │  ● PLAYLISTS                    │             │
│                     │       │    Blinding Pop Hits            │             │
│                     │       ╰─────────────────────────────────╯             │
│                     │                               │                        │
├─────────────────────┴───────────────────────────────┴────────────────────────┤
│  Esc close   j/k move   Enter play   Tab next section   a add to queue       │
╰──────────────────────────────────────────────────────────────────────────────╯
```

---

## Layout — Device Switcher Overlay

```
╭──────────────────────────────────────────────────────────────────────────────╮
│  ...                                                                         │
│                     ╭──────────────────────────────╮                         │
│                     │  DEVICES                     │                         │
│                     │  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄   │                         │
│                     │  ◉ MacBook Pro   [active]    │                         │
│                     │  ○ iPhone 14                 │                         │
│                     │  ○ Kitchen Speaker           │                         │
│                     │  ○ Living Room TV            │                         │
│                     ╰──────────────────────────────╯                         │
│                     ...                                                      │
╰──────────────────────────────────────────────────────────────────────────────╯
```

---

## Layout — Stats View

Stats replaces the three-pane layout. Accessible via `2` key.

```
╭──────────────────────────────────────────────────────────────────────────────╮
│  Spotnik  [STATS]                                     ◉ MacBook Pro Speakers   │
├─────────────────────────────────┬────────────────────────────────────────────┤
│  TOP TRACKS                     │  TOP ARTISTS                               │
│  ─────────────────────────────  │  ──────────────────────────────────────── │
│  Time range: [4wk] [6mo] [all]  │  Time range: [4wk] [6mo] [all]            │
│                                 │                                            │
│  1  Blinding Lights    The Week │  1  The Weeknd                             │
│  2  Levitating         Dua Lipa │  2  Dua Lipa                               │
│  3  Save Your Tears    The Week │  3  Post Malone                            │
│  4  Peaches            Justin B │  4  Justin Bieber                          │
│  5  Mood               24kGoldn │  5  Taylor Swift                           │
│  ...                            │  ...                                       │
│  25 items                       │  25 items                                  │
├─────────────────────────────────┴────────────────────────────────────────────┤
│  RECENTLY PLAYED                                                              │
│  ─────────────────────────────────────────────────────────────────────────── │
│  Blinding Lights  ·  The Weeknd  ·  After Hours          3 min ago            │
│  Levitating       ·  Dua Lipa    ·  Future Nostalgia     18 min ago           │
│  Starboy          ·  The Weeknd  ·  Starboy              34 min ago           │
├───────────────────────────────────────────────────────────────────────────────┤
│  1 library   j/k move   Enter play   [4wk]/[6mo]/[all] toggle range          │
╰──────────────────────────────────────────────────────────────────────────────╯
```

---

## Layout — Help Overlay

Pressing `?` shows a full-screen help overlay.

```
╭──────────────────────────────────────────────────────────────────────────────╮
│  KEYBOARD SHORTCUTS                                                    ? close│
├────────────────────────────┬─────────────────────────────────────────────────┤
│  PLAYBACK                  │  NAVIGATION                                     │
│  Space    Play / Pause     │  j / ↓     Move down                            │
│  n / →    Next track       │  k / ↑     Move up                              │
│  p / ←    Previous track   │  Tab       Next pane                            │
│  + / -    Volume up/down   │  Shift+Tab Previous pane                        │
│  s        Toggle shuffle   │  Enter     Select / Play                        │
│  r        Cycle repeat     │  Esc       Close overlay                        │
│                            │                                                  │
│  VIEWS                     │  LIBRARY                                        │
│  1        Library view     │  /         Search                               │
│  2        Stats view       │  a         Add to queue                         │
│  3        Playlist manager │  l         Like / unlike track                  │
│  d        Device switcher  │  PgUp/Dn   Scroll page                          │
│  ?        This help        │                                                  │
│  q        Quit             │                                                  │
╰────────────────────────────┴─────────────────────────────────────────────────╯
```

---

## Color System

All color values must come from `internal/ui/theme/`. **Never hardcode hex values in component code.**
Always reference a token from the active theme via the `Theme` interface.

---

### Default Theme — True Black

The default is **True Black** — pure `#000000` background. No warm tints, no purple haze.
This is the most honest terminal aesthetic: it blends into the terminal window itself,
works perfectly on OLED screens (true black pixels consume zero power), and makes no
visual opinion on the user before they've chosen one.

| Token | Hex | Usage |
|---|---|---|
| `Base` | `#000000` | App background — pure black |
| `Surface` | `#0f0f0f` | Panel lift — barely perceptible |
| `SurfaceAlt` | `#1a1a1a` | Overlay backgrounds (search, device modal) |
| `InactiveBorder` | `#1e1e1e` | Pane separators, inactive borders |
| `ActiveBorder` | `#00afff` | Focused pane border — bright ice blue |
| `TextPrimary` | `#f0f0f0` | Primary text — near white |
| `TextSecondary` | `#888888` | Secondary text — artist names, subtitles |
| `TextMuted` | `#444444` | Muted — timestamps, counts, hints |
| `SelectedBg` | `#1c3a5e` | Selected list item background — dark blue |
| `SelectedFg` | `#f0f0f0` | Selected list item text |
| `SectionHeader` | `#00afff` | Section labels (LIBRARY, QUEUE, etc.) — bold |
| `PlayingIndicator` | `#00ff88` | ▶ symbol, now playing — terminal green |
| `SeekBar` | `#00afff` | Seek bar fill — matches active border |
| `VolumeBar` | `#00afff` | Volume bar fill |
| `Success` | `#00ff88` | Success states — same as playing indicator |
| `Warning` | `#ffcc00` | Caution notices |
| `Error` | `#ff5555` | Error text — classic terminal red |
| `DeviceActive` | `#00e5cc` | ◉ active device indicator — teal |
| `StatusBarBg` | `#000000` | Status bar — same as base, no separation |
| `StatusBarFg` | `#444444` | Status bar text |
| `KeyHint` | `#00afff` | Keybinding labels (Space, Tab, etc.) |

**What this looks like in practice:**
- The terminal window and app background are indistinguishable — the UI appears to float on nothing
- Pane borders are barely visible when inactive, sharp blue when focused
- The only strong colors are: ice blue (focus/accent), terminal green (playing), and red (errors)
- Everything else is shades of grey — calm, focused, zero distraction

---

### Usage Rules (Theme-Agnostic)

These rules apply to **all themes**. They use semantic token names, not hex values.

- **Active pane border**: `ActiveBorder`
- **Inactive pane border**: `InactiveBorder`
- **Playing track `▶` symbol**: `PlayingIndicator`
- **Selected list item**: `SelectedBg` background + `SelectedFg` text
- **Section headers** (LIBRARY, QUEUE, NOW PLAYING): `SectionHeader`, bold
- **Seek bar fill**: `SeekBar`
- **Volume bar fill**: `VolumeBar`
- **Error messages**: `Error`
- **Status bar**: `StatusBarBg` background, `StatusBarFg` text, `KeyHint` for key labels
- **Overlays** (search, devices, help): `SurfaceAlt` background, `ActiveBorder` border

---

### Theme Interface

All themes must implement this interface exactly. No exceptions, no missing methods.
The interface is the contract between themes and components — components never
call a specific theme directly, only through this interface.

```go
// Theme defines all color tokens used by the UI.
// Implement this interface to create a new theme.
// All methods return a lipgloss.Color (hex string or ANSI code).
type Theme interface {
    // Backgrounds
    Base() lipgloss.Color        // App background
    Surface() lipgloss.Color     // Panel surface
    SurfaceAlt() lipgloss.Color  // Overlay backgrounds

    // Borders
    ActiveBorder() lipgloss.Color    // Focused pane
    InactiveBorder() lipgloss.Color  // Unfocused panes

    // Text
    TextPrimary() lipgloss.Color    // Main content text
    TextSecondary() lipgloss.Color  // Supporting text
    TextMuted() lipgloss.Color      // Timestamps, counts, hints

    // Selection
    SelectedBg() lipgloss.Color  // Selected list item background
    SelectedFg() lipgloss.Color  // Selected list item text

    // Semantic
    SectionHeader() lipgloss.Color    // LIBRARY, QUEUE labels
    PlayingIndicator() lipgloss.Color // ▶ symbol
    SeekBar() lipgloss.Color          // Seek bar fill
    VolumeBar() lipgloss.Color        // Volume bar fill
    Success() lipgloss.Color          // Success states
    Warning() lipgloss.Color          // Caution notices
    Error() lipgloss.Color            // Error messages
    DeviceActive() lipgloss.Color     // ◉ active device

    // Status bar
    StatusBarBg() lipgloss.Color  // Status bar background
    StatusBarFg() lipgloss.Color  // Status bar text
    KeyHint() lipgloss.Color      // Keybinding key labels

    // Metadata
    Name() string  // Theme display name e.g. "True Black"
    ID() string    // Theme config ID e.g. "black"
}
```

---

### Available Themes

| ID | Name | Default | Background | Character |
|---|---|---|---|---|
| `black` | True Black | ✅ **Yes** | `#000000` | Pure black, OLED-friendly, minimal |
| `monokai` | Monokai | — | `#272822` | Classic dark, warm orange/green accents |
| `catppuccin` | Catppuccin Mocha | — | `#1e1e2e` | Warm dark purple, developer favourite |
| `nord` | Nord | — | `#2e3440` | Cool arctic blues and greys |
| `light` | Light | — | `#eff1f5` | Catppuccin Latte — for the brave |

**Config:**
```toml
[ui]
theme = "black"       # default — change to any ID above
```

**Adding a new theme:** Implement the `Theme` interface, register it in
`internal/ui/theme/theme.go`, add it to the table above. Full token values for all
five themes are in `docs/features/01-theme-system.md`.

---

## Typography

- **Font**: monospace only (inherit from terminal)
- **Title text** (track name, section headers): bold
- **Body text** (artist name, list items): normal weight
- **Muted text** (album name, timestamps, counts): `TextMuted()` token, normal weight
- **Keybinding hints**: `KeyHint()` token for key label (e.g., `Space`), `TextMuted()` token for description

---

## Box Drawing

Use **rounded corners** exclusively. This is a distinguishing visual choice.

```
╭─────────────╮   ← Use this
│             │
╰─────────────╯

┌─────────────┐   ← Never use this
│             │
└─────────────┘
```

Pane separators use `│` for vertical and `─` for horizontal.

---

## Component Specifications

### Progress / Seek Bar

```
████████████████░░░░░░░░░░░░░░
2:34 ─────────────────── 4:12
```

- Fill character: `█` (U+2588)
- Empty character: `░` (U+2591)
- Width: fills the center pane minus padding
- Time labels: left-aligned elapsed, right-aligned total, with `─` fill between
- Color: `SeekBar()` token for filled, `Surface()` token for empty

### Volume Bar

```
VOL  ████████░░░░░░  65%
```

- Same character set as seek bar
- Width: 14 characters fixed
- Percentage shown on right
- Color: `VolumeBar()` token for filled

### Transport Controls

```
|<   ||   >|      ~   =>
```

- Symbols: `|<` (prev), `||` / `>` (pause/play), `>|` (next), `~` (shuffle), `=>` (repeat)
- **Use ASCII symbols, not Unicode emoji** — emoji render inconsistently across terminals
- Active state (shuffle on, repeat active): symbol rendered in `PlayingIndicator()` token
- Inactive state: `TextSecondary()` token
- Spacing: 3 spaces between each symbol

### Device Indicator (Header)

```
◉ MacBook Pro Speakers     ← active device
○ No device                ← nothing playing
```

- Active device: `◉` in `DeviceActive()` token
- No device: `○` in `TextMuted()` token
- Right-aligned in the header bar

### Playing Indicator in Lists

```
▶ Blinding Lights    ← currently playing
  Save Your Tears    ← not playing
```

- `▶` in `PlayingIndicator()` token for the currently playing track
- 2-space indent for non-playing items (to align with indicator width)

### Spinner (Loading State)

Use Bubbles' built-in spinner component. Default spinner type: `Dot`. Color: `ActiveBorder()` token.

```
⣾ Loading library...
⣽ Loading library...
```

---

## Animations & Transitions

All animations are **purely cosmetic** and must be implemented as `tea.Tick`-based commands, never blocking.

### Seek Bar Update
- Updates every **1000ms** (matching the playback polling rate)
- No animation between updates — just jump to new position
- The elapsed time label updates simultaneously

### Loading Spinner
- Appears when any async operation is in flight (API call, initial load)
- Disappears immediately when data arrives
- Use `bubbles/spinner` — do not build a custom spinner

### Track Change Transition
- When a new track starts, the center pane content **replaces instantly** — no fade
- The progress bar resets to 0 immediately

### Focus Change (Pane Border)
- When focus moves between panes, the border color changes from `InactiveBorder()` to `ActiveBorder()` token
- This happens in the same render cycle — no animation needed, the color change is the feedback

### Search Typing Debounce
- Input appears instantly (no delay)
- Results refresh 300ms after the last keypress (debounced)
- Show spinner in results area during the 300ms wait

### Error Messages
- Appear as toast notifications overlaid on the current view (bottom-right)
- Color: `Error()` token (`✗` prefix)
- Auto-dismiss after a fixed duration (configured in `components/notifications.go`)
- Routed exclusively through `app.go` — panes never render inline error text

---

## Toast Notifications

API errors and user feedback surface as floating toast overlays positioned at the bottom-right
of the terminal, rendered by `go.dalton.dog/bubbleup` via `internal/ui/components.NewNotifications`.

| Alert type | Prefix | Color | Trigger |
|---|---|---|---|
| `success` | `✓` | `Success()` | Successful actions (add to queue, device transfer) |
| `error` | `✗` | `Error()` | API failures, auth errors |
| `warning` | `!` | `Warning()` | Soft failures (Premium required) |
| `info` | `→` | `KeyHint()` | Informational (transfer in progress) |
| `ratelimit` | `⧖` | `Warning()` | 429 rate-limit back-off |

Toasts auto-dismiss after a fixed duration. No user action required.

---

## Status Bar

The bottom status bar is **always visible** and shows keybinding hints:

```
  /search   j/k move   Space play   Tab pane   d devices   ? help   q quit
```

The status bar no longer has an "error mode" — errors appear as toast overlays.

**Context mode** — hints change based on active pane:
- Library pane: show `Enter play  a queue  l like  2 stats  3 playlists`
- Player pane: show `Space play  +/- vol  s shuffle  r repeat  2 stats  3 playlists`
- Queue pane: show `Enter play  x remove  2 stats  3 playlists`
- Search overlay: show `Enter play  a queue  Tab section  Esc close`

**Status bar completeness:** The status bar MUST include ALL discoverable keys, including
view switchers (`1`/`2`/`3`). If a key exists in the help overlay, it should appear in
the status bar's context hints for the relevant state.

**Status bar ownership:** Only `app.go` renders the status bar. Panes and overlays MUST NOT
render their own help/hint bars. This prevents duplicate status bars and ensures consistent
styling. Panes return their content only; the root model wraps it with the status bar.

---

## Minimum Terminal Requirements

| Dimension | Minimum | Recommended |
|---|---|---|
| Width | 100 columns | 140+ columns |
| Height | 24 rows | 36+ rows |
| Colors | 256 | TrueColor (16M) |

On startup, check terminal dimensions. If below minimum, show:
```
╭─────────────────────────────────────────╮
│  Spotnik needs more space                 │
│                                         │
│  Current:  78 × 20                      │
│  Required: 100 × 24                     │
│                                         │
│  Please resize your terminal and retry. │
╰─────────────────────────────────────────╯
```

Check `os.Getenv("COLORTERM")` for TrueColor support. If not `truecolor` or `24bit`, fall back to the nearest 256-color approximation automatically (Lip Gloss handles this).

---

## Pane Scrolling

Panes with variable-length content MUST implement height-capped scrolling with scroll indicators.

- `View()` output MUST NOT exceed the height set by `SetSize()`
- Content that exceeds the visible area must be scrollable with `j`/`k`
- Show scroll indicators (`▲` at top, `▼` at bottom) when content extends beyond view
- Track `scrollOffset` to determine the visible window
- Affected panes: Queue, Library (when expanded sections have many items)

---

## Error State Rendering

All API errors are routed through the toast notification system (`internal/ui/components`).

- Pane `View()` methods MUST NOT render inline error boxes or check store error fields
- Store error fields (e.g. `store.DevicesError()`, `store.StatsError()`) are preserved for
  retry logic only — panes check them to decide whether to re-request on `f`/`Enter`
- `RenderError` from `components/errorview.go` is deprecated for pane error display; use
  `a.alerts.NewAlertCmd("error", message)` in `app.go` Update handlers instead
- The `DeviceOverlay`, `StatsView`, `PlaylistManager`, and `SearchOverlay` render their
  empty state (e.g. "No devices found") when no data is loaded — the toast informs the user why

---

## Volume Control

- Volume step size: configurable via `config.toml` (`volume_step`), default 5%
- Some Spotify Connect devices don't support volume control via the web API
- On volume control failure, show "Volume control not supported on this device"
- Never show raw API error JSON to the user

---

## Accessibility

- All state changes must be visible via color AND text/symbol — never color alone
  - Playing: `▶` symbol + green color (not just green)
  - Error: `✗` symbol + pink color + text message
  - Selected: highlight background + visual border, not just color
- Status bar always shows current context — users can always tell where they are
- `?` help is always available — never hide it

---

## What Agents Must Never Change (Design Guardrails)

1. **The three-pane layout** — proportions, arrangement, presence of all three panes
2. **Rounded corner box drawing** — `╭╮╰╯` not `┌┐└┘`
3. **Default theme (True Black)** — `black` is the default, never change without owner sign-off
4. **The status bar** — always at the bottom, always visible
5. **Keybinding assignments** — listed in CLAUDE.md, cross-referenced here
6. **The `▶` playing indicator** — always `PlayingIndicator` color, always left of currently playing
7. **Active pane border** — always `ActiveBorder` token from the active theme
8. **Search as an overlay** — never replace a pane with search
9. **Font weight rules** — bold for titles, normal for body, never all-caps for body text
10. **Pane section headers** — always `SectionHeader` token, always bold, always `─` divider
11. **No hardcoded hex values in components** — always use the `Theme` interface tokens
12. **Theme config key is `theme`** — never rename this config key

---

*Last updated: 2026-03-23*
*Status: FROZEN — changes require owner approval and version bump*
