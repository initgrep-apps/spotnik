---
title: "teatest setup + QueuePane golden POC"
feature: 21-test-infrastructure
status: open
---

## Background

Spotnik has ~3456 unit tests covering logic but zero tests verifying what the user actually
sees. A change to `PaneChrome` or column layout can break every pane silently. This story
adds the `teatest` dependency and proves the golden-file pattern on the simplest pane
(QueuePane) before expanding to all panes.

QueuePane is chosen as POC because: single data source (`QueueLoadedMsg`), simple table
layout, no sub-views or complex state transitions. Success here establishes the pattern
for all other panes.

## Design

### Dependency

```
go get github.com/charmbracelet/x/exp/teatest@latest
```

### Golden test helper: `internal/goldentest/golden.go`

Central helper to avoid repeating test boilerplate across 14 pane files:

```go
package goldentest

import (
    "bytes"
    "io"
    "os"
    "path/filepath"
    "testing"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/x/exp/teatest"
)

// NewPaneTest creates a teatest.TestModel with the given model and initial size.
// Convenience wrapper for pane golden tests.
func NewPaneTest(t *testing.T, model tea.Model, width, height int) *teatest.TestModel {
    t.Helper()
    return teatest.NewTestModel(t, model,
        teatest.WithInitialTermSize(width, height),
    )
}

// AssertGolden compares got against a golden file in testdata/.
// The golden file is named testdata/<t.Name()>.golden.
// Use `go test -update` to regenerate.
func AssertGolden(t *testing.T, got string) {
    t.Helper()
    name := filepath.Join("testdata", t.Name()+".golden")
    if os.Getenv("UPDATE_GOLDEN") != "" || updateGolden {
        os.MkdirAll(filepath.Dir(name), 0755)
        os.WriteFile(name, []byte(got), 0644)
        return
    }
    want, err := os.ReadFile(name)
    if err != nil {
        t.Fatalf("golden file missing: %s (run `go test -update` to generate)", name)
    }
    if got != string(want) {
        t.Fatalf("golden mismatch (-want +got):\n%s", diff(string(want), got))
    }
}

// ReadOutput reads all output from the test model's output reader.
func ReadOutput(tm *teatest.TestModel) string {
    t.Helper()
    out, _ := io.ReadAll(tm.Output())
    return string(out)
}

var updateGolden bool

func init() {
    // Register -update flag for golden regeneration
    testing.Init()
    for _, arg := range os.Args[1:] {
        if arg == "-update" {
            updateGolden = true
        }
    }
}
```

### QueuePane golden test: `internal/ui/panes/queue_golden_test.go`

```go
package panes_test

import (
    "testing"

    "github.com/initgrep-apps/spotnik/internal/domain"
    "github.com/initgrep-apps/spotnik/internal/goldentest"
    "github.com/initgrep-apps/spotnik/internal/state"
    "github.com/initgrep-apps/spotnik/internal/ui/panes"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

func TestQueuePane_View_WithTracks_Normal(t *testing.T) {
    s := state.New()
    th := theme.Load("black")
    s.SetQueue([]domain.QueueItem{
        {Type: domain.QueueItemTypeTrack, Track: &domain.Track{
            Name: "Blinding Lights", URI: "spotify:track:1",
            Artists: []domain.Artist{{Name: "The Weeknd"}},
            DurationMs: 200000,
        }},
        {Type: domain.QueueItemTypeTrack, Track: &domain.Track{
            Name: "Shape of You", URI: "spotify:track:2",
            Artists: []domain.Artist{{Name: "Ed Sheeran"}},
            DurationMs: 240000,
        }},
    })

    pane := panes.NewQueuePane(s, th)
    pane.SetSize(78, 10)
    pane.SetFocused(true)

    tm := goldentest.NewPaneTest(t, pane, 80, 24)
    goldentest.AssertGolden(t, goldentest.ReadOutput(tm))
}

func TestQueuePane_View_Empty(t *testing.T) {
    s := state.New()
    th := theme.Load("black")
    pane := panes.NewQueuePane(s, th)
    pane.SetSize(78, 10)
    pane.SetFocused(false)

    tm := goldentest.NewPaneTest(t, pane, 80, 24)
    goldentest.AssertGolden(t, goldentest.ReadOutput(tm))
}

func TestQueuePane_View_WithEpisodes_Narrow(t *testing.T) {
    s := state.New()
    th := theme.Load("black")
    s.SetQueue([]domain.QueueItem{
        {Type: domain.QueueItemTypeEpisode, Episode: &domain.Episode{
            Name: "The Future of AI", URI: "spotify:episode:1",
            DurationMs: 3600000,
            Show: &domain.Show{Name: "Tech Weekly"},
        }},
    })

    pane := panes.NewQueuePane(s, th)
    pane.SetSize(38, 10)
    pane.SetFocused(true)

    tm := goldentest.NewPaneTest(t, pane, 40, 24)
    goldentest.AssertGolden(t, goldentest.ReadOutput(tm))
}

func TestQueuePane_View_MixedContent(t *testing.T) {
    s := state.New()
    th := theme.Load("black")
    s.SetQueue([]domain.QueueItem{
        {Type: domain.QueueItemTypeTrack, Track: &domain.Track{
            Name: "Blinding Lights", URI: "spotify:track:1",
            Artists: []domain.Artist{{Name: "The Weeknd"}},
            DurationMs: 200000,
        }},
        {Type: domain.QueueItemTypeEpisode, Episode: &domain.Episode{
            Name: "The Future of AI", URI: "spotify:episode:1",
            DurationMs: 3600000,
            Show: &domain.Show{Name: "Tech Weekly"},
        }},
        {Type: domain.QueueItemTypeTrack, Track: &domain.Track{
            Name: "Shape of You", URI: "spotify:track:2",
            Artists: []domain.Artist{{Name: "Ed Sheeran"}},
            DurationMs: 240000,
        }},
    })

    pane := panes.NewQueuePane(s, th)
    pane.SetSize(78, 10)
    pane.SetFocused(true)

    tm := goldentest.NewPaneTest(t, pane, 80, 24)
    goldentest.AssertGolden(t, goldentest.ReadOutput(tm))
}

func TestQueuePane_View_FilterActive(t *testing.T) {
    s := state.New()
    th := theme.Load("black")
    s.SetQueue([]domain.QueueItem{
        {Type: domain.QueueItemTypeTrack, Track: &domain.Track{
            Name: "Blinding Lights", URI: "spotify:track:1",
            Artists: []domain.Artist{{Name: "The Weeknd"}},
            DurationMs: 200000,
        }},
        {Type: domain.QueueItemTypeTrack, Track: &domain.Track{
            Name: "Shape of You", URI: "spotify:track:2",
            Artists: []domain.Artist{{Name: "Ed Sheeran"}},
            DurationMs: 240000,
        }},
        {Type: domain.QueueItemTypeTrack, Track: &domain.Track{
            Name: "Starboy", URI: "spotify:track:3",
            Artists: []domain.Artist{{Name: "The Weeknd"}},
            DurationMs: 230000,
        }},
    })

    pane := panes.NewQueuePane(s, th)
    pane.SetSize(78, 10)
    pane.SetFocused(true)
    // Activate filter with query matching "The Weeknd"
    pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
    // Feed filter keystrokes via pane Update
    for _, r := range "The Weeknd" {
        pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
    }

    tm := goldentest.NewPaneTest(t, pane, 80, 24)
    goldentest.AssertGolden(t, goldentest.ReadOutput(tm))
}
```

## Files

### Create

- `internal/goldentest/golden.go` — golden test helper package (NewPaneTest, AssertGolden, ReadOutput)
- `internal/ui/panes/queue_golden_test.go` — QueuePane golden snapshot tests
- `internal/ui/panes/testdata/TestQueuePane_View_WithTracks_Normal.golden`
- `internal/ui/panes/testdata/TestQueuePane_View_Empty.golden`
- `internal/ui/panes/testdata/TestQueuePane_View_WithEpisodes_Narrow.golden`

### Modify

- `go.mod` — add `github.com/charmbracelet/x/exp/teatest`
- `.gitignore` — ensure `testdata/` is NOT ignored (golden files must be committed)

## Acceptance Criteria

- [ ] `github.com/charmbracelet/x/exp/teatest` added and `go mod tidy` succeeds
- [ ] `internal/goldentest/` package provides `AssertGolden`, `ReadOutput`, `NewPaneTest`
- [ ] `go test -update` regenerates golden files correctly
- [ ] `go test ./internal/ui/panes/ -run TestQueuePane_View` passes with committed golden files
- [ ] QueuePane snapshots include: tracks (normal width), empty state, episodes (narrow width), mixed content (tracks+episodes), filter active with matches, filter active with no matches
- [ ] Mixed content snapshot shows ♪ for tracks and ◆ for episodes in type column
- [ ] Filter-active snapshot shows filter input bar and filtered rows
- [ ] Golden files contain recognizable pane borders, column headers, and track names
- [ ] `make ci` passes

## Tasks

- [ ] Add teatest dependency and run `go mod tidy`
      - test: none (infrastructure)
- [ ] Create `internal/goldentest/golden.go` with AssertGolden, ReadOutput, NewPaneTest
      - test: `TestAssertGolden_Match`, `TestAssertGolden_Mismatch`, `TestReadOutput_ReturnsString`
- [ ] Create `internal/ui/panes/queue_golden_test.go` with 6 golden tests
      - test: `TestQueuePane_View_WithTracks_Normal`, `TestQueuePane_View_Empty`, `TestQueuePane_View_WithEpisodes_Narrow`, `TestQueuePane_View_MixedContent`, `TestQueuePane_View_FilterActive`, `TestQueuePane_View_FilterActive_NoMatches`
- [ ] Generate golden files: `go test ./internal/ui/panes/ -run TestQueuePane_View -update`
      - test: golden files committed, tests pass without `-update`
- [ ] Verify `make ci` passes with new dependency
      - test: `make ci`
