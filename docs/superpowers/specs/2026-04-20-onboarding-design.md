# Onboarding & Auth UX Redesign

**Date:** 2026-04-20
**Feature:** Feature 09 — Auth, Bootstrap & User Profile
**Status:** Approved — ready for implementation planning

---

## Problem Statement

The current auth flow is CLI-only and fragmented:

- `spotnik auth` triggers a raw OAuth flow in the terminal with no guidance
- `PrintMissingClientIDInstructions` prints to stdout, never appears in the TUI
- The TUI `viewAuth` panel is a single box with a truncated URL and a status string — no steps, no instructions, no retry
- There is no registration flow for first-time users who need to create their own Spotify Developer app
- Spotify does not allow shared OAuth credentials across users, so the embedded ldflags client ID model is fundamentally broken for public distribution

The redesign introduces a guided TUI onboarding flow, a fixed callback port, CLI parity, and profile overlay actions for session management.

---

## Design Decisions

### 1. Client ID Source

**Embedded ldflags client ID (`spotifyClientID`) is removed.**

Client ID is config-first:
- If `~/.config/spotnik/config.toml` contains `[spotify] client_id = "..."` → use it, skip registration
- If absent → show TUI registration flow, user enters their own client ID, it is saved to config

Everyone — including the app developer — goes through the registration flow on first launch. There is no embedded ID, no special developer mode, no pre-populated config. One path for all users.

### 2. Fixed Callback Port

The callback server switches from random port (`net.Listen("tcp", "127.0.0.1:0")`) to a **fixed configurable port**.

Config key: `[spotify] callback_port = 8888` (default: `8888`)

- Registration screen shows `http://127.0.0.1:{callback_port}/callback` — the exact string the user must add to their Spotify Developer app
- This URL never changes between launches, so it only needs to be registered once
- If the port is already in use on launch, show a clear error before the onboarding screen:
  `"Port 8888 is in use — set a different callback_port in ~/.config/spotnik/config.toml"`
- The callback server starts during/after the splash screen so the port is known before registration screen renders

### 3. View Mode Architecture

Three top-level view modes handle all auth states:

```
viewSplash → [no client ID]             → viewOnboarding → viewGrid
viewSplash → [client ID, no tokens]     → viewAuth       → viewGrid
viewSplash → [client ID + valid tokens] →                  viewGrid
```

**`viewOnboarding`** — new view mode. Owns two steps:
- `stepRegister` — client ID input + developer dashboard instructions
- `stepOAuth` — browser wait, full URL shown, retry on error

**`viewAuth`** — existing, shrunk to pure OAuth wait screen. Only shown when client ID already exists but tokens are missing or expired. No registration logic.

New fields on `App`:
```go
onboardingStep  int              // stepRegister | stepOAuth | stepError
onboardingInput textinput.Model  // bubbles/textinput for client ID entry
onboardingError string           // error message shown on stepError
onboardingCodeCh <-chan api.CallbackResult // held open from server start until callback
onboardingServerClose func()     // cleanup for callback server
```

`AppOptions` gains:
```go
NeedsRegister bool   // true when no client_id in config
CallbackPort  int    // resolved port (defaults 8888); 0 means port-busy error already shown
```

New message types:
```go
onboardingServerReadyMsg   // callback server is up, carries port + codeCh
onboardingClientIDSavedMsg // client ID written to config, triggers OAuth
onboardingRetryMsg         // user pressed r — back to stepRegister
```

### 4. Logout / Forget Actions

**Logout** — clears tokens only, quits immediately.
**Forget** — clears tokens + removes `client_id` from `config.toml`, quits immediately.

On next launch after Logout: straight to `viewAuth`.
On next launch after Forget: back to `viewOnboarding` (stepRegister).

Both require a single keypress confirmation before executing (press the key twice).

### 5. Error Recovery

When OAuth fails (invalid client ID, redirect URI mismatch, user denied):
- Show `stepError` screen with error message and common causes
- `r` → back to `stepRegister` (re-enter client ID)
- `l` → retry OAuth with current client ID
- `q` → quit

### 6. Auth URL Display

The full Spotify authorization URL is **never truncated** in any screen. It is shown in a bordered box and wraps across multiple lines. `c` copies the full URL to the clipboard via `pbcopy` (macOS), `xclip` (Linux X11), or `wl-copy` (Wayland). Failure to copy is silent — the URL remains visible for manual selection.

---

## Screen Designs

### Screen 1 — Registration (viewOnboarding, stepRegister)

The callback server has already started. The redirect URI with exact port is shown in step 3.

```
                                 ♪  spotnik
                      A terminal Spotify client for developers


  ╭── Step 1 of 2 — Set up your Spotify Developer App ───────────────────────────────────────╮
  │                                                                                            │
  │  Spotnik requires your own Spotify Developer credentials. Spotify does not allow          │
  │  shared app credentials, so this is a one-time setup. Takes about 2 minutes.             │
  │                                                                                            │
  │  1.  Open  →  https://developer.spotify.com/dashboard                                    │
  │  2.  Click "Create app" — any name and description will do                               │
  │  3.  Under "Redirect URIs" paste this URL exactly:                                       │
  │                                                                                            │
  │      ╭──────────────────────────────────────────────╮                                     │
  │      │  http://127.0.0.1:8888/callback               │  ← copy and paste this             │
  │      ╰──────────────────────────────────────────────╯                                     │
  │                                                                                            │
  │  4.  Tick "Web API" under "Which API/SDKs are you planning to use?"                      │
  │  5.  Click Save → open Settings → copy your Client ID (32-character hex string)         │
  │                                                                                            │
  │  ⚠   Spotify Premium is required to use playback controls                                │
  │  ✓   Your Client ID will be saved to ~/.config/spotnik/config.toml                      │
  │                                                                                            │
  │  ╭─ Paste your Client ID here ─────────────────────────────────────────────────────╮     │
  │  │  > _                                                                            │     │
  │  ╰─────────────────────────────────────────────────────────────────────────────────╯     │
  │                                                                                            │
  ╰────────────────────────────────────────────────────────────────────────────────────────────╯

                          Enter  confirm  ·  q  quit
```

- Input field uses `bubbles/textinput` with placeholder `"your-client-id-here"`
- Redirect URI uses the real runtime port from the callback server
- `⚠` Premium notice always visible — users who skip Premium cannot use playback controls

### Screen 2 — OAuth Wait (viewOnboarding, stepOAuth / viewAuth)

Browser auto-opens. Full untruncated URL shown for headless/manual use. Wraps across lines.

```
                                 ♪  spotnik
                      A terminal Spotify client for developers


  ╭── Step 2 of 2 — Authorize Spotnik with Spotify ───────────────────────────────────────────╮
  │                                                                                             │
  │  A browser window has been opened for you. Log in to Spotify and click Agree.             │
  │                                                                                             │
  │  On a headless server or browser didn't open? Copy and visit this URL:                   │
  │                                                                                             │
  │  ╭──────────────────────────────────────────────────────────────────────────────────────╮  │
  │  │  https://accounts.spotify.com/authorize?client_id=a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4  │  │
  │  │  &response_type=code&redirect_uri=http%3A%2F%2F127.0.0.1%3A8888%2Fcallback          │  │
  │  │  &code_challenge_method=S256&code_challenge=abc123...                                │  │
  │  │  &scope=user-read-playback-state+user-modify-playback-state+...                      │  │
  │  ╰──────────────────────────────────────────────────────────────────────────────────────╯  │
  │                                                                                             │
  │  ⟳  Waiting for authorization...  (times out in 5 minutes)                               │
  │                                                                                             │
  │  Once you approve in the browser, Spotnik continues automatically.                        │
  │                                                                                             │
  ╰─────────────────────────────────────────────────────────────────────────────────────────────╯

                          c  copy URL  ·  q  quit
```

- `viewAuth` (returning user) uses the same layout with title "Re-authenticate with Spotify" and no step indicator
- URL is word-wrapped at box boundary — never truncated with `...`
- `c` copies the full raw URL to clipboard via `pbcopy`/`xclip`/`wl-copy`

### Screen 3 — Error + Retry (viewOnboarding, stepError)

Shown when OAuth returns an error. `r` takes the user back to Step 1.

```
                                 ♪  spotnik
                      A terminal Spotify client for developers


  ╭── Step 2 of 2 — Authorization Failed ─────────────────────────────────────────────────────╮
  │                                                                                             │
  │  ✗  Authorization failed                                                                   │
  │                                                                                             │
  │  Error: invalid_client — The Client ID was not recognised by Spotify.                     │
  │                                                                                             │
  │  Common causes:                                                                             │
  │    •  Client ID was mistyped or truncated                                                  │
  │    •  Redirect URI in your Spotify app does not match:                                     │
  │       http://127.0.0.1:8888/callback                                                      │
  │    •  The Spotify app was deleted or suspended                                             │
  │                                                                                             │
  │  What would you like to do?                                                                │
  │                                                                                             │
  │    r  Re-enter Client ID  (go back to Step 1)                                             │
  │    l  Try again           (keep current Client ID, retry OAuth)                            │
  │    q  Quit                                                                                 │
  │                                                                                             │
  ╰─────────────────────────────────────────────────────────────────────────────────────────────╯
```

### Screen 4 — Profile Overlay (enhanced)

Two new actions below a separator. Confirmation required (press key twice).

```
  ╭── Profile ──────────────────────────────╮
  │  Irshad Sheikh                           │
  │  ────────────────────                    │
  │  ♛  Premium                              │
  │  ◎  IN                                   │
  │                                          │
  │  ────────────────────                    │
  │  l  Logout                               │
  │     ends session · keeps Client ID       │
  │                                          │
  │  f  Forget                               │
  │     removes session + Client ID          │
  │                                          │
  │  Esc  close                              │
  ╰──────────────────────────────────────────╯
```

Confirmation state (after first keypress):
```
  │  !! Press l again to confirm logout      │
  │  !! Press f again to confirm forget      │
```

---

## CLI Commands

All live under `spotnik auth`:

| Command | Behaviour |
|---------|-----------|
| `spotnik auth register` | Show step-by-step instructions, prompt for client ID, save to config, then immediately run OAuth — **register and authenticate in one step**. For first-time setup. |
| `spotnik auth login` | Force re-authentication. Requires client ID already in config. Clears existing tokens and runs OAuth flow. |
| `spotnik auth logout` | Clear session tokens only. Client ID remains in config. Exits with code 0. |
| `spotnik auth forget` | Clear session tokens **and** remove `client_id` from config. Full reset. Exits with code 0. |
| `spotnik auth status` | Print: client ID present (yes/no), tokens present (yes/no), token expiry if available. |

`spotnik auth` with no subcommand prints usage/help.

---

## Config Changes

Add `callback_port` to `[spotify]` section in `config.toml`:

```toml
[spotify]
# client_id = "your-client-id-here"
# callback_port = 8888   # port for the OAuth callback server (default: 8888)
```

The `spotifyConfig` struct in `internal/config/config.go` gains:
```go
type spotifyConfig struct {
    ClientID     string `toml:"client_id"`
    CallbackPort int    `toml:"callback_port"`
}
```

Default: `8888`. If port is busy on startup, print a clear error and exit before any TUI renders.

---

## Data Flow (first-time user)

```
spotnik launched
  └─ config.Bootstrap() — creates config.toml if missing
  └─ config.Load() — reads config
  └─ client_id absent → needsRegister = true
  └─ StartCallbackServer(cfg.CallbackPort) → port known
  └─ App.Init() → splashTimer
       └─ splashDismissMsg
            └─ needsRegister → currentView = viewOnboarding, step = stepRegister
                 └─ user pastes client ID, presses Enter
                      └─ saveClientIDCmd → write client_id to config.toml
                           └─ onboardingClientIDSavedMsg
                                └─ step = stepOAuth
                                └─ prepareAuthCmd (server already running, reuse codeCh)
                                     └─ browser opens, URL shown
                                          └─ waitForCallbackCmd
                                               └─ authSuccessMsg → viewGrid
```

---

## Implementation Constraints

- **Pane / screen structure**: All new screens (`viewOnboarding`, `viewAuth` render helpers) must follow the same patterns as existing panes — see `docs/PANE-TEMPLATE.md` and actual implementations in `internal/ui/panes/`. Pure `View()`, messages for state changes, no API calls inside `Update()`.
- **Theme-aware**: Every colour token must come from the `theme.Theme` interface. No hardcoded hex values anywhere in the onboarding screens or profile overlay changes.
- **`bubbles/textinput`**: Use `github.com/charmbracelet/bubbles/textinput` for the client ID input field on the registration screen. Already a project dependency — no new imports needed.
- **`bubbles/spinner`**: Use `github.com/charmbracelet/bubbles/spinner` for the `⟳` waiting indicator on the OAuth screen.

## Implementation Notes

- **`spotifyClientID` var in `cmd/root.go` is removed** — the ldflags `-X cmd.spotifyClientID` injection is dropped. `loadConfigFromPath` no longer accepts an `embeddedClientID` parameter.
- **Callback server starts in `cmd/root.go`** before `tea.NewProgram` — when `needsRegister` is true, `StartCallbackServer(cfg.CallbackPort)` runs immediately and the port + codeCh are passed into `AppOptions`. The server sits idle until the user completes registration and the browser opens.
- **`spotnik auth` (no subcommand)** currently runs `runApp`. After this change it prints usage/help listing all subcommands. `spotnik auth login` becomes the explicit entry point for forced re-auth.
- **`f` and `l` keys in profile overlay** — neither is currently bound in any overlay or global key map. Safe to use.
- **`forget` in CLI writes config** — uses a new `config.ClearClientID(path)` function that reads the TOML, removes the `client_id` field, and writes it back. Preferences and other settings are preserved.

---

## What Is NOT Changing

- Elm architecture (messages, commands, Store) — unchanged
- `viewAuth` view mode — kept, shrunk to OAuth-only
- Splash screen — unchanged
- All existing Spotify API integration
- Toast notification routing for API errors
- Theme system

---

## Naming: "Forget"

The full-reset action is named **Forget** everywhere:
- Profile overlay key: `f`
- CLI subcommand: `spotnik auth forget`
- Error messages: "Client ID and session forgotten."

Rationale: "logout" is scoped to the session. "forget" implies the app has no memory of you — clear, honest, and not alarming.
