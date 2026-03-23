package api

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaybackState_Unmarshal(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/playback_state.json")
	require.NoError(t, err, "reading fixture file")

	var state PlaybackState
	err = json.Unmarshal(data, &state)
	require.NoError(t, err, "unmarshaling PlaybackState")

	assert.True(t, state.IsPlaying)
	assert.Equal(t, 154000, state.ProgressMs)
	assert.Equal(t, "off", state.RepeatState)
	assert.False(t, state.ShuffleState)
}

func TestPlaybackState_Unmarshal_NowPlaying(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/playback_state.json")
	require.NoError(t, err, "reading fixture file")

	var state PlaybackState
	err = json.Unmarshal(data, &state)
	require.NoError(t, err, "unmarshaling PlaybackState")

	assert.True(t, state.IsPlaying)
	require.NotNil(t, state.Item, "Item should not be nil when playing")
	assert.Equal(t, "Blinding Lights", state.Item.Name)
	require.NotNil(t, state.Device, "Device should not be nil when playing")
	assert.Equal(t, "MacBook Pro", state.Device.Name)
	assert.Equal(t, 65, state.Device.VolumePercent)
}

func TestTrack_Unmarshal(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/playback_state.json")
	require.NoError(t, err, "reading fixture file")

	var state PlaybackState
	err = json.Unmarshal(data, &state)
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
