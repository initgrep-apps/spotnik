package components

import (
	"image/color"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eliukblau/pixterm/pkg/ansimage"
)

// AlbumArtFetchedMsg carries the pixterm-rendered rows for the current track's
// album art. Rows is nil when the fetch failed or no image was available.
type AlbumArtFetchedMsg struct {
	TrackID string
	Rows    []string // one ANSI-escaped string per terminal row; nil on error
	Err     error
}

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

// SetLoading marks the renderer as loading for the given track, clearing any
// previously cached rows.
func (a *AlbumArtRenderer) SetLoading(trackID string) {
	a.lastTrackID = trackID
	a.loading = true
	a.rows = nil
}

// SetResult stores the rendered rows if the track ID matches the current load.
// A stale result (wrong track ID) is silently ignored.
func (a *AlbumArtRenderer) SetResult(trackID string, rows []string) {
	if a.lastTrackID != trackID {
		return // stale
	}
	a.loading = false
	a.rows = rows
}

// FetchAlbumArtCmd downloads the image at url, renders it with pixterm to
// (rows × cols) terminal cells, splits the result into rows, and returns an
// AlbumArtFetchedMsg. cols should be rows*2 for a square appearance.
// The cmd must be dispatched via tea.Batch or returned from Update().
func FetchAlbumArtCmd(trackID, url string, rows, cols int) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(url) //nolint:gosec // public CDN, no user-controlled input
		if err != nil {
			return albumArtResult(trackID, nil, err)
		}
		defer func() { _ = resp.Body.Close() }()
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

func albumArtResult(trackID string, rows []string, err error) AlbumArtFetchedMsg {
	return AlbumArtFetchedMsg{TrackID: trackID, Rows: rows, Err: err}
}
