---
title: "Docs, Dead Code & Defensive Init"
feature: 00-architecture
status: open
---

## Background
PR reviews of features 29-33 identified several documentation inaccuracies, dead code, and a fragile initialization pattern. These are all non-behavioral changes that improve code quality and correctness of documentation.

**Source:** `docs/issues.md` -- PR #34 issues 1-3, 5; PR #37 issue 9

**Depends on:** Nothing

## Design

### Task 1: Fix stale store.go documentation

**Problem:** Two doc comments in `internal/state/store.go` reference the old "Commands write to the store" pattern. Since Feature 29, the rule is "only Update() writes to the store via data-carrying Msg payloads."

**Fix:**
1. Update package doc (lines 1-3):
   - **Before:** `"Only Commands write to the store, never pane Update() or View() directly."`
   - **After:** `"Only the root app.Update() writes to the store via data-carrying Msg payloads, never Commands, pane Update(), or View() directly."`

2. Update Store struct doc (lines 40-42):
   - **Before:** `"only tea.Cmd callbacks write to it"`
   - **After:** `"only the root app.Update() writes to it via Msg payloads"`

### Task 2: Remove dead unmarshalJSON helper

**Problem:** `internal/api/models.go` contains an `unmarshalJSON` helper function (lines 16-20) that is no longer used. All types with custom unmarshalers moved to `internal/domain/` during Feature 29.

**Fix:**
1. Remove the `unmarshalJSON` function from `internal/api/models.go` (lines 16-20)
2. Remove the `"encoding/json"` import if it becomes unused after removal
3. Verify: `grep -r 'unmarshalJSON' internal/` should show ZERO matches after removal

### Task 3: Initialize statsFetchedAt map in New()

**Problem:** `internal/state/store.go` declares `statsFetchedAt map[string]time.Time` (line 61) but never initializes it in `New()` (lines 112-116). The current code relies on lazy initialization in `SetTopTracks()` and `SetTopArtists()` which is correct but fragile -- any future code that reads the map before the first Set call would panic.

**Fix:**
1. In `New()` (store.go line 112-116), add `statsFetchedAt: make(map[string]time.Time)`:
   ```go
   func New() *Store {
       return &Store{
           netLog:         NewNetLog(),
           statsFetchedAt: make(map[string]time.Time),
       }
   }
   ```
2. Remove the lazy initialization guards in `SetTopTracks()` and `SetTopArtists()` (the `if s.statsFetchedAt == nil { s.statsFetchedAt = make(...) }` blocks)

### Task 4: Mark resolved issues and update issues.md

**Problem:** Issue #5 from PR #34 ("buildFetchDevicesCmd error fallthrough") was found to be false -- the code already returns early with the error correctly.

**Fix:** Update `docs/issues.md` to mark resolved issues from PR #34 and PR #37.

### Verification

```bash
# Stale docs fixed
grep -n 'Only Commands write' internal/state/store.go
# Expected: ZERO matches

# Dead function removed
grep -r 'unmarshalJSON' internal/api/models.go
# Expected: ZERO matches

# statsFetchedAt initialized
grep -n 'statsFetchedAt: make' internal/state/store.go
# Expected: 1 match in New()

make ci
# Expected: Full pass
```

## Acceptance Criteria
- [ ] Store.go doc comments reflect "only Update() writes via Msg payloads" rule
- [ ] Dead `unmarshalJSON` helper removed from `internal/api/models.go`
- [ ] `statsFetchedAt` map initialized in `New()` with `make()`
- [ ] Lazy initialization guards removed from `SetTopTracks()` and `SetTopArtists()`
- [ ] Resolved issues marked in `docs/issues.md`
- [ ] `make ci` passes

## Tasks
- [ ] Fix stale store.go documentation to reflect Elm purity rule
      - test: None needed (doc-only change)
- [ ] Remove dead `unmarshalJSON` helper from `internal/api/models.go`
      - test: `make ci` to verify no compilation errors
- [ ] Initialize `statsFetchedAt` map in `New()` and remove lazy init guards
      - test: verify `New()` returns a Store with non-nil statsFetchedAt; verify `StatsStale()` works on a fresh Store (no panic)
- [ ] Mark resolved issues in `docs/issues.md`
      - test: None needed (documentation-only)
