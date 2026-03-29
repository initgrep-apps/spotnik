package domain

import "time"

// RequestPriority classifies a request so the gateway can apply different policies.
type RequestPriority int

const (
	// PriorityBackground is for polling, pre-fetch, and any non-user-initiated request.
	PriorityBackground RequestPriority = iota
	// PriorityInteractive is for requests triggered by user key presses.
	PriorityInteractive
)

// EventKind classifies a gateway event for the event journal.
type EventKind int

const (
	// EventRequestEntered means a request arrived at the gateway.
	EventRequestEntered EventKind = iota
	// EventTokenConsumed means a background request consumed a token bucket token.
	EventTokenConsumed
	// EventTokenRefilled means tokens recovered (periodic internal event).
	EventTokenRefilled
	// EventSemaphoreAcquired means a request acquired a concurrency semaphore slot.
	EventSemaphoreAcquired
	// EventSemaphoreReleased means a request released its concurrency semaphore slot.
	EventSemaphoreReleased
	// EventBackoffStarted means the gateway received a 429 and entered backoff mode.
	EventBackoffStarted
	// EventBackoffExpired means the 429 backoff period cleared.
	EventBackoffExpired
	// EventRequestAllowed means a request passed through the gateway normally.
	EventRequestAllowed
	// EventRequestWaited means an interactive request waited on the backoff timer.
	EventRequestWaited
	// EventRequestBlocked means a background request was rejected by active backoff.
	EventRequestBlocked
	// EventDedupJoined means a GET request joined an existing in-flight GET (dedup).
	EventDedupJoined
	// EventDedupResolved means dedup waiters received the shared response.
	EventDedupResolved
	// EventHttpCompleted means an HTTP response was received from Spotify.
	EventHttpCompleted
)

// GatewayStateSnapshot holds a frozen copy of gateway internal state at a specific
// moment in time. Embedded in GatewayEvent so the UI can replay state transitions
// without polling. Unlike GatewayState, this has no watermark fields — watermarks
// are replaced by the event journal itself.
type GatewayStateSnapshot struct {
	// TokensAvailable is the token bucket level at this moment (0–10).
	TokensAvailable int
	// TokensMax is the token bucket capacity (always 10).
	TokensMax int
	// ConcurrentActive is the number of in-flight requests holding semaphore slots.
	ConcurrentActive int
	// ConcurrentMax is the semaphore capacity (always 5).
	ConcurrentMax int
	// BackoffRemaining is seconds until the 429 backoff clears (0 if not throttled).
	BackoffRemaining float64
	// DedupWaiters is the number of in-flight GET requests in the dedup map.
	DedupWaiters int
	// InFlightKeys lists string descriptions of currently in-flight GET requests.
	InFlightKeys []string
}

// GatewayEvent records a single gateway lifecycle event with a snapshot of the
// gateway's state at the exact moment the event occurred. Events are stored in a
// ring buffer and replayed by the UI at human-observable speed.
//
// For request-scoped events, RequestID links all events belonging to the same
// request. For internal events (TokenRefilled, BackoffExpired), RequestID is 0.
type GatewayEvent struct {
	// Timestamp is when the event occurred.
	Timestamp time.Time
	// Kind classifies the event.
	Kind EventKind
	// RequestID links events for the same request (0 for internal events).
	RequestID uint64
	// Method is the HTTP method (empty for internal events).
	Method string
	// Path is the API path, e.g. "/me/player" (empty for internal events).
	Path string
	// Priority is Interactive or Background (zero value for internal events).
	Priority RequestPriority
	// StatusCode is the HTTP response status (only set on EventHttpCompleted).
	StatusCode int
	// DurationMs is the HTTP round-trip time (only set on EventHttpCompleted).
	DurationMs int64
	// Snapshot is the gateway's state at this exact moment.
	Snapshot GatewayStateSnapshot
}

// GatewayEventRecorder records gateway lifecycle events.
// Implemented by *state.Store.
type GatewayEventRecorder interface {
	RecordEvent(event GatewayEvent)
}
