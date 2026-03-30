---
title: "Theme System"
status: done
---

## Description
Provides a token-based color theming infrastructure with five built-in themes (True Black, Monokai, Catppuccin, Nord, Light) so every UI component renders with consistent, configurable colors without hardcoding hex values. The Theme interface is the foundation that every UI component depends on.

The initial theme system established the core infrastructure: a `Theme` interface with 23 color token methods, a registry/loader that maps config IDs to concrete implementations, five built-in themes, and the startup wiring that injects the active theme into every pane constructor. Components call theme methods in `View()` and never store raw hex strings.

As the UI evolved toward a btop-inspired redesign with gradient bars, an audio visualizer, dense table panes, and per-pane colored borders, 16 additional color tokens were needed. These tokens -- for gradients, visualizer foreground, table headers, preset indicators, and 10 per-pane border accents -- were added to the Theme interface and implemented across all five themes, bringing the total to 42 methods.

## Acceptance Criteria
- [ ] All five themes compile and implement the full `Theme` interface (42 methods each)
- [ ] `Load()` returns the correct concrete theme for every known ID
- [ ] Unknown theme IDs never panic -- `Load()` always falls back to the default
- [ ] `DefaultThemeID` is `"black"` and can never be empty
- [ ] No component file contains a raw hex colour string -- all colour comes from `Theme` methods
- [ ] `theme = "monokai"` in config.toml results in Monokai colours being used throughout the UI
- [ ] Theme is injected at startup and passed to all pane constructors -- panes never call `Load()` themselves
- [ ] 100% test coverage on `theme.go` (registry, load, fallback)
- [ ] `make ci` passes (lint + tests + coverage)
