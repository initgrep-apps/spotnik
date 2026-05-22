package components

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// canBindLocalhost checks whether the test environment allows binding to
// 127.0.0.1 (required for httptest). Sandbox environments often block this.
func canBindLocalhost() bool {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

// newLocalServer creates an httptest.Server bound to 127.0.0.1 instead of [::1]
// to work around sandbox IPv6 restrictions.
func newLocalServer(handler http.Handler) *httptest.Server {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	srv := httptest.NewUnstartedServer(handler)
	srv.Listener = l
	srv.Start()
	return srv
}

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

// TestRenderFromReader_Success verifies that a valid PNG byte stream renders
// to non-empty rows via pixterm without needing a real HTTP server.
func TestRenderFromReader_Success(t *testing.T) {
	msg := renderFromReader("track-1", bytes.NewReader(tinyPNG()), 4, 8)
	result, ok := msg.(AlbumArtFetchedMsg)
	require.True(t, ok, "should return AlbumArtFetchedMsg, got %T", msg)

	assert.Equal(t, "track-1", result.TrackID)
	assert.NotNil(t, result.Rows)
	assert.Greater(t, len(result.Rows), 0, "rendered rows should not be empty")
	assert.Nil(t, result.Err)
}

// TestRenderFromReader_InvalidData verifies that non-image input produces an error.
func TestRenderFromReader_InvalidData(t *testing.T) {
	msg := renderFromReader("track-1", bytes.NewReader([]byte("not an image")), 4, 8)
	result := msg.(AlbumArtFetchedMsg)
	assert.NotNil(t, result.Err, "invalid image data should produce an error")
	assert.Nil(t, result.Rows)
}

// TestFetchAlbumArtCmd_Success verifies that a mock HTTP server returning a valid
// PNG produces an AlbumArtFetchedMsg with non-empty rows and no error.
// Skipped in sandbox environments that block localhost binding.
func TestFetchAlbumArtCmd_Success(t *testing.T) {
	if !canBindLocalhost() {
		t.Skip("localhost binding not available in this environment")
	}
	server := newLocalServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
// Skipped in sandbox environments that block localhost binding.
func TestFetchAlbumArtCmd_HTTP404(t *testing.T) {
	if !canBindLocalhost() {
		t.Skip("localhost binding not available in this environment")
	}
	server := newLocalServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
// Skipped in sandbox environments that block localhost binding.
func TestFetchAlbumArtCmd_NetworkError(t *testing.T) {
	if !canBindLocalhost() {
		t.Skip("localhost binding not available in this environment")
	}
	server := newLocalServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close() // close immediately to force connection failure

	cmd := FetchAlbumArtCmd("track-1", server.URL, 4, 8)
	msg := cmd()
	result := msg.(AlbumArtFetchedMsg)
	assert.NotNil(t, result.Err, "closed server should produce an error")
}
