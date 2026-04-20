---
title: "App — viewOnboarding Mode, Struct Fields, Commands, Clipboard"
feature: 09-auth-and-profile
status: open
---

## Background

This story wires the onboarding state machine into the `App` struct and implements the commands
that drive it. No UI rendering happens here — that is Story 139. No key routing or message
handlers — that is Story 138. This story produces a compiling, testable foundation.

**New elements introduced:**
- `viewOnboarding` constant and `stepRegister / stepOAuth / stepError` sub-step constants
- New fields on `App`: `needsRegister`, `onboardingInput`, `onboardingSpinner`,
  `onboardingPort`, `onboardingCodeCh`, `onboardingClose`, `onboardingAuthURL`,
  `onboardingError`, `onboardingStep`
- Extended `AppOptions`: `NeedsRegister`, `CallbackPort`, `CallbackCodeCh`, `CallbackClose`
- New message types: `onboardingClientIDSavedMsg`, `onboardingRetryMsg`
- Renamed command: `prepareOAuthCmd` (replaces `prepareAuthCmd`; server pre-started, no
  server startup inside the command)
- New command: `saveClientIDCmd`
- New file: `internal/app/clipboard.go` — `copyToClipboard(text string) error`

**Depends on:** Stories 134, 135, 136 (`CallbackPort` in config, `StartCallbackServer(port)`,
`AppOptions.NeedsRegister` + `CallbackCodeCh` fields expected by `runApp`).

## Design

### `internal/app/app.go`

**View mode constants** — add `viewOnboarding` between `viewSplash` and `viewAuth`:

```go
const (
    viewSplash      viewMode = iota
    viewOnboarding           // First-time registration + OAuth flow
    viewAuth                 // OAuth-only for returning user with no tokens
    viewGrid
)
```

**Onboarding step constants** — new block after the view mode block:

```go
const (
    stepRegister = iota // Step 1: client ID input + instructions
    stepOAuth           // Step 2: browser wait + full URL display
    stepError           // Step 2 error: retry options
)
```

**New imports** in `app.go`:

```go
"github.com/charmbracelet/bubbles/spinner"
"github.com/charmbracelet/bubbles/textinput"
```

**New `App` fields** (add after `needsAuth bool`):

```go
needsRegister     bool
onboardingStep    int
onboardingInput   textinput.Model
onboardingError   string
onboardingPort    int
onboardingCodeCh  <-chan api.CallbackResult
onboardingClose   func()
onboardingAuthURL string
onboardingSpinner spinner.Model
```

**Extended `AppOptions`**:

```go
type AppOptions struct {
    NeedsRegister   bool
    NeedsAuth       bool
    ClientID        string
    TokenStore      keychain.TokenStore
    TokenBaseURL    string
    Version         string
    CallbackPort    int
    CallbackCodeCh  <-chan api.CallbackResult
    CallbackClose   func()
}
```

**`New()` initialisation** — add after existing field assignments:

```go
ti := textinput.New()
ti.Placeholder = "your-client-id-here"
ti.CharLimit = 64
ti.Width = 60

sp := spinner.New()
sp.Spinner = spinner.Dot

callbackClose := opts.CallbackClose
if callbackClose == nil {
    callbackClose = func() {}
}
```

And include in the `App` struct literal:

```go
needsRegister:     opts.NeedsRegister,
onboardingInput:   ti,
onboardingSpinner: sp,
onboardingPort:    opts.CallbackPort,
onboardingCodeCh:  opts.CallbackCodeCh,
onboardingClose:   callbackClose,
```

**`Init()` — defer pane init when unauthenticated** — only the splash timer and alerts init
fire when `needsRegister || needsAuth` is true. All pane init commands run after
`authSuccessMsg` arrives:

```go
func (a *App) Init() tea.Cmd {
    splashTimer := tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
        return splashDismissMsg{}
    })
    alertsInitCmd := a.alerts.Init()

    if a.needsRegister || a.needsAuth {
        return tea.Batch(splashTimer, alertsInitCmd)
    }
    // ... existing full init unchanged ...
}
```

### `internal/app/auth.go`

**New message types** (add at the top, after existing types):

```go
// onboardingClientIDSavedMsg is sent when client_id has been written to config.toml.
type onboardingClientIDSavedMsg struct {
    clientID string
}

// onboardingRetryMsg is sent when the user presses 'r' on the error screen.
type onboardingRetryMsg struct{}
```

**`saveClientIDCmd(path, clientID string) tea.Cmd`** — calls `config.SetClientID`, returns
`onboardingClientIDSavedMsg` on success, `authErrorMsg` on failure:

```go
func saveClientIDCmd(path, clientID string) tea.Cmd {
    return func() tea.Msg {
        if err := config.SetClientID(path, clientID); err != nil {
            return authErrorMsg{err: fmt.Errorf("saving client ID: %w", err)}
        }
        return onboardingClientIDSavedMsg{clientID: clientID}
    }
}
```

**`prepareOAuthCmd`** — replaces `prepareAuthCmd`. The callback server is already running;
this command only generates PKCE credentials, builds the auth URL, and opens the browser:

```go
func prepareOAuthCmd(clientID string, port int, codeCh <-chan api.CallbackResult, serverClose func()) tea.Cmd {
    return func() tea.Msg {
        verifier, err := api.GenerateCodeVerifier()
        if err != nil {
            return authErrorMsg{err: fmt.Errorf("generating PKCE verifier: %w", err)}
        }
        challenge := api.ComputeCodeChallenge(verifier)
        redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
        authURL := api.BuildAuthURL(clientID, redirectURI, challenge, api.SpotifyScopes)
        browserErr := api.OpenBrowser(authURL)
        return authPreparedMsg{
            authURL:     authURL,
            codeCh:      codeCh,
            verifier:    verifier,
            redirectURI: redirectURI,
            serverClose: serverClose,
            browserErr:  browserErr,
        }
    }
}
```

`authPreparedMsg` gains `serverClose func()` so the handler can pass it to
`waitForCallbackCmd`. The `waitForCallbackCmd` signature and body remain unchanged except it
now calls `defer serverClose()` rather than closing a server it started itself.

Remove the old `prepareAuthCmd` function entirely after updating all call sites.

### `internal/app/clipboard.go` — new file

```go
package app

import (
    "fmt"
    "os/exec"
    "strings"
)

// copyToClipboard attempts to copy text to the system clipboard.
// Tries pbcopy (macOS), xclip -selection clipboard (Linux X11), wl-copy (Wayland).
// Returns an error if all methods fail.
// Callers treat failure silently — the URL remains visible for manual selection.
func copyToClipboard(text string) error {
    commands := [][]string{
        {"pbcopy"},
        {"xclip", "-selection", "clipboard"},
        {"wl-copy"},
    }
    for _, args := range commands {
        cmd := exec.Command(args[0], args[1:]...)
        cmd.Stdin = strings.NewReader(text)
        if err := cmd.Run(); err == nil {
            return nil
        }
    }
    return fmt.Errorf("no clipboard command available (tried pbcopy, xclip, wl-copy)")
}
```

### Tests — `internal/app/auth_test.go`

Write **failing** tests first:

```go
func TestSaveClientIDCmd_writesAndEmitsMsg(t *testing.T) {
    // Arrange: temp config file with [spotify] section, no client_id.
    // Act: cmd := saveClientIDCmd(path, "testclientid"); msg := cmd().
    // Assert: msg is onboardingClientIDSavedMsg{clientID: "testclientid"}.
    // Assert: config.Load(path).ClientID == "testclientid".
}

func TestSaveClientIDCmd_writeError_emitsErrorMsg(t *testing.T) {
    // Act: saveClientIDCmd("/nonexistent/path/config.toml", "id")().
    // Assert: msg is authErrorMsg.
}
```

## Acceptance Criteria

- [ ] `viewOnboarding` constant exists and is between `viewSplash` and `viewAuth`
- [ ] `stepRegister`, `stepOAuth`, `stepError` constants defined in `app.go`
- [ ] `App` struct has all 9 new onboarding fields
- [ ] `AppOptions` has `NeedsRegister`, `CallbackPort`, `CallbackCodeCh`, `CallbackClose`
- [ ] `New()` initialises `textinput.Model` and `spinner.Model`; `onboardingClose` defaults to
      no-op when `opts.CallbackClose == nil`
- [ ] `Init()` only runs splash timer + alerts when `needsRegister || needsAuth`
- [ ] `onboardingClientIDSavedMsg` and `onboardingRetryMsg` types defined in `auth.go`
- [ ] `saveClientIDCmd` returns `onboardingClientIDSavedMsg` on success, `authErrorMsg` on failure
- [ ] `prepareOAuthCmd` takes `(clientID string, port int, codeCh, serverClose)` — does not start
      a server
- [ ] `prepareAuthCmd` removed; all former call sites updated to `prepareOAuthCmd`
- [ ] `copyToClipboard` in `clipboard.go` tries pbcopy, xclip, wl-copy in order
- [ ] `TestSaveClientIDCmd_*` tests pass; `go build ./...` passes; `make ci` passes

## Tasks

- [ ] Write failing tests in `internal/app/auth_test.go` for `saveClientIDCmd`
      - test: `go test ./internal/app/... -run "TestSaveClientIDCmd" -v` → compile errors
- [ ] Add `viewOnboarding`, step constants, new `App` fields, updated `AppOptions`, updated
      `New()` and `Init()` in `internal/app/app.go`
      - test: `go build ./...` → clean (may require stub methods for new view in render/routing)
- [ ] Add `onboardingClientIDSavedMsg`, `onboardingRetryMsg`, `saveClientIDCmd`,
      `prepareOAuthCmd` in `internal/app/auth.go`; remove `prepareAuthCmd`; update all call
      sites
      - test: `TestSaveClientIDCmd_*` → PASS; `go build ./...` clean
- [ ] Create `internal/app/clipboard.go` with `copyToClipboard`
      - test: `go build ./...` clean
- [ ] `make ci` passes
