---
name: project_spotnik_feature29_complete
description: Feature 29 (Elm Purity: Data-Carrying Messages) — domain package, data-carrying msgs, store write placement, parallel stats, test update patterns
type: project
---

## Feature 29 — Elm Purity: Data-Carrying Messages

**What was built:**
- New `internal/domain/types.go` package. All shared domain types extracted from `api/models.go`
- `api/models.go` replaced with type aliases re-exporting from domain (backward compat)
- All 9 `build*Cmd`/`fetch*Cmd` in `commands.go` return data in Msg payloads (zero store writes)
- `app.go` Update() handlers write to Store from Msg payload fields
- `routing.go` `PlaylistTracksLoadedMsg` handler writes to store
- `DeviceOverlay.Update()` handles `devicesLoadedMsg` store writes (devicesLoadedMsg unexported)
- `buildFetchStatsCmd` parallelized with `sync.WaitGroup` (no errgroup — CLAUDE.md restricts deps)
- New test file: `internal/app/elm_purity_test.go`. 20+ tests

**Key files:**
- `internal/domain/types.go` — all shared types: Track, PlaybackState, Device, etc.
- `internal/api/models.go` — only type aliases (type X = domain.X)
- `internal/app/commands.go` — zero store writes, verified by grep
- `internal/app/app.go` — new handlers for LibraryLoadedMsg, AlbumsLoadedMsg, etc.
- `internal/app/elm_purity_test.go` — TDD tests for new message patterns

**Patterns established:**
- Data-carrying Msg: `type QueueLoadedMsg struct { Tracks []domain.Track; Err error }`
- Update() handler pattern: check Err → write store → forward to pane
- `fetchPlaybackStateCmd(player api.PlayerAPI)` — no store param
- `fetchQueueCmd(player api.PlayerAPI)` — no store param
- `sync.WaitGroup` fan-out for parallel API calls in single Cmd

**Gotchas:**
- `devicesLoadedMsg` UNEXPORTED — app.go can't switch on it. Fix: store writes stay in DeviceOverlay.Update() (still Elm-compliant since only Update() writes store)
- `SearchResultsMsg.Results` is `*panes.SearchResultData` (UI type), not `*api.SearchResult` (store type). Can't write Results to store.SearchResults(). store.SearchResults() field dead/unused in production — only tests use directly.
- Tests calling `cmd()` without feeding result back to `Update()` broke silently after refactor. Fix pattern: `msg := cmd(); m, _ := a.Update(msg); a = m.(*app.App)`
- `TestPollingLoop_FetchesAndUpdatesStore` needed update: old test manually wrote state to store then sent empty PlaybackStateFetchedMsg{}. New pattern: send `PlaybackStateFetchedMsg{State: newState}`.
- `TestApp_BuildFetchDevicesCmd_*` tests needed device overlay open first ('d' key) so devicesLoadedMsg routes to DeviceOverlay (only routed when deviceOverlayOpen==true)
- Parallel stats fetch: check tracksErr first, then artistsErr separately — can't combine with || since need different messages per error type

**Testing notes:**
- Coverage: 81.7% total (above 80% threshold)
- TDD: wrote elm_purity_test.go before updating implementation
- Test helper pattern: `runFetchPlaybackCmd(a)` + `extractPlaybackMsg(msg)` to extract msgs from BatchMsg chains
- State comparison: `a.Store().PlaybackState()` after `a = m.(*app.App)` — always re-assign a after Update()