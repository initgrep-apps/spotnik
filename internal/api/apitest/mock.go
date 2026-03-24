package apitest

import (
	"context"

	"github.com/initgrep-apps/spotnik/internal/api"
)

// MockPlayer is a test double for PlayerAPI.
// Set the Result/Err fields before calling the method to control behavior.
// Called booleans record whether mutating methods were invoked.
type MockPlayer struct {
	PlaybackStateResult *api.PlaybackState
	PlaybackStateErr    error
	PlayErr             error
	PauseErr            error
	NextErr             error
	PreviousErr         error
	SeekErr             error
	SetVolumeErr        error
	SetShuffleErr       error
	SetRepeatErr        error
	AddToQueueErr       error
	QueueResult         *api.QueueResponse
	QueueErr            error

	PlayCalled       bool
	PauseCalled      bool
	NextCalled       bool
	PreviousCalled   bool
	SeekCalled       bool
	SetVolumeCalled  bool
	SetShuffleCalled bool
	SetRepeatCalled  bool
	AddToQueueCalled bool
}

// Compile-time assertion: *MockPlayer must implement PlayerAPI.
var _ api.PlayerAPI = (*MockPlayer)(nil)

// PlaybackState returns the configured PlaybackStateResult and error.
func (m *MockPlayer) PlaybackState(_ context.Context) (*api.PlaybackState, error) {
	return m.PlaybackStateResult, m.PlaybackStateErr
}

// Play records the call and returns the configured error.
func (m *MockPlayer) Play(_ context.Context, _ api.PlayOptions) error {
	m.PlayCalled = true
	return m.PlayErr
}

// Pause records the call and returns the configured error.
func (m *MockPlayer) Pause(_ context.Context) error {
	m.PauseCalled = true
	return m.PauseErr
}

// Next records the call and returns the configured error.
func (m *MockPlayer) Next(_ context.Context) error {
	m.NextCalled = true
	return m.NextErr
}

// Previous records the call and returns the configured error.
func (m *MockPlayer) Previous(_ context.Context) error {
	m.PreviousCalled = true
	return m.PreviousErr
}

// Seek records the call and returns the configured error.
func (m *MockPlayer) Seek(_ context.Context, _ int) error {
	m.SeekCalled = true
	return m.SeekErr
}

// SetVolume records the call and returns the configured error.
func (m *MockPlayer) SetVolume(_ context.Context, _ int) error {
	m.SetVolumeCalled = true
	return m.SetVolumeErr
}

// SetShuffle records the call and returns the configured error.
func (m *MockPlayer) SetShuffle(_ context.Context, _ bool) error {
	m.SetShuffleCalled = true
	return m.SetShuffleErr
}

// SetRepeat records the call and returns the configured error.
func (m *MockPlayer) SetRepeat(_ context.Context, _ string) error {
	m.SetRepeatCalled = true
	return m.SetRepeatErr
}

// AddToQueue records the call and returns the configured error.
func (m *MockPlayer) AddToQueue(_ context.Context, _ string) error {
	m.AddToQueueCalled = true
	return m.AddToQueueErr
}

// Queue returns the configured QueueResult and error.
func (m *MockPlayer) Queue(_ context.Context) (*api.QueueResponse, error) {
	return m.QueueResult, m.QueueErr
}

// MockLibrary is a test double for LibraryAPI.
type MockLibrary struct {
	PlaylistsResult      []api.SimplePlaylist
	PlaylistsErr         error
	PlaylistTracksResult []api.Track
	PlaylistTracksErr    error
	SavedAlbumsResult    []api.SavedAlbum
	SavedAlbumsErr       error
	LikedTracksResult    []api.SavedTrack
	LikedTracksErr       error
	RecentlyPlayedResult []api.PlayHistory
	RecentlyPlayedErr    error
	LikeErr              error
	UnlikeErr            error

	LikeTrackCalled   bool
	UnlikeTrackCalled bool
}

// Compile-time assertion: *MockLibrary must implement LibraryAPI.
var _ api.LibraryAPI = (*MockLibrary)(nil)

// Playlists returns the configured result and error.
func (m *MockLibrary) Playlists(_ context.Context, _, _ int) ([]api.SimplePlaylist, error) {
	return m.PlaylistsResult, m.PlaylistsErr
}

// PlaylistTracks returns the configured result and error.
func (m *MockLibrary) PlaylistTracks(_ context.Context, _ string, _, _ int) ([]api.Track, error) {
	return m.PlaylistTracksResult, m.PlaylistTracksErr
}

// SavedAlbums returns the configured result and error.
func (m *MockLibrary) SavedAlbums(_ context.Context, _, _ int) ([]api.SavedAlbum, error) {
	return m.SavedAlbumsResult, m.SavedAlbumsErr
}

// LikedTracks returns the configured result and error.
func (m *MockLibrary) LikedTracks(_ context.Context, _, _ int) ([]api.SavedTrack, error) {
	return m.LikedTracksResult, m.LikedTracksErr
}

// RecentlyPlayed returns the configured result and error.
func (m *MockLibrary) RecentlyPlayed(_ context.Context, _ int) ([]api.PlayHistory, error) {
	return m.RecentlyPlayedResult, m.RecentlyPlayedErr
}

// LikeTrack records the call and returns the configured error.
func (m *MockLibrary) LikeTrack(_ context.Context, _ string) error {
	m.LikeTrackCalled = true
	return m.LikeErr
}

// UnlikeTrack records the call and returns the configured error.
func (m *MockLibrary) UnlikeTrack(_ context.Context, _ string) error {
	m.UnlikeTrackCalled = true
	return m.UnlikeErr
}

// MockSearch is a test double for SearchAPI.
type MockSearch struct {
	SearchResult *api.SearchResult
	SearchErr    error
}

// Compile-time assertion: *MockSearch must implement SearchAPI.
var _ api.SearchAPI = (*MockSearch)(nil)

// Search returns the configured result and error.
func (m *MockSearch) Search(_ context.Context, _ string, _ []string, _ int) (*api.SearchResult, error) {
	return m.SearchResult, m.SearchErr
}

// MockDevices is a test double for DevicesAPI.
type MockDevices struct {
	DevicesResult          []api.Device
	DevicesErr             error
	TransferErr            error
	TransferPlaybackCalled bool
}

// Compile-time assertion: *MockDevices must implement DevicesAPI.
var _ api.DevicesAPI = (*MockDevices)(nil)

// Devices returns the configured result and error.
func (m *MockDevices) Devices(_ context.Context) ([]api.Device, error) {
	return m.DevicesResult, m.DevicesErr
}

// TransferPlayback records the call and returns the configured error.
func (m *MockDevices) TransferPlayback(_ context.Context, _ string, _ bool) error {
	m.TransferPlaybackCalled = true
	return m.TransferErr
}

// MockUser is a test double for UserAPI.
type MockUser struct {
	TopTracksResult      []api.Track
	TopTracksErr         error
	TopArtistsResult     []api.FullArtist
	TopArtistsErr        error
	RecentlyPlayedResult []api.PlayHistory
	RecentlyPlayedErr    error
}

// Compile-time assertion: *MockUser must implement UserAPI.
var _ api.UserAPI = (*MockUser)(nil)

// TopTracks returns the configured result and error.
func (m *MockUser) TopTracks(_ context.Context, _ string, _ int) ([]api.Track, error) {
	return m.TopTracksResult, m.TopTracksErr
}

// TopArtists returns the configured result and error.
func (m *MockUser) TopArtists(_ context.Context, _ string, _ int) ([]api.FullArtist, error) {
	return m.TopArtistsResult, m.TopArtistsErr
}

// RecentlyPlayed returns the configured result and error.
func (m *MockUser) RecentlyPlayed(_ context.Context, _ int) ([]api.PlayHistory, error) {
	return m.RecentlyPlayedResult, m.RecentlyPlayedErr
}

// MockPlaylists is a test double for PlaylistsAPI.
type MockPlaylists struct {
	CreateResult         *api.SimplePlaylist
	CreateErr            error
	UpdateErr            error
	AddTracksErr         error
	RemoveTracksErr      error
	ReorderErr           error
	CreatePlaylistCalled bool
	UpdatePlaylistCalled bool
}

// Compile-time assertion: *MockPlaylists must implement PlaylistsAPI.
var _ api.PlaylistsAPI = (*MockPlaylists)(nil)

// CreatePlaylist records the call and returns the configured result and error.
func (m *MockPlaylists) CreatePlaylist(_ context.Context, _, _ string, _ bool) (*api.SimplePlaylist, error) {
	m.CreatePlaylistCalled = true
	return m.CreateResult, m.CreateErr
}

// UpdatePlaylist records the call and returns the configured error.
func (m *MockPlaylists) UpdatePlaylist(_ context.Context, _, _, _ string) error {
	m.UpdatePlaylistCalled = true
	return m.UpdateErr
}

// AddTracksToPlaylist returns the configured error.
func (m *MockPlaylists) AddTracksToPlaylist(_ context.Context, _ string, _ []string) error {
	return m.AddTracksErr
}

// RemoveTracksFromPlaylist returns the configured error.
func (m *MockPlaylists) RemoveTracksFromPlaylist(_ context.Context, _ string, _ []string) error {
	return m.RemoveTracksErr
}

// ReorderPlaylistTracks returns the configured error.
func (m *MockPlaylists) ReorderPlaylistTracks(_ context.Context, _ string, _, _, _ int) error {
	return m.ReorderErr
}
