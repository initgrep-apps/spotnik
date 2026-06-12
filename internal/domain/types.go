// Package domain contains the shared data types for the Spotnik application.
// These types represent Spotify API entities and are used across the api/, state/,
// and ui/ packages without creating import cycles.
//
// Design: domain/ has no dependencies on other internal packages, allowing both
// api/ and ui/ to import it safely.
package domain

import (
	"encoding/json"
	"fmt"
)

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
	Item *Track `json:"-"`

	// Device is the currently active playback device. May be nil if no device.
	Device *Device `json:"device"`

	// CurrentlyPlayingType is the type of the currently playing item:
	// "track", "episode", "ad", or "unknown".
	CurrentlyPlayingType string `json:"currently_playing_type"`

	// Episode is the currently playing episode. Populated via custom UnmarshalJSON
	// when currently_playing_type is "episode". May be nil otherwise.
	Episode *Episode `json:"-"`

	// Context describes the playback context (album, playlist, show, etc.).
	Context *PlayContext `json:"context"`
}

// UnmarshalJSON implements custom unmarshaling to handle both track and episode
// items based on the currently_playing_type field.
func (p *PlaybackState) UnmarshalJSON(data []byte) error {
	raw := &struct {
		IsPlaying            bool             `json:"is_playing"`
		ProgressMs           int              `json:"progress_ms"`
		ShuffleState         bool             `json:"shuffle_state"`
		RepeatState          string           `json:"repeat_state"`
		Item                 json.RawMessage  `json:"item"`
		Device               *Device          `json:"device"`
		CurrentlyPlayingType string           `json:"currently_playing_type"`
		Context              *PlayContext     `json:"context"`
	}{}
	if err := unmarshalJSON(data, raw); err != nil {
		return fmt.Errorf("unmarshaling playback state: %w", err)
	}

	p.IsPlaying = raw.IsPlaying
	p.ProgressMs = raw.ProgressMs
	p.ShuffleState = raw.ShuffleState
	p.RepeatState = raw.RepeatState
	p.Device = raw.Device
	p.CurrentlyPlayingType = raw.CurrentlyPlayingType
	p.Context = raw.Context

	if raw.Item != nil && string(raw.Item) != "null" {
		if raw.CurrentlyPlayingType == "episode" {
			var ep Episode
			if err := unmarshalJSON(raw.Item, &ep); err != nil {
				return fmt.Errorf("unmarshaling episode: %w", err)
			}
			p.Episode = &ep
			p.Item = nil
		} else {
			var t Track
			if err := unmarshalJSON(raw.Item, &t); err != nil {
				return fmt.Errorf("unmarshaling track: %w", err)
			}
			p.Item = &t
			p.Episode = nil
		}
	}

	return nil
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

// AlbumImage is a single size variant of an album's cover art from Spotify.
type AlbumImage struct {
	// URL is the direct HTTPS URL to the cover art image.
	URL string `json:"url"`

	// Width is the image width in pixels.
	Width int `json:"width"`

	// Height is the image height in pixels.
	Height int `json:"height"`
}

// Album represents a simplified Spotify album (as returned within track objects).
type Album struct {
	// ID is the Spotify album ID.
	ID string `json:"id"`

	// Name is the display name of the album.
	Name string `json:"name"`

	// Images is the list of cover art size variants for this album.
	Images []AlbumImage `json:"images"`
}

// BestImage returns the smallest image where both Width and Height are >= minSize,
// falling back to the explicitly largest image. Returns nil if Images is empty.
func (a Album) BestImage(minSize int) *AlbumImage {
	var best *AlbumImage
	for i := range a.Images {
		img := &a.Images[i]
		if img.Width >= minSize && img.Height >= minSize {
			if best == nil || img.Width < best.Width {
				best = img
			}
		}
	}
	if best != nil {
		return best
	}
	// fallback: return explicitly largest
	var largest *AlbumImage
	for i := range a.Images {
		img := &a.Images[i]
		if largest == nil || img.Width > largest.Width {
			largest = img
		}
	}
	return largest
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
	// Total is the number of Spotify followers for this artist.
	Total int `json:"total"`
}

// Show represents a Spotify podcast show.
type Show struct {
	// ID is the Spotify show ID.
	ID string `json:"id"`

	// Name is the display name of the show.
	Name string `json:"name"`

	// Publisher is the publisher of the show.
	Publisher string `json:"publisher"`

	// Description is the show description in plain text.
	Description string `json:"description"`

	// TotalEpisodes is the total number of episodes in the show.
	TotalEpisodes int `json:"total_episodes"`

	// Images is the list of cover art size variants for this show.
	Images []AlbumImage `json:"images"`

	// MediaType is the type of media ("audio", "mixed", "video").
	MediaType string `json:"media_type"`

	// Explicit indicates whether the show contains explicit content.
	Explicit bool `json:"explicit"`
}

// Restrictions describes reason for content restriction.
type Restrictions struct {
	// Reason is the restriction reason code.
	Reason string `json:"reason"`
}

// ResumePoint represents the user's resume position in an episode.
type ResumePoint struct {
	// FullyPlayed indicates whether the episode has been fully played.
	FullyPlayed bool `json:"fully_played"`

	// ResumePositionMs is the position in milliseconds where playback was left off.
	ResumePositionMs int `json:"resume_position_ms"`
}

// Episode represents a Spotify podcast episode.
type Episode struct {
	// ID is the Spotify episode ID.
	ID string `json:"id"`

	// Name is the display name of the episode.
	Name string `json:"name"`

	// Description is the episode description in plain text.
	Description string `json:"description"`

	// HTMLDescription is the episode description in HTML format.
	HTMLDescription string `json:"html_description,omitempty"`

	// DurationMs is the total duration in milliseconds.
	DurationMs int `json:"duration_ms"`

	// ReleaseDate is the release date of the episode.
	ReleaseDate string `json:"release_date"`

	// Explicit indicates whether the episode contains explicit content.
	Explicit bool `json:"explicit"`

	// IsPlayable indicates whether the episode is playable.
	IsPlayable bool `json:"is_playable"`

	// IsExternallyHosted indicates whether the episode is hosted externally.
	IsExternallyHosted bool `json:"is_externally_hosted"`

	// AudioPreviewURL is a 30-second preview URL for the episode.
	AudioPreviewURL string `json:"audio_preview_url"`

	// Language is the language code of the episode.
	Language string `json:"language"`

	// URI is the Spotify URI of the episode.
	URI string `json:"uri"`

	// Show is the podcast show this episode belongs to.
	Show *Show `json:"show"`

	// ResumePoint is the user's resume position for this episode.
	ResumePoint ResumePoint `json:"resume_point"`

	// Restrictions describes content restrictions for this episode.
	Restrictions Restrictions `json:"restrictions"`
}

// SavedShow represents a show saved in the user's library.
type SavedShow struct {
	// AddedAt is the ISO 8601 timestamp when the show was saved.
	AddedAt string `json:"added_at"`

	// Show contains the full show details.
	Show Show `json:"show"`
}

// SavedEpisode represents an episode saved in the user's library.
type SavedEpisode struct {
	// AddedAt is the ISO 8601 timestamp when the episode was saved.
	AddedAt string `json:"added_at"`

	// Episode contains the full episode details.
	Episode Episode `json:"episode"`
}

// PlayContext describes the context in which playback is occurring.
type PlayContext struct {
	// Type is the context type (e.g. "album", "playlist", "show").
	Type string `json:"type"`

	// URI is the Spotify URI of the context.
	URI string `json:"uri"`
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
