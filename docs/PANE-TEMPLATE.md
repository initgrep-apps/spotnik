# Adding a New Pane — Step-by-Step Guide

This document walks through every change required to add a new pane to Spotnik.
Follow the steps in order; each step has a verification checkpoint.

The example below adds a hypothetical `ListenCountPane` (Page A, toggle key `9`).
Replace `ListenCount`, `PaneListenCount`, `listencount`, etc. with your pane's names.

---

## Overview of Changes

1. Add `PaneID` constant in `internal/ui/layout/`
2. Create the pane file in `internal/ui/panes/`
3. Add message types to `internal/ui/panes/messages.go`
4. Wire the pane into `internal/app/app.go` (constructor + handler)
5. Register the pane in the preset grid
6. Write tests and verify coverage

---

## Step 1 — Add `PaneID` Constant

**File:** `internal/ui/layout/layout.go` (or wherever `PaneID` constants are defined)

Add a new constant in the `iota` block. Page A panes use IDs 0–7; Page B uses 8–9.
New panes go after the last existing constant:

```go
const (
    PaneNowPlaying    PaneID = iota
    PaneQueue
    PanePlaylists
    PaneAlbums
    PaneLikedSongs
    PaneRecentlyPlayed
    PaneTopTracks
    PaneTopArtists
    PaneRequestFlow
    PaneNetworkLog
    PaneListenCount   // ← add here
)
```

**Verification:** `go build ./internal/ui/layout/...` — must compile.

---

## Step 2 — Create the Pane File

**File:** `internal/ui/panes/listencount_pane.go`

Use this scaffold (adapt types and API calls to your pane):

```go
package panes

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/initgrep-apps/spotnik/internal/state"
    "github.com/initgrep-apps/spotnik/internal/ui/layout"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// ListenCountPane displays the user's listening count statistics.
type ListenCountPane struct {
    store   *state.Store
    theme   theme.Theme
    focused bool
    width   int
    height  int
    // pane-specific fields
    count int
}

// NewListenCountPane creates a ListenCountPane backed by the given store and theme.
func NewListenCountPane(store *state.Store, th theme.Theme) *ListenCountPane {
    return &ListenCountPane{
        store: store,
        theme: th,
    }
}

// Init implements tea.Model. Returns nil — data is fetched via FetchListenCountRequestMsg.
func (p *ListenCountPane) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (p *ListenCountPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch m := msg.(type) {
    case ListenCountLoadedMsg:
        if m.Err == nil {
            p.count = m.Count
        }
    case tea.KeyMsg:
        if !p.focused {
            return p, nil
        }
        switch m.String() {
        case "q":
            // pane-specific key handling
        }
    }
    return p, nil
}

// View implements tea.Model. Must be pure — read state, return string, no I/O.
func (p *ListenCountPane) View() string {
    if p.width == 0 || p.height == 0 {
        return ""
    }
    // Render pane content within p.width x p.height
    return ""
}

// SetSize sets the content area dimensions (inside border).
func (p *ListenCountPane) SetSize(width, height int) {
    p.width = width
    p.height = height
}

// SetFocused sets keyboard focus state.
func (p *ListenCountPane) SetFocused(focused bool) { p.focused = focused }

// IsFocused returns keyboard focus state.
func (p *ListenCountPane) IsFocused() bool { return p.focused }

// ID returns the pane's slot identifier.
func (p *ListenCountPane) ID() layout.PaneID { return layout.PaneListenCount }

// Title returns the display title for the border.
func (p *ListenCountPane) Title() string { return "Listen Count" }

// ToggleKey returns the toggle key number (1-8 for Page A, 0 if not toggleable).
func (p *ListenCountPane) ToggleKey() int { return 0 } // adjust if Page A

// Actions returns pane-specific shortcuts for the border display.
func (p *ListenCountPane) Actions() []layout.Action { return nil }

// SetTheme updates the pane's theme for runtime switching.
// Table-based panes must rebuild their tables here with new column colors.
func (p *ListenCountPane) SetTheme(th theme.Theme) { p.theme = th }

// CountForTest exposes the internal count field for white-box unit tests.
// Only add helpers like this when testing internal state directly; prefer View() checks otherwise.
func (p *ListenCountPane) CountForTest() int { return p.count }
```

**Verification:** `go build ./internal/ui/panes/...` — must compile.

---

## Step 3 — Add Message Types

**File:** `internal/ui/panes/messages.go`

Add request and loaded message types for your pane's data fetch:

```go
// FetchListenCountRequestMsg triggers a listen count fetch.
type FetchListenCountRequestMsg struct{}

// ListenCountLoadedMsg carries the result of a listen count fetch.
type ListenCountLoadedMsg struct {
    Count int
    Err   error
}
```

**Convention:** `<Noun><Verb>Msg` — exported, with a data payload field and `Err error`.

**Verification:** `go build ./internal/ui/panes/...` — must compile.

---

## Step 4 — Wire Into `app.go`

Four changes in `internal/app/app.go`:

### 4a — Add field to `App` struct

```go
type App struct {
    // ... existing fields ...
    listenCountPane *panes.ListenCountPane
}
```

### 4b — Initialise in `New()`

```go
func New(store *state.Store, cfg *config.Config) *App {
    // ... existing init ...
    a.listenCountPane = panes.NewListenCountPane(store, th)
    return a
}
```

### 4c — Add command builder in `internal/app/commands.go`

```go
// buildFetchListenCountCmd fetches listen count data from the API.
// Returns ListenCountLoadedMsg with data or Err payload.
func buildFetchListenCountCmd(client UserAPI) tea.Cmd {
    return func() tea.Msg {
        count, err := client.ListenCount(context.Background())
        if err != nil {
            return panes.ListenCountLoadedMsg{Err: err}
        }
        return panes.ListenCountLoadedMsg{Count: count}
    }
}
```

### 4d — Handle the loaded message in `handleMsg`

```go
case panes.FetchListenCountRequestMsg:
    return a, buildFetchListenCountCmd(a.userAPI)

case panes.ListenCountLoadedMsg:
    if m.Err != nil {
        return a, a.alerts.NewAlertCmd("error", m.Err.Error())
    }
    a.store.SetListenCount(m.Count)
    return a, nil
```

**Architecture rules:**
- `buildFetchListenCountCmd` must never write to the Store
- `handleMsg` is the only place that writes to the Store
- Errors always route through `a.alerts.NewAlertCmd` — never render inline in `View()`

---

## Step 5 — Register in the Preset Grid

**File:** `internal/ui/layout/presets.go` (or wherever Page A presets are defined)

Add your pane to the appropriate preset row(s):

```go
// PresetDashboard — Full Dashboard
var PresetDashboard = Preset{
    Name: "Full Dashboard",
    Visible: map[PaneID]bool{
        PaneNowPlaying: true, PaneQueue: true, PanePlaylists: true,
        PaneAlbums: true, PaneLikedSongs: true, PaneRecentlyPlayed: true,
        PaneTopTracks: true, PaneTopArtists: true,
    },
    Grid: []Row{
        // Row 1: NowPlaying (full width)
        {HeightWeight: 2, Cells: []Cell{{PaneNowPlaying, 1}}},
        // Row 2: Library panes
        {HeightWeight: 3, Cells: []Cell{
            {PanePlaylists, 1}, {PaneAlbums, 1}, {PaneLikedSongs, 1},
        }},
        // Row 3: Stats panes
        {HeightWeight: 3, Cells: []Cell{
            {PaneQueue, 1}, {PaneRecentlyPlayed, 1},
            {PaneTopTracks, 1}, {PaneTopArtists, 1},
        }},
    },
}
```

For Page B panes, add to the Page B preset instead.

If the pane is toggleable (Page A, keys 1–8), set `ToggleKey()` to return the
appropriate number and add the toggle routing in `routing.go`.

---

## Step 6 — Write Tests

**File:** `internal/ui/panes/listencount_pane_test.go`

Use table-driven tests for all `Update()` paths and `View()` output:

```go
package panes_test

import (
    "errors"
    "testing"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/initgrep-apps/spotnik/internal/state"
    "github.com/initgrep-apps/spotnik/internal/ui/layout"
    "github.com/initgrep-apps/spotnik/internal/ui/panes"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func newTestListenCountPane(t *testing.T) *panes.ListenCountPane {
    t.Helper()
    return panes.NewListenCountPane(state.New(), theme.NewBlack())
}

func TestListenCountPane_ID(t *testing.T) {
    pane := newTestListenCountPane(t)
    assert.Equal(t, layout.PaneListenCount, pane.ID())
}

func TestListenCountPane_Update_LoadedMsg(t *testing.T) {
    tests := []struct {
        name      string
        msg       panes.ListenCountLoadedMsg
        wantCount int
        wantErr   bool
    }{
        {
            name:      "success",
            msg:       panes.ListenCountLoadedMsg{Count: 42},
            wantCount: 42,
        },
        {
            name:    "error preserved count zero",
            msg:     panes.ListenCountLoadedMsg{Err: errors.New("API error")},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            pane := newTestListenCountPane(t)
            pane.SetSize(80, 24)

            updated, cmd := pane.Update(tt.msg)
            model := updated.(*panes.ListenCountPane)

            assert.Nil(t, cmd)
            if !tt.wantErr {
                assert.Equal(t, tt.wantCount, model.CountForTest())
            }
        })
    }
}

func TestListenCountPane_View_EmptyBeforeResize(t *testing.T) {
    pane := newTestListenCountPane(t)
    assert.Equal(t, "", pane.View())
}
```

**Verification:**
```bash
go test ./internal/ui/panes/... -run TestListenCount -v
make test-coverage   # must stay ≥ 80%
make ci              # full gate must pass
```

---

## Checklist

- [ ] `PaneID` constant added to `internal/ui/layout/`
- [ ] Pane struct and all 9 interface methods implemented in `internal/ui/panes/`
- [ ] `SetTheme` implemented (rebuilds table if table-based)
- [ ] Message types added to `internal/ui/panes/messages.go`
- [ ] Command builder added to `internal/app/commands.go` (no Store writes)
- [ ] `handleMsg` handler added to `internal/app/app.go` (Store writes only here)
- [ ] Pane registered in preset grid
- [ ] Toggle key wired in `routing.go` (if Page A)
- [ ] Tests cover all `Update()` paths and `View()` states
- [ ] `make ci` passes
