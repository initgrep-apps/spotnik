package api

import "context"

// MockPlayer is a test double for PlayerAPI.
// Set the Result/Err fields before calling the method to control behavior.
// Called booleans record whether mutating methods were invoked.
type MockPlayer struct {
	PlaybackStateResult *PlaybackState
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
	QueueResult         *QueueResponse
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
var _ PlayerAPI = (*MockPlayer)(nil)

// GetPlaybackState returns the configured PlaybackStateResult and error.
func (m *MockPlayer) GetPlaybackState(_ context.Context) (*PlaybackState, error) {
	return m.PlaybackStateResult, m.PlaybackStateErr
}

// Play records the call and returns the configured error.
func (m *MockPlayer) Play(_ context.Context, _ PlayOptions) error {
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

// GetQueue returns the configured QueueResult and error.
func (m *MockPlayer) GetQueue(_ context.Context) (*QueueResponse, error) {
	return m.QueueResult, m.QueueErr
}

// MockLibrary is a test double for LibraryAPI.
type MockLibrary struct {
	PlaylistsResult      []SimplePlaylist
	PlaylistsErr         error
	PlaylistTracksResult []Track
	PlaylistTracksErr    error
	SavedAlbumsResult    []SavedAlbum
	SavedAlbumsErr       error
	LikedTracksResult    []SavedTrack
	LikedTracksErr       error
	RecentlyPlayedResult []PlayHistory
	RecentlyPlayedErr    error
	LikeErr              error
	UnlikeErr            error

	LikeTrackCalled   bool
	UnlikeTrackCalled bool
}

// Compile-time assertion: *MockLibrary must implement LibraryAPI.
var _ LibraryAPI = (*MockLibrary)(nil)

// GetPlaylists returns the configured result and error.
func (m *MockLibrary) GetPlaylists(_ context.Context, _, _ int) ([]SimplePlaylist, error) {
	return m.PlaylistsResult, m.PlaylistsErr
}

// GetPlaylistTracks returns the configured result and error.
func (m *MockLibrary) GetPlaylistTracks(_ context.Context, _ string, _, _ int) ([]Track, error) {
	return m.PlaylistTracksResult, m.PlaylistTracksErr
}

// GetSavedAlbums returns the configured result and error.
func (m *MockLibrary) GetSavedAlbums(_ context.Context, _, _ int) ([]SavedAlbum, error) {
	return m.SavedAlbumsResult, m.SavedAlbumsErr
}

// GetLikedTracks returns the configured result and error.
func (m *MockLibrary) GetLikedTracks(_ context.Context, _, _ int) ([]SavedTrack, error) {
	return m.LikedTracksResult, m.LikedTracksErr
}

// GetRecentlyPlayed returns the configured result and error.
func (m *MockLibrary) GetRecentlyPlayed(_ context.Context, _ int) ([]PlayHistory, error) {
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
	SearchResult *SearchResult
	SearchErr    error
}

// Compile-time assertion: *MockSearch must implement SearchAPI.
var _ SearchAPI = (*MockSearch)(nil)

// Search returns the configured result and error.
func (m *MockSearch) Search(_ context.Context, _ string, _ []string, _ int) (*SearchResult, error) {
	return m.SearchResult, m.SearchErr
}

// MockDevices is a test double for DevicesAPI.
type MockDevices struct {
	DevicesResult          []Device
	DevicesErr             error
	TransferErr            error
	TransferPlaybackCalled bool
}

// Compile-time assertion: *MockDevices must implement DevicesAPI.
var _ DevicesAPI = (*MockDevices)(nil)

// GetDevices returns the configured result and error.
func (m *MockDevices) GetDevices(_ context.Context) ([]Device, error) {
	return m.DevicesResult, m.DevicesErr
}

// TransferPlayback records the call and returns the configured error.
func (m *MockDevices) TransferPlayback(_ context.Context, _ string, _ bool) error {
	m.TransferPlaybackCalled = true
	return m.TransferErr
}

// MockUser is a test double for UserAPI.
type MockUser struct {
	TopTracksResult      []Track
	TopTracksErr         error
	TopArtistsResult     []FullArtist
	TopArtistsErr        error
	RecentlyPlayedResult []PlayHistory
	RecentlyPlayedErr    error
}

// Compile-time assertion: *MockUser must implement UserAPI.
var _ UserAPI = (*MockUser)(nil)

// GetTopTracks returns the configured result and error.
func (m *MockUser) GetTopTracks(_ context.Context, _ string, _ int) ([]Track, error) {
	return m.TopTracksResult, m.TopTracksErr
}

// GetTopArtists returns the configured result and error.
func (m *MockUser) GetTopArtists(_ context.Context, _ string, _ int) ([]FullArtist, error) {
	return m.TopArtistsResult, m.TopArtistsErr
}

// GetRecentlyPlayed returns the configured result and error.
func (m *MockUser) GetRecentlyPlayed(_ context.Context, _ int) ([]PlayHistory, error) {
	return m.RecentlyPlayedResult, m.RecentlyPlayedErr
}

// MockPlaylists is a test double for PlaylistsAPI.
type MockPlaylists struct {
	CreateResult         *SimplePlaylist
	CreateErr            error
	UpdateErr            error
	AddTracksErr         error
	RemoveTracksErr      error
	ReorderErr           error
	CreatePlaylistCalled bool
	UpdatePlaylistCalled bool
}

// Compile-time assertion: *MockPlaylists must implement PlaylistsAPI.
var _ PlaylistsAPI = (*MockPlaylists)(nil)

// CreatePlaylist records the call and returns the configured result and error.
func (m *MockPlaylists) CreatePlaylist(_ context.Context, _, _ string, _ bool) (*SimplePlaylist, error) {
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
