---
name: project_spotnik_feature63_complete
description: Feature 63 (Request Flow Boxed Guards): 4 defensive guards for viewBoxed(), comment accuracy fix caught in review
type: project
---

## Feature 63 — Request Flow Boxed Layout Defensive Guards

**Built:**
- Post-clamp overflow guard `viewBoxed()`: column min widths sum > contentWidth → fallback `viewFlat()`
- Height fallback `viewBoxed()`: replaced `boxAreaHeight = 3` clamp w/ `return p.viewFlat()` when `boxAreaHeight < 3`
- `maxRows <= 0` guard added to both `buildLeftArrowLines` and `buildRightArrowLines`
- Doc comment on `renderSubBox` documenting caller precondition

**Files:**
- `internal/ui/panes/requestflow_pane.go` — Tasks 1+2: overflow guard + height fallback
- `internal/ui/panes/requestflow_boxed.go` — Task 3: arrow builder guards; Task 4: renderSubBox doc
- `internal/ui/panes/requestflow_pane_test.go` — `TestRequestFlowPane_View_ShortHeightFallback`
- `internal/ui/panes/requestflow_boxed_test.go` — `TestBuildLeftArrowLines_ZeroMaxRows`, `TestBuildRightArrowLines_ZeroMaxRows`

**Gotchas:**
- Task 1 overflow guard unreachable at runtime w/ current minimums (10+7+12+7+10=46, threshold width >= 60). Tests note via comment vs unreachable test.
- `innerRows < 1` clamp (requestflow_pane.go:266-268) now dead code — height guard ensures `boxAreaHeight >= 3` → `innerRows >= 1`. Harmless, kept as safety net.
- PR review flagged wrong inline comment: "separator + status = 5" incorrect — both already subtracted from `boxAreaHeight` pre-guard. Fixed to "Need at least 3 rows for a meaningful box (top border + 1 content row + bottom border)."

**Testing:**
- 3 new tests, internal/external split per F62
- Coverage: 90.0% `internal/ui/panes`, 86.4% overall
- Task 1 overflow guard tested via comment (unreachable at current minimums)