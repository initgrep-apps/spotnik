---
title: "End-to-End Test Infrastructure"
status: open
stories: 256–266
---

## Description

Adds two new layers to Spotnik's testing pyramid above the existing ~3456 unit tests:
component golden-file snapshots and integration flow tests. Uses
`github.com/charmbracelet/x/exp/teatest` (official Charmbracelet, same org as Bubble Tea)
for in-process model testing with `tea.WithInput` / `tea.WithOutput` — no PTY, no subprocess.

**Component tests:** Each pane and overlay gets a golden file snapshot of its `View()` output
at fixed terminal dimensions (80×24, 40×24). Golden files stored in `testdata/` catch visual
regressions: layout shifts, border breakage, padding changes, glyph misalignment.

**Integration tests:** Root `app.App` model driven through multi-step sequences with
mock API backends. Verifies cross-pane message routing, overlay lifecycle, focus rotation,
toast notification delivery, and error resilience flows.

**Golden file protocol:** `go test -update` regenerates all golden files. Committed golden
files are the source of truth for visual output. PRs that intentionally change output must
include regenerated golden files with diff review.

## Stories

| Story | Title | Status |
|-------|-------|--------|
| 256 | teatest setup + QueuePane golden POC | done |
| 257 | Library component + integration (Playlists, Albums, LikedSongs) | done |
| 258 | Stats component + integration (TopTracks, TopArtists, RecentlyPlayed) | done |
| 259 | NowPlaying component + playback integration | done |
| 260 | Search component + integration | done |
| 261 | Podcast component + integration (FollowedShows, SavedEpisodes, EpisodeDetails) | open |
| 262 | Overlays component (Theme, Help, Profile, Devices) | open |
| 263 | DevTools panes component (GatewayHealth, PollingTraffic, GatewayLive, NetworkLog) | open |
| 264 | Cross-cutting integration flows (navigation, overlay lifecycle, error resilience) | open |
| 265 | CI enforcement + docs | open |
| 266 | Onboarding + Splash golden tests | open |

## Acceptance Criteria

- [ ] `github.com/charmbracelet/x/exp/teatest` added to `go.mod`
- [ ] Golden helper package `internal/goldentest/` provides `AssertOutput(t, tm)` and `AssertGolden(t, got, name)`
- [ ] Every pane has a `*_golden_test.go` file with snapshots at 80×24 and 40×24 dimensions
- [ ] Every overlay has a `*_golden_test.go` file with snapshots
- [ ] Integration flow tests exist for playback, search, navigation, overlay lifecycle, and error resilience
- [ ] `make ci` runs golden tests; `go test -update` regenerates golden files
- [ ] Golden files committed to `testdata/` directories alongside test files
- [ ] `AGENTS.md` Reading Order references golden test protocol
- [ ] `docs/system/sanity-tests.md` updated with golden test references
- [ ] `make ci` passes (all tests including golden tests)

## Dependencies

- `github.com/charmbracelet/x/exp/teatest` — official Charmbracelet TUI testing library (v1 API for Bubble Tea v1)
- No other new dependencies
