# Feature 58 — NowPlaying Split Layout (btop-inspired)

> Rewrite NowPlayingPane View() with a btop-inspired split layout: InfoBox left (~1/4),
> Visualizer right (~3/4), gradient seek bar spanning full width at the bottom.

---

## Feature Acceptance Criteria

1. NowPlayingPane View() renders a horizontal split: InfoBox on the left, Visualizer on the right
2. A gradient seek bar spans full width at the bottom
3. The InfoBox contains track name, artist, album, controls, and volume bar — vertically centered
4. The compact bool field and renderCompact() method are removed
5. Title() shows track info in the title bar when height < 8 (no separate compact flag)
6. The layout adapts proportionally to different pane sizes
7. All existing tests updated to match new layout
8. New tests verify split layout contains InfoBox borders, braille, and seek bar
9. `make ci` passes (lint + tests + 80% coverage)

---

## Task 1: Rewrite NowPlayingPane View() with split layout

### Files to Modify

- `internal/ui/panes/nowplaying.go`
- `internal/ui/panes/nowplaying_test.go`

### What to Build

**Add InfoBox field to NowPlayingPane struct:**
- Add `infoBox *components.InfoBox` field
- Initialize it in `NewNowPlayingPane` constructor: `infoBox: components.NewInfoBox(t)`

**Update SetSize() to compute split dimensions:**
```go
func (p *NowPlayingPane) SetSize(width, height int) {
    p.width = width
    p.height = height

    contentWidth := paneMax(width-4, 10)

    // Split layout dimensions
    infoWidth := paneMax(contentWidth/4, 28) // minimum 28 chars for controls
    vizWidth := contentWidth - infoWidth - 1  // -1 for gap between regions

    progressHeight := 1
    bodyHeight := paneMax(height-4, 4) - progressHeight // subtract border + progress

    p.infoBox.SetSize(infoWidth, bodyHeight)
    p.visualizer.SetSize(vizWidth, bodyHeight)
    p.seekBar.SetWidth(contentWidth)
    p.volumeBar.SetWidth(infoWidth - 4) // fits inside InfoBox with border padding
}
```

**Remove compact mode:**
- Remove `compact bool` field from NowPlayingPane struct
- Delete `renderCompact()` method entirely
- Delete `interpolateHexCompact()`, `parseHexParts()`, `lerpByte()` helper functions (they were only used by renderCompact)

**Rewrite View():**
```go
func (p *NowPlayingPane) View() string {
    ps := p.store.PlaybackState()
    if ps == nil || ps.Item == nil {
        return p.renderEmpty()
    }

    t := ps.Item

    // Build info lines for the InfoBox
    primaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary()).Bold(true)
    secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
    mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

    artistNames := make([]string, len(t.Artists))
    for i, a := range t.Artists {
        artistNames[i] = a.Name
    }

    volume := 0
    if ps.Device != nil {
        volume = ps.Device.VolumePercent
    }
    ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)

    infoLines := []string{
        primaryStyle.Render(t.Name),
        secondaryStyle.Render(strings.Join(artistNames, ", ")),
        mutedStyle.Render(t.Album.Name),
        "",
        ctrl.Render(),
        p.volumeBar.Render(volume),
    }

    // Render InfoBox (left) and Visualizer (right)
    infoView := p.infoBox.Render("Track Info", infoLines, p.focused)
    vizView := p.visualizer.View()

    // Join left and right side by side
    body := lipgloss.JoinHorizontal(lipgloss.Top, infoView, " ", vizView)

    // Seek bar at bottom (full width)
    seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)

    return lipgloss.JoinVertical(lipgloss.Left, body, seekBar)
}
```

**Update Title():**
```go
func (p *NowPlayingPane) Title() string {
    if p.height < 8 {
        ps := p.store.PlaybackState()
        if ps != nil && ps.Item != nil {
            t := ps.Item
            artistNames := make([]string, len(t.Artists))
            for i, a := range t.Artists {
                artistNames[i] = a.Name
            }
            playSymbol := "▶"
            if !ps.IsPlaying {
                playSymbol = "⏸"
            }
            current := formatDurationMs(p.localProgressMs)
            total := formatDurationMs(t.DurationMs)
            return fmt.Sprintf("Now Playing ── %s · %s ── %s %s/%s",
                t.Name, strings.Join(artistNames, ", "), playSymbol, current, total)
        }
    }
    return "Now Playing"
}
```

### Test Updates

**Tests to update:**
- `TestNowPlayingPane_CompactMode_EnabledAtHeight3` — DELETE (no compact mode)
- `TestNowPlayingPane_CompactMode_DisabledAtHeight10` — DELETE
- `TestNowPlayingPane_CompactMode_DisabledAtHeight4` — DELETE
- `TestNowPlayingPane_CompactTitle_IncludesTrackInfo` — UPDATE: test Title() when height < 8
- `TestNowPlayingPane_CompactView_SingleContentLine` — DELETE
- `TestNowPlayingPane_CompactView_ContainsVol` — DELETE
- `TestNowPlayingPane_NoVisualizerInCompactMode` — DELETE
- `TestNowPlayingPane_Transition_FullToCompact` — DELETE
- `TestNowPlayingPane_CompactView_NilState` — DELETE

**New tests to add:**
- `TestNowPlayingPane_SplitLayout_ContainsInfoBoxBorders` — View() at 80x24 contains `╭` and `╰` (InfoBox rounded borders)
- `TestNowPlayingPane_SplitLayout_ContainsBraille` — View() contains braille characters (visualizer on right)
- `TestNowPlayingPane_SplitLayout_ContainsSeekBar` — View() contains time stamps from seek bar
- `TestNowPlayingPane_SplitLayout_ContainsVolumeInInfoBox` — View() contains "VOL"
- `TestNowPlayingPane_SplitLayout_ContainsControls` — View() contains control characters
- `TestNowPlayingPane_Title_ShowsTrackInfoWhenSmall` — Title() at height 6 includes track name
- `TestNowPlayingPane_Title_DefaultWhenTall` — Title() at height 24 is "Now Playing"
- `TestNowPlayingPane_SplitLayout_AdaptsToDifferentSizes` — different sizes produce different output

### Acceptance Criteria

- [ ] NowPlayingPane has `infoBox` field initialized in constructor
- [ ] SetSize computes split dimensions for InfoBox, Visualizer, seek bar, volume bar
- [ ] `compact` field and `renderCompact()` removed
- [ ] `interpolateHexCompact`, `parseHexParts`, `lerpByte` removed
- [ ] View() renders horizontal split: InfoBox left, Visualizer right, seek bar bottom
- [ ] Title() uses height < 8 check instead of compact flag
- [ ] All old compact-mode tests deleted
- [ ] New split layout tests added and passing
- [ ] `make ci` passes

---

## Design Diagram

```
Full Mode (any preset with height >= 6):

╭─ ¹Now Playing ─────────────────────────────────────── ᐅs shuffle ─ ᐅr repeat ─╮
│                                  │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿  │
│ ╭─ Track Info ────────────╮      │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿  │
│ │ Rein Me In              │      │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿  │
│ │ Sam Fender, Olivia Dean │      │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿  │
│ │ Live at London Stadium  │      │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿  │
│ │                         │      │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿  │
│ │ |<  ||  >|   ~  =>     │      │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿  │
│ │ VOL ████████░░ 65%      │      │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿  │
│ ╰─────────────────────────╯      │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿  │
│ 2:58 ██████████████████████████████████████████████░░░░░░░░░░░░░░░░░░░░░ 5:39  │
╰────────────────────────────────────────────────────────────────────────────────╯
```
