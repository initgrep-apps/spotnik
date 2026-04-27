---
name: project_spotnik_feature58b_complete
description: Feature 58b (NowPlaying Design Docs): DESIGN.md updates for split layout, animation patterns, preset diagrams
type: project
---

## Feature 58b — Update DESIGN.md with NowPlaying Split Layout

**Built:**
- Docs-only feature — `docs/DESIGN.md` only, no code
- §2 Key Notes: added NowPlaying split layout bullet
- §4 Presets: Preset 0/1 diagrams show InfoBox+Visualizer side-by-side; Presets 2/3/Page B "compact strip" → "small strip (height < 8)" matching impl
- §11 Visualizer: added Animation patterns block (3 patterns, v-key cycling)
- §11: new "NowPlaying Split Layout (btop-inspired)" subsection — layout proportions, responsive behavior

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/docs/DESIGN.md` — all changes

**Patterns:**
- Docs-only features still need `make ci` (passes trivially, no code)
- Edit tool silently fails on big files — verify w/ grep post-edit
- Animation patterns edit silently skipped first attempt (linter touched file between read/edit). Re-read + verify after edits to large docs.

**Gotchas:**
- Edit tool on big files (DESIGN.md ~1000 lines) fails w/ "File has been modified since read" — linter/formatter touches file between Read/Edit. Re-read before retry.
- Chained edits: second may fail (first edit modified file). Re-read section before each edit.
- Animation patterns block silently skipped first commit → needed second commit. `grep` verify before commit.

**Testing:**
- No code tests — docs only
- CI passes 86.1% coverage (unchanged)
- PR review zero issues — docs matched code