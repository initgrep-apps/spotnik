package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/initgrep-apps/spotnik/internal/keychain"
)

// TokenProvider resolves an access token for each API request.
// This allows future implementations (e.g. RefreshableTokenProvider) to
// silently refresh the token when it expires, without restarting the app.
type TokenProvider interface {
	// AccessToken returns a valid access token for use in an Authorization header.
	// Implementations may perform I/O (e.g. token refresh) and may return an error.
	AccessToken(ctx context.Context) (string, error)
}

// StaticTokenProvider returns a fixed token. Used in tests and initial construction
// via NewBaseClient, which wraps the caller-supplied string automatically.
type StaticTokenProvider struct {
	// Token is the fixed access token returned on every call.
	Token string
}

// AccessToken returns the fixed token without any I/O. It never returns an error.
func (s *StaticTokenProvider) AccessToken(_ context.Context) (string, error) {
	return s.Token, nil
}

const refreshThreshold = 5 * time.Minute

// RefreshableTokenProvider implements TokenProvider with proactive token refresh.
// On every AccessToken() call, it checks whether the stored token is expiring
// within refreshThreshold (5 minutes). If so, it refreshes the token via the
// Spotify token endpoint before returning. This prevents 401 responses during
// normal operation — the existing 401→unauthorizedMsg→refresh flow remains as
// a safety net for edge cases (e.g. token revoked server-side).
type RefreshableTokenProvider struct {
	mu           sync.Mutex
	store        keychain.TokenStore
	clientID     string
	tokenBaseURL string
	httpClient   *http.Client
}

// NewRefreshableTokenProvider creates a provider that reads tokens from store
// and proactively refreshes them before expiry.
func NewRefreshableTokenProvider(store keychain.TokenStore, clientID, tokenBaseURL string, httpClient *http.Client) *RefreshableTokenProvider {
	return &RefreshableTokenProvider{
		store:        store,
		clientID:     clientID,
		tokenBaseURL: tokenBaseURL,
		httpClient:   httpClient,
	}
}

// AccessToken returns a valid access token, refreshing it proactively if it is
// expiring within refreshThreshold. The mutex ensures only one refresh runs at
// a time — concurrent callers block briefly and receive the freshly-refreshed token.
func (p *RefreshableTokenProvider) AccessToken(_ context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	token, err := p.store.Get(keychain.KeyAccessToken)
	if err != nil {
		return "", fmt.Errorf("reading access token: %w", err)
	}
	if token == "" {
		return "", fmt.Errorf("no access token available")
	}

	if !p.expiringSoon() {
		return token, nil
	}

	refreshToken, err := p.store.Get(keychain.KeyRefreshToken)
	if err != nil || refreshToken == "" {
		return token, nil
	}

	if err := Refresh(context.Background(), p.httpClient, p.tokenBaseURL, refreshToken, p.clientID, p.store); err != nil {
		return token, nil
	}

	newToken, err := p.store.Get(keychain.KeyAccessToken)
	if err != nil || newToken == "" {
		return token, nil
	}

	return newToken, nil
}

func (p *RefreshableTokenProvider) expiringSoon() bool {
	expiryStr, err := p.store.Get(keychain.KeyTokenExpiry)
	if err != nil {
		return true
	}
	expiryUnix, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil {
		return true
	}
	return time.Now().Add(refreshThreshold).After(time.Unix(expiryUnix, 0))
}
