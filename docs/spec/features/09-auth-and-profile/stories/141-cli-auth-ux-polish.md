---
title: "CLI — Auth UX Polish + Error Handling"
feature: 09-auth-and-profile
status: open
---

## Background

Post-launch testing revealed several CLI auth UX issues:

1. **Logout/forget error on already-logged-out state** — `KeychainTokenStore.Delete()` calls
   `gokeyring.Delete()` on each of the three token keys. When the keys don't exist,
   `gokeyring.Delete()` returns `gokeyring.ErrNotFound` ("secret not found in keyring").
   The method collects all errors and returns them. `LogoutTokens` and `RunForget` both
   propagate this error, so `spotnik auth logout` (when already logged out) fails with:
   ```
   Error: logging out: deleting keychain tokens: [deleting spotnik:access_token: secret not
   found in keyring ...]
   ```
   The correct behaviour: silently skip keys that don't exist — the end state is the same.

2. **`spotnik auth status` — plain text, no visual hierarchy** — `PrintAuthStatus` emits plain
   `fmt.Fprintln` lines with no colour. CLI output should use lipgloss to highlight key values.

3. **`spotnik auth forget` — awkward message** — "Forgotten. Tokens and client_id removed from
   config." The "Forgotten." prefix reads as filler. The message should describe what happened.

4. **`spotnik auth register` — "Starting spotnik…" but not starting** — `RunAuthFlow` prints
   "Authorization successful! Starting spotnik…" then returns nil. `runRegister` exits. The
   user is left at the shell with no TUI. Fix: after `RunAuthFlow` returns nil in `runRegister`,
   call `runApp` to launch the TUI, and update the message printed in `RunAuthFlow` to not
   promise something it cannot deliver.

5. **Subtitle string** — "A terminal Spotify client for developers" appears in `cmd/root.go`
   `Short:` and `Long:` fields. Drop "for developers" to match the agreed shorter tagline.

## Design

### `internal/keychain/keychain.go`

**Fix `Delete()`** — skip `gokeyring.ErrNotFound` errors silently:

```go
import gokeyring "github.com/zalando/go-keyring"

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

This means `logout` on an already-logged-out user silently succeeds. `forget` after `logout`
also succeeds — it skips the absent tokens and only removes the client ID from config.

### `cmd/root.go`

**Update `PrintAuthStatus`** — use lipgloss for coloured output. The function writes to `w io.Writer`
which may not be a terminal (e.g. in tests), so use `lipgloss.NewStyle().Renderer(lipgloss.NewRenderer(w))` 
approach or just use ANSI escapes directly. Since we target real terminals and lipgloss handles
TTY detection, use regular `lipgloss.NewStyle()`:

```
Client ID  ✓ present          (Success colour)   or   ✗ not set      (Error colour)
Status     ✓ authenticated    (Success colour)   or   ✗ not authenticated (Error colour)
Expires    Tue, 21 Apr 2026 13:15:32 CEST        (TextPrimary)
Note       token expiring soon — will refresh    (Warning colour) [only when expiring]
```

Concrete implementation — build output as a list of styled lines written to `w`:

```go
func PrintAuthStatus(store keychain.TokenStore, configPath string, w io.Writer) error {
    cfg, err := loadConfigFromPath(configPath)
    if err != nil {
        cfg = config.Default()
    }

    t := theme.NewBlack() // fixed theme for CLI output

    okStyle  := lipgloss.NewStyle().Foreground(t.Success())
    errStyle := lipgloss.NewStyle().Foreground(t.Error())
    keyStyle := lipgloss.NewStyle().Foreground(t.TextPrimary()).Bold(true)
    valStyle := lipgloss.NewStyle().Foreground(t.TextPrimary())
    warnStyle := lipgloss.NewStyle().Foreground(t.Warning())

    row := func(label, value string) string {
        return keyStyle.Render(fmt.Sprintf("%-12s", label)) + valStyle.Render(value)
    }

    if cfg.ClientID != "" {
        fmt.Fprintln(w, row("Client ID", okStyle.Render("✓")+" present"))
    } else {
        fmt.Fprintln(w, row("Client ID", errStyle.Render("✗")+" not set"))
    }

    access, err := store.Get(keychain.KeyAccessToken)
    if err != nil || access == "" {
        fmt.Fprintln(w, row("Status", errStyle.Render("✗")+" not authenticated"))
        return nil
    }

    fmt.Fprintln(w, row("Status", okStyle.Render("✓")+" authenticated"))

    expiry, err := store.GetExpiry()
    if err == nil {
        fmt.Fprintln(w, row("Expires", expiry.Format(time.RFC1123)))
    }

    expiringSoon, _ := store.IsExpiringSoon()
    if expiringSoon {
        fmt.Fprintln(w, warnStyle.Render("  token expiring soon — will refresh automatically"))
    }

    return nil
}
```

**Update `authForgetCmd` message** — replace "Forgotten. Tokens and client_id removed from
config." with:

```go
_, _ = fmt.Fprintln(c.OutOrStdout(), "✓ Credentials cleared. Tokens removed from keychain; Client ID removed from config.")
```

**Update `authLogoutCmd` message** — keep "Logged out." but add more detail:

```go
_, _ = fmt.Fprintln(c.OutOrStdout(), "✓ Logged out. Session tokens removed from keychain.")
```

**Update `RunAuthFlow`** — change the success print from "Authorization successful! Starting
spotnik..." to just signal success without the false promise:

```go
fmt.Fprintln(os.Stdout, "\n✓ Authorization successful!")
return nil
```

**Update `runRegister`** — after `RunAuthFlow` returns nil, launch the TUI:

```go
store := keychain.NewKeychainTokenStore()
if err := RunAuthFlow(cfg, store, ""); err != nil {
    return err
}
fmt.Fprintln(w, "\nLaunching spotnik...")
return runApp(nil, nil)
```

Note: `runApp` calls `loadConfig()` which re-reads from disk, so the newly saved client ID and
tokens are available. `CheckAuthState` will return `needsRegister=false, needsAuth=false` and
the TUI launches directly to the grid view.

**Update subtitle strings** in `cmd/root.go`:
- `Short: "A terminal Spotify client"` (drop "for developers")
- `Long:  "Spotnik — keyboard-driven Spotify client for the terminal."` (drop "developers who live in")

### Tests — `cmd/root_test.go`

```go
func TestLogoutTokens_alreadyLoggedOut_returnsNil(t *testing.T) {
    // Arrange: empty in-memory store (no tokens set).
    store := keychain.NewInMemoryTokenStore()
    // Act + Assert: should not error — nothing to delete.
    require.NoError(t, LogoutTokens(store))
}

func TestRunForget_afterLogout_clearsClientID(t *testing.T) {
    // Arrange: config with client_id; empty store (already logged out).
    store := keychain.NewInMemoryTokenStore()
    cfg := writeTempConfig(t, "test-client-id")
    // Act: RunForget should succeed even when tokens are absent.
    require.NoError(t, RunForget(store, cfg))
    // Assert: client_id is gone from config.
    loaded, _ := config.Load(cfg)
    assert.Equal(t, "", loaded.ClientID)
}

func TestPrintAuthStatus_colouredOutput_containsCheckMark(t *testing.T) {
    // Arrange: config with client_id, store with access token.
    store := keychain.NewInMemoryTokenStore()
    _ = store.Set(keychain.KeyAccessToken, "tok")
    _ = store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))
    var buf bytes.Buffer
    path := writeTempConfigWithID(t, "id123")
    require.NoError(t, PrintAuthStatus(store, path, &buf))
    assert.Contains(t, buf.String(), "✓")
    assert.Contains(t, buf.String(), "authenticated")
}
```

Note: `InMemoryTokenStore.Delete()` already ignores absent keys (it calls `delete(map, key)` which is a no-op). The `ErrNotFound` fix is only needed for the real `KeychainTokenStore`. The in-memory store's Delete is already safe — existing tests remain green.

## Acceptance Criteria

- [ ] `spotnik auth logout` on an already-logged-out user exits 0 without errors
- [ ] `spotnik auth forget` after a previous `logout` exits 0 and removes client ID
- [ ] `KeychainTokenStore.Delete()` skips `gokeyring.ErrNotFound` silently
- [ ] `spotnik auth status` output uses colour: ✓ in Success, ✗ in Error, label in Bold
- [ ] `spotnik auth status` shows "✓ present" / "✗ not set" for Client ID
- [ ] `spotnik auth status` shows "✓ authenticated" / "✗ not authenticated" for Status
- [ ] `spotnik auth status` shows formatted expiry and warning when expiring soon
- [ ] `spotnik auth forget` prints "✓ Credentials cleared..." (no "Forgotten." prefix)
- [ ] `spotnik auth logout` prints "✓ Logged out. Session tokens removed from keychain."
- [ ] `RunAuthFlow` success message no longer says "Starting spotnik..."
- [ ] `spotnik auth register` launches the TUI after successful OAuth
- [ ] `cmd/root.go` Short and Long descriptions use "A terminal Spotify client" (no "for developers")
- [ ] `TestLogoutTokens_alreadyLoggedOut_returnsNil` passes
- [ ] `TestRunForget_afterLogout_clearsClientID` passes
- [ ] `TestPrintAuthStatus_colouredOutput_containsCheckMark` passes
- [ ] `make ci` passes

## Tasks

- [ ] Fix `KeychainTokenStore.Delete()` to skip `gokeyring.ErrNotFound` in `internal/keychain/keychain.go`
      - test: `go build ./...` → clean
- [ ] Write failing tests in `cmd/root_test.go` for the three new test cases above
      - test: `go test ./cmd/... -run "TestLogoutTokens_alreadyLoggedOut|TestRunForget_afterLogout|TestPrintAuthStatus_coloured" -v` → FAIL (behaviour not yet fixed)
- [ ] Update `PrintAuthStatus` in `cmd/root.go` to use lipgloss coloured output
      - test: `TestPrintAuthStatus_colouredOutput_containsCheckMark` → PASS
- [ ] Update `authForgetCmd` and `authLogoutCmd` confirmation messages in `cmd/root.go`
      - test: manual `bin/spotnik auth forget` / `bin/spotnik auth logout` output review
- [ ] Update `RunAuthFlow` success message; update `runRegister` to call `runApp` after success
      - test: `go build ./...` → clean; manual test: `spotnik auth register` → TUI launches after auth
- [ ] Update `Short`/`Long` subtitle strings in `cmd/root.go`
      - test: `bin/spotnik --help` shows updated strings
- [ ] `make ci` passes
