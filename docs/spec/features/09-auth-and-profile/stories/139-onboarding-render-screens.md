---
title: "Onboarding — Render Functions (Register, OAuth Wait, Error Screens)"
feature: 09-auth-and-profile
status: open
---

## Background

This story implements the pure `View()` layer for the three onboarding screens. All functions
are side-effect-free: they read app state → return a string. No messages, no commands.

The three screens follow the visual design from `docs/superpowers/specs/2026-04-20-onboarding-design.md`:

- **Step 1 — Register** (`stepRegister`): Spotify Developer instructions, redirect URI in a
  bordered box, `bubbles/textinput` for client ID entry.
- **Step 2 — OAuth Wait** (`stepOAuth`): full untruncated auth URL, spinner, browser-open
  instructions.
- **Step 2 Error** (`stepError`): error message, common causes, `r`/`l`/`q` retry options.

The existing `viewAuth` `renderAuthPanel` is also updated: URL is no longer truncated.

**No new Lip Gloss tokens invented.** Every colour must come from the `theme.Theme` interface.

**Depends on:** Stories 137 and 138 (all onboarding fields and dispatch must exist; `buildView()`
must compile).

## Design

### `internal/app/render.go`

**`wrapURL(rawURL string, width int) string`** — wraps a long URL across multiple lines at the
given character width. Tries to break just before an `&` query-parameter boundary in the second
half of the window; falls back to a hard break at `width` if no `&` is found.

```go
func wrapURL(rawURL string, width int) string {
    if len(rawURL) <= width {
        return rawURL
    }
    var lines []string
    for len(rawURL) > width {
        breakAt := width
        if idx := strings.LastIndex(rawURL[:width], "&"); idx > width/2 {
            breakAt = idx
        }
        lines = append(lines, rawURL[:breakAt])
        rawURL = rawURL[breakAt:]
    }
    if rawURL != "" {
        lines = append(lines, rawURL)
    }
    return strings.Join(lines, "\n")
}
```

**`(a *App) onboardingTitle() string`** — shared header for all onboarding screens:

```
♪  spotnik                    (TextPrimary, Bold)
A terminal Spotify client for developers  (TextMuted)
```

Rendered with `lipgloss.JoinVertical(lipgloss.Center, ...)`.

**`(a *App) renderOnboarding() string`** — dispatch to step renderer; wrap with
`lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, body)` when dimensions
are known.

**`(a *App) renderOnboardingRegister() string`** — Step 1 layout:

```
╭── Step 1 of 2 — Set up your Spotify Developer App ──────────────╮
│                                                                  │
│  [instructions text]                                             │
│                                                                  │
│  3. Under "Redirect URIs" paste this URL exactly:               │
│     ╭──────────────────────────────────────────╮                 │
│     │  http://127.0.0.1:{port}/callback  ← copy │                │
│     ╰──────────────────────────────────────────╯                 │
│                                                                  │
│  ⚠  Spotify Premium is required for playback controls           │
│  ✓  Your Client ID will be saved to ~/.config/spotnik/config.toml│
│                                                                  │
│  ╭─ Paste your Client ID here ─────────────────────────────────╮ │
│  │  > _                                                        │ │
│  ╰─────────────────────────────────────────────────────────────╯ │
╰──────────────────────────────────────────────────────────────────╯

              Enter  confirm  ·  q  quit
```

- Outer panel: `RoundedBorder()`, `ActiveBorder()` colour
- Redirect URI inner box: `RoundedBorder()`, `TextMuted()` colour
- `⚠` in `Warning()`, `✓` in `Success()`
- Input box: `RoundedBorder()`, `ActiveBorder()` colour; `a.onboardingInput.View()` inside
- Port taken from `a.onboardingPort`

**`(a *App) renderOnboardingOAuth() string`** — Step 2 layout:

```
╭── Step 2 of 2 — Authorize Spotnik with Spotify ──────────────────╮
│                                                                   │
│  A browser window has been opened. Log in and click Agree.        │
│                                                                   │
│  On a headless server or browser didn't open? Visit this URL:     │
│  ╭────────────────────────────────────────────────────────────╮   │
│  │  https://accounts.spotify.com/authorize?client_id=...      │   │
│  │  &response_type=code&redirect_uri=...                      │   │
│  ╰────────────────────────────────────────────────────────────╯   │
│                                                                   │
│  ⟳  Waiting for authorization...  (times out in 5 minutes)        │
│                                                                   │
╰───────────────────────────────────────────────────────────────────╯

                    c  copy URL  ·  q  quit
```

- URL rendered via `wrapURL(a.onboardingAuthURL, innerW)` — **never truncated**
- URL box: `RoundedBorder()`, `TextMuted()` colour; URL text in `ActiveBorder()` colour
- Spinner: `a.onboardingSpinner.View()` + status text in `TextMuted()`
- Outer panel: `RoundedBorder()`, `ActiveBorder()` colour

**`(a *App) renderOnboardingError() string`** — Step 2 Error layout:

```
╭── Step 2 of 2 — Authorization Failed ────────────────────────────╮
│                                                                   │
│  ✗  Authorization failed                                          │
│  Error: {a.onboardingError}                                       │
│                                                                   │
│  Common causes:                                                   │
│    •  Client ID mistyped or truncated                             │
│    •  Redirect URI does not match: http://127.0.0.1:{port}/callback│
│    •  Spotify app deleted or suspended                            │
│                                                                   │
│    r  Re-enter Client ID  (go back to Step 1)                     │
│    l  Try again           (keep current Client ID, retry OAuth)   │
│    q  Quit                                                        │
╰───────────────────────────────────────────────────────────────────╯
```

- Outer panel border in `Error()` colour
- `✗` and error text in `Error()` colour
- Port taken from `a.onboardingPort`

**Update `renderAuthPanel`** (for `viewAuth` — returning user):

- Title: "Re-authenticate with Spotify" (no step indicator)
- URL rendered via `wrapURL(authURL, innerW)` — **never truncated**
- URL box same style as OAuth wait screen
- Status text + `c  copy URL  ·  q  quit` hint

**Update `buildView()`** — add `viewOnboarding` dispatch before `viewAuth`:

```go
if a.currentView == viewOnboarding {
    return a.renderOnboarding()
}
if a.currentView == viewAuth {
    return renderAuthPanel(a.theme, a.width, a.height, a.authURL, a.authStatus)
}
```

### Tests — `internal/app/render_test.go`

Write **failing** tests first (TDD):

```go
func TestWrapURL_shortURL_unchanged(t *testing.T) {
    url := "https://example.com/short"
    assert.Equal(t, url, wrapURL(url, 80))
}

func TestWrapURL_longURL_breaksAtAmpersand(t *testing.T) {
    url := "https://accounts.spotify.com/authorize?client_id=abc123&response_type=code" +
        "&redirect_uri=http://127.0.0.1:8888/callback"
    result := wrapURL(url, 60)
    lines := strings.Split(result, "\n")
    assert.Greater(t, len(lines), 1)
    for _, line := range lines {
        assert.LessOrEqual(t, len(line), 60)
    }
}

func TestWrapURL_noAmpersand_breaksAtWidth(t *testing.T) {
    url := strings.Repeat("a", 150)
    result := wrapURL(url, 60)
    lines := strings.Split(result, "\n")
    assert.Equal(t, 3, len(lines)) // 60 + 60 + 30
}
```

## Acceptance Criteria

- [ ] `wrapURL` returns input unchanged when `len(rawURL) <= width`
- [ ] `wrapURL` breaks at `&` boundaries when possible
- [ ] `wrapURL` hard-breaks at `width` when no `&` is present
- [ ] `renderOnboardingRegister`: contains step title, instructions, redirect URI with port,
      `⚠` Premium notice, `✓` config path notice, textinput view
- [ ] `renderOnboardingOAuth`: contains step title, full URL via `wrapURL` (not truncated),
      spinner output, `c  copy URL` hint
- [ ] `renderOnboardingError`: contains error text, common causes, redirect URI with port, `r`/`l`/`q` options
- [ ] `renderAuthPanel`: URL rendered via `wrapURL` (not truncated), title "Re-authenticate with Spotify"
- [ ] `buildView()` dispatches `viewOnboarding` before `viewAuth`
- [ ] No hardcoded hex colour values anywhere in the new render code
- [ ] All colour tokens come from `theme.Theme` interface methods
- [ ] `TestWrapURL_*` tests pass; `make ci` passes

## Tasks

- [ ] Write failing tests in `internal/app/render_test.go` for `TestWrapURL_*`
      - test: `go test ./internal/app/... -run "TestWrapURL" -v` → compile error (`wrapURL` undefined)
- [ ] Implement `wrapURL` in `internal/app/render.go`
      - test: `TestWrapURL_*` → PASS
- [ ] Implement `onboardingTitle`, `renderOnboarding`, `renderOnboardingRegister`,
      `renderOnboardingOAuth`, `renderOnboardingError` in `internal/app/render.go`
      - test: `go build ./...` → clean
- [ ] Update `renderAuthPanel` to use `wrapURL` instead of truncation
      - test: `go build ./...` → clean
- [ ] Update `buildView()` to dispatch `viewOnboarding`
      - test: `go test ./internal/app/... -v` → PASS (no regressions)
- [ ] `make ci` passes
