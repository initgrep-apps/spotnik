---
title: "Request Flow Boxed Layout Defensive Guards"
feature: 14-nerd-status
status: done
---

## Background
Feature 62 introduced the boxed layout for the Request Flow pane. The code is correct under normal conditions, but several edge-case paths can silently produce corrupted output if constants are later changed or if the layout manager provides unexpected dimensions. These are small, targeted fixes from PR #76 review (issues I62-1 through I62-4).

## Design

### Task 1: Post-Clamp Overflow Guard
After clamping column widths to minimums, add check that their sum doesn't exceed contentWidth. Falls back to viewFlat().

### Task 2: Height Fallback Guard
When boxAreaHeight < 3, fall back to viewFlat() instead of clamping to 3 (which produces oversized output).

### Task 3: Arrow Builder maxRows Guard
Add `if maxRows <= 0 { return nil }` to buildLeftArrowLines and buildRightArrowLines.

### Task 4: renderSubBox Width Precondition
Document that viewBoxed() guarantees width >= 10 for all boxes via minimum clamps. No code change beyond comment.

## Acceptance Criteria
- [ ] viewBoxed() falls back to viewFlat() when column minimums exceed pane width
- [ ] viewBoxed() falls back to viewFlat() when pane height < 5
- [ ] buildLeftArrowLines(0, w) and buildRightArrowLines(0, w) return nil
- [ ] renderSubBox documents its caller precondition
- [ ] All existing tests pass unchanged
- [ ] New guard tests added
- [ ] make ci passes

## Tasks
- [ ] Add post-clamp overflow guard in viewBoxed() in requestflow_pane.go
      - test: guard exists as safety net (current minimums don't trigger it)
- [ ] Add height fallback guard in viewBoxed()
      - test: SetSize(80, 4) -> View() does NOT contain boxed layout (uses flat)
- [ ] Add maxRows <= 0 guard to arrow builders in requestflow_boxed.go
      - test: buildLeftArrowLines(0, 10) returns nil; buildRightArrowLines(0, 10) returns nil
- [ ] Add caller guard documentation for renderSubBox
      - test: docs comment only, no code change
