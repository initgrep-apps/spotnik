package app_test

// search_prefix_test.go — Tests for Story 85: Search Prefix Autocomplete
//
// Task 8: SearchRequestMsg handler uses msg.Types when set; defaults to all 4 types when empty.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchRequestMsg_UsesTypesFromMsg verifies that when SearchRequestMsg.Types
// is set (e.g. from a locked prefix), the app handler uses those types for the
// search batch command rather than the default all-types list.
func TestSearchRequestMsg_UsesTypesFromMsg(t *testing.T) {
	var capturedTypeParam string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTypeParam = r.URL.Query().Get("type")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return a response with total=0 so the chain stops after the first page.
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":0},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Fire SearchRequestMsg with Types=["track"] (as set by :songs prefix).
	_, cmd := a.Update(panes.SearchRequestMsg{Query: "kk", Types: []string{"track"}})

	require.NotNil(t, cmd, "SearchRequestMsg with Types should dispatch a search command")

	// Story 100: SearchRequestMsg returns a batch of (loadingCmd, fetchCmd).
	// Execute the batch to trigger the API call.
	batchMsg := cmd()
	if batch, ok := batchMsg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				c() //nolint:errcheck
			}
		}
	}

	// The API should have been called with type=track only.
	assert.Equal(t, "track", capturedTypeParam,
		"app should use msg.Types for the search API call when set")
}

// TestSearchRequestMsg_DefaultsToAllTypesWhenEmpty verifies that when
// SearchRequestMsg.Types is empty, the app handler defaults to all 4 types.
func TestSearchRequestMsg_DefaultsToAllTypesWhenEmpty(t *testing.T) {
	var capturedTypeParam string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTypeParam = r.URL.Query().Get("type")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":0},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Fire SearchRequestMsg with empty Types (no prefix locked).
	_, cmd := a.Update(panes.SearchRequestMsg{Query: "jazz"})

	require.NotNil(t, cmd, "SearchRequestMsg with empty Types should dispatch a search command")

	// Story 100: SearchRequestMsg returns a batch of (loadingCmd, fetchCmd).
	// Execute the batch to trigger the API call.
	batchMsg := cmd()
	if batch, ok := batchMsg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				c() //nolint:errcheck
			}
		}
	}

	// The API should have been called with all 4 types.
	for _, expectedType := range []string{"track", "artist", "album", "playlist"} {
		assert.True(t, strings.Contains(capturedTypeParam, expectedType),
			"default types should include %s, got: %s", expectedType, capturedTypeParam)
	}
}
