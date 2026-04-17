---
title: "Auth, Bootstrap & User Profile"
status: in-progress
---

## Description

Authentication via PKCE OAuth flow, token refresh on 401, and keychain-backed token storage. First-launch bootstrap generates a config file with embedded default client ID and prompts the user through setup. The preference store persists user choices (theme, layout preset, visualizer type) with debounced flush on change. The user profile overlay (`u` key) displays name, subscription tier, and country. Premium gating blocks playback controls and device transfer for Free users with a splash toast notice.

## Acceptance Criteria

- [ ] PKCE OAuth flow completes and stores tokens in system keychain
- [ ] Token refresh fires automatically on 401; original request retried once
- [ ] First launch creates config file with embedded client ID; no manual setup required
- [ ] Preference store persists theme/preset/visualizer selection across restarts
- [ ] Profile overlay (`u`) displays name, subscription tier, and country
- [ ] Playback keys and device transfer blocked for Free tier with Premium-required toast
- [ ] Open: story 17 (auth UX improvements)
