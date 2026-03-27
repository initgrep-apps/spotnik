---
name: project_spotnik_feature53_complete
description: Feature 53 (Cleanup): dead code removal, FetchStatsMsg rescue, formatDuration relocation, DESIGN_OLD.md removal, CLAUDE.md update
type: project
---

## Feature 53 — Cleanup (Final UI Redesign Feature)

**What was built:**
- Deleted old pane files: library.go, stats.go, playlists.go + their test files
- Deleted netlog_test.go (tested StatsView's old NetLog section; NetworkLogPane has its own tests)
- Deleted unused old components: progress.go, volume.go, errorview.go + their tests
- Moved FetchStatsMsg from deleted stats.go into messages.go (still emitted by TopTracksPane/TopArtistsPane)
- Moved formatDuration helper from deleted progress.go into gradient.go (its only remaining user)
- Updated stale viewGrid comment in app.go
- Deleted docs/DESIGN_OLD.md, updated DESIGN.md references
- Updated CLAUDE.md: added bubble-table/bubbletea-overlay to tech stack, updated layout tree and design rules
- Marked features 40-53 as ✅ Complete in 00-overview.md

**Key files:**
- `internal/ui/panes/messages.go` — FetchStatsMsg now lives here (was in stats.go)
- `internal/ui/components/gradient.go` — formatDuration helper added here (was in progress.go)

**Patterns established:**
- When deleting a file that contains types used elsewhere, always grep for all usages before deleting and relocate types to appropriate files (messages.go for message types, the primary user file for helpers)
- netlog_test.go tested StatsView integration — was properly deleted since NetworkLogPane has its own comprehensive tests in networklog_pane_test.go

**Gotchas:**
- `FetchStatsMsg` was defined in stats.go but also used by TopTracksPane, TopArtistsPane, and many app tests — must be moved to messages.go, not just deleted
- `formatDuration` was in progress.go but also used by gradient.go's GradientSeekBar.Render() — must be moved to gradient.go before deleting progress.go
- gofmt aligns comment spacing in const blocks — after editing a const block comment, gofmt may reformat the alignment of all sibling constants' comments

**Testing notes:**
- Coverage: 86.1% overall (was 86.2% before deletions — essentially stable)
- All packages passed; deleting old test files did not drop coverage below threshold because the new pane implementations (F45-F48) already had thorough test coverage
- PR #58
