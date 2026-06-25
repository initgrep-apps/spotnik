---
title: "Theming & Appearance"
status: done
stories: 01, 40, 70–75, 77–79, 207, 208
---

## Description

Token-based color theming with a Theme interface implemented by 11 built-in themes. Original themes (black, dracula-inspired, etc.) established the Theme interface and 16 semantic color tokens. Vivid themes added TOML config-driven loading, always-colorful pane borders, per-column accent colors, and 6 additional vibrant themes (Dracula, Gruvbox, Tokyo Night, Rose Pine, Solarized, Synthwave). The runtime theme switcher overlay (`t` key) lets users preview and select themes; the selection persists on exit via the preference store.

**Mono themes (stories 207–208, absorbed from feature 16):** Added `mono-dark` and `mono-light` themes — uniform grayscale palettes picked up automatically by ConfigTheme. Also renamed Page A/B → Music/Stats across all identifiers, comments, tests, and docs.

## Stories

| # | Story | File |
|---|---|---|
| 01 | Base theme interface + black theme | `stories/01-base-theme-system.md` |
| 40 | Vivid themes + TOML loading | `stories/40-vivid-themes.md` |
| 70 | Theme interface additions | `stories/70-theme-interface-additions.md` |
| 71 | Theme switcher overlay | `stories/71-theme-switcher-overlay.md` |
| 72 | Theme persistence | `stories/72-theme-persistence.md` |
| 73 | Runtime theme preview | `stories/73-theme-switcher-overlay.md` |
| 74 | Per-column accent colors | `stories/74-per-column-accent-colors.md` |
| 75 | Always-colorful pane borders | `stories/75-always-colorful-pane-borders.md` |
| 77 | Overlay border + status bar fixes | `stories/77-overlay-border-statusbar-fixes.md` |
| 78 | Theme cleanup | `stories/78-theme-cleanup.md` |
| 79 | Preference store engine | `stories/79-preference-store-engine.md` |
| 207 | Rename Page A/B → Music/Stats | `stories/207-page-music-stats-rename.md` |
| 208 | Add mono-dark and mono-light themes | `stories/208-mono-themes.md` |

## Acceptance Criteria

- [ ] Theme interface implemented by all 11 built-in themes — no missing methods
- [ ] TOML theme files load correctly at startup; invalid TOML shows error toast
- [ ] Pane borders always display the active theme's border color
- [ ] Runtime switcher (`t`) previews and applies themes without restart
- [ ] Selected theme persists across app restarts via preference store
- [ ] No hardcoded hex values in component code — all colors via Theme tokens
- [ ] `mono-dark.toml` and `mono-light.toml` are valid themes loaded by `theme.Available()`
- [ ] Page labels render "Music" / "Stats" (not "A" / "B")
