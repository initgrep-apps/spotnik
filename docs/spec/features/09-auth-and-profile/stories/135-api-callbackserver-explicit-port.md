---
title: "API — StartCallbackServer Accepts Explicit Port"
feature: 09-auth-and-profile
status: done
---

## Background

The onboarding redesign starts the OAuth callback server **before** the TUI renders, so the
redirect URI `http://127.0.0.1:{port}/callback` is known at registration screen render time.
The current `StartCallbackServer()` accepts no arguments and binds to a random OS-assigned port,
making it impossible to show the exact URI in the onboarding UI.

The new signature takes an explicit `port int`. Passing `0` delegates to the OS (random port),
which is the correct behaviour for tests and the old CLI flow.

**Depends on:** Story 134 (`Config.CallbackPort` field must exist so `cmd/root.go` can pass it).

## Design

### `internal/api/auth.go`

**New signature:**

```go
// StartCallbackServer starts a local HTTP server on the given port to receive
// the OAuth callback from Spotify. Pass port=0 to let the OS assign a random
// port (useful in tests). Returns the server, a result channel, and any error.
func StartCallbackServer(port int) (*callbackServer, <-chan CallbackResult, error) {
    addr := fmt.Sprintf("127.0.0.1:%d", port)
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return nil, nil, fmt.Errorf("starting callback server on %s: %w", addr, err)
    }
    // ... rest of existing body unchanged, adapted to use ln ...
}
```

The `callbackServer.URL` field is set to `"http://" + ln.Addr().String()` so the actual bound
address (including the resolved port when 0 was passed) is always accessible.

**All existing callers** must be updated to pass `0` as a temporary placeholder so the codebase
compiles. Definitive wiring (`cfg.CallbackPort`) happens in Story 136.

Existing callers at time of writing:
- `cmd/root.go` `RunAuthFlow`: `api.StartCallbackServer(0)`
- `internal/app/auth.go` `prepareAuthCmd` (or equivalent): `api.StartCallbackServer(0)`

### Tests — `internal/api/auth_test.go`

Write **failing** tests first (TDD):

```go
func TestStartCallbackServer_fixedPort(t *testing.T) {
    // Bind to a known port; assert srv.URL contains that port.
    srv, ch, err := StartCallbackServer(18765)
    require.NoError(t, err)
    defer srv.Close()
    assert.Contains(t, srv.URL, ":18765")
    assert.NotNil(t, ch)
}

func TestStartCallbackServer_portZero_randomPort(t *testing.T) {
    // Port 0 → OS-assigned; URL must be non-empty.
    srv, ch, err := StartCallbackServer(0)
    require.NoError(t, err)
    defer srv.Close()
    assert.NotEmpty(t, srv.URL)
    assert.NotNil(t, ch)
}

func TestStartCallbackServer_portBusy_returnsError(t *testing.T) {
    ln, err := net.Listen("tcp", "127.0.0.1:18766")
    require.NoError(t, err)
    defer ln.Close()

    _, _, err = StartCallbackServer(18766)
    assert.Error(t, err)
}
```

The test ports (18765, 18766) are in the ephemeral range and unlikely to conflict. If flakiness
is observed in CI, use `t.Setenv` or a helper that finds a free port.

## Acceptance Criteria

- [ ] `StartCallbackServer(port int)` signature compiles and all existing callers updated
- [ ] `srv.URL` reflects the actual bound address (resolves port when 0 was passed)
- [ ] `StartCallbackServer(18765)`: `srv.URL` contains `:18765`
- [ ] `StartCallbackServer(0)`: binds to a random port, URL non-empty
- [ ] `StartCallbackServer` on a busy port returns a non-nil error
- [ ] `go build ./...` passes after updating all callers
- [ ] All 3 new tests pass; `make ci` passes

## Tasks

- [ ] Write failing tests in `internal/api/auth_test.go` for the 3 `TestStartCallbackServer_*`
      cases
      - test: `go test ./internal/api/... -run "TestStartCallbackServer" -v`
        → compile error or FAIL (wrong number of args)
- [ ] Update `StartCallbackServer` signature and body in `internal/api/auth.go` to accept
      `port int`; bind via `net.Listen("tcp", "127.0.0.1:{port}")`
      - test: `TestStartCallbackServer_*` → PASS
- [ ] Update all call sites (`cmd/root.go`, `internal/app/auth.go`) to pass `0` so code
      compiles
      - test: `go build ./...` → clean
- [ ] `make ci` passes
