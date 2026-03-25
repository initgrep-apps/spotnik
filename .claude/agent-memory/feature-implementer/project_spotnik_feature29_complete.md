---
name: project_spotnik_feature29_complete
description: Feature 29 (Elm Purity: Data-Carrying Messages) — domain package, data-carrying msgs, store write placement, parallel stats, test update patterns
type: project
---

## Feature 29 — Elm Purity: Data-Carrying Messages

**What was built:**
- New `internal/domain/types.go` package with all shared domain types extracted from `api/models.go`
- `api/models.go` replaced with type aliases re-exporting from domain (backward compat)
- All 9 `build*Cmd`/`fetch*Cmd` in `commands.go` now return data in Msg payloads (zero store writes)
- `app.go` Update() handlers now write to Store from Msg payload fields
- `routing.go` `PlaylistTracksLoadedMsg` handler writes to store
- `DeviceOverlay.Update()` handles `devicesLoadedMsg` store writes (devicesLoadedMsg is unexported)
- `buildFetchStatsCmd` parallelized with `sync.WaitGroup` (no errgroup — CLAUDE.md restricts deps)
- New test file: `internal/app/elm_purity_test.go` with 20+ tests

**Key files:**
- `internal/domain/types.go` — all shared types: Track, PlaybackState, Device, etc.
- `internal/api/models.go` — now only type aliases (type X = domain.X)
- `internal/app/commands.go` — zero store writes, verified by grep
- `internal/app/app.go` — new handlers for LibraryLoadedMsg, AlbumsLoadedMsg, etc.
- `internal/app/elm_purity_test.go` — TDD tests for all new message patterns

**Patterns established:**
- Data-carrying Msg: `type QueueLoadedMsg struct { Tracks []domain.Track; Err error }`
- Update() handler pattern: check Err → write store → forward to pane
- `fetchPlaybackStateCmd(player api.PlayerAPI)` — no store param
- `fetchQueueCmd(player api.PlayerAPI)` — no store param
- `sync.WaitGroup` fan-out for parallel API calls in a single Cmd

**Gotchas:**
- `devicesLoadedMsg` is UNEXPORTED — app.go can't switch on it. Fix: store writes stay in DeviceOverlay.Update() (still Elm-compliant since only Update() writes store)
- `SearchResultsMsg.Results` is `*panes.SearchResultData` (UI type), not `*api.SearchResult` (store type). Cannot write Results to store.SearchResults(). The store.SearchResults() field is dead/unused in production — only tests use it directly.
- Tests that called `cmd()` but didn't feed result back to `Update()` broke silently after refactor — pattern fix: `msg := cmd(); m, _ := a.Update(msg); a = m.(*app.App)`
- `TestPollingLoop_FetchesAndUpdatesStore` needed update: old test manually wrote state to store then sent empty PlaybackStateFetchedMsg{}. New pattern: send `PlaybackStateFetchedMsg{State: newState}`.
- `TestApp_BuildFetchDevicesCmd_*` tests needed to open device overlay first ('d' key) so devicesLoadedMsg gets routed to DeviceOverlay (only routed when deviceOverlayOpen==true)
- Parallel stats fetch: check tracksErr first, then artistsErr separately — can't combine with || since we need different messages per error type

**Testing notes:**
- Coverage: 81.7% total (above 80% threshold)
- TDD approach: wrote elm_purity_test.go before updating implementation
- Test helper pattern: `runFetchPlaybackCmd(a)` + `extractPlaybackMsg(msg)` for extracting msgs from BatchMsg chains
- State comparison: `a.Store().PlaybackState()` after `a = m.(*app.App)` — always re-assign a after Update()
