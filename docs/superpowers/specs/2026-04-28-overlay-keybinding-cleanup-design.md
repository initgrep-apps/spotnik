# Overlay Keybinding Cleanup — Design Spec

> Status: draft
> Owner: Irshad
> Date: 2026-04-28

## Goal

Two overlays carry keybinding presentation cruft that contradicts the documented
hint pattern in `docs/TUI-DESIGN-SYSTEM.md` and duplicates information across
border notches and stacked help panels.

This spec eliminates that duplication, removes one truly dead binding
(`Ctrl+U` clear), unifies the keybinding renderer to `uikit.KeyBar`, and
restructures the profile overlay into an aligned icon+value table with an
inline KeyBar for actions.

## Scope

**In scope**
- `internal/ui/panes/search.go` — search overlay keymap, render, Update
- `internal/ui/panes/search_test.go` — affected tests
- `internal/ui/panes/profile.go` — profile overlay layout + glyph wiring
- `internal/ui/panes/profile_test.go` — affected tests
- `docs/keybinding.md`, `docs/DESIGN.md §17`, `internal/ui/panes/help_overlay.go` `helpContent` — keybinding doc trio (CLAUDE.md rule #15)
- `docs/TUI-DESIGN-SYSTEM.md` — only if a new glyph role is added (it is not; see §4)

**Out of scope**
- Other overlays (devices, themes, help itself, onboarding) — they already
  follow the KeyBar pattern.
- Onboarding screens — referenced as the visual model only; no edits there.
- Behavioral changes to Tab cycling, prefix state machine, pagination, double-press
  confirmation logic.

## Search Overlay

### Keymap

`searchKeyMap` shrinks from 8 bindings to 5. Removed bindings and rationale:

| Removed | Reason |
|---|---|
| `Play` (`enter`) | Implicit primary action, like Esc — never advertised. Update() still handles `tea.KeyEnter`. |
| `Close` (`esc`) | Implicit dismiss, universal across overlays. Update() still handles `tea.KeyEsc`. |
| `Clear` (`ctrl+u`) | Functionality removed entirely — query clears only when the user mutates the input. The Ctrl+U handler in Update() is deleted. |

`bubbles/help.Model` and the `searchKeyMap.FullHelp() / ShortHelp()` methods are
deleted along with the `help` field on `SearchOverlay`. The `KeyMap`
struct keeps `Queue`, `TabNext`, `TabPrev`, `nextPage`, `prevPage`. `TabNext`
help text changes from `tab filter` to `tab category` (matches actual behavior:
cycle the result tab bar `[All] Songs Artists Albums Playlists`).

### Render

`renderHelpPanel(w, h)` swaps from `help.View(keyMap)` to a single-line
`uikit.KeyBar` over a hand-built binding list:

```
ctrl+a queue · tab/shift+tab category · pgdn/pgup page
```

To produce the grouped middle and right cells without inventing new key.Binding
plumbing, build three synthetic `key.Binding` entries with composite Key strings
(`"tab/shift+tab"`, `"pgdn/pgup"`) using `key.WithHelp` only — they exist solely
for `KeyBar.Render()` consumption, are never matched against input, and live in
a private helper `searchHintBindings()` next to the new render function.

The Keys panel border stays — height becomes 3 (top border + single content
line + bottom border), down from 4 today. `panelHeights()` is updated
accordingly. `BorderConfig` for the Keys panel keeps `Title: ""` and
`Actions: nil`.

### Results panel border

`renderResultsPanel` BorderConfig drops `Actions`. The `Enter play / Ctrl+A queue`
corner notches disappear entirely — the bottom KeyBar is the single source of
truth for visible bindings.

### Update flow

- `tea.KeyEnter` → `handleEnter()` unchanged (Play behavior preserved, just
  no longer advertised).
- `tea.KeyEsc` → close overlay unchanged.
- `case key.Matches(m, o.keyMap.Clear):` and the surrounding Ctrl+U branch
  in `handleKey` are deleted. `ctrl+u` reaches the textinput which ignores it.

### Tests

- `search_test.go` cases asserting on `help.View()` output / FullHelp grouping
  are rewritten against `KeyBar` rendered output. ANSI-aware substring matches
  (existing `stripANSI` helper) for `ctrl+a queue`, `tab/shift+tab category`,
  `pgdn/pgup page`.
- `Ctrl+U clears query` test deleted; new test asserts Ctrl+U is a no-op (query
  unchanged after key press).
- `KeysPanelTitle` test stays (title is still empty).
- Pagination height regressions: a height-arithmetic test confirms
  `panelHeights()` returns `helpH=3` and that `resultsH` grew by 1.

## Profile Overlay

### Layout

```
╭─Profile──────────────╮
│ ◉  Irshad            │
│ ♛  Premium           │
│ ◎  IN                │
│                      │
│ l logout · f forget  │
╰──────────────────────╯
```

Three icon+value rows, one blank spacer line, one `uikit.KeyBar` line, all inside
the existing single border. Width stays 34 inner. Height becomes
`3 (rows) + 1 (spacer) + 1 (keys) + 2 (borders) = 7`.

### Glyphs

All three icons sourced from `uikit.GlyphFor(role, ActiveMode())`:

| Row | Role | Unicode |
|---|---|---|
| Name | `GlyphActive` | `◉` |
| Plan | `GlyphPremium` (Premium) / `GlyphFreeTier` (Free) | `♛` / `○` |
| Region | `GlyphInactive` | `◎` |

The hardcoded `"♛"`, `"◎"`, `"○"` runes in `View()` are replaced.
No new glyph role is introduced — `GlyphActive` is reused for the Name row
(profile has exactly one user; semantic overlap with the active-radio meaning
is acceptable and keeps the glyph table stable).

### Row rendering

Rows use a tiny inline helper rather than `uikit.ListRow`:

```go
func (p *ProfileOverlay) renderRow(glyph string, glyphColor, valueColor lipgloss.Color, value string) string {
    g := lipgloss.NewStyle().Foreground(glyphColor).Render(glyph)
    v := lipgloss.NewStyle().Foreground(valueColor).Render(value)
    return g + "  " + v
}
```

Reason: `ListRow` carries label-truncation and selection chrome that this card
does not need; a 4-line helper is clearer than configuring `ListRow` to disable
both. Width is the existing `innerWidth = 34`; lipgloss left-aligns naturally.

### Key bar

```go
bindings := []key.Binding{
    key.NewBinding(key.WithHelp("l", "logout")),
    key.NewBinding(key.WithHelp("f", "forget")),
}
keys := uikit.KeyBar{Bindings: bindings, Theme: p.theme}.Render()
```

Synthetic `key.Binding`s — never matched, only rendered. Real key handling
stays in `Update()` as today, including the double-press confirm armed-state
machine and `ProfileConfirmToastMsg`.

### Removed elements

- The two `sepStyle` separator rules (`────────────────────`).
- The bold name styling block (replaced by glyph row).
- `truncateRunes` call on display name — kept as a helper (still used to cap
  the Name row at `maxProfileNameLen=20`).
- `renderActions()` method and its `ListRow` usage.

### Tests

- `profile_test.go` cases asserting on `Logout` / `Forget` rendered as
  separate lines via `ListRow` are rewritten to assert a single line containing
  both via `KeyBar`.
- A new test asserts the three rows render with glyph + two-space gap + value
  in order (Name, Plan, Region).
- A new test asserts `GlyphASCII` mode produces `(*) Irshad`, `*P Premium`,
  `( ) IN` (validates glyph system wiring).
- Existing double-press confirm test stays intact (Update() unchanged).

## Doc Trio Update

Per CLAUDE.md rule #15, all three keybinding doc surfaces update in the same
commit:

- `docs/keybinding.md` — Search Overlay section: delete the rows for `Enter`
  (`Play selected result`), `Ctrl+U` (`Clear search input`), and `Esc`
  (`Close search overlay`). The `Tab / Shift+Tab` row already reads
  `Cycle search category`, so no relabel needed there. Profile section
  stays unchanged (already says `l Logout`, `f Forget`).
- `docs/DESIGN.md §17` — same row deletions in the search-overlay table.
  Inspect §17 first; if it currently uses a `filter` label for Tab, change
  to `category` to match keybinding.md.
- `internal/ui/panes/help_overlay.go` `helpContent` — verified: there is no
  Search Overlay section in `helpContent` today (the search overlay carries
  its own in-place key strip). Profile Overlay section already reads
  `l Logout / f Forget` and stays unchanged. **No edits required here**;
  the rule #15 trio still applies because the search overlay's own key
  strip (rendered by `renderHelpPanel`) is the third surface and is
  rewritten by this spec's Search Overlay section.

## Files Touched

```
internal/ui/panes/search.go              (~80 lines net delete)
internal/ui/panes/search_test.go         (~40 lines edit)
internal/ui/panes/profile.go             (~60 lines net delete + helper)
internal/ui/panes/profile_test.go        (~30 lines edit)
internal/ui/panes/help_overlay.go        (helpContent string edit)
docs/keybinding.md                       (search section edits)
docs/DESIGN.md                           (§17 edits)
```

## Risks & Mitigations

- **Synthetic key.Binding entries** — used only for KeyBar display, never
  for matching. Risk: future maintainer wires them to handlers and gets
  `tab/shift+tab` as a literal key. Mitigation: comment on the helper
  explaining their display-only purpose.
- **Ctrl+U muscle memory** — users accustomed to clearing via Ctrl+U lose
  that. Mitigated by the requirement itself: the user wants clearing to
  happen only via input mutation. No deprecation period.
- **Test brittleness** — ANSI-aware substring matching is the project pattern
  (see story 90 memory); reuse `stripANSI`.

## Acceptance Criteria

1. `make ci` passes (lint + tests + 80% coverage).
2. Search overlay shows a single-line bottom keybar matching the mockup.
3. Search Results panel has no corner-notch actions.
4. Ctrl+U in search overlay does not clear the query.
5. Profile overlay matches the mockup: 3 icon+value rows, blank spacer,
   single KeyBar line, all inside one border.
6. Profile uses `uikit.GlyphFor` for all three glyphs; ASCII mode produces
   ASCII forms.
7. Help overlay, `docs/keybinding.md`, `docs/DESIGN.md §17` all reflect the
   new search keybindings — no stale `enter play / esc close / ctrl+u clear`
   entries for the search overlay.
8. Double-press confirm for logout/forget still works.
