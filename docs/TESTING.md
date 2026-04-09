# Spotnik Testing Guide

This document describes the test architecture, patterns, and how to run each category
of tests.

---

## Make Targets

| Target | When to use |
|--------|-------------|
| `make test` | Quick feedback loop during development — runs unit tests only |
| `make test-integration` | Before pushing — runs integration tests (not included in `make ci`) |
| `make test-coverage` | Before PR — ensures coverage stays above 80% |
| `make ci` | Before pushing — full gate: `fmt-check → tidy-check → lint → test-coverage → build` (unit tests only; does **not** run integration tests) |

> **Note:** `make ci` does not run integration tests. Run `make test-integration` separately
> before submitting a PR that touches API client code or multi-component flows.

---

## Coverage Requirement

**80% minimum**, enforced by `make test-coverage`. CI fails below this threshold.
Every function in `api/`, `state/`, and `config/` must be covered.

---

## Test Patterns

### Table-Driven Tests

All new tests must use the table-driven style:

```go
func TestFormatRelativeTime(t *testing.T) {
    tests := []struct {
        name  string
        input string
        want  string
    }{
        {name: "seconds ago", input: "2024-03-01T22:15:00Z", want: "just now"},
        {name: "minutes ago", input: "2024-03-01T22:00:00Z", want: "15m ago"},
        {name: "hours ago",   input: "2024-03-01T20:00:00Z", want: "2h ago"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := FormatRelativeTime(tt.input, referenceTime)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### HTTP Mock Pattern (API Clients)

Use `httptest.NewServer` for all API client tests — no external mock libraries:

```go
func TestGetDevices_Success(t *testing.T) {
    fixture := testhelpers.LoadFixture(t, "devices_response.json")

    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/v1/me/player/devices", r.URL.Path)
        assert.Equal(t, http.MethodGet, r.Method)
        assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(fixture)
    }))
    defer srv.Close()

    client := NewDevicesClient(srv.URL, "test-token")
    devices, err := client.Devices(context.Background())

    require.NoError(t, err)
    require.Len(t, devices, 3)
    assert.Equal(t, "MacBook Pro Speakers", devices[0].Name)
}
```

### LoadFixture Pattern

Use `testhelpers.LoadFixture` to read JSON fixtures from `testdata/fixtures/`. Never use
inline `os.ReadFile` calls — the helper resolves paths relative to its own source file,
making it work from any package depth:

```go
import "github.com/initgrep-apps/spotnik/internal/testhelpers"

// In your test:
fixture := testhelpers.LoadFixture(t, "playback_state.json")
```

Fixture files live in `testdata/fixtures/<name>.json` at the project root.

### Pane Update Tests

Test Bubble Tea model `Update()` handlers by sending messages and asserting on the
returned model and command:

```go
func TestQueuePane_Update_KeyJ(t *testing.T) {
    pane := NewQueuePane(store, theme)
    pane.SetSize(80, 24)

    updated, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
    model := updated.(QueuePane)

    assert.Equal(t, 1, model.cursor)
    assert.Nil(t, cmd)
}
```

For commands that return a `tea.Cmd`, execute it to get the resulting message:

```go
updated, cmd := pane.Update(someMsg)
if cmd != nil {
    resultMsg := cmd()
    // assert on resultMsg type and payload
}
```

### Elm Architecture Purity Tests

Commands must not write to the Store. Test this by verifying the Store is unchanged
after a command runs:

```go
func TestBuildFetchCmd_DoesNotWriteStore(t *testing.T) {
    store := state.New()
    cmd := buildFetchCmd(mockClient)
    _ = cmd() // execute the command
    // Store should still be zero-value; only Update() writes to it
    assert.Nil(t, store.Playback())
}
```

---

## Integration Tests

Integration tests verify multi-component interactions: message routing through the root
model, state updates across panes, and end-to-end workflows with mocked HTTP.

### File Convention

Integration test files must:

1. Start with the `//go:build integration` build constraint
2. Use the filename pattern `*_integration_test.go`
3. Sit alongside their unit test counterparts in the same package

```go
//go:build integration

package mypackage_test

import "testing"

func TestIntegration_FullFlow(t *testing.T) {
    // ...
}
```

### What Qualifies as an Integration Test

- Tests that exercise message routing through the root `app.Model`
- Tests that verify state changes propagate from one pane to another
- Tests that combine `httptest.NewServer` with multiple model updates in sequence
- Tests that verify the polling tick produces correct downstream state changes

### What Stays as a Unit Test

- Individual API client methods with `httptest.NewServer` (testing one function)
- Store mutation methods (Get/Set)
- Bubble Tea model `Update()` handlers (testing one key → one command)
- `View()` output assertions
- Config loading, PKCE helpers, time formatters

### Running Integration Tests

```bash
make test-integration
```

This runs `go test -tags integration ./... -race -count=1` with `GOFLAGS=""` to prevent
the build's `-trimpath` flag from interfering with the `testhelpers.LoadFixture` path
resolver.

---

## Fixtures

JSON fixtures for API response mocking live in `testdata/fixtures/`. Name them
descriptively after the response they represent:

| File | Content |
|------|---------|
| `playback_state.json` | Full playback state with item, device, progress |
| `queue_response.json` | Currently playing + 2-track queue |
| `devices_response.json` | 3 devices (Computer, Smartphone, Speaker) |
| `devices_empty.json` | Empty device list |
| `playlists_response.json` | Paginated playlists |
| `playlist_tracks_response.json` | 2 tracks, no next page |
| `saved_albums_response.json` | 1 saved album |
| `liked_tracks_response.json` | 2 liked tracks |
| `recently_played_response.json` | 2 recent plays with timestamps |
| `top_tracks_response.json` | 2 top tracks |
| `top_artists_response.json` | 2 top artists with genres |
| `search_result.json` | Track, artist, album, playlist results |
| `play_history.json` | Single play history entry |
| `simple_playlist.json` | Single playlist object |
| `saved_album.json` | Single saved album object |
| `saved_track.json` | Single saved track object |

---

## Race Detector

All targets pass `-race`. Run explicitly:

```bash
go test -race ./...
```

Any data race is treated as a test failure and will block CI.
