// in-flight request deduplication — same (Method, Path) key → one HTTP
// call; all concurrent waiters receive a copy of the response body.
package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

// inflightEntry tracks an in-flight HTTP request for deduplication.
// All goroutines waiting on the same key share this entry.
type inflightEntry struct {
	done chan struct{}
	resp *http.Response
	body []byte
	err  error
}

// interactiveDebounceEntry holds the cancel function and ready channel for one
// pending Interactive request waiting in the 100ms debounce hold window.
// cancel stops the hold; ready is closed when the entry exits (used by the
// replacement to wait before registering itself).
type interactiveDebounceEntry struct {
	cancel context.CancelFunc
	ready  chan struct{}
}

// cloneResponseWithBody creates a shallow copy of resp with the Body replaced
// by a new reader over body. Used so multiple waiters each get their own Body.
func cloneResponseWithBody(resp *http.Response, body []byte) *http.Response {
	clone := *resp
	clone.Body = io.NopCloser(bytes.NewReader(body))
	return &clone
}

// interactiveDebounce implements a 100ms hold window for Interactive requests
// keyed by API path (query params ignored). If a newer request for the same
// path arrives within 100ms, the older one is cancelled and returns an error.
// Only the last request in any burst window proceeds.
//
// The wrappedCtx allows a competing request to cancel this hold without also
// cancelling the caller's ctx (which controls the downstream HTTP call).
func (g *Gateway) interactiveDebounce(ctx context.Context, path string) error {
	// wrappedCtx allows a competing request to cancel this hold without
	// cancelling the caller's ctx (which would affect the HTTP call too).
	wrappedCtx, wrappedCancel := context.WithCancel(ctx)

	g.debounceMu.Lock()
	if prev, ok := g.debounceEntries[path]; ok {
		// Cancel the prior request's hold and wait for it to finish unregistering.
		prev.cancel()
		g.debounceMu.Unlock()
		<-prev.ready
		g.debounceMu.Lock()
	}
	entry := &interactiveDebounceEntry{
		cancel: wrappedCancel,
		ready:  make(chan struct{}),
	}
	g.debounceEntries[path] = entry
	g.debounceMu.Unlock()

	// Cleanup: remove from map and signal ready when we exit.
	defer func() {
		wrappedCancel()
		g.debounceMu.Lock()
		if g.debounceEntries[path] == entry {
			delete(g.debounceEntries, path)
		}
		g.debounceMu.Unlock()
		close(entry.ready)
	}()

	// Hold for 100ms. The first request to survive the full hold proceeds.
	// Use time.NewTimer instead of time.After to prevent timer leaks when
	// ctx is cancelled before the 100ms window expires.
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil // proceed to HTTP
	case <-wrappedCtx.Done():
		return wrappedCtx.Err() // cancelled by newer request for same path
	case <-ctx.Done():
		return ctx.Err() // cancelled by caller (Esc, new query from app.go)
	}
}
