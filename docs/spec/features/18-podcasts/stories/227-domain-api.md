---
title: "Domain types + PodcastAPI client"
feature: 18-podcasts
status: open
---

## Background

The podcasts page needs shared domain types for Show and Episode entities,
search result types for shows and episodes, and a new PodcastAPI interface
with its HTTP implementation. The auth scope and player state query parameter
must also be updated to support episode data.

## Design

### Domain types (`internal/domain/types.go`)

Add after the existing `Album` / `SimplePlaylist` section:

```go
type Show struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Publisher     string       `json:"publisher"`
	Description   string       `json:"description"`
	TotalEpisodes int          `json:"total_episodes"`
	Images        []AlbumImage `json:"images"`
	MediaType     string       `json:"media_type"`    // "audio", "mixed", "video"
	Explicit      bool         `json:"explicit"`
}

type Restrictions struct {
	Reason string `json:"reason"`
}

type ResumePoint struct {
	FullyPlayed      bool `json:"fully_played"`
	ResumePositionMs int  `json:"resume_position_ms"`
}

type Episode struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	HTMLDescription   string       `json:"html_description,omitempty"`
	DurationMs        int          `json:"duration_ms"`
	ReleaseDate       string       `json:"release_date"`
	Explicit          bool         `json:"explicit"`
	IsPlayable        bool         `json:"is_playable"`
	IsExternallyHosted bool         `json:"is_externally_hosted"`
	AudioPreviewURL   string       `json:"audio_preview_url"`
	Language          string       `json:"language"`
	URI               string       `json:"uri"`
	Show              *Show        `json:"show"`
	ResumePoint       ResumePoint  `json:"resume_point"`
	Restrictions      Restrictions `json:"restrictions"`
}

type SavedShow struct {
	AddedAt string `json:"added_at"`
	Show    Show   `json:"show"`
}

type SavedEpisode struct {
	AddedAt string  `json:"added_at"`
	Episode Episode `json:"episode"`
}

type PlayContext struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
}
```

### Updated `PlaybackState` (`internal/domain/types.go`)

Add two new fields to the existing struct:

```go
type PlaybackState struct {
	IsPlaying            bool         `json:"is_playing"`
	ProgressMs           int          `json:"progress_ms"`
	ShuffleState         bool         `json:"shuffle_state"`
	RepeatState          string       `json:"repeat_state"`
	Item                 *Track       `json:"item"`
	Device               *Device      `json:"device"`
	CurrentlyPlayingType string       `json:"currently_playing_type"` // "track", "episode", "ad", "unknown"
	Episode              *Episode     `json:"-"`                       // populated via custom UnmarshalJSON
	Context              *PlayContext `json:"context"`
}
```

Add a custom `UnmarshalJSON` method on `PlaybackState` that:
1. Unmarshals JSON into an intermediate struct with `json.RawMessage` for `item`
2. Sets `CurrentlyPlayingType` from the raw field
3. If `currently_playing_type == "episode"`, unmarshal `item` into `Episode`
4. Otherwise, unmarshal `item` into `Track` (existing behaviour)

### Search types (`internal/domain/search.go`)

Add after the `SearchPlaylistsResult` struct:

```go
type SearchShow struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Publisher     string   `json:"publisher"`
	Description   string   `json:"description"`
	TotalEpisodes int      `json:"total_episodes"`
	MediaType     string   `json:"media_type"`
	Explicit      bool     `json:"explicit"`
	Languages     []string `json:"languages"`
	Images        []AlbumImage `json:"images"`
	URI           string   `json:"uri"`
}

type SearchEpisode struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	DurationMs  int      `json:"duration_ms"`
	Explicit    bool     `json:"explicit"`
	IsPlayable  bool     `json:"is_playable"`
	ReleaseDate string   `json:"release_date"`
	Images      []AlbumImage `json:"images"`
	URI         string   `json:"uri"`
}

type SearchShowsResult struct {
	Items []SearchShow `json:"items"`
	Total int          `json:"total"`
}

type SearchEpisodesResult struct {
	Items []SearchEpisode `json:"items"`
	Total int             `json:"total"`
}
```

Update `SearchResult` to add optional Shows and Episodes fields:

```go
type SearchResult struct {
	Tracks    SearchTracksResult    `json:"tracks"`
	Artists   SearchArtistsResult   `json:"artists"`
	Albums    SearchAlbumsResult    `json:"albums"`
	Playlists SearchPlaylistsResult `json:"playlists"`
	Shows     *SearchShowsResult    `json:"shows,omitempty"`
	Episodes  *SearchEpisodesResult `json:"episodes,omitempty"`
}
```

### API model aliases (`internal/api/models.go`)

```go
type Show = domain.Show
type Episode = domain.Episode
type SavedShow = domain.SavedShow
type SavedEpisode = domain.SavedEpisode
type ResumePoint = domain.ResumePoint
type PlayContext = domain.PlayContext
type SearchShow = domain.SearchShow
type SearchEpisode = domain.SearchEpisode
type SearchShowsResult = domain.SearchShowsResult
type SearchEpisodesResult = domain.SearchEpisodesResult
```

### Auth scope (`internal/api/auth.go`)

Append `user-read-playback-position` to the scope constant:

```go
const SpotifyScopes = "user-read-playback-state user-modify-playback-state " +
	"user-read-currently-playing playlist-read-private playlist-read-collaborative " +
	"playlist-modify-public playlist-modify-private user-library-read " +
	"user-library-modify user-read-private user-read-email " +
	"user-top-read user-follow-read user-read-recently-played " +
	"user-read-playback-position"
```

### Player state fetch (`internal/api/player.go`)

Add `additional_types=episode` query parameter to the playback state fetch request:

```go
q := req.URL.Query()
q.Set("market", "from_token")
q.Set("additional_types", "episode")
req.URL.RawQuery = q.Encode()
```

### PodcastAPI interface (`internal/api/podcast_interfaces.go` — new)

```go
package api

import (
	"context"
	"github.com/initgrep-apps/spotnik/internal/domain"
)

type PodcastAPI interface {
	Show(ctx context.Context, showID string) (*domain.Show, error)
	ShowEpisodes(ctx context.Context, showID string, limit, offset int) ([]domain.Episode, int, bool, error)
	FollowedShows(ctx context.Context, limit, offset int) ([]domain.SavedShow, error)
	Episode(ctx context.Context, episodeID string) (*domain.Episode, error)
	SavedEpisodes(ctx context.Context, limit, offset int) ([]domain.SavedEpisode, error)
}

var _ PodcastAPI = (*PodcastClient)(nil)
```

Response wrapper types in same file:

```go
type savedShowsResponse struct {
	Items []domain.SavedShow `json:"items"`
	Total int                `json:"total"`
}

type savedEpisodesResponse struct {
	Items []domain.SavedEpisode `json:"items"`
	Total int                   `json:"total"`
}

type showWithEpisodesResponse struct {
	domain.Show
	Episodes struct {
		Items []domain.Episode `json:"items"`
		Total int              `json:"total"`
	} `json:"episodes"`
}

type showEpisodesResponse struct {
	Items []domain.Episode `json:"items"`
	Total int              `json:"total"`
}
```

### PodcastClient (`internal/api/podcast.go` — new)

```go
package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"github.com/initgrep-apps/spotnik/internal/domain"
)

type PodcastClient struct {
	BaseClient
}

func NewPodcastClient(baseURL, accessToken string) *PodcastClient {
	return &PodcastClient{BaseClient: NewBaseClient(baseURL, accessToken)}
}

func (p *PodcastClient) Show(ctx context.Context, showID string) (*domain.Show, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/shows/"+showID, nil)
	if err != nil {
		return nil, fmt.Errorf("creating get show request: %w", err)
	}
	q := req.URL.Query()
	q.Set("market", "from_token")
	req.URL.RawQuery = q.Encode()
	var resp showWithEpisodesResponse
	if err := p.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("getting show: %w", err)
	}
	resp.Show.TotalEpisodes = resp.Episodes.Total
	return &resp.Show, nil
}

func (p *PodcastClient) ShowEpisodes(ctx context.Context, showID string, limit, offset int) ([]domain.Episode, int, bool, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/shows/"+showID+"/episodes", nil)
	if err != nil {
		return nil, 0, false, fmt.Errorf("creating get show episodes request: %w", err)
	}
	q := req.URL.Query()
	q.Set("market", "from_token")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = q.Encode()
	var resp showEpisodesResponse
	if err := p.doJSON(req, &resp); err != nil {
		return nil, 0, false, fmt.Errorf("getting show episodes: %w", err)
	}
	hasNext := offset+limit < resp.Total
	return resp.Items, resp.Total, hasNext, nil
}

func (p *PodcastClient) FollowedShows(ctx context.Context, limit, offset int) ([]domain.SavedShow, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/me/shows", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get followed shows request: %w", err)
	}
	q := req.URL.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = q.Encode()
	var resp savedShowsResponse
	if err := p.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("getting followed shows: %w", err)
	}
	return resp.Items, nil
}

func (p *PodcastClient) Episode(ctx context.Context, episodeID string) (*domain.Episode, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/episodes/"+episodeID, nil)
	if err != nil {
		return nil, fmt.Errorf("creating get episode request: %w", err)
	}
	q := req.URL.Query()
	q.Set("market", "from_token")
	req.URL.RawQuery = q.Encode()
	var episode domain.Episode
	if err := p.doJSON(req, &episode); err != nil {
		return nil, fmt.Errorf("getting episode: %w", err)
	}
	return &episode, nil
}

func (p *PodcastClient) SavedEpisodes(ctx context.Context, limit, offset int) ([]domain.SavedEpisode, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/me/episodes", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get saved episodes request: %w", err)
	}
	q := req.URL.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = q.Encode()
	var resp savedEpisodesResponse
	if err := p.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("getting saved episodes: %w", err)
	}
	return resp.Items, nil
}
```

Note: `BaseClient.doJSON` performs the actual HTTP call. `PodcastClient` uses `BaseClient.newRequest` and `BaseClient.doJSON`/`doNoContent` which already route through the Gateway when attached — no explicit Gateway awareness needed in this client.

## Acceptance Criteria

- [ ] `domain.Show`, `domain.Episode`, `domain.ResumePoint`, `domain.Restrictions`, `domain.SavedShow`, `domain.SavedEpisode`, `domain.PlayContext` compile
- [ ] `PlaybackState` has new `CurrentlyPlayingType`, `Episode`, `Context` fields; custom `UnmarshalJSON` populates `Item` (for tracks) or `Episode` (for episodes)
- [ ] `SearchShow`, `SearchEpisode`, `SearchShowsResult`, `SearchEpisodesResult` compile; `SearchResult` has optional `Shows` and `Episodes` fields
- [ ] API type aliases (`api.Show`, `api.Episode`, etc.) compile
- [ ] `SpotifyScopes` includes `user-read-playback-position`
- [ ] Player state fetch sends `additional_types=episode` query param
- [ ] `PodcastAPI` interface compiles; `PodcastClient` satisfies it via compile-time check
- [ ] `PodcastClient.Show`, `.ShowEpisodes`, `.FollowedShows`, `.Episode`, `.SavedEpisodes` compile
- [ ] `go test ./internal/domain/... ./internal/api/...` passes

## Tasks

- [ ] Add domain types (`Show`, `Episode`, `ResumePoint`, `Restrictions`, `SavedShow`, `SavedEpisode`, `PlayContext`) to `internal/domain/types.go`
- [ ] Add tests: `TestShow_Fields`, `TestEpisode_Fields`, `TestResumePoint_Fields`, `TestRestrictions_Fields` — verify field access and zero values — in `internal/domain/types_test.go`
- [ ] Add custom `UnmarshalJSON` to `PlaybackState` to handle both track and episode items
- [ ] Add `CurrentlyPlayingType`, `Episode`, `Context` fields to existing `PlaybackState` struct
- [ ] Add tests: `TestPlaybackState_UnmarshalTrack`, `TestPlaybackState_UnmarshalEpisode` — verify correct field population based on `currently_playing_type` — in `internal/domain/types_test.go`
- [ ] Add search types to `internal/domain/search.go`; update `SearchResult` with optional Shows/Episodes
- [ ] Add tests: verify `SearchResult` can unmarshal JSON with `shows` and `episodes` fields — in `internal/domain/search_test.go`
- [ ] Add type aliases to `internal/api/models.go`
- [ ] Append `user-read-playback-position` to `SpotifyScopes` in `internal/api/auth.go`
- [ ] Add test: verify `SpotifyScopes` contains `"user-read-playback-position"` — in `internal/api/auth_test.go`
- [ ] Add `additional_types=episode` query param to player state fetch in `internal/api/player.go`
- [ ] Add test: verify `additional_types=episode` appears in the request URL — in `internal/api/player_test.go`
- [ ] Create `internal/api/podcast_interfaces.go` with `PodcastAPI` interface + response wrappers
- [ ] Add compile-time check: `var _ PodcastAPI = (*PodcastClient)(nil)` in `podcast_interfaces.go`
- [ ] Create `internal/api/podcast.go` with `PodcastClient` implementation
- [ ] Add table-driven tests for `PodcastClient` (Show, ShowEpisodes, FollowedShows, Episode, SavedEpisodes) using `httptest.NewServer` and JSON fixtures from `testdata/fixtures/` — in `internal/api/podcast_test.go`
- [ ] Run `go build ./...` and verify compilation
- [ ] Run `go test ./internal/domain/... ./internal/api/...` — all pass
