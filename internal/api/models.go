// Package api provides the Spotify HTTP client, OAuth authentication flow,
// and all typed API response models. It never imports ui/ — data flows via messages and store.
package api

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
