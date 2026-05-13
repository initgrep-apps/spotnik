---
title: "Error Resilience & Universal Polling"
status: open
---

## Description

Comprehensive error handling overhaul across the app. Replaces the one-time startup
`initialFetchCmds()` with a universal tick-driven polling model for all data panes, adding
per-pane exponential backoff and first-load retry. Fixes every silent failure and raw error
string identified in the codebase: prefs flush silence, search offset silent drop, missing HTTP
timeout, raw Go errors in auth onboarding, and overlay states that either hang forever or
conflate error with empty. All API error toasts gain specific, actionable recovery hints.

Four problem classes addressed:
1. **Silent failures** — polling threshold too high, prefs flush silent, search offset drops request, no HTTP timeout
2. **One-time loads with no retry** — library panes fetch once at startup; failure leaves users stranded
3. **Overlay gaps** — profile overlay hangs on "Loading..." forever; devices overlay conflates error and empty
4. **Raw error strings** — auth onboarding shows Go error strings; ErrorMapper 403 bodies are generic

## Acceptance Criteria

- [ ] All library panes (playlists, albums, liked songs, recently played, stats) load via polling — no `initialFetchCmds`
- [ ] No network at startup: panes poll every 5s, first failure emits toast, data loads automatically on recovery, recovery `ToastInfo` confirms it
- [ ] Playback polling error toast fires on 3rd consecutive error (not 5th)
- [ ] Preference flush failure emits `ToastWarning` — no silent stderr
- [ ] Search offset ≥ 1000 returns `SearchPageLoadedMsg{Err}` — no silent nil drop
- [ ] HTTP calls time out after 30s
- [ ] ErrorMapper 403 bodies are operation-specific and actionable
- [ ] Profile overlay emits a self-fetch when store is empty on open; shows error state if fetch fails; "Loading..." never persists indefinitely
- [ ] Devices overlay: error state and empty-devices state are visually distinct
- [ ] Auth onboarding `stepError` shows user-friendly mapped messages — no raw Go error strings
- [ ] `make ci` passes (lint + tests + ≥ 80% coverage)
