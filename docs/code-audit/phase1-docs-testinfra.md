# Phase 1 — Docs & Test Infrastructure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add all missing onboarding docs, fix ARCHITECTURE.md and DESIGN.md, centralise fixture loading, and add `make test-integration`.

**Architecture:** Purely additive — no behaviour changes. New docs land in `docs/`, new helper in `internal/testhelpers/`, fixture load calls replaced in-place in `internal/api/*_test.go`.

**Tech Stack:** Go 1.22+, standard `testing` package, `runtime` package for path resolution.

---

## File Map

| Action | Path | Purpose |
|--------|------|---------|
| Create | `internal/testhelpers/fixtures.go` | Centralised fixture loader |
| Create | `README.md` | Project overview, install, quick start |
| Create | `CONTRIBUTING.md` | PR process, code style, test rules |
| Create | `docs/DEV-SETUP.md` | Local dev prerequisites and workflow |
| Create | `docs/TESTING.md` | How to write tests, fixtures, mocks |
| Create | `docs/PANE-TEMPLATE.md` | Step-by-step guide for adding a new pane |
| Modify | `Makefile` | Add `test-integration` target |
| Modify | `docs/ARCHITECTURE.md` | Add 6 missing sections, fix Pane interface |
| Modify | `docs/DESIGN.md` | Fix §2, §17, §18 — themes, keybindings, interface |
| Modify | `internal/api/models_test.go` | Use `testhelpers.LoadFixture` |
| Modify | `internal/api/player_test.go` | Use `testhelpers.LoadFixture` |
| Modify | `internal/api/library_test.go` | Use `testhelpers.LoadFixture` |
| Modify | `internal/api/search_test.go` | Use `testhelpers.LoadFixture` |
| Modify | `internal/api/devices_test.go` | Use `testhelpers.LoadFixture` |
| Modify | `internal/api/playlists_test.go` | Use `testhelpers.LoadFixture` |
| Modify | `internal/api/user_test.go` | Use `testhelpers.LoadFixture` |

---

### Task 1: Create branch

- [ ] **Step 1: Create and switch to feature branch**

```bash
git checkout main && git pull origin main
git checkout -b chore/audit-phase1-docs-testinfra
```

---

### Task 2: Centralised fixture loader

**Files:**
- Create: `internal/testhelpers/fixtures.go`

- [ ] **Step 1: Write the fixture loader**

```go
// Package testhelpers provides shared utilities for tests across the Spotnik codebase.
// Import this package only in test files (_test.go).
package testhelpers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// fixturesDir is resolved once relative to this file's location so that
// LoadFixture works regardless of which package's test calls it.
var fixturesDir = func() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("testhelpers: cannot determine file location via runtime.Caller")
	}
	// internal/testhelpers/fixtures.go → project root is three levels up
	root := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(root, "testdata", "fixtures")
}()

// LoadFixture reads a JSON fixture file from testdata/fixtures/ by name.
// Fails the test immediately if the file cannot be read.
//
//	data := testhelpers.LoadFixture(t, "playback_state.json")
func LoadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(fixturesDir, name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "testhelpers.LoadFixture: failed to read %s", path)
	return data
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/irshadsheikh/dev/github/apps/spotnik && go build ./internal/testhelpers/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/testhelpers/fixtures.go
git commit -m "test(testhelpers): add centralised LoadFixture helper"
```

---

### Task 3: Replace inline fixture loads in api tests

**Files:**
- Modify: `internal/api/models_test.go`, `player_test.go`, `library_test.go`, `search_test.go`, `devices_test.go`, `playlists_test.go`, `user_test.go`

For each file that contains `os.ReadFile(` with a path to `testdata/fixtures/`:

- [ ] **Step 1: Add testhelpers import**

Add `"github.com/initgrep-apps/spotnik/internal/testhelpers"` to the import block of each affected file.

- [ ] **Step 2: Replace the load pattern**

Find every block matching this pattern:
```go
fixture, err := os.ReadFile("../../testdata/fixtures/<name>.json")
require.NoError(t, err)
```

Replace with:
```go
fixture := testhelpers.LoadFixture(t, "<name>.json")
```

The variable name (`fixture`, `data`, etc.) may differ — match the existing name. Remove the now-unused `err` variable and the `require.NoError` line.

- [ ] **Step 3: Remove unused `os` import** if `os` is no longer referenced in that file.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/api/... -race -count=1
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/api/
git commit -m "test(api): replace inline fixture loads with testhelpers.LoadFixture"
```

---

### Task 4: Add `make test-integration`

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Add target after the `test` target**

Open `Makefile`. After the `test:` block (around line 48), insert:

```makefile
## Run integration tests (requires build tag)
test-integration:
	@echo "→ Running integration tests..."
	$(GO) test -tags integration ./... -race -count=1
	@echo "✓ Integration tests passed"
```

Also add `test-integration` to the `.PHONY` line at the top.

- [ ] **Step 2: Verify**

```bash
make test-integration
```

Expected: integration tests run and pass (there are 5 integration test files).

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "chore(makefile): add test-integration target for build-tagged tests"
```

---

### Task 5: Write README.md

**Files:**
- Modify: `README.md` (currently 3 lines)

- [ ] **Step 1: Replace the file content**

```markdown
# Spotnik

A keyboard-driven terminal Spotify client for developers who live in the terminal.
Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Single binary. No Electron.

> Requires Spotify Premium.

## Features

- Now playing with seek bar and visualizer
- Keyboard-driven playlist, album, liked songs, and queue management
- Search with prefix autocomplete and per-category results
- Device switcher and theme switcher overlays
- 11 built-in themes (black, catppuccin, dracula, gruvbox, monokai, nord, rosé pine, solarized, synthwave, tokyo night, light)
- Btop-inspired multi-pane grid layout with 4 presets per page
- Real-time API gateway observability (Page B)

## Installation

**From source (requires Go 1.22+):**

```bash
git clone https://github.com/initgrep-apps/spotnik
cd spotnik
make build
./bin/spotnik auth   # one-time OAuth setup
./bin/spotnik        # launch
```

## Key Bindings

| Key | Action |
|-----|--------|
| `Space` | Play / Pause |
| `n` | Next track |
| `←` / `→` | Seek |
| `+` / `-` | Volume |
| `s` / `r` | Shuffle / Repeat |
| `/` | Search |
| `d` | Device switcher |
| `t` | Theme switcher |
| `p` | Cycle layout preset |
| `0` | Toggle Page A ↔ Page B |
| `1`–`8` | Toggle individual panes |
| `Tab` | Move focus |
| `f` | Filter (in list panes) |
| `q` | Quit |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [docs/DEV-SETUP.md](docs/DEV-SETUP.md).
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: expand README with features, install, and keybindings"
```

---

### Task 6: Write CONTRIBUTING.md

**Files:**
- Create: `CONTRIBUTING.md`

- [ ] **Step 1: Write the file**

```markdown
# Contributing to Spotnik

## Getting Started

See [docs/DEV-SETUP.md](docs/DEV-SETUP.md) for prerequisites and local setup.

## Branch Naming

```
feat/NN-feature-name    # new feature
fix/short-description   # bug fix
chore/short-description # non-feature work (docs, refactors, deps)
refactor/short-description
```

Never work directly on `main`.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(playback): add seek bar with keyboard controls
fix(auth): handle token refresh race condition
test(library): add table tests for pagination
refactor(state): extract polling into ticker command
chore(deps): upgrade bubbletea to v0.27.1
docs(architecture): document preference store subsystem
```

## Before Opening a PR

Run the full CI gate locally:

```bash
make ci
```

This runs: `fmt-check` → `tidy-check` → `lint` → `test-coverage` → `build`.
The PR will be rejected if any step fails.

## Test Requirements

- **80% coverage minimum** — enforced by `make test-coverage`
- Every new function in `api/`, `state/`, `config/` must have a test
- Use **table-driven tests** for functions with multiple input variants
- Use `httptest.NewServer` for API client tests — no external mock libraries
- Use `testhelpers.LoadFixture(t, "name.json")` to load JSON fixtures from `testdata/fixtures/`
- Integration tests (multi-component workflows) go in `*_integration_test.go` files with `//go:build integration` at the top

See [docs/TESTING.md](docs/TESTING.md) for patterns and examples.

## Code Style

- `gofmt` — enforced by `make fmt-check`
- `golangci-lint` — enforced by `make lint`
- Exported types, functions, and constants must have a doc comment
- Comments explain *why*, not *what*
- `// NOTE:` for non-obvious decisions
- No `panic()` in production code paths
- No `time.Sleep()` — use `tea.Tick`
- Never hardcode hex colour values — always use `Theme` interface tokens

## Architecture Rules

Read [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) before writing any code. Hard rules:

- All API data lives in the Store — never in a pane struct
- Side effects only via Commands — never call API inside `Update()` directly
- `View()` must be pure — no external calls, no state mutation
- Panes never talk to each other — only through messages routed via root model
- `ui/` never imports `api/` — data flows through messages and store only
- All API errors surface as toast notifications — never inline error boxes in `View()`

## Adding a New Pane

See [docs/PANE-TEMPLATE.md](docs/PANE-TEMPLATE.md) for a step-by-step guide.

## PR Process

1. Open a PR with title: `feat(name): brief description`
2. Body: tasks completed + test summary
3. A maintainer reviews and merges — do not merge your own PR
```

- [ ] **Step 2: Commit**

```bash
git add CONTRIBUTING.md
git commit -m "docs: add CONTRIBUTING.md with PR process, test rules, style guide"
```

---

### Task 7: Write docs/DEV-SETUP.md

**Files:**
- Create: `docs/DEV-SETUP.md`

- [ ] **Step 1: Write the file**

```markdown
# Developer Setup

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.22+ | `go version` to check |
| golangci-lint | latest | `brew install golangci-lint` or see [install guide](https://golangci-lint.run/usage/install/) |
| Spotify account | Premium | Required for playback API |
| Spotify app credentials | — | See below |

## Spotify App Setup

1. Go to [developer.spotify.com/dashboard](https://developer.spotify.com/dashboard)
2. Create an app (any name)
3. Set redirect URI to: `http://127.0.0.1:8080/callback`
4. Copy the **Client ID**

## Environment Setup

Create a `.env` file in the project root (gitignored):

```
SPOTIFY_CLIENT_ID=your_client_id_here
```

This is injected into the binary at build time via ldflags. The `.env` file is
loaded automatically by `make`.

## Build & Run

```bash
make build          # compile → bin/spotnik
make run            # build + run immediately
./bin/spotnik auth  # one-time OAuth authentication
./bin/spotnik       # launch the TUI
```

## Testing

```bash
make test                # unit tests (fast, default)
make test-integration    # integration tests (slower, multi-component)
make test-coverage       # unit tests + coverage report (80% minimum)
make ci                  # full gate: fmt + lint + test-coverage + build
```

## Linting

```bash
make lint    # run golangci-lint
make fmt     # auto-format all Go files
```

## Debugging Tips

- **Auth issues:** Run `./bin/spotnik auth` to re-authenticate. Tokens are stored in the OS keychain.
- **API rate limits:** The gateway backs off automatically. Watch Page B (press `0`) for gateway observability.
- **Terminal too small:** Spotnik requires at least 120×30. Resize your terminal.
- **Theme not applying:** Ensure `theme` in `~/.config/spotnik/config.toml` matches one of the 11 built-in theme IDs.

## Project Structure

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full layout with annotations.
Key packages:

| Package | Purpose |
|---------|---------|
| `internal/app/` | Root Bubble Tea model, message routing, render pipeline |
| `internal/api/` | Spotify HTTP clients and API gateway |
| `internal/state/` | Central Store — single source of truth |
| `internal/domain/` | Shared types bridging api/ and ui/ |
| `internal/ui/panes/` | Individual pane models (10 panes) |
| `internal/ui/components/` | Reusable UI components (table, filter, visualizer) |
| `internal/ui/layout/` | Grid layout engine and preset system |
| `internal/ui/theme/` | Theme interface and 11 implementations |
| `internal/config/` | TOML config loading and defaults |
| `internal/prefs/` | Runtime preference persistence (theme, preset, visualizer) |
```

- [ ] **Step 2: Commit**

```bash
git add docs/DEV-SETUP.md
git commit -m "docs: add DEV-SETUP.md with prerequisites, Spotify setup, and debugging tips"
```

---

### Task 8: Write docs/TESTING.md

**Files:**
- Create: `docs/TESTING.md`

- [ ] **Step 1: Write the file**

```markdown
# Testing Guide

## Running Tests

```bash
make test                # unit tests only (fast)
make test-integration    # integration tests only
make test-coverage       # unit tests + coverage (80% minimum enforced)
make ci                  # full gate including lint and build
```

## Test Philosophy

- **Table-driven tests** for functions with multiple input variants
- **`httptest.NewServer`** for all API client tests — no external mock libraries
- **`testhelpers.LoadFixture`** for loading JSON fixtures — no inline `os.ReadFile`
- **80% coverage minimum** — `make ci` fails below this threshold

## Writing a Unit Test

### API Client Test (httptest pattern)

```go
func TestPlayer_CurrentState_Success(t *testing.T) {
    fixture := testhelpers.LoadFixture(t, "playback_state.json")

    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/v1/me/player", r.URL.Path)
        assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(fixture)
    }))
    defer srv.Close()

    client := NewPlayer(srv.URL, "test-token")
    state, err := client.CurrentState(context.Background())
    require.NoError(t, err)
    assert.Equal(t, "Test Track", state.Item.Name)
}
```

### Table-Driven Test Pattern

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        wantErr  bool
        wantVal  int
    }{
        {name: "valid input", input: "ok", wantErr: false, wantVal: 42},
        {name: "empty input", input: "", wantErr: true, wantVal: 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            val, err := Something(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.wantVal, val)
        })
    }
}
```

### Pane Test Pattern

```go
func TestQueuePane_EnterPlaysTrack(t *testing.T) {
    store := state.New()
    store.SetQueue([]domain.Track{
        {ID: "t1", Name: "Test Track", Artists: []domain.Artist{{Name: "Artist"}}},
    })
    pane := NewQueuePane(store, theme.NewBlackTheme(), true)
    pane.SetSize(80, 20)

    _, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
    require.NotNil(t, cmd)

    msg := cmd()
    playMsg, ok := msg.(PlayTrackMsg)
    require.True(t, ok, "expected PlayTrackMsg, got %T", msg)
    assert.Equal(t, "t1", playMsg.TrackID)
}
```

## Mock API Clients

All Spotify API interfaces have hand-written mocks in `internal/api/apitest/mock.go`:

```go
// Inject results directly into mock fields
mock := &apitest.MockPlayer{
    PlaybackStateResult: &domain.PlaybackState{IsPlaying: true},
    PlaybackStateErr:    nil,
}

// Check if a method was called
assert.True(t, mock.PlayCalled)
```

## JSON Fixtures

Fixtures live in `testdata/fixtures/`. Load them with:

```go
data := testhelpers.LoadFixture(t, "playback_state.json")
```

To add a new fixture: capture a real Spotify API response, anonymise any user data, and save it as `testdata/fixtures/<descriptive-name>.json`.

## Integration Tests

Integration tests verify multi-component interactions. They live in `*_integration_test.go` files.

Every integration test file must start with:
```go
//go:build integration
```

What qualifies as an integration test:
- Tests that exercise message routing through the root `app.Model`
- Tests that verify state changes propagate from one pane to another
- Tests combining `httptest.NewServer` with multiple model `Update()` calls in sequence

Run integration tests with: `make test-integration`

## Elm Architecture Purity Tests

The codebase enforces Elm purity in dedicated test files:
- `internal/app/elm_purity_test.go` — verifies `View()` has no side effects
- `internal/app/command_safety_test.go` — verifies commands don't write to the Store

If you add a new command or modify a pane's `View()`, run these tests explicitly:
```bash
go test ./internal/app/... -run TestElmPurity -v
go test ./internal/app/... -run TestCommandSafety -v
```
```

- [ ] **Step 2: Commit**

```bash
git add docs/TESTING.md
git commit -m "docs: add TESTING.md with httptest, table-driven, fixture, and integration patterns"
```

---

### Task 9: Write docs/PANE-TEMPLATE.md

**Files:**
- Create: `docs/PANE-TEMPLATE.md`

- [ ] **Step 1: Write the file**

```markdown
# Adding a New Pane

This guide walks through adding a new pane to Spotnik. Use the `QueuePane`
(`internal/ui/panes/queue.go`) as a reference.

## Step 1: Add the PaneID constant

In `internal/ui/layout/pane.go`, add a new constant to the `PaneID` iota block:

```go
const (
    PaneNowPlaying     PaneID = iota
    // ... existing ...
    PaneMyNewPane                    // add here
)
```

## Step 2: Create the pane file

Create `internal/ui/panes/mynewpane.go`:

```go
package panes

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/initgrep-apps/spotnik/internal/state"
    "github.com/initgrep-apps/spotnik/internal/ui/components"
    "github.com/initgrep-apps/spotnik/internal/ui/layout"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Compile-time check: MyNewPane implements layout.Pane.
var _ layout.Pane = &MyNewPane{}

// MyNewPane displays [description].
type MyNewPane struct {
    store   *state.Store
    theme   theme.Theme
    focused bool
    width   int
    height  int
    table   *components.Table
    filter  *components.Filter
}

// NewMyNewPane creates a new MyNewPane.
func NewMyNewPane(store *state.Store, th theme.Theme, focused bool) *MyNewPane {
    columns := []components.ColumnDef{
        {Key: "index", Header: "#",     FlexFactor: 1, Color: th.ColumnIndex()},
        {Key: "name",  Header: "Name",  FlexFactor: 9, Color: th.ColumnPrimary()},
    }
    t := components.NewTable(components.TableConfig{
        Columns: columns, Theme: th, PlayingIndex: -1, ShowHeader: true,
    })
    p := &MyNewPane{store: store, theme: th, focused: focused, table: t, filter: components.NewFilter(th)}
    t.SetFocused(focused)
    p.refreshRows()
    return p
}

func (p *MyNewPane) ID() layout.PaneID    { return layout.PaneMyNewPane }
func (p *MyNewPane) Title() string         { return "My Pane" }
func (p *MyNewPane) ToggleKey() int        { return 0 } // set to 1-8 for Page A panes
func (p *MyNewPane) IsFocused() bool       { return p.focused }
func (p *MyNewPane) HasActiveFilter() bool { return p.filter.IsActive() }
func (p *MyNewPane) Actions() []layout.Action {
    return []layout.Action{{Key: "f", Label: "filter"}}
}

func (p *MyNewPane) Init() tea.Cmd { return nil }

func (p *MyNewPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if !p.focused { return p, nil }
        // handle keys
    }
    return p, nil
}

func (p *MyNewPane) View() string {
    // Pure render — read from store, return string. No side effects.
    return p.table.View()
}

func (p *MyNewPane) SetFocused(f bool) {
    p.focused = f
    p.table.SetFocused(f && !p.filter.IsActive())
}

func (p *MyNewPane) SetSize(w, h int) {
    p.width = w
    p.height = h
    p.table.SetSize(w, h)
}

func (p *MyNewPane) SetTheme(th theme.Theme) {
    p.theme = th
    p.filter = components.NewFilter(th)
    columns := []components.ColumnDef{
        {Key: "index", Header: "#",    FlexFactor: 1, Color: th.ColumnIndex()},
        {Key: "name",  Header: "Name", FlexFactor: 9, Color: th.ColumnPrimary()},
    }
    t := components.NewTable(components.TableConfig{
        Columns: columns, Theme: th, PlayingIndex: -1, ShowHeader: true,
    })
    t.SetFocused(p.focused && !p.filter.IsActive())
    p.table = t
    p.refreshRows()
}

func (p *MyNewPane) refreshRows() {
    // Build table.Row slice from store data and call p.table.SetRows(rows)
}
```

## Step 3: Define message types

Add to `internal/ui/panes/messages.go`:

```go
// MyNewPaneLoadedMsg carries data from the API for MyNewPane.
type MyNewPaneLoadedMsg struct {
    Data []domain.SomeType
    Err  error
}
```

## Step 4: Wire into app.go

In `internal/app/app.go` (or `handlers.go` after Phase 2 split):

```go
case panes.MyNewPaneLoadedMsg:
    if m.Err != nil {
        return a, a.alerts.NewAlertCmd("error", m.Err.Error())
    }
    a.store.SetMyNewData(m.Data)
```

Add the pane to the pane slice in `New()` and pass it to the layout manager.

## Step 5: Add to a preset

In `internal/ui/layout/presets.go`, add `PaneMyNewPane` to the appropriate `Row`/`Cell` definition.

## Step 6: Write tests

Create `internal/ui/panes/mynewpane_test.go`:

```go
func newTestMyNewPane(focused bool) *MyNewPane {
    store := state.New()
    return NewMyNewPane(store, theme.NewBlackTheme(), focused)
}

func TestMyNewPane_View_Empty(t *testing.T) {
    pane := newTestMyNewPane(true)
    pane.SetSize(80, 20)
    out := pane.View()
    assert.NotEmpty(t, out)
}
```

Run: `make test-coverage` to verify 80% threshold is maintained.
```

- [ ] **Step 2: Commit**

```bash
git add docs/PANE-TEMPLATE.md
git commit -m "docs: add PANE-TEMPLATE.md with step-by-step guide for new panes"
```

---

### Task 10: Update docs/ARCHITECTURE.md

**Files:**
- Modify: `docs/ARCHITECTURE.md`

- [ ] **Step 1: Add PreferenceStore section**

After the "State Management" section, insert:

```markdown
## Preference Store

`internal/prefs/prefs.go` provides debounced, thread-safe persistence for runtime preferences.
Three preferences are currently supported: `theme`, `preset`, and `visualizer`.

Preferences are written to `~/.config/spotnik/prefs.toml` with a debounce to avoid
excessive disk writes during rapid key presses (e.g., cycling themes quickly).

A generation counter pattern discards stale flush timers:
1. On preference change, increment `dirtyGen` and schedule a `schedulePrefsFlush` timer
2. On timer fire, compare captured gen to current `dirtyGen` — discard if stale
3. Only write to disk if gen matches

**Key files:**
- `internal/prefs/prefs.go` — `PreferenceStore` type, `Get`, `Set`, `Flush`
- `internal/app/app.go` — `schedulePrefsFlush()`, `PrefsDirtyGen()`, timer dispatch
```

- [ ] **Step 2: Add Page/Preset/Toggle system section**

After the "Render Pipeline" section, insert:

```markdown
## Page / Preset / Toggle System

The layout manager (`internal/ui/layout/layout.go`) manages three levels of view organisation:

| Level | Key | Behaviour |
|-------|-----|-----------|
| **Page** | `0` | Toggle between Page A (Music) and Page B (Nerd Status) |
| **Preset** | `p` | Cycle through layout presets within the current page |
| **Pane toggle** | `1`–`8` | Show/hide individual panes within Page A |

- Page A has 4 presets (Full Dashboard, Listening, Library, Discovery)
- Page B has 1 preset (Nerd Status)
- Preset choice and pane visibility are persisted via the PreferenceStore

**Key LayoutManager methods:**
- `TogglePage()` — switches between PageA and PageB
- `CyclePreset()` — advances to next preset in current page's list
- `TogglePane(id PaneID)` — shows/hides a single pane; rebuilds focus order
```

- [ ] **Step 3: Update Pane interface definition**

Find the section that describes the `Pane` interface. Add `SetTheme` to the method list:

```markdown
- `SetTheme(th theme.Theme)` — updates the pane's theme for runtime switching; table-based panes must rebuild their tables with new column colors
```

- [ ] **Step 4: Add View Lifecycle section**

After the "Architectural Overview" section, insert:

```markdown
## View Lifecycle

The app has three view modes managed by `currentView` in `internal/app/app.go`:

| Mode | Constant | Renders |
|------|----------|---------|
| Startup | `viewSplash` | ASCII banner, 5-second timer |
| Auth needed | `viewAuth` | OAuth instructions panel |
| Normal | `viewGrid` | Full pane grid with header and status bar |

**Transitions:**
- On startup: `viewSplash` → (5s timer fires `splashDismissMsg`)
- If unauthenticated: → `viewAuth`
- If authenticated: → `viewGrid`
- After successful auth: `viewAuth` → `viewGrid`
```

- [ ] **Step 5: Add Overlay Routing section**

In the "Message Flow" section, expand the routing description to include:

```markdown
### Overlay Routing Precedence

Key events are tested against guards in strict priority order:

1. **Theme overlay open** → all keys to ThemeOverlay
2. **Device overlay open** → all keys to DeviceOverlay
3. **Search overlay open** → all keys to SearchOverlay
4. **Auth view** → only quit keys pass through
5. **Pane has active filter** (`FilterablePane.HasActiveFilter()`) → all keys to focused pane
6. **Global shortcuts** (`q`, `/`, `d`, `t`, `p`, `0`, `1`–`8`, `Tab`)
7. **Playback keys** (`Space`, `n`, `+`, `-`, `s`, `r`, `v`, `←`, `→`) → always NowPlayingPane
8. **All other keys** → focused pane

`FilterablePane` is implemented by panes that support in-pane text filtering.
```

- [ ] **Step 6: Note the http.DefaultClient issue**

In the "API Client Design" section, add a note:

```markdown
> **Known issue (Phase 3):** `postTokenRequest` in `internal/api/auth.go` uses
> `http.DefaultClient` instead of the injected `*http.Client`, making it untestable
> via `httptest.NewServer`. This will be fixed in Phase 3.
```

- [ ] **Step 7: Verify doc still renders**

```bash
# Quick sanity: no broken markdown headings
grep "^#" docs/ARCHITECTURE.md | head -30
```

- [ ] **Step 8: Commit**

```bash
git add docs/ARCHITECTURE.md
git commit -m "docs(architecture): add PreferenceStore, page/preset/toggle, view lifecycle, overlay routing"
```

---

### Task 11: Update docs/DESIGN.md

**Files:**
- Modify: `docs/DESIGN.md`

- [ ] **Step 1: Fix §2 Pane interface — add SetTheme()**

Find the `type Pane interface` definition in §2. Add after the existing methods:

```markdown
| `SetTheme(th theme.Theme)` | Updates the pane's theme for runtime theme switching. Table-based panes must rebuild their tables with new column colors. |
```

- [ ] **Step 2: Fix §17 Keybindings — mark `?` as unimplemented**

Find the `?` row in the keybindings table. Change it from:

```
| `?` | Help | Global |
```

to:

```
| `?` | Help *(PLANNED — not yet implemented)* | Global |
```

- [ ] **Step 3: Add 6 missing themes to §18**

Read the six missing theme TOML files to get their color values:

```bash
cat internal/ui/theme/themes/dracula.toml
cat internal/ui/theme/themes/gruvbox.toml
cat internal/ui/theme/themes/rosepine.toml
cat internal/ui/theme/themes/solarized.toml
cat internal/ui/theme/themes/synthwave.toml
cat internal/ui/theme/themes/tokyonight.toml
```

For each, add a subsection to §18 following the same structure as the existing themes:
theme ID, display name, and a table of all token → hex value mappings.

Also update the theme count from "5 existing themes" to "11 themes".

- [ ] **Step 4: Condense repeated token structure in §18**

Instead of repeating the full token definition table for each theme, define tokens once
at the top of §18 and replace each per-theme token table with a compact hex-only table:

```markdown
### Token Reference

| Token method | Purpose |
|---|---|
| `Base()` | Terminal background (used for dimming) |
| `Background()` | Pane background |
| `Border()` | Pane border colour |
| `TextPrimary()` | Main text (track name, pane title) |
| `TextSecondary()` | Secondary text (artist name) |
| `TextMuted()` | De-emphasised text (index, duration) |
| `Accent()` | Active/selected highlight |
| `Success()` | Toast: success |
| `Warning()` | Toast: warning / rate-limit |
| `Error()` | Toast: error |
| `KeyHint()` | Keyboard shortcut hints |
| `ColumnIndex()` | Table index column |
| `ColumnPrimary()` | Table primary column |
| `ColumnSecondary()` | Table secondary column |
| `ColumnTertiary()` | Table tertiary column |
| `SeekBar()` | Seek bar fill |

### Theme Colour Values

| Theme | Base | Background | Border | TextPrimary | TextSecondary | TextMuted | Accent | Success | Warning | Error | KeyHint |
|---|---|---|---|---|---|---|---|---|---|---|---|
| black | ... | ... | ... (fill from toml files) |
| catppuccin | ... |
| ... |
```

- [ ] **Step 5: Commit**

```bash
git add docs/DESIGN.md
git commit -m "docs(design): add SetTheme to Pane interface, mark ? as planned, add 6 missing themes"
```

---

### Task 12: Final verification and PR

- [ ] **Step 1: Full CI gate**

```bash
make ci
```

Expected: all steps pass (fmt-check, tidy-check, lint, test-coverage, build).

- [ ] **Step 2: Integration tests**

```bash
make test-integration
```

Expected: all 5 integration test files pass.

- [ ] **Step 3: Push and open PR**

```bash
git push origin chore/audit-phase1-docs-testinfra
```

Open PR with title: `chore(docs): phase 1 — onboarding docs, ARCHITECTURE/DESIGN updates, fixture loader`

Body:
```
## Changes

- Add README.md, CONTRIBUTING.md, docs/DEV-SETUP.md, docs/TESTING.md, docs/PANE-TEMPLATE.md
- Add internal/testhelpers.LoadFixture — centralise fixture loading across 7 api test files
- Add `make test-integration` target
- Update docs/ARCHITECTURE.md: PreferenceStore, page/preset/toggle system, view lifecycle, overlay routing precedence, SetTheme in Pane interface
- Update docs/DESIGN.md: SetTheme in §2, mark ? key as planned, add 6 missing themes, condense token tables

## Test Summary

- make ci passes
- make test-integration passes
- All 7 api test files compile with testhelpers.LoadFixture
```
