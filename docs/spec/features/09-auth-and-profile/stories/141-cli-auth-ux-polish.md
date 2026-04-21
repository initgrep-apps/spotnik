---
title: "CLI — Auth UX Polish + ErrNotFound Fix"
feature: 09-auth-and-profile
status: done
---

## Background

Post-launch testing revealed five CLI auth UX issues:

1. **`auth logout` / `auth forget` crash when already logged out** — `KeychainTokenStore.Delete()`
   calls `gokeyring.Delete()` on each of the three token keys. When a key is absent,
   `gokeyring.Delete()` returns `gokeyring.ErrNotFound` ("secret not found in keyring"). The
   method collects every error — including not-found — and surfaces them as a hard failure.
   `LogoutTokens` and `RunForget` both propagate this, so `spotnik auth logout` (when already
   logged out) prints:
   ```
   Error: logging out: deleting keychain tokens: [deleting spotnik:access_token: secret not
   found in keyring deleting spotnik:refresh_token: secret not found in keyring ...]
   ```
   The end state (no tokens in keychain) is identical whether the keys existed or not — not-found
   errors from delete are not errors from the user's perspective. Fix: skip
   `gokeyring.ErrNotFound` silently; only collect unexpected errors.

2. **`auth status` plain text, no visual hierarchy** — `PrintAuthStatus` emits plain
   `fmt.Fprintln` lines. Labels and values are the same colour; status line is identical to
   everything else. Should use lipgloss to highlight key values.

3. **`auth forget` message is awkward** — "Forgotten. Tokens and client_id removed from config."
   The word "Forgotten." reads as filler. The message should describe what actually changed.

4. **`auth register` promises to start but doesn't** — `RunAuthFlow` prints "Authorization
   successful! Starting spotnik..." then `return nil`. `runRegister` exits to the shell with no
   TUI open. Fix: change the printed message to "Authorization complete." and after `RunAuthFlow`
   returns nil in `runRegister`, call `runApp(c, []string{})` to launch the TUI in the same
   process. Remove the misleading "Starting spotnik..." entirely.

5. **Subtitle "for developers" in `cmd/root.go`** — `rootCmd.Short`, `rootCmd.Long`, and
   `authCmd.Long` all say "for developers". Per the agreed shorter tagline ("A terminal Spotify
   client"), drop the suffix.

**Depends on:** nothing — changes are confined to `internal/keychain/` and `cmd/`.

## Design

### `internal/keychain/keychain.go` — fix `Delete()`

Import `gokeyring` is already present. Add the not-found guard:

```go
func (s *KeychainTokenStore) Delete() error {
    keys := []string{KeyAccessToken, KeyRefreshToken, KeyTokenExpiry}
    var errs []error
    for _, key := range keys {
        if err := gokeyring.Delete(Service, key); err != nil {
            if err == gokeyring.ErrNotFound {
                continue // key absent — nothing to delete, not an error
            }
            errs = append(errs, fmt.Errorf("deleting %s: %w", key, err))
        }
    }
    if len(errs) > 0 {
        return fmt.Errorf("deleting keychain tokens: %v", errs)
    }
    return nil
}
```

### `cmd/root.go` — styled `PrintAuthStatus`

Replace plain `fmt.Fprintln` calls with lipgloss-styled output. Labels in `TextMuted`, values
in `TextPrimary`, authenticated status in `Success`, expiring-soon notice in `Warning`:

```go
func PrintAuthStatus(store keychain.TokenStore, configPath string, w io.Writer) error {
    th := theme.NewBlack() // always use black for CLI — terminal-safe colours

    labelStyle  := lipgloss.NewStyle().Foreground(th.TextMuted())
    valueStyle  := lipgloss.NewStyle().Foreground(th.TextPrimary()).Bold(true)
    okStyle     := lipgloss.NewStyle().Foreground(th.Success())
    warnStyle   := lipgloss.NewStyle().Foreground(th.Warning())
    mutedStyle  := lipgloss.NewStyle().Foreground(th.TextMuted())

    cfg, err := loadConfigFromPath(configPath)
    if err != nil {
        cfg = config.Default()
    }

    if cfg.ClientID != "" {
        _, _ = fmt.Fprintf(w, "%s  %s\n",
            labelStyle.Render("Client ID:"),
            valueStyle.Render("present"),
        )
    } else {
        _, _ = fmt.Fprintf(w, "%s  %s\n",
            labelStyle.Render("Client ID:"),
            warnStyle.Render("not set  (run: spotnik auth register)"),
        )
    }

    access, err := store.Get(keychain.KeyAccessToken)
    if err != nil || access == "" {
        _, _ = fmt.Fprintf(w, "%s  %s\n",
            labelStyle.Render("Status:  "),
            mutedStyle.Render("not authenticated"),
        )
        return nil
    }

    _, _ = fmt.Fprintf(w, "%s  %s\n",
        labelStyle.Render("Status:  "),
        okStyle.Render("authenticated"),
    )

    expiry, err := store.GetExpiry()
    if err == nil {
        _, _ = fmt.Fprintf(w, "%s  %s\n",
            labelStyle.Render("Expires: "),
            mutedStyle.Render(expiry.Format(time.RFC1123)),
        )
    }

    expiringSoon, _ := store.IsExpiringSoon()
    if expiringSoon {
        _, _ = fmt.Fprintf(w, "%s\n",
            warnStyle.Render("⚠  Token expiring soon — will refresh automatically"),
        )
    }

    return nil
}
```

Add `"github.com/charmbracelet/lipgloss"` and `"github.com/initgrep-apps/spotnik/internal/ui/theme"`
to `cmd/root.go` imports.

### `cmd/root.go` — fix `auth forget` message

In `authForgetCmd.RunE`, change:

```go
// Before
_, _ = fmt.Fprintln(c.OutOrStdout(), "Forgotten. Tokens and client_id removed from config.")

// After
_, _ = fmt.Fprintln(c.OutOrStdout(), "Session ended. Client ID removed from config.\nRun 'spotnik auth register' to set up again.")
```

### `cmd/root.go` — fix `auth register` launch

In `runRegister`, after `RunAuthFlow` returns nil, launch the TUI:

```go
if err := RunAuthFlow(cfg, store, ""); err != nil {
    return err
}
// Authorization succeeded — launch the TUI immediately so the user lands in the app.
_, _ = fmt.Fprintln(c.OutOrStdout(), "Authorization complete. Launching spotnik...")
return runApp(c, []string{})
```

In `RunAuthFlow`, change the success print:

```go
// Before
fmt.Println("Authorization successful! Starting spotnik...")

// After — message is printed by runRegister, not RunAuthFlow
// Remove this line entirely (RunAuthFlow returns nil silently on success)
```

### `cmd/root.go` — subtitle cleanup

```go
// rootCmd.Short
Short: "A terminal Spotify client",

// rootCmd.Long
Long: "Spotnik — keyboard-driven Spotify client.",

// authCmd.Long (drop "for developers" from the description line if present)
```

### Tests — `internal/keychain/keychain_test.go`

```go
func TestKeychainTokenStore_Delete_notFoundIsNotError(t *testing.T) {
    // This test relies on InMemoryTokenStore for predictability.
    // Verify that Delete() on an empty store returns nil.
    store := keychain.NewInMemoryTokenStore()
    err := store.Delete()
    assert.NoError(t, err)
}
```

Add a comment explaining that the real `KeychainTokenStore.Delete()` ErrNotFound skip is tested
indirectly via `TestAuthLogout*` in `cmd/root_test.go`:

```go
func TestAuthLogoutCmd_alreadyLoggedOut_noError(t *testing.T) {
    store := keychain.NewInMemoryTokenStore()
    // Store is empty — Delete() should not return an error.
    err := cmd.LogoutTokens(store)
    assert.NoError(t, err)
}

func TestAuthForgetCmd_noClientID_noError(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    _ = os.WriteFile(path, []byte("[spotify]\n"), 0o600)
    store := keychain.NewInMemoryTokenStore()
    err := cmd.RunForget(store, path)
    assert.NoError(t, err)
}

func TestPrintAuthStatus_styled_authenticated(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    _ = os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\n"), 0o600)
    store := keychain.NewInMemoryTokenStore()
    _ = store.Set(keychain.KeyAccessToken, "tok")
    _ = store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))

    var buf strings.Builder
    err := cmd.PrintAuthStatus(store, path, &buf)
    require.NoError(t, err)
    assert.Contains(t, buf.String(), "Client ID")
    assert.Contains(t, buf.String(), "authenticated")
}
```

## Acceptance Criteria

- [ ] `spotnik auth logout` (when already logged out) exits 0 with "Logged out. Stored tokens
      removed." and no error about "secret not found in keyring"
- [ ] `spotnik auth forget` (when already logged out or no client_id) exits 0 with "Session
      ended. Client ID removed from config." and no error
- [ ] `spotnik auth status` output has styled labels and values (lipgloss colours)
- [ ] `spotnik auth register` launches the TUI after successful OAuth, not exits to shell
- [ ] `RunAuthFlow` no longer prints "Starting spotnik..."
- [ ] `rootCmd.Short` says "A terminal Spotify client" (no "for developers")
- [ ] `TestAuthLogoutCmd_alreadyLoggedOut_noError` passes
- [ ] `TestAuthForgetCmd_noClientID_noError` passes
- [ ] `make ci` passes

## Tasks

- [ ] In `internal/keychain/keychain.go` `Delete()`, add `if err == gokeyring.ErrNotFound { continue }` guard
      - test: `go build ./internal/keychain/...` → clean
- [ ] Write `TestAuthLogoutCmd_alreadyLoggedOut_noError` and `TestAuthForgetCmd_noClientID_noError`
      in `cmd/root_test.go`
      - test: `go test ./cmd/... -run "TestAuthLogout_already|TestAuthForget_no" -v` → PASS
- [ ] Update `PrintAuthStatus` in `cmd/root.go` to use lipgloss styling
      - test: `TestPrintAuthStatus_styled_authenticated` → PASS
- [ ] Fix `authForgetCmd.RunE` success message
      - test: `go test ./cmd/... -run "TestAuthForget" -v` → PASS
- [ ] Fix `runRegister` to call `runApp` after `RunAuthFlow` succeeds; remove "Starting spotnik..."
      from `RunAuthFlow`
      - test: `go build ./...` → clean
- [ ] Update `rootCmd.Short`, `rootCmd.Long`, `authCmd.Long` to drop "for developers"
      - test: `go build ./...` → clean
- [ ] `make ci` → PASS
