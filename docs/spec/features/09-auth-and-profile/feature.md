---
title: "Auth, Bootstrap & User Profile"
status: in-progress
---

## Description

Authentication via PKCE OAuth flow, token refresh on 401, and keychain-backed token storage.

**First-launch experience (redesigned):** Spotify does not allow shared OAuth credentials across
users. The old embedded ldflags client ID model has been removed. Every user — including the app
developer — registers their own Spotify Developer app on first launch. The TUI guides them
through a two-step onboarding flow (`viewOnboarding`): Step 1 collects their client ID and shows
the exact redirect URI to register; Step 2 opens the browser for OAuth authorization. The client
ID is saved to `~/.config/spotnik/config.toml`. The OAuth callback server binds to a fixed
configurable port (default 8888) so the redirect URI never changes between launches.

**Auth CLI:** `spotnik auth` exposes five subcommands — `register`, `login`, `logout`, `forget`,
`status`. These provide CLI parity for every TUI auth action.

**Profile overlay:** The `u`-triggered profile overlay displays name, subscription tier, and
country. Session management actions (`l` logout, `f` forget) are accessible from the overlay
with double-key confirmation.

**Premium gating:** Playback controls and device transfer are blocked for Free tier users with a
toast notice.

## Acceptance Criteria

- [ ] PKCE OAuth flow completes and stores tokens in system keychain
- [ ] Token refresh fires automatically on 401; original request retried once
- [ ] No embedded client ID at build time — client ID is config-first only
- [ ] First launch with no client ID in config shows `viewOnboarding` (stepRegister) after splash
- [ ] Registration screen shows exact redirect URI with the configured callback port
- [ ] Client ID saved to `~/.config/spotnik/config.toml` after Step 1
- [ ] Step 2 shows full untruncated OAuth URL; browser opens automatically
- [ ] `c` copies the full auth URL to clipboard on Step 2 and on `viewAuth`
- [ ] OAuth error shows Step 2 error screen with `r`/`l`/`q` retry options
- [ ] Returning user with no tokens → `viewAuth` (OAuth-only, no registration)
- [ ] `spotnik auth register` — guides through setup, prompts for client ID, runs OAuth
- [ ] `spotnik auth login` — clears tokens and re-runs OAuth; errors if no client ID in config
- [ ] `spotnik auth logout` — clears tokens only; exits 0
- [ ] `spotnik auth forget` — clears tokens and removes client ID from config; exits 0
- [ ] `spotnik auth status` — prints client ID presence and token state
- [ ] Preference store persists theme/preset/visualizer selection across restarts
- [ ] Profile overlay (`u`) displays name, subscription tier, and country
- [ ] Profile overlay `l` (logout) clears tokens and quits; requires double-key confirmation
- [ ] Profile overlay `f` (forget) clears tokens + client ID and quits; requires double-key confirmation
- [ ] Playback keys and device transfer blocked for Free tier with Premium-required toast
