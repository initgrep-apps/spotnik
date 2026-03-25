package api

import (
	"fmt"
	"net/http"
)

// RateLimitError is returned when the Spotify API responds with 429.
type RateLimitError struct {
	RetryAfter int // seconds to wait before retrying
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited: retry after %ds", e.RetryAfter)
}

// ForbiddenError is returned when the Spotify API responds with 403.
type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("forbidden: %s", e.Message)
}

// UnauthorizedError is returned when the Spotify API responds with 401.
type UnauthorizedError struct{}

func (e *UnauthorizedError) Error() string {
	return "unauthorized: token expired or invalid"
}

// checkResponseStatus inspects an HTTP response and returns a typed error
// for known error status codes (401, 403, 429). For other non-2xx codes,
// it returns a generic error. Returns nil for success responses.
func checkResponseStatus(resp *http.Response, body []byte) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &UnauthorizedError{}

	case http.StatusForbidden:
		msg := string(body)
		if msg == "" {
			msg = "Spotify Premium required"
		}
		return &ForbiddenError{Message: msg}

	case http.StatusTooManyRequests:
		return &RateLimitError{RetryAfter: parseRetryAfter(resp)}

	default:
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
}
