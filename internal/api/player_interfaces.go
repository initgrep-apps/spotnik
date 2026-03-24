package api

import "context"

// PlayerAPI defines all Spotify playback control operations.
// Concrete implementation: *Player.
type PlayerAPI interface {
	GetPlaybackState(ctx context.Context) (*PlaybackState, error)
	Play(ctx context.Context, opts PlayOptions) error
	Pause(ctx context.Context) error
	Next(ctx context.Context) error
	Previous(ctx context.Context) error
	Seek(ctx context.Context, positionMs int) error
	SetVolume(ctx context.Context, volume int) error
	SetShuffle(ctx context.Context, state bool) error
	SetRepeat(ctx context.Context, mode string) error
	AddToQueue(ctx context.Context, trackURI string) error
	GetQueue(ctx context.Context) (*QueueResponse, error)
}

// Compile-time assertion: *Player must implement PlayerAPI.
var _ PlayerAPI = (*Player)(nil)
