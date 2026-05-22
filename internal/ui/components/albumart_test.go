package components

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tinyPNG returns a 1x1 red PNG encoded as bytes.
func tinyPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// TestAlbumArtRenderer_HasImage verifies HasImage is false initially and true after SetResult.
func TestAlbumArtRenderer_HasImage(t *testing.T) {
	var r AlbumArtRenderer
	assert.False(t, r.HasImage(), "new renderer should have no image")

	r.SetLoading("t1")
	r.SetResult("t1", []string{"row1", "row2"})
	assert.True(t, r.HasImage(), "renderer should have image after valid SetResult")
}

// TestAlbumArtRenderer_LoadingFlag verifies IsLoading transitions.
func TestAlbumArtRenderer_LoadingFlag(t *testing.T) {
	var r AlbumArtRenderer
	assert.False(t, r.IsLoading(), "new renderer should not be loading")

	r.SetLoading("t1")
	assert.True(t, r.IsLoading(), "renderer should be loading after SetLoading")
	assert.False(t, r.HasImage(), "loading clears cached rows")

	r.SetResult("t1", []string{"row1"})
	assert.False(t, r.IsLoading(), "SetResult clears loading flag")
}

// TestAlbumArtRenderer_SetResult_StaleTrackID verifies that a result for a different
// track ID is silently ignored and does not overwrite current state.
func TestAlbumArtRenderer_SetResult_StaleTrackID(t *testing.T) {
	var r AlbumArtRenderer
	r.SetLoading("t1")

	// Stale result arrives for a different track.
	r.SetResult("t2", []string{"row1"})
	assert.True(t, r.IsLoading(), "stale result must not clear loading flag")
	assert.False(t, r.HasImage(), "stale result must not set rows")

	// Correct result arrives.
	r.SetResult("t1", []string{"row1", "row2"})
	assert.False(t, r.IsLoading())
	assert.True(t, r.HasImage())
	assert.Equal(t, []string{"row1", "row2"}, r.Rows())
}

// TestFetchAlbumArtCmd_Success verifies that a mock HTTP server returning a valid
// PNG produces an AlbumArtFetchedMsg with non-empty rows and no error.
func TestFetchAlbumArtCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(tinyPNG())
	}))
	defer server.Close()

	cmd := FetchAlbumArtCmd("track-1", server.URL, 4, 8)
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(AlbumArtFetchedMsg)
	require.True(t, ok, "cmd should return AlbumArtFetchedMsg, got %T", msg)

	assert.Equal(t, "track-1", result.TrackID)
	assert.NotNil(t, result.Rows)
	assert.Greater(t, len(result.Rows), 0, "rendered rows should not be empty")
	assert.Nil(t, result.Err)
}

// TestFetchAlbumArtCmd_HTTP404 verifies that a 404 response from the server
// results in an AlbumArtFetchedMsg with a non-nil error and nil rows.
func TestFetchAlbumArtCmd_HTTP404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer server.Close()

	cmd := FetchAlbumArtCmd("track-1", server.URL, 4, 8)
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(AlbumArtFetchedMsg)
	require.True(t, ok, "cmd should return AlbumArtFetchedMsg, got %T", msg)

	assert.Nil(t, result.Rows)
	assert.NotNil(t, result.Err, "404 should produce an error")
}

// TestFetchAlbumArtCmd_NetworkError verifies that a failed HTTP connection
// (server closed) produces an error.
func TestFetchAlbumArtCmd_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close() // close immediately to force connection failure

	cmd := FetchAlbumArtCmd("track-1", server.URL, 4, 8)
	msg := cmd()
	result := msg.(AlbumArtFetchedMsg)
	assert.NotNil(t, result.Err, "closed server should produce an error")
}
