package panes

import (
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
)

// animationPhase tracks where a request is in its visual lifecycle.
type animationPhase int

const (
	phaseEntered   animationPhase = iota // appeared in APP box
	phaseAtGateway                       // gateway decision rendered
	phaseInFlight                        // HTTP call in progress (arrow animating right)
	phaseCompleted                       // response received
	phaseDone                            // aged out, ready for removal
)

// requestAnimation tracks one request's visual state across all three boxes.
type requestAnimation struct {
	requestID  uint64
	method     string
	path       string
	priority   domain.RequestPriority
	phase      animationPhase
	decision   domain.EventKind // the gateway decision event kind
	statusCode int
	durationMs int64
	enteredAt  time.Time // when it appeared in APP box (replay time, not event time)
}

// decisionEntry is one line in the GATEWAY box's scrolling decision log.
type decisionEntry struct {
	kind       domain.EventKind
	label      string                 // formatted: "✓ GET /player allowed", "↻ refilled → 10"
	shownAt    time.Time              // when this entry was added (for age-out)
	priority   domain.RequestPriority // for EventRequestEntered color
	statusCode int                    // for EventHttpCompleted color
}

// replayDisplayState is the render model that View() reads from.
// Updated by the replay loop on each viz.TickMsg.
type replayDisplayState struct {
	// snapshot is the gateway state from the most recently replayed event.
	snapshot domain.GatewayStateSnapshot
	// requests tracks active requests keyed by RequestID.
	requests map[uint64]*requestAnimation
	// decisions is the scrolling decision log for the GATEWAY box.
	decisions []decisionEntry
}
