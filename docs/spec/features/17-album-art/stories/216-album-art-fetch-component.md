---
title: "Album art fetch + pixterm render component"
feature: 17-album-art
status: open
---

## Background

Spotify CDN image URLs in `domain.Album.Images` are public HTTPS URLs — no auth
header required. We fetch the image bytes with Go's stdlib `net/http`, decode via
the standard `image` package, and pass the reader to `pixterm/pkg/ansimage` which
renders it as ANSI block-character art. The rendered string is split into rows and
cached by track ID so identical tracks across re-polls never trigger a second
fetch.

The component is intentionally decoupled from the NowPlaying pane — it lives in
`internal/ui/components/` and communicates through a message (`AlbumArtFetchedMsg`).
Wiring into the pane's `Init()` and `handlePlaybackFetched()` happens here too.

## Design

### Dependency

```bash
go get github.com/eliukblau/pixterm@latest
```

Add to `go.mod` / `go.sum`. Import path used: `github.com/eliukblau/pixterm/pkg/ansimage`.

### `internal/ui/panes/messages.go` — new message type

```go
// AlbumArtFetchedMsg carries the pixterm-rendered rows for the current track's
// album art. Rows is nil when the fetch failed or no image was available.
type AlbumArtFetchedMsg struct {
    TrackID string
    Rows    []string // one ANSI-escaped string per terminal row; nil on error
    Err     error
}
```

### `internal/ui/components/albumart.go` — new file

```go
package components

import (
    "image/color"
    "io"
    "net/http"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/eliukblau/pixterm/pkg/ansimage"
)

// AlbumArtRenderer caches the last rendered album art and tracks loading state.
// It is value-safe to copy; the caller holds it by value inside the pane struct.
type AlbumArtRenderer struct {
    lastTrackID string
    rows        []string // nil when unavailable or not yet fetched
    loading     bool
}

// HasImage reports whether rendered rows are ready for display.
func (a *AlbumArtRenderer) HasImage() bool { return len(a.rows) > 0 }

// IsLoading reports whether a fetch is in flight.
func (a *AlbumArtRenderer) IsLoading() bool { return a.loading }

// Rows returns the rendered ANSI rows. Caller must check HasImage first.
func (a *AlbumArtRenderer) Rows() []string { return a.rows }

// NeedsRefresh reports whether trackID differs from the cached track.
func (a *AlbumArtRenderer) NeedsRefresh(trackID string) bool {
    return a.lastTrackID != trackID
}

// HandleFetched updates the renderer state from an AlbumArtFetchedMsg.
// It is idempotent — stale messages (wrong track ID) are ignored.
func (a *AlbumArtRenderer) HandleFetched(msg AlbumArtFetchedMsg) {
    // Import cycle avoided: AlbumArtFetchedMsg is declared in panes/messages.go;
    // the component package does not import panes. Callers (nowplaying.go) call
    // HandleFetched after type-asserting the tea.Msg — see wiring section below.
}
```

**Cycle note:** `components` must not import `panes` (would create a cycle).
`AlbumArtFetchedMsg` is declared in `panes/messages.go`. The pane's `Update()`
type-asserts the message and calls `a.artRenderer.HandleFetched(...)` passing
only the fields it needs. Alternatively, move the message type to a shared
`internal/ui/msgs` sub-package — whichever approach avoids the cycle is fine;
prefer the simpler one. The recommended path: keep `AlbumArtFetchedMsg` in
`panes/messages.go` and pass the fields explicitly to `AlbumArtRenderer`:

```go
// In AlbumArtRenderer:
func (a *AlbumArtRenderer) SetLoading(trackID string) {
    a.lastTrackID = trackID
    a.loading = true
    a.rows = nil
}
func (a *AlbumArtRenderer) SetResult(trackID string, rows []string) {
    if a.lastTrackID != trackID {
        return // stale
    }
    a.loading = false
    a.rows = rows
}
```

### `FetchAlbumArtCmd` — the Cmd

Declare as a package-level function in `internal/ui/components/albumart.go`.
It returns a `tea.Cmd` that must be dispatched as a Bubble Tea command — never
called directly.

```go
// FetchAlbumArtCmd downloads the image at url, renders it with pixterm to
// (rows × cols) terminal cells, splits the result into rows, and returns an
// AlbumArtFetchedMsg. cols should be rows*2 for a square appearance.
// The cmd must be dispatched via tea.Batch or returned from Update().
func FetchAlbumArtCmd(trackID, url string, rows, cols int) tea.Cmd {
    return func() tea.Msg {
        resp, err := http.Get(url) //nolint:gosec // public CDN, no user-controlled input
        if err != nil {
            return albumArtResult(trackID, nil, err)
        }
        defer resp.Body.Close()
        return renderFromReader(trackID, resp.Body, rows, cols)
    }
}

func renderFromReader(trackID string, r io.Reader, rows, cols int) tea.Msg {
    img, err := ansimage.NewScaledFromReader(
        r, rows, cols,
        color.Black,
        ansimage.ScaleModeResize,
        ansimage.NoDithering,
    )
    if err != nil {
        return albumArtResult(trackID, nil, err)
    }
    rendered := img.Render()
    return albumArtResult(trackID, strings.Split(strings.TrimRight(rendered, "\n"), "\n"), nil)
}

// albumArtResult is the internal constructor for AlbumArtFetchedMsg.
// Declared here to avoid importing panes — the caller wraps or re-uses
// the same struct fields. See wiring below for the actual return type used.
```

The return type of `FetchAlbumArtCmd` is `tea.Msg` — at the pane level the
`Update()` switch case receives it as `AlbumArtFetchedMsg`. To avoid the
import cycle, `FetchAlbumArtCmd` can return a locally-defined result struct
and the pane wraps it, OR `AlbumArtFetchedMsg` is moved to a shared package.
**Implementer decision:** choose whichever compiles cleanly without a cycle.
Document the choice in a `// NOTE:` comment.

### Wiring into `NowPlayingPane`

**`internal/ui/panes/nowplaying.go`**

Add field:
```go
artRenderer components.AlbumArtRenderer
```

In `Init()`:
```go
ps := a.store.PlaybackState()
if ps != nil && ps.Item != nil {
    if img := ps.Item.Album.BestImage(100); img != nil {
        cmds = append(cmds, components.FetchAlbumArtCmd(ps.Item.ID, img.URL, imageRows, imageCols))
        a.artRenderer.SetLoading(ps.Item.ID)
    }
}
```

Where `imageRows, imageCols` are computed from the current `bodyHeight` (see story
217 for exact formula). At `Init()` time use a conservative `imageRows = 8,
imageCols = 16` — the pane will re-render with correct dimensions after the first
`SetSize()` call.

In `handlePlaybackFetched(msg PlaybackStateFetchedMsg)`:
```go
if msg.State != nil && msg.State.Item != nil {
    track := msg.State.Item
    if a.artRenderer.NeedsRefresh(track.ID) {
        if img := track.Album.BestImage(100); img != nil {
            a.artRenderer.SetLoading(track.ID)
            return components.FetchAlbumArtCmd(track.ID, img.URL, a.imageRows(), a.imageCols())
        }
    }
}
```

`imageRows()` / `imageCols()` are helper methods on `NowPlayingPane` that return
the current image dimensions based on `bodyHeight` and render tier (see story 217).

In `Update()` — handle `AlbumArtFetchedMsg`:
```go
case AlbumArtFetchedMsg:
    a.artRenderer.SetResult(m.TrackID, m.Rows)
    return a, nil
```

### Loading placeholder

When `artRenderer.IsLoading()` is true and no image rows are ready, the image
column in `View()` renders a box of muted-color spaces (solid-background block)
using `theme.TextMuted()`. Width = `imageCols`, height = `imageRows`. This
signals to the user that art is incoming without showing broken layout.

## Acceptance Criteria

- [ ] `github.com/eliukblau/pixterm` added to `go.mod` / `go.sum`
- [ ] `AlbumArtFetchedMsg` declared (location: panes/messages.go or shared msgs package)
- [ ] `AlbumArtRenderer` struct with `HasImage`, `IsLoading`, `NeedsRefresh`, `SetLoading`, `SetResult`, `Rows` methods
- [ ] `FetchAlbumArtCmd` returns a `tea.Cmd`; renders via pixterm; splits output by `\n`
- [ ] Stale `AlbumArtFetchedMsg` (wrong track ID) ignored by `SetResult`
- [ ] NowPlaying `Init()` dispatches fetch if playback active at startup
- [ ] `handlePlaybackFetched` dispatches fetch on track ID change; skips same track
- [ ] No import cycle between `components` and `panes`
- [ ] `make ci` passes

## Tasks

- [ ] Run `go get github.com/eliukblau/pixterm@latest` and commit `go.mod` / `go.sum`
      - test: `go build ./...` compiles with new dependency

- [ ] Declare `AlbumArtFetchedMsg` in `internal/ui/panes/messages.go`
      - test: `go build ./internal/ui/panes/...` compiles

- [ ] Implement `AlbumArtRenderer` in `internal/ui/components/albumart.go`
      with `HasImage`, `IsLoading`, `NeedsRefresh`, `SetLoading`, `SetResult`, `Rows`
      - test: unit tests for stale-message guard (`SetResult` with wrong TrackID
        does not overwrite); loading flag toggled correctly; `HasImage` false before
        first result, true after valid result

- [ ] Implement `FetchAlbumArtCmd` in the same file; use pixterm `NewScaledFromReader`
      with `ScaleModeResize` + `NoDithering`; split rendered string by `\n`
      - test: mock HTTP server returns a 1×1 PNG; assert returned `AlbumArtFetchedMsg`
        has `len(Rows) > 0` and `Err == nil`; mock server returns 404 → `Rows == nil`
        and `Err != nil`

- [ ] Wire `artRenderer` field into `NowPlayingPane`; add `Init()` dispatch for
      active-at-startup case; add `handlePlaybackFetched` dispatch on track change
      - test: `TestNowPlayingPane_Init_FetchesArtWhenPlaying` — store has a track
        with `Images` populated; `Init()` returns a non-nil cmd; execute cmd and
        assert result is `AlbumArtFetchedMsg` with `TrackID == track.ID`

- [ ] Handle `AlbumArtFetchedMsg` in `NowPlayingPane.Update()`
      - test: feed a valid `AlbumArtFetchedMsg`; assert `artRenderer.HasImage() == true`;
        feed a stale one (wrong TrackID); assert `Rows()` unchanged

- [ ] `make ci` passes
