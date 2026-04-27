---
name: project_spotnik_feature53_complete
description: Feature 53 (Cleanup): dead code removal, FetchStatsMsg rescue, formatDuration relocation, DESIGN_OLD.md removal, CLAUDE.md update
type: project
---

## Feature 53 — Cleanup (Final UI Redesign Feature)

**Built:**
- Deleted old pane files: library.go, stats.go, playlists.go + test files
- Deleted netlog_test.go (tested StatsView old NetLog section; NetworkLogPane has own tests)
- Deleted unused old components: progress.go, volume.go, errorview.go + tests
- Moved FetchStatsMsg from deleted stats.go to messages.go (still emitted by TopTracksPane/TopArtistsPane)
- Moved formatDuration helper from deleted progress.go to gradient.go (only remaining user)
- Updated stale viewGrid comment in app.go
- Deleted docs/DESIGN_OLD.md, updated DESIGN.md refs
- Updated CLAUDE.md: added bubble-table/bubbletea-overlay to tech stack, updated layout tree + design rules
- Marked features 40-53 ✅ Complete in 00-overview.md

**Key files:**
- `internal/ui/panes/messages.go` — FetchStatsMsg lives here (was stats.go)
- `internal/ui/components/gradient.go` — formatDuration helper added here (was progress.go)

**Patterns:**
- Deleting file with types used elsewhere: grep all usages first, relocate types (messages.go for message types, primary user file for helpers)
- netlog_test.go tested StatsView integration — properly deleted; NetworkLogPane has comprehensive tests in networklog_pane_test.go

**Gotchas:**
- `FetchStatsMsg` defined in stats.go but used by TopTracksPane, TopArtistsPane, many app tests — move to messages.go, don't delete
- `formatDuration` in progress.go but used by gradient.go GradientSeekBar.Render() — move to gradient.go before deleting progress.go
- gofmt aligns comment spacing in const blocks — editing const block comment may reformat sibling constant comment alignment

**Testing:**
- Coverage: 86.1% overall (was 86.2% pre-deletion — stable)
- All packages passed; new pane implementations (F45-F48) already had thorough coverage, held threshold
- PR #58