package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestPlayer creates a Player pointing at the given base URL using the provided token.
func newTestPlayer(baseURL, token string) *Player {
	return NewPlayer(baseURL, token)
}

func TestGetPlaybackState_Success(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/playback_state.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/player", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "test-token")
	state, err := player.GetPlaybackState(context.Background())

	require.NoError(t, err)
	require.NotNil(t, state)
	assert.True(t, state.IsPlaying)
	assert.Equal(t, "Blinding Lights", state.Item.Name)
	assert.Equal(t, 65, state.Device.VolumePercent)
}

func TestGetPlaybackState_204(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/player", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "test-token")
	state, err := player.GetPlaybackState(context.Background())

	require.NoError(t, err, "204 should return nil state, not error")
	assert.Nil(t, state, "nil state expected for 204 response")
}

func TestGetPlaybackState_429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "test-token")
	state, err := player.GetPlaybackState(context.Background())

	require.Error(t, err)
	assert.Nil(t, state)
	assert.Contains(t, err.Error(), "429")
}

func TestPlay_SendsCorrectBody(t *testing.T) {
	tests := []struct {
		name      string
		opts      PlayOptions
		wantField string
		wantValue interface{}
	}{
		{
			name:      "context URI",
			opts:      PlayOptions{ContextURI: "spotify:album:abc"},
			wantField: "context_uri",
			wantValue: "spotify:album:abc",
		},
		{
			name:      "track URIs",
			opts:      PlayOptions{URIs: []string{"spotify:track:xyz"}},
			wantField: "uris",
			wantValue: []interface{}{"spotify:track:xyz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/me/player/play", r.URL.Path)
				assert.Equal(t, http.MethodPut, r.Method)

				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, tt.wantValue, body[tt.wantField])

				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			player := newTestPlayer(srv.URL, "test-token")
			err := player.Play(context.Background(), tt.opts)
			require.NoError(t, err)
		})
	}
}

func TestPause_SendsPUT(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/player/pause", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "test-token")
	err := player.Pause(context.Background())
	require.NoError(t, err)
}

func TestNext_SendsPOST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/player/next", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "test-token")
	err := player.Next(context.Background())
	require.NoError(t, err)
}

func TestPrevious_SendsPOST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/player/previous", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "test-token")
	err := player.Previous(context.Background())
	require.NoError(t, err)
}

func TestSeek_SendsPositionMs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/player/seek", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		posStr := r.URL.Query().Get("position_ms")
		pos, _ := strconv.Atoi(posStr)
		assert.Equal(t, 30000, pos)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "test-token")
	err := player.Seek(context.Background(), 30000)
	require.NoError(t, err)
}

func TestSetVolume_ClampsRange(t *testing.T) {
	tests := []struct {
		name       string
		input      int
		wantVolume int
	}{
		{"normal", 65, 65},
		{"over 100 clamped", 120, 100},
		{"under 0 clamped", -5, 0},
		{"zero", 0, 0},
		{"exact 100", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/me/player/volume", r.URL.Path)
				assert.Equal(t, http.MethodPut, r.Method)

				volStr := r.URL.Query().Get("volume_percent")
				vol, _ := strconv.Atoi(volStr)
				assert.Equal(t, tt.wantVolume, vol, "volume should be clamped to %d", tt.wantVolume)

				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			player := newTestPlayer(srv.URL, "test-token")
			err := player.SetVolume(context.Background(), tt.input)
			require.NoError(t, err)
		})
	}
}

func TestSetShuffle_SendsState(t *testing.T) {
	tests := []struct {
		name  string
		state bool
		want  string
	}{
		{"shuffle on", true, "true"},
		{"shuffle off", false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/me/player/shuffle", r.URL.Path)
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, tt.want, r.URL.Query().Get("state"))
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			player := newTestPlayer(srv.URL, "test-token")
			err := player.SetShuffle(context.Background(), tt.state)
			require.NoError(t, err)
		})
	}
}

func TestSetRepeat_SendsMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{"off", "off"},
		{"context", "context"},
		{"track", "track"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/me/player/repeat", r.URL.Path)
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, tt.mode, r.URL.Query().Get("state"))
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			player := newTestPlayer(srv.URL, "test-token")
			err := player.SetRepeat(context.Background(), tt.mode)
			require.NoError(t, err)
		})
	}
}

// TestSetRepeat_InvalidMode verifies the API rejects invalid repeat modes.
func TestSetRepeat_InvalidMode(t *testing.T) {
	player := newTestPlayer("http://localhost", "token")
	err := player.SetRepeat(context.Background(), "invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repeat mode")
}

// TestPlayer_AuthorizationHeader verifies all requests include the Bearer token.
func TestPlayer_AuthorizationHeader(t *testing.T) {
	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "my-access-token")
	_ = player.Pause(context.Background())

	assert.Equal(t, "Bearer my-access-token", capturedAuth)
}

// TestPlayer_BaseURL verifies paths are correct relative to base URL.
func TestPlayer_BaseURL(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "token")
	_ = player.Next(context.Background())

	assert.Equal(t, "/v1/me/player/next", capturedPath)
}

// TestAddToQueue_Success verifies AddToQueue sends the correct URI param.
func TestAddToQueue_Success(t *testing.T) {
	var capturedURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/player/queue", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		capturedURI = r.URL.Query().Get("uri")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "test-token")
	err := player.AddToQueue(context.Background(), "spotify:track:abc123")

	require.NoError(t, err)
	assert.Equal(t, "spotify:track:abc123", capturedURI, "URI query param should match the track URI")
}

// TestAddToQueue_ServerError verifies AddToQueue returns a descriptive error on failure.
func TestAddToQueue_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": "Premium required"}`))
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "test-token")
	err := player.AddToQueue(context.Background(), "spotify:track:abc123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "403", "error should include status code")
}
