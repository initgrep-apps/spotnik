package uikit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/initgrep-apps/spotnik/internal/api"
)

// RecoveryHint is a reusable user-facing recovery instruction for error toasts.
// These are referenced by handlers so the vocabulary stays consistent across
// the application (spec §3.15).
type RecoveryHint string

const (
	// RecoveryCheckConnection is shown for network-level failures.
	RecoveryCheckConnection RecoveryHint = "Check your connection."
	// RecoveryRetryInMoment is the generic fallback for Spotify service errors.
	RecoveryRetryInMoment RecoveryHint = "Spotify is having trouble. Try again in a moment."
	// RecoveryPressEnterRetry is shown for playlist/album track sub-view failures.
	RecoveryPressEnterRetry RecoveryHint = "Press Enter to retry."
	// RecoveryRunAuth is shown when the session has expired.
	RecoveryRunAuth RecoveryHint = "Run: spotnik auth."
)

// Operation identifies the user-facing domain that failed. It is used by
// ErrorMapper.Map to produce an operation-specific title in the returned Toast.
type Operation string

const (
	// OpPlayback covers playback commands (play, pause, seek, next, prev, repeat, shuffle).
	OpPlayback Operation = "playback"
	// OpVolume covers volume change commands.
	OpVolume Operation = "volume"
	// OpSeek covers seek (position) commands.
	OpSeek Operation = "seek"
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
	// OpPlaylistTracks covers playlist track list fetches.
	OpPlaylistTracks Operation = "playlist-tracks"
)

// opTitle maps an Operation to its sentence-case user-facing title for error toasts.
var opTitle = map[Operation]string{
	OpPlayback:       "Playback command failed",
	OpVolume:         "Volume change failed",
	OpSeek:           "Seek failed",
	OpSearch:         "Search failed",
	OpQueue:          "Queue update failed",
	OpDevices:        "Failed to load devices",
	OpTransfer:       "Device transfer failed",
	OpStats:          "Failed to load stats",
	OpLibrary:        "Failed to load library",
	OpPlaylists:      "Failed to load playlists",
	OpAlbums:         "Failed to load albums",
	OpLikedTracks:    "Failed to load liked tracks",
	OpRecent:         "Failed to load recently played",
	OpAddToQueue:     "Add to queue failed",
	OpPlaylistTracks: "Failed to load playlist tracks",
}

// opForbiddenBody maps an Operation to its 403-specific recovery hint.
// OpQueue and OpAddToQueue return a "no active device" hint because Spotify
// returns 403 for queue actions when no device is active, not for Premium.
var opForbiddenBody = map[Operation]string{
	OpPlayback:       "Premium required for this action.",
	OpVolume:         "Premium required for volume control.",
	OpSeek:           "Premium required for seek control.",
	OpSearch:         "Premium required for search.",
	OpQueue:          "No active device. Open Spotify first.",
	OpAddToQueue:     "No active device. Open Spotify first.",
	OpTransfer:       "Premium required for device control.",
	OpAlbums:         "Premium required to view album tracks.",
	OpPlaylistTracks: "No permission to view this playlist.",
}

// ErrorMapper turns any API-layer error into a user-friendly Toast.
// It is pure (no side effects, no store access) and safe to call from any handler.
type ErrorMapper struct{}

// Map returns a Toast with Intent, Title, and Body filled from a closed
// vocabulary of messages. The caller may override TTL or append context.
//
// Classification priority:
//  1. nil error → ToastNone (no notification needed)
//  2. api.UnauthorizedError → ToastNone (caller routes to unauthorizedMsg handler)
//  3. api.RateLimitError → ToastWarning "Rate-limited" with retry-after body
//  4. api.ForbiddenError → ToastWarning with operation-specific title and body
//  5. context.Canceled / context.DeadlineExceeded → ToastError "Request took too long."
//  6. net.Error (timeout) or *url.Error / *net.DNSError → ToastError "Check your connection."
//  7. Everything else (5xx, generic) → ToastError "Spotify is having trouble."
//
// A Toast with Intent == ToastNone means silent drop or delegated path.
// Callers should check `toast.Intent == ToastNone` before dispatching.
func (em *ErrorMapper) Map(op Operation, err error) Toast {
	if err == nil {
		return Toast{Intent: ToastNone}
	}

	// Priority 2: UnauthorizedError — delegated to the unauthorizedMsg handler.
	var unauthorizedErr *api.UnauthorizedError
	if errors.As(err, &unauthorizedErr) {
		return Toast{Intent: ToastNone}
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

	// Priority 4: ForbiddenError — operation-specific advisory.
	var forbiddenErr *api.ForbiddenError
	if errors.As(err, &forbiddenErr) {
		body := forbiddenErr.Message
		if body == "" || body == "Spotify Premium required" {
			body = em.forbiddenBodyFor(op)
		}
		return Toast{
			Intent: ToastWarning,
			Title:  em.titleFor(op),
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
	// NOTE: net.Error.Temporary() is deprecated since Go 1.18; use Timeout() only.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return Toast{
			Intent: ToastError,
			Title:  em.titleFor(op),
			Body:   string(RecoveryCheckConnection),
		}
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return Toast{
			Intent: ToastError,
			Title:  em.titleFor(op),
			Body:   string(RecoveryCheckConnection),
		}
	}
	// DNS errors implement net.Error but may not set Timeout/Temporary on all Go versions;
	// handle them explicitly via *net.DNSError.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return Toast{
			Intent: ToastError,
			Title:  em.titleFor(op),
			Body:   string(RecoveryCheckConnection),
		}
	}

	// Priority 7: Generic / 5xx — generic Spotify service body.
	return Toast{
		Intent: ToastError,
		Title:  em.titleFor(op),
		Body:   string(RecoveryRetryInMoment),
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

// forbiddenBodyFor returns the operation-specific body text for a 403 response.
// Falls back to the generic Premium subscription message for operations not listed
// in opForbiddenBody.
func (em *ErrorMapper) forbiddenBodyFor(op Operation) string {
	if b, ok := opForbiddenBody[op]; ok {
		return b
	}
	return "A Premium subscription is required for this feature."
}
