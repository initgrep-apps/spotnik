---
title: "Elm Architecture Purity"
feature: 11-api-gateway
status: done
---

## Background
Three violations of the Elm Architecture's unidirectional data flow were found: store mutations in command builders, direct store writes from a pane, and hardcoded theme values. The Elm Architecture requires side effects only via Commands, store writes only from Update() or inside command closures, and all styling via theme tokens.

## Design

### Task 1: Move store mutations out of buildSearchCmd
`buildSearchCmd()` calls `store.SetSearchQuery(query)` and `store.SetSearchLoading(true)` synchronously before returning the command closure. Fix: move both writes into the Update() handler for SearchRequestMsg, before calling buildSearchCmd().

### Task 2: Emit SearchClearedMsg instead of direct store write
`SearchOverlay.Update()` directly writes to the store on Ctrl+U. Fix: define SearchClearedMsg in messages.go, emit it as a command, handle in app.go Update().

### Task 3: Replace hardcoded #000000 with theme.Base()
Two overlay rendering functions hardcode `lipgloss.Color("#000000")` for whitespace foreground. Fix: replace with `a.theme.Base()`.

## Acceptance Criteria
- [ ] `buildSearchCmd` contains zero store writes before the returned closure
- [ ] Store search state is set in the `Update()` handler before building the command
- [ ] `SearchOverlay` has zero direct `store.Set*` calls
- [ ] `SearchClearedMsg` exists in `messages.go`
- [ ] `app.Update(SearchClearedMsg)` clears search state in store
- [ ] Zero hardcoded hex color values in `app.go`
- [ ] All existing tests pass
- [ ] `make ci` passes

## Tasks
- [ ] Move store mutations out of buildSearchCmd into Update() handler
      - test: buildSearchCmd does not call SetSearchQuery or SetSearchLoading
- [ ] Emit SearchClearedMsg instead of direct store write on Ctrl+U
      - test: SearchOverlay Ctrl+U returns command producing SearchClearedMsg
      - test: app.Update(SearchClearedMsg) clears store search state
- [ ] Replace hardcoded #000000 with theme.Base() in overlay rendering
      - test: render overlay with non-black theme, verify no #000000 in style output
