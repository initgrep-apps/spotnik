---
title: "Subscription Gating, Splash Notice, and Keybinding Docs"
feature: 23-user-profile-subscription
status: open
---

## Background

After Stories 114–115 the subscription tier is in the Store and the profile UI is live.
This story adds the behavioural enforcement: free-tier users are blocked from Premium-only
operations at the key-handler level (before any API call), with a clear toast. The 403 fallback
message is also improved. A static notice on the splash screen makes the Premium requirement
visible at first launch. Finally the `u` keybinding is added to all three required locations.

**Depends on:** Story 114 (`store.IsPremium()`), Story 115 (`u` key wiring already present)

## Design

### Premium-only operations to gate

| Key / Message | Handler location |
|---------------|------------------|
| `Space`, `n`, `← →`, `+ -`, `s`, `r`, `a` | `isPlaybackKey` branch in `routing.go` `handleKeyMsg` |
| `Enter` inside device overlay → `TransferPlaybackMsg` | `TransferPlaybackMsg` case in `routing.go` (or `app.go` — wherever the handler lives) |

### Playback key gate (`internal/app/routing.go`)

In the `isPlaybackKey` branch, insert the premium check **before** forwarding to `NowPlayingPane`:

```go
if isPlaybackKey(m) {
    if !a.store.IsPremium() {
        return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required")
    }
    // ... existing routing to NowPlayingPane unchanged
}
```

Free-tier users never reach `NowPlayingPane.Update()` — no playback command is ever dispatched.

### Transfer playback gate (`TransferPlaybackMsg` handler)

Find the `TransferPlaybackMsg` case. Add the premium check immediately after `a.deviceOverlayOpen = false`:

```go
case panes.TransferPlaybackMsg:
    a.deviceOverlayOpen = false
    if !a.store.IsPremium() {
        return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required")
    }
    // existing Batch(buildTransferPlaybackCmd, info toast) unchanged
```

### 403 safety net (`PlaybackCmdSentMsg` handler)

The `ForbiddenError` branch already emits a toast — update the message string only:

```go
// Before
a.alerts.NewAlertCmd("warning", "Playback control not available on this device")

// After
a.alerts.NewAlertCmd("warning", "Spotify Premium required")
```

### Splash notice (`internal/app/splash.go`)

Add a static notice line to `renderSplashView()` — no store read, no timing dependency:

```
╭─────────────────────────────────────╮
│        S P O T N I K                │
│   terminal Spotify client  v1.1.0   │
│                                     │
│  ♛ Playback controls require        │
│    Spotify Premium                  │
╰─────────────────────────────────────╯
```

Implementation: join the existing content with a `notice` line styled in `TextMuted()`.

### Keybinding docs — all three locations (CLAUDE.md rule)

**`docs/keybinding.md`** — add near the `d` (devices) row:

```
| u    | Open user profile overlay | Global (Page A and B) |
```

**`docs/DESIGN.md §17`** — add `u` to the global keybinding table near `d`.

**`internal/ui/panes/help_overlay.go` `helpContent`** — add the `u` line in the same position.

All three must be updated in the same commit.

## Acceptance Criteria

- [ ] Free user pressing Space, n, ←, →, +, -, s, r, or a gets
      "Spotify Premium required" toast; no command is dispatched to `NowPlayingPane`
- [ ] Premium user pressing the same keys dispatches normally
- [ ] Free user receiving `TransferPlaybackMsg` gets "Spotify Premium required" toast; no
      `buildTransferPlaybackCmd` is called
- [ ] `PlaybackCmdSentMsg` with `ForbiddenError` emits "Spotify Premium required"
      (not the previous generic message)
- [ ] Splash screen contains "Playback controls require" and "Spotify Premium"
- [ ] `u` keybinding documented in all three locations in the same commit
- [ ] `make ci` passes

## Tasks

- [ ] Add `TestPremiumGate_FreeUser_PlaybackKeyEmitsToast` and
      `TestPremiumGate_PremiumUser_PlaybackKeyDispatches` to `internal/app/user_profile_test.go`
      - test: `go test ./internal/app/... -run TestPremiumGate -v` → FAIL
- [ ] Add premium gate to `isPlaybackKey` branch in `routing.go`
      - test: both premium gate tests → PASS
- [ ] Add `TestPremiumGate_FreeUser_TransferPlaybackEmitsToast` to `user_profile_test.go`
      - test: `go test ./internal/app/... -run TestPremiumGate_FreeUser_Transfer -v` → FAIL
- [ ] Add premium gate to `TransferPlaybackMsg` handler
      - test: transfer gate test → PASS; all app tests → PASS
- [ ] Update `ForbiddenError` toast message in `PlaybackCmdSentMsg` handler to
      `"Spotify Premium required"`; update any existing test that asserts the old string
      - test: `go test ./internal/app/... -v` → PASS
- [ ] Add `TestRenderSplash_ContainsPremiumNotice` to `internal/app/splash_test.go`
      - test: `go test ./internal/app/... -run TestRenderSplash_ContainsPremiumNotice -v` → FAIL
- [ ] Add static Premium notice to `renderSplashView()` in `splash.go`
      - test: splash test → PASS
- [ ] Update `docs/keybinding.md`, `docs/DESIGN.md §17`, and
      `internal/ui/panes/help_overlay.go` `helpContent` — all in a single commit
      - test: `go build ./...` clean; grep confirms `u` appears in all three files
- [ ] `make ci` passes
