---
title: "Update design.md and tui.md for pane content design language"
feature: 20-pane-content-design
status: done
---

## Background

The design system documentation in `docs/system/design.md` and `docs/system/tui.md` must reflect the new design language: column ordering rules, priority thresholds, responsive column behavior, header guidelines, empty state requirements, and the removal of PlayingIndex and `#` column.

**Depends on:** All prior stories (245–251) — documents the final state after all code changes are implemented.

## Design

### `docs/system/design.md` changes

#### §9 Dense Table Formatting — remove # column references

Update all pane column tables to remove the `#` (index) column. Current tables show `#` as the first column — remove entirely.

#### §9 — add "Column Ordering Rule"

```markdown
### Column Ordering

Icon/glyph columns MUST be the first data column in every pane table. Ordering:

[Icon/Glyph] → [Primary Identifier] → [Secondary Info] → [Tertiary/Metadata]
```

#### §9 — add "Column Priority Thresholds"

```markdown
### Column Priority Thresholds

Columns have a `Priority` field (1-3). At render time, columns are filtered by
pane width:

| Priority | Label | Threshold | Behavior |
|----------|-------|-----------|----------|
| 1 | Always | Any width | Always rendered |
| 2 | Default | ≥ 40 cols | Hidden when narrow |
| 3 | Wide-only | ≥ 60 cols | Hidden unless spacious |

The table wrapper (`components.Table`) applies this filter in `rebuild()`. Width
changes across threshold boundaries trigger a table rebuild.
```

#### §9 — add "Column Header Guidelines"

```markdown
### Column Header Guidelines

Headers must not be wider than typical cell content. Use short names:
- `Dur` instead of `Duration`
- `Pop` instead of `Popularity`
- `Pub` instead of `Publisher`
- `Eps` instead of `Episodes`
```

#### §6 Content Containment — add rules

Add: "No blank rows between pane border and table content. Table must fill available height. Empty data sets render `EmptyState`, not empty table."

#### Remove PlayingIndex references

Search design.md for "PlayingIndex", "▶", "playing indicator". Remove all references.

#### Update column tables per pane

Each pane's column reference table in §9 must show the new column sets (no `#`, correct icon positions, updated headers, priority values).

### `docs/system/tui.md` changes

#### Update `TableChrome` primitive spec

Add `Priority` field to column definitions. Remove `PlayingIndex` reference from the `Table` model description.

#### Add "Responsive Columns" to Table behavior section

Document priority-based column hiding, threshold crossing, and the `crossesThreshold()` mechanism.

## Files

### Modify

- `docs/system/design.md` — update §9 column tables, add ordering/priority/header rules, update §6, remove PlayingIndex references
- `docs/system/tui.md` — update TableChrome primitive spec, add Responsive Columns section, remove PlayingIndex references

## Acceptance Criteria

- [ ] `docs/system/design.md` §9: no `#` column in any pane column table
- [ ] `docs/system/design.md` §9: icon columns listed first where applicable
- [ ] `docs/system/design.md` §9: column priority thresholds documented
- [ ] `docs/system/design.md` §9: column ordering rule documented
- [ ] `docs/system/design.md` §9: column header guidelines documented
- [ ] `docs/system/design.md` §6: no-blank-rows and EmptyState consistency rules added
- [ ] `docs/system/design.md`: zero `PlayingIndex` or `▶` references
- [ ] `docs/system/tui.md`: `Priority` field added to column definition spec
- [ ] `docs/system/tui.md`: responsive columns behavior documented
- [ ] `docs/system/tui.md`: zero `PlayingIndex` references

## Tasks

- [ ] **Task 1: Update docs/system/design.md**
  Remove `#` column from all §9 pane column tables. Add column ordering rule, priority thresholds table, and header guidelines to §9. Update §6 content containment with no-blank-rows rule. Remove all PlayingIndex/▶ references.
  - test: visual review of rendered markdown

- [ ] **Task 2: Update docs/system/tui.md**
  Add `Priority` field to TableChrome column definition spec. Add Responsive Columns behavior section. Remove PlayingIndex references from Table model.
  - test: visual review of rendered markdown
