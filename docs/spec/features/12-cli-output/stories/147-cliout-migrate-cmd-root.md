---
title: "Migrate cmd/root.go auth subcommand output to internal/cliout"
feature: 12-cli-output
status: done
---

## Background

Story 146 shipped `internal/cliout` without changing `cmd/root.go`. This story
replaces every user-facing output composition in `cmd/root.go` with `cliout.*`
calls and removes the local style vars + helpers (`cliGreen/cliRed/cliYellow/
cliDim/cliAccentS/cliDimS/cliErrS/cliWarnS/cliWrap/cliOut/cliLine/cliKV`).

Output matches what ships today except for one deliberate layout change: the
`spotnik auth register` instructions drop the redirect URI onto its own line
(accent-coloured) below the "Add this redirect URI:" step. This trades
byte-identity for a readability win and removes inline style composition from
the Steps body (see `runRegister` section below). Golden files added in this
story lock the new layout; every other auth subcommand is byte-identical.

The feature-level acceptance criterion in `feature.md` has been relaxed
accordingly.

**Depends on:** Story 146 (`internal/cliout` exists).

## Design

### Scope — every user-facing print in `cmd/root.go`

The functions below currently compose styled output via local helpers. All of
them migrate to `cliout.*` in this story.

| Function | Current helpers used | New equivalent |
|---|---|---|
| `Execute()` (error fallback) | `cliWrap.Render(cliErrS.Render("✗")+" "+err.Error())` | `cliout.Write(os.Stderr, cliout.Step{Status: cliout.StatusFailure, Text: err.Error()})` |
| `PrintLogoutSuccess` | `cliOut` | `cliout.Write(w, cliout.Step{Status: cliout.StatusSuccess, Text: "Signed out"})` |
| `PrintForgetSuccess` | `cliOut` | `cliout.Write(w, Step{Success, "Session ended"}, Paragraph{Dim: true, Text: "Tokens and client ID removed"}, Hint{Verb: "Run", Cmd: "spotnik auth register", Tail: "to set up again"})` |
| `PrintAuthStatus` | `cliOut`, `cliKV` | `cliout.Write` with Header + KV + Hint per state |
| `EnsureAuthenticated` (refresh warning) | `cliWrap.Render(cliWarnS.Render("⚠")+...)` | `cliout.Write(os.Stderr, cliout.Step{Status: cliout.StatusWarning, Text: "Session expired — please re-authenticate"})` |
| `RunAuthFlow` (URL block + progress) | `cliOut`, `cliLine` | `cliout.Write(w, URL{Label: "Visit this URL to authorize:", Href: authURL})`, `cliout.WriteInline(w, cliout.Paragraph{Dim: true, Text: "Waiting for callback…"})`, `cliout.WriteInline(w, cliout.Step{Success, "Browser authentication complete"})`, etc. |
| `runRegister` (instructions + success) | `cliOut`, `cliKV`, `cliLine` | Header + Steps + URL (redirect URI) + WriteInline for "Client ID saved" |
| `runRegister` (failure block) | `cliOut`, `cliKV` | Step{Failure} + KV + Hint |
| `runAuthLogin` (no client_id error) | `cliOut`, `cliKV` | Step{Failure} + KV + Hint |
| `runAuthLogin` (OAuth failure) | `cliOut`, `cliKV` | Step{Failure} + KV + Hint |
| `runAuthLogin` + `runRegister` (signed in) | `cliOut`, `cliLine` | Header{Active, "Signed in", ""} + Hint{"", "", "Launching spotnik…"} with arrow handled by Hint |
| `PrintMissingClientIDInstructions` | `cliOut`, `cliKV` | Header + Steps + Hint |

### Exact output mapping — walkthroughs

The shipped output for each command is reproduced below; each walkthrough is
accompanied by the exact `cliout.*` expression that must produce it.

#### `PrintLogoutSuccess` → "Signed out"

Current output (after leading blank, 2-char indent):
```

  ✓ Signed out
```

New:
```go
func PrintLogoutSuccess(w io.Writer) {
    cliout.Write(w, cliout.Step{Status: cliout.StatusSuccess, Text: "Signed out"})
}
```

#### `PrintForgetSuccess`

Current:
```

  ✓ Session ended
  Tokens and client ID removed
  → Run spotnik auth register to set up again
```

New:
```go
func PrintForgetSuccess(w io.Writer) {
    cliout.Write(w,
        cliout.Step{Status: cliout.StatusSuccess, Text: "Session ended"},
        cliout.Paragraph{Text: "Tokens and client ID removed", Dim: true},
        cliout.Hint{Verb: "Run", Cmd: "spotnik auth register", Tail: "to set up again"},
    )
}
```

#### `PrintAuthStatus`

The four states (not registered / not authenticated / authenticated / expiring)
each map to exact `cliout.*` compositions:

```go
func PrintAuthStatus(store keychain.TokenStore, configPath string, w io.Writer) error {
    cfg, err := loadConfigFromPath(configPath)
    if err != nil {
        cfg = config.Default()
    }

    if cfg.ClientID == "" {
        cliout.Write(w,
            cliout.Header{Status: cliout.Inactive, Subject: "Spotnik", State: "not registered"},
            cliout.Hint{Verb: "Run", Cmd: "spotnik auth register", Tail: "to connect your Spotify account"},
        )
        return nil
    }

    access, _ := store.Get(keychain.KeyAccessToken)
    if access == "" {
        cliout.Write(w,
            cliout.Header{Status: cliout.Inactive, Subject: "Spotnik", State: "not authenticated"},
            cliout.KV{Pairs: []cliout.KVPair{cliout.Pair("Client ID", "present")}},
            cliout.Hint{Verb: "Run", Cmd: "spotnik auth login", Tail: "to connect"},
        )
        return nil
    }

    expiringSoon, expiryErr := store.IsExpiringSoon()
    if expiryErr != nil {
        cliout.Write(w,
            cliout.Header{Status: cliout.StatusWarning, Subject: "Spotnik", State: "session state unknown"},
            cliout.Paragraph{Text: "Could not read token state from keychain", Dim: true},
            cliout.Hint{Verb: "Run", Cmd: "spotnik auth login", Tail: "to re-authenticate"},
        )
        return nil
    }

    expiry, _ := store.GetExpiry()
    expiryVal := expiry.Format("Mon, 02 Jan 2006 15:04 UTC")
    var pairs []cliout.KVPair
    pairs = append(pairs, cliout.Pair("Client ID", "present"))

    if expiringSoon {
        pairs = append(pairs, cliout.PairWithCaption("Expires", expiryVal, "auto-refresh pending"))
        cliout.Write(w,
            cliout.Header{Status: cliout.StatusWarning, Subject: "Spotnik", State: "session expiring"},
            cliout.KV{Pairs: pairs},
            cliout.Hint{Verb: "Run", Cmd: "spotnik auth login", Tail: "to re-authenticate if auto-refresh fails"},
        )
        return nil
    }

    pairs = append(pairs, cliout.Pair("Expires", expiryVal))
    cliout.Write(w,
        cliout.Header{Status: cliout.Active, Subject: "Spotnik", State: "authenticated"},
        cliout.KV{Pairs: pairs},
    )
    return nil
}
```

**Note on the caption field:** story 145 rendered the expiring caption as
`"...  ·  auto-refresh pending"` inline with the value (no separate row).
`cliout.KVPair.Caption` reproduces this exactly (see `KV.render` in Story 146).

#### `RunAuthFlow` URL + progress block

Current:
```go
cliOut(w, cliDimS.Render("Visit this URL to authorize:"))
cliLine(w, cliAccentS.Render(authURL))
cliLine(w, cliDimS.Render("Waiting for callback…"))
```

New:
```go
cliout.Write(w, cliout.URL{Label: "Visit this URL to authorize:", Href: authURL})
cliout.WriteInline(w, cliout.Paragraph{Text: "Waiting for callback…", Dim: true})
```

**Spacing note:** `cliOut(...)` adds a leading blank line; `cliLine` does not.
The current sequence produces one blank line before "Visit this URL" and then
URL + waiting lines on consecutive lines. `cliout.Write` produces the leading
blank; `cliout.WriteInline` keeps subsequent lines compact. The two-call
migration preserves the current spacing exactly.

Progress lines inside the `select` branch:
```go
cliout.WriteInline(w, cliout.Step{Status: cliout.StatusSuccess, Text: "Browser authentication complete"})
// ...
cliout.WriteInline(w, cliout.Step{Status: cliout.StatusSuccess, Text: "Token exchange successful"})
```

#### `runRegister` instructions

Current:
```go
cliOut(w,
    cliDimS.Render("◎ Spotnik  ")+"not registered",
    cliKV([][2]string{
        {"1", "Go to developer.spotify.com/dashboard"},
        {"2", "Create or select a Spotify app"},
        {"3", "Add this redirect URI: " + cliAccentS.Render(redirectURI)},
    }),
)
_, _ = fmt.Fprint(w, "  Client ID: ")
```

New:
```go
cliout.Write(w,
    cliout.Header{Status: cliout.Inactive, Subject: "Spotnik", State: "not registered"},
    cliout.Steps{Items: []string{
        "Go to developer.spotify.com/dashboard",
        "Create or select a Spotify app",
        "Add this redirect URI: " + redirectURI,
    }},
)
// Note: the redirect URI itself is no longer accent-coloured in the Steps line
// because Steps.Items entries render as plain text. To preserve the accent on
// the URI, use a single URL message on its own line instead:
cliout.Write(w,
    cliout.Header{Status: cliout.Inactive, Subject: "Spotnik", State: "not registered"},
    cliout.Steps{Items: []string{
        "Go to developer.spotify.com/dashboard",
        "Create or select a Spotify app",
        "Add this redirect URI:",
    }},
    cliout.URL{Href: redirectURI}, // URI on its own line, Accent-coloured
)
_, _ = fmt.Fprint(w, "  Client ID: ")
```

**Pick the second form** (URL on its own line). It preserves the Accent colour
on the URI and removes inline style composition. Produces one extra line break
compared to the shipped output — acceptable improvement, not a regression:
previous layout embedded the URI in the same line as "Add this redirect URI:",
new layout drops the URI to the next line for readability. Update the golden
file to match.

#### Sign-in / launching sequence (shared by `runRegister` + `runAuthLogin`)

Current:
```go
cliOut(w, cliAccentS.Render("◉")+" Signed in")
cliLine(c.OutOrStdout(), cliAccentS.Render("→")+" Launching spotnik…")
```

New — uses Header then Hint on adjacent cliout calls:
```go
cliout.Write(w, cliout.Header{Status: cliout.Active, Subject: "Signed in", State: ""})
cliout.WriteInline(w, cliout.Hint{Tail: "Launching spotnik…"})
```

**Gotcha:** `Hint.Tail` alone renders `→ Launching spotnik…` because `Verb` and
`Cmd` are empty strings (Hint.render drops empty fields).

### Removal checklist

Delete from `cmd/root.go` after all migrations complete:

```go
// Delete these package-level vars and helpers:
var cliGreen, cliRed, cliYellow, cliDim   // lines ~28–33
var cliAccentS, cliDimS, cliErrS, cliWarnS, cliWrap  // lines ~35–44
func cliOut(w io.Writer, lines ...string)            // lines ~54–57
func cliLine(w io.Writer, text string)               // lines ~62–64
func cliKV(pairs [][2]string) string                 // lines ~67–80
```

Remove `"github.com/charmbracelet/lipgloss"` import from `cmd/root.go` if no
remaining uses (verify with `go vet`).

Add `"github.com/initgrep-apps/spotnik/internal/cliout"` import.

### Golden-file tests

Create `cmd/testdata/golden/` with one file per auth command output state.
Content is the ANSI-stripped output (test mode pins profile to Ascii).

```
cmd/testdata/golden/
├── auth_logout.txt
├── auth_forget.txt
├── auth_status_not_registered.txt
├── auth_status_not_authenticated.txt
├── auth_status_authenticated.txt
├── auth_status_expiring.txt
├── auth_status_expiry_unreadable.txt
├── auth_login_no_clientid.txt
├── auth_register_instructions.txt
├── signed_in_launching.txt
└── execute_error_fallback.txt
```

Helper in `cmd/root_test.go`:

```go
// goldenFile reads the expected output for a named case.
// Pass -update to refresh all golden files.
var updateGolden = flag.Bool("update", false, "update golden files")

func assertGolden(t *testing.T, name, got string) {
    t.Helper()
    path := filepath.Join("testdata", "golden", name+".txt")
    if *updateGolden {
        require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
        require.NoError(t, os.WriteFile(path, []byte(got), 0o644))
        return
    }
    want, err := os.ReadFile(path)
    require.NoError(t, err, "missing golden file %s — run with -update", path)
    assert.Equal(t, string(want), got, "golden mismatch for %s", name)
}

func TestMain(m *testing.M) {
    cliout.SetTestMode(true) // pins termenv.Ascii, no animation
    os.Exit(m.Run())
}
```

**TestMain conflict:** before adding the snippet above, `grep -n "TestMain"
cmd/root_test.go` — if one already exists, merge the `SetTestMode(true)` call
into the existing body rather than declare a second. Go only allows one
`TestMain` per package.

New tests:

```go
func TestGolden_AuthLogout(t *testing.T) {
    var buf bytes.Buffer
    PrintLogoutSuccess(&buf)
    assertGolden(t, "auth_logout", buf.String())
}

func TestGolden_AuthForget(t *testing.T) {
    var buf bytes.Buffer
    PrintForgetSuccess(&buf)
    assertGolden(t, "auth_forget", buf.String())
}

// ... one per golden file above
```

Existing `TestPrintAuthStatus_*` / `TestAuth*Cmd_*` tests in `cmd/root_test.go`
keep working because they assert on `strings.Contains`, not exact output. Keep
them — they provide behavioural coverage; golden files add layout coverage.

### Migration order

1. Add cliout import to `cmd/root.go`; keep old helpers temporarily.
2. Migrate function by function — each commit keeps the build green.
3. Add golden-file tests progressively as each function is migrated.
4. Remove old helpers once every call site is migrated.
5. Remove unused imports.

## Acceptance Criteria

- [ ] `cmd/root.go` no longer declares `cliGreen/cliRed/cliYellow/cliDim/cliAccentS/
      cliDimS/cliErrS/cliWarnS/cliWrap/cliOut/cliLine/cliKV`
- [ ] `cmd/root.go` imports `internal/cliout`
- [ ] `cmd/root.go` does not import `lipgloss` (unless a non-CLI need emerges;
      `go vet ./cmd/...` → clean)
- [ ] `cmd/testdata/golden/` contains eleven `.txt` files covering the auth
      command output states listed above (includes
      `auth_status_expiry_unreadable.txt` for the IsExpiringSoon-errored branch)
- [ ] `go test ./cmd/... -v` → PASS (including new `TestGolden_*` tests)
- [ ] `go test ./cmd/... -update` refreshes golden files; re-running without
      `-update` still passes
- [ ] Existing `TestPrintAuthStatus_*` tests continue to pass unchanged
- [ ] Visual check: `bin/spotnik auth status` output matches the shipped look
      (one blank line + 2-char indent + coloured glyphs)
- [ ] `make ci` passes

## Tasks

- [ ] Add `"github.com/initgrep-apps/spotnik/internal/cliout"` import to
      `cmd/root.go`; build passes with old helpers still present
      - test: `go build ./...` → clean

- [ ] Migrate `PrintLogoutSuccess` to call `cliout.Write`
      - test: `go build ./cmd/...` → clean

- [ ] Create `cmd/testdata/golden/` directory; add `TestMain` calling
      `cliout.SetTestMode(true)` and the `assertGolden` helper + `-update` flag
      to `cmd/root_test.go`
      - test: `go build ./cmd/...` → clean

- [ ] Write `TestGolden_AuthLogout`; run with `-update` once to generate
      `auth_logout.txt`; re-run without `-update`
      - test: `go test ./cmd/... -run TestGolden_AuthLogout -update`;
        `go test ./cmd/... -run TestGolden_AuthLogout` → PASS

- [ ] Migrate `PrintForgetSuccess` + golden test `TestGolden_AuthForget`
      - test: `go test ./cmd/... -run TestGolden_AuthForget -update`;
        re-run without `-update` → PASS

- [ ] Migrate `PrintAuthStatus` for all four states; add
      `TestGolden_AuthStatus_*` tests (one per state)
      - test: all four `TestGolden_AuthStatus_*` tests PASS without `-update`;
        existing `TestPrintAuthStatus_*` tests still PASS

- [ ] Migrate `runAuthLogin` — no-client-id branch + OAuth failure branch +
      signed-in success; add `TestGolden_AuthLogin_NoClientID`,
      `TestGolden_SignedInLaunching` (the Hint + Header pair is shared with
      `runRegister`, one golden covers both)
      - test: golden tests PASS

- [ ] Migrate `runRegister` instructions + client-ID saved line + failure branch;
      add `TestGolden_AuthRegister_Instructions`. Accept the
      "redirect URI on its own line" layout change; note in commit message
      - test: `TestGolden_AuthRegister_Instructions` → PASS;
        `bin/spotnik auth register` → URI on its own line, Accent-coloured

- [ ] Migrate `RunAuthFlow` URL block, "Waiting for callback…", and the two step
      lines (`✓ Browser authentication complete`, `✓ Token exchange successful`)
      - test: `go build ./cmd/...` → clean;
        `go test ./cmd/... -v` → all passing

- [ ] Migrate `EnsureAuthenticated` refresh-warning line; migrate `Execute()`
      error fallback; add `TestGolden_ExecuteErrorFallback`
      - test: `TestGolden_ExecuteErrorFallback` → PASS

- [ ] Migrate `PrintMissingClientIDInstructions`; add
      `TestGolden_MissingClientIDInstructions`
      - test: golden PASSES

- [ ] Delete `cliGreen/cliRed/cliYellow/cliDim/cliAccentS/cliDimS/cliErrS/
      cliWarnS/cliWrap` var declarations and `cliOut/cliLine/cliKV` function
      definitions from `cmd/root.go`
      - test: `go build ./...` → clean; `grep -E 'cliGreen|cliOut|cliKV|cliLine' cmd/root.go`
        → no matches

- [ ] Remove `"github.com/charmbracelet/lipgloss"` import from `cmd/root.go` if
      unused; `go vet ./...` → clean
      - test: `go vet ./...` → clean; `goimports -l cmd/root.go` → empty output

- [ ] `make ci` → PASS
