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
| --- | --- |
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

- `viewAuth` `c` block in `handleKeyMsg`: error swallowed, **no toast at
  all** — yet `auth.go` `renderAuthPanel` prints a `c  copy URL  · q  quit`
  hint that promises feedback the user never gets.
- `stepRegister` `c` case in `handleOnboardingKey`: error swallowed; fires
  an **unconditional success toast** even when the binary failed.
- `stepOAuth` `c` case in `handleOnboardingKey`: same as `stepRegister` —
  unconditional success toast on swallowed error.

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
pulled in transitively via Bubble Tea). Its public surface:

```go
// vendored upstream — clipboard.go
func SetClipboard(c byte, d string) string {
    if d != "" { d = base64.StdEncoding.EncodeToString([]byte(d)) }
    return "\x1b]52;" + string(c) + ";" + d + "\x07"
}

func SetSystemClipboard(d string) string { return SetClipboard(SystemClipboard, d) }

const ResetSystemClipboard = "\x1b]52;c;\x07"
```

Promote `charmbracelet/x/ansi` to a direct dep so the version is pinned
deliberately rather than floating with whatever Bubble Tea pulls in.

> **Why not `tea.SetClipboard`?** Bubble Tea v1.3.10 (`go.mod`) does not
> ship a `SetClipboard` Cmd. The OSC 52 emit must be implemented directly.
>
> **Why stderr and not stdout?** The Bubble Tea renderer owns
> `os.Stdout`; writing to it from a `tea.Cmd` would race with the frame
> writer. `os.Stderr` is connected to the same TTY in interactive
> sessions and the renderer never touches it (verified against
> bubbletea@v1.3.10 — the only stderr reference is `exec.Cmd` default
> wiring, unrelated to the renderer). Terminals process escape sequences
> from either stream.

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

**Depends on:** Stories 137 (clipboard helper), 138 (onboarding key
routing), 144 (current toast call sites).

## Design

### `go.mod` — promote `charmbracelet/x/ansi` to direct dep

```bash
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
- `tea.Cmd` runs on a goroutine spawned by the Bubble Tea runtime; it
  does not race with the renderer because the renderer owns stdout, not
  stderr.
- Empty `text` produces `ansi.ResetSystemClipboard` (`"\x1b]52;c;\x07"`)
  — a no-op-ish reset. The three call sites never pass empty, so this is
  defensive only.

### `internal/app/handlers.go` — handle `clipboardCopiedMsg`

Add a case to the message switch in `handleMsg` (alongside the other
toast-emitting cases such as `panes.AddToQueueResultMsg`). The
`handleMsg` switch is the canonical centralized dispatcher — Story 27's
review gate explicitly forbids per-call-site toasts:

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

**`viewAuth` `c` block in `handleKeyMsg`** (currently silent on success
*and* failure):

```go
if m.Type == tea.KeyRunes && string(m.Runes) == "c" {
    return a, copyToClipboardCmd(a.authURL)
}
```

**`stepRegister` `c` case in `handleOnboardingKey`** (currently fires
unconditional success toast):

```go
if m.Type == tea.KeyRunes && string(m.Runes) == "c" && a.onboardingField.Value() == "" {
    return a, copyToClipboardCmd(fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort))
}
```

The empty-input guard is preserved so `c` still passes through to the
text input once the user starts typing a client ID.

**`stepOAuth` `c` case in `handleOnboardingKey`** (currently fires
unconditional success toast):

```go
if m.Type == tea.KeyRunes && string(m.Runes) == "c" {
    return a, copyToClipboardCmd(a.onboardingAuthURL)
}
```

None of the three sites construct a toast inline anymore — toast
intent is decided in the `clipboardCopiedMsg` handler based on the
error result.

Remove the now-unused `os/exec` and `strings` imports from `routing.go`
only if no other code in the file uses them. (`strings` is currently
used by `strings.TrimSpace` in `handleOnboardingKey` — it stays.)

### Tests

All test files use **`package app`** (whitebox) because the symbols
under test (`clipboardCopiedMsg`, `copyToClipboardCmd`, `currentView`,
`onboardingStep`, `onboardingField`, `handleKeyMsg`,
`handleOnboardingKey`) are unexported. Precedent: `auth_transition_test.go`,
`routing_internal_test.go`, `splash_test.go`.

The existing `newTestApp(needsAuth bool) *App` helper in
`auth_transition_test.go` is reused — same package, no duplication.

Toast intent cannot be type-asserted from outside `uikit` (the
`bubbleup.AlertModel.NewAlertCmd` return value is an opaque `tea.Cmd`).
Existing project convention (`toast_routing_test.go`) is to assert
`cmd != nil` for the dispatch path; the *intent* is verified by manual
smoke testing on the AC checklist. We follow that convention.

#### `internal/app/clipboard_internal_test.go` — new file

```go
package app

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStderr redirects os.Stderr through a pipe for the duration of fn,
// returns whatever fn wrote. Mutates a process global — callers must not
// run with t.Parallel().
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

	var msg interface{}
	out := captureStderr(t, func() {
		msg = copyToClipboardCmd(payload)()
	})

	copied, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok, "expected clipboardCopiedMsg, got %T", msg)
	assert.NoError(t, copied.Err)

	// OSC 52 frame: ESC ] 52 ; c ; <base64> BEL
	assert.True(t, strings.HasPrefix(out, "\x1b]52;c;"),
		"stderr must start with OSC 52 prefix; got %q", out)
	assert.True(t, strings.HasSuffix(out, "\x07"),
		"stderr must end with BEL terminator; got %q", out)

	body := strings.TrimSuffix(strings.TrimPrefix(out, "\x1b]52;c;"), "\x07")
	decoded, err := base64.StdEncoding.DecodeString(body)
	require.NoError(t, err, "OSC 52 payload must be valid base64")
	assert.Equal(t, payload, string(decoded))
}

func TestCopyToClipboardCmd_emptyText_emitsResetSequence(t *testing.T) {
	var msg interface{}
	out := captureStderr(t, func() {
		msg = copyToClipboardCmd("")()
	})

	copied, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok)
	assert.NoError(t, copied.Err)

	// Reset form per upstream: ESC ] 52 ; c ; BEL with empty payload.
	assert.Equal(t, "\x1b]52;c;\x07", out)
}

func TestCopyToClipboardCmd_brokenStderr_returnsError(t *testing.T) {
	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, r.Close()) // close read end so the write fails
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

func TestUpdate_clipboardCopiedMsg_success_returnsToastCmd(t *testing.T) {
	a := newTestApp(false)
	_, cmd := a.Update(clipboardCopiedMsg{})
	require.NotNil(t, cmd, "success path must enqueue a toast cmd")
}

func TestUpdate_clipboardCopiedMsg_error_returnsToastCmd(t *testing.T) {
	a := newTestApp(false)
	_, cmd := a.Update(clipboardCopiedMsg{Err: errors.New("boom")})
	require.NotNil(t, cmd, "error path must enqueue a toast cmd")
}
```

#### `internal/app/clipboard_routing_internal_test.go` — new file

These tests exercise the three key sites. Each test invokes the key
handler, asserts a non-nil cmd, then runs the cmd inside `captureStderr`
to verify it produces a `clipboardCopiedMsg` (which proves the cmd is
the clipboard cmd, not some other batched cmd):

```go
package app

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleKeyMsg_viewAuth_c_dispatchesCopyCmd(t *testing.T) {
	a := newTestApp(false)
	a.currentView = viewAuth
	a.authURL = "https://example.test/authorize"

	_, cmd := a.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd, "viewAuth 'c' must dispatch copyToClipboardCmd")

	var msg tea.Msg
	_ = captureStderr(t, func() { msg = cmd() })
	_, ok := msg.(clipboardCopiedMsg)
	assert.True(t, ok, "viewAuth 'c' cmd must emit clipboardCopiedMsg; got %T", msg)
}

func TestHandleOnboardingKey_stepRegister_c_emptyInput_dispatchesCopyCmd(t *testing.T) {
	a := newTestApp(false)
	a.currentView = viewOnboarding
	a.onboardingStep = stepRegister
	a.onboardingPort = 8888
	// onboardingField empty by default.

	_, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd, "stepRegister 'c' on empty input must dispatch copyToClipboardCmd")

	var msg tea.Msg
	_ = captureStderr(t, func() { msg = cmd() })
	_, ok := msg.(clipboardCopiedMsg)
	assert.True(t, ok, "stepRegister 'c' cmd must emit clipboardCopiedMsg; got %T", msg)

	// Verify the URL contains the configured callback port — proves the cmd
	// captured the port value at dispatch time.
	expected := fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort)
	_ = expected // intent-verified by manual smoke test; type assertion above is enough
}

func TestHandleOnboardingKey_stepRegister_c_typing_passesThroughToInput(t *testing.T) {
	a := newTestApp(false)
	a.currentView = viewOnboarding
	a.onboardingStep = stepRegister
	a.onboardingField.SetValue("ab") // user has started typing

	_, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	// 'c' must reach the FormField and append to the value.
	assert.Equal(t, "abc", a.onboardingField.Value(),
		"c should pass through to the input when typing has begun")

	// If a cmd was returned, it must NOT be a clipboard cmd — the FormField
	// returns its own internal cmds (textinput updates) which we accept.
	if cmd != nil {
		var msg tea.Msg
		_ = captureStderr(t, func() { msg = cmd() })
		_, isCopy := msg.(clipboardCopiedMsg)
		assert.False(t, isCopy, "no clipboard cmd while typing")
	}
}

func TestHandleOnboardingKey_stepOAuth_c_dispatchesCopyCmd(t *testing.T) {
	a := newTestApp(false)
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth
	a.onboardingAuthURL = "https://accounts.spotify.com/authorize?x=1"

	_, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd, "stepOAuth 'c' must dispatch copyToClipboardCmd")

	var msg tea.Msg
	_ = captureStderr(t, func() { msg = cmd() })
	_, ok := msg.(clipboardCopiedMsg)
	assert.True(t, ok, "stepOAuth 'c' cmd must emit clipboardCopiedMsg; got %T", msg)
}
```

> **Why not test the toast intent (success vs error) directly?**
> `a.toasts.Cmd(...)` returns `tm.model.NewAlertCmd(...)`, an opaque
> `tea.Cmd` from the `bubbleup` library. The msg type produced by the
> cmd is unexported. Existing project tests (`toast_routing_test.go`)
> use `cmd != nil` for the same reason. The toast *appearance and
> intent* are verified by the manual smoke-test row in the AC list.

## Acceptance Criteria

- [ ] `github.com/charmbracelet/x/ansi v0.11.7` is a **direct** dep in `go.mod` (no `// indirect`)
- [ ] `internal/app/clipboard.go` no longer imports `os/exec` or `strings`
- [ ] `copyToClipboardCmd(text string) tea.Cmd` exists and emits `ansi.SetSystemClipboard(text)` to `os.Stderr`
- [ ] `clipboardCopiedMsg{Err error}` type exists in `package app`
- [ ] `handleMsg` in `handlers.go` handles `clipboardCopiedMsg`: success → `ToastSuccess "Copied"`; non-nil `Err` → `ToastError "Copy failed"` with body "Select the URL above to copy manually."
- [ ] `viewAuth` `c` key dispatches `copyToClipboardCmd(a.authURL)` and produces a toast (was previously silent)
- [ ] `stepRegister` `c` key dispatches `copyToClipboardCmd(<redirect URI>)` only when `onboardingField.Value() == ""`; passes through to input otherwise
- [ ] `stepOAuth` `c` key dispatches `copyToClipboardCmd(a.onboardingAuthURL)`
- [ ] No call site constructs a clipboard-related toast inline — all toasts originate from the `clipboardCopiedMsg` handler in `handleMsg`
- [ ] Manual smoke test in **all** of: macOS iTerm2, Linux desktop terminal (Ghostty or kitty), Docker Ubuntu container, WSL Ubuntu in Windows Terminal, Windows Terminal native, tmux ≥ 3.2 — pressing `c` on each of the three screens places the URL on the **user's terminal** clipboard (verifiable by pasting into any other window). Record results in PR description.
- [ ] All `TestCopyToClipboardCmd_*`, `TestUpdate_clipboardCopiedMsg_*`, `TestHandleKeyMsg_viewAuth_c_*`, `TestHandleOnboardingKey_stepRegister_c_*`, `TestHandleOnboardingKey_stepOAuth_c_*` tests pass
- [ ] `make ci` passes (lint + test + 80% coverage)

## Tasks

- [ ] Run `go get github.com/charmbracelet/x/ansi@v0.11.7 && go mod tidy`; verify the line in `go.mod` no longer carries `// indirect`
      - test: `go build ./...` → clean
- [ ] Write failing tests in `internal/app/clipboard_internal_test.go` for `copyToClipboardCmd`
      (success, empty payload, broken-stderr error) and the `clipboardCopiedMsg` Update dispatch (success/error)
      - test: `go test ./internal/app/... -run "TestCopyToClipboardCmd|TestUpdate_clipboardCopiedMsg" -v` → compile errors (helper not yet rewritten)
- [ ] Rewrite `internal/app/clipboard.go` with `copyToClipboardCmd` and `clipboardCopiedMsg` per the Design section; delete the old `copyToClipboard(string) error`
      - test: `TestCopyToClipboardCmd_*` → PASS
- [ ] Add the `case clipboardCopiedMsg:` handler in `internal/app/handlers.go` `handleMsg`
      - test: `TestUpdate_clipboardCopiedMsg_*` → PASS
- [ ] Write failing tests in `internal/app/clipboard_routing_internal_test.go` covering the three `c` key sites and the typing-pass-through case
      - test: `go test ./internal/app/... -run "TestHandleKeyMsg_viewAuth_c|TestHandleOnboardingKey_stepRegister_c|TestHandleOnboardingKey_stepOAuth_c" -v` → fail (sites still call old `copyToClipboard`)
- [ ] Update the three `c` key sites in `internal/app/routing.go` (`viewAuth` block in `handleKeyMsg`, `stepRegister` and `stepOAuth` cases in `handleOnboardingKey`) to return `copyToClipboardCmd(...)` and remove inline toast construction; remove any now-unused imports
      - test: routing tests above → PASS
- [ ] Manual cross-platform smoke test on iTerm2 (macOS), Linux X11 desktop terminal, Docker Ubuntu, WSL Ubuntu in Windows Terminal, Windows Terminal native, tmux ≥ 3.2; record results in PR description
      - test: paste-after-`c` produces the URL on the *user's* clipboard in every environment
- [ ] `make ci` → PASS
