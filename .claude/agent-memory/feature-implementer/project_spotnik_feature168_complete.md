---
name: project_spotnik_feature168_complete
description: Story 168 (TUI Design System docs rewrite): TUI-DESIGN-SYSTEM.md creation, DESIGN.md primitive stripping, PANE-TEMPLATE.md + CLAUDE.md updates, struct accuracy pitfalls
type: project
---

## Story 168 — TUI Design System Docs Rewrite (feature 13, final story)

**What was built:**
- Created `docs/TUI-DESIGN-SYSTEM.md` — 7-section canonical reference (~1000 lines) with all 18 primitive contracts (6-block format: Purpose, Fields, Rendering, Roles, Glyphs, Lifecycle, Tests), verbatim glyph catalogue from design record §5, role/colour matrix from §6, 6 feedback surfaces, relationship to other docs
- Stripped `docs/DESIGN.md` §5 (Embedded Shortcut Borders) entirely; §12–§15 (Notifications, Search Overlay, Device Switcher, Header & Status Bar) reduced to pointers; added §0 authority paragraph
- Removed all `ᐅ` from `docs/DESIGN.md`: preset diagrams in §4 rewritten to notch format, §11 Unicode note deleted, §19 Network Log diagram updated
- Replaced `⚠` with `◬` in §19
- Updated `docs/PANE-TEMPLATE.md` Step 2 `View()` scaffold to use `uikit.PaneChrome` struct literal; added `strconv` + `uikit` imports
- Added Reading Order + NEVER-Do rule #17 to `CLAUDE.md`
- Marked feature 13 `done` in `docs/spec/00-overview.md`

**Key files:**
- `docs/TUI-DESIGN-SYSTEM.md` — new canonical reference (created)
- `docs/DESIGN.md` — stripped primitive sections, added §0
- `docs/PANE-TEMPLATE.md` — Step 2 scaffold updated
- `CLAUDE.md` — Reading Order + rule #17 added

**Gotchas — Struct accuracy (critical for doc-only stories):**
- `ListRow` does NOT have `Width` or `Focused` fields — `Render(width int)` takes width as a parameter. The actual extra field is `RowBackground lipgloss.TerminalColor` (for cursor-highlight continuity). If documenting struct fields, always grep the actual Go file.
- `HeaderBar.Page` is `"A"` or `"B"` (short form), NOT `"Page A"` / `"Page B"`. Always verify string literal values in the Go source.
- `HeaderBar.Preset` uses `-1` sentinel to hide the preset segment (Page B); doc should note this, not just say "preset index (0-based)".
- The design record's struct literals in §7.3–§7.5 describe the design intent, not the final implementation — actual Go source may differ. Always check the actual `.go` files.

**ᐅ in retained sections:**
- §4 preset diagrams (Preset 0, Preset 1) had `ᐅs shfl ─ ᐅr rpt` etc. in the action ribbons — these are RETAINED sections but still needed `ᐅ` cleaned. Replaced with notch format `╮ s shfl ╭─╮ r rpt ╭` etc.
- §19 Network Log diagram had `ᐅf filter` in the border — replaced with `╮ f filter ╮`
- §19 also had `⚠` in the 429 row marker text — replaced with `◬`
- grep ALL sections of DESIGN.md for ᐅ before declaring done; don't just strip the removed sections

**TableChrome location:**
- Story 157 placed TableChrome in `internal/ui/components/table_chrome.go` (not `internal/uikit/`). The doc notes the actual location and that the struct wraps `components/table.go`.

**CI for doc-only stories:**
- `make ci` may fail in sandbox due to Go build cache permissions (`/Library/Caches/go-build`) — retry with `dangerouslyDisableSandbox: true`. Doc-only changes always pass once the sandbox is lifted.

**Section pointer accuracy:**
- DESIGN.md pointer sections reference `docs/TUI-DESIGN-SYSTEM.md §3.N` — verify section numbers match actual headings in the new doc before committing.
