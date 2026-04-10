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
- All three keybinding doc locations already had 'u' from story 115 — no changes needed

**Key files:**
- `internal/app/routing.go` — gate at isPlaybackKey branch, new isPremiumOnlyPlaybackKey helper
- `internal/app/handlers.go` — gates at TransferPlaybackMsg and AddToQueueMsg cases
- `internal/app/splash.go` — notice line using TextMuted() theme token
- `internal/app/user_profile_test.go` — premium gate tests (white-box, package app)
- `internal/app/app_test.go`, `command_safety_test.go`, `error_resilience_test.go` — updated to set premium profile

**Patterns established:**
- Gate pattern: `if !a.store.IsPremium() { return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required") }`
- Place gate AFTER state mutations (e.g., deviceOverlayOpen = false) but BEFORE API call dispatch
- 'v' visualizer key is in isPlaybackKey but NOT Premium-gated — it's local UI, not API

**Gotchas:**
- `v` (visualizer cycle) is in `isPlaybackKey` function but does NOT call a Spotify API — it cycles a local animation pattern. Blocking it for free users would be wrong. Used separate `isPremiumOnlyPlaybackKey()` that excludes 'v'
- Any existing test that calls `AddToQueueMsg`, `TransferPlaybackMsg` or uses playback keys on a default (no-profile) App will fail after gating — must add `SetUserProfile({Product: "premium"})` to those tests
- The gate test for BatchMsg vs single cmd works because: without gate, TransferPlaybackMsg returns `tea.Batch(...)` which when executed gives `tea.BatchMsg`; with gate, returns single toast cmd which gives internal alertMsg (not BatchMsg)
- `AddToQueueMsg` is NOT routed through `isPlaybackKey` — it comes from pane messages (pressing 'A' in list panes). Must gate it in the handler, not the key routing branch

**Testing notes:**
- White-box gate tests live in `user_profile_test.go` (package app) to access a.store directly
- isPlaybackRequestMsg helper: executes cmd and checks for panes.PlaybackRequestMsg in result (including batch)
- For TransferPlaybackMsg gate test: check cmd() returns non-BatchMsg (gate returns single cmd, ungated returns BatchMsg)
- Visualizer gate exemption test: verify free user and premium user get same message type from 'v' key using fmt.Sprintf("%T", msg) comparison
