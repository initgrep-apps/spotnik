// Package domain contains the shared data types for the Spotnik application.
// These types represent Spotify API entities and are used across the api/, state/,
// and ui/ packages without creating import cycles.
//
// Design: domain/ has no dependencies on other internal packages, allowing both
// api/ and ui/ to import it safely.
package domain

import "encoding/json"

// unmarshalJSON is a package-level helper for custom UnmarshalJSON methods.
func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// UserProfile represents the authenticated user's Spotify profile,
// as returned by GET /v1/me.
type UserProfile struct {
	// ID is the Spotify user ID. Used to distinguish owned vs followed playlists.
	ID string `json:"id"`

	// DisplayName is the user's Spotify display name.
	DisplayName string `json:"display_name"`

	// Product is the subscription tier: "premium" or "free".
	Product string `json:"product"`

	// Country is the ISO 3166-1 alpha-2 country code (e.g. "DE").
	Country string `json:"country"`
}

// PlaybackState represents the full playback state returned by GET /me/player.
// When Spotify returns 204 (nothing playing), this struct is nil in the store.
type PlaybackState struct {
	// IsPlaying indicates whether Spotify is actively playing audio.
	IsPlaying bool `json:"is_playing"`

	// ProgressMs is the current playback position in milliseconds.
	ProgressMs int `json:"progress_ms"`

	// ShuffleState indicates whether shuffle is enabled.
	ShuffleState bool `json:"shuffle_state"`

	// RepeatState is one of "off", "context", or "track".
	RepeatState string `json:"repeat_state"`

	// Item is the currently playing track. May be nil if nothing is playing.
	Item *Track `json:"item"`

	// Device is the currently active playback device. May be nil if no device.
	Device *Device `json:"device"`
}

// Track represents a Spotify track item returned in the playback state.
type Track struct {
	// ID is the Spotify track ID.
	ID string `json:"id"`

	// Name is the display name of the track.
	Name string `json:"name"`

	// URI is the Spotify URI of the track (e.g. "spotify:track:...").
	URI string `json:"uri"`

	// DurationMs is the total duration of the track in milliseconds.
	DurationMs int `json:"duration_ms"`

	// Artists is the list of artists for this track.
	Artists []Artist `json:"artists"`

	// Album is the album this track belongs to.
	Album Album `json:"album"`

	// Explicit indicates whether the track contains explicit content.
	Explicit bool `json:"explicit"`
}

// Artist represents a Spotify artist.
type Artist struct {
	// ID is the Spotify artist ID.
	ID string `json:"id"`

	// Name is the display name of the artist.
	Name string `json:"name"`
}

// Album represents a simplified Spotify album (as returned within track objects).
type Album struct {
	// ID is the Spotify album ID.
	ID string `json:"id"`

	// Name is the display name of the album.
	Name string `json:"name"`
}

// SimplePlaylistOwner represents the owner of a playlist.
type SimplePlaylistOwner struct {
	// ID is the Spotify user ID of the owner.
	ID string `json:"id"`

	// DisplayName is the human-readable name of the owner.
	DisplayName string `json:"display_name"`
}

// SimplePlaylist represents a simplified Spotify playlist as returned
// in the GET /me/playlists response items.
// NOTE: UnmarshalJSON is required because the Spotify API nests track count
// under "tracks.total" rather than exposing it as a flat field.
type SimplePlaylist struct {
	// ID is the Spotify playlist ID.
	ID string `json:"id"`

	// Name is the display name of the playlist.
	Name string `json:"name"`

	// URI is the Spotify URI of the playlist.
	URI string `json:"uri"`

	// TrackCount is the total number of tracks in the playlist.
	// Populated from the nested "tracks.total" field in the Spotify response.
	TrackCount int `json:"-"`

	// Owner is the playlist owner.
	Owner SimplePlaylistOwner `json:"owner"`
}

// UnmarshalJSON implements custom unmarshaling to extract the nested tracks.total
// into the flat TrackCount field.
func (p *SimplePlaylist) UnmarshalJSON(data []byte) error {
	// Use a raw struct (not alias) to capture both flat fields and nested tracks.
	raw := &struct {
		ID    string              `json:"id"`
		Name  string              `json:"name"`
		URI   string              `json:"uri"`
		Owner SimplePlaylistOwner `json:"owner"`
		// Spotify renamed the "tracks" container to "items" in February 2026.
		Items struct {
			Total int `json:"total"`
		} `json:"items"`
	}{}
	if err := unmarshalJSON(data, raw); err != nil {
		return err
	}
	p.ID = raw.ID
	p.Name = raw.Name
	p.URI = raw.URI
	p.Owner = raw.Owner
	p.TrackCount = raw.Items.Total
	return nil
}

// FullAlbum represents a Spotify album with full details, used within SavedAlbum.
type FullAlbum struct {
	// ID is the Spotify album ID.
	ID string `json:"id"`

	// Name is the display name of the album.
	Name string `json:"name"`

	// URI is the Spotify URI of the album.
	URI string `json:"uri"`

	// TotalTracks is the total number of tracks in the album.
	TotalTracks int `json:"total_tracks"`

	// ReleaseDate is the release date string (e.g. "2020-03-20").
	ReleaseDate string `json:"release_date"`

	// Artists is the list of artists for this album.
	Artists []Artist `json:"artists"`
}

// SavedAlbum represents an album saved in the user's library,
// as returned by GET /me/albums.
type SavedAlbum struct {
	// AddedAt is the ISO 8601 timestamp when the album was saved.
	AddedAt string `json:"added_at"`

	// Album contains the full album details.
	Album FullAlbum `json:"album"`
}

// SavedTrack represents a track saved in the user's library,
// as returned by GET /me/tracks.
type SavedTrack struct {
	// AddedAt is the ISO 8601 timestamp when the track was saved.
	AddedAt string `json:"added_at"`

	// Track contains the full track details.
	Track Track `json:"track"`
}

// PlayHistory represents a recently played track item,
// as returned by GET /me/player/recently-played.
type PlayHistory struct {
	// Track is the track that was played.
	Track Track `json:"track"`

	// PlayedAt is the ISO 8601 timestamp when the track was played.
	PlayedAt string `json:"played_at"`
}

// QueueResponse represents the response from GET /me/player/queue.
// It contains the currently playing track and the list of queued tracks.
type QueueResponse struct {
	// CurrentlyPlaying is the track currently playing (may be zero-value if nothing is playing).
	CurrentlyPlaying Track `json:"currently_playing"`

	// Queue contains the upcoming tracks in the user's play queue.
	Queue []Track `json:"queue"`
}

// Device represents a Spotify Connect playback device.
type Device struct {
	// ID is the Spotify device ID.
	ID string `json:"id"`

	// IsActive indicates whether this device is the currently active one.
	IsActive bool `json:"is_active"`

	// IsPrivateSession indicates if the session is private.
	IsPrivateSession bool `json:"is_private_session"`

	// IsRestricted indicates if device is restricted (no web API control).
	IsRestricted bool `json:"is_restricted"`

	// Name is the human-readable device name (e.g. "MacBook Pro").
	Name string `json:"name"`

	// Type is the device type (e.g. "Computer", "Smartphone", "Speaker").
	Type string `json:"type"`

	// VolumePercent is the current volume as a percentage (0–100).
	VolumePercent int `json:"volume_percent"`
}

// PlayOffset specifies where within a context to start playback.
type PlayOffset struct {
	// URI is the Spotify track URI to start from within the context.
	URI string `json:"uri,omitempty"`
}

// PlayOptions specifies what to play.
// Provide ContextURI + Offset for collections (albums, playlists, liked songs).
// Provide URIs for an explicit ordered track list with no collection context.
type PlayOptions struct {
	// ContextURI is a Spotify URI for an album, playlist, or artist context.
	ContextURI string `json:"context_uri,omitempty"`

	// URIs is a list of Spotify track URIs to play directly.
	URIs []string `json:"uris,omitempty"`

	// Offset specifies where within the context to start playback.
	// When non-nil, playback starts at the given track URI within the context.
	Offset *PlayOffset `json:"offset,omitempty"`
}

// ArtistFollowers holds the follower count from Spotify's nested followers object.
type ArtistFollowers struct {
	Total int `json:"total"`
}

// FullArtist represents a Spotify artist with full details, as returned by
// the top-artists and search endpoints. It extends the basic Artist struct
// with genre, popularity, followers, and external URL fields.
type FullArtist struct {
	// ID is the Spotify artist ID.
	ID string `json:"id"`

	// Name is the display name of the artist.
	Name string `json:"name"`

	// URI is the Spotify URI of the artist (e.g. "spotify:artist:...").
	URI string `json:"uri"`

	// Genres is the list of musical genres associated with this artist.
	Genres []string `json:"genres"`

	// Popularity is the artist's popularity score (0–100).
	Popularity int `json:"popularity"`

	// Followers holds the total follower count.
	Followers ArtistFollowers `json:"followers"`

	// ExternalURLs maps service name to external URL (e.g. "spotify" → URL).
	ExternalURLs map[string]string `json:"external_urls"`
}
