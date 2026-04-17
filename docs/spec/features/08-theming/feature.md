---
title: "Theming & Appearance"
status: done
---

## Description

Token-based color theming with a Theme interface implemented by 11 built-in themes. Original themes (black, dracula-inspired, etc.) established the Theme interface and 16 semantic color tokens. Vivid themes added TOML config-driven loading, always-colorful pane borders, per-column accent colors, and 6 additional vibrant themes (Dracula, Gruvbox, Tokyo Night, Rose Pine, Solarized, Synthwave). The runtime theme switcher overlay (`t` key) lets users preview and select themes; the selection persists on exit via the preference store.

## Acceptance Criteria

- [ ] Theme interface implemented by all 11 built-in themes — no missing methods
- [ ] TOML theme files load correctly at startup; invalid TOML shows error toast
- [ ] Pane borders always display the active theme's border color
- [ ] Runtime switcher (`t`) previews and applies themes without restart
- [ ] Selected theme persists across app restarts via preference store
- [ ] No hardcoded hex values in component code — all colors via Theme tokens
