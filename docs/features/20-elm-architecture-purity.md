# Feature 20 — Elm Architecture Purity

> **Refactoring:** Fix three violations of the Elm Architecture's unidirectional data flow:
> store mutations in command builders, direct store writes from a pane, and hardcoded
> theme values.

## Context

The Elm Architecture requires: side effects only via Commands, store writes only from
`Update()` or inside command closures, and all styling via theme tokens. Three violations
were found in the architecture review.

---

## Task 1: Move store mutations out of buildSearchCmd

**Problem:** `buildSearchCmd()` in `app.go` (lines 1125-1126) calls
`store.SetSearchQuery(query)` and `store.SetSearchLoading(true)` synchronously
*before* returning the command closure. This breaks the invariant that store writes
happen only inside commands (the returned `func() tea.Msg`).

**Current code pattern:**
```go
func (a *App) buildSearchCmd(query string) tea.Cmd {
    store := a.store
    store.SetSearchQuery(query)      // ← violation: synchronous write
    store.SetSearchLoading(true)     // ← violation: synchronous write
    return func() tea.Msg {
        // ... API call
    }
}
```

**Fix:** Move both writes into the `Update()` handler for `SearchRequestMsg`, before
calling `buildSearchCmd()`. The command builder should only build the closure.

```go
// In Update(), before building the command:
case SearchRequestMsg:
    a.store.SetSearchQuery(msg.Query)
    a.store.SetSearchLoading(true)
    return a, a.buildSearchCmd(msg.Query)
```

Then remove the two store writes from `buildSearchCmd()`.

**Files:**
- `internal/app/app.go` — Move writes to Update handler, clean buildSearchCmd

**Tests:**
- Unit test: verify `buildSearchCmd` does not call `SetSearchQuery` or `SetSearchLoading`
  (if MockClient is not available, verify by inspecting the code path)
- Existing search tests should continue to pass

---

## Task 2: Emit SearchClearedMsg instead of direct store write

**Problem:** `SearchOverlay.Update()` in `search.go` (lines 204-209) directly writes to
the store on Ctrl+U:
```go
case tea.KeyCtrlU:
    o.store.SetSearchResults(nil)    // ← violation
    o.store.SetSearchQuery("")       // ← violation
```

This is the only pane that directly writes to the store. Per architecture rules, panes
must emit messages and let the root model handle store writes.

**Fix:**
1. Define a new `SearchClearedMsg` struct in `internal/ui/panes/messages.go`
2. In `SearchOverlay.Update()` for Ctrl+U, emit `SearchClearedMsg` as a command instead
   of writing to the store directly. Still clear the local `o.input` value.
3. In `app.go` `Update()`, handle `SearchClearedMsg` by calling
   `store.SetSearchResults(nil)` and `store.SetSearchQuery("")`

**Files:**
- `internal/ui/panes/search.go` — Replace direct store writes with message emission
- `internal/ui/panes/messages.go` — Add `SearchClearedMsg` struct
- `internal/app/app.go` — Add handler for `SearchClearedMsg`

**Tests:**
- Unit test: SearchOverlay Ctrl+U returns a command that produces SearchClearedMsg
- Unit test: app.Update(SearchClearedMsg) clears store search state

---

## Task 3: Replace hardcoded #000000 with theme.Base()

**Problem:** Two overlay rendering functions in `app.go` (lines 1363 and 1384) hardcode
`lipgloss.Color("#000000")` for whitespace foreground. This bypasses the theme system
and looks wrong on non-black themes (Monokai, Catppuccin, Nord, Light).

**Fix:** Replace both instances of `lipgloss.Color("#000000")` with `a.theme.Base()`.

**Files:** `internal/app/app.go`

**Tests:**
- Unit test: render overlay with non-black theme, verify no `#000000` in style output
  (or verify the style uses the theme's Base() color)

---

## Acceptance Criteria

- [ ] `buildSearchCmd` contains zero store writes before the returned closure
- [ ] Store search state is set in the `Update()` handler before building the command
- [ ] `SearchOverlay` has zero direct `store.Set*` calls
- [ ] `SearchClearedMsg` exists in `messages.go`
- [ ] `app.Update(SearchClearedMsg)` clears search state in store
- [ ] Zero hardcoded hex color values in `app.go`
- [ ] All existing tests pass
- [ ] `make ci` passes
