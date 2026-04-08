# Phase 3 — Design Pattern Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate pane boilerplate via `BasePane`, extract `RebuildTableTheme` helper, and fix the `http.DefaultClient` bypass in auth — all with zero behaviour change.

**Architecture:** `BasePane` is embedded in 8 panes; shared field declarations and trivial method implementations are deleted from each pane. `RebuildTableTheme` lives in `internal/ui/components/`. The auth fix changes `postTokenRequest` to accept an injected `*http.Client`.

**Tech Stack:** Go 1.22+, `internal/ui/panes`, `internal/ui/components`, `internal/api`.

**Prerequisite:** Phase 2 branch merged to `main` before starting.

---

## File Map

| Action | Path | Purpose |
|--------|------|---------|
| Create | `internal/ui/panes/base_pane.go` | Shared struct + trivial Pane methods |
| Create | `internal/ui/components/table_theme.go` | `RebuildTableTheme` helper |
| Modify | `internal/ui/panes/nowplaying.go` | Embed BasePane, delete duplicate fields/methods |
| Modify | `internal/ui/panes/queue.go` | Embed BasePane, delete duplicate fields/methods |
| Modify | `internal/ui/panes/playlists_pane.go` | Embed BasePane, delete duplicate fields/methods |
| Modify | `internal/ui/panes/albums_pane.go` | Embed BasePane, delete duplicate fields/methods |
| Modify | `internal/ui/panes/likedsongs_pane.go` | Embed BasePane, delete duplicate fields/methods |
| Modify | `internal/ui/panes/recentlyplayed_pane.go` | Embed BasePane, delete duplicate fields/methods |
| Modify | `internal/ui/panes/toptracks_pane.go` | Embed BasePane, delete duplicate fields/methods |
| Modify | `internal/ui/panes/topartists_pane.go` | Embed BasePane, delete duplicate fields/methods |
| Modify | `internal/ui/panes/*.go` (8 panes) | Use RebuildTableTheme in SetTheme() |
| Modify | `internal/api/auth.go` | postTokenRequest accepts injected *http.Client |
| Modify | `internal/api/auth_test.go` | Add test for injected client in postTokenRequest |

---

### Task 1: Create branch

- [ ] **Step 1: Create and switch to feature branch**

```bash
git checkout main && git pull origin main
git checkout -b refactor/audit-phase3-patterns
```

---

### Task 2: Create BasePane

**Files:**
- Create: `internal/ui/panes/base_pane.go`

Before writing, read 2-3 pane files to confirm the exact field names and types:

```bash
head -60 internal/ui/panes/queue.go
head -60 internal/ui/panes/toptracks_pane.go
head -60 internal/ui/panes/likedsongs_pane.go
```

- [ ] **Step 1: Write BasePane**

```go
// Package panes — BasePane holds the fields and trivial Pane interface methods
// shared by all 8 table-based Page A panes. Embed this struct in each pane to
// eliminate repeated declarations.
//
// What BasePane provides:
//   - store, theme, focused, width, height fields
//   - IsFocused(), SetFocused(), SetSize() implementations
//   - HasActiveFilter() returning false (override in panes with a filter)
//
// What BasePane does NOT provide:
//   - Table or Filter fields — column layouts differ per pane
//   - SetTheme() — table rebuild is pane-specific
//   - ID(), Title(), ToggleKey(), Actions() — unique per pane
//   - Init(), Update(), View() — unique per pane
package panes

import (
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// BasePane is embedded in all 8 Page A panes. Do not use it directly as a layout.Pane —
// it does not implement ID(), Title(), ToggleKey(), Actions(), Init(), Update(), or View().
type BasePane struct {
	store   state.StateReader
	theme   theme.Theme
	focused bool
	width   int
	height  int
}

// IsFocused returns whether this pane currently has keyboard focus.
func (b *BasePane) IsFocused() bool { return b.focused }

// SetFocused updates the keyboard focus state.
// Panes that own a table or filter must also call table.SetFocused() — do this
// by overriding SetFocused() in the embedding pane:
//
//	func (p *MyPane) SetFocused(f bool) {
//	    p.BasePane.SetFocused(f)
//	    p.table.SetFocused(f && !p.filter.IsActive())
//	}
func (b *BasePane) SetFocused(f bool) { b.focused = f }

// SetSize stores the content area dimensions passed by the layout manager.
// Panes that own a table must also call table.SetSize() — override SetSize()
// in the embedding pane:
//
//	func (p *MyPane) SetSize(w, h int) {
//	    p.BasePane.SetSize(w, h)
//	    p.table.SetSize(w, h)
//	}
func (b *BasePane) SetSize(w, h int) { b.width = w; b.height = h }

// HasActiveFilter returns false. Panes with an active filter must override this:
//
//	func (p *MyPane) HasActiveFilter() bool { return p.filter.IsActive() }
func (b *BasePane) HasActiveFilter() bool { return false }
```

- [ ] **Step 2: Build**

```bash
go build ./internal/ui/panes/...
```

Expected: no errors. BasePane adds no interface implementation yet — it's just a struct.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/panes/base_pane.go
git commit -m "refactor(panes): add BasePane with shared fields and trivial Pane methods"
```

---

### Task 3: Embed BasePane in each pane — one pane at a time

Do one pane, run tests, then repeat. Start with the simplest pane (no sub-views):

**Order:** `toptracks_pane.go` → `topartists_pane.go` → `recentlyplayed_pane.go` → `likedsongs_pane.go` → `queue.go` → `nowplaying.go` → `albums_pane.go` → `playlists_pane.go`

For **each** pane file:

- [ ] **Step 1: Add BasePane embedding to the struct**

Find the struct definition. Remove the fields that BasePane now provides:
`store`, `theme`, `focused`, `width`, `height`.

Add `BasePane` as an embedded field:

```go
// Before (example: TopTracksPane):
type TopTracksPane struct {
    store   state.StateReader
    theme   theme.Theme
    focused bool
    width   int
    height  int
    table   *components.Table
    filter  *components.Filter
    // pane-specific fields...
}

// After:
type TopTracksPane struct {
    BasePane               // provides store, theme, focused, width, height + IsFocused, SetFocused, SetSize, HasActiveFilter
    table   *components.Table
    filter  *components.Filter
    // pane-specific fields...
}
```

- [ ] **Step 2: Update the constructor**

Change field assignments from named fields to the embedded struct:

```go
// Before:
p := &TopTracksPane{
    store:   store,
    theme:   th,
    focused: focused,
    table:   t,
    filter:  components.NewFilter(th),
}

// After:
p := &TopTracksPane{
    BasePane: BasePane{store: store, theme: th, focused: focused},
    table:    t,
    filter:   components.NewFilter(th),
}
```

- [ ] **Step 3: Delete the now-duplicate methods**

Remove `IsFocused()` and `HasActiveFilter()` (if present) from the pane — BasePane provides them.

For `SetFocused()`: if the pane's version only sets `focused` with no other logic, delete it.
If it also calls `table.SetFocused()`, **keep it** as an override:

```go
func (p *TopTracksPane) SetFocused(f bool) {
    p.BasePane.SetFocused(f)
    p.table.SetFocused(f && !p.filter.IsActive())
}
```

For `SetSize()`: if the pane's version also calls `table.SetSize()`, **keep it** as an override:

```go
func (p *TopTracksPane) SetSize(w, h int) {
    p.BasePane.SetSize(w, h)
    p.table.SetSize(w, h)
}
```

- [ ] **Step 4: Fix all references to removed fields**

In the pane's methods, change direct field access to go through the embedded struct.
With embedding, `p.store` still works directly — no change needed.

If there is a compile error like `p.focused undefined`, it means the embedding wasn't
applied correctly. Double-check Step 1.

- [ ] **Step 5: Build and test after each pane**

```bash
go build ./internal/ui/panes/...
go test ./internal/ui/panes/... -race -count=1
```

Expected: no errors, all tests pass.

- [ ] **Step 6: Commit after each pane**

```bash
git add internal/ui/panes/<panefile>.go
git commit -m "refactor(panes): embed BasePane in <PaneName>"
```

---

### Task 4: Create RebuildTableTheme helper

**Files:**
- Create: `internal/ui/components/table_theme.go`

First, read a pane's `SetTheme()` to understand the exact pattern:

```bash
grep -A 20 "func.*SetTheme" internal/ui/panes/queue.go
```

- [ ] **Step 1: Write the helper**

```go
// Package components — RebuildTableTheme reconstructs a Table with updated theme
// colours and re-applies existing row data. Called by pane SetTheme() methods to
// avoid repeating the same 10-line pattern across all 8 panes.
package components

import (
	table "github.com/evertras/bubble-table/table"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// RebuildTableTheme creates a new Table using the given theme and columns, resets
// the filter, and re-applies the rows from the old table.
//
// Usage in a pane's SetTheme():
//
//	func (p *MyPane) SetTheme(th theme.Theme) {
//	    p.theme = th
//	    cols := []ColumnDef{
//	        {Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
//	        {Key: "track", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
//	    }
//	    p.table, p.filter = RebuildTableTheme(th, cols, p.table.Rows(), p.focused)
//	}
func RebuildTableTheme(
	th theme.Theme,
	cols []ColumnDef,
	rows []table.Row,
	focused bool,
) (*Table, *Filter) {
	t := NewTable(TableConfig{
		Columns:      cols,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})
	t.SetRows(rows)
	t.SetFocused(focused)
	f := NewFilter(th)
	return t, f
}
```

> **Note:** Check the actual `Table` and `Filter` types in `internal/ui/components/table.go`
> and `filter.go` to confirm the method names (`SetRows`, `Rows`, `SetFocused`, `NewFilter`).
> Adjust if they differ.

- [ ] **Step 2: Build**

```bash
go build ./internal/ui/components/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/components/table_theme.go
git commit -m "refactor(components): add RebuildTableTheme helper for pane theme switching"
```

---

### Task 5: Use RebuildTableTheme in each pane's SetTheme()

For **each** of the 8 panes, update `SetTheme()`:

- [ ] **Step 1: Read the current SetTheme()**

```bash
grep -A 25 "func.*SetTheme" internal/ui/panes/queue.go
```

The current pattern typically:
1. Sets `p.theme = th`
2. Rebuilds column defs with new theme colours
3. Calls `NewTable(...)` with new columns
4. Calls `t.SetRows(p.table.Rows())`
5. Calls `t.SetFocused(...)`
6. Creates a new `components.NewFilter(th)`
7. Assigns `p.table = t` and `p.filter = f`

- [ ] **Step 2: Replace with RebuildTableTheme**

```go
// Before (example: QueuePane):
func (q *QueuePane) SetTheme(th theme.Theme) {
    q.theme = th
    columns := []components.ColumnDef{
        {Key: "index",    Header: "#",        FlexFactor: 1, Color: th.ColumnIndex()},
        {Key: "track",    Header: "Track",    FlexFactor: 9, Color: th.ColumnPrimary()},
        {Key: "artist",   Header: "Artist",   FlexFactor: 7, Color: th.ColumnSecondary()},
        {Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
    }
    t := components.NewTable(components.TableConfig{
        Columns: columns, Theme: th, PlayingIndex: -1, ShowHeader: true,
    })
    t.SetRows(q.table.Rows())
    t.SetFocused(q.focused && !q.filter.IsActive())
    q.table = t
    q.filter = components.NewFilter(th)
}

// After:
func (q *QueuePane) SetTheme(th theme.Theme) {
    q.theme = th
    cols := []components.ColumnDef{
        {Key: "index",    Header: "#",        FlexFactor: 1, Color: th.ColumnIndex()},
        {Key: "track",    Header: "Track",    FlexFactor: 9, Color: th.ColumnPrimary()},
        {Key: "artist",   Header: "Artist",   FlexFactor: 7, Color: th.ColumnSecondary()},
        {Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
    }
    q.table, q.filter = components.RebuildTableTheme(th, cols, q.table.Rows(), q.focused && !q.filter.IsActive())
}
```

Apply to all 8 panes.

- [ ] **Step 3: Build and test**

```bash
go build ./internal/ui/panes/...
go test ./internal/ui/panes/... -race -count=1
```

Expected: no errors, all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/panes/
git commit -m "refactor(panes): use components.RebuildTableTheme in all 8 pane SetTheme() methods"
```

---

### Task 6: Fix http.DefaultClient in auth.go

**Files:**
- Modify: `internal/api/auth.go`
- Modify: `internal/api/auth_test.go`

The issue: `postTokenRequest` (line ~272) uses `http.DefaultClient.Do(req)` instead of
an injected client. This prevents test-time HTTP interception via `httptest.NewServer`.

- [ ] **Step 1: Read the full function and its two call sites**

```bash
sed -n '170,220p' internal/api/auth.go  # lines around the two call sites
sed -n '271,310p' internal/api/auth.go  # postTokenRequest itself
```

- [ ] **Step 2: Add httpClient parameter to postTokenRequest**

Change the signature from:
```go
func postTokenRequest(ctx context.Context, endpoint string, formData url.Values) (TokenPair, error) {
```

To:
```go
func postTokenRequest(ctx context.Context, httpClient *http.Client, endpoint string, formData url.Values) (TokenPair, error) {
```

Inside the function, change:
```go
resp, err := http.DefaultClient.Do(req)
```

To:
```go
resp, err := httpClient.Do(req)
```

- [ ] **Step 3: Update the two call sites**

Find every call to `postTokenRequest(ctx, endpoint, formData)` in `auth.go`.

The function is called from `Exchange` and `Refresh` (around lines 181 and 216).
Both callers need to pass the injected HTTP client.

Look at how those functions receive the client. If `Exchange` and `Refresh` are methods
on a struct that holds `*http.Client`, pass it directly:

```go
// If Exchange is a method on Auth struct with a.http *http.Client:
pair, err := postTokenRequest(ctx, a.http, endpoint, formData)
```

If `Exchange` and `Refresh` are standalone functions receiving `*http.Client` as a
parameter, add `httpClient *http.Client` to their call chain and pass it through.

Read the actual function signatures first:
```bash
grep -n "^func Exchange\|^func Refresh\|^func.*Exchange\|^func.*Refresh" internal/api/auth.go
```

Adjust based on what you find.

- [ ] **Step 4: Build**

```bash
go build ./internal/api/...
```

Expected: no errors.

- [ ] **Step 5: Run existing auth tests**

```bash
go test ./internal/api/... -run TestAuth -race -count=1 -v
```

Expected: all existing auth tests pass.

- [ ] **Step 6: Commit the fix**

```bash
git add internal/api/auth.go
git commit -m "fix(auth): postTokenRequest uses injected *http.Client instead of http.DefaultClient"
```

---

### Task 7: Add auth test for injected client

**Files:**
- Modify: `internal/api/auth_test.go`

- [ ] **Step 1: Write a test that verifies the injected client is used**

Add a test that:
1. Starts an `httptest.NewServer` that records whether it was called
2. Creates an `*http.Client` that points to the test server
3. Calls the token exchange/refresh function with that client
4. Asserts the test server was reached (not `http.DefaultClient`)

```go
func TestPostTokenRequest_UsesInjectedClient(t *testing.T) {
    called := false
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        called = true
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{
            "access_token":"new-access",
            "refresh_token":"new-refresh",
            "expires_in": 3600
        }`))
    }))
    defer srv.Close()

    client := &http.Client{}
    // point the client at our test server by using srv.URL as the token endpoint
    pair, err := postTokenRequest(
        context.Background(),
        client,
        srv.URL+"/api/token",
        url.Values{"grant_type": {"client_credentials"}},
    )

    require.NoError(t, err)
    assert.True(t, called, "expected test server to be called — injected client not used")
    assert.Equal(t, "new-access", pair.AccessToken)
}
```

> **Note:** `postTokenRequest` is package-private (`lowercase`). This test must be in
> `package api` (not `package api_test`) to access it. Check the existing test file's
> package declaration to confirm.

- [ ] **Step 2: Run the new test**

```bash
go test ./internal/api/... -run TestPostTokenRequest -race -count=1 -v
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/api/auth_test.go
git commit -m "test(auth): verify postTokenRequest uses injected *http.Client"
```

---

### Task 8: Final verification and PR

- [ ] **Step 1: Full CI gate**

```bash
make ci
```

Expected: all steps pass.

- [ ] **Step 2: Confirm no *state.Store in pane structs**

```bash
grep -rn "\*state\.Store" internal/ui/panes/ | grep -v "_test.go"
```

Expected: no output.

- [ ] **Step 3: Confirm BasePane is embedded in all 8 panes**

```bash
grep -l "BasePane" internal/ui/panes/*.go | grep -v "_test.go" | grep -v "base_pane.go"
```

Expected: 8 files listed.

- [ ] **Step 4: Confirm http.DefaultClient is gone from auth.go**

```bash
grep "DefaultClient" internal/api/auth.go
```

Expected: no output.

- [ ] **Step 5: Integration tests**

```bash
make test-integration
```

Expected: all integration tests pass.

- [ ] **Step 6: Push and open PR**

```bash
git push origin refactor/audit-phase3-patterns
```

Open PR with title: `refactor: phase 3 — BasePane embedding, RebuildTableTheme, fix auth DefaultClient`

Body:
```
## Changes

- Add `BasePane` embedded struct — eliminates repeated field declarations and trivial
  method implementations across all 8 Page A panes
- Add `components.RebuildTableTheme()` helper — each pane's SetTheme() is now 3 lines
  instead of 10 identical lines
- Fix `postTokenRequest` in auth.go — now uses injected *http.Client instead of
  http.DefaultClient; add test to verify
- All 8 panes now accept `state.StateReader` (from Phase 2) via embedded BasePane

## No Behaviour Changes

All existing tests pass unchanged. This is structural improvement only.

## Test Summary

- make ci passes
- make test-integration passes
- grep "\*state.Store" internal/ui/panes/ → zero hits
- grep "DefaultClient" internal/api/auth.go → zero hits
- grep "BasePane" internal/ui/panes/*.go → 8 files
```
