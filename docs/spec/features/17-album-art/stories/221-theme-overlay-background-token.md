---
title: "Theme.OverlayBackground token + InfoBox solid background fill"
feature: 17-album-art
status: open
---

## Background

The next story (222) removes album art and rebuilds NowPlaying as a single
overlay region: the visualizer fills the full content area, and the InfoBox is
composited on top of the left ~25% of the visualizer with a solid background
fill. Without a solid background, the visualizer animation bleeds through the
InfoBox interior and makes the text unreadable.

The current `InfoBox` renders its interior as plain text (spaces, no
background style). The fix needs a theme token that names "the solid colour to
fill panels that overlay dynamic content" so future overlay components can
reuse the same primitive. Every existing theme should map this token to its
own `Base()` — the darkest solid canvas colour each theme already exposes.

This story ships the theme token and the InfoBox fill independently of the
NowPlaying rewrite so the theme + component change can land and bake before
the larger pane refactor in story 222.

## Design

### `internal/ui/theme/theme.go`

Add one method to the `Theme` interface, alongside the existing background
tokens (`Base`, `Surface`, `SurfaceAlt`):

```go
// OverlayBackground is the solid background used for floating panels that
// sit on top of dynamic content (e.g., the NowPlaying InfoBox overlaid on
// the visualizer). Falls back to Base() in all built-in themes.
OverlayBackground() lipgloss.Color
```

### `internal/ui/theme/config_theme.go`

Add the method on `ConfigTheme` (the single concrete implementation behind
all 11 TOML-loaded themes — see feature 16). Aliasing to `Base()` satisfies
every built-in theme without touching the TOML files:

```go
// OverlayBackground returns the solid background for floating panels
// overlaid on dynamic content. Currently aliased to Base() — every theme
// uses its base canvas colour.
func (t *ConfigTheme) OverlayBackground() lipgloss.Color {
    return t.Base()
}
```

### `internal/ui/components/infobox.go`

In `Render`, after the interior string is assembled (`interior := sb.String()`)
and before the `PaneChrome.Render` call, wrap the interior with a lipgloss
style that sets the background to `th.OverlayBackground()`:

```go
// Apply the overlay background fill so the InfoBox sits on top of
// dynamic content (e.g., the NowPlaying visualizer). PaneChrome renders
// the border; we only need to fill the interior.
bgStyle := lipgloss.NewStyle().Background(b.th.OverlayBackground())
interior = bgStyle.Render(interior)
```

The border styling stays with `PaneChrome` — unchanged. The interior already
sizes to the InfoBox inner width/height, so the background fill covers exactly
the right region.

### Tests

`internal/ui/theme/theme_test.go`:

```go
func TestTheme_OverlayBackground_EqualsBase(t *testing.T) {
    themes := AllThemes()
    require.NotEmpty(t, themes, "no themes loaded")
    for _, th := range themes {
        t.Run(th.ID(), func(t *testing.T) {
            assert.Equal(t, th.Base(), th.OverlayBackground(),
                "OverlayBackground() must equal Base() for theme %q", th.ID())
        })
    }
}
```

`internal/ui/components/infobox_test.go`:

```go
func TestInfoBox_OverlayBackground_AppliedToInterior(t *testing.T) {
    th := theme.Load("black")
    ib := components.NewInfoBox(th)
    ib.SetSize(20, 6)
    out := ib.Render("Track Info", []string{"line1", "line2"}, true)
    // Any ANSI background escape (48;2;... truecolor or 4x basic) is acceptable.
    assert.Contains(t, out, "\x1b[4",
        "InfoBox output must contain an ANSI background escape")
}
```

## Acceptance Criteria

- [ ] `Theme.OverlayBackground() lipgloss.Color` added to interface in `internal/ui/theme/theme.go`
- [ ] `ConfigTheme.OverlayBackground()` returns `t.Base()`
- [ ] All 11 built-in themes (black, jellyfish, gruvbox, dracula, nord, tokyo-night, catppuccin, solarized-dark, solarized-light, mono-dark, mono-light) return their own `Base()` from `OverlayBackground()`
- [ ] `InfoBox.Render` wraps interior with `lipgloss.NewStyle().Background(th.OverlayBackground())` before passing to `PaneChrome.Render`
- [ ] `TestTheme_OverlayBackground_EqualsBase` passes for every theme returned by `AllThemes()`
- [ ] `TestInfoBox_OverlayBackground_AppliedToInterior` passes — rendered output contains an ANSI background escape
- [ ] Existing InfoBox tests still pass (update any expected-output assertions if they previously assumed no background escape)
- [ ] No change to InfoBox border, title, or layout primitives — only the interior fill
- [ ] `make ci` passes

## Tasks

- [ ] Write failing test `TestTheme_OverlayBackground_EqualsBase` in `internal/ui/theme/theme_test.go`
      iterating over `AllThemes()` and asserting `th.OverlayBackground() == th.Base()`
      - run: `rtk go test ./internal/ui/theme/... -run TestTheme_OverlayBackground -v` → expect FAIL (method undefined)

- [ ] Add `OverlayBackground() lipgloss.Color` to the `Theme` interface in `internal/ui/theme/theme.go`
      directly after the `SurfaceAlt()` method declaration in the `// Backgrounds` section,
      with a doc comment naming the use case (overlay panels on dynamic content)

- [ ] Add `OverlayBackground()` method on `ConfigTheme` in `internal/ui/theme/config_theme.go`
      after the `Accent()` method, returning `t.Base()`
      - run: `rtk go test ./internal/ui/theme/... -run TestTheme_OverlayBackground -v` → expect PASS

- [ ] Commit theme changes:
      `feat(theme): add OverlayBackground() token returning Base()`

- [ ] Write failing test `TestInfoBox_OverlayBackground_AppliedToInterior` in
      `internal/ui/components/infobox_test.go` — render an InfoBox with `theme.Load("black")`,
      assert the output contains the ANSI escape prefix `\x1b[4` (matches any background colour code)
      - run: `rtk go test ./internal/ui/components/... -run TestInfoBox_OverlayBackground -v` → expect FAIL

- [ ] In `internal/ui/components/infobox.go` `Render` method, after `interior := sb.String()`
      and before the `PaneChrome.Render` delegate block, wrap `interior` with
      `lipgloss.NewStyle().Background(b.th.OverlayBackground()).Render(interior)`
      - run: `rtk go test ./internal/ui/components/... -run TestInfoBox_OverlayBackground -v` → expect PASS

- [ ] Run the full InfoBox test suite to catch regressions:
      `rtk go test ./internal/ui/components/... -v`
      - Update any existing exact-output assertions to include the new background escape
        if they break (snapshot-style tests, not behaviour tests)

- [ ] Commit InfoBox changes:
      `feat(infobox): apply OverlayBackground fill to interior`

- [ ] Run `make ci` — must pass lint, all tests, and 80% coverage gate
