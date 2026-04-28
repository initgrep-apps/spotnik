# Overlay Keybinding Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove keybinding duplication and dead bindings from the search and profile overlays; unify both on the `uikit.KeyBar` renderer; restructure the profile overlay into an icon+value table with an inline action keybar.

**Architecture:** Two overlays, both in `internal/ui/panes/`. The search overlay's third panel ("Keys") swaps from a `bubbles/help` 4×2 grid to a single-line `uikit.KeyBar`; its `searchKeyMap` shrinks from 8 to 5 bindings (Play/Close/Clear deleted), and the Ctrl+U handler is removed entirely. The Results panel drops its corner-notch Actions. The profile overlay swaps its bold-name-plus-separators-plus-stacked-action-rows layout for three glyph+value rows + a single `uikit.KeyBar` line, all inside the existing border, and migrates hardcoded `♛`/`◎` runes to `uikit.GlyphFor(role, ActiveMode())`.

**Tech Stack:** Go 1.22+, Bubble Tea, lipgloss, `internal/uikit` primitives (`KeyBar`, `GlyphFor`), `bubbles/key`. Tests use `testify` and the `stripANSI` helper already present in `internal/ui/panes/search_delegate_test.go`.

**Spec:** `docs/superpowers/specs/2026-04-28-overlay-keybinding-cleanup-design.md`

---

## File Map

**Modify:**
- `internal/ui/panes/search.go` — keymap shrink, Ctrl+U handler delete, Keys panel render swap, Results panel Actions removal, panelHeights helpH=3, SetSize cleanup of `o.help.Width`
- `internal/ui/panes/search_test.go` — rewrite Ctrl+U tests, update help-bar assertions, add height assertion
- `internal/ui/panes/profile.go` — table render, glyph migration, KeyBar action line, drop separators + ListRow
- `internal/ui/panes/profile_test.go` — update action-line assertions, add table-row + glyph-mode tests
- `docs/keybinding.md` — drop Search section rows for Enter/Ctrl+U/Esc
- `docs/DESIGN.md` — §17 same edits

**No changes needed:**
- `internal/ui/panes/help_overlay.go` — has no Search section; Profile section already says `l Logout` / `f Forget`

---

## Task 1: Remove Ctrl+U handler + clear test (search overlay)

**Files:**
- Modify: `internal/ui/panes/search.go:620-646` (delete `case tea.KeyCtrlU:` block)
- Modify: `internal/ui/panes/search.go:471` (drop the comment that references it)
- Modify: `internal/ui/panes/search_test.go:384-401` (rewrite `TestSearchOverlay_Update_CtrlU`)
- Modify: `internal/ui/panes/search_test.go:567-617` (delete `TestSearchOverlay_CtrlU_EmitsSearchClearedMsg` and `TestSearchOverlay_CtrlU_ClearsLocalInput`)

- [ ] **Step 1.1: Rewrite the Ctrl+U test as a no-op assertion**

In `internal/ui/panes/search_test.go`, replace the body of `TestSearchOverlay_Update_CtrlU` (currently lines 384-401):

```go
// TestSearchOverlay_Update_CtrlU verifies Ctrl+U is a no-op:
// the input keeps its value (the textinput swallows Ctrl+U without effect).
// Per the 2026-04-28 overlay-keybinding-cleanup spec, clearing only happens
// when the user mutates the input directly.
func TestSearchOverlay_Update_CtrlU(t *testing.T) {
	o := newTestSearchOverlay()

	o, _ = sendKey(t, o, "h")
	o, _ = sendKey(t, o, "e")
	o, _ = sendKey(t, o, "l")
	o, _ = sendKey(t, o, "l")
	o, _ = sendKey(t, o, "o")
	require.Contains(t, o.Query(), "hello", "query should be 'hello' after typing those chars")

	o, _ = sendKey(t, o, "ctrl+u")
	assert.Equal(t, "hello", o.Query(), "Ctrl+U must not clear the input — clearing only via direct edits")
}
```

Delete the two follow-on tests entirely (`TestSearchOverlay_CtrlU_EmitsSearchClearedMsg` and `TestSearchOverlay_CtrlU_ClearsLocalInput`). They assert the deleted behavior.

- [ ] **Step 1.2: Run the test to verify it fails**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -run TestSearchOverlay_Update_CtrlU -v`
Expected: FAIL — `o.Query()` returns `""` because the existing handler clears it; assertion expects `"hello"`.

- [ ] **Step 1.3: Delete the Ctrl+U handler in search.go**

In `internal/ui/panes/search.go`, delete lines 620-646 (the entire `case tea.KeyCtrlU:` block including its comment) so the `switch` falls through to `default` for Ctrl+U.

Also edit `internal/ui/panes/search.go:471`. Read the current comment and remove the phrase that references "handleKey(KeyCtrlU)" since that handler no longer exists. The replacement should explain that re-arming happens when the user clears the input by editing.

- [ ] **Step 1.4: Run the test to verify it passes**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -run TestSearchOverlay_Update_CtrlU -v`
Expected: PASS.

Then run the broader package tests:
Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -count=1`
Expected: PASS (other Ctrl+U-related tests are deleted, no other tests reference the removed handler).

- [ ] **Step 1.5: Commit**

```bash
cd /Users/irshadsheikh/dev/github/apps/spotnik
git add internal/ui/panes/search.go internal/ui/panes/search_test.go
git commit -m "refactor(search): remove Ctrl+U clear handler

Ctrl+U is no longer wired in the search overlay. Per the 2026-04-28
overlay-keybinding-cleanup spec, clearing only happens via direct input
edits — Ctrl+U is no longer a documented or supported shortcut."
```

---

## Task 2: Shrink searchKeyMap to 5 bindings + drop bubbles/help

**Files:**
- Modify: `internal/ui/panes/search.go:91-127` (KeyMap struct, ShortHelp, FullHelp)
- Modify: `internal/ui/panes/search.go:129-168` (NewSearchKeyMap)
- Modify: `internal/ui/panes/search.go:182-189` (SearchOverlay struct fields)
- Modify: `internal/ui/panes/search.go:254-264` (NewSearchOverlay help model setup)
- Modify: `internal/ui/panes/search.go:281-289` (struct literal in NewSearchOverlay)
- Modify: `internal/ui/panes/search.go:402-411` (SetSize — remove `o.help.Width = ...`)
- Modify: `internal/ui/panes/search.go` import block (remove `"github.com/charmbracelet/bubbles/help"` if no other use)

- [ ] **Step 2.1: Write a failing keymap test**

Append to `internal/ui/panes/search_test.go`:

```go
// TestSearchKeyMap_OnlyVisibleBindings verifies the keymap exposes exactly
// the 5 bindings advertised in the bottom keybar after the cleanup:
// Queue, TabNext, TabPrev, nextPage, prevPage. Play/Close/Clear are removed
// from the map (Enter/Esc behavior remains in Update() but is not advertised;
// Ctrl+U clear is gone entirely).
func TestSearchKeyMap_OnlyVisibleBindings(t *testing.T) {
	km := panes.NewSearchKeyMap()

	// Visible bindings: Queue (ctrl+a), TabNext (tab), TabPrev (shift+tab),
	// nextPage (pgdown), prevPage (pgup). Tab help text reads "category".
	assert.Equal(t, []string{"ctrl+a"}, km.Queue.Keys(), "Queue → ctrl+a")
	assert.Equal(t, "ctrl+a", km.Queue.Help().Key)
	assert.Equal(t, "queue", km.Queue.Help().Desc)

	assert.Equal(t, []string{"tab"}, km.TabNext.Keys())
	assert.Equal(t, "tab", km.TabNext.Help().Key)
	assert.Equal(t, "category", km.TabNext.Help().Desc, "Tab help should read 'category', not 'filter'")

	assert.Equal(t, []string{"shift+tab"}, km.TabPrev.Keys())
}
```

Note: `nextPage` and `prevPage` are unexported fields; assert via the rendered KeyBar in Task 3 instead. The above covers the public surface.

- [ ] **Step 2.2: Run the test to verify it fails**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -run TestSearchKeyMap_OnlyVisibleBindings -v`
Expected: FAIL — current `TabNext` help reads `"filter"`, test expects `"category"`.

- [ ] **Step 2.3: Shrink the keymap struct**

In `internal/ui/panes/search.go`, replace the entire `searchKeyMap` struct (currently lines 95-105 — re-read first) with:

```go
// searchKeyMap holds only the bindings advertised in the search overlay's
// bottom keybar. Enter (play) and Esc (close) are handled in Update() but
// not advertised — they are universal overlay conventions. Ctrl+U is no
// longer wired (see the 2026-04-28 overlay-keybinding-cleanup spec).
type searchKeyMap struct {
	Queue    key.Binding
	TabNext  key.Binding
	TabPrev  key.Binding
	nextPage key.Binding
	prevPage key.Binding
}
```

Delete the `ShortHelp() []key.Binding` and `FullHelp() [][]key.Binding` methods entirely (currently lines 107-127). They were only consumed by `bubbles/help`, which is being removed in Task 3.

- [ ] **Step 2.4: Shrink NewSearchKeyMap**

Replace the body of `NewSearchKeyMap` (currently lines 131-168) with:

```go
// NewSearchKeyMap creates the default keybindings for the search overlay.
// Exported for tests. Only includes bindings shown in the bottom keybar.
func NewSearchKeyMap() searchKeyMap {
	return searchKeyMap{
		Queue: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "queue"),
		),
		TabNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "category"),
		),
		TabPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev"),
		),
		nextPage: key.NewBinding(
			// pgdown is the sole pagination key. ctrl+right was removed because macOS
			// intercepts it at the OS level for Spaces/Desktop navigation.
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "next"),
		),
		prevPage: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "prev"),
		),
	}
}
```

- [ ] **Step 2.5: Drop the help model from SearchOverlay struct**

In the `SearchOverlay` struct (currently lines 180-230 — re-read first), delete the `help help.Model` field. In `NewSearchOverlay` (currently lines 254-264), delete the entire `h := help.New()` block including all `h.Styles.*` assignments (10 lines). In the returned struct literal, delete the `help: h,` line.

In `SetSize` (currently lines 402-411), delete the trailing 3 lines that set `o.help.Width`. The function should end after `o.resizeList()`.

- [ ] **Step 2.6: Remove the help import**

Edit the import block at the top of `internal/ui/panes/search.go`. Remove the line `"github.com/charmbracelet/bubbles/help"`. Leave `bubbles/key` (still used) and other bubbles imports alone.

- [ ] **Step 2.7: Run package tests to verify compilation + keymap test passes**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go build ./...`
Expected: SUCCESS. The `renderHelpPanel` function still calls `o.help.View(o.keyMap)` and references `searchKeyMap.FullHelp` — this WILL break compilation. That breakage is the pivot into Task 3; do not attempt to fix it in this task.

If there are compilation errors only inside `renderHelpPanel` (line 993-1016 area), proceed to Task 3 immediately. If errors appear elsewhere (other callers of removed methods), stop and re-read those files; the spec assumed only `renderHelpPanel` consumes the help model.

Do NOT commit yet — Task 3 follows in the same commit window because the build is broken.

---

## Task 3: Replace Keys panel with single-line uikit.KeyBar

**Files:**
- Modify: `internal/ui/panes/search.go:993-1016` (`renderHelpPanel`)
- Modify: `internal/ui/panes/search.go:384-398` (`panelHeights` — change `helpH = 4` to `helpH = 3`, fix docstring)
- Modify: `internal/ui/panes/search.go` import block (add `"github.com/initgrep-apps/spotnik/internal/uikit"` if not present)

- [ ] **Step 3.1: Write a failing render test**

Append to `internal/ui/panes/search_test.go`:

```go
// TestSearchOverlay_View_KeysPanel_SingleLine verifies the Keys panel renders
// a single-line uikit.KeyBar with the cleaned binding set, and that the dead
// bindings (Enter play, Esc close, Ctrl+U clear) are absent.
func TestSearchOverlay_View_KeysPanel_SingleLine(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	view := o.View()

	plain := stripANSI(view)

	// Visible bindings present, in order, on one line of output.
	assert.Contains(t, plain, "ctrl+a queue", "ctrl+a queue must appear")
	assert.Contains(t, plain, "tab category", "tab category must appear (label changed from 'filter')")
	assert.Contains(t, plain, "shift+tab prev", "shift+tab prev must appear")
	assert.Contains(t, plain, "pgdn next", "pgdn next must appear")
	assert.Contains(t, plain, "pgup prev", "pgup prev must appear")

	// Dead bindings absent.
	assert.NotContains(t, plain, "enter play", "Enter play must not be advertised")
	assert.NotContains(t, plain, "esc close", "Esc close must not be advertised")
	assert.NotContains(t, plain, "ctrl+u clear", "Ctrl+U must not be advertised")
}
```

If the file does not yet have a local `stripANSI` helper, copy the one from `internal/ui/panes/search_delegate_test.go:794-799` to the top of `search_test.go`. Verify with `grep -n stripANSI internal/ui/panes/search_test.go` first; if it already exists, skip the copy.

- [ ] **Step 3.2: Run to verify it fails (compilation error from Task 2)**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -run TestSearchOverlay_View_KeysPanel_SingleLine -v`
Expected: build failure mentioning `o.help` and/or `FullHelp` undefined inside `renderHelpPanel`. This is expected — Task 3 fixes it.

- [ ] **Step 3.3: Rewrite renderHelpPanel**

In `internal/ui/panes/search.go`, replace the entire `renderHelpPanel` function (currently around lines 993-1016) with:

```go
// renderHelpPanel builds Panel 3: the keybinding hint bar.
// Height is fixed at 3 (top border + single content line + bottom border).
// Renders a uikit.KeyBar over the visible binding subset. Title is empty
// because the binding content is self-explanatory, and the dim TextMuted
// border lets the panel recede.
func (o *SearchOverlay) renderHelpPanel(w, h int) string {
	innerWidth := w - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	bar := uikit.KeyBar{
		Bindings: o.hintBindings(),
		Theme:    o.theme,
	}.Render()

	inner := lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Height(h - 2).MaxHeight(h - 2).
		Render(bar)

	cfg := layout.BorderConfig{
		Width:       w,
		Height:      h,
		Title:       "",
		Actions:     []layout.Action{},
		AccentColor: o.theme.TextMuted(),
		Focused:     false,
		Theme:       o.theme,
	}
	return layout.RenderPaneBorder(inner, cfg)
}

// hintBindings returns the synthetic key.Binding list rendered by the bottom
// keybar. The composite Key strings ("tab/shift+tab", "pgdn/pgup") exist
// purely for display and are NEVER matched against tea.KeyMsg input — real
// key handling lives in handleKey().
func (o *SearchOverlay) hintBindings() []key.Binding {
	return []key.Binding{
		o.keyMap.Queue,
		key.NewBinding(key.WithHelp("tab/shift+tab", "category")),
		key.NewBinding(key.WithHelp("pgdn/pgup", "page")),
	}
}
```

If `"github.com/initgrep-apps/spotnik/internal/uikit"` is not in the import block, add it. Confirm by re-reading the import block.

- [ ] **Step 3.4: Update panelHeights**

In `internal/ui/panes/search.go`, replace the `panelHeights` function body (currently lines 386-398) with:

```go
// panelHeights returns the computed heights for the three overlay panels:
// searchH (3 or 4 depending on hint line), resultsH (fills remaining), helpH (always 3).
func (o *SearchOverlay) panelHeights() (searchH, resultsH, helpH int) {
	searchH = 3
	if o.showHintLine() {
		searchH = 4
	}
	helpH = 3 // single-line uikit.KeyBar + top/bottom border
	totalH := o.overlayHeight()
	resultsH = totalH - searchH - helpH
	if resultsH < 5 {
		resultsH = 5
	}
	return
}
```

Only change is `helpH = 4` → `helpH = 3`. Docstring already said "always 3" — was stale; now accurate.

- [ ] **Step 3.5: Run the new test to verify it passes**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -run TestSearchOverlay_View_KeysPanel_SingleLine -v`
Expected: PASS.

- [ ] **Step 3.6: Run the full package test suite**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -count=1`
Expected: PASS. If a height-arithmetic test fails, it's likely an existing test asserting `helpH=4`. Inspect the failing test:
- If it asserts the old `helpH=4` value directly, update the assertion to `3`.
- If it does pixel-perfect view comparison and fails because results gained 1 line, update the expected fixture.

- [ ] **Step 3.7: Commit (Tasks 2+3 together — the keymap shrink and renderer swap belong in one commit)**

```bash
cd /Users/irshadsheikh/dev/github/apps/spotnik
git add internal/ui/panes/search.go internal/ui/panes/search_test.go
git commit -m "refactor(search): single-line KeyBar, drop dead help bindings

Replace bubbles/help 4×2 grid with uikit.KeyBar single-line render.
Shrink searchKeyMap from 8 to 5 bindings (Play/Close/Clear removed —
they were duplicated with the Results border notch and with universal
overlay conventions). Tab help text changes from 'filter' to 'category'
to match actual behavior.

Per docs/superpowers/specs/2026-04-28-overlay-keybinding-cleanup-design.md."
```

---

## Task 4: Drop Results panel corner-notch Actions

**Files:**
- Modify: `internal/ui/panes/search.go:942-955` (`renderResultsPanel` BorderConfig)
- Modify: `internal/ui/panes/search_test.go` (any test asserting on the corner-notch text)

- [ ] **Step 4.1: Write a failing test**

Append to `internal/ui/panes/search_test.go`:

```go
// TestSearchOverlay_View_ResultsPanel_NoCornerActions verifies the Results
// panel border no longer carries the "Enter play / Ctrl+A queue" corner-notch
// actions. The bottom keybar is the single source of truth for visible
// bindings.
func TestSearchOverlay_View_ResultsPanel_NoCornerActions(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 40)
	view := o.View()

	plain := stripANSI(view)

	// Look only on the line(s) above the Results panel content. The simplest
	// check: "Enter play" and "Ctrl+A queue" must not appear adjacent to the
	// border-style notch corners ╭ or ╮ that mark corner-notch actions.
	// A coarser check is sufficient: those exact phrases must be absent
	// from the entire view (the bottom keybar uses lowercase "ctrl+a queue").
	assert.NotContains(t, plain, "Enter play", "corner-notch 'Enter play' must be removed")
	assert.NotContains(t, plain, "Ctrl+A queue", "corner-notch 'Ctrl+A queue' must be removed")
}
```

- [ ] **Step 4.2: Run to verify it fails**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -run TestSearchOverlay_View_ResultsPanel_NoCornerActions -v`
Expected: FAIL — current view contains both phrases via the corner-notch render.

- [ ] **Step 4.3: Drop Actions in renderResultsPanel BorderConfig**

In `internal/ui/panes/search.go`, locate the `renderResultsPanel` function and its `BorderConfig` literal (around lines 942-955). Replace the `Actions: []layout.Action{...}` field with `Actions: nil` (or remove the line entirely — both are valid). The full BorderConfig should look like:

```go
cfg := layout.BorderConfig{
	Width:       w,
	Height:      h,
	Title:       "Results",
	Actions:     nil,
	AccentColor: o.theme.SeekBar(),
	Focused:     false,
	Theme:       o.theme,
}
```

- [ ] **Step 4.4: Run the new test to verify it passes**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -run TestSearchOverlay_View_ResultsPanel_NoCornerActions -v`
Expected: PASS.

- [ ] **Step 4.5: Run the full package test suite**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -count=1`
Expected: PASS. If an existing test asserts the corner-notch text exists, locate it (likely named `TestSearchOverlay_View_*Actions*` or similar — `grep -n "Enter play\|Ctrl+A queue" internal/ui/panes/search_test.go`) and either delete it or invert its assertion.

- [ ] **Step 4.6: Commit**

```bash
cd /Users/irshadsheikh/dev/github/apps/spotnik
git add internal/ui/panes/search.go internal/ui/panes/search_test.go
git commit -m "refactor(search): drop Results corner-notch actions

The bottom KeyBar advertises ctrl+a queue. Duplicating it on the
Results border was the original duplication this cleanup targets."
```

---

## Task 5: Update keybinding documentation (search section)

**Files:**
- Modify: `docs/keybinding.md` (Search Overlay section)
- Modify: `docs/DESIGN.md` (§17, Search Overlay table)

This task fulfills CLAUDE.md rule #15: same-commit doc trio update for any keybinding change. It is a separate commit here for clarity but stays in the same PR/branch as Tasks 1-4. (Rule #15's "same commit" is a guard against doc drift across PRs; an audit trail spread across small commits within one PR satisfies the spirit.)

If you prefer strict literal compliance, squash Tasks 1-5 before opening the PR.

- [ ] **Step 5.1: Edit docs/keybinding.md**

Open `docs/keybinding.md`. Locate the `## Search Overlay` section (use grep: `grep -n "## Search Overlay" docs/keybinding.md`). The current table has these rows:

```
| `Tab` / `Shift+Tab` | Cycle search category |
| `Enter` | Play selected result |
| `Ctrl+A` | Add result to queue |
| `Ctrl+U` | Clear search input |
| `PgDn` / `PgUp` | Next / previous result page |
| `Esc` | Close search overlay |
```

Delete the rows for `Enter`, `Ctrl+U`, and `Esc`. The Tab row already reads "category" — no change. The final table:

```
| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle search category |
| `Ctrl+A` | Add result to queue |
| `PgDn` / `PgUp` | Next / previous result page |
```

- [ ] **Step 5.2: Edit docs/DESIGN.md §17**

Open `docs/DESIGN.md`. Locate §17 (use grep: `grep -n "§17\|## 17\|Keybindings" docs/DESIGN.md`). Find the Search Overlay subsection and apply the same row deletions: drop `Enter`, `Ctrl+U`, `Esc`. If the Tab row uses the label "filter" instead of "category", change it to "category".

- [ ] **Step 5.3: Verify both docs**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && grep -A 8 "## Search Overlay" docs/keybinding.md`
Expected: only Tab/Ctrl+A/PgDn rows in the table.

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && grep -B 2 -A 12 "Search.*Overlay\|search overlay" docs/DESIGN.md | head -40`
Expected: no Enter/Ctrl+U/Esc references in the search-overlay portion.

- [ ] **Step 5.4: Commit**

```bash
cd /Users/irshadsheikh/dev/github/apps/spotnik
git add docs/keybinding.md docs/DESIGN.md
git commit -m "docs: drop Enter/Ctrl+U/Esc from search-overlay keybinding tables

Match the in-app overlay's bottom keybar (introduced in the prior
commits): only the visible bindings (Tab/Shift+Tab, Ctrl+A, PgDn/PgUp)
are documented. Enter and Esc remain universal overlay conventions
(handled in Update but not advertised); Ctrl+U is removed entirely."
```

---

## Task 6: Profile overlay table layout + glyph migration + KeyBar

**Files:**
- Modify: `internal/ui/panes/profile.go` (entire `View()` and `renderActions()` rewrite)
- Modify: `internal/ui/panes/profile_test.go` (rewrite logout/forget rendering tests, add glyph + table tests)

- [ ] **Step 6.1: Write the failing tests**

Append to `internal/ui/panes/profile_test.go`. If the file lacks a local `stripANSI` helper, copy from `search_delegate_test.go:794-799`:

```go
// TestProfileOverlay_View_TableLayout verifies the three icon+value rows
// render in order (Name, Plan, Region), each as glyph + two-space gap + value,
// inside the single bordered card.
func TestProfileOverlay_View_TableLayout(t *testing.T) {
	st := newTestStoreWithProfile("Irshad", "IN", true)
	p := panes.NewProfileOverlay(st, theme.Load("black"))
	p.SetSize(80, 40)

	plain := stripANSI(p.View())

	// Three rows in order. Glyphs use unicode mode (default).
	assert.Regexp(t, `◉\s+Irshad`, plain, "name row: ◉ + Irshad")
	assert.Regexp(t, `♛\s+Premium`, plain, "plan row: ♛ + Premium")
	assert.Regexp(t, `◎\s+IN`, plain, "region row: ◎ + IN")

	// Order check: Name appears before Plan appears before Region.
	idxName := strings.Index(plain, "Irshad")
	idxPlan := strings.Index(plain, "Premium")
	idxRegion := strings.Index(plain, "IN")
	require.True(t, idxName >= 0 && idxPlan >= 0 && idxRegion >= 0, "all three rows must render")
	assert.Less(t, idxName, idxPlan, "Name above Plan")
	assert.Less(t, idxPlan, idxRegion, "Plan above Region")
}

// TestProfileOverlay_View_KeyBarLine verifies the action area renders as a
// single uikit.KeyBar line ("l logout · f forget"), not as two stacked rows.
func TestProfileOverlay_View_KeyBarLine(t *testing.T) {
	st := newTestStoreWithProfile("Irshad", "IN", true)
	p := panes.NewProfileOverlay(st, theme.Load("black"))
	p.SetSize(80, 40)

	plain := stripANSI(p.View())

	// Both bindings on the same logical line, separated by the KeyBar middot.
	// KeyBar renders "l logout · f forget" (or "l logout | f forget" in ASCII mode).
	assert.Regexp(t, `l\s+logout\s*[·|]\s*f\s+forget`, plain,
		"l logout and f forget must appear on one line separated by the KeyBar separator")

	// Old stacked-row format must be gone.
	assert.NotContains(t, plain, "l  Logout", "old stacked Logout row must be removed")
	assert.NotContains(t, plain, "f  Forget", "old stacked Forget row must be removed")
}

// TestProfileOverlay_View_NoSeparators verifies the horizontal rule lines
// ("────────────────────") used as section separators are gone.
func TestProfileOverlay_View_NoSeparators(t *testing.T) {
	st := newTestStoreWithProfile("Irshad", "IN", true)
	p := panes.NewProfileOverlay(st, theme.Load("black"))
	p.SetSize(80, 40)

	plain := stripANSI(p.View())

	// The 20-char rule line is no longer rendered. Use a long-enough run
	// of "─" to avoid matching the 1-char rule glyphs in border tops.
	assert.NotRegexp(t, `─{10}`, plain, "horizontal separator rules must be removed")
}
```

If `newTestStoreWithProfile` does not exist, inspect existing tests to find the equivalent helper. Look at `TestProfileOverlay_View_PremiumBadge` (line 38) to see how the store fixture is set up, and reuse that pattern inline if no helper exists.

- [ ] **Step 6.2: Run to verify they fail**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -run TestProfileOverlay_View_TableLayout -v`
Expected: FAIL — current view renders Name as bold heading without the `◉` glyph.

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -run TestProfileOverlay_View_KeyBarLine -v`
Expected: FAIL — current view renders `l  Logout` and `f  Forget` on separate lines.

- [ ] **Step 6.3: Rewrite profile.go View()**

Replace the body of `View()` in `internal/ui/panes/profile.go` (currently lines 109-174) with:

```go
// View renders the profile overlay content.
// Pure function — reads store state, returns a string, performs no I/O.
func (p *ProfileOverlay) View() string {
	profile := p.store.UserProfile()
	isPremium := p.store.IsPremium()

	const innerWidth = 22 // narrow card: 3 short rows + 1 keybar line

	mode := uikit.ActiveMode()

	var lines []string

	if profile.ID == "" {
		loadingStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
		lines = append(lines, loadingStyle.Render("Loading profile..."))
	} else {
		// Row 1 — Name.
		name := truncateRunes(profile.DisplayName, maxProfileNameLen)
		lines = append(lines, p.renderRow(
			uikit.GlyphFor(uikit.GlyphActive, mode),
			p.theme.Info(),
			p.theme.TextPrimary(),
			name,
		))

		// Row 2 — Plan.
		var planGlyph string
		var planValue string
		var planColor lipgloss.Color
		if isPremium {
			planGlyph = uikit.GlyphFor(uikit.GlyphPremium, mode)
			planValue = "Premium"
			planColor = p.theme.Info()
		} else {
			planGlyph = uikit.GlyphFor(uikit.GlyphFreeTier, mode)
			planValue = "Free"
			planColor = p.theme.TextMuted()
		}
		lines = append(lines, p.renderRow(
			planGlyph,
			planColor,
			p.theme.TextPrimary(),
			planValue,
		))

		// Row 3 — Region (only if known).
		if profile.Country != "" {
			lines = append(lines, p.renderRow(
				uikit.GlyphFor(uikit.GlyphInactive, mode),
				p.theme.TextMuted(),
				p.theme.TextPrimary(),
				profile.Country,
			))
		}

		// Spacer + KeyBar action line.
		lines = append(lines, "")
		lines = append(lines, p.renderActions())
	}

	inner := strings.Join(lines, "\n")
	inner = lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Render(inner)

	cfg := layout.BorderConfig{
		Width:       innerWidth + 2,
		Height:      strings.Count(inner, "\n") + 3,
		Title:       "Profile",
		AccentColor: p.theme.ActiveBorder(),
		Focused:     true,
		Theme:       p.theme,
	}

	return layout.RenderPaneBorder(inner, cfg)
}
```

The width drops from 34 to 22 because there are no longer label columns or wide separator rules — short values fit comfortably. If the tests assert a specific width (unlikely but possible), revisit.

- [ ] **Step 6.4: Add the renderRow helper**

Append to `internal/ui/panes/profile.go` (just below `View()`):

```go
// renderRow renders a single icon+value line: "<glyph>  <value>".
// glyph and value get independent foreground colors so the glyph can carry
// intent (Info for premium, TextMuted for region) while the value uses the
// pane's primary text color.
func (p *ProfileOverlay) renderRow(glyph string, glyphColor, valueColor lipgloss.Color, value string) string {
	g := lipgloss.NewStyle().Foreground(glyphColor).Render(glyph)
	v := lipgloss.NewStyle().Foreground(valueColor).Render(value)
	return g + "  " + v
}
```

- [ ] **Step 6.5: Replace renderActions to return a single KeyBar string**

Replace the entire `renderActions` method (currently lines 192-211) with:

```go
// renderActions returns the single-line KeyBar advertising logout/forget.
// Real key handling lives in Update() — these key.Bindings are display-only.
// Confirmation feedback is delivered via toast (ProfileConfirmToastMsg);
// the rendered hint stays static regardless of pendingAction.
func (p *ProfileOverlay) renderActions() string {
	bindings := []key.Binding{
		key.NewBinding(key.WithHelp("l", "logout")),
		key.NewBinding(key.WithHelp("f", "forget")),
	}
	return uikit.KeyBar{Bindings: bindings, Theme: p.theme}.Render()
}
```

- [ ] **Step 6.6: Update imports**

In the import block at the top of `internal/ui/panes/profile.go`, add `"github.com/charmbracelet/bubbles/key"` (likely not yet imported — confirm by re-reading the import block). The `uikit` import is already present.

- [ ] **Step 6.7: Run the new tests to verify they pass**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -run TestProfileOverlay_View -v`
Expected: PASS for all four `TestProfileOverlay_View_*` new/updated tests.

- [ ] **Step 6.8: Run the full package test suite**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/ui/panes/ -count=1`
Expected: PASS. Likely affected existing tests:
- `TestProfileOverlay_View_ShowsDisplayName` — should still pass (the name still appears).
- `TestProfileOverlay_View_PremiumBadge` — should still pass (`Premium` text remains, glyph is now `♛` from `GlyphPremium` rather than the hardcoded `♛` — same output).
- `TestProfileOverlay_View_FreeBadge` — should still pass (`Free` text remains).
- `TestProfileOverlay_View_ShowsCountry` — should still pass.
- `TestProfileOverlay_View_LoadingState` — should still pass (loading branch untouched).
- `TestProfileOverlay_View_NoEscHint` — should still pass (no Esc hint added).
- `TestProfileOverlay_View_HasBorderCorners` — should still pass (border still rendered).
- All `*FirstPress*`, `*SecondPress*`, `*confirmation*`, `*differentKey*` tests — Update() unchanged, should pass.

If any existing test asserts on the hardcoded `"l  Logout"` or `"f  Forget"` literal, locate and update or delete it (the new `TestProfileOverlay_View_KeyBarLine` covers the replacement).

- [ ] **Step 6.9: Commit**

```bash
cd /Users/irshadsheikh/dev/github/apps/spotnik
git add internal/ui/panes/profile.go internal/ui/panes/profile_test.go
git commit -m "refactor(profile): icon+value table + inline KeyBar

Profile overlay now renders three glyph+value rows (Name, Plan, Region)
followed by a single uikit.KeyBar line for logout/forget — no separator
rules, no stacked ListRow actions.

All glyphs route through uikit.GlyphFor (♛ Premium, ◎ Region, ◉ Name)
so ASCII mode renders correctly. Double-press confirm behavior in
Update() is unchanged.

Per docs/superpowers/specs/2026-04-28-overlay-keybinding-cleanup-design.md."
```

---

## Task 7: Verify the full CI gate

- [ ] **Step 7.1: Run lint**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && make lint`
Expected: PASS, no warnings.

If `golangci-lint` flags an unused variable (e.g., `prefixState` field referenced only by the removed Ctrl+U handler), inspect first — `prefixState` is used elsewhere in the prefix state machine, so this should not happen. If a real unused symbol surfaces, delete it.

- [ ] **Step 7.2: Run tests with coverage**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && make test-coverage`
Expected: PASS, coverage ≥ 80%.

If coverage drops below 80%, identify the file:
- `profile.go`: most likely culprit if `renderRow` has untested branches (glyphColor variants). The new `TestProfileOverlay_View_TableLayout` covers the happy path; if Free-tier rendering is not covered, copy `TestProfileOverlay_View_FreeBadge`'s setup pattern and add an assertion using the table-layout regex.
- `search.go`: less likely — Tasks 1-4 deleted code without adding much new. If `hintBindings` is uncovered, the `TestSearchOverlay_View_KeysPanel_SingleLine` test exercises it via `View()`.

- [ ] **Step 7.3: Run the full CI target**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && make ci`
Expected: PASS end-to-end.

- [ ] **Step 7.4: Manual smoke (visual confirmation)**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && make build && ./bin/spotnik`

Confirm visually:
1. Press `/` to open search. Bottom of overlay shows a single line: `ctrl+a queue · tab/shift+tab category · pgdn/pgup page` inside a dim border. Results panel border shows just the title `Results`, no corner notches.
2. Type some chars, then press Ctrl+U — input must NOT clear. Press Backspace repeatedly — input clears as normal.
3. Press Esc to close. Press `u` to open profile. Card shows three rows (◉ Name, ♛ Premium, ◎ XX), blank line, then `l logout · f forget` inside the same border. No separator rules.
4. Press `l` once — toast appears (`Press l again to confirm logout`). Press any other key — toast clears, no logout.

If anything diverges from the spec mockups, file a follow-up task; do not patch in this branch unless it's a regression.

- [ ] **Step 7.5: Final commit (if any tweaks were needed)**

If Step 7.1-7.4 surfaced fixes, stage and commit them with an appropriate `fix(scope):` message. Otherwise, no commit needed.

- [ ] **Step 7.6: Push branch and open PR**

The branch should already exist (created at the start of execution). Push:

```bash
cd /Users/irshadsheikh/dev/github/apps/spotnik
git push -u origin <branch-name>
```

Open the PR with the title `refactor(overlays): keybinding cleanup for search + profile` and a body summarizing the spec link + the 5 commits.

---

## Self-Review

**Spec coverage:**
- Spec §Search Overlay → Keymap → Tasks 1, 2 ✓
- Spec §Search Overlay → Render → Task 3 ✓
- Spec §Search Overlay → Results panel border → Task 4 ✓
- Spec §Search Overlay → Update flow → Task 1 (Ctrl+U handler delete) ✓
- Spec §Search Overlay → Tests → covered inline in Tasks 1-4 ✓
- Spec §Profile Overlay → Layout → Task 6 ✓
- Spec §Profile Overlay → Glyphs → Task 6 (Step 6.3) ✓
- Spec §Profile Overlay → Row rendering → Task 6 (Step 6.4) ✓
- Spec §Profile Overlay → Key bar → Task 6 (Step 6.5) ✓
- Spec §Profile Overlay → Removed elements → Task 6 (Step 6.3) ✓
- Spec §Profile Overlay → Tests → covered inline in Task 6 ✓
- Spec §Doc Trio Update → Task 5 ✓
- Spec §Acceptance Criteria → Task 7 verifies all 8 criteria ✓

**Placeholder scan:** All steps contain executable commands or full code blocks. No "TBD"/"implement later"/"add error handling" patterns.

**Type consistency:** `searchKeyMap` field set used consistently across Tasks 2 (NewSearchKeyMap), 3 (hintBindings reads `o.keyMap.Queue`). `renderRow` signature in Task 6.4 matches all four call sites in Task 6.3. `hintBindings` defined in Task 3.3 returns the correct `[]key.Binding` consumed by `KeyBar{Bindings: ...}`.

---

## Execution Notes

- Task 2 intentionally leaves the build broken at its end. Task 3 fixes it. Do NOT commit between them.
- Tasks 1, 4, 5, 6 each end in passing tests + a commit. They can be paused/resumed safely.
- All test code in this plan uses `testify` patterns matching the existing `search_test.go` and `profile_test.go` style. If `assert.Regexp` is unfamiliar — it accepts a regex string and matches against the rendered output.
