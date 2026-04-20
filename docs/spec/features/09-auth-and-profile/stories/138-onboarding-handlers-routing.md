---
title: "Onboarding — Message Handlers and Key Routing"
feature: 09-auth-and-profile
status: open
---

## Background

Story 137 introduced the onboarding state machine, message types, and commands. This story
wires them into the Bubble Tea Update loop: `handlers.go` receives the new message types and
transitions state; `routing.go` intercepts key events during `viewOnboarding` and dispatches
them to the correct step handler.

All state transitions must happen via messages — no API calls or file I/O inside `Update()`.

**Depends on:** Story 137 (all types, constants, and commands must exist).

## Design

### `internal/app/handlers.go`

**Update `splashDismissMsg` case** — route to `viewOnboarding` when `needsRegister` is true:

```go
case splashDismissMsg:
    if a.currentView == viewSplash {
        switch {
        case a.needsRegister:
            a.currentView = viewOnboarding
            a.onboardingStep = stepRegister
            a.onboardingInput.Focus()
        case a.needsAuth:
            a.currentView = viewAuth
            a.authStatus = "Opening browser for authorization..."
            return a, prepareOAuthCmd(a.clientID, a.onboardingPort, a.onboardingCodeCh, a.onboardingClose)
        default:
            a.currentView = viewGrid
        }
    }
    return a, nil
```

**Add `onboardingClientIDSavedMsg` handler**:

```go
case onboardingClientIDSavedMsg:
    a.clientID = m.clientID
    a.onboardingStep = stepOAuth
    a.authStatus = "Opening browser for authorization..."
    return a, tea.Batch(
        prepareOAuthCmd(a.clientID, a.onboardingPort, a.onboardingCodeCh, a.onboardingClose),
        a.onboardingSpinner.Tick,
    )
```

**Add `onboardingRetryMsg` handler**:

```go
case onboardingRetryMsg:
    a.onboardingStep = stepRegister
    a.onboardingError = ""
    a.onboardingInput.Reset()
    a.onboardingInput.Focus()
    return a, nil
```

**Update `authPreparedMsg` handler** — store auth URL in both `a.onboardingAuthURL` and
`a.authURL` (viewAuth uses the latter):

```go
case authPreparedMsg:
    a.onboardingAuthURL = m.authURL
    a.authURL = m.authURL
    if m.browserErr != nil {
        a.authStatus = "Browser didn't open. Visit the URL above manually."
    } else {
        a.authStatus = "Waiting for authorization..."
    }
    return a, waitForCallbackCmd(a.clientID, a.tokenStore, m.verifier, m.redirectURI, m.codeCh, m.serverClose)
```

**Update `authErrorMsg` handler** — branch on current view:

```go
case authErrorMsg:
    if a.currentView == viewOnboarding {
        a.onboardingStep = stepError
        a.onboardingError = m.err.Error()
        return a, nil
    }
    a.authStatus = fmt.Sprintf("Error: %s — press q to quit", m.err.Error())
    return a, nil
```

**Add `spinner.TickMsg` handler** — only ticks when on `stepOAuth`:

```go
case spinner.TickMsg:
    if a.currentView == viewOnboarding && a.onboardingStep == stepOAuth {
        var cmd tea.Cmd
        a.onboardingSpinner, cmd = a.onboardingSpinner.Update(m)
        return a, cmd
    }
    return a, nil
```

Add import `"github.com/charmbracelet/bubbles/spinner"` to `handlers.go`.

### `internal/app/routing.go`

**Intercept `viewOnboarding` before any other routing** — add at the top of `handleKeyMsg`,
before the `viewAuth` guard:

```go
if a.currentView == viewOnboarding {
    return a.handleOnboardingKey(m)
}
```

**`handleOnboardingKey(m tea.KeyMsg) (tea.Model, tea.Cmd)`**:

- `q` or `Ctrl+C` → `tea.Quit` from any step
- `stepRegister`:
  - `Enter` → trim input value; if empty ignore; else `saveClientIDCmd(config.DefaultConfigPath(), clientID)`
  - All other keys → delegate to `a.onboardingInput.Update(m)`
- `stepOAuth`:
  - `c` → `_ = copyToClipboard(a.onboardingAuthURL)` (silent on failure)
  - All others → no-op
- `stepError`:
  - `r` → emit `onboardingRetryMsg{}`
  - `l` → set `a.onboardingStep = stepOAuth`; return `tea.Batch(prepareOAuthCmd(...), a.onboardingSpinner.Tick)`
  - All others → no-op

**Update `viewAuth` key guard** — add `c` for clipboard copy alongside existing `q`/`Ctrl+C`/`Esc`:

```go
if a.currentView == viewAuth {
    if m.Type == tea.KeyCtrlC || (m.Type == tea.KeyRunes && string(m.Runes) == "q") || m.Type == tea.KeyEsc {
        return a, tea.Quit
    }
    if m.Type == tea.KeyRunes && string(m.Runes) == "c" {
        _ = copyToClipboard(a.authURL)
        return a, nil
    }
    return a, nil
}
```

Add imports `"strings"` and `"github.com/initgrep-apps/spotnik/internal/config"` to
`routing.go` as needed.

### Tests

No new test files — the handlers and routing are exercised indirectly by the render tests
(Story 139) and the profile overlay tests (Story 140). However, `go test ./internal/app/... -v`
must continue to pass (no regressions).

If the existing app test suite uses a mock or fake `AppOptions`, update it to supply the new
`CallbackClose` field (or ensure the no-op default path is exercised).

## Acceptance Criteria

- [ ] `splashDismissMsg` routes to `viewOnboarding` when `needsRegister` is true
- [ ] `splashDismissMsg` routes to `viewAuth` (with `prepareOAuthCmd`) when `needsAuth` is true
- [ ] `onboardingClientIDSavedMsg`: sets `a.clientID`, transitions to `stepOAuth`, batches
      `prepareOAuthCmd` + spinner tick
- [ ] `onboardingRetryMsg`: resets to `stepRegister`, clears error, focuses input
- [ ] `authPreparedMsg`: stores both `onboardingAuthURL` and `authURL`; calls
      `waitForCallbackCmd`
- [ ] `authErrorMsg` on `viewOnboarding`: sets `stepError` and `onboardingError`
- [ ] `authErrorMsg` on `viewAuth`: sets `authStatus` string
- [ ] `spinner.TickMsg`: updates spinner only during `viewOnboarding` + `stepOAuth`
- [ ] `handleOnboardingKey`: `q`/`Ctrl+C` quits from any step
- [ ] `stepRegister`: empty Enter ignored; non-empty Enter → `saveClientIDCmd`; other keys → textinput
- [ ] `stepOAuth`: `c` copies `onboardingAuthURL` to clipboard (silent failure)
- [ ] `stepError`: `r` → retry msg; `l` → re-run prepareOAuthCmd; others → no-op
- [ ] `viewAuth` key guard handles `c` for clipboard copy of `authURL`
- [ ] `go test ./internal/app/... -v` passes (no regressions); `make ci` passes

## Tasks

- [ ] Update `splashDismissMsg`, `authPreparedMsg`, `authErrorMsg` handlers in
      `internal/app/handlers.go`
      - test: `go build ./...` → clean
- [ ] Add `onboardingClientIDSavedMsg`, `onboardingRetryMsg`, `spinner.TickMsg` handlers in
      `internal/app/handlers.go`
      - test: `go build ./...` → clean
- [ ] Add `handleOnboardingKey` method and `viewOnboarding` intercept in
      `internal/app/routing.go`; update `viewAuth` guard to handle `c`
      - test: `go test ./internal/app/... -v` → PASS (no regressions)
- [ ] `make ci` passes
