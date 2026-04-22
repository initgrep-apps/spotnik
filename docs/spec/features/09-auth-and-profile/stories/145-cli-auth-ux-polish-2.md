---
title: "CLI — Auth UX Polish Round 2: TUI Launch, Error Styling, Emoji Output"
feature: 09-auth-and-profile
status: open
---

## Background

Post-launch testing against the auth CLI (stories 136 + 141) found three issues:

### 1. `auth login` — no direction after OAuth completes

`runAuthLogin` calls `RunAuthFlow` then returns `nil`. The terminal exits with no
"what now" guidance. Story 141 applied TUI launch (`runApp`) after success in
`runRegister` but missed `runAuthLogin`. The user is left staring at a shell prompt
with no idea the app is ready.

**Root cause:** `runAuthLogin` is declared as
`func runAuthLogin(_ *cobra.Command, _ []string) error` (discards the cobra command
argument), so it can't call `c.OutOrStdout()` or `runApp(c, ...)`. The fix is to
capture the cobra command and call `runApp` after `RunAuthFlow` succeeds, exactly as
`runRegister` does.

### 2. Error double-printing + usage spam

When `auth login` (or any auth subcommand) returns an error, the output is:

```
Error: no client_id in config — run: spotnik auth register
Usage:
  spotnik auth login [flags]

Flags:
  -h, --help   help for login

no client_id in config — run: spotnik auth register
```

The message appears **twice**: once from cobra's default error+usage printer, once
from `Execute()` (`fmt.Fprintln(os.Stderr, err)`). Printing usage on a runtime error
(wrong state, not wrong flag) is also confusing.

**Root cause:** `rootCmd` has no `SilenceErrors` and auth subcommands have no
`SilenceUsage`. Fix: set `rootCmd.SilenceErrors = true` in `init()` (cobra prints
nothing; `Execute()` handles the single styled print) and set `SilenceUsage: true` on
`authLoginCmd` (and all auth subcommands) to suppress usage on runtime errors.

### 3. No emoji / lipgloss styling on auth command output

Story 141 styled `PrintAuthStatus`. All other auth command outputs (`register` box,
`login` success, `logout` confirmation, `forget` confirmation, error messages) still
use plain `fmt.Fprintln`. The user expects styled, emoji-enriched output consistent
with the rest of the app.

**Root cause:** Story 141 spec only covered `PrintAuthStatus`. `runRegister`,
`runAuthLogin`, `authLogoutCmd.RunE`, and `authForgetCmd.RunE` were not updated.

---

## Design

All changes are confined to `cmd/root.go`.

### 2a. Suppress double-print: `init()` in `cmd/root.go`

```go
func init() {
    // Silence cobra's built-in error printer — Execute() handles styled output.
    rootCmd.SilenceErrors = true
    // ... existing init() registrations
}
```

### 2b. Styled error in `Execute()`

```go
func Execute(version string) {
    appVersion = version
    rootCmd.Version = version
    if err := rootCmd.Execute(); err != nil {
        th := theme.Load(theme.DefaultThemeID)
        errStyle := lipgloss.NewStyle().Foreground(th.Error()).Bold(true)
        fmt.Fprintln(os.Stderr, errStyle.Render("✗  "+err.Error()))
        os.Exit(1)
    }
}
```

Import `"github.com/initgrep-apps/spotnik/internal/ui/theme"` if not already present
in `cmd/root.go` (it is, from story 141). Check that `theme.Load` accepts
`theme.DefaultThemeID` — currently the call in `PrintAuthStatus` is `theme.Load(theme.DefaultThemeID)`.

### 2c. SilenceUsage on auth subcommands

Add `SilenceUsage: true` to `authLoginCmd`, `authLogoutCmd`, `authForgetCmd`,
`authStatusCmd`, and `authRegisterCmd`. Usage on a runtime error (missing config,
wrong state) is not helpful:

```go
var authLoginCmd = &cobra.Command{
    Use:          "login",
    Short:        "Re-authenticate with Spotify (clears existing tokens)",
    Long:         "Force a fresh Spotify authentication...",
    SilenceUsage: true,
    RunE:         runAuthLogin,
}
// Same SilenceUsage: true for authLogoutCmd, authForgetCmd, authStatusCmd, authRegisterCmd.
```

### 1. Fix `runAuthLogin` — capture cobra cmd, launch TUI after success

Change the signature so the cobra command is available:

```go
var authLoginCmd = &cobra.Command{
    ...
    RunE: func(c *cobra.Command, args []string) error {
        return runAuthLogin(c, args)
    },
}

func runAuthLogin(c *cobra.Command, _ []string) error {
    cfg, err := loadConfig()
    if err != nil {
        return err
    }
    if cfg.ClientID == "" {
        return fmt.Errorf("no client_id in config — run: spotnik auth register")
    }

    store := keychain.NewKeychainTokenStore()
    _ = store.Delete() // force fresh login

    if err := RunAuthFlow(cfg, store, ""); err != nil {
        return err
    }
    // Authorization succeeded — launch TUI immediately, same as runRegister.
    _, _ = fmt.Fprintln(c.OutOrStdout(), okStyle(c).Render("✓  Authorization complete. Launching spotnik..."))
    return runApp(c, []string{})
}
```

Where `okStyle(c)` is a helper or inlined lipgloss style using `th.Success()`:

```go
th := theme.Load(theme.DefaultThemeID)
okS := lipgloss.NewStyle().Foreground(th.Success()).Bold(true)
_, _ = fmt.Fprintln(c.OutOrStdout(), okS.Render("✓  Authorization complete. Launching spotnik..."))
```

### 3a. Styled `runRegister` box

Replace hardcoded `fmt.Fprintln` lines with a lipgloss-rendered panel:

```go
func runRegister(c *cobra.Command, r io.Reader) error {
    w := c.OutOrStdout()
    th := theme.Load(theme.DefaultThemeID)

    borderStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(th.Primary()).
        Padding(1, 3).
        Width(55)

    labelStyle  := lipgloss.NewStyle().Foreground(th.TextPrimary()).Bold(true)
    mutedStyle  := lipgloss.NewStyle().Foreground(th.TextMuted())
    linkStyle   := lipgloss.NewStyle().Foreground(th.Secondary())

    body := lipgloss.JoinVertical(lipgloss.Left,
        labelStyle.Render("🎵  Spotnik — first-time setup"),
        "",
        "  1. "+mutedStyle.Render("Go to")+" "+linkStyle.Render("https://developer.spotify.com/dashboard"),
        "  2. "+mutedStyle.Render("Create (or pick) a Spotify app."),
        "  3. "+mutedStyle.Render("In Redirect URIs, add:"),
        "     "+linkStyle.Render(fmt.Sprintf("http://127.0.0.1:%d/callback", cfg.CallbackPort)),
        "     "+mutedStyle.Render("(change the port if you set callback_port)"),
    )

    _, _ = fmt.Fprintln(w, borderStyle.Render(body))
    _, _ = fmt.Fprintln(w, "")
    // ... rest of prompt
```

Note: `cfg` must be loaded before printing the box so `cfg.CallbackPort` is available.
Load config at the top of `runRegister`:

```go
func runRegister(c *cobra.Command, r io.Reader) error {
    w := c.OutOrStdout()
    th := theme.Load(theme.DefaultThemeID)

    configPath := config.DefaultConfigPath()
    if err := config.Bootstrap(configPath); err != nil {
        return fmt.Errorf("bootstrapping config: %w", err)
    }
    cfg, _ := loadConfigFromPath(configPath) // non-fatal; use default port if load fails
    if cfg == nil {
        cfg = config.Default()
    }

    // ... render box using cfg.CallbackPort ...
```

### 3b. Styled `auth logout` confirmation

```go
var authLogoutCmd = &cobra.Command{
    ...
    RunE: func(c *cobra.Command, args []string) error {
        store := keychain.NewKeychainTokenStore()
        if err := LogoutTokens(store); err != nil {
            return err
        }
        th := theme.Load(theme.DefaultThemeID)
        okS := lipgloss.NewStyle().Foreground(th.Success()).Bold(true)
        _, _ = fmt.Fprintln(c.OutOrStdout(), okS.Render("✓  Logged out."))
        return nil
    },
}
```

### 3c. Styled `auth forget` confirmation

```go
var authForgetCmd = &cobra.Command{
    ...
    RunE: func(c *cobra.Command, args []string) error {
        store := keychain.NewKeychainTokenStore()
        if err := RunForget(store, config.DefaultConfigPath()); err != nil {
            return err
        }
        th := theme.Load(theme.DefaultThemeID)
        okS   := lipgloss.NewStyle().Foreground(th.Success()).Bold(true)
        mutedS := lipgloss.NewStyle().Foreground(th.TextMuted())
        _, _ = fmt.Fprintln(c.OutOrStdout(), okS.Render("✓  Session ended. Tokens and client ID removed."))
        _, _ = fmt.Fprintln(c.OutOrStdout(), mutedS.Render("   Run 'spotnik auth register' to set up again."))
        return nil
    },
}
```

---

## Acceptance Criteria

- [ ] `spotnik auth login` (with valid client_id, after OAuth callback) prints
      `✓  Authorization complete. Launching spotnik...` in green and then launches
      the TUI — same behaviour as `auth register`
- [ ] `spotnik auth login` (no client_id) prints styled red error `✗  no client_id
      in config — run: spotnik auth register` **once** (not twice); usage block is
      suppressed
- [ ] `spotnik auth register` renders a lipgloss-bordered box with emoji header and
      coloured links; uses the configured callback port in the redirect URI shown
- [ ] `spotnik auth logout` prints `✓  Logged out.` in green
- [ ] `spotnik auth forget` prints `✓  Session ended. Tokens and client ID removed.`
      in green, then muted `Run 'spotnik auth register' to set up again.`
- [ ] All other error paths (e.g. OAuth exchange fails) print styled red error once
- [ ] `make ci` passes

---

## Tasks

- [ ] In `init()` (or the `authLoginCmd`/`authLogoutCmd`/`authForgetCmd`/
      `authRegisterCmd`/`authStatusCmd` var declarations), set `SilenceUsage: true`
      on all auth subcommands
      - test: `bin/spotnik auth login` (no client_id) → no usage block, one error line
- [ ] Set `rootCmd.SilenceErrors = true` in the root `init()` function in `cmd/root.go`
      - test: same as above — error now appears once
- [ ] Style the error line in `Execute()` using lipgloss + `th.Error()` + `✗` prefix
      - test: `bin/spotnik auth login` (no client_id) → red styled error once
- [ ] Fix `runAuthLogin`: capture cobra cmd arg; after `RunAuthFlow` success print
      styled confirmation and call `runApp(c, []string{})`
      - test: manual OAuth flow → TUI launches after callback
- [ ] Style `runRegister` instructions box with lipgloss border + emoji + color links;
      load config before rendering so callback port is accurate
      - test: `bin/spotnik auth register` → coloured bordered box visible
- [ ] Style `authLogoutCmd.RunE` success message: `✓  Logged out.` in `th.Success()`
      - test: `bin/spotnik auth logout` → green checkmark line
- [ ] Style `authForgetCmd.RunE` success message: `✓  Session ended...` + muted hint
      - test: `bin/spotnik auth forget` → green + muted lines
- [ ] `make ci` → PASS
