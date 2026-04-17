---
title: "BasePane Embedding, RebuildTableTheme & Auth Client Fix"
feature: 22-developer-foundations
status: done
---

## Background

The 2026-04-08 audit identified three patterns that repeat across panes with no shared
abstraction:

1. **Repeated pane boilerplate** — all 8 Page A panes declare the same five fields
   (`store`, `theme`, `focused`, `width`, `height`) and implement the same four trivial
   methods (`IsFocused()`, `SetFocused()`, `SetSize()`, `HasActiveFilter()`). 40 lines
   duplicated × 8 = 320 lines of noise.
2. **Repeated `SetTheme()` body** — each of the 8 panes contains the same 10-line
   pattern: rebuild column defs with new theme colors, call `NewTable()`, copy rows,
   re-apply focus, create new `Filter`. No shared helper exists.
3. **`http.DefaultClient` bypass in `auth.go`** — `postTokenRequest()` calls
   `http.DefaultClient.Do(req)` instead of the injected `*http.Client`. This makes
   the token exchange path untestable via `httptest.NewServer` and is inconsistent
   with every other HTTP call in the codebase.

All changes are structural. No behaviour changes, no new features.

**Source:** `docs/code-audit/code-audit-design.md` §1–§3,
`docs/code-audit/phase3-design-patterns.md`

**Depends on:** Story 110 (pane constructors already accept `state.StateReader`, which
`BasePane` will embed as its `store` field type).

---

## Design

### Task 1 — `BasePane` embedded struct

**File to create:** `internal/ui/panes/base_pane.go`

```go
// BasePane holds the fields and trivial Pane interface methods shared by all 8
// table-based Page A panes. Embed this struct in each pane to eliminate repeated
// declarations.
//
// What BasePane provides: store, theme, focused, width, height fields;
// IsFocused(), SetFocused(), SetSize(), HasActiveFilter() implementations.
//
// What BasePane does NOT provide: Table or Filter fields (column layouts differ
// per pane); SetTheme() (table rebuild is pane-specific); ID(), Title(),
// ToggleKey(), Actions(); Init(), Update(), View().
type BasePane struct {
    store   state.StateReader
    theme   theme.Theme
    focused bool
    width   int
    height  int
}

func (b *BasePane) IsFocused() bool       { return b.focused }
func (b *BasePane) SetFocused(f bool)     { b.focused = f }
func (b *BasePane) SetSize(w, h int)      { b.width = w; b.height = h }
func (b *BasePane) HasActiveFilter() bool { return false }
```

`SetFocused` and `SetSize` are the base implementations. Panes that also need
to forward focus/size to their table must **override** these (calling
`b.BasePane.SetFocused(f)` first). Document this in the comment.

`HasActiveFilter` returns `false` — panes with a filter override it:
```go
func (p *MyPane) HasActiveFilter() bool { return p.filter.IsActive() }
```

### Task 2 — Embed `BasePane` in all 8 panes

**Files to modify (all in `internal/ui/panes/`):**
`nowplaying.go`, `queue.go`, `playlists_pane.go`, `albums_pane.go`,
`likedsongs_pane.go`, `recentlyplayed_pane.go`, `toptracks_pane.go`, `topartists_pane.go`

**Do one pane at a time, run tests between each, then commit.**
Recommended order (simplest to most complex):
`toptracks_pane.go` → `topartists_pane.go` → `recentlyplayed_pane.go` →
`likedsongs_pane.go` → `queue.go` → `nowplaying.go` → `albums_pane.go` → `playlists_pane.go`

For each pane:

**Step A — Update the struct:**
Remove `store`, `theme`, `focused`, `width`, `height` fields.
Add `BasePane` as the first embedded field.

```go
// Before:
type QueuePane struct {
    store   state.StateReader
    theme   theme.Theme
    focused bool
    width   int
    height  int
    table   *components.Table
    filter  *components.Filter
}

// After:
type QueuePane struct {
    BasePane
    table  *components.Table
    filter *components.Filter
}
```

**Step B — Update the constructor:**
Change named field assignments to initialize via `BasePane{}`:

```go
// Before:
p := &QueuePane{store: store, theme: th, focused: focused, table: t, filter: f}

// After:
p := &QueuePane{BasePane: BasePane{store: store, theme: th, focused: focused}, table: t, filter: f}
```

**Step C — Delete now-duplicate methods:**
- Delete `IsFocused()` — BasePane provides it.
- Delete `HasActiveFilter()` — **only if** the pane's version simply returned
  `p.filter.IsActive()` keep it as an override; if it returned `false`, delete it.
- For `SetFocused()` — if it also calls `table.SetFocused()`, **keep it** as an
  override:
  ```go
  func (p *QueuePane) SetFocused(f bool) {
      p.BasePane.SetFocused(f)
      p.table.SetFocused(f && !p.filter.IsActive())
  }
  ```
- For `SetSize()` — if it also calls `table.SetSize()`, **keep it** as an override:
  ```go
  func (p *QueuePane) SetSize(w, h int) {
      p.BasePane.SetSize(w, h)
      p.table.SetSize(w, h)
  }
  ```

**Step D — Verify references:**
With embedding, `p.store`, `p.theme`, etc. still resolve directly through promotion.
No method body changes are needed unless there is a compile error.

After each pane: `go build ./internal/ui/panes/... && go test ./internal/ui/panes/... -race -count=1`

Verify when all 8 are done:
```bash
grep -l "BasePane" internal/ui/panes/*.go | grep -v "_test.go" | grep -v "base_pane.go"
# Expected: 8 files
```

### Task 3 — `RebuildTableTheme` helper

**File to create:** `internal/ui/components/table_theme.go`

Read one pane's current `SetTheme()` before writing to confirm the exact method names
(`SetRows`, `Rows()`, `SetFocused`, `NewFilter`):
```bash
grep -A 25 "func.*SetTheme" internal/ui/panes/queue.go
```

```go
// RebuildTableTheme creates a new Table with updated theme colors and re-applies
// the existing rows from the old table. Called by pane SetTheme() to avoid
// repeating the same 10-line pattern across all 8 panes.
//
// Usage:
//
//   func (p *MyPane) SetTheme(th theme.Theme) {
//       p.theme = th
//       cols := []ColumnDef{
//           {Key: "index", Header: "#",     FlexFactor: 1, Color: th.ColumnIndex()},
//           {Key: "track", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
//       }
//       p.table, p.filter = components.RebuildTableTheme(th, cols, p.table.Rows(), p.focused && !p.filter.IsActive())
//   }
func RebuildTableTheme(
    th theme.Theme,
    cols []ColumnDef,
    rows []table.Row,
    focused bool,
) (*Table, *Filter) {
    t := NewTable(TableConfig{Columns: cols, Theme: th, PlayingIndex: -1, ShowHeader: true})
    t.SetRows(rows)
    t.SetFocused(focused)
    return t, NewFilter(th)
}
```

Adjust method names if the actual `Table` and `Filter` types use different names.

### Task 4 — Use `RebuildTableTheme` in all 8 pane `SetTheme()` methods

**Files:** same 8 pane files from Task 2.

For each pane, replace the 10-line `SetTheme()` body with:

```go
func (p *QueuePane) SetTheme(th theme.Theme) {
    p.theme = th
    cols := []components.ColumnDef{
        {Key: "index",    Header: "#",        FlexFactor: 1, Color: th.ColumnIndex()},
        {Key: "track",    Header: "Track",    FlexFactor: 9, Color: th.ColumnPrimary()},
        {Key: "artist",   Header: "Artist",   FlexFactor: 7, Color: th.ColumnSecondary()},
        {Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
    }
    p.table, p.filter = components.RebuildTableTheme(th, cols, p.table.Rows(), p.focused && !p.filter.IsActive())
}
```

Column defs differ per pane — copy them from the pane's constructor, substituting
`th.Column*()` tokens for the new theme.

Verify: `go test ./internal/ui/panes/... -race -count=1`

### Task 5 — Fix `postTokenRequest` in `auth.go`

**File:** `internal/api/auth.go`

Read the function and its call sites first:
```bash
grep -n "^func.*postTokenRequest\|postTokenRequest(" internal/api/auth.go
```

**Change the signature** to accept an injected `*http.Client`:
```go
// Before:
func postTokenRequest(ctx context.Context, endpoint string, formData url.Values) (TokenPair, error)

// After:
func postTokenRequest(ctx context.Context, httpClient *http.Client, endpoint string, formData url.Values) (TokenPair, error)
```

Inside the function, replace:
```go
resp, err := http.DefaultClient.Do(req)
```
with:
```go
resp, err := httpClient.Do(req)
```

**Update all call sites** — `Exchange` and `Refresh` (or equivalent functions).
Locate how these callers access their HTTP client (usually a field on a struct or a
parameter), and pass it through. Read the actual call site code before editing.

### Task 6 — Test the injected client

**File:** `internal/api/auth_test.go`

Add a test that proves `postTokenRequest` reaches the injected client rather than
`http.DefaultClient`:

```go
func TestPostTokenRequest_UsesInjectedClient(t *testing.T) {
    called := false
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        called = true
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"access_token":"a","refresh_token":"r","expires_in":3600}`))
    }))
    defer srv.Close()

    pair, err := postTokenRequest(
        context.Background(),
        &http.Client{},
        srv.URL+"/token",
        url.Values{"grant_type": {"client_credentials"}},
    )

    require.NoError(t, err)
    assert.True(t, called, "test server was not reached — http.DefaultClient still used")
    assert.Equal(t, "a", pair.AccessToken)
}
```

Note: `postTokenRequest` is package-private. The test must be in `package api`
(not `package api_test`). Confirm the existing test file's package declaration.

---

## Acceptance Criteria

- [ ] `internal/ui/panes/base_pane.go` exists and compiles
- [ ] All 8 pane structs embed `BasePane`; none re-declare `store`, `theme`, `focused`,
      `width`, or `height` fields
- [ ] `grep -l "BasePane" internal/ui/panes/*.go | grep -v _test.go | grep -v base_pane.go`
      → 8 files
- [ ] `internal/ui/components/table_theme.go` exists with `RebuildTableTheme`
- [ ] All 8 pane `SetTheme()` methods use `components.RebuildTableTheme`
- [ ] `grep "DefaultClient" internal/api/auth.go` → zero hits
- [ ] `TestPostTokenRequest_UsesInjectedClient` passes
- [ ] `go test ./internal/ui/panes/... -race -count=1` passes (no test changes required)
- [ ] `make ci` passes
- [ ] Remove the `http.DefaultClient` known-issue note from `docs/ARCHITECTURE.md`
      (added in Story 109 Task 4)

## Tasks

- [ ] Create `internal/ui/panes/base_pane.go` with `BasePane` struct and four methods
      - test: `go build ./internal/ui/panes/...` clean
- [ ] Embed `BasePane` in all 8 pane structs (one pane at a time, test after each)
      - test after all 8: `grep -l "BasePane" internal/ui/panes/*.go | grep -v _test.go | grep -v base_pane.go` → 8 files;
        `go test ./internal/ui/panes/... -race -count=1` passes
- [ ] Create `internal/ui/components/table_theme.go` with `RebuildTableTheme`
      - test: `go build ./internal/ui/components/...` clean
- [ ] Use `RebuildTableTheme` in all 8 pane `SetTheme()` methods
      - test: `go test ./internal/ui/panes/... -race -count=1` passes
- [ ] Fix `postTokenRequest` to use injected `*http.Client`
      - test: `grep "DefaultClient" internal/api/auth.go` → zero hits; `go build ./internal/api/...` clean
- [ ] Add `TestPostTokenRequest_UsesInjectedClient` to `auth_test.go`
      - test: `go test ./internal/api/... -run TestPostTokenRequest -race -count=1 -v` → PASS
- [ ] Remove `http.DefaultClient` known-issue note from `docs/ARCHITECTURE.md`
      - test: note is gone; `make ci` passes
