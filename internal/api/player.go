package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// spotifyAPIBaseURL is the Spotify Web API base.
// Tests override this via NewPlayer/NewBaseClient baseURL parameter.
const spotifyAPIBaseURL = "https://api.spotify.com"

// Player provides all Spotify playback control API calls.
// It embeds BaseClient for shared HTTP functionality.
type Player struct {
	BaseClient
}

// NewPlayer constructs a Player using the given base URL and access token.
// Pass "" for baseURL to use the production Spotify API.
func NewPlayer(baseURL, accessToken string) *Player {
	return &Player{BaseClient: NewBaseClient(baseURL, accessToken)}
}

// SetHTTPClient overrides the default HTTP client used for API calls.
// Used to inject a LoggingTransport for network call recording.
func (p *Player) SetHTTPClient(c *http.Client) {
	p.setHTTPClient(c)
}

// PlaybackState fetches the current playback state from GET /me/player.
// Returns nil, nil when Spotify returns 204 (nothing playing).
// Returns an error on 429 or other non-2xx status codes.
// Routes through the gateway when one is attached for rate limiting and dedup.
func (p *Player) PlaybackState(ctx context.Context) (*PlaybackState, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/me/player", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get playback state request: %w", err)
	}

	var state PlaybackState
	ok, err := p.doJSONOptional(req, &state)
	if err != nil {
		return nil, fmt.Errorf("getting playback state: %w", err)
	}
	if !ok {
		// 204: nothing is playing.
		return nil, nil
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

// Queue fetches the current play queue from GET /me/player/queue.
// Returns the currently playing track and the list of upcoming tracks.
// Returns nil, nil when Spotify returns 204 (nothing playing).
// Routes through the gateway when one is attached for rate limiting and dedup.
func (p *Player) Queue(ctx context.Context) (*QueueResponse, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/v1/me/player/queue", nil)
	if err != nil {
		return nil, fmt.Errorf("creating get queue request: %w", err)
	}

	var qr QueueResponse
	ok, err := p.doJSONOptional(req, &qr)
	if err != nil {
		return nil, fmt.Errorf("getting queue: %w", err)
	}
	if !ok {
		// 204: nothing is playing.
		return nil, nil
	}
	return &qr, nil
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
