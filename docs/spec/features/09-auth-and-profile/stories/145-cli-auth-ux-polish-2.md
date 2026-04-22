---
title: "CLI — Auth UX Polish Round 2: TUI Launch, Error Dedup, Structured Output"
feature: 09-auth-and-profile
status: open
---

## Background

Post-launch testing against the auth CLI (stories 136 + 141) found three issues:

### 1. `auth login` — no direction after OAuth completes

`runAuthLogin` calls `RunAuthFlow` then returns `nil`. The terminal exits silently.
Story 141 applied TUI launch (`runApp`) after success in `runRegister` but missed
`runAuthLogin`.

**Root cause:** `runAuthLogin` discards the cobra command argument
(`func runAuthLogin(_ *cobra.Command, _ []string) error`), so it can't call
`c.OutOrStdout()` or `runApp(c, ...)`. Fix: capture the cobra command; after
`RunAuthFlow` succeeds print confirmation and call `runApp`.

### 2. Error double-printing + usage spam

`auth login` with no client_id emits:

```
Error: no client_id in config — run: spotnik auth register
Usage:
  spotnik auth login [flags]

Flags:
  -h, --help   help for login

no client_id in config — run: spotnik auth register
```

Error appears **twice**: cobra's built-in printer fires first, then `Execute()` prints
again via `fmt.Fprintln`. Usage on a runtime error (wrong state, not wrong flag) is
also unhelpful.

**Root cause:** `rootCmd` has no `SilenceErrors`; auth subcommands have no
`SilenceUsage`. Fix: `rootCmd.SilenceErrors = true` in `init()` so cobra prints
nothing; `Execute()` owns the single styled print. Add `SilenceUsage: true` to all
auth subcommand definitions.

### 3. CLI output has no visual structure

Story 141 styled `PrintAuthStatus` using theme colors and lipgloss. All other auth
command outputs — register instructions, login/logout/forget confirmations, the OAuth
URL block, and error messages — still use plain `fmt.Fprintln` or are unstyled. The
user expects consistent, structured output following the project CLI design guidelines.

**Root cause:** Story 141 spec only covered `PrintAuthStatus`. The other commands were
not updated, and no shared CLI style layer was established.

---

## CLI Output Design

> All CLI output follows the project CLI Output Design Guidelines. Key rules:
> - No emoji. No box borders. No ASCII art.
> - One accent colour: Spotify green `#1DB954` (via `lipgloss.AdaptiveColor`).
> - Dim/faint for labels, default foreground for values, green for brand glyphs and `→`.
> - All output wrapped in `lipgloss.NewStyle().Padding(1, 2)`.
> - Build multi-line blocks with `lipgloss.JoinVertical`, not embedded `\n`.
> - Glyphs: `◉` active/authenticated, `◎` inactive, `✓` success, `✗` failure,
>   `⚠` warning, `→` next-action hint.
> - No `[OK]`, `[X]`, exclamation marks, or trailing punctuation on status lines.
> - Sentence case. Imperative for hints ("Run spotnik auth login", not "you can run").

### Shared CLI style vars — add to `cmd/root.go`

Define these package-level vars immediately after the `import` block. They are
independent of the user's theme selection.

```go
// CLI output colours — fixed, not theme-dependent.
// AdaptiveColor keeps output readable on light terminals.
var (
    cliGreen  = lipgloss.AdaptiveColor{Dark: "#1DB954", Light: "#1A8C41"}
    cliRed    = lipgloss.AdaptiveColor{Dark: "#FF5555", Light: "#CC0000"}
    cliYellow = lipgloss.AdaptiveColor{Dark: "#F1C40F", Light: "#B8860B"}
    cliDim    = lipgloss.AdaptiveColor{Dark: "#6C7083", Light: "#888888"}
)

var (
    cliAccentS = lipgloss.NewStyle().Foreground(cliGreen).Bold(true)
    cliDimS    = lipgloss.NewStyle().Foreground(cliDim)
    cliErrS    = lipgloss.NewStyle().Foreground(cliRed).Bold(true)
    cliWarnS   = lipgloss.NewStyle().Foreground(cliYellow)
    // cliWrap pads every top-level CLI output block.
    cliWrap = lipgloss.NewStyle().Padding(1, 2)
)
```

Remove the old `theme.Load(theme.DefaultThemeID)` calls in `PrintAuthStatus` and
`Execute()`. Use `cliAccentS`, `cliDimS`, etc. directly.

### Helper: `cliOut(w io.Writer, lines ...string)`

A single helper that renders and writes a block. Every auth command uses this instead
of raw `fmt.Fprintln` loops.

```go
// cliOut writes lines joined vertically, wrapped in the standard CLI padding.
func cliOut(w io.Writer, lines ...string) {
    block := lipgloss.JoinVertical(lipgloss.Left, lines...)
    _, _ = fmt.Fprintln(w, cliWrap.Render(block))
}
```

### Helper: `cliKV(pairs [][2]string) string`

Renders a key-value block with right-padded labels so values align.

```go
// cliKV renders aligned key-value pairs. Labels are dim; values are default.
func cliKV(pairs [][2]string) string {
    maxKey := 0
    for _, p := range pairs {
        if len(p[0]) > maxKey {
            maxKey = len(p[0])
        }
    }
    lines := make([]string, len(pairs))
    for i, p := range pairs {
        pad := strings.Repeat(" ", maxKey-len(p[0]))
        lines[i] = cliDimS.Render(p[0]+pad) + "  " + p[1]
    }
    return strings.Join(lines, "\n")
}
```

---

## Exact Output Designs

### `auth status`

**State: not registered (no client_id)**
```
  ◎ Spotnik  not registered

  → Run spotnik auth register to connect your Spotify account
```

**State: registered, not authenticated**
```
  ◎ Spotnik  not authenticated

    Client ID  present

  → Run spotnik auth login to connect
```

**State: authenticated, token healthy**
```
  ◉ Spotnik  authenticated

    Client ID  present
    Expires    Wed, 23 Apr 2026 10:00 UTC

```

**State: authenticated, token expiring soon**
```
  ⚠ Spotnik  session expiring

    Client ID  present
    Expires    Wed, 23 Apr 2026 10:00 UTC  ·  auto-refresh pending

  → Run spotnik auth login to re-authenticate if auto-refresh fails
```

Implementation — rewrite `PrintAuthStatus` to use `cliOut` + `cliKV`:

```go
func PrintAuthStatus(store keychain.TokenStore, configPath string, w io.Writer) error {
    cfg, err := loadConfigFromPath(configPath)
    if err != nil {
        cfg = config.Default()
    }

    switch {
    case cfg.ClientID == "":
        // Not registered.
        cliOut(w,
            cliDimS.Render("◎ Spotnik  ")+"not registered",
            "",
            cliAccentS.Render("→")+" Run spotnik auth register to connect your Spotify account",
        )
        return nil

    default:
        access, _ := store.Get(keychain.KeyAccessToken)
        if access == "" {
            // Registered, not authenticated.
            cliOut(w,
                cliDimS.Render("◎ Spotnik  ")+"not authenticated",
                "",
                cliKV([][2]string{{"Client ID", "present"}}),
                "",
                cliAccentS.Render("→")+" Run spotnik auth login to connect",
            )
            return nil
        }

        expiringSoon, _ := store.IsExpiringSoon()
        expiry, expiryErr := store.GetExpiry()

        var expiryVal string
        if expiryErr == nil {
            expiryVal = expiry.Format("Mon, 02 Jan 2006 15:04 UTC")
        }
        if expiringSoon {
            expiryVal += "  ·  auto-refresh pending"
        }

        kvPairs := [][2]string{{"Client ID", "present"}}
        if expiryVal != "" {
            kvPairs = append(kvPairs, [2]string{"Expires", expiryVal})
        }

        if expiringSoon {
            cliOut(w,
                cliWarnS.Render("⚠")+" Spotnik  session expiring",
                "",
                cliKV(kvPairs),
                "",
                cliAccentS.Render("→")+" Run spotnik auth login to re-authenticate if auto-refresh fails",
            )
        } else {
            cliOut(w,
                cliAccentS.Render("◉")+" Spotnik  authenticated",
                "",
                cliKV(kvPairs),
            )
        }
        return nil
    }
}
```

---

### `auth login` — error: no client_id

```
  ✗ Authentication failed

    Reason  no client_id configured

  → Run spotnik auth register to set up your Spotify app
```

This error is returned from `runAuthLogin` and rendered by `Execute()`. The error
message returned from `runAuthLogin` should be a structured string that `Execute()`
wraps in the standard error block (see Execute() design below). Alternatively,
`runAuthLogin` can detect the no-client-id case, print the styled block itself, then
return `fmt.Errorf("authentication failed")` (a sentinel that causes exit 1 without
re-printing details).

**Preferred approach:** `runAuthLogin` prints the error block itself (using `cliOut` +
`cliKV`) and returns a plain `errors.New("authentication failed")` sentinel. `Execute()`
prints the sentinel only if it's NOT already been printed — or better: let `Execute()`
always print its styled error block (which duplicates info), which is acceptable since
the sentinel message is short. The simplest safe approach:

```go
func runAuthLogin(c *cobra.Command, _ []string) error {
    cfg, err := loadConfig()
    if err != nil {
        return err
    }
    if cfg.ClientID == "" {
        cliOut(c.OutOrStdout(),
            cliErrS.Render("✗")+" Authentication failed",
            "",
            cliKV([][2]string{{"Reason", "no client_id configured"}}),
            "",
            cliAccentS.Render("→")+" Run spotnik auth register to set up your Spotify app",
        )
        os.Exit(1) // skip Execute()'s error block — message already printed
    }
    // ...
}
```

Using `os.Exit(1)` after printing is the pattern for "already handled" errors in
cobra CLIs. No sentinel leaks through.

---

### `auth login` — OAuth flow output (during and after)

`RunAuthFlow` currently prints with `fmt.Printf`. It must be updated to accept an
`io.Writer` parameter so the caller controls where output goes.

**New signature:**
```go
func RunAuthFlow(cfg *config.Config, store keychain.TokenStore, tokenBaseURL string, w io.Writer) error
```

Update all callers (`runAuthLogin`, `runRegister`, `EnsureAuthenticated` — which passes
`nil` or `os.Stdout` for its existing non-CLI context, though in practice
`EnsureAuthenticated` is not called in the CLI path).

**During the flow, `RunAuthFlow` prints to `w`:**

Step 1 — URL generated + waiting (printed before blocking):
```
  Visit this URL to authorize:
  https://accounts.spotify.com/authorize?…

  Waiting for callback…
```

Rendered as:
```go
urlLine := cliDimS.Render("Visit this URL to authorize:")
cliOut(w,
    urlLine,
    authURL,         // full URL, default foreground, no truncation
    "",
    cliDimS.Render("Waiting for callback…"),
)
```

Step 2 — Callback received (print before code exchange):
```go
_, _ = fmt.Fprintln(w, cliWrap.Render(cliAccentS.Render("✓")+" Browser authentication complete"))
```

Step 3 — Token exchange succeeded (print after ExchangeCode returns nil):
```go
_, _ = fmt.Fprintln(w, cliWrap.Render(cliAccentS.Render("✓")+" Token exchange successful"))
```

Step 4 — Error mid-flow (return from `RunAuthFlow`, print in caller's RunE):
```
  ✗ Authentication failed

    Reason  <err.Error()>

  → Run spotnik auth login to try again
```

The caller (`runAuthLogin`, `runRegister`) prints the error block on non-nil return,
then returns `fmt.Errorf("authentication failed")` sentinel so `Execute()` exits 1
without redundant output. Same `os.Exit(1)` pattern as above.

**After `RunAuthFlow` returns nil — caller prints:**

`runAuthLogin` and `runRegister` both print:
```
  ◉ Signed in

  → Launching spotnik…
```

Then call `runApp(c, []string{})`.

```go
cliOut(c.OutOrStdout(),
    cliAccentS.Render("◉")+" Signed in",
    "",
    cliAccentS.Render("→")+" Launching spotnik…",
)
return runApp(c, []string{})
```

---

### `auth register` — instructions block

No box border. No emoji. Steps are numbered key-value pairs.

```
  ◎ Spotnik  not registered

    1  Go to developer.spotify.com/dashboard
    2  Create or select a Spotify app
    3  Add this redirect URI to your app:
       http://127.0.0.1:8888/callback

  Client ID:
```

`Client ID:` is the stdin prompt — printed with `fmt.Fprint(w, ...)` (no newline) so
cursor stays on the same line.

The redirect URI uses the actual `cfg.CallbackPort`. Config must be loaded before
printing the instructions block. Load order in `runRegister`:

```go
func runRegister(c *cobra.Command, r io.Reader) error {
    w := c.OutOrStdout()
    configPath := config.DefaultConfigPath()
    if err := config.Bootstrap(configPath); err != nil {
        return fmt.Errorf("bootstrapping config: %w", err)
    }
    cfg, err := loadConfigFromPath(configPath)
    if err != nil {
        cfg = config.Default()
    }

    redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", cfg.CallbackPort)

    cliOut(w,
        cliDimS.Render("◎ Spotnik  ")+"not registered",
        "",
        cliKV([][2]string{
            {"1", "Go to developer.spotify.com/dashboard"},
            {"2", "Create or select a Spotify app"},
            {"3", "Add this redirect URI to your app:"},
        }),
        "   "+redirectURI,  // hanging indent under "3"
    )
    _, _ = fmt.Fprint(w, "  Client ID: ")
    // ... scanner reads clientID ...
```

After client ID saved:
```go
_, _ = fmt.Fprintln(w, cliWrap.Render(cliAccentS.Render("✓")+" Client ID saved"))
```

Then `RunAuthFlow(cfg, store, "", w)` runs (see OAuth flow above).

---

### `auth logout`

```
  ✓ Signed out
```

```go
cliOut(c.OutOrStdout(), cliAccentS.Render("✓")+" Signed out")
```

---

### `auth forget`

```
  ✓ Session ended

    Tokens and client ID removed

  → Run spotnik auth register to set up again
```

```go
cliOut(c.OutOrStdout(),
    cliAccentS.Render("✓")+" Session ended",
    "",
    cliDimS.Render("Tokens and client ID removed"),
    "",
    cliAccentS.Render("→")+" Run spotnik auth register to set up again",
)
```

---

### `Execute()` — global error block

When a command returns a non-nil error and has not already printed+exited, `Execute()`
renders the standard error block:

```
  ✗ <err.Error()>
```

For most auth errors the printed block is already handled (via `os.Exit(1)`). For
unexpected errors that bubble up (config parse errors, disk write errors, etc.)
`Execute()` is the fallback:

```go
func Execute(version string) {
    appVersion = version
    rootCmd.Version = version
    if err := rootCmd.Execute(); err != nil {
        // cobra is silenced (SilenceErrors=true); we print once, styled.
        _, _ = fmt.Fprintln(os.Stderr, cliWrap.Render(cliErrS.Render("✗")+" "+err.Error()))
        os.Exit(1)
    }
}
```

Remove `theme.Load` and `th.Error()` from this function — use `cliErrS` directly.

---

## Implementation Notes

1. **`RunAuthFlow` signature change** breaks existing callers and tests.
   - `EnsureAuthenticated` (called from `runApp`): pass `io.Discard` — `runApp` is
     the TUI path, auth output goes through the TUI, not the CLI. Actually
     `EnsureAuthenticated` is not called in production `runApp` (the TUI handles auth
     via the onboarding view). Confirm by grepping: if unused in production paths,
     pass `io.Discard`. If used, pass `os.Stdout`.
   - Tests in `cmd/root_test.go` that call `RunAuthFlow` directly: pass
     `&bytes.Buffer{}` or `io.Discard`.
   - Tests that call `EnsureAuthenticated`: no change needed (it passes through).

2. **Remove `theme.Load` / `theme.DefaultThemeID` from CLI functions**. All CLI
   output now uses the `cliAccentS`, `cliDimS`, etc. package vars. This means the
   `"github.com/initgrep-apps/spotnik/internal/ui/theme"` import may be removable
   from `cmd/root.go` after this story — confirm and clean up if so.

3. **`cliOut`, `cliKV` are unexported helpers** in `cmd/root.go`. No new files.

4. **`SilenceErrors` + `SilenceUsage`**: set in the first `init()` block, before
   `rootCmd.AddCommand(...)` calls. Add to the existing `init()` — do not create a
   second one.

---

## Acceptance Criteria

- [ ] `spotnik auth status` (not registered) renders `◎ Spotnik  not registered`
      with green `→` hint, no box, no emoji
- [ ] `spotnik auth status` (registered, not authenticated) renders `◎ Spotnik  not
      authenticated` with aligned `Client ID  present` kv block
- [ ] `spotnik auth status` (authenticated) renders `◉ Spotnik  authenticated` with
      aligned kv block; token expiring soon renders `⚠` glyph + hint
- [ ] `spotnik auth login` (no client_id) prints the `✗ Authentication failed` block
      with `Reason` kv row and `→` hint; exits 1; no double-print; no usage block
- [ ] `spotnik auth login` (after successful OAuth callback) prints progressive
      `✓ Browser authentication complete`, `✓ Token exchange successful`,
      `◉ Signed in`, `→ Launching spotnik…` then launches TUI
- [ ] `spotnik auth register` shows `◎ Spotnik  not registered` header, numbered
      step kv block with correct callback port, prompts for client_id inline;
      after OAuth: same progressive steps as login then TUI launch
- [ ] `spotnik auth logout` prints `✓ Signed out` only
- [ ] `spotnik auth forget` prints `✓ Session ended` + dim detail + `→` hint
- [ ] No emoji anywhere in CLI output
- [ ] No box borders in CLI output
- [ ] All output has 1-line top/bottom pad and 2-char left indent (Padding(1,2))
- [ ] `make ci` passes

---

## Tasks

- [ ] Add `cliGreen`/`cliRed`/`cliYellow`/`cliDim` colour vars and
      `cliAccentS`/`cliDimS`/`cliErrS`/`cliWarnS`/`cliWrap` style vars to
      `cmd/root.go`; add `cliOut` and `cliKV` helpers
      - test: `go build ./cmd/...` → clean

- [ ] Set `rootCmd.SilenceErrors = true` and `SilenceUsage: true` on all five auth
      subcommand var declarations in `cmd/root.go`
      - test: `bin/spotnik auth login` (no client_id) → one error block, no usage block

- [ ] Update `Execute()`: remove `theme.Load` call; use `cliErrS` + `cliWrap` for the
      error fallback print
      - test: `go build ./cmd/...` → clean; error path prints styled `✗` line

- [ ] Rewrite `PrintAuthStatus` using `cliOut` + `cliKV` per the four-state design
      above; remove `theme.Load` dependency
      - test: `TestPrintAuthStatus_*` tests pass; update test assertions to new output strings

- [ ] Update `authLogoutCmd.RunE`: replace `fmt.Fprintln` with `cliOut(... "✓ Signed out")`
      - test: `bin/spotnik auth logout` → single styled line

- [ ] Update `authForgetCmd.RunE`: replace `fmt.Fprintln` with `cliOut(...)` per
      the three-part forget design above
      - test: `bin/spotnik auth forget` → styled three-part block

- [ ] Add `io.Writer` parameter to `RunAuthFlow`; update its internal prints to use
      `cliOut`/`fmt.Fprintln(w, ...)` for URL block, step confirmations, and waiting
      message; update all callers (`runRegister`, `runAuthLogin`, `EnsureAuthenticated`)
      and all tests that call `RunAuthFlow` directly
      - test: `go test ./cmd/... -v` → PASS; OAuth URL appears in buffer in tests

- [ ] Fix `runAuthLogin`: capture cobra cmd; add no-client-id branch with `cliOut`
      block + `os.Exit(1)`; after `RunAuthFlow` success print `◉ Signed in` + `→
      Launching spotnik…` via `cliOut`; call `runApp(c, []string{})`
      - test: `go build ./...` → clean; manual OAuth → TUI launches

- [ ] Rewrite `runRegister` instructions block: load config for port first; replace
      box with `cliOut` + numbered kv block; change success print to `✓ Client ID
      saved` via `cliOut`; pass `w` to `RunAuthFlow`; print `◉ Signed in` + `→
      Launching spotnik…` after OAuth; call `runApp`
      - test: `bin/spotnik auth register` → new layout visible; no border box

- [ ] Remove `"github.com/initgrep-apps/spotnik/internal/ui/theme"` import from
      `cmd/root.go` if no remaining usage; run `go build ./...` to confirm
      - test: `go build ./...` → clean; `go vet ./...` → clean

- [ ] `make ci` → PASS
