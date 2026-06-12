package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchArtist_UnmarshalJSON verifies that Genres, Followers, and Popularity
// are correctly extracted from the nested API JSON structure.
func TestSearchArtist_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		json           string
		wantGenres     []string
		wantFollowers  int
		wantPopularity int
		wantErr        bool
	}{
		{
			name: "full artist with all fields",
			json: `{
				"id": "a1",
				"name": "Hans Zimmer",
				"uri": "spotify:artist:a1",
				"genres": ["film score", "soundtrack"],
				"popularity": 79,
				"followers": {"href": null, "total": 7625607}
			}`,
			wantGenres:     []string{"film score", "soundtrack"},
			wantFollowers:  7625607,
			wantPopularity: 79,
		},
		{
			name: "empty genres returns empty slice",
			json: `{
				"id": "a2",
				"name": "New Artist",
				"uri": "spotify:artist:a2",
				"genres": [],
				"popularity": 20,
				"followers": {"total": 0}
			}`,
			wantGenres:     []string{},
			wantFollowers:  0,
			wantPopularity: 20,
		},
		{
			name: "missing genres field returns nil",
			json: `{
				"id": "a3",
				"name": "No Genre",
				"uri": "spotify:artist:a3",
				"popularity": 50,
				"followers": {"total": 1000}
			}`,
			wantGenres:     nil,
			wantFollowers:  1000,
			wantPopularity: 50,
		},
		{
			name:    "invalid json returns error",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a SearchArtist
			err := json.Unmarshal([]byte(tt.json), &a)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantGenres, a.Genres)
			assert.Equal(t, tt.wantFollowers, a.Followers)
			assert.Equal(t, tt.wantPopularity, a.Popularity)
		})
	}
}

// TestSearchAlbum_AlbumType verifies that the AlbumType field unmarshals from album_type.
func TestSearchAlbum_AlbumType(t *testing.T) {
	tests := []struct {
		name          string
		json          string
		wantAlbumType string
	}{
		{
			name: "album type compilation",
			json: `{
				"id": "al1",
				"name": "Greatest Hits",
				"uri": "spotify:album:al1",
				"album_type": "compilation",
				"total_tracks": 20,
				"release_date": "2005-06-15",
				"artists": []
			}`,
			wantAlbumType: "compilation",
		},
		{
			name: "album type single",
			json: `{
				"id": "al2",
				"name": "New Single",
				"uri": "spotify:album:al2",
				"album_type": "single",
				"total_tracks": 1,
				"release_date": "2023-01-10",
				"artists": []
			}`,
			wantAlbumType: "single",
		},
		{
			name: "album type album",
			json: `{
				"id": "al3",
				"name": "Full Album",
				"uri": "spotify:album:al3",
				"album_type": "album",
				"total_tracks": 12,
				"release_date": "2021-05-20",
				"artists": []
			}`,
			wantAlbumType: "album",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var al SearchAlbum
			require.NoError(t, json.Unmarshal([]byte(tt.json), &al))
			assert.Equal(t, tt.wantAlbumType, al.AlbumType)
		})
	}
}

// TestSearchPlaylist_Description verifies that the Description field is extracted
// from the JSON response.
// TestSearchResult_UnmarshalWithShowsEpisodes verifies that SearchResult can unmarshal
// JSON with shows and episodes fields.
func TestSearchResult_UnmarshalWithShowsEpisodes(t *testing.T) {
	raw := `{
		"tracks": {"items": [], "total": 0},
		"artists": {"items": [], "total": 0},
		"albums": {"items": [], "total": 0},
		"playlists": {"items": [], "total": 0},
		"shows": {
			"items": [
				{
					"id": "show-1",
					"name": "Test Show",
					"publisher": "Test Publisher",
					"description": "A test show",
					"total_episodes": 42,
					"media_type": "audio",
					"explicit": false,
					"languages": ["en"],
					"images": [],
					"uri": "spotify:show:show-1"
				}
			],
			"total": 1
		},
		"episodes": {
			"items": [
				{
					"id": "ep-1",
					"name": "Test Episode",
					"description": "An episode",
					"duration_ms": 1800000,
					"explicit": false,
					"is_playable": true,
					"release_date": "2024-01-15",
					"images": [],
					"uri": "spotify:episode:ep-1"
				}
			],
			"total": 1
		}
	}`

	var sr SearchResult
	err := json.Unmarshal([]byte(raw), &sr)
	require.NoError(t, err)

	require.NotNil(t, sr.Shows)
	assert.Equal(t, 1, sr.Shows.Total)
	require.Len(t, sr.Shows.Items, 1)
	assert.Equal(t, "show-1", sr.Shows.Items[0].ID)
	assert.Equal(t, "Test Show", sr.Shows.Items[0].Name)
	assert.Equal(t, "Test Publisher", sr.Shows.Items[0].Publisher)

	require.NotNil(t, sr.Episodes)
	assert.Equal(t, 1, sr.Episodes.Total)
	require.Len(t, sr.Episodes.Items, 1)
	assert.Equal(t, "ep-1", sr.Episodes.Items[0].ID)
	assert.Equal(t, "Test Episode", sr.Episodes.Items[0].Name)
}

// TestSearchResult_UnmarshalWithoutShowsEpisodes verifies that SearchResult still works
// when shows and episodes fields are absent.
func TestSearchResult_UnmarshalWithoutShowsEpisodes(t *testing.T) {
	raw := `{
		"tracks": {"items": [], "total": 0},
		"artists": {"items": [], "total": 0},
		"albums": {"items": [], "total": 0},
		"playlists": {"items": [], "total": 0}
	}`

	var sr SearchResult
	err := json.Unmarshal([]byte(raw), &sr)
	require.NoError(t, err)
	assert.Nil(t, sr.Shows)
	assert.Nil(t, sr.Episodes)
}

func TestSearchPlaylist_Description(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantDesc string
	}{
		{
			name: "playlist with description",
			json: `{
				"id": "pl1",
				"name": "Rock Classics",
				"uri": "spotify:playlist:pl1",
				"owner": {"id": "user1", "display_name": "Alice"},
				"tracks": {"total": 45},
				"description": "My cool playlist"
			}`,
			wantDesc: "My cool playlist",
		},
		{
			name: "playlist with empty description",
			json: `{
				"id": "pl2",
				"name": "Empty Desc",
				"uri": "spotify:playlist:pl2",
				"owner": {"id": "user2", "display_name": "Bob"},
				"tracks": {"total": 10},
				"description": ""
			}`,
			wantDesc: "",
		},
		{
			name: "playlist without description field",
			json: `{
				"id": "pl3",
				"name": "No Desc",
				"uri": "spotify:playlist:pl3",
				"owner": {"id": "user3", "display_name": "Carol"},
				"tracks": {"total": 5}
			}`,
			wantDesc: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p SearchPlaylist
			require.NoError(t, json.Unmarshal([]byte(tt.json), &p))
			assert.Equal(t, tt.wantDesc, p.Description)
		})
	}
}
