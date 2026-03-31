---
title: "Vivid Theme System"
status: done
---

## Description
Replaces the hardcoded Go-struct theme system with a config-driven architecture where each theme is a TOML file. Built-in themes ship embedded in the binary; users can drop new `.toml` files into `~/.config/spotnik/themes/` for plug-and-play custom themes. Adds always-colorful pane borders (no more grey/faint unfocused panes), per-column table color tokens, 6 new vibrant themes (Dracula, Gruvbox Dark, Tokyo Night, Rose Pine, Solarized Dark, Synthwave '84), and a runtime theme switcher overlay toggled with `t`.

## Goals
- Every pane border is colorful at all times -- focused panes are bright, unfocused are dimmed but still show their accent color
- Table columns have distinct, vibrant colors per theme -- not just white/grey text
- Themes are TOML config files -- adding a new theme requires zero Go code changes
- Users can switch themes at runtime via an overlay, with the selection persisted to config.toml
- Ship 11 themes total (5 existing reworked + 6 new)

## Acceptance Criteria
- [ ] All theme Go struct files (`black.go`, `monokai.go`, etc.) are replaced by embedded TOML files
- [ ] A single `ConfigTheme` struct implements the `Theme` interface by loading values from TOML
- [ ] Built-in themes load from `//go:embed themes/*.toml`; user themes load from `~/.config/spotnik/themes/*.toml`
- [ ] User themes with the same ID override built-in themes
- [ ] Unknown theme IDs fall back to `DefaultThemeID` ("black")
- [ ] 4 new column color tokens (`ColumnIndex`, `ColumnPrimary`, `ColumnSecondary`, `ColumnTertiary`) exist in every theme
- [ ] All pane tables use column color tokens instead of `TextMuted`/`TextPrimary`/`TextSecondary`
- [ ] Unfocused pane borders show their per-pane accent color (dimmed), not `Faint(true)` grey
- [ ] Focused pane borders show full accent color with bold title
- [ ] 6 new themes ship as embedded TOML files: Dracula, Gruvbox Dark, Tokyo Night, Rose Pine, Solarized Dark, Synthwave '84
- [ ] `t` key opens the theme switcher overlay; Enter applies; Esc closes
- [ ] Theme changes take effect immediately without restart
- [ ] Selected theme is persisted to `~/.config/spotnik/config.toml`
- [ ] `make ci` passes (lint + tests + 80% coverage)
