// Package api provides the Spotify HTTP client, OAuth authentication flow,
// and all typed API response models. It never imports ui/ — data flows via messages and store.
//
// NOTE: Core domain types (Track, PlaybackState, Device, etc.) now live in
// internal/domain/types.go. They are re-exported here as type aliases so that
// existing code referencing api.Track, api.PlaybackState, etc. continues to work
// without changes. New code should import internal/domain directly.
package api

import (
	"github.com/initgrep-apps/spotnik/internal/domain"
)

// PlaybackState re-exports domain.PlaybackState for backward compatibility.
type PlaybackState = domain.PlaybackState

// Track re-exports domain.Track for backward compatibility.
type Track = domain.Track

// Artist re-exports domain.Artist for backward compatibility.
type Artist = domain.Artist

// Album re-exports domain.Album for backward compatibility.
type Album = domain.Album

// SimplePlaylistOwner re-exports domain.SimplePlaylistOwner for backward compatibility.
type SimplePlaylistOwner = domain.SimplePlaylistOwner

// SimplePlaylist re-exports domain.SimplePlaylist for backward compatibility.
type SimplePlaylist = domain.SimplePlaylist

// FullAlbum re-exports domain.FullAlbum for backward compatibility.
type FullAlbum = domain.FullAlbum

// SavedAlbum re-exports domain.SavedAlbum for backward compatibility.
type SavedAlbum = domain.SavedAlbum

// SavedTrack re-exports domain.SavedTrack for backward compatibility.
type SavedTrack = domain.SavedTrack

// PlayHistory re-exports domain.PlayHistory for backward compatibility.
type PlayHistory = domain.PlayHistory

// QueueResponse re-exports domain.QueueResponse for backward compatibility.
type QueueResponse = domain.QueueResponse

// Device re-exports domain.Device for backward compatibility.
type Device = domain.Device

// PlayOptions re-exports domain.PlayOptions for backward compatibility.
type PlayOptions = domain.PlayOptions

// FullArtist re-exports domain.FullArtist for backward compatibility.
type FullArtist = domain.FullArtist

// SearchArtist re-exports domain.SearchArtist for backward compatibility.
type SearchArtist = domain.SearchArtist

// SearchAlbum re-exports domain.SearchAlbum for backward compatibility.
type SearchAlbum = domain.SearchAlbum

// SearchPlaylist re-exports domain.SearchPlaylist for backward compatibility.
type SearchPlaylist = domain.SearchPlaylist

// SearchTracksResult re-exports domain.SearchTracksResult for backward compatibility.
type SearchTracksResult = domain.SearchTracksResult

// SearchArtistsResult re-exports domain.SearchArtistsResult for backward compatibility.
type SearchArtistsResult = domain.SearchArtistsResult

// SearchAlbumsResult re-exports domain.SearchAlbumsResult for backward compatibility.
type SearchAlbumsResult = domain.SearchAlbumsResult

// SearchPlaylistsResult re-exports domain.SearchPlaylistsResult for backward compatibility.
type SearchPlaylistsResult = domain.SearchPlaylistsResult

// SearchResult re-exports domain.SearchResult for backward compatibility.
// state/ should import domain.SearchResult directly rather than api.SearchResult.
type SearchResult = domain.SearchResult

// UserProfile re-exports domain.UserProfile so api/ callers can reference the type
// without importing domain/ directly.
type UserProfile = domain.UserProfile
