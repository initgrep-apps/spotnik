package uikit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/initgrep-apps/spotnik/internal/api"
)

// Operation identifies the user-facing domain that failed. It is used by
// ErrorMapper.Map to produce an operation-specific title in the returned Toast.
type Operation string

const (
	// OpPlayback covers playback commands (play, pause, seek, next, prev, repeat, shuffle).
	OpPlayback Operation = "playback"
	// OpVolume covers volume change commands.
	OpVolume Operation = "volume"
	// OpSearch covers search page fetches.
	OpSearch Operation = "search"
	// OpQueue covers queue update operations.
	OpQueue Operation = "queue"
	// OpDevices covers device list fetches.
	OpDevices Operation = "devices"
	// OpTransfer covers device transfer/switch commands.
	OpTransfer Operation = "transfer"
	// OpStats covers stats (top tracks, top artists, recently played) fetches.
	OpStats Operation = "stats"
	// OpLibrary covers library (playlists, albums, liked tracks) fetches.
	OpLibrary Operation = "library"
	// OpPlaylists covers playlist list fetches.
	OpPlaylists Operation = "playlists"
	// OpAlbums covers album list fetches.
	OpAlbums Operation = "albums"
	// OpLikedTracks covers liked tracks fetches.
	OpLikedTracks Operation = "liked-tracks"
	// OpRecent covers recently played fetches.
	OpRecent Operation = "recently-played"
	// OpAddToQueue covers add-to-queue commands.
	OpAddToQueue Operation = "add-to-queue"
)

// opTitle maps an Operation to its sentence-case user-facing title for error toasts.
var opTitle = map[Operation]string{
	OpPlayback:    "Playback command failed",
	OpVolume:      "Volume change failed",
	OpSearch:      "Search failed",
	OpQueue:       "Queue update failed",
	OpDevices:     "Failed to load devices",
	OpTransfer:    "Device transfer failed",
	OpStats:       "Failed to load stats",
	OpLibrary:     "Failed to load library",
	OpPlaylists:   "Failed to load playlists",
	OpAlbums:      "Failed to load albums",
	OpLikedTracks: "Failed to load liked tracks",
	OpRecent:      "Failed to load recently played",
	OpAddToQueue:  "Add to queue failed",
}

// ErrorMapper turns any API-layer error into a user-friendly Toast.
// It is pure (no side effects, no store access) and safe to call from any handler.
type ErrorMapper struct{}

// Map returns a Toast with Intent, Title, and Body filled from a closed
// vocabulary of messages. The caller may override TTL or append context.
//
// Classification priority:
//  1. nil error → zero Toast (no notification needed)
//  2. api.UnauthorizedError → zero Toast (caller routes to unauthorizedMsg handler)
//  3. api.RateLimitError → ToastWarning "Rate-limited" with retry-after body
//  4. api.ForbiddenError → ToastWarning "Spotify Premium required"
//  5. context.Canceled / context.DeadlineExceeded → ToastError "Request took too long."
//  6. net.Error (timeout/temporary) or *url.Error → ToastError "Check your connection."
//  7. Everything else (5xx, generic) → ToastError "Spotify is having trouble."
//
// A zero Toast (Intent == 0) means silent drop or delegated path.
// Callers should check `toast.Intent == 0` before dispatching.
func (em *ErrorMapper) Map(op Operation, err error) Toast {
	if err == nil {
		return Toast{}
	}

	// Priority 2: UnauthorizedError — delegated to the unauthorizedMsg handler.
	var unauthorizedErr *api.UnauthorizedError
	if errors.As(err, &unauthorizedErr) {
		return Toast{}
	}

	// Priority 3: RateLimitError — advisory warning with retry-after.
	var rateLimitErr *api.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return Toast{
			Intent: ToastWarning,
			Title:  "Rate-limited",
			Body:   fmt.Sprintf("Wait %ds before retrying.", rateLimitErr.RetryAfter),
		}
	}

	// Priority 4: ForbiddenError — Premium required advisory.
	var forbiddenErr *api.ForbiddenError
	if errors.As(err, &forbiddenErr) {
		body := forbiddenErr.Message
		// Omit body when it is empty or the default "Spotify Premium required" string
		// (title already covers the same information, so the body would be redundant).
		if body == "" || body == "Spotify Premium required" {
			body = ""
		}
		return Toast{
			Intent: ToastWarning,
			Title:  "Spotify Premium required",
			Body:   body,
		}
	}

	// Priority 5: Context cancellation / deadline exceeded.
	// NOTE: context.DeadlineExceeded implements net.Error with Timeout()=true, so
	// context checks must come BEFORE the net.Error check to avoid misclassifying
	// deadline errors as network connectivity issues.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return Toast{
			Intent: ToastError,
			Title:  em.titleFor(op),
			Body:   "Request took too long. Try again.",
		}
	}

	// Priority 6: Network-level errors — connection guidance.
	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return Toast{
			Intent: ToastError,
			Title:  em.titleFor(op),
			Body:   "Check your connection.",
		}
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return Toast{
			Intent: ToastError,
			Title:  em.titleFor(op),
			Body:   "Check your connection.",
		}
	}
	// DNS errors implement net.Error but may not set Timeout/Temporary on all Go versions;
	// handle them explicitly via *net.DNSError.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return Toast{
			Intent: ToastError,
			Title:  em.titleFor(op),
			Body:   "Check your connection.",
		}
	}

	// Priority 7: Generic / 5xx — generic Spotify service body.
	return Toast{
		Intent: ToastError,
		Title:  em.titleFor(op),
		Body:   "Spotify is having trouble. Try again in a moment.",
	}
}

// titleFor returns the sentence-case title for the given operation.
// Falls back to "Operation failed" for unknown operations.
func (em *ErrorMapper) titleFor(op Operation) string {
	if t, ok := opTitle[op]; ok {
		return t
	}
	return "Operation failed"
}
