package api

import "context"

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
