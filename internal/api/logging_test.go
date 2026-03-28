package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRecorder collects RecordNetCall invocations for testing.
type mockRecorder struct {
	calls []recordedCall
}

type recordedCall struct {
	Method     string
	Path       string
	StatusCode int
	DurationMs int64
}

func (r *mockRecorder) RecordNetCall(method, path string, statusCode int, durationMs int64) {
	r.calls = append(r.calls, recordedCall{
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		DurationMs: durationMs,
	})
}

func TestLoggingTransport_RecordsRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rec := &mockRecorder{}
	transport := api.NewLoggingTransport(http.DefaultTransport, rec)
	client := &http.Client{Transport: transport}

	resp, err := client.Get(srv.URL + "/v1/me/player")
	require.NoError(t, err)
	_ = resp.Body.Close()

	require.Len(t, rec.calls, 1)
	assert.Equal(t, "GET", rec.calls[0].Method)
	assert.Equal(t, "/v1/me/player", rec.calls[0].Path)
	assert.Equal(t, 200, rec.calls[0].StatusCode)
	assert.GreaterOrEqual(t, rec.calls[0].DurationMs, int64(0))
}

func TestLoggingTransport_Records429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	rec := &mockRecorder{}
	transport := api.NewLoggingTransport(http.DefaultTransport, rec)
	client := &http.Client{Transport: transport}

	resp, err := client.Get(srv.URL + "/v1/me/player/queue")
	require.NoError(t, err)
	_ = resp.Body.Close()

	require.Len(t, rec.calls, 1)
	assert.Equal(t, 429, rec.calls[0].StatusCode)
}

func TestLoggingTransport_RecordsOnError(t *testing.T) {
	rec := &mockRecorder{}
	transport := api.NewLoggingTransport(http.DefaultTransport, rec)
	client := &http.Client{Transport: transport}

	// Connect to a closed port to trigger a transport error.
	_, err := client.Get("http://127.0.0.1:1/v1/bad")
	require.Error(t, err)

	require.Len(t, rec.calls, 1)
	assert.Equal(t, "GET", rec.calls[0].Method)
	assert.Equal(t, 0, rec.calls[0].StatusCode, "status should be 0 on transport error")
}

func TestLoggingTransport_SkipsRecordingWhenGatewayRecorded(t *testing.T) {
	// When the gateway has already recorded a request (gatewayRecordedKey set),
	// LoggingTransport must NOT call RecordNetCall to prevent double-recording.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rec := &mockRecorder{}
	transport := api.NewLoggingTransport(http.DefaultTransport, rec)
	client := &http.Client{Transport: transport}

	// Inject the gateway-recorded marker into the request context.
	req, err := http.NewRequest("GET", srv.URL+"/v1/me/player", http.NoBody)
	require.NoError(t, err)
	req = api.MarkGatewayRecorded(req)

	resp, err := client.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()

	assert.Empty(t, rec.calls, "LoggingTransport must not record when gateway-recorded marker is set")
}
