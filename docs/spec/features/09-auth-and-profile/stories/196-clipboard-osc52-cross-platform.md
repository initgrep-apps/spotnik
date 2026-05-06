---
title: "Fix: clipboard copy works cross-platform via OSC 52"
feature: 09-auth-and-profile
status: open
---

## Background

The `c` shortcut for copying the redirect URI (Step 1 of onboarding), the
OAuth URL (Step 2 of onboarding), and the auth URL on `viewAuth` works on
macOS in iTerm2/Terminal.app but **silently fails on every other supported
platform** the project targets.

### Symptoms

| Environment | Behaviour today |
|---|---|
| macOS iTerm2 / Terminal | Works — `pbcopy` is shipped with the OS |
| Linux desktop with `xclip` or `wl-copy` installed | Works |
| Linux desktop without those binaries | Toast says "Copied"; clipboard untouched |
| Docker Ubuntu container | Toast says "Copied"; clipboard untouched (no clipboard binary in slim images) |
| WSL Ubuntu | Toast says "Copied"; clipboard untouched |
| Windows PowerShell | Toast says "Copied"; clipboard untouched (no `pbcopy`/`xclip`/`wl-copy` exists on Windows) |
| Spotnik run over plain SSH from any of the above | `pbcopy` writes to the **remote** machine's clipboard, not the user's local clipboard — toast lies even when the binary "succeeds" |

### Root cause

`internal/app/clipboard.go` shells out to `pbcopy` → `xclip -selection
clipboard` → `wl-copy`. This approach has two structural problems:

1. **Missing platforms.** No Windows backend at all. No fallback when the
   Linux binaries are not installed (Docker, WSL, slim images). A pure-Go
   `clip.exe` shim would only paper over the next problem.
2. **Wrong clipboard.** `pbcopy`/`xclip`/`wl-copy` always target the
   *local* X/Wayland/macOS clipboard of the box where Spotnik runs. Over
   SSH that is the remote machine, not the user's terminal. The user
   wants the URL on their *terminal's* clipboard.

The three call sites in `internal/app/routing.go` then make it worse:

```go
// routing.go:128 — viewAuth
_ = copyToClipboard(a.authURL)             // error swallowed, no toast at all
return a, nil

// routing.go:512 — stepRegister
_ = copyToClipboard(fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort))
return a, a.toasts.Cmd(uikit.Toast{Intent: uikit.ToastSuccess, Title: "Copied"})

// routing.go:536 — stepOAuth
_ = copyToClipboard(a.onboardingAuthURL)
return a, a.toasts.Cmd(uikit.Toast{Intent: uikit.ToastSuccess, Title: "Copied"})
```

Both onboarding sites discard the error and fire a success toast
unconditionally. `viewAuth` is silent on success and silent on failure —
also wrong, because `auth.go:211` renders a `c  copy URL  · q  quit`
hint that promises feedback the user never gets.

### Fix direction

Use **OSC 52** as the sole clipboard mechanism. OSC 52 is an ANSI escape
sequence (`ESC ] 52 ; c ; <base64> BEL`) consumed by the *terminal
emulator* — not the OS — so it writes to whichever clipboard is on the
user's screen, regardless of where Spotnik is running. Forwarded by SSH,
Docker, tmux ≥ 3.2 (with `set -g set-clipboard on`, the default), and
WSL. Supported by every modern terminal users actually run: iTerm2,
Terminal.app (Sonoma+), Windows Terminal, Ghostty, kitty, alacritty,
wezterm, foot, xterm.

The escape generator already lives in `github.com/charmbracelet/x/ansi`,
which is currently an **indirect** dependency in `go.mod` (`v0.11.7`,
pulled in transitively via Bubble Tea). It exposes:

```go
// vendored upstream
func SetSystemClipboard(d string) string {
    if d != "" { d = base64.StdEncoding.EncodeToString([]byte(d)) }
    return "\x1b]52;c;" + d + "\x07"
}
```

Promote `charmbracelet/x/ansi` to a direct dep so the version is pinned
deliberately rather than floating with whatever Bubble Tea pulls in.

### Trade-off accepted

OSC 52 has no acknowledgement protocol. Terminals that ignore the
sequence (very old conhost on Windows pre-Terminal, restricted SSH/CI
environments, `screen` without forwarding) silently drop it; we cannot
detect that. For those edge cases the URL remains visible on screen and
the user can select it manually. We do **not** add a native-binary
fallback because:

- it actively writes to the wrong clipboard over SSH, and
- it adds `os/exec` cross-platform branching for marginal coverage gain
  on terminals our target users do not run.

### What is *not* in scope

- The `copiedFeedbackMsg` / `onboardingCopied` inline `✓  Copied!`
  pattern from story 144's design was **not** implemented — the shipped
  code uses the existing toast. This story keeps the toast pattern for
  consistency with the rest of the app and to avoid a second round of
  render churn.
- `CLAUDE.md` says "Bubble Tea v0.27+" but the project is on `v1.3.10`.
  Doc cleanup tracked separately; not part of this fix.

**Depends on:** Stories 137 (clipboard helper), 138 (onboarding key
routing), 144 (current toast call sites).

## Design

### `go.mod` — promote `charmbracelet/x/ansi` to direct dep

Run:

```
go get github.com/charmbracelet/x/ansi@v0.11.7
go mod tidy
```

Result: the `// indirect` comment on the `charmbracelet/x/ansi` line is
gone. Version is unchanged so no compatibility risk. No other deps
move.

### `internal/app/clipboard.go` — full rewrite

Replace the file in its entirety with:

```go
package app

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// clipboardCopiedMsg is emitted after copyToClipboardCmd attempts a write.
// Err is non-nil only when emitting the OSC 52 sequence to stderr failed
// (broken stderr pipe). A nil Err does NOT guarantee the terminal actually
// wrote to the system clipboard — OSC 52 has no acknowledgement protocol.
type clipboardCopiedMsg struct {
	Err error
}

// copyToClipboardCmd returns a tea.Cmd that copies text to the user's
// terminal clipboard via OSC 52 (ESC ] 52 ; c ; <base64> BEL).
//
// The escape sequence is consumed by the terminal emulator — not the
// host OS — so it targets the clipboard on the screen the user is
// looking at, regardless of whether spotnik is running locally, in
// Docker, over SSH, or inside tmux passthrough.
//
// Writes to os.Stderr because os.Stdout is owned by the Bubble Tea
// renderer; stderr is connected to the same TTY in interactive
// sessions and terminals process escape sequences from either stream.
//
// Terminals without OSC 52 support silently ignore the sequence; we
// cannot detect that. Callers should keep the source URL/text visible
// on screen as a manual-copy fallback.
func copyToClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		_, err := fmt.Fprint(os.Stderr, ansi.SetSystemClipboard(text))
		if err != nil {
			return clipboardCopiedMsg{Err: fmt.Errorf("emitting OSC 52: %w", err)}
		}
		return clipboardCopiedMsg{}
	}
}
```

Notes:

- The old `copyToClipboard(text string) error` is removed. Every caller
  is updated to dispatch the Cmd instead.
- `tea.Cmd` is a goroutine-safe wrapper; writing to stderr inside it
  does not race with the renderer.
- Empty `text` is allowed; `ansi.SetSystemClipboard("")` is well-defined
  by the spec (resets the clipboard) and is treated as success here.

### `internal/app/handlers.go` — handle `clipboardCopiedMsg`

Add a case to the message switch in the existing `Update` dispatch
(near the other onboarding/auth message handlers):

```go
case clipboardCopiedMsg:
    if m.Err != nil {
        return a, a.toasts.Cmd(uikit.Toast{
            Intent: uikit.ToastError,
            Title:  "Copy failed",
            Body:   "Select the URL above to copy manually.",
        })
    }
    return a, a.toasts.Cmd(uikit.Toast{
        Intent: uikit.ToastSuccess,
        Title:  "Copied",
    })
```

Centralising the toast here means **all three** copy sites get
consistent behaviour through a single handler — no per-site toast
calls.

### `internal/app/routing.go` — update all three copy sites

**`viewAuth` (currently silent on success and failure)** — replace
`routing.go:127–130`:

```go
if m.Type == tea.KeyRunes && string(m.Runes) == "c" {
    return a, copyToClipboardCmd(a.authURL)
}
```

**`stepRegister` (currently fires unconditional success toast)** —
replace `routing.go:512–515`:

```go
if m.Type == tea.KeyRunes && string(m.Runes) == "c" && a.onboardingField.Value() == "" {
    return a, copyToClipboardCmd(fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort))
}
```

The empty-input guard is preserved so `c` still passes through to the
text input once the user starts typing a client ID.

**`stepOAuth` (currently fires unconditional success toast)** — replace
`routing.go:536–539`:

```go
if m.Type == tea.KeyRunes && string(m.Runes) == "c" {
    return a, copyToClipboardCmd(a.onboardingAuthURL)
}
```

None of the three sites construct a toast inline anymore — toast
intent is decided in the `clipboardCopiedMsg` handler based on the
error result.

### Tests

#### `internal/app/clipboard_test.go` — new file

```go
package app

import (
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStderr redirects os.Stderr through a pipe for the duration of fn,
// returns whatever fn wrote.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	require.NoError(t, w.Close())
	os.Stderr = orig
	return <-done
}

func TestCopyToClipboardCmd_emitsOSC52ToStderr(t *testing.T) {
	const payload = "https://accounts.spotify.com/authorize?client_id=test"

	out := captureStderr(t, func() {
		msg := copyToClipboardCmd(payload)()
		copied, ok := msg.(clipboardCopiedMsg)
		require.True(t, ok, "expected clipboardCopiedMsg, got %T", msg)
		assert.NoError(t, copied.Err)
	})

	// OSC 52 frame: ESC ] 52 ; c ; <base64> BEL
	assert.True(t, strings.HasPrefix(out, "\x1b]52;c;"), "stderr must start with OSC 52 prefix; got %q", out)
	assert.True(t, strings.HasSuffix(out, "\x07"), "stderr must end with BEL terminator; got %q", out)

	body := strings.TrimSuffix(strings.TrimPrefix(out, "\x1b]52;c;"), "\x07")
	decoded, err := base64.StdEncoding.DecodeString(body)
	require.NoError(t, err, "OSC 52 payload must be valid base64")
	assert.Equal(t, payload, string(decoded))
}

func TestCopyToClipboardCmd_emptyText_emitsResetSequence(t *testing.T) {
	out := captureStderr(t, func() {
		msg := copyToClipboardCmd("")()
		copied, ok := msg.(clipboardCopiedMsg)
		require.True(t, ok)
		assert.NoError(t, copied.Err)
	})

	// Reset form per upstream: \x1b]52;c;\x07 with empty payload.
	assert.Equal(t, "\x1b]52;c;\x07", out)
}

func TestCopyToClipboardCmd_brokenStderr_returnsError(t *testing.T) {
	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, r.Close()) // close read end so the write fails (EPIPE)
	os.Stderr = w
	defer func() {
		_ = w.Close()
		os.Stderr = orig
	}()

	msg := copyToClipboardCmd("anything")()
	copied, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok)
	assert.Error(t, copied.Err)
}
```

Note: `captureStderr` mutates a process global. The three tests above
are not safe to run with `-parallel`. Mark the file as such by simply
not calling `t.Parallel()` in any of them.

#### `internal/app/handlers_test.go` — extend existing file

```go
func TestUpdate_clipboardCopiedMsg_success_firesSuccessToast(t *testing.T) {
	a := newTestApp(t)
	_, cmd := a.Update(clipboardCopiedMsg{})
	require.NotNil(t, cmd)
	msg := cmd()
	toastMsg, ok := msg.(uikit.ToastEnqueueMsg) // or whatever the project's toast Cmd produces
	require.True(t, ok)
	assert.Equal(t, uikit.ToastSuccess, toastMsg.Toast.Intent)
	assert.Equal(t, "Copied", toastMsg.Toast.Title)
}

func TestUpdate_clipboardCopiedMsg_error_firesErrorToast(t *testing.T) {
	a := newTestApp(t)
	_, cmd := a.Update(clipboardCopiedMsg{Err: errors.New("boom")})
	require.NotNil(t, cmd)
	msg := cmd()
	toastMsg, ok := msg.(uikit.ToastEnqueueMsg)
	require.True(t, ok)
	assert.Equal(t, uikit.ToastError, toastMsg.Toast.Intent)
	assert.Equal(t, "Copy failed", toastMsg.Toast.Title)
}
```

The exact `uikit.ToastEnqueueMsg` type name should match what
`a.toasts.Cmd(...)` actually emits in the codebase — adjust during
implementation.

#### `internal/app/routing_test.go` — extend existing file

```go
func TestHandleKeyMsg_viewAuth_c_dispatchesCopyCmd(t *testing.T) {
	a := newTestApp(t)
	a.currentView = viewAuth
	a.authURL = "https://example.test/authorize"

	_, cmd := a.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd)

	msg := cmd()
	copied, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok, "viewAuth 'c' must dispatch copyToClipboardCmd; got %T", msg)
	assert.NoError(t, copied.Err)
}

func TestHandleOnboardingKey_stepRegister_c_emptyInput_dispatchesCopyCmd(t *testing.T) {
	a := newTestApp(t)
	a.currentView = viewOnboarding
	a.onboardingStep = stepRegister
	a.onboardingPort = 8888
	// onboardingField empty by default

	_, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok)
}

func TestHandleOnboardingKey_stepRegister_c_typing_passesThroughToInput(t *testing.T) {
	a := newTestApp(t)
	a.currentView = viewOnboarding
	a.onboardingStep = stepRegister
	a.onboardingField.SetValue("ab")  // user has started typing

	_, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	// Should NOT produce a clipboardCopiedMsg — the 'c' goes to the input.
	if cmd != nil {
		msg := cmd()
		_, isCopy := msg.(clipboardCopiedMsg)
		assert.False(t, isCopy, "c should pass through to input when typing has begun")
	}
	assert.Equal(t, "abc", a.onboardingField.Value())
}

func TestHandleOnboardingKey_stepOAuth_c_dispatchesCopyCmd(t *testing.T) {
	a := newTestApp(t)
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth
	a.onboardingAuthURL = "https://accounts.spotify.com/authorize?x=1"

	_, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok)
}
```

`newTestApp(t)` already exists in render_test.go per story 144's plan;
extend if it's missing the fields above.

## Acceptance Criteria

- [ ] `github.com/charmbracelet/x/ansi v0.11.7` is a **direct** dep in `go.mod` (no `// indirect`)
- [ ] `internal/app/clipboard.go` no longer imports `os/exec` or `strings` for command building
- [ ] `copyToClipboardCmd(text string) tea.Cmd` exists and emits `ansi.SetSystemClipboard(text)` to `os.Stderr`
- [ ] `clipboardCopiedMsg{Err error}` type exists
- [ ] `Update` handles `clipboardCopiedMsg`: success → `ToastSuccess "Copied"`; non-nil `Err` → `ToastError "Copy failed"` with body "Select the URL above to copy manually."
- [ ] `viewAuth` `c` key dispatches `copyToClipboardCmd(a.authURL)` and produces a toast (was previously silent)
- [ ] `stepRegister` `c` key dispatches `copyToClipboardCmd(<redirect URI>)` only when `onboardingField.Value() == ""`; passes through to input otherwise
- [ ] `stepOAuth` `c` key dispatches `copyToClipboardCmd(a.onboardingAuthURL)`
- [ ] No call site constructs a clipboard-related toast inline — all toasts originate from the `clipboardCopiedMsg` handler
- [ ] Manual smoke test in **all** of: macOS iTerm2, Linux desktop terminal, Docker Ubuntu container, WSL Ubuntu, Windows Terminal — pressing `c` on each of the three screens places the URL on the **user's terminal** clipboard (verifiable by pasting into any other window)
- [ ] All `TestCopyToClipboardCmd_*`, `TestUpdate_clipboardCopiedMsg_*`, `TestHandleKeyMsg_viewAuth_c_*`, `TestHandleOnboardingKey_stepRegister_c_*`, `TestHandleOnboardingKey_stepOAuth_c_*` tests pass
- [ ] `make ci` passes (lint + test + 80% coverage)

## Tasks

- [ ] Run `go get github.com/charmbracelet/x/ansi@v0.11.7 && go mod tidy`; verify the line in `go.mod` no longer carries `// indirect`
      - test: `go build ./...` → clean
- [ ] Write failing tests in `internal/app/clipboard_test.go` for `copyToClipboardCmd`
      (success, empty payload, broken-stderr error)
      - test: `go test ./internal/app/... -run "TestCopyToClipboardCmd" -v` → compile errors (helper not yet rewritten)
- [ ] Rewrite `internal/app/clipboard.go` with `copyToClipboardCmd` and `clipboardCopiedMsg` per the Design section; delete the old `copyToClipboard(string) error`
      - test: `TestCopyToClipboardCmd_*` → PASS
- [ ] Add the `case clipboardCopiedMsg:` handler in `internal/app/handlers.go`
      - test: `go build ./...` → clean
- [ ] Write failing tests in `internal/app/handlers_test.go` for the success and error toast paths
      - test: `go test ./internal/app/... -run "TestUpdate_clipboardCopiedMsg" -v` → compile errors / fails until implementation lands
- [ ] Update the three `c` key sites in `internal/app/routing.go` (`viewAuth`, `stepRegister`, `stepOAuth`) to return `copyToClipboardCmd(...)` and remove inline toast construction; remove any now-unused imports (`os/exec`, `strings` if present)
      - test: `TestHandleKeyMsg_viewAuth_c_*`, `TestHandleOnboardingKey_stepRegister_c_*`, `TestHandleOnboardingKey_stepOAuth_c_*` → PASS
- [ ] Manual cross-platform smoke test on iTerm2 (macOS), Linux X11 desktop, Docker Ubuntu, WSL Ubuntu, Windows Terminal; record results in PR description
      - test: paste-after-`c` produces the URL on the *user's* clipboard in every environment
- [ ] `make ci` → PASS
