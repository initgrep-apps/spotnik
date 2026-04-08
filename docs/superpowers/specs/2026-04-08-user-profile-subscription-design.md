# User Profile & Subscription Awareness — Design Spec

**Date:** 2026-04-08
**Status:** Approved

---

## Overview

Add user profile display and subscription-aware behaviour to Spotnik. The feature has three
parts that build on each other:

1. **Expand the profile pipeline** — parse all useful fields from the existing `GET /me` call
2. **Profile overlay** — show name, tier, and country in a `u`-triggered overlay; display name + tier badge in the header
3. **Subscription gating** — block Premium-only key actions early (key handler) for Free users; improve 403 error messages as a safety net

---

## Premium vs Free Classification

Based on `docs/API-CAPABILITY.md §21`.

### Premium-only operations (require `product == "premium"`)

| Key | Action |
|-----|--------|
| `Space` | Play / Pause |
| `n` | Next track |
| `← →` | Seek |
| `+ -` | Volume |
| `s` | Shuffle |
| `r` | Repeat |
| `a` | Add to queue |
| `Enter` (device overlay) | Transfer playback |

### Free-tier operations (no gate needed)

- Get playback state, queue, devices
- All Library operations (like/unlike, save albums)
- All Playlist operations (create, modify, reorder)
- All Search operations
- All User profile operations (`GET /me`)

---

## Data Layer

### `internal/domain/types.go` — expand `UserProfile`

```go
type UserProfile struct {
    ID          string // Spotify user ID (existing)
    DisplayName string `json:"display_name"`
    Product     string `json:"product"` // "premium" or "free"
    Country     string `json:"country"` // ISO 3166-1 alpha-2
}
```

### `internal/state/store.go` — replace bare `userID` with full profile

Replace `userID string` field with `userProfile domain.UserProfile`.

New accessors:
```go
func (s *Store) UserID() string                    // returns s.userProfile.ID (no call-site changes)
func (s *Store) UserProfile() domain.UserProfile
func (s *Store) SetUserProfile(p domain.UserProfile)
func (s *Store) IsPremium() bool                   // returns p.Product == "premium"
```

`UserID()` continues to return `s.userProfile.ID` so all existing call sites
(playlist ownership check) require no changes.

### `internal/app/app.go` — expand `userProfileLoadedMsg`

```go
type userProfileLoadedMsg struct {
    profile domain.UserProfile
    err     error
}
```

### `internal/app/commands.go` — `buildFetchCurrentUserCmd`

Already calls `GET /me` and maps to `domain.UserProfile`. Stop discarding the new
fields — return the full profile in the message. No new API call, no new interface method.

### `internal/app/routing.go` — `userProfileLoadedMsg` handler

Replace `a.store.SetUserID(m.userID)` with `a.store.SetUserProfile(m.profile)`.
Forward `panes.UserProfileReadyMsg{}` to `PanePlaylists` as today.

---

## UI: Header

**File:** `internal/app/render.go` — `renderHeader()`

Right side order: device info → profile chip (rightmost).

```
◉ MacBook Pro   Irshad Sheikh ♛
◉ MacBook Pro   John Doe ○
```

- `♛` in `theme.Info()` for Premium
- `○` in `theme.TextMuted()` for Free
- Display name truncated to ~20 chars on narrow terminals
- Profile chip omitted entirely if profile not yet loaded (graceful — loads within ~500ms)

---

## UI: Profile Overlay

**File:** `internal/ui/panes/profile.go` — new `ProfileOverlay` struct

```
╭─ Profile ──────────────────────╮
│                                │
│   Irshad Sheikh                │
│   ────────────────────────     │
│   ♛  Premium                   │
│   ◎  Germany (DE)              │
│                                │
│   esc  close                   │
╰────────────────────────────────╯
```

Styling:
- `Irshad Sheikh` — `TextPrimary()` + Bold
- `────────` separator — `TextMuted()`
- `♛  Premium` — `Info()` color
- `○  Free` — `TextMuted()` color
- `◎  Germany (DE)` — icon in `TextMuted()`, value in `TextFg()`
- `esc  close` — `TextMuted()`
- Border — `BorderFocused()` (same as device overlay)

**Behaviour:**
- Reads from `store.UserProfile()` — pure `View()`, no local state beyond `width/height`
- No `Init()` needed — data already in store before user can press `u`
- If profile not yet loaded: render `"Loading profile..."` placeholder
- Triggered by `u` key → `profileOverlayOpen bool` on `App`
- `Esc` closes via `ProfileOverlayClosedMsg{}`
- Composited via `bubbletea-overlay`, positioned top-right (same as device overlay)
- All keys intercepted while open (same routing guard pattern as device overlay)

**Key binding addition in `docs/DESIGN.md`:**
- `u` — open profile overlay (global, Page A and B)

---

## Subscription Gating

### Key handler gate (`internal/app/routing.go`)

Insert one check in the `isPlaybackKey` branch before dispatching to `NowPlayingPane`:

```
isPlaybackKey?
  ├─ store.IsPremium() == false
  │     → return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required")
  └─ true
        → route to NowPlayingPane as today
```

`Enter` inside the device overlay emits `TransferPlaybackMsg` from within
`DeviceOverlay.Update()` — the premium check lives in the `TransferPlaybackMsg`
handler in `routing.go` (not the key handler), before `buildTransferPlaybackCmd` is dispatched.

### 403 safety net (`internal/app/routing.go` — `PlaybackCmdSentMsg` handler)

```
// Before
"Playback control not available on this device"

// After
"Spotify Premium required"
```

### `store.IsPremium()` safe defaults

- Returns `false` when `Product` is empty or any value other than `"premium"`
- Free-tier behaviour is the safe default — no Premium-only calls go out if profile fetch fails

---

## Splash Screen

**File:** `internal/app/splash.go` — `renderSplashView()`

Add a static notice line — always shown, no dependency on profile load timing:

```
╭─────────────────────────────────────╮
│        S P O T N I K                │
│   terminal Spotify client  v1.1.0   │
│                                     │
│  ♛ Playback controls require        │
│    Spotify Premium                  │
╰─────────────────────────────────────╯
```

Static text, no store read, no race condition.

---

## Error Handling

| Scenario | Behaviour |
|----------|-----------|
| Profile fetch fails at startup | Existing warning toast "Could not load your Spotify profile"; `IsPremium()` returns `false` (safe default) |
| Profile overlay opened before load | Renders `"Loading profile..."` placeholder; resolves when `userProfileLoadedMsg` arrives |
| Free user presses Premium key | Toast "Spotify Premium required"; no API call made |
| Premium user hits 403 (safety net) | Toast "Spotify Premium required" (improved from generic message) |
| `product` field empty or unexpected | `IsPremium()` returns `false` — treats unknown as Free |

---

## Testing

### `internal/domain/types_test.go`
- `UserProfile` new fields parse correctly from JSON

### `internal/state/store_test.go`
- `SetUserProfile` / `UserProfile()` round-trip
- `IsPremium()`: true for `"premium"`, false for `"free"`, false for empty string

### `internal/api/user_test.go`
- `Profile()` parses `display_name`, `product`, `country` from fixture JSON

### `internal/app/user_profile_test.go` (extend existing)
- Full profile stored on `userProfileLoadedMsg`
- Premium user pressing playback key → command dispatched
- Free user pressing playback key → warning toast emitted, no command

### `internal/ui/panes/profile_test.go` (new)
- `View()` renders display name, `♛ Premium` / `○ Free`, country
- Empty profile renders `"Loading profile..."` placeholder

### `internal/app/splash_test.go` (extend existing)
- Splash view contains the Premium notice line

---

## Files Changed

| File | Change |
|------|--------|
| `internal/domain/types.go` | Add `DisplayName`, `Product`, `Country` to `UserProfile` |
| `internal/state/store.go` | Replace `userID string` with `userProfile domain.UserProfile`; add `IsPremium()`, `UserProfile()`, `SetUserProfile()` |
| `internal/app/app.go` | Expand `userProfileLoadedMsg`; add `profileOverlayOpen bool`, `profilePane *panes.ProfileOverlay` |
| `internal/app/commands.go` | Parse full profile in `buildFetchCurrentUserCmd` |
| `internal/app/routing.go` | Premium gate in `isPlaybackKey` branch; `u` key opens profile overlay; improve 403 message; update `userProfileLoadedMsg` handler |
| `internal/app/render.go` | Header right side: device then profile chip; `renderWithProfileOverlay()`; `buildView()` compositing |
| `internal/app/splash.go` | Add static Premium notice line to `renderSplashView()` |
| `internal/ui/panes/profile.go` | New `ProfileOverlay` struct and `View()` |
| `internal/ui/panes/messages.go` | Add `ProfileOverlayClosedMsg{}` |
| `docs/DESIGN.md` | Add `u` keybinding to the keybinding table |

---

## What Does NOT Change

- `UserAPI` interface — `Profile()` method signature unchanged
- All existing call sites of `store.UserID()` — method preserved, same return value
- Gateway — no premium gating at gateway level
- Library, playlist, search operations — no gating (free tier)
- `PlaylistsPane` — `UserProfileReadyMsg` routing unchanged
