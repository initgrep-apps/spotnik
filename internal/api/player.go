package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// spotifyAPIBaseURL is the Spotify Web API base.
// Tests override this via Player.baseURL.
const spotifyAPIBaseURL = "https://api.spotify.com"

// PlayOptions specifies what to play. Provide ContextURI for albums/playlists,
// or URIs for specific tracks.
type PlayOptions struct {
	// ContextURI is a Spotify URI for an album, playlist, or artist context.
	ContextURI string `json:"context_uri,omitempty"`

	// URIs is a list of Spotify track URIs to play directly.
	URIs []string `json:"uris,omitempty"`
}

// Player provides all Spotify playback control API calls.
// It holds the base URL (overridable for tests) and the Bearer access token.
type Player struct {
	baseURL     string
	accessToken string
	client      *http.Client
}

// NewPlayer constructs a Player using the given base URL and access token.
// Pass "" for baseURL to use the production Spotify API.
func NewPlayer(baseURL, accessToken string) *Player {
	base := baseURL
	if base == "" {
		base = spotifyAPIBaseURL
	}
	return &Player{
		baseURL:     strings.TrimRight(base, "/"),
		accessToken: accessToken,
		client:      &http.Client{},
	}
}

// GetPlaybackState fetches the current playback state from GET /me/player.
// Returns nil, nil when Spotify returns 204 (nothing playing).
// Returns an error on 429 or other non-2xx status codes.
func (p *Player) GetPlaybackState(ctx context.Context) (*PlaybackState, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/me/player", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get playback state request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting playback state: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNoContent {
		// 204: nothing is playing.
		return nil, nil
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		return nil, fmt.Errorf("429 rate limited: retry after %s seconds", retryAfter)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get playback state: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading playback state response: %w", err)
	}

	var state PlaybackState
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, fmt.Errorf("parsing playback state: %w", err)
	}

	return &state, nil
}

// Play starts or resumes playback using the given PlayOptions.
// Sends PUT /me/player/play.
func (p *Player) Play(ctx context.Context, opts PlayOptions) error {
	body, err := json.Marshal(opts)
	if err != nil {
		return fmt.Errorf("marshaling play options: %w", err)
	}

	req, err := p.newRequest(ctx, http.MethodPut, "/v1/me/player/play", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating play request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return p.doNoContent(req)
}

// Pause pauses playback via PUT /me/player/pause.
func (p *Player) Pause(ctx context.Context) error {
	req, err := p.newRequest(ctx, http.MethodPut, "/v1/me/player/pause", nil)
	if err != nil {
		return fmt.Errorf("creating pause request: %w", err)
	}
	return p.doNoContent(req)
}

// Next skips to the next track via POST /me/player/next.
func (p *Player) Next(ctx context.Context) error {
	req, err := p.newRequest(ctx, http.MethodPost, "/v1/me/player/next", nil)
	if err != nil {
		return fmt.Errorf("creating next request: %w", err)
	}
	return p.doNoContent(req)
}

// Previous skips to the previous track via POST /me/player/previous.
func (p *Player) Previous(ctx context.Context) error {
	req, err := p.newRequest(ctx, http.MethodPost, "/v1/me/player/previous", nil)
	if err != nil {
		return fmt.Errorf("creating previous request: %w", err)
	}
	return p.doNoContent(req)
}

// Seek moves playback to positionMs via PUT /me/player/seek?position_ms=<ms>.
func (p *Player) Seek(ctx context.Context, positionMs int) error {
	req, err := p.newRequest(ctx, http.MethodPut, "/v1/me/player/seek", nil)
	if err != nil {
		return fmt.Errorf("creating seek request: %w", err)
	}

	q := req.URL.Query()
	q.Set("position_ms", strconv.Itoa(positionMs))
	req.URL.RawQuery = q.Encode()

	return p.doNoContent(req)
}

// SetVolume sets the volume via PUT /me/player/volume?volume_percent=<vol>.
// Volume is clamped to [0, 100] before sending.
func (p *Player) SetVolume(ctx context.Context, volume int) error {
	if volume > 100 {
		volume = 100
	}
	if volume < 0 {
		volume = 0
	}

	req, err := p.newRequest(ctx, http.MethodPut, "/v1/me/player/volume", nil)
	if err != nil {
		return fmt.Errorf("creating set volume request: %w", err)
	}

	q := req.URL.Query()
	q.Set("volume_percent", strconv.Itoa(volume))
	req.URL.RawQuery = q.Encode()

	return p.doNoContent(req)
}

// SetShuffle enables or disables shuffle via PUT /me/player/shuffle?state=<bool>.
func (p *Player) SetShuffle(ctx context.Context, state bool) error {
	req, err := p.newRequest(ctx, http.MethodPut, "/v1/me/player/shuffle", nil)
	if err != nil {
		return fmt.Errorf("creating set shuffle request: %w", err)
	}

	q := req.URL.Query()
	q.Set("state", strconv.FormatBool(state))
	req.URL.RawQuery = q.Encode()

	return p.doNoContent(req)
}

// AddToQueue adds a track to the user's playback queue via POST /me/player/queue?uri=<uri>.
// trackURI must be a Spotify track URI (e.g. "spotify:track:...").
func (p *Player) AddToQueue(ctx context.Context, trackURI string) error {
	req, err := p.newRequest(ctx, http.MethodPost, "/v1/me/player/queue", nil)
	if err != nil {
		return fmt.Errorf("creating add to queue request: %w", err)
	}

	q := req.URL.Query()
	q.Set("uri", trackURI)
	req.URL.RawQuery = q.Encode()

	return p.doNoContent(req)
}

// SetRepeat sets the repeat mode via PUT /me/player/repeat?state=<mode>.
// mode must be one of "off", "context", or "track".
func (p *Player) SetRepeat(ctx context.Context, mode string) error {
	switch mode {
	case "off", "context", "track":
		// valid
	default:
		return fmt.Errorf("invalid repeat mode %q: must be off, context, or track", mode)
	}

	req, err := p.newRequest(ctx, http.MethodPut, "/v1/me/player/repeat", nil)
	if err != nil {
		return fmt.Errorf("creating set repeat request: %w", err)
	}

	q := req.URL.Query()
	q.Set("state", mode)
	req.URL.RawQuery = q.Encode()

	return p.doNoContent(req)
}

// newRequest builds an HTTP request with the Authorization header set.
func (p *Player) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	u := p.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	return req, nil
}

// doNoContent executes req and returns nil if the response is 2xx.
// Returns an error for non-2xx responses.
func (p *Player) doNoContent(req *http.Request) error {
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: unexpected status %d: %s", req.Method, req.URL.Path, resp.StatusCode, string(body))
	}

	return nil
}
