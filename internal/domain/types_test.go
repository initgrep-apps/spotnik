package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSimplePlaylist_UnmarshalJSON verifies that the custom unmarshaler extracts
// the nested tracks.total field into the flat TrackCount field.
func TestSimplePlaylist_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		json           string
		wantID         string
		wantName       string
		wantURI        string
		wantTrackCount int
		wantOwnerID    string
		wantErr        bool
	}{
		{
			name: "full playlist with track count",
			json: `{
				"id": "pl1",
				"name": "Chill Vibes",
				"uri": "spotify:playlist:pl1",
				"owner": {"id": "user1", "display_name": "Alice"},
				"items": {"total": 42}
			}`,
			wantID:         "pl1",
			wantName:       "Chill Vibes",
			wantURI:        "spotify:playlist:pl1",
			wantTrackCount: 42,
			wantOwnerID:    "user1",
		},
		{
			name: "zero track count",
			json: `{
				"id": "pl2",
				"name": "Empty",
				"uri": "spotify:playlist:pl2",
				"owner": {"id": "user2", "display_name": "Bob"},
				"items": {"total": 0}
			}`,
			wantID:         "pl2",
			wantName:       "Empty",
			wantURI:        "spotify:playlist:pl2",
			wantTrackCount: 0,
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p SimplePlaylist
			err := json.Unmarshal([]byte(tt.json), &p)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, p.ID)
			assert.Equal(t, tt.wantName, p.Name)
			assert.Equal(t, tt.wantURI, p.URI)
			assert.Equal(t, tt.wantTrackCount, p.TrackCount)
			if tt.wantOwnerID != "" {
				assert.Equal(t, tt.wantOwnerID, p.Owner.ID)
			}
		})
	}
}

// TestTrack_Explicit verifies that the Explicit field unmarshals correctly from JSON.
func TestTrack_Explicit(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantExpl bool
	}{
		{
			name:     "explicit true",
			json:     `{"id":"t1","name":"Bad","uri":"spotify:track:t1","explicit":true}`,
			wantExpl: true,
		},
		{
			name:     "explicit false",
			json:     `{"id":"t2","name":"Clean","uri":"spotify:track:t2","explicit":false}`,
			wantExpl: false,
		},
		{
			name:     "explicit missing defaults to false",
			json:     `{"id":"t3","name":"Old","uri":"spotify:track:t3"}`,
			wantExpl: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tr Track
			require.NoError(t, json.Unmarshal([]byte(tt.json), &tr))
			assert.Equal(t, tt.wantExpl, tr.Explicit)
		})
	}
}

// TestDomainTypes_JSONRoundtrip verifies that domain types serialize and
// deserialize correctly.
func TestDomainTypes_JSONRoundtrip(t *testing.T) {
	track := Track{
		ID:         "t1",
		Name:       "Test Track",
		URI:        "spotify:track:t1",
		DurationMs: 200000,
		Artists:    []Artist{{ID: "a1", Name: "Artist One"}},
		Album:      Album{ID: "al1", Name: "Album One"},
	}

	data, err := json.Marshal(track)
	require.NoError(t, err)

	var decoded Track
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, track.ID, decoded.ID)
	assert.Equal(t, track.Name, decoded.Name)
	assert.Equal(t, track.DurationMs, decoded.DurationMs)
	require.Len(t, decoded.Artists, 1)
	assert.Equal(t, "Artist One", decoded.Artists[0].Name)
}

// TestPlaybackState_Fields verifies PlaybackState fields marshal/unmarshal correctly.
func TestPlaybackState_Fields(t *testing.T) {
	raw := `{
		"is_playing": true,
		"progress_ms": 45000,
		"shuffle_state": true,
		"repeat_state": "context",
		"item": {"id": "t1", "name": "Song", "uri": "spotify:track:t1", "duration_ms": 200000},
		"device": {"id": "d1", "name": "MacBook", "type": "Computer", "volume_percent": 70}
	}`

	var ps PlaybackState
	require.NoError(t, json.Unmarshal([]byte(raw), &ps))
	assert.True(t, ps.IsPlaying)
	assert.Equal(t, 45000, ps.ProgressMs)
	assert.True(t, ps.ShuffleState)
	assert.Equal(t, "context", ps.RepeatState)
	require.NotNil(t, ps.Item)
	assert.Equal(t, "t1", ps.Item.ID)
	require.NotNil(t, ps.Device)
	assert.Equal(t, "MacBook", ps.Device.Name)
	assert.Equal(t, 70, ps.Device.VolumePercent)
}

// TestFullArtist_Fields verifies FullArtist fields marshal/unmarshal correctly,
// including the nested followers object that Spotify returns as {"href":null,"total":N}.
func TestFullArtist_Fields(t *testing.T) {
	raw := `{
		"id": "a1",
		"name": "The Weeknd",
		"uri": "spotify:artist:a1",
		"genres": ["pop", "r&b"],
		"popularity": 95,
		"followers": {"href": null, "total": 35000000},
		"external_urls": {"spotify": "https://open.spotify.com/artist/a1"}
	}`

	var fa FullArtist
	require.NoError(t, json.Unmarshal([]byte(raw), &fa))
	assert.Equal(t, "a1", fa.ID)
	assert.Equal(t, "The Weeknd", fa.Name)
	assert.Equal(t, []string{"pop", "r&b"}, fa.Genres)
	assert.Equal(t, 95, fa.Popularity)
	assert.Equal(t, 35000000, fa.Followers.Total)
	assert.Equal(t, "https://open.spotify.com/artist/a1", fa.ExternalURLs["spotify"])
}

// TestQueueResponse_Fields verifies QueueResponse fields unmarshal correctly.
func TestQueueResponse_Fields(t *testing.T) {
	raw := `{
		"currently_playing": {"id": "t1", "name": "Now Playing", "uri": "spotify:track:t1"},
		"queue": [
			{"id": "t2", "name": "Next Song", "uri": "spotify:track:t2"},
			{"id": "t3", "name": "Third Song", "uri": "spotify:track:t3"}
		]
	}`

	var qr QueueResponse
	require.NoError(t, json.Unmarshal([]byte(raw), &qr))
	assert.Equal(t, "t1", qr.CurrentlyPlaying.ID)
	require.Len(t, qr.Queue, 2)
	assert.Equal(t, "Next Song", qr.Queue[0].Name)
}

// TestAlbumImage_Unmarshal verifies that AlbumImage fields and the nested
// album.images array unmarshal correctly from JSON.
func TestAlbumImage_Unmarshal(t *testing.T) {
	raw := `{
		"id": "album-after-hours",
		"name": "After Hours",
		"images": [
			{"url": "https://i.scdn.co/image/ab640", "height": 640, "width": 640},
			{"url": "https://i.scdn.co/image/ab300", "height": 300, "width": 300},
			{"url": "https://i.scdn.co/image/ab64", "height": 64, "width": 64}
		]
	}`

	var album Album
	require.NoError(t, json.Unmarshal([]byte(raw), &album))
	assert.Equal(t, "album-after-hours", album.ID)
	assert.Equal(t, "After Hours", album.Name)
	require.Len(t, album.Images, 3)
	assert.Equal(t, "https://i.scdn.co/image/ab640", album.Images[0].URL)
	assert.Equal(t, 640, album.Images[0].Width)
	assert.Equal(t, 640, album.Images[0].Height)
	assert.Equal(t, "https://i.scdn.co/image/ab300", album.Images[1].URL)
	assert.Equal(t, 300, album.Images[1].Width)
	assert.Equal(t, 300, album.Images[1].Height)
	assert.Equal(t, "https://i.scdn.co/image/ab64", album.Images[2].URL)
	assert.Equal(t, 64, album.Images[2].Width)
	assert.Equal(t, 64, album.Images[2].Height)
}

// TestAlbum_BestImage verifies the BestImage selection logic.
func TestAlbum_BestImage(t *testing.T) {
	tests := []struct {
		name       string
		images     []AlbumImage
		minSize    int
		wantNil    bool
		wantURL    string
		wantWidth  int
		wantHeight int
	}{
		{
			name:    "empty images returns nil",
			images:  []AlbumImage{},
			minSize: 100,
			wantNil: true,
		},
		{
			name:       "all images below minSize falls back to largest",
			images:     []AlbumImage{{URL: "s", Width: 50, Height: 50}, {URL: "m", Width: 80, Height: 80}},
			minSize:    100,
			wantURL:    "m",
			wantWidth:  80,
			wantHeight: 80,
		},
		{
			name:       "multiple images above minSize returns smallest width",
			images:     []AlbumImage{{URL: "l", Width: 640, Height: 640}, {URL: "m", Width: 300, Height: 300}, {URL: "s", Width: 64, Height: 64}},
			minSize:    100,
			wantURL:    "m",
			wantWidth:  300,
			wantHeight: 300,
		},
		{
			name:       "exactly one image above minSize returns it",
			images:     []AlbumImage{{URL: "only", Width: 200, Height: 200}},
			minSize:    100,
			wantURL:    "only",
			wantWidth:  200,
			wantHeight: 200,
		},
		{
			name:       "image with only width above minSize but height too small is skipped",
			images:     []AlbumImage{{URL: "wide", Width: 200, Height: 50}, {URL: "tall", Width: 50, Height: 200}, {URL: "ok", Width: 150, Height: 150}},
			minSize:    100,
			wantURL:    "ok",
			wantWidth:  150,
			wantHeight: 150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Album{Images: tt.images}
			got := a.BestImage(tt.minSize)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.wantURL, got.URL)
			assert.Equal(t, tt.wantWidth, got.Width)
			assert.Equal(t, tt.wantHeight, got.Height)
		})
	}
}

// TestPlayOptions_JSON verifies PlayOptions marshals omitempty correctly.
func TestPlayOptions_JSON(t *testing.T) {
	// Only ContextURI set
	opts := PlayOptions{ContextURI: "spotify:playlist:pl1"}
	data, err := json.Marshal(opts)
	require.NoError(t, err)
	assert.Contains(t, string(data), "context_uri")
	assert.NotContains(t, string(data), "uris")

	// Only URIs set
	opts2 := PlayOptions{URIs: []string{"spotify:track:t1"}}
	data2, err := json.Marshal(opts2)
	require.NoError(t, err)
	assert.Contains(t, string(data2), "uris")
	assert.NotContains(t, string(data2), "context_uri")
}

// TestPlayOptions_WithOffset_MarshalJSON verifies that PlayOptions with an Offset
// field marshals to the expected JSON and that zero Offset is omitted.
func TestPlayOptions_WithOffset_MarshalJSON(t *testing.T) {
	tests := []struct {
		name       string
		opts       PlayOptions
		wantJSON   string
		notWantKey string
	}{
		{
			name: "context_uri with offset uri",
			opts: PlayOptions{
				ContextURI: "spotify:collection:tracks",
				Offset:     &PlayOffset{URI: "spotify:track:abc"},
			},
			wantJSON: `{"context_uri":"spotify:collection:tracks","offset":{"uri":"spotify:track:abc"}}`,
		},
		{
			name: "context_uri without offset omits offset field",
			opts: PlayOptions{
				ContextURI: "spotify:playlist:pl1",
			},
			wantJSON:   `{"context_uri":"spotify:playlist:pl1"}`,
			notWantKey: "offset",
		},
		{
			name: "uris list without offset omits offset field",
			opts: PlayOptions{
				URIs: []string{"spotify:track:t1", "spotify:track:t2"},
			},
			notWantKey: "offset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.opts)
			require.NoError(t, err)
			if tt.wantJSON != "" {
				assert.JSONEq(t, tt.wantJSON, string(data))
			}
			if tt.notWantKey != "" {
				assert.NotContains(t, string(data), tt.notWantKey,
					"JSON should not contain %q when Offset is nil", tt.notWantKey)
			}
		})
	}
}
