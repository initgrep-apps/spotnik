package state

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestStore_SetGetPlaybackState(t *testing.T) {
	s := New()

	// Initially nil.
	assert.Nil(t, s.PlaybackState())

	track := &api.Track{
		ID:         "track-1",
		Name:       "Test Track",
		DurationMs: 200000,
	}
	state := &api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 5000,
		Item:       track,
	}

	s.SetPlaybackState(state)
	got := s.PlaybackState()

	assert.NotNil(t, got)
	assert.True(t, got.IsPlaying)
	assert.Equal(t, 5000, got.ProgressMs)
	assert.Equal(t, "Test Track", got.Item.Name)
}

func TestStore_SetPlaybackState_Nil(t *testing.T) {
	s := New()

	// Set something first.
	s.SetPlaybackState(&api.PlaybackState{IsPlaying: true})
	assert.NotNil(t, s.PlaybackState())

	// Clear it.
	s.SetPlaybackState(nil)
	assert.Nil(t, s.PlaybackState())
}

func TestStore_SetGetActiveDevice(t *testing.T) {
	s := New()

	assert.Nil(t, s.ActiveDevice())

	device := &api.Device{
		ID:            "dev-1",
		Name:          "MacBook Pro",
		VolumePercent: 70,
	}

	s.SetActiveDevice(device)
	got := s.ActiveDevice()

	assert.NotNil(t, got)
	assert.Equal(t, "MacBook Pro", got.Name)
	assert.Equal(t, 70, got.VolumePercent)
}
