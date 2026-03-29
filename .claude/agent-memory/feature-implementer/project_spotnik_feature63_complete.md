---
name: project_spotnik_feature63_complete
description: Feature 63 (Request Flow Boxed Guards): 4 defensive guards for viewBoxed(), comment accuracy fix caught in review
type: project
---

## Feature 63 — Request Flow Boxed Layout Defensive Guards

**What was built:**
- Post-clamp overflow guard in `viewBoxed()`: if column width minimums sum > contentWidth, falls back to `viewFlat()`
- Height fallback in `viewBoxed()`: replaced `boxAreaHeight = 3` clamp with `return p.viewFlat()` when `boxAreaHeight < 3`
- `maxRows <= 0` guard added to both `buildLeftArrowLines` and `buildRightArrowLines`
- Doc comment added to `renderSubBox` documenting the caller precondition

**Key files:**
- `internal/ui/panes/requestflow_pane.go` — Tasks 1 & 2: overflow guard + height fallback
- `internal/ui/panes/requestflow_boxed.go` — Task 3: arrow builder guards; Task 4: renderSubBox doc comment
- `internal/ui/panes/requestflow_pane_test.go` — `TestRequestFlowPane_View_ShortHeightFallback`
- `internal/ui/panes/requestflow_boxed_test.go` — `TestBuildLeftArrowLines_ZeroMaxRows`, `TestBuildRightArrowLines_ZeroMaxRows`

**Gotchas:**
- The overflow guard (Task 1) cannot be triggered at runtime with current minimums (10+7+12+7+10=46, threshold is width >= 60). Tests document this in a comment rather than adding a test that can't realistically exercise the guard.
- The `innerRows < 1` clamp (lines 266-268 in requestflow_pane.go) is now dead code because the height guard ensures `boxAreaHeight >= 3` → `innerRows >= 1`. It's harmless and was left in place as an extra safety net.
- PR review caught an inaccurate inline comment: "separator + status = 5" was wrong because both are already subtracted from `boxAreaHeight` before the guard. Fixed to "Need at least 3 rows for a meaningful box (top border + 1 content row + bottom border)."

**Testing notes:**
- 3 new tests, all internal/external split as established in F62
- Coverage: 90.0% for `internal/ui/panes`, 86.4% overall
- Task 1 overflow guard tested via comment (not executable at current minimums)
