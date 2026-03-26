package domain

// RequestPriority classifies a request so the gateway can apply different policies.
type RequestPriority int

const (
	// PriorityBackground is for polling, pre-fetch, and any non-user-initiated request.
	PriorityBackground RequestPriority = iota
	// PriorityInteractive is for requests triggered by user key presses.
	PriorityInteractive
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
}

// GatewaySnapshotter provides a read-only view into gateway internal state.
// Implemented by *api.Gateway; used by RequestFlowPane to avoid ui/ importing api/.
type GatewaySnapshotter interface {
	Snapshot() GatewayState
}
