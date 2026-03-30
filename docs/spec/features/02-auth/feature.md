---
title: "Authentication"
status: done
---

## Description
Enables secure Spotify OAuth (PKCE) login, automatic token refresh, and keychain-based credential storage so users authenticate once and never need to log in again. The auth system runs entirely before the Bubble Tea TUI starts, in `cmd/root.go`. Tokens are stored in the OS keychain via `go-keyring`, never in plaintext files or environment variables. This feature also establishes the config loading system (`internal/config/`) which reads `~/.config/spotnik/config.toml` for the user's Spotify `client_id` and UI preferences like theme selection. Together, config + auth form the foundation that every subsequent feature depends on.

## Acceptance Criteria
- [ ] First-time user runs `spotnik` and completes auth via browser within 60 seconds
- [ ] Returning user runs `spotnik` and app starts in under 500ms (no browser, no prompts)
- [ ] Expired token refreshes silently without user intervention
- [ ] Failed refresh triggers re-auth flow, never crashes
- [ ] `spotnik auth logout` clears all tokens; next run requires fresh auth
- [ ] Missing `client_id` in config shows clear, actionable error and exits
- [ ] No credentials appear in logs, error output, or tracked files
- [ ] All auth code has >= 80% test coverage
