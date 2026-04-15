// in-flight request deduplication — same (Method, Path) key → one HTTP
// call; all concurrent waiters receive a copy of the response body.
package api

import (
	"bytes"
	"io"
	"net/http"
)

// inflightEntry tracks an in-flight HTTP request for deduplication.
// All goroutines waiting on the same key share this entry.
type inflightEntry struct {
	done chan struct{}
	resp *http.Response
	body []byte
	err  error
}

// cloneResponseWithBody creates a shallow copy of resp with the Body replaced
// by a new reader over body. Used so multiple waiters each get their own Body.
func cloneResponseWithBody(resp *http.Response, body []byte) *http.Response {
	clone := *resp
	clone.Body = io.NopCloser(bytes.NewReader(body))
	return &clone
}
