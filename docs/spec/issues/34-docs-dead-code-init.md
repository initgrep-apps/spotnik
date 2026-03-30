# Feature 34 — Docs, Dead Code & Defensive Init

> **Feature:** Fix stale documentation, remove dead code, and add defensive initialization.
> Quick-win cleanup from PR review issues.

## Context

PR reviews of features 29-33 identified several documentation inaccuracies, dead code,
and a fragile initialization pattern. These are all non-behavioral changes that improve
code quality and correctness of documentation.

**Source:** `docs/issues.md` — PR #34 issues 1-3, 5; PR #37 issue 9

**Depends on:** Nothing

---

## Task 1: Fix stale store.go documentation

**Problem:** Two doc comments in `internal/state/store.go` reference the old "Commands write
to the store" pattern. Since Feature 29, the rule is "only Update() writes to the store
via data-carrying Msg payloads."

**Fix:**

1. Update package doc (lines 1-3):
   - **Before:** `"Only Commands write to the store, never pane Update() or View() directly."`
   - **After:** `"Only the root app.Update() writes to the store via data-carrying Msg payloads, never Commands, pane Update(), or View() directly."`

2. Update Store struct doc (lines 40-42):
   - **Before:** `"only tea.Cmd callbacks write to it"`
   - **After:** `"only the root app.Update() writes to it via Msg payloads"`

**Files:**
- Modify: `internal/state/store.go` — lines 1-3 and 40-42

**Tests:**
- None needed (doc-only change)

**Commit:** `docs(state): fix stale store.go comments to reflect Elm purity rule`

---

## Task 2: Remove dead unmarshalJSON helper

**Problem:** `internal/api/models.go` contains an `unmarshalJSON` helper function (lines 16-20)
that is no longer used. All types with custom unmarshalers moved to `internal/domain/`
during Feature 29.

**Fix:**

1. Remove the `unmarshalJSON` function from `internal/api/models.go` (lines 16-20)
2. Remove the `"encoding/json"` import if it becomes unused after removal
3. Verify: `grep -r 'unmarshalJSON' internal/` should show ZERO matches after removal

**Files:**
- Modify: `internal/api/models.go` — remove lines 16-20 and unused import

**Tests:**
- Run `make ci` to verify no compilation errors (nothing references this function)

**Commit:** `chore(api): remove dead unmarshalJSON helper`

---

## Task 3: Initialize statsFetchedAt map in New()

**Problem:** `internal/state/store.go` declares `statsFetchedAt map[string]time.Time` (line 61)
but never initializes it in `New()` (lines 112-116). The current code relies on lazy
initialization in `SetTopTracks()` and `SetTopArtists()` which is correct but fragile —
any future code that reads the map before the first Set call would panic.

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

2. Remove the lazy initialization guards in `SetTopTracks()` and `SetTopArtists()`
   (the `if s.statsFetchedAt == nil { s.statsFetchedAt = make(...) }` blocks)

**Files:**
- Modify: `internal/state/store.go` — New(), SetTopTracks(), SetTopArtists()

**Tests:**
- Unit: verify `New()` returns a Store with non-nil statsFetchedAt
- Unit: verify `StatsStale()` works on a fresh Store (no panic)

**Commit:** `fix(state): initialize statsFetchedAt map in New()`

---

## Task 4: Mark resolved issue and update issues.md

**Problem:** Issue #5 from PR #34 ("buildFetchDevicesCmd error fallthrough") was found to be
false — the code already returns early with the error correctly.

**Fix:**

1. Update `docs/issues.md` PR #34 section:
   - Change `- [ ] **buildFetchDevicesCmd error fallthrough**` to `- [x] **buildFetchDevicesCmd error fallthrough** — Verified: already handles errors correctly with early return`
   - Also mark issues 1, 2, 3 from PR #34 as resolved (done in this feature)
   - Mark issue 9 from PR #37 as resolved (done in this feature)

**Files:**
- Modify: `docs/issues.md`

**Tests:**
- None needed (documentation-only)

**Commit:** `docs: mark resolved issues in issues.md`

---

## Verification

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

---

*Depends on: Nothing*
*Blocks: Feature 35*
