---
title: "Fix: Help bar key names use muted colors and ctrl+u missing from short view"
feature: 19-search-redesign
status: done
---

## Background

The `bubbles/help` component renders key binding help text. By default it uses its
own built-in styles which are hard-coded to muted/dim colors (`#626262` dark,
`#909090` light). No theme colors are applied, so the key names are barely readable
and do not match the rest of the overlay's visual language.

Additionally, `ctrl+u` (clear search) is defined in `searchKeyMap.Clear` and appears
in `FullHelp()` but is **missing from `ShortHelp()`**. The user has no way to discover
it from the compact bar shown at the bottom of the overlay.

### How bubbles/help styles work

`help.Model` has a `Styles` field of type `help.Styles`:

```go
type Styles struct {
    ShortKey       lipgloss.Style  // style for key names in compact view
    ShortDesc      lipgloss.Style  // style for key descriptions in compact view
    ShortSeparator lipgloss.Style  // style for "•" separators between bindings
    Ellipsis       lipgloss.Style  // style for "…" when bindings are truncated
    FullKey        lipgloss.Style  // style for key names in full view
    FullDesc       lipgloss.Style  // style for key descriptions in full view
    FullSeparator  lipgloss.Style  // style for separators in full view
}
```

Default values from `charmbracelet/bubbles/help/help.go`:

```go
DefaultStyles = Styles{
    ShortKey:       lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#909090", Dark: "#626262"}),
    ShortDesc:      lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#B2B2B2", Dark: "#4A4A4A"}),
    ShortSeparator: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#DDDADA", Dark: "#3C3C3C"}),
    ...
}
```

These are completely independent of the `Theme` interface — they never update when
the user switches themes.

### Current initialization (search.go, NewSearchOverlay)

```go
h := help.New()
km := NewSearchKeyMap()
```

No style overrides. `h.Styles` is the default muted palette.

### Fix

After `h := help.New()`, set `ShortKey` and `ShortDesc` from theme tokens:

```go
h := help.New()
h.Styles.ShortKey = lipgloss.NewStyle().Foreground(t.Info())
h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(t.TextMuted())
h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(t.TextMuted())
h.Styles.FullKey = lipgloss.NewStyle().Foreground(t.Info())
h.Styles.FullDesc = lipgloss.NewStyle().Foreground(t.TextMuted())
h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(t.TextMuted())
```

`t.Info()` gives a visible, colored (typically cyan/blue) foreground for key names.
`t.TextMuted()` keeps descriptions and separators secondary without losing readability.

### Theme switching

`SetTheme()` in `search.go` already propagates theme changes to the spinner, delegate,
and input styles. It must also update `o.help.Styles` so help colors change with the theme:

```go
func (o *SearchOverlay) SetTheme(t theme.Theme) {
    o.theme = t
    // ... existing propagations ...

    // Propagate to help styles.
    o.help.Styles.ShortKey = lipgloss.NewStyle().Foreground(t.Info())
    o.help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(t.TextMuted())
    o.help.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(t.TextMuted())
    o.help.Styles.FullKey = lipgloss.NewStyle().Foreground(t.Info())
    o.help.Styles.FullDesc = lipgloss.NewStyle().Foreground(t.TextMuted())
    o.help.Styles.FullSeparator = lipgloss.NewStyle().Foreground(t.TextMuted())
}
```

### Add ctrl+u to ShortHelp

`ShortHelp()` currently returns 5 bindings (Play, Queue, TabNext, TabPrev, Close).
`Clear` is only in `FullHelp()`. Add `Clear` to `ShortHelp()` at position 5 (before Close
or replacing it, since Close is discoverable via Esc which is standard):

```go
func (k searchKeyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Play, k.Queue, k.TabNext, k.TabPrev, k.Clear, k.Close}
}
```

**File: `internal/ui/panes/search.go`**
- In `NewSearchOverlay()`: set `h.Styles.ShortKey`, `ShortDesc`, `ShortSeparator`,
  `FullKey`, `FullDesc`, `FullSeparator` using theme tokens
- In `SetTheme()`: propagate the same six style overrides
- In `ShortHelp()`: add `k.Clear` binding before `k.Close`

## Acceptance Criteria

- [ ] Key names (e.g. `enter`, `ctrl+a`, `tab`) in the help bar are rendered in
      `theme.Info()` color, not muted gray
- [ ] Key descriptions (e.g. `play`, `queue`, `filter`) are rendered in `theme.TextMuted()`
- [ ] `ctrl+u  clear` appears in the compact short help bar (the bar shown at the
      bottom of the overlay)
- [ ] Switching themes (via `SetTheme()`) immediately updates the help bar colors
- [ ] All six `help.Styles` fields (ShortKey, ShortDesc, ShortSeparator, FullKey,
      FullDesc, FullSeparator) are set from theme tokens both at construction and on
      theme switch

## Tasks

- [ ] In `NewSearchOverlay()`, override all six `h.Styles` fields after `h := help.New()`
      - test: construct `NewSearchOverlay(store, theme)`; call `o.help.View(o.keyMap)`;
        verify the output contains the ANSI foreground sequence for `theme.Info()` color
        (which wraps key names) — use `lipgloss.NewStyle().Foreground(t.Info()).Render("enter")`
        as the reference; verify the rendered help text contains that sequence

- [ ] In `SetTheme()`, propagate the same six style overrides to `o.help.Styles`
      - test: construct overlay with theme A; call `SetTheme(themeB)`;
        verify `o.help.Styles.ShortKey` foreground equals `themeB.Info()`;
        verify `o.help.Styles.ShortDesc` foreground equals `themeB.TextMuted()`

- [ ] Add `k.Clear` to `ShortHelp()` return slice (position 5, before `k.Close`)
      - test: `km := NewSearchKeyMap(); bindings := km.ShortHelp()`; verify `len(bindings) == 6`;
        verify one binding has key `"ctrl+u"` and help text `"clear"`;
        verify `k.Close` is also present

- [ ] `make ci` passes with no regressions
