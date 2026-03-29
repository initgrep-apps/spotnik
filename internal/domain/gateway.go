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

// GatewayDecision classifies the outcome of a request's passage through the gateway.
type GatewayDecision int

const (
	// DecisionAllowed means the request passed through the gateway normally.
	DecisionAllowed GatewayDecision = iota
	// DecisionWaited means the request waited at the token bucket or semaphore.
	DecisionWaited
	// DecisionDeduped means the request joined an existing in-flight GET (dedup hit).
	DecisionDeduped
	// DecisionBlocked means the request was rejected by 429 backoff (Background only).
	DecisionBlocked
)

// GatewayState holds a read-only snapshot of gateway internals for display.
// All fields are safe to read without holding any lock — they are copied under
// the gateway's mutex inside Snapshot().
type GatewayState struct {
	// TokensAvailable is the current token bucket level (0-10).
	TokensAvailable int
	// TokensMax is the token bucket capacity (always 10).
	TokensMax int
	// ConcurrentActive is the number of in-flight requests currently holding a semaphore slot.
	ConcurrentActive int
	// ConcurrentMax is the semaphore capacity (always 5).
	ConcurrentMax int
	// BackoffRemaining is the seconds until the 429 backoff period clears (0 if not throttled).
	BackoffRemaining float64
	// DedupWaiters is the number of in-flight GET requests tracked in the dedup map.
	DedupWaiters int
	// InFlightKeys lists string representations of currently in-flight GET requests.
	InFlightKeys []string

	// PeakConcurrent is the highest ConcurrentActive seen since the last watermark reset.
	// Tracked inside the Gateway at the moment of semaphore acquisition.
	PeakConcurrent int
	// MinTokens is the lowest TokensAvailable seen since the last watermark reset.
	// Tracked inside the tokenBucket at the moment of token consumption.
	MinTokens int
}

// GatewaySnapshotter provides a read-only view into gateway internal state.
// Implemented by *api.Gateway; used by RequestFlowPane to avoid ui/ importing api/.
type GatewaySnapshotter interface {
	Snapshot() GatewayState
	// ResetWatermarks resets peak activity watermarks to current values. Called by
	// the UI on each 1-second boundary so annotations reflect only recent activity.
	ResetWatermarks()
}

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
