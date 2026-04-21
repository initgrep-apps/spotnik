---
title: "Splash Screen — Duration, Font, and Layout Redesign"
feature: 09-auth-and-profile
status: done
---

## Background

The splash screen has four issues found during post-launch testing:

1. **Duration too long** — `Init()` fires `tea.Tick(5*time.Second, ...)` for the
   `splashDismissMsg`. 5 seconds is noticeably slow; 3 seconds is the right balance between
   a visible brand moment and a snappy startup.

2. **Font clashes with visual language** — `renderSplashView` uses
   `figure.NewFigure("SPOTNIK", "serifcap", false)`. The serifcap font produces curved serif
   characters (like `╔╗║`) that look out of place in a TUI built on box-drawing and block
   characters. The braille visualizer and controls bar use Unicode block/dot characters. The
   banner should match that aesthetic. Target: `"dotmatrix"` go-figure font, which renders
   letters as a dot-grid matching the braille visualizer. If dotmatrix does not render cleanly
   at 120-column width, `"banner3-D"` (solid block fills) is the fallback.

3. **Version and premium notice have no visual framing** — the version string and "Playback
   controls require Spotify Premium" line are plain `TextMuted` lines below the banner. They
   look like an afterthought. They should be grouped in a small rounded-border info panel using
   theme colours.

4. **Subtitle says "for developers"** — `splash.go` renders "A terminal Spotify client for
   developers". Per the agreed shorter tagline, drop "for developers".

**Depends on:** nothing — changes are confined to `internal/app/app.go` and `internal/app/splash.go`.

## Design

### `internal/app/app.go` — timer duration

Change the splash timer from 5 seconds to 3 seconds:

```go
// Before
splashTimer := tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
    return splashDismissMsg{}
})

// After
splashTimer := tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
    return splashDismissMsg{}
})
```

Update the comment on `splashDismissMsg` type declaration in `app.go`:

```go
// splashDismissMsg is sent after 3 seconds to close the splash screen.
type splashDismissMsg struct{}
```

### `internal/app/splash.go` — font, layout, and subtitle

**Font**: Replace `"serifcap"` with `"dotmatrix"`. The implementer must render both at 120
columns and choose whichever looks cleaner. If dotmatrix produces garbled output (some go-figure
fonts require very wide terminals), use `"banner3-D"` instead.

```go
fig := figure.NewFigure("SPOTNIK", "dotmatrix", false)
```

**Subtitle**: Change "A terminal Spotify client for developers" → "A terminal Spotify client".

**Info panel**: Wrap the version string and premium notice in a bordered panel instead of bare
`TextMuted` lines:

```go
infoPanelStyle := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(t.TextMuted()).
    Padding(0, 2)

versionLine := lipgloss.NewStyle().
    Foreground(t.TextPrimary()).
    Render("Version " + version)

premiumLine := lipgloss.NewStyle().
    Foreground(t.Warning()).
    Render("⚠  Playback controls require Spotify Premium")

infoPanel := infoPanelStyle.Render(
    lipgloss.JoinVertical(lipgloss.Left, versionLine, premiumLine),
)
```

Full `renderSplashView` after changes:

```go
func renderSplashView(t theme.Theme, version string, width, height int) string {
    fig := figure.NewFigure("SPOTNIK", "dotmatrix", false)
    banner := fig.String()

    bannerStyle := lipgloss.NewStyle().
        Foreground(t.ActiveBorder()).
        Bold(true)

    tagline := lipgloss.NewStyle().
        Foreground(t.TextMuted()).
        Render("A terminal Spotify client")

    infoPanelStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(t.TextMuted()).
        Padding(0, 2)

    versionLine := lipgloss.NewStyle().
        Foreground(t.TextPrimary()).
        Render("Version " + version)

    premiumLine := lipgloss.NewStyle().
        Foreground(t.Warning()).
        Render("⚠  Playback controls require Spotify Premium")

    infoPanel := infoPanelStyle.Render(
        lipgloss.JoinVertical(lipgloss.Left, versionLine, premiumLine),
    )

    content := lipgloss.JoinVertical(lipgloss.Center,
        bannerStyle.Render(banner),
        "",
        tagline,
        "",
        infoPanel,
    )

    return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
```

### Tests — `internal/app/splash_test.go`

Update existing tests to reflect the new subtitle and panel structure:

```go
func TestRenderSplashView_containsTagline(t *testing.T) {
    th := theme.NewBlack()
    view := renderSplashView(th, "v0.1.0", 120, 40)
    assert.Contains(t, view, "A terminal Spotify client")
    assert.NotContains(t, view, "for developers")
}

func TestRenderSplashView_containsVersion(t *testing.T) {
    th := theme.NewBlack()
    view := renderSplashView(th, "v1.2.3", 120, 40)
    assert.Contains(t, view, "v1.2.3")
}

func TestRenderSplashView_smallTerminal_noPanic(t *testing.T) {
    th := theme.NewBlack()
    // Should not panic even at small sizes.
    view := renderSplashView(th, "dev", 40, 10)
    assert.NotEmpty(t, view)
}
```

## Acceptance Criteria

- [ ] Splash dismisses after 3 seconds (not 5)
- [ ] Banner font is `"dotmatrix"` (or `"banner3-D"` if dotmatrix renders poorly at 120 cols)
- [ ] Subtitle reads "A terminal Spotify client" — no "for developers"
- [ ] Version string and premium notice are inside a `RoundedBorder` panel using theme colours
- [ ] Version uses `TextPrimary`, premium notice uses `Warning` colour
- [ ] `TestRenderSplashView_containsTagline` passes and asserts no "for developers"
- [ ] `make ci` passes

## Tasks

- [ ] Change `5*time.Second` → `3*time.Second` in `internal/app/app.go`; update `splashDismissMsg` comment
      - test: `go build ./...` → clean
- [ ] In `internal/app/splash.go`: change font to `"dotmatrix"`, update subtitle, add `infoPanelStyle` layout
      - test: `go run . 2>/dev/null` or visual inspection — banner renders at 120 cols without garbling; if garbled, switch to `"banner3-D"`
- [ ] Update `TestRenderSplashView_*` tests in `internal/app/splash_test.go`
      - test: `go test ./internal/app/... -run "TestRenderSplash" -v` → PASS
- [ ] `make ci` → PASS
