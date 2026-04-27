---
name: project_spotnik_feature168_complete
description: Story 168 (TUI Design System docs rewrite): TUI-DESIGN-SYSTEM.md creation, DESIGN.md primitive stripping, PANE-TEMPLATE.md + CLAUDE.md updates, struct accuracy pitfalls
type: project
---

## Story 168 — TUI Design System Docs Rewrite (feature 13, final story)

**What was built:**
- Created `docs/TUI-DESIGN-SYSTEM.md` — 7-section canonical ref (~1000 lines), all 18 primitive contracts (6-block: Purpose, Fields, Rendering, Roles, Glyphs, Lifecycle, Tests), verbatim glyph catalogue from design record §5, role/colour matrix from §6, 6 feedback surfaces, rel to other docs
- Stripped `docs/DESIGN.md` §5 (Embedded Shortcut Borders) entire; §12–§15 (Notifications, Search Overlay, Device Switcher, Header & Status Bar) reduced to pointers; added §0 authority para
- Removed all `ᐅ` from `docs/DESIGN.md`: §4 preset diagrams rewritten to notch format, §11 Unicode note deleted, §19 Network Log diagram updated
- Replaced `⚠` with `◬` in §19
- Updated `docs/PANE-TEMPLATE.md` Step 2 `View()` scaffold use `uikit.PaneChrome` struct literal; added `strconv` + `uikit` imports
- Added Reading Order + NEVER-Do rule #17 to `CLAUDE.md`
- Marked feature 13 `done` in `docs/spec/00-overview.md`

**Key files:**
- `docs/TUI-DESIGN-SYSTEM.md` — new canonical ref (created)
- `docs/DESIGN.md` — stripped primitive sections, added §0
- `docs/PANE-TEMPLATE.md` — Step 2 scaffold updated
- `CLAUDE.md` — Reading Order + rule #17 added

**Gotchas — Struct accuracy (critical for doc-only stories):**
- `ListRow` no `Width` or `Focused` fields — `Render(width int)` takes width as param. Actual extra field: `RowBackground lipgloss.TerminalColor` (cursor-highlight continuity). Doc struct fields → grep actual Go file.
- `HeaderBar.Page` is `"A"` or `"B"` (short), NOT `"Page A"` / `"Page B"`. Verify string literals in Go source.
- `HeaderBar.Preset` uses `-1` sentinel to hide preset segment (Page B); doc must note, not just say "preset index (0-based)".
- Design record struct literals §7.3–§7.5 describe intent, not final impl — actual Go source may differ. Check actual `.go` files.

**ᐅ in retained sections:**
- §4 preset diagrams (Preset 0, Preset 1) had `ᐅs shfl ─ ᐅr rpt` etc. in action ribbons — RETAINED sections but `ᐅ` still needed cleaning. Replaced with notch format `╮ s shfl ╭─╮ r rpt ╭` etc.
- §19 Network Log diagram had `ᐅf filter` in border — replaced with `╮ f filter ╮`
- §19 also had `⚠` in 429 row marker text — replaced with `◬`
- grep ALL sections of DESIGN.md for ᐅ before declaring done; don't just strip removed sections

**TableChrome location:**
- Story 157 placed TableChrome in `internal/ui/components/table_chrome.go` (not `internal/uikit/`). Doc notes actual location + struct wraps `components/table.go`.

**CI for doc-only stories:**
- `make ci` may fail in sandbox due to Go build cache perms (`/Library/Caches/go-build`) — retry with `dangerouslyDisableSandbox: true`. Doc-only changes pass once sandbox lifted.

**Section pointer accuracy:**
- DESIGN.md pointer sections reference `docs/TUI-DESIGN-SYSTEM.md §3.N` — verify section numbers match actual headings in new doc before commit.