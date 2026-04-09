---
title: "Developer Foundations"
status: done
---

## Description

A structured codebase health initiative addressing three categories of improvement identified
in the 2026-04-08 code audit: missing onboarding documentation and test infrastructure,
structural file-size and interface issues, and repeating design patterns across panes.

None of these stories change user-visible behaviour. Together they make the codebase more
navigable for new contributors, easier to test, and more consistent in its Go patterns.

## Goals

- New contributors can clone, set up, test, and contribute without having to read source code
  to understand conventions
- Fixture-loading boilerplate is eliminated from 15+ test call sites via a shared helper
- `gateway.go` (747 LOC) and `app.go` (1807 LOC) are each split into focused files ≤ 700 LOC
- All 8 table-based panes accept a read-only `StateReader` interface instead of `*Store`,
  making pane unit tests lighter and import boundaries stricter
- Repeated pane boilerplate (5 fields, 4 trivial methods per pane × 8 panes) is eliminated
  via `BasePane` embedding
- Token exchange in `auth.go` is testable via `httptest.NewServer` (injected client,
  not `http.DefaultClient`)

## Acceptance Criteria

- [ ] `README.md`, `CONTRIBUTING.md`, `docs/DEV-SETUP.md`, `docs/TESTING.md`,
      `docs/PANE-TEMPLATE.md` exist and are accurate
- [ ] `make test-integration` target runs and passes the existing integration tests
- [ ] `testhelpers.LoadFixture` used in all `internal/api/*_test.go` files — no inline
      `os.ReadFile` calls for fixtures
- [ ] `docs/ARCHITECTURE.md` documents PreferenceStore, page/preset/toggle system,
      view lifecycle, and overlay routing precedence
- [ ] `docs/DESIGN.md` §2 includes `SetTheme`, §17 marks `?` as planned, §18 lists all 11 themes
- [ ] `internal/state.StateReader` read-only interface exists with compile-time assertion
- [ ] All 8 pane constructors accept `state.StateReader` instead of `*state.Store`
- [ ] `internal/api/gateway.go` split into `gateway.go`, `gateway_bucket.go`, `gateway_dedup.go`
- [ ] `internal/app/app.go` split into `app.go`, `handlers.go`, `prefs.go`
- [ ] `player_test.go`, `library_test.go`, `playlists_test.go` use table-driven style throughout
- [ ] `BasePane` embedded in all 8 Page A panes; no pane struct re-declares `store`, `theme`,
      `focused`, `width`, or `height`
- [ ] `components.RebuildTableTheme` helper used in all 8 pane `SetTheme()` methods
- [ ] `postTokenRequest` in `auth.go` uses the injected `*http.Client`; a test verifies it
- [ ] `make ci` passes after every story merge

## Stories

| # | Title | Status |
|---|-------|--------|
| 109 | Onboarding docs, test infrastructure & doc corrections | open |
| 110 | StateReader interface, file splits & table-driven tests | open |
| 111 | BasePane embedding, RebuildTableTheme & auth client fix | open |
