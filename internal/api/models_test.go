package api

import (
	"encoding/json"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaybackState_Unmarshal(t *testing.T) {
	data := testhelpers.LoadFixture(t, "playback_state.json")

	var state PlaybackState
	err := json.Unmarshal(data, &state)
	require.NoError(t, err, "unmarshaling PlaybackState")

	assert.True(t, state.IsPlaying)
	assert.Equal(t, 154000, state.ProgressMs)
	assert.Equal(t, "off", state.RepeatState)
	assert.False(t, state.ShuffleState)
}

func TestPlaybackState_Unmarshal_NowPlaying(t *testing.T) {
	data := testhelpers.LoadFixture(t, "playback_state.json")

	var state PlaybackState
	err := json.Unmarshal(data, &state)
	require.NoError(t, err, "unmarshaling PlaybackState")

	assert.True(t, state.IsPlaying)
	require.NotNil(t, state.Item, "Item should not be nil when playing")
	assert.Equal(t, "Blinding Lights", state.Item.Name)
	require.NotNil(t, state.Device, "Device should not be nil when playing")
	assert.Equal(t, "MacBook Pro", state.Device.Name)
	assert.Equal(t, 65, state.Device.VolumePercent)
}

func TestTrack_Unmarshal(t *testing.T) {
	data := testhelpers.LoadFixture(t, "playback_state.json")

	var state PlaybackState
	err := json.Unmarshal(data, &state)
	require.NoError(t, err, "unmarshaling PlaybackState")

	require.NotNil(t, state.Item)
	track := state.Item

	assert.Equal(t, "track-xyz789", track.ID)
	assert.Equal(t, "Blinding Lights", track.Name)
	assert.Equal(t, "spotify:track:track-xyz789", track.URI)
	assert.Equal(t, 252000, track.DurationMs)

	require.Len(t, track.Artists, 1)
	assert.Equal(t, "artist-weeknd", track.Artists[0].ID)
	assert.Equal(t, "The Weeknd", track.Artists[0].Name)

	assert.Equal(t, "album-after-hours", track.Album.ID)
	assert.Equal(t, "After Hours", track.Album.Name)
}

// TestSimplePlaylist_Unmarshal verifies JSON fixture parsing for SimplePlaylist.
func TestSimplePlaylist_Unmarshal(t *testing.T) {
	data := testhelpers.LoadFixture(t, "simple_playlist.json")

	var pl SimplePlaylist
	err := json.Unmarshal(data, &pl)
	require.NoError(t, err, "unmarshaling SimplePlaylist")

	assert.Equal(t, "playlist-abc123", pl.ID)
	assert.Equal(t, "Chill Vibes", pl.Name)
	assert.Equal(t, "spotify:playlist:playlist-abc123", pl.URI)
	assert.Equal(t, 42, pl.TrackCount)
	assert.Equal(t, "irshad", pl.Owner.DisplayName)
}

// TestSavedAlbum_Unmarshal verifies JSON fixture parsing for SavedAlbum.
func TestSavedAlbum_Unmarshal(t *testing.T) {
	data := testhelpers.LoadFixture(t, "saved_album.json")

	var sa SavedAlbum
	err := json.Unmarshal(data, &sa)
	require.NoError(t, err, "unmarshaling SavedAlbum")

	assert.Equal(t, "2024-01-15T10:30:00Z", sa.AddedAt)
	assert.Equal(t, "album-after-hours", sa.Album.ID)
	assert.Equal(t, "After Hours", sa.Album.Name)
	assert.Equal(t, "spotify:album:album-after-hours", sa.Album.URI)
	assert.Equal(t, 14, sa.Album.TotalTracks)
	require.Len(t, sa.Album.Artists, 1)
	assert.Equal(t, "The Weeknd", sa.Album.Artists[0].Name)
}

// TestSavedTrack_Unmarshal verifies JSON fixture parsing for SavedTrack.
func TestSavedTrack_Unmarshal(t *testing.T) {
	data := testhelpers.LoadFixture(t, "saved_track.json")

	var st SavedTrack
	err := json.Unmarshal(data, &st)
	require.NoError(t, err, "unmarshaling SavedTrack")

	assert.Equal(t, "2024-02-20T14:00:00Z", st.AddedAt)
	assert.Equal(t, "track-xyz789", st.Track.ID)
	assert.Equal(t, "Blinding Lights", st.Track.Name)
	assert.Equal(t, "spotify:track:track-xyz789", st.Track.URI)
	assert.Equal(t, 252000, st.Track.DurationMs)
}

// TestPlayHistory_Unmarshal verifies JSON fixture parsing for PlayHistory with played_at timestamp.
func TestPlayHistory_Unmarshal(t *testing.T) {
	data := testhelpers.LoadFixture(t, "play_history.json")

	var ph PlayHistory
	err := json.Unmarshal(data, &ph)
	require.NoError(t, err, "unmarshaling PlayHistory")

	assert.Equal(t, "2024-03-01T22:15:00Z", ph.PlayedAt)
	assert.Equal(t, "track-xyz789", ph.Track.ID)
	assert.Equal(t, "Blinding Lights", ph.Track.Name)
}

func TestPlaybackState_NilItem(t *testing.T) {
	// Simulate a 204-like scenario with a null item field.
	raw := `{
		"is_playing": false,
		"progress_ms": 0,
		"device": null,
		"shuffle_state": false,
		"repeat_state": "off",
		"item": null
	}`

	var state PlaybackState
	err := json.Unmarshal([]byte(raw), &state)
	require.NoError(t, err, "unmarshaling PlaybackState with null item")

	assert.False(t, state.IsPlaying)
	assert.Nil(t, state.Item, "Item should be nil")
	assert.Nil(t, state.Device, "Device should be nil")
}
