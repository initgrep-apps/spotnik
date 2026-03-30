---
title: "Reusable Components"
feature: 12-layout
status: done
---

## Background
The new DESIGN.md specifies that all 7 list panes plus the Network Log pane render data in dense aligned columns with per-column colors, scrolling, and real-time filtering. Currently, panes render lists manually with fmt.Sprintf and fixed-width padding. There are no reusable table or filter components. This story creates three components: a Table wrapper around github.com/evertras/bubble-table with Spotnik conventions, an in-pane Filter using bubbles/textinput, and rune-aware truncation utilities.

Design reference: docs/DESIGN.md sections 6, 8, 9.

## Design

### Truncation Utilities
`Truncate(s string, maxWidth int) string`, `PadRight(s string, width int) string`, `TruncateOrPad(s string, width int) string` -- all using lipgloss.Width() for measurement. Handles CJK 2-cell, emoji, combining marks, ANSI escapes.

### Table Component
Wraps bubble-table with: flex columns via btable.NewFlexColumn(), per-column colors, selected row highlighting, playing indicator (▶), borderless mode, WithRowStyleFunc for selection styling, WithTargetWidth(width), WithPageSize(height - headerLines).

### Filter Component
Wraps textinput with: toggle, case-insensitive matching via strings.Contains(strings.ToLower(...)), border label, Esc deactivates+clears, Enter deactivates+keeps query, CharLimit 50.

## Acceptance Criteria
- [ ] `bubble-table` dependency added to go.mod
- [ ] Truncate/PadRight/TruncateOrPad use lipgloss.Width() for measurement
- [ ] Table wraps bubble-table with flex columns, per-column colors, selected row highlighting, playing indicator, borderless mode
- [ ] Filter wraps textinput with toggle, case-insensitive matching, border label, Esc/Enter handling
- [ ] Table.SetSize() recalculates column widths
- [ ] Filter.Matches() returns true when filter is inactive
- [ ] All 3 components have independent tests
- [ ] No imports from app/, api/, state/
- [ ] `make ci` passes

## Tasks
- [ ] Add bubble-table dependency
      - test: go build ./... succeeds
- [ ] Create truncation utilities in internal/ui/layout/truncate.go
      - test: shorter/equal/longer strings; CJK; ANSI escapes; empty; maxWidth=0/1; PadRight; TruncateOrPad
- [ ] Create Table component wrapper in internal/ui/components/table.go
      - test: NewTable; SetSize; SetRows; SelectedIndex; keyboard nav; SetPlayingIndex; header/column colors; selected row override; empty rows; width recalculation
- [ ] Create Filter component in internal/ui/components/filter.go
      - test: starts inactive; Toggle; Query; Matches case-insensitive; MatchesAny; BorderLabel; View; Esc/Enter
- [ ] Component integration tests
      - test: Table with Filter; resize cycle; filter activation/deactivation; Truncate in table cells; empty rows; empty query
