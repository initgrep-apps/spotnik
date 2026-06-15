package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestPlayer creates a Player pointing at the given base URL using the provided token.
func newTestPlayer(baseURL, token string) *Player {
	return NewPlayer(baseURL, token)
}

// TestPlaybackState_SendsAdditionalTypes verifies that the PlaybackState request
// includes the additional_types=episode query parameter.
func TestPlaybackState_SendsAdditionalTypes(t *testing.T) {
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "test-token")
	_, _ = player.PlaybackState(context.Background())

	assert.Contains(t, capturedQuery, "additional_types=episode")
	assert.Contains(t, capturedQuery, "market=from_token")
}

func TestGetPlaybackState(t *testing.T) {
	fixture := testhelpers.LoadFixture(t, "playback_state.json")

	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		wantNil    bool
		checkState func(t *testing.T, state *PlaybackState, err error)
	}{
		{
			name: "success parses state",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/me/player", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(fixture)
			},
			checkState: func(t *testing.T, state *PlaybackState, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, state)
				assert.True(t, state.IsPlaying)
				assert.Equal(t, "Blinding Lights", state.Item.Name)
				assert.Equal(t, 65, state.Device.VolumePercent)
			},
		},
		{
			name: "204 returns nil state no error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/me/player", r.URL.Path)
				w.WriteHeader(http.StatusNoContent)
			},
			checkState: func(t *testing.T, state *PlaybackState, err error) {
				t.Helper()
				require.NoError(t, err, "204 should return nil state, not error")
				assert.Nil(t, state, "nil state expected for 204 response")
			},
		},
		{
			name: "429 returns RateLimitError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Retry-After", "5")
				w.WriteHeader(http.StatusTooManyRequests)
			},
			checkState: func(t *testing.T, state *PlaybackState, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, state)
				var rateLimitErr *RateLimitError
				require.ErrorAs(t, err, &rateLimitErr)
				assert.Equal(t, 5, rateLimitErr.RetryAfter)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			player := newTestPlayer(srv.URL, "test-token")
			state, err := player.PlaybackState(context.Background())
			tt.checkState(t, state, err)
		})
	}
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

// TestAddToQueue verifies AddToQueue sends the correct URI param and handles errors.
func TestAddToQueue(t *testing.T) {
	tests := []struct {
		name       string
		trackURI   string
		statusCode int
		body       string
		wantErr    bool
		checkErr   func(t *testing.T, err error)
		checkReq   func(t *testing.T, r *http.Request)
	}{
		{
			name:       "success sends uri param",
			trackURI:   "spotify:track:abc123",
			statusCode: http.StatusNoContent,
			checkReq: func(t *testing.T, r *http.Request) {
				t.Helper()
				assert.Equal(t, "/v1/me/player/queue", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "spotify:track:abc123", r.URL.Query().Get("uri"), "URI query param should match the track URI")
			},
		},
		{
			name:       "403 returns ForbiddenError",
			trackURI:   "spotify:track:abc123",
			statusCode: http.StatusForbidden,
			body:       `{"error": "Premium required"}`,
			wantErr:    true,
			checkErr: func(t *testing.T, err error) {
				t.Helper()
				var forbiddenErr *ForbiddenError
				assert.ErrorAs(t, err, &forbiddenErr, "error should be ForbiddenError")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.checkReq != nil {
					tt.checkReq(t, r)
				}
				w.WriteHeader(tt.statusCode)
				if tt.body != "" {
					_, _ = w.Write([]byte(tt.body))
				}
			}))
			defer srv.Close()

			player := newTestPlayer(srv.URL, "test-token")
			err := player.AddToQueue(context.Background(), tt.trackURI)

			if tt.wantErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGetQueue verifies GetQueue parses the queue JSON correctly and handles errors.
func TestGetQueue(t *testing.T) {
	fixture := testhelpers.LoadFixture(t, "queue_response.json")

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		checkResp func(t *testing.T, resp *QueueResponse, err error)
	}{
		{
			name: "success parses queue",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/me/player/queue", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(fixture)
			},
			checkResp: func(t *testing.T, resp *QueueResponse, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, "Blinding Lights", resp.CurrentlyPlaying.Name, "currently_playing track name should match")
				require.Len(t, resp.Queue, 2, "queue should have 2 tracks")
				assert.Equal(t, "Save Your Tears", resp.Queue[0].Track.Name)
				assert.Equal(t, "Starboy", resp.Queue[1].Track.Name)
			},
		},
		{
			name: "500 returns error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error": "server error"}`))
			},
			checkResp: func(t *testing.T, resp *QueueResponse, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "500")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			player := newTestPlayer(srv.URL, "test-token")
			resp, err := player.Queue(context.Background())
			tt.checkResp(t, resp, err)
		})
	}
}

// TestQueueResponse_Parse verifies that the QueueResponse struct correctly
// deserialises both currently_playing and queue fields from the fixture.
func TestQueueResponse_Parse(t *testing.T) {
	fixture := testhelpers.LoadFixture(t, "queue_response.json")

	var qr QueueResponse
	err := json.Unmarshal(fixture, &qr)
	require.NoError(t, err)

	assert.Equal(t, "Blinding Lights", qr.CurrentlyPlaying.Name)
	assert.Equal(t, "spotify:track:track-xyz789", qr.CurrentlyPlaying.URI)
	require.Len(t, qr.Queue, 2)
	assert.Equal(t, "The Weeknd", qr.Queue[0].Track.Artists[0].Name)
}

// TestPlayerClient_GetPlaybackState_ImagesPopulated verifies that the PlaybackState
// response from /me/player correctly populates Album.Images through the JSON pipeline.
func TestPlayerClient_GetPlaybackState_ImagesPopulated(t *testing.T) {
	fixture := testhelpers.LoadFixture(t, "playback_with_images.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/player", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	player := newTestPlayer(srv.URL, "test-token")
	state, err := player.PlaybackState(context.Background())
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, state.Item)
	assert.Equal(t, "After Hours", state.Item.Album.Name)
	require.Len(t, state.Item.Album.Images, 3)
	assert.Equal(t, "https://i.scdn.co/image/ab640", state.Item.Album.Images[0].URL)
	assert.Equal(t, 640, state.Item.Album.Images[0].Width)
	assert.Equal(t, 640, state.Item.Album.Images[0].Height)
	assert.Equal(t, "https://i.scdn.co/image/ab300", state.Item.Album.Images[1].URL)
	assert.Equal(t, 300, state.Item.Album.Images[1].Width)
	assert.Equal(t, 300, state.Item.Album.Images[1].Height)
	assert.Equal(t, "https://i.scdn.co/image/ab64", state.Item.Album.Images[2].URL)
	assert.Equal(t, 64, state.Item.Album.Images[2].Width)
	assert.Equal(t, 64, state.Item.Album.Images[2].Height)
}
