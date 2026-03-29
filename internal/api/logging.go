package api

import (
	"net/http"
	"time"
)

// NetLogRecorder is implemented by state.Store to record API call metrics.
// Defined here to avoid an import cycle (api/ cannot import state/).
type NetLogRecorder interface {
	RecordNetCall(method, path string, statusCode int, durationMs int64)
}

// LoggingTransport is an http.RoundTripper that records every request's
// method, path, status code, and duration via a NetLogRecorder.
type LoggingTransport struct {
	inner    http.RoundTripper
	recorder NetLogRecorder
}

// NewLoggingTransport wraps inner with API call logging to recorder.
func NewLoggingTransport(inner http.RoundTripper, recorder NetLogRecorder) *LoggingTransport {
	return &LoggingTransport{
		inner:    inner,
		recorder: recorder,
	}
}

// RoundTrip implements http.RoundTripper. It records timing and status
// for every HTTP request that passes through the transport.
func (lt *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := lt.inner.RoundTrip(req)
	durationMs := time.Since(start).Milliseconds()

	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}

	lt.recorder.RecordNetCall(req.Method, req.URL.Path, statusCode, durationMs)

	return resp, err
}
