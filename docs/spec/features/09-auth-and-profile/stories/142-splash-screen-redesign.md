---
title: "Splash Screen — Duration, Font, and Layout Redesign"
feature: 09-auth-and-profile
status: open
---

## Background

The splash screen has three issues found during launch testing:

1. **Duration too long** — `Init()` fires `tea.Tick(5*time.Second, ...)` for the
   `splashDismissMsg`. 5 seconds feels too long; 2–3 seconds is the right range.

2. **Font doesn't match visual language** — `renderSplashView` uses `figure.NewFigure("SPOTNIK",
   "serifcap", false)`. The serifcap font uses curved serif characters that look out of place in
   a terminal app. The rest of Spotnik's TUI uses block/box drawing characters. The banner should
   use a font built from solid block characters (e.g. `▀▄█`) or the `dotmatrix` go-figure font
   which uses dot-grid characters matching the visualizer aesthetic.

3. **Version and premium notice have no visual framing** — the version string and "Playback
   controls require Spotify Premium" notice are plain `TextMuted` lines. They should be placed
   in a small bordered info panel below the banner, using theme colours so they look intentional
   rather than like an afterthought.

4. **Subtitle text** — "A terminal Spotify client for developers" → "A terminal Spotify client"
   (drop "for developers" per agreed tagline, Story 141 handles `cmd/root.go`; this story handles
   `internal/app/splash.go`).

## Design

### `internal/app/app.go` — splash timer duration

Change `5*time.Second` to `3*time.Second`:

```go
splashTimer := tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
    return splashDismissMsg{}
})
```

Update the comment that references the duration:
```go
// splashDismissMsg is sent after 3 seconds to close the splash screen.
```

### `internal/app/splash.go` — font and layout

**Font change**: Replace `"serifcap"` with `"dotmatrix"`. The dotmatrix font uses a dot-grid
style that pairs naturally with the bar visualizer's block characters. If `dotmatrix` does not
render cleanly at the available width (some go-figure fonts require wide terminals), fall back to
`"banner3-D"` which uses solid block characters. The implementer should test both and pick the
one that renders better at 120-column width.

```go
fig := figure.NewFigure("SPOTNIK", "dotmatrix", false)
```

**Banner styling**: Apply a gradient-style render by splitting the banner lines and coloring them
with alternating theme tokens (e.g. `ActiveBorder` for odd lines, `TextPrimary` for even lines),
or apply a single solid `ActiveBorder` colour. Keep it simple — one colour is fine; a two-tone
alternating scheme adds depth. The implementer chooses based on visual output.

**Info panel below the banner**: Wrap version + premium notice in a `RoundedBorder` box:

```
╭─────────────────────────────────────────────╮
│  v0.2.1      Spotify Premium required        │
╰─────────────────────────────────────────────╯
```

- Border in `TextMuted()` colour
- Version text in `Info()` colour
- Premium notice in `Warning()` colour
- `⚠` prefix before premium notice

Implementation:

```go
infoPanel := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(t.TextMuted()).
    Padding(0, 2).
    Render(
        lipgloss.JoinHorizontal(lipgloss.Center,
            lipgloss.NewStyle().Foreground(t.Info()).Render(version),
            "  ",
            lipgloss.NewStyle().Foreground(t.Warning()).Render("⚠  Spotify Premium required for playback"),
        ),
    )
```

**Updated subtitle**: change from "A terminal Spotify client for developers" to
"A terminal Spotify client":

```go
tagline := lipgloss.NewStyle().
    Foreground(t.TextMuted()).
    Render("A terminal Spotify client")
```

**Revised full layout**:

```go
content := lipgloss.JoinVertical(lipgloss.Center,
    bannerStyle.Render(banner),
    "",
    tagline,
    "",
    infoPanel,
)
```

The outer `lipgloss.Place` call is unchanged — it still centers `content` in the terminal.

### Tests — `internal/app/splash_test.go`

Add to the existing test file:

```go
func TestRenderSplashView_containsTagline(t *testing.T) {
    t.Setenv("TERM", "dumb") // disable colour codes in output
    out := renderSplashView(theme.NewBlack(), "v1.0.0", 160, 40)
    assert.Contains(t, out, "A terminal Spotify client")
    assert.NotContains(t, out, "for developers")
}

func TestRenderSplashView_containsVersion(t *testing.T) {
    out := renderSplashView(theme.NewBlack(), "v2.0.0", 160, 40)
    assert.Contains(t, out, "v2.0.0")
}

func TestRenderSplashView_containsPremiumNotice(t *testing.T) {
    out := renderSplashView(theme.NewBlack(), "v1.0.0", 160, 40)
    assert.Contains(t, out, "Premium")
}
```

## Acceptance Criteria

- [ ] Splash screen dismisses after **3 seconds** (not 5)
- [ ] Banner uses `"dotmatrix"` font (or `"banner3-D"` if dotmatrix renders poorly at 120 cols)
- [ ] Banner styled with `ActiveBorder()` theme colour
- [ ] Version and premium notice rendered in a `RoundedBorder` panel below the banner
- [ ] Version in `Info()` colour; premium notice in `Warning()` colour with `⚠` prefix
- [ ] Subtitle reads "A terminal Spotify client" (no "for developers")
- [ ] No hardcoded hex colour values anywhere in `splash.go`
- [ ] `TestRenderSplashView_containsTagline` — contains "A terminal Spotify client", NOT "for developers"
- [ ] `TestRenderSplashView_containsVersion` — contains injected version string
- [ ] `TestRenderSplashView_containsPremiumNotice` — contains "Premium"
- [ ] `make ci` passes

## Tasks

- [ ] Write failing tests in `internal/app/splash_test.go` for the three test cases above
      - test: `go test ./internal/app/... -run "TestRenderSplashView" -v` → FAIL (old text still present)
- [ ] Change splash timer from `5*time.Second` to `3*time.Second` in `internal/app/app.go`
      - test: `go build ./...` → clean; update comment referencing "2 seconds"
- [ ] Update `renderSplashView` in `internal/app/splash.go`:
        - Change font to `"dotmatrix"` (or `"banner3-D"`)
        - Update tagline string
        - Replace version+notice plain lines with `infoPanel` bordered box
      - test: all `TestRenderSplashView_*` → PASS; `go build ./...` → clean
- [ ] `make ci` passes
