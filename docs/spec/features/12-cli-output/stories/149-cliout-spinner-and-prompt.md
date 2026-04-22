---
title: "Spinner + Prompt — dynamic message types and OAuth flow integration"
feature: 12-cli-output
status: open
---

## Background

Stories 146–148 shipped the static `internal/cliout` surface and wired palette
resolution. `Spinner` and `Prompt` message types exist but `render()` panics.

This story:

1. Implements `Spinner` — a hand-rolled goroutine spinner that redraws on TTY
   using `\r`, falls back to a single static line on non-TTY, supports
   `Done/Fail/Stop`, hides/restores the cursor, and registers a SIGINT handler.
2. Implements `Prompt` — a `bufio.Scanner`-based validated-input loop with a
   3-retry cap, styled label, placeholder, and `ErrAborted` on EOF/Ctrl+C.
3. Integrates both into `RunAuthFlow` (spinner for callback wait) and
   `runRegister` (prompt for client ID).
4. Updates `cliout.SetTestMode(true)` to disable animation and SIGINT handlers so
   tests stay deterministic.

**Depends on:** Stories 146 (package), 147 (call-site migration done), 148
(palette resolved at startup — affects spinner colour choice).

## Design

### `internal/cliout/spinner.go`

```go
package cliout

import (
    "context"
    "fmt"
    "io"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "github.com/charmbracelet/lipgloss"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const spinnerInterval = 100 * time.Millisecond

// SpinnerHandle controls a running spinner.
type SpinnerHandle struct {
    w       io.Writer
    text    string
    cancel  context.CancelFunc
    done    chan struct{} // closed when goroutine exits
    onTTY   bool
    resolved bool
    mu      sync.Mutex
}

// StartSpinner writes the initial frame and returns a handle. On non-TTY or
// in test mode, writes a single static line ("◌ <text>") and returns a no-op
// handle.
func StartSpinner(w io.Writer, text string) *SpinnerHandle {
    if rec := activeRecorder(); rec != nil {
        rec.append(Spinner{Text: text})
        return &SpinnerHandle{w: w, text: text} // no-op
    }

    onTTY := isTTY(w) && !checkNoColor() && !inTestMode()
    h := &SpinnerHandle{w: w, text: text, onTTY: onTTY, done: make(chan struct{})}

    if !onTTY {
        // Static single-line output; render as Step{Pending}.
        Write(w, Step{Status: Pending, Text: text})
        close(h.done)
        return h
    }

    installSIGINTHandler()
    registerHandle(h)

    ctx, cancel := context.WithCancel(context.Background())
    h.cancel = cancel

    // Hide cursor.
    _, _ = fmt.Fprint(w, "\x1b[?25l")

    go h.run(ctx)
    return h
}

// run redraws the spinner every spinnerInterval until ctx cancels.
func (h *SpinnerHandle) run(ctx context.Context) {
    defer close(h.done)
    p := current()
    frameStyle := lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
    textStyle := lipgloss.NewStyle().Foreground(p.Muted)
    padding := "  " // 2-char indent to match Write/WriteInline

    ticker := time.NewTicker(spinnerInterval)
    defer ticker.Stop()

    i := 0
    render := func() {
        line := padding + frameStyle.Render(spinnerFrames[i%len(spinnerFrames)]) + " " + textStyle.Render(h.text)
        _, _ = fmt.Fprint(h.w, "\r\x1b[K"+line)
        i++
    }
    render() // immediate first frame

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            render()
        }
    }
}

// Done resolves the spinner with a success step.
func (h *SpinnerHandle) Done(text string) { h.resolve(StatusSuccess, text) }

// Fail resolves the spinner with a failure step.
func (h *SpinnerHandle) Fail(text string) { h.resolve(StatusFailure, text) }

// Stop cancels silently. Idempotent.
func (h *SpinnerHandle) Stop() { h.resolve(-1, "") }

func (h *SpinnerHandle) resolve(s Status, text string) {
    h.mu.Lock()
    if h.resolved {
        h.mu.Unlock()
        return
    }
    h.resolved = true
    h.mu.Unlock()

    if h.cancel != nil {
        h.cancel()
        <-h.done // wait for goroutine to exit before rewriting the line
    }

    if h.onTTY {
        // Clear the spinner line and restore cursor.
        _, _ = fmt.Fprint(h.w, "\r\x1b[K\x1b[?25h")
    }

    unregisterHandle(h)

    if s == -1 {
        // Silent cancel — no resolution line.
        if h.onTTY {
            _, _ = fmt.Fprintln(h.w) // leave cursor on a fresh line
        }
        return
    }

    // Print resolution as a standard Step.
    if h.onTTY {
        WriteInline(h.w, Step{Status: s, Text: text})
    } else {
        // Non-TTY: the "◌ <text>" line was printed at start; append the resolution.
        WriteInline(h.w, Step{Status: s, Text: text})
    }
}

// Package-level registry of active handles for SIGINT cleanup.
var (
    handlesMu    sync.Mutex
    handles      = map[*SpinnerHandle]struct{}{}
    sigOnce      sync.Once
)

func registerHandle(h *SpinnerHandle) {
    handlesMu.Lock()
    defer handlesMu.Unlock()
    handles[h] = struct{}{}
}

func unregisterHandle(h *SpinnerHandle) {
    handlesMu.Lock()
    defer handlesMu.Unlock()
    delete(handles, h)
}

func installSIGINTHandler() {
    sigOnce.Do(func() {
        if inTestMode() {
            return
        }
        ch := make(chan os.Signal, 1)
        signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
        go func() {
            <-ch
            handlesMu.Lock()
            for h := range handles {
                if h.cancel != nil {
                    h.cancel()
                }
                if h.onTTY {
                    _, _ = fmt.Fprint(h.w, "\r\x1b[K\x1b[?25h")
                }
            }
            handlesMu.Unlock()
            os.Exit(130) // standard SIGINT exit code
        }()
    })
}

// Spinner.render is still unreachable — Spinner messages only flow through
// StartSpinner (dynamic) or the Recorder (test capture). Keep the panic as a
// safety net: if a caller ever passes Spinner{} to Write, that's a usage bug.
func (s Spinner) renderForCapture() Spinner { return s }
```

Update `message.go`: replace `Spinner.render` body:

```go
func (Spinner) render(_ Palette) string {
    panic("cliout.Spinner: pass to cliout.StartSpinner, not cliout.Write")
}
```

### `internal/cliout/prompt.go`

```go
package cliout

import (
    "bufio"
    "errors"
    "fmt"
    "io"
    "strings"

    "github.com/charmbracelet/lipgloss"
)

// ErrAborted is returned from Ask when the user aborts (EOF, Ctrl+C, or
// exhausts retry attempts).
var ErrAborted = errors.New("prompt aborted")

const maxPromptAttempts = 3

// Ask renders a Prompt and returns the validated value. Reads from r (typically
// os.Stdin). Writes label + retries to w.
func Ask(r io.Reader, w io.Writer, p Prompt) (string, error) {
    if rec := activeRecorder(); rec != nil {
        rec.append(p)
        // Capture mode doesn't consume input — return empty string.
        return "", nil
    }

    scanner := bufio.NewScanner(r)
    palette := current()
    labelStyle := lipgloss.NewStyle().Foreground(palette.Muted)
    placeholderStyle := lipgloss.NewStyle().Foreground(palette.Muted)

    for attempt := 0; attempt < maxPromptAttempts; attempt++ {
        // Render "  <Label>: " (muted label, colon, space, cursor stays here).
        line := "  " + labelStyle.Render(p.Label+":") + " "
        _, _ = fmt.Fprint(w, line)
        if p.Placeholder != "" && attempt == 0 {
            // Placeholder shown only on first attempt, in parentheses after label.
            _, _ = fmt.Fprint(w, placeholderStyle.Render("("+p.Placeholder+") "))
        }

        if !scanner.Scan() {
            // EOF or read error.
            if err := scanner.Err(); err != nil {
                return "", err
            }
            return "", ErrAborted
        }
        value := strings.TrimSpace(scanner.Text())

        if p.Validate == nil {
            return value, nil
        }
        if err := p.Validate(value); err == nil {
            return value, nil
        } else {
            // Validation failed — write error step, loop.
            WriteInline(w, Step{Status: StatusFailure, Text: err.Error()})
        }
    }

    Write(w, Step{Status: StatusFailure, Text: fmt.Sprintf("Giving up after %d attempts", maxPromptAttempts)})
    return "", ErrAborted
}

// Update Prompt.render to be consistent with Spinner — panic if rendered.
func (Prompt) render(_ Palette) string {
    panic("cliout.Prompt: pass to cliout.Ask, not cliout.Write")
}
```

### Integration into `cmd/root.go`

#### `RunAuthFlow` — replace the static "Waiting for callback…" with Spinner

Before:
```go
cliout.Write(w, cliout.URL{Label: "Visit this URL to authorize:", Href: authURL})
cliout.WriteInline(w, cliout.Paragraph{Text: "Waiting for callback…", Dim: true})
// ... OpenBrowser ...
// ... select on codeCh ...
cliout.WriteInline(w, cliout.Step{Status: cliout.StatusSuccess, Text: "Browser authentication complete"})
// ... ExchangeCode ...
cliout.WriteInline(w, cliout.Step{Status: cliout.StatusSuccess, Text: "Token exchange successful"})
```

After:
```go
cliout.Write(w, cliout.URL{Label: "Visit this URL to authorize:", Href: authURL})

// Best-effort browser open before spinner so the browser-opening log (if any)
// doesn't land on top of the spinner line.
_ = api.OpenBrowser(authURL)

spin := cliout.StartSpinner(w, "Waiting for authorization")

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

select {
case result := <-codeCh:
    if result.Err != nil {
        spin.Fail(fmt.Sprintf("authorization denied: %s", result.Err))
        return fmt.Errorf("authorization failed: %w", result.Err)
    }
    spin.Done("Authorization received")

    _, err := api.ExchangeCode(context.Background(), http.DefaultClient,
        tokenBaseURL, result.Code, verifier, redirectURI, cfg.ClientID, store)
    if err != nil {
        cliout.WriteInline(w, cliout.Step{Status: cliout.StatusFailure, Text: "Token exchange failed"})
        return fmt.Errorf("exchanging authorization code: %w", err)
    }
    cliout.WriteInline(w, cliout.Step{Status: cliout.StatusSuccess, Text: "Token exchange successful"})
    return nil

case <-ctx.Done():
    spin.Fail("Authorization timed out after 5 minutes")
    return fmt.Errorf("authorization timed out after 5 minutes — please try again")
}
```

`api.OpenBrowser` moves above `StartSpinner` (line ordering) so any browser-open
output doesn't clobber the spinner's `\r`-rewritten line.

#### `runRegister` — replace scanner with `cliout.Ask`

Before:
```go
_, _ = fmt.Fprint(w, "  Client ID: ")
scanner := bufio.NewScanner(r)
scanner.Scan()
if err := scanner.Err(); err != nil {
    return fmt.Errorf("reading client_id: %w", err)
}
clientID := strings.TrimSpace(scanner.Text())
if clientID == "" {
    return fmt.Errorf("client_id cannot be empty")
}
```

After:
```go
clientID, err := cliout.Ask(r, w, cliout.Prompt{
    Label:       "Client ID",
    Placeholder: "32 hex characters from developer.spotify.com/dashboard",
    Validate:    validateClientID,
})
if err != nil {
    return errAlreadyPrinted
}
```

Add the validator in `cmd/root.go`:

```go
// validateClientID enforces the Spotify client-ID shape: 32 lowercase hex chars.
func validateClientID(s string) error {
    s = strings.TrimSpace(s)
    if len(s) != 32 {
        return fmt.Errorf("client ID must be 32 characters (got %d)", len(s))
    }
    if _, err := hex.DecodeString(s); err != nil {
        return fmt.Errorf("client ID must be hexadecimal")
    }
    return nil
}
```

Import `"encoding/hex"` in `cmd/root.go` if not already present.

### Tests

#### `internal/cliout/spinner_test.go`

Test-mode only — animation is disabled, so the spinner just writes start +
resolution lines.

```go
func TestMain(m *testing.M) {
    cliout.SetTestMode(true)
    os.Exit(m.Run())
}

func TestStartSpinner_nonTTY_writesStaticPendingLine(t *testing.T) {
    var buf bytes.Buffer
    h := cliout.StartSpinner(&buf, "Waiting")
    h.Done("Done")
    out := buf.String()
    assert.Contains(t, out, "◌") // Pending glyph on start
    assert.Contains(t, out, "Waiting")
    assert.Contains(t, out, "✓") // Success glyph on resolution
    assert.Contains(t, out, "Done")
}

func TestSpinner_Fail_writesFailureStep(t *testing.T) {
    var buf bytes.Buffer
    h := cliout.StartSpinner(&buf, "Waiting")
    h.Fail("timed out")
    assert.Contains(t, buf.String(), "✗")
    assert.Contains(t, buf.String(), "timed out")
}

func TestSpinner_Stop_silentNoResolutionLine(t *testing.T) {
    var buf bytes.Buffer
    h := cliout.StartSpinner(&buf, "Waiting")
    h.Stop()
    // Stop in test mode doesn't print a resolution line (it would print a
    // blank newline on TTY, but test mode is non-TTY).
    out := buf.String()
    assert.NotContains(t, out, "✓")
    assert.NotContains(t, out, "✗")
}

func TestSpinner_ResolveIdempotent(t *testing.T) {
    var buf bytes.Buffer
    h := cliout.StartSpinner(&buf, "Waiting")
    h.Done("first")
    h.Done("second") // should no-op
    out := buf.String()
    count := strings.Count(out, "✓")
    assert.Equal(t, 1, count, "second Done must be a no-op")
}

func TestSpinner_Capture_recordsSpinnerMessage(t *testing.T) {
    got := cliout.Capture(func(w io.Writer) {
        h := cliout.StartSpinner(w, "Waiting")
        h.Done("Done")
    })
    // Capture records the Spinner on Start. Done/Fail write Step via
    // WriteInline, which also goes to the recorder.
    require.GreaterOrEqual(t, len(got), 1)
    assert.Equal(t, cliout.Spinner{Text: "Waiting"}, got[0])
}
```

#### `internal/cliout/prompt_test.go`

```go
func TestAsk_validFirstAttempt_returnsValue(t *testing.T) {
    r := strings.NewReader("hello\n")
    var buf bytes.Buffer
    got, err := cliout.Ask(r, &buf, cliout.Prompt{Label: "Name"})
    require.NoError(t, err)
    assert.Equal(t, "hello", got)
    assert.Contains(t, buf.String(), "Name:")
}

func TestAsk_trimsWhitespace(t *testing.T) {
    r := strings.NewReader("  hello  \n")
    got, err := cliout.Ask(r, &bytes.Buffer{}, cliout.Prompt{Label: "Name"})
    require.NoError(t, err)
    assert.Equal(t, "hello", got)
}

func TestAsk_validatorFails_thenSucceeds(t *testing.T) {
    r := strings.NewReader("bad\nabc\n")
    var buf bytes.Buffer
    got, err := cliout.Ask(r, &buf, cliout.Prompt{
        Label: "Code",
        Validate: func(s string) error {
            if s != "abc" {
                return errors.New("must be abc")
            }
            return nil
        },
    })
    require.NoError(t, err)
    assert.Equal(t, "abc", got)
    assert.Contains(t, buf.String(), "✗")
    assert.Contains(t, buf.String(), "must be abc")
}

func TestAsk_threeValidationFailures_returnsErrAborted(t *testing.T) {
    r := strings.NewReader("bad\nbad\nbad\n")
    var buf bytes.Buffer
    _, err := cliout.Ask(r, &buf, cliout.Prompt{
        Label: "Code",
        Validate: func(s string) error {
            return errors.New("nope")
        },
    })
    require.ErrorIs(t, err, cliout.ErrAborted)
    assert.Contains(t, buf.String(), "Giving up after 3 attempts")
}

func TestAsk_EOF_returnsErrAborted(t *testing.T) {
    r := strings.NewReader("")
    _, err := cliout.Ask(r, &bytes.Buffer{}, cliout.Prompt{Label: "Name"})
    require.ErrorIs(t, err, cliout.ErrAborted)
}

func TestAsk_placeholderShownFirstAttemptOnly(t *testing.T) {
    r := strings.NewReader("bad\nok\n")
    var buf bytes.Buffer
    _, _ = cliout.Ask(r, &buf, cliout.Prompt{
        Label:       "Code",
        Placeholder: "type ok",
        Validate: func(s string) error {
            if s != "ok" {
                return errors.New("nope")
            }
            return nil
        },
    })
    // Placeholder should appear once (first attempt only).
    count := strings.Count(buf.String(), "type ok")
    assert.Equal(t, 1, count)
}

func TestAsk_capture_recordsPromptMessage(t *testing.T) {
    got := cliout.Capture(func(w io.Writer) {
        _, _ = cliout.Ask(strings.NewReader(""), w, cliout.Prompt{Label: "Name"})
    })
    require.Len(t, got, 1)
    assert.Equal(t, "Name", got[0].(cliout.Prompt).Label)
}
```

#### `cmd/root_test.go` — integration

```go
func TestRunRegister_promptValidatesClientID(t *testing.T) {
    // Feed "short\n" first (validation fail), then a valid 32-char hex ID.
    validID := strings.Repeat("a", 32)
    input := "short\n" + validID + "\n"

    dir := t.TempDir()
    _ = os.Setenv("XDG_CONFIG_HOME", dir)
    t.Cleanup(func() { _ = os.Unsetenv("XDG_CONFIG_HOME") })

    // Intercept stdin equivalent via runRegister's r parameter (it already accepts io.Reader).
    var buf bytes.Buffer
    cmd := RootCommand()
    cmd.SetOut(&buf)
    cmd.SetErr(&buf)

    err := runRegister(cmd, strings.NewReader(input))
    // runRegister will proceed past the prompt but fail on the OAuth flow (no
    // callback server / browser); assert we at least printed validation errors
    // and accepted the valid ID.
    _ = err
    out := buf.String()
    assert.Contains(t, out, "Client ID:")
    assert.Contains(t, out, "✗") // validation-fail step
    assert.Contains(t, out, "client ID must be 32 characters")
}
```

(Full OAuth integration path is hard to test end-to-end without mocking
`StartCallbackServer` + `OpenBrowser`; this smoke test asserts the prompt +
validator wiring.)

## Acceptance Criteria

- [ ] `internal/cliout/spinner.go` implements `StartSpinner`, `SpinnerHandle`,
      `Done`, `Fail`, `Stop`; `render()` panics with clear message
- [ ] Spinner on TTY hides cursor (`\x1b[?25l`) on start, restores
      (`\x1b[?25h`) on any resolve or SIGINT
- [ ] Spinner on non-TTY writes a single `◌ <text>` line on start and a
      `✓`/`✗` step on resolve; no `\r` escapes
- [ ] Spinner under `SetTestMode(true)` behaves like non-TTY (no animation,
      deterministic output)
- [ ] Repeat calls to `Done`/`Fail`/`Stop` on the same handle are no-ops after
      the first resolve
- [ ] `installSIGINTHandler` is installed lazily on first `StartSpinner`, only
      once (via `sync.Once`), and only when not in test mode
- [ ] `internal/cliout/prompt.go` implements `Ask`, `ErrAborted`,
      `maxPromptAttempts = 3`
- [ ] Prompt on EOF returns `ErrAborted`
- [ ] Prompt retries up to 3 times on validation failure; each failure writes
      a `✗` step; after three failures, writes a "Giving up after 3 attempts"
      step and returns `ErrAborted`
- [ ] Placeholder appears only on the first attempt
- [ ] `RunAuthFlow` uses `cliout.StartSpinner` for the callback wait; spinner
      resolves to `✓ Authorization received` or `✗ Authorization timed out…`
- [ ] `runRegister` uses `cliout.Ask` with `validateClientID` (32-char hex)
- [ ] `validateClientID` rejects non-32-char input and non-hex input with
      clear messages
- [ ] All new tests in `internal/cliout/spinner_test.go`,
      `internal/cliout/prompt_test.go`, and `cmd/root_test.go` pass
- [ ] `internal/cliout` coverage ≥ 90%
- [ ] `make ci` passes
- [ ] Visual check: `bin/spotnik auth login` with valid client ID shows animated
      braille spinner on the "Waiting for authorization" line; Ctrl+C restores
      the cursor and exits 130

## Tasks

- [ ] Create `internal/cliout/spinner.go` with `StartSpinner`, `SpinnerHandle`,
      `run`, `Done`, `Fail`, `Stop`, `resolve`, `registerHandle`,
      `unregisterHandle`, `installSIGINTHandler`. Update `Spinner.render` in
      `message.go` to panic with "pass to cliout.StartSpinner, not cliout.Write"
      - test: `go build ./internal/cliout/...` → clean

- [ ] Write `internal/cliout/spinner_test.go` with the five Spinner tests; add
      `TestMain` calling `SetTestMode(true)` (if not already present)
      - test: `go test ./internal/cliout/... -run TestSpinner -v` → PASS;
        `go test ./internal/cliout/... -run TestStartSpinner -v` → PASS

- [ ] Create `internal/cliout/prompt.go` with `ErrAborted`, `maxPromptAttempts`,
      `Ask`. Update `Prompt.render` in `message.go` to panic with "pass to
      cliout.Ask, not cliout.Write"
      - test: `go build ./internal/cliout/...` → clean

- [ ] Write `internal/cliout/prompt_test.go` with the seven Ask tests
      - test: `go test ./internal/cliout/... -run TestAsk -v` → PASS

- [ ] Update `cmd/root.go` `RunAuthFlow`: move `api.OpenBrowser` before the
      spinner; replace the "Waiting for callback…" paragraph with
      `cliout.StartSpinner`; resolve via `spin.Done("Authorization received")`,
      `spin.Fail("Authorization timed out after 5 minutes")`,
      `spin.Fail("authorization denied: ...")`
      - test: `go build ./cmd/...` → clean

- [ ] Add `validateClientID` helper to `cmd/root.go`; import `"encoding/hex"`
      - test: `go build ./cmd/...` → clean

- [ ] Update `cmd/root.go` `runRegister`: replace the `fmt.Fprint("Client ID: ")`
      + `bufio.Scanner` block with `cliout.Ask(r, w, cliout.Prompt{...})`
      - test: `go build ./cmd/...` → clean

- [ ] Write `TestRunRegister_promptValidatesClientID` in `cmd/root_test.go`
      - test: `go test ./cmd/... -run TestRunRegister_promptValidatesClientID -v` → PASS

- [ ] Update `cmd/root_test.go` golden files that previously asserted on
      "Waiting for callback…" — that line is now a Spinner; in test mode it
      renders as `◌ Waiting for authorization` + the resolution step. Refresh
      any affected golden files with `-update`
      - test: `go test ./cmd/... -run TestGolden -update`;
        re-run without `-update` → PASS

- [ ] Manual TTY check: `bin/spotnik auth login` (assuming a valid client ID in
      config) — spinner animates, Ctrl+C restores cursor, exits 130
      - test: visual confirmation

- [ ] Manual pipe check: `bin/spotnik auth login 2>&1 | tee /tmp/out.log` (with
      an unreachable OAuth server, `Ctrl+C` after 2 seconds) — no `\r`/`\x1b`
      escapes in `/tmp/out.log`; static `◌` line present
      - test: `grep -c $'\r' /tmp/out.log` → 0; `grep -c $'\x1b' /tmp/out.log`
        → 0

- [ ] Run `go test -cover ./internal/cliout/...` → coverage ≥ 90.0%

- [ ] `make ci` → PASS
