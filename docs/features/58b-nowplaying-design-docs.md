# Feature 58b — Update DESIGN.md with NowPlaying Split Layout Design

> Update the design documentation to reflect the new btop-inspired NowPlaying pane
> split layout with InfoBox, Visualizer, and animation patterns.

---

## Feature Acceptance Criteria

1. DESIGN.md §2 (Pane Definitions) updated to mention the NowPlaying split layout
2. DESIGN.md §4 (Presets) updated — all preset diagrams for NowPlaying reflect the new split layout (InfoBox left, Visualizer right, seek bar bottom) instead of the old vertical stack
3. DESIGN.md §11 (Visual Components) updated to document the 3 animation patterns and `v` key cycling
4. DESIGN.md §11 NowPlaying layout section added describing the split layout proportions and responsive behavior
5. All preset diagrams (0, 1, 2, 3) show the new NowPlaying rendering style
6. `make ci` passes (the docs change shouldn't break anything)

---

## Task 1: Update DESIGN.md with NowPlaying split layout documentation

### Files to Modify

- `docs/DESIGN.md`

### What to Build

**§2 Pane Definitions — Key Notes section (around line 92-98):**
Add a note about the NowPlaying pane's split layout:
```
- NowPlaying pane uses a btop-inspired horizontal split layout: InfoBox sub-pane (~1/4 width, left) + Visualizer (~3/4 width, right) + gradient seek bar (full width, bottom)
```

**§4 Presets — Update all preset diagrams:**

For Preset 0 (Full Dashboard), Preset 1 (Listening) — update the NowPlaying row to show the split layout with InfoBox and Visualizer side by side.

For Preset 2 (Library) and Preset 3 (Discovery) — NowPlaying shows as a compact strip when height is small. When height < 8, the title bar embeds track info. The body still uses the split layout but scaled down.

Example for Preset 0 NowPlaying row:
```
╭─ ¹Now Playing ───────────────────────────────────────────── ᐅs shuffle ─ ᐅr repeat ─╮  Row 1 (weight 2)
│ ╭─ Track Info ──────╮ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿              │
│ │ Martbaan          │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿              │
│ │ Samar Mehdi       │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿              │
│ │ |<  ||  >|  ~ =>  │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿              │
│ │ VOL ████░░ 65%    │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿              │
│ ╰───────────────────╯ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿              │
│ 1:41  ████████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  5:30   │
╰──────────────────────────────────────────────────────────────────────────────╯
```

Example for Preset 1 (Listening) — NowPlaying gets more height, expanded visualizer:
```
╭─ ¹Now Playing ───────────────────────────────────────── ᐅs shuffle ─ ᐅr repeat ─╮  Row 1 (weight 3)
│                                                                                  │
│ ╭─ Track Info ──────────────╮  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ │                           │  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ │ Martbaan                  │  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ │ Samar Mehdi, June         │  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ │ Martbaan (Album)          │  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ │                           │  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ │ |<   ||   >|   ~   =>    │  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ │ VOL  ████████░░  65%     │  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ │                           │  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ ╰───────────────────────────╯  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│                                                                                  │
│ ▶  1:41  ████████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  5:30          │
╰──────────────────────────────────────────────────────────────────────────────────╯
```

**§11 Visual Components — Update Braille-Dot Audio Visualizer section:**

Add after the existing "Animation strategy" bullet:
```markdown
**Animation patterns:**
The visualizer supports 3 animation patterns, cycled manually via the `v` key:
- **Pattern 0 (Dual Sine Wave):** Two overlapping sine waves at different frequencies, producing a flowing ocean-like motion. This is the default pattern.
- **Pattern 1 (Standing Wave):** Interference of two counter-propagating waves creating stationary nodes and antinodes — bars pulse in place rather than traveling.
- **Pattern 2 (Pulse/Ripple):** A Gaussian peak travels left-to-right with a trailing ripple, like a sonar ping sweeping across the display.

Pattern state is local to the pane (not stored in the Store). `v` key always routes to NowPlaying via `isPlaybackKey()`.
```

**§11 — Add new "NowPlaying Split Layout" subsection after the Gradient-Colored Bars section:**

```markdown
### NowPlaying Split Layout (btop-inspired)

The NowPlaying pane uses a horizontal split layout inspired by btop's CPU pane:

**Layout proportions:**
- **Left (~1/4 width, min 28 chars):** InfoBox sub-pane — rounded border (`╭╮╰╯`), "Track Info" title, containing:
  - Track name (bold, `TextPrimary()`)
  - Artist names (`TextSecondary()`)
  - Album name (`TextMuted()`) — omitted when insufficient height
  - Controls row (`|< || >| ~ =>`)
  - Volume bar (`VOL ████░░ 65%`)
  - All content vertically centered within the InfoBox
- **Right (~3/4 width):** Animated braille visualizer (full body height)
- **Bottom (full width, 1 line):** Gradient seek bar with time labels

**Responsive behavior:**
- `infoWidth = max(contentWidth/4, 28)` — minimum 28 ensures controls fit
- `vizWidth = contentWidth - infoWidth - 1` — gap between regions
- `bodyHeight = paneHeight - borders - progressBar`
- When height < 8: title bar embeds track info (`Now Playing ── Track · Artist ── ▶ 1:41/5:30`)
- No separate compact mode — the split layout scales proportionally even in small presets

**InfoBox border:** Uses the project's standard rounded corners. Border color follows `ActiveBorder()`/`InactiveBorder()` based on pane focus state.
```

### Acceptance Criteria

- [ ] §2 updated with NowPlaying split layout note
- [ ] §4 preset diagrams updated for split layout
- [ ] §11 Visualizer section documents 3 animation patterns
- [ ] §11 new NowPlaying Split Layout subsection added
- [ ] `make ci` passes
