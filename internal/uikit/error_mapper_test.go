package uikit_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorMapper_NilError verifies Map returns a zero Toast for a nil error.
func TestErrorMapper_NilError(t *testing.T) {
	em := &uikit.ErrorMapper{}
	toast := em.Map(uikit.OpSearch, nil)
	assert.Equal(t, uikit.ToastNone, toast.Intent, "nil error must return ToastNone")
	assert.Empty(t, toast.Title)
	assert.Empty(t, toast.Body)
}

// TestErrorMapper_UnauthorizedError verifies Map returns a zero Toast for
// UnauthorizedError so the caller can route to the unauthorizedMsg handler.
func TestErrorMapper_UnauthorizedError(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.UnauthorizedError{}
	toast := em.Map(uikit.OpPlayback, err)
	assert.Equal(t, uikit.ToastNone, toast.Intent, "UnauthorizedError must return ToastNone")
	assert.Empty(t, toast.Title)
	assert.Empty(t, toast.Body)
}

// TestErrorMapper_RateLimitError verifies Map returns a ToastWarning with
// "Rate-limited" title and the correct retry-after body.
func TestErrorMapper_RateLimitError(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.RateLimitError{RetryAfter: 15}
	toast := em.Map(uikit.OpSearch, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "Rate-limited", toast.Title)
	assert.Equal(t, "Wait 15s before retrying.", toast.Body)
}

// TestErrorMapper_RateLimitError_WrappedError verifies the rate-limit check
// works when the error is wrapped.
func TestErrorMapper_RateLimitError_WrappedError(t *testing.T) {
	em := &uikit.ErrorMapper{}
	inner := &api.RateLimitError{RetryAfter: 30}
	err := fmt.Errorf("gateway: %w", inner)
	toast := em.Map(uikit.OpDevices, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "Rate-limited", toast.Title)
	assert.Equal(t, "Wait 30s before retrying.", toast.Body)
}

// TestErrorMapper_ForbiddenError_WithMessage verifies Map returns ToastWarning
// with the operation-specific title and the server-supplied message as the body.
func TestErrorMapper_ForbiddenError_WithMessage(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: "Premium required to use this endpoint."}
	toast := em.Map(uikit.OpVolume, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "Volume change failed", toast.Title)
	assert.Equal(t, "Premium required to use this endpoint.", toast.Body)
}

// TestErrorMapper_ForbiddenError_DefaultMessage verifies that when the
// ForbiddenError message is the default "Spotify Premium required" string,
// the operation-specific body is substituted so the toast is never empty.
func TestErrorMapper_ForbiddenError_DefaultMessage(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: "Spotify Premium required"}
	toast := em.Map(uikit.OpPlayback, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "Playback command failed", toast.Title)
	assert.Equal(t, "Premium required for this action.", toast.Body)
}

// TestErrorMapper_ForbiddenError_EmptyMessage verifies that an empty
// ForbiddenError.Message receives the operation-specific recovery body.
func TestErrorMapper_ForbiddenError_EmptyMessage(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: ""}
	toast := em.Map(uikit.OpQueue, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "Queue update failed", toast.Title)
	assert.Equal(t, "No active device. Open Spotify first.", toast.Body)
}

// TestErrorMapper_NetworkTimeout verifies that a net.Error with Timeout()=true
// produces a connection-check body.
func TestErrorMapper_NetworkTimeout(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &mockNetError{timeout: true}
	toast := em.Map(uikit.OpSearch, err)
	require.Equal(t, uikit.ToastError, toast.Intent)
	assert.Equal(t, "Search failed", toast.Title)
	assert.Equal(t, "Check your connection.", toast.Body)
}

// TestErrorMapper_URLError verifies that a *url.Error produces a connection-check body.
func TestErrorMapper_URLError(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &url.Error{Op: "Get", URL: "https://api.spotify.com", Err: fmt.Errorf("connection refused")}
	toast := em.Map(uikit.OpDevices, err)
	require.Equal(t, uikit.ToastError, toast.Intent)
	assert.Equal(t, "Failed to load devices", toast.Title)
	assert.Equal(t, "Check your connection.", toast.Body)
}

// TestErrorMapper_ContextCanceled verifies that context.Canceled produces a
// "Request took too long" body.
func TestErrorMapper_ContextCanceled(t *testing.T) {
	em := &uikit.ErrorMapper{}
	toast := em.Map(uikit.OpSearch, context.Canceled)
	require.Equal(t, uikit.ToastError, toast.Intent)
	assert.Equal(t, "Search failed", toast.Title)
	assert.Equal(t, "Request took too long. Try again.", toast.Body)
}

// TestErrorMapper_ContextDeadlineExceeded verifies that context.DeadlineExceeded
// produces a "Request took too long" body.
func TestErrorMapper_ContextDeadlineExceeded(t *testing.T) {
	em := &uikit.ErrorMapper{}
	toast := em.Map(uikit.OpPlayback, context.DeadlineExceeded)
	require.Equal(t, uikit.ToastError, toast.Intent)
	assert.Equal(t, "Playback command failed", toast.Title)
	assert.Equal(t, "Request took too long. Try again.", toast.Body)
}

// TestErrorMapper_GenericError verifies that a generic error (e.g. fmt.Errorf)
// produces the Spotify-trouble body — never exposes raw err.Error().
func TestErrorMapper_GenericError(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := fmt.Errorf("unexpected status 500: internal server error")
	toast := em.Map(uikit.OpQueue, err)
	require.Equal(t, uikit.ToastError, toast.Intent)
	assert.Equal(t, "Queue update failed", toast.Title)
	assert.Equal(t, "Spotify is having trouble. Try again in a moment.", toast.Body)
	// Confirm body never contains the raw error string.
	assert.NotContains(t, toast.Body, err.Error())
}

// TestErrorMapper_NeverExposesRawError ensures that no matter what error
// is passed, the Body never equals err.Error().
func TestErrorMapper_NeverExposesRawError(t *testing.T) {
	em := &uikit.ErrorMapper{}
	rawErrors := []error{
		fmt.Errorf("unexpected status 500: some internal message"),
		fmt.Errorf("sending request: dial tcp connection refused"),
		errors.New("some random internal error"),
	}
	for _, err := range rawErrors {
		toast := em.Map(uikit.OpStats, err)
		if toast.Intent == uikit.ToastNone {
			continue // silent drop — acceptable
		}
		assert.NotEqual(t, err.Error(), toast.Body,
			"Body must never equal raw err.Error() for %v", err)
	}
}

// TestErrorMapper_OperationTitles verifies every Operation produces the correct
// title string for a generic error.
func TestErrorMapper_OperationTitles(t *testing.T) {
	tests := []struct {
		op        uikit.Operation
		wantTitle string
	}{
		{uikit.OpPlayback, "Playback command failed"},
		{uikit.OpVolume, "Volume change failed"},
		{uikit.OpSearch, "Search failed"},
		{uikit.OpDevices, "Failed to load devices"},
		{uikit.OpTransfer, "Device transfer failed"},
		{uikit.OpAddToQueue, "Add to queue failed"},
		{uikit.OpQueue, "Queue update failed"},
		{uikit.OpStats, "Failed to load stats"},
		{uikit.OpLibrary, "Failed to load library"},
		{uikit.OpPlaylists, "Failed to load playlists"},
		{uikit.OpAlbums, "Failed to load albums"},
		{uikit.OpLikedTracks, "Failed to load liked tracks"},
		{uikit.OpRecent, "Failed to load recently played"},
		{uikit.OpPlaylistTracks, "Failed to load playlist tracks"},
	}
	em := &uikit.ErrorMapper{}
	genericErr := fmt.Errorf("unexpected status 503: service unavailable")
	for _, tt := range tests {
		t.Run(string(tt.op), func(t *testing.T) {
			toast := em.Map(tt.op, genericErr)
			require.Equal(t, uikit.ToastError, toast.Intent)
			assert.Equal(t, tt.wantTitle, toast.Title)
		})
	}
}

// TestErrorMapper_WrappedContextCanceled verifies wrapped context.Canceled is
// classified as the "too long" bucket.
func TestErrorMapper_WrappedContextCanceled(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := fmt.Errorf("fetching devices: %w", context.Canceled)
	toast := em.Map(uikit.OpDevices, err)
	require.Equal(t, uikit.ToastError, toast.Intent)
	assert.Equal(t, "Request took too long. Try again.", toast.Body)
}

// TestErrorMapper_DNSError verifies a DNS-level net.Error also maps to the
// connection-check body.
func TestErrorMapper_DNSError(t *testing.T) {
	em := &uikit.ErrorMapper{}
	dnsErr := &net.DNSError{Err: "no such host", Name: "api.spotify.com", IsNotFound: true}
	toast := em.Map(uikit.OpSearch, dnsErr)
	require.Equal(t, uikit.ToastError, toast.Intent)
	assert.Equal(t, "Check your connection.", toast.Body)
}

// mockNetError implements net.Error for test purposes.
type mockNetError struct {
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return "mock net error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

// ---------------------------------------------------------------------------
// Story 201: per-operation 403 bodies
// ---------------------------------------------------------------------------

// TestErrorMapper_Forbidden_PlaybackBody verifies that OpPlayback returns
// a Premium-specific body for 403 errors.
func TestErrorMapper_Forbidden_PlaybackBody(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: ""}
	toast := em.Map(uikit.OpPlayback, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "Premium required for this action.", toast.Body)
}

// TestErrorMapper_Forbidden_QueueBody verifies that OpQueue returns
// a "no active device" body for 403 errors (Spotify returns 403 for
// queue actions when no device is active, not for Premium).
func TestErrorMapper_Forbidden_QueueBody(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: ""}
	toast := em.Map(uikit.OpQueue, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "No active device. Open Spotify first.", toast.Body)
}

// TestErrorMapper_Forbidden_AddToQueueBody verifies that OpAddToQueue returns
// a "no active device" body for 403 errors.
func TestErrorMapper_Forbidden_AddToQueueBody(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: ""}
	toast := em.Map(uikit.OpAddToQueue, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "No active device. Open Spotify first.", toast.Body)
}

// TestErrorMapper_Forbidden_TransferBody verifies that OpTransfer returns
// a Premium-specific body for 403 errors.
func TestErrorMapper_Forbidden_TransferBody(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: ""}
	toast := em.Map(uikit.OpTransfer, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "Premium required for device control.", toast.Body)
}

// TestErrorMapper_Forbidden_VolumeBody verifies that OpVolume returns
// a Premium-specific body for 403 errors.
func TestErrorMapper_Forbidden_VolumeBody(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: ""}
	toast := em.Map(uikit.OpVolume, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "Premium required for volume control.", toast.Body)
}

// TestErrorMapper_Forbidden_SearchBody verifies that OpSearch returns
// a Premium-specific body for 403 errors.
func TestErrorMapper_Forbidden_SearchBody(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: ""}
	toast := em.Map(uikit.OpSearch, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "Premium required for search.", toast.Body)
}

// TestErrorMapper_Forbidden_PlaylistTracksBody verifies that OpPlaylistTracks
// returns a playlist-specific body for 403 errors.
func TestErrorMapper_Forbidden_PlaylistTracksBody(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: ""}
	toast := em.Map(uikit.OpPlaylistTracks, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "No permission to view this playlist.", toast.Body)
}

// TestErrorMapper_Forbidden_GenericFallback verifies that an operation
// not listed in opForbiddenBody falls back to the generic Premium message.
func TestErrorMapper_Forbidden_GenericFallback(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: ""}
	toast := em.Map(uikit.OpPlaylists, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "A Premium subscription is required for this feature.", toast.Body)
}

// TestErrorMapper_Forbidden_OpSpecificTitle verifies that 403 toasts use
// the operation-specific title (via titleFor) instead of the old hardcoded
// "Spotify Premium required" string.
func TestErrorMapper_Forbidden_OpSpecificTitle(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: ""}
	toast := em.Map(uikit.OpPlayback, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "Playback command failed", toast.Title)
}

// TestErrorMapper_Forbidden_WithMessage_StillOverridesBody verifies that when
// the API returns a non-empty, non-default message, it still takes precedence
// over the operation-specific body. This preserves the existing behavior for
// servers that send meaningful 403 bodies.
func TestErrorMapper_Forbidden_WithMessage_StillOverridesBody(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: "Premium required to use this endpoint."}
	toast := em.Map(uikit.OpQueue, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	// When the server provides a meaningful message, it takes precedence
	// over the operation-specific default.
	assert.Equal(t, "Premium required to use this endpoint.", toast.Body)
}

// TestErrorMapper_OpLikeTracks_Forbidden verifies that a 403 from the like-track
// endpoint produces the operation-specific title and body (story 269).
func TestErrorMapper_OpLikeTracks_Forbidden(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := &api.ForbiddenError{Message: ""}
	toast := em.Map(uikit.OpLikeTracks, err)
	require.Equal(t, uikit.ToastWarning, toast.Intent)
	assert.Equal(t, "Like track failed", toast.Title)
	assert.Equal(t, "Premium required to like tracks.", toast.Body)
}

// TestErrorMapper_OpLikeTracks_Generic verifies that a generic (non-403) error
// for the like-tracks operation uses the operation-specific title with the
// generic Spotify service body.
func TestErrorMapper_OpLikeTracks_Generic(t *testing.T) {
	em := &uikit.ErrorMapper{}
	err := errors.New("spotify 500")
	toast := em.Map(uikit.OpLikeTracks, err)
	require.Equal(t, uikit.ToastError, toast.Intent)
	assert.Equal(t, "Like track failed", toast.Title)
	assert.Equal(t, string(uikit.RecoveryRetryInMoment), toast.Body)
}
