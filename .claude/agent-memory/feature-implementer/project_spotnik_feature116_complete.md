---
name: project_spotnik_feature116_complete
description: Story 116 (Subscription Gating): premium gate patterns, visualizer exemption, test update pattern for gated handlers
type: project
---

## Story 116 — Subscription Gating, Splash Notice, Keybinding Docs

**What was built:**
- Premium gate in `isPlaybackKey` branch (routing.go) — free users get toast, never reach NowPlayingPane
- `isPremiumOnlyPlaybackKey()` helper — same as isPlaybackKey but excludes 'v' (visualizer)
- Premium gate in `TransferPlaybackMsg` handler (handlers.go) — after overlay closes, before API call
- Premium gate in `AddToQueueMsg` handler (handlers.go) — before buildAddToQueueCmd
- `PlaybackCmdSentMsg` 403 ForbiddenError message updated to "Spotify Premium required"
- Static "Playback controls require Spotify Premium" notice in renderSplashView() (splash.go)
- Three keybinding doc locations already had 'u' from story 115 — no changes

**Key files:**
- `internal/app/routing.go` — gate at isPlaybackKey branch, new isPremiumOnlyPlaybackKey helper
- `internal/app/handlers.go` — gates at TransferPlaybackMsg, AddToQueueMsg cases
- `internal/app/splash.go` — notice line uses TextMuted() theme token
- `internal/app/user_profile_test.go` — premium gate tests (white-box, package app)
- `internal/app/app_test.go`, `command_safety_test.go`, `error_resilience_test.go` — updated to set premium profile

**Patterns established:**
- Gate pattern: `if !a.store.IsPremium() { return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required") }`
- Place gate AFTER state mutations (e.g., deviceOverlayOpen = false), BEFORE API call dispatch
- 'v' visualizer key in isPlaybackKey but NOT Premium-gated — local UI, not API

**Gotchas:**
- `v` (visualizer cycle) in `isPlaybackKey` but no Spotify API call — cycles local animation. Blocking free users wrong. Use separate `isPremiumOnlyPlaybackKey()` excluding 'v'
- Existing tests calling `AddToQueueMsg`, `TransferPlaybackMsg` or playback keys on default (no-profile) App fail after gating — add `SetUserProfile({Product: "premium"})`
- Gate test BatchMsg vs single cmd works because: ungated TransferPlaybackMsg returns `tea.Batch(...)` → `tea.BatchMsg`; gated returns single toast cmd → internal alertMsg (not BatchMsg)
- `AddToQueueMsg` NOT routed via `isPlaybackKey` — comes from pane messages ('A' in list panes). Gate in handler, not key routing branch

**Testing notes:**
- White-box gate tests in `user_profile_test.go` (package app) for direct a.store access
- isPlaybackRequestMsg helper: executes cmd, checks panes.PlaybackRequestMsg in result (including batch)
- TransferPlaybackMsg gate test: check cmd() returns non-BatchMsg (gate=single cmd, ungated=BatchMsg)
- Visualizer exemption test: verify free + premium user get same message type from 'v' via fmt.Sprintf("%T", msg) compare