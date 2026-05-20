---
title: "Search Placeholder Redesign: Colored Pill Prompt + Action Text"
feature: 06-search
status: open
---

## Background

The search overlay placeholder currently cycles through plain strings like `:songs search for tracks...`. The `:prefix` part is visually indistinguishable from the action text, so users don't learn that `:` triggers a category filter. Meanwhile, the prefix hint pills below the input have been removed (story 212), leaving the placeholder as the sole discovery mechanism for prefix syntax.

This story redesigns the placeholder to visually separate the filter command from the action text: the `:prefix` renders as a background pill (same style as a locked-prefix Prompt tag), and the action text renders dim.

## Design

### Placeholder Data

| Prefix      | Action text        |
|-------------|--------------------|
| `:songs`    | `search tracks`    |
| `:artists`  | `find artists`     |
| `:albums`   | `browse albums`    |
| `:playlists`| `discover playlists`|

### Mechanics

- **On placeholder tick** (every 2 s, input empty, no prefix locked):
  - `Prompt` → `buildPromptTag(prefix)` (styled pill with `SelectedBg`/`SelectedFg` + bold + padding)
  - `Placeholder` → action text
  - `PlaceholderStyle` → `Foreground(TextMuted())`
- **On first keystroke** (input transitions empty → non-empty):
  - If no prefix is locked: restore `Prompt` to `"> "`.
  - Placeholder tick stops naturally (tick handler checks `Value() == ""`).
- **On backspace clearing input** (input transitions non-empty → empty):
  - Immediately restore the current cycle's pill Prompt + action Placeholder.
  - Re-arm the placeholder tick.
- **Prefix lock / tab cycle** behavior is unchanged:
  - When a prefix locks via typing or Tab acceptance, `promoteToPromptTag()` sets the Prompt to the locked prefix pill and Placeholder to `"search..."`.
  - When tab-cycling, `syncInputToTab()` swaps the Prompt pill to the new tab's prefix.
- **On overlay open** (`Reset()`):
  - Show the first pill immediately (not after 2-second tick delay).

### Visual Treatment

```
╭─ Search ───────────────────────────────────────────────────────╮
│ [ :songs ] search tracks                                       │
╰────────────────────────────────────────────────────────────────╯
```

The `[ :songs ]` portion uses `SelectedBg()` background + `SelectedFg()` foreground + bold + padding — identical to a locked-prefix Prompt tag. The `search tracks` portion uses `TextMuted()` foreground.

## Acceptance Criteria

- [ ] `searchPlaceholders` is a struct slice with `prefix` and `text` fields
- [ ] `NewSearchOverlay()` sets `Prompt` to the first pill and `Placeholder` to `"search tracks"`
- [ ] `PlaceholderStyle` uses `TextMuted()` (not `Info()`)
- [ ] Placeholder tick sets both `Prompt` (pill) and `Placeholder` (action text)
- [ ] First keystroke on empty input resets `Prompt` from pill to `"> "`
- [ ] Backspace clearing input restores the current cycle's pill Prompt + action Placeholder
- [ ] `Reset()` shows the first pill immediately
- [ ] `syncInputToTab()` restores pill Prompt when cycling back to TabAll with empty input
- [ ] `demoteFromPromptTag()` restores the cycling placeholder text
- [ ] `SetTheme()` updates `PlaceholderStyle` to the new theme's `TextMuted()`
- [ ] Full test suite passes (`make test`)

## Tasks

1. **Extract standalone `buildPromptTag`** — make it callable from `NewSearchOverlay()` before `SearchOverlay` struct exists.
2. **Replace placeholder data with struct slice** — update `search_prefix.go` and `search.go`.
3. **Change `PlaceholderStyle` to `TextMuted()`** — in `NewSearchOverlay()` and `SetTheme()`.
4. **Update placeholder tick handler** — set Prompt pill + Placeholder text.
5. **Update typing handler** — reset Prompt when transitioning empty → non-empty.
6. **Update backspace handler** — restore pill when transitioning non-empty → empty.
7. **Update `Reset()`** — show first pill immediately.
8. **Update `syncInputToTab()`** — handle TabAll with empty input.
9. **Update `demoteFromPromptTag()`** — restore cycling placeholder text.
10. **Update tests** — adjust placeholder assertions to expect struct fields / action text.
