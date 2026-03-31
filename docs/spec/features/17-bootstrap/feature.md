---
title: "First-Launch Bootstrap & Preference Persistence Engine"
status: open
---

## Description

Spotnik should work out of the box on first launch without requiring manual config
file creation. The Spotify client ID ships embedded in the binary via ldflags, and
a preferences file is created automatically on first run with sensible defaults
and a commented placeholder for users who want to use their own Spotify app credentials.

Beyond bootstrapping, this feature introduces a **PreferenceStore engine** — an async,
coalescing preference writer that plugs into the Bubble Tea event loop. Instead of
one-off `PersistTheme()` calls, all runtime preference changes (theme, layout preset,
visualizer pattern) flow through a single engine that debounces rapid changes and
writes them to disk in a single flush. The engine uses a generation-counter pattern
to ensure only the latest change triggers a disk write, with validation/clamping on
load to handle invalid values from manual config edits.

## Goals

- Zero-config first launch — `spotnik` just works after install
- Client ID embedded in binary, no user setup required
- Config file created on first launch as self-documenting preferences file
- Config client_id overrides embedded value for power users
- Async preference persistence via coalescing engine (theme, preset, visualizer)
- Invalid preference values clamped to valid range on load — never crash

## Acceptance Criteria

- [ ] App launches without any config file present
- [ ] First launch creates `~/.config/spotnik/config.toml` with template
- [ ] Template includes commented client_id placeholder with instructions
- [ ] Template includes default preferences (theme, volume_step, preset, visualizer)
- [ ] Embedded client_id is used when config has none
- [ ] Config client_id overrides embedded when present
- [ ] PreferenceStore replaces PersistTheme — single engine for all preferences
- [ ] Preference changes debounce at 500ms — rapid cycling causes one disk write
- [ ] Theme, preset, and visualizer selections persist across app restarts
- [ ] Invalid config values (e.g. preset=99) are clamped to valid range on load
- [ ] `make ci` passes
