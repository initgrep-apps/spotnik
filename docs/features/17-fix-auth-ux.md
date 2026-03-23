# Feature 17 — Fix Auth UX

> **Bug fix:** Auth screen is not TUI-styled. URL overflows terminal width.
> No splash/branding screen on app startup.

## Bugs Addressed

| # | Issue | Root Cause |
|---|---|---|
| B17 | Auth URL overflows, not centered, no TUI design | No styled auth flow screen |
| B18 | No splash screen on app load | Missing branding/welcome screen |

---

## Root Cause

The auth flow (Feature 02) focused on PKCE functionality, not UX. The auth prompt uses
plain `fmt.Println` for the URL and instructions. No splash screen exists.

---

## Fix

### 1. Splash Screen

On app startup, show a centered ASCII art "SPOTNIK" banner with version and a brief tagline.
Use hand-crafted ASCII art or a simple Go ASCII art approach.

- Show for 1-2 seconds or until first data arrives
- Use theme tokens for styling (ActiveBorder for the banner, TextMuted for tagline)
- Dismiss automatically when playback state loads

### 2. Auth Screen

When auth is needed, render a TUI-styled centered box:

```
╭──────────────────────────────────────╮
│                                      │
│       Authentication Required        │
│                                      │
│  Visit this URL to authorize:        │
│  https://accounts.spotify.com/...    │
│                                      │
│  Press Enter to open in browser      │
│                                      │
╰──────────────────────────────────────╯
```

- URL wrapped/truncated to fit terminal width
- Styled with theme tokens (`SurfaceAlt`, `ActiveBorder`)
- "Press Enter to open in browser" instruction

---

## Files

- `internal/app/app.go` (or new `internal/app/splash.go`) — Splash screen model/view
- `cmd/root.go` — Auth flow TUI rendering
- Tests for splash and auth rendering

---

## Acceptance Criteria

- [ ] App shows ASCII art splash screen with "SPOTNIK", version, tagline on startup
- [ ] Auth screen is a centered, bordered TUI box
- [ ] Auth URL doesn't overflow terminal width (wrapped or truncated)
- [ ] Splash and auth screens use theme tokens, not hardcoded styles
- [ ] Tests verify splash and auth rendering
