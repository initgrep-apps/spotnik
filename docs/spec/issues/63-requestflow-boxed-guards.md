# Feature 63 — Request Flow Boxed Layout Defensive Guards

> **Fix:** Add defensive guards to the Request Flow boxed layout to prevent silent
> rendering failures at edge-case dimensions. All issues are from PR #76 review
> (issues I62-1 through I62-4).

## Background

Feature 62 introduced the boxed layout for the Request Flow pane. The code is correct
under normal conditions, but several edge-case paths can silently produce corrupted
output if constants are later changed or if the layout manager provides unexpected
dimensions. These are small, targeted fixes in two files.

---

## Task 1: Add post-clamp overflow guard in `viewBoxed()`

**Issue:** I62-1 — After clamping column widths to minimums, there is no check that
their sum exceeds `contentWidth`. Currently safe but fragile.

**Fix:**

In `internal/ui/panes/requestflow_pane.go`, in `viewBoxed()`, after all minimum width
clamps (after the `if spotifyBoxW < 10` block around line 256), add:

```go
// Guard: if minimums push total beyond pane width, fall back to flat layout.
if appBoxW+arrowW+gwBoxW+arrowW+spotifyBoxW > contentWidth {
    return p.viewFlat()
}
```

**Tests:**
- `TestRequestFlowPane_View_BoxedOverflowFallback` — set width to 61 but temporarily
  test the logic by confirming the guard path works. Since current minimums sum to 46,
  we can't trigger this at width=61. Instead, test indirectly: confirm that at width=60
  the boxed layout renders (proving the guard didn't fire), and add a comment-only test
  documenting the guard exists.
- Actually, the simplest approach: just add the guard with no new test (it's a pure
  fallback safety net). The existing flat fallback tests cover `viewFlat()` behavior.

**Commit:** `fix(ui): add post-clamp overflow guard in viewBoxed`

---

## Task 2: Add height fallback guard in `viewBoxed()`

**Issue:** I62-4 — When `p.height` is very small (1-4), `boxAreaHeight` clamps to 3,
producing output taller than the allocated pane height.

**Fix:**

In `internal/ui/panes/requestflow_pane.go`, in `viewBoxed()`, replace the existing
`boxAreaHeight` clamp:

```go
boxAreaHeight := p.height - statusStripHeight - 1
if boxAreaHeight < 3 {
    boxAreaHeight = 3
}
```

With:

```go
boxAreaHeight := p.height - statusStripHeight - 1
// Minimum meaningful boxed layout: 2 border rows + 1 content row + separator + status = 5.
if boxAreaHeight < 3 {
    return p.viewFlat()
}
```

This falls back to flat layout when the pane is too short for boxes, rather than
producing oversized output.

**Tests:**
- `TestRequestFlowPane_View_ShortHeightFallback` — `SetSize(80, 4)` → View() does NOT
  contain `╭─ APP` (uses flat layout instead of producing oversized boxes)

**Commit:** `fix(ui): fall back to flat layout when pane height too small for boxes`

---

## Task 3: Add `maxRows <= 0` guard to arrow builders

**Issue:** I62-3 — `buildLeftArrowLines` and `buildRightArrowLines` don't guard against
`maxRows <= 0`, unlike the box content builders which return nil.

**Fix:**

In `internal/ui/panes/requestflow_boxed.go`, add a guard at the top of both functions:

```go
func (p *RequestFlowPane) buildLeftArrowLines(maxRows, colWidth int) []string {
    if maxRows <= 0 {
        return nil
    }
    // ... rest unchanged
}

func (p *RequestFlowPane) buildRightArrowLines(maxRows, colWidth int) []string {
    if maxRows <= 0 {
        return nil
    }
    // ... rest unchanged
}
```

**Tests:**
- `TestBuildLeftArrowLines_ZeroMaxRows` — `buildLeftArrowLines(0, 10)` returns nil
- `TestBuildRightArrowLines_ZeroMaxRows` — `buildRightArrowLines(0, 10)` returns nil

**Commit:** `fix(ui): add maxRows guard to arrow line builders for consistency`

---

## Task 4: Add caller guard for empty `renderSubBox` output

**Issue:** I62-2 — `renderSubBox` returns `""` for width < 8, but `viewBoxed()` doesn't
check before compositing.

**Fix:**

This is already covered by Task 1's overflow guard — if any box would be < 8 wide, the
total column sum would far exceed `contentWidth`, triggering the fallback. Add a comment
in `renderSubBox` to document this precondition:

```go
// renderSubBox renders a small bordered box with a title label.
// ...existing doc...
// If width < 8, returns empty string (too narrow for a meaningful box).
// NOTE: viewBoxed() guarantees width >= 10 for all boxes via minimum clamps
// and falls back to viewFlat() if totals exceed pane width.
func (p *RequestFlowPane) renderSubBox(title string, lines []string, width int) string {
```

No code change needed beyond the comment. No new test needed.

**Commit:** `docs(ui): document renderSubBox width precondition`

---

## Acceptance Criteria

- [ ] `viewBoxed()` falls back to `viewFlat()` when column width minimums exceed pane width
- [ ] `viewBoxed()` falls back to `viewFlat()` when pane height < 5
- [ ] `buildLeftArrowLines(0, w)` and `buildRightArrowLines(0, w)` return nil
- [ ] `renderSubBox` documents its caller precondition
- [ ] All existing tests pass unchanged
- [ ] New guard tests added
- [ ] `make ci` passes

---

## Verification

```bash
# Overflow guard exists
grep 'viewFlat' internal/ui/panes/requestflow_pane.go | grep -c 'contentWidth'

# Height guard exists
grep -A2 'boxAreaHeight < 3' internal/ui/panes/requestflow_pane.go

# Arrow guards exist
grep -A1 'func.*buildLeftArrowLines\|func.*buildRightArrowLines' internal/ui/panes/requestflow_boxed.go

# Full CI
make ci
```

---

*Depends on: Feature 62*
*Blocks: Nothing*
