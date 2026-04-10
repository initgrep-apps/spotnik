---
title: "User Profile & Subscription Awareness"
status: done
---

## Description

Fetch the authenticated user's full Spotify profile at startup and use it throughout the app.
Display name and subscription tier appear in the header and in a `u`-triggered profile overlay.
Premium-only playback operations are blocked early (key handler) for free-tier users, with a
clear "Spotify Premium required" toast replacing the existing generic 403 message.

Three parts that build on each other:

1. **Data layer** — expand `UserProfile` with `DisplayName`, `Product`, `Country`; replace bare
   `userID string` in the Store with the full profile; wire the existing `GET /me` command to
   return the complete profile.
2. **Profile UI** — `ProfileOverlay` pane (name, tier badge, country) triggered by `u`; profile
   chip (name + badge) on the header right side.
3. **Subscription gating** — block Premium-only key actions before any API call; improve the 403
   fallback message; add a static Premium notice to the splash screen.

## Goals

- No new API calls — the existing `GET /me` fetch at startup already returns all needed fields.
- `UserID()` call-site compatibility is fully preserved — no existing callers change.
- Free-tier behaviour is the safe default: `IsPremium()` returns `false` when profile is not yet
  loaded or when `Product` is empty/unexpected.

## Acceptance Criteria

- [ ] `domain.UserProfile` has `DisplayName string`, `Product string`, `Country string`
- [ ] `store.UserProfile()`, `store.SetUserProfile()`, `store.IsPremium()` exist and are tested
- [ ] `store.UserID()` still returns `userProfile.ID` — no call-site changes elsewhere
- [ ] Header right side shows `◉ DeviceName   DisplayName ♛` (Premium) or `○` (Free)
- [ ] `u` key opens the profile overlay; `Esc` closes it
- [ ] Profile overlay renders name, tier badge, country; falls back to "Loading profile..." when
      profile not yet loaded
- [ ] Free user pressing any playback key (Space, n, ←, →, +, -, s, r, a) gets a
      "Spotify Premium required" toast; no API call is made
- [ ] Free user selecting a device for transfer gets a "Spotify Premium required" toast
- [ ] 403 response from any playback API shows "Spotify Premium required" (was generic)
- [ ] Splash screen contains the static Premium notice line
- [ ] `u` keybinding added to all three locations: `docs/keybinding.md`, `docs/DESIGN.md §17`,
      `internal/ui/panes/help_overlay.go` `helpContent`
- [ ] `make ci` passes (lint + tests + ≥ 80% coverage)

## Stories

| # | Title | Status |
|---|-------|--------|
| 114 | Data layer — UserProfile expansion and profile pipeline | done |
| 115 | Profile UI — overlay pane, header chip, App wiring | done |
| 116 | Subscription gating, splash notice, keybinding docs | done |
| 117 | Profile overlay UX fixes — remove duplicate hint, add status bar binding | open |

## Premium-Only Operations (from `docs/API-CAPABILITY.md §21`)

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
