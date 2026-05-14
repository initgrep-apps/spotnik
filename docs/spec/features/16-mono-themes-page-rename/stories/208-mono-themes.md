---
title: "Add mono-dark and mono-light themes"
feature: 16-mono-themes-page-rename
status: open
---

## Background

Two new monochrome themes that strip all per-pane colour differentiation. Every accent becomes grayscale. Useful for users who want maximum visual minimalism or accessibility-friendly low-contrast.

**Depends on:** Feature 08 (Theming) — done. `ConfigTheme` + `Load()` already support embedded TOML files with no code changes.

**Scope:** Two new TOML files only. Zero Go code changes.

---

## Design

### `mono-dark.toml` — Light on Black

| Token | Value | Role |
|---|---|---|
| `base` | `#000000` | Canvas |
| `surface` | `#0a0a0a` | Pane interior |
| `surface_alt` | `#141414` | Overlay bg |
| `active_border` | `#cccccc` | Focused pane border |
| `inactive_border` | `#222222` | Unfocused border |
| `text_primary` | `#e0e0e0` | Body text |
| `text_secondary` | `#a0a0a0` | Subtitles |
| `text_muted` | `#666666` | Timestamps |
| `selected_bg` | `#222222` | Selected row bg |
| `selected_fg` | `#ffffff` | Selected row text |
| `section_header` | `#999999` | Section labels |
| `playing_indicator` | `#ffffff` | Playing glyph |
| `seek_bar` | `#aaaaaa` | Seek fill |
| `volume_bar` | `#aaaaaa` | Volume fill |
| `success` | `#aaaaaa` | Success (mono) |
| `warning` | `#888888` | Warning (mono) |
| `error` | `#666666` | Error (mono) |
| `info` | `#999999` | Info (mono) |
| `header_chip_fg` | `#cccccc` | Chip text |
| `status_bar_bg` | `#000000` | Status bar bg |
| `status_bar_fg` | `#666666` | Status bar text |
| `key_hint` | `#999999` | Key labels |
| `gradient1` | `#444444` | Low band |
| `gradient2` | `#888888` | Mid band |
| `gradient3` | `#aaaaaa` | High band |
| `visualizer_fg` | `#aaaaaa` | Braille dots |
| `table_header` | `#666666` | Column headers |
| `preset_indicator` | `#999999` | Preset label |
| `column_index` | `#555555` | # column |
| `column_primary` | `#e0e0e0` | Main data |
| `column_secondary` | `#a0a0a0` | Supporting |
| `column_tertiary` | `#666666` | Metadata |
| `accent` | `#aaaaaa` | CLI accent |

All `pane_borders` entries: `#808080` (single neutral gray for every pane).

### `mono-light.toml` — Black on Light

| Token | Value | Role |
|---|---|---|
| `base` | `#ffffff` | Canvas |
| `surface` | `#f5f5f5` | Pane interior |
| `surface_alt` | `#eeeeee` | Overlay bg |
| `active_border` | `#333333` | Focused border |
| `inactive_border` | `#dddddd` | Unfocused border |
| `text_primary` | `#111111` | Body text |
| `text_secondary` | `#444444` | Subtitles |
| `text_muted` | `#888888` | Timestamps |
| `selected_bg` | `#eeeeee` | Selected row bg |
| `selected_fg` | `#000000` | Selected row text |
| `section_header` | `#555555` | Section labels |
| `playing_indicator` | `#000000` | Playing glyph |
| `seek_bar` | `#555555` | Seek fill |
| `volume_bar` | `#555555` | Volume fill |
| `success` | `#555555` | Success |
| `warning` | `#777777` | Warning |
| `error` | `#999999` | Error |
| `info` | `#555555` | Info |
| `header_chip_fg` | `#333333` | Chip text |
| `status_bar_bg` | `#ffffff` | Status bar bg |
| `status_bar_fg` | `#888888` | Status bar text |
| `key_hint` | `#555555` | Key labels |
| `gradient1` | `#cccccc` | Low band |
| `gradient2` | `#888888` | Mid band |
| `gradient3` | `#555555` | High band |
| `visualizer_fg` | `#555555` | Braille dots |
| `table_header` | `#888888` | Column headers |
| `preset_indicator` | `#555555` | Preset label |
| `column_index` | `#999999` | # column |
| `column_primary` | `#111111` | Main data |
| `column_secondary` | `#444444` | Supporting |
| `column_tertiary` | `#888888` | Metadata |
| `accent` | `#555555` | CLI accent |

All `pane_borders` entries: `#999999`.

### IDs and Names

| ID | Display Name |
|---|---|
| `mono-dark` | `Mono Dark` |
| `mono-light` | `Mono Light` |

Both appear in `theme.Available()` automatically after the TOML files are embedded.

---

## Tasks

### Task 1: Create `mono-dark.toml`

**File:** `internal/ui/theme/themes/mono-dark.toml`

Write TOML with all tokens listed above. Ensure `id = "mono-dark"`, `name = "Mono Dark"`. All `pane_borders` keys present and set to `#808080`.

### Task 2: Create `mono-light.toml`

**File:** `internal/ui/theme/themes/mono-light.toml`

Write TOML with all tokens listed above. Ensure `id = "mono-light"`, `name = "Mono Light"`. All `pane_borders` keys present and set to `#999999`.

### Task 3: Verify loading

```bash
rtk go test ./internal/ui/theme/... -v -run "TestAvailable"
```

Expected: `mono-dark` and `mono-light` appear in `Available()` output.

### Task 4: Run full CI

```bash
rtk make ci
```

Expected: passes (no Go code changes → no coverage impact).

---

## Acceptance Criteria

- [ ] `mono-dark.toml` exists and is valid TOML
- [ ] `mono-light.toml` exists and is valid TOML
- [ ] Both themes appear in `theme.Available()`
- [ ] Both themes load without error via `theme.Load("mono-dark")` and `theme.Load("mono-light")`
- [ ] All `pane_borders` entries are the same neutral gray within each theme
- [ ] `make ci` passes
- [ ] No Go code changes required
