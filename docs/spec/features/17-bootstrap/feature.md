---
title: "First-Launch Bootstrap & Embedded Client ID"
status: open
---

## Description

Spotnik should work out of the box on first launch without requiring manual config
file creation. The Spotify client ID ships embedded in the binary via ldflags, and
a preferences file is created automatically on first run with sensible defaults
and a commented placeholder for users who want to use their own Spotify app credentials.

## Goals

- Zero-config first launch — `spotnik` just works after install
- Client ID embedded in binary, no user setup required
- Config file created on first launch as self-documenting preferences file
- Config client_id overrides embedded value for power users
- Theme preference and future preferences persist across launches

## Acceptance Criteria

- [ ] App launches without any config file present
- [ ] First launch creates `~/.config/spotnik/config.toml` with template
- [ ] Template includes commented client_id placeholder with instructions
- [ ] Template includes default preferences (theme, volume_step)
- [ ] Embedded client_id is used when config has none
- [ ] Config client_id overrides embedded when present
- [ ] `PersistTheme()` continues to work with bootstrapped config
- [ ] `make ci` passes
