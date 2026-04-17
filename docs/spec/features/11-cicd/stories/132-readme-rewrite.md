---
title: "README Rewrite"
feature: 11-cicd
status: open
---

## Background
`README.md` is a 2-line placeholder. This story replaces it with a complete project README
covering all installation channels, prerequisites, quick start, keybindings, configuration,
building from source, contributing, and license.

## Design

### Sections

1. **Header** — project name, one-line description
2. **Badges** — CI status, latest release, Go version, license
3. **Screenshot** — placeholder (terminal GIF to be added later)
4. **Installation** — 6 channels:
   - Homebrew: `brew install initgrep-apps/tap/spotnik`
   - Scoop: `scoop bucket add spotnik https://github.com/initgrep-apps/scoop-bucket && scoop install spotnik`
   - DEB: wget + `sudo dpkg -i spotnik_<version>_linux_amd64.deb`
   - RPM: wget + `sudo rpm -i spotnik_<version>_linux_amd64.rpm`
   - Go install: `go install github.com/initgrep-apps/spotnik@latest`
   - Binary: link to GitHub Releases page
5. **Prerequisites** — Spotify Premium account, Spotify Developer app with redirect URI for PKCE
6. **Quick Start** — run `spotnik`, auth flow opens browser, start playing
7. **Keybindings** — subset: Tab/Shift+Tab, Space, n/p, +/-, /, a, d, q/Ctrl+C; link to `docs/DESIGN.md` for full table
8. **Configuration** — `~/.config/spotnik/config.toml`, key options: `theme`, default device
9. **Building from Source** — Go version from `go.mod`; `git clone`, `make build`, `./bin/spotnik`
10. **Contributing** — Conventional Commits required, `make ci` before push, one feature per branch
11. **License** — MIT

## Acceptance Criteria
- [ ] README covers all 6 installation channels with copy-paste commands
- [ ] Prerequisites section present
- [ ] Keybindings subset present with link to full table in `docs/DESIGN.md`
- [ ] Contributing section present with Conventional Commits requirement
- [ ] CI and release badges present

## Tasks
- [ ] Rewrite `README.md` with all 11 sections
      - test: README contains "Homebrew", "Prerequisites", "Contributing", "Keybindings" headings
