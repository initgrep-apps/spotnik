package app_test

// prefetch_test.go — Tests for Story 83: Search Prefetch Pagination Engine
//
// Updated in Story 98 (search message types refactor):
//   - SearchPrefetchMsg and SearchTabChangedMsg removed; their tests are deleted.
//   - SearchPageLoadedMsg.Offset replaced by SearchPageLoadedMsg.Page (1-based).
//   - Chain-through-Update engine removed from SearchPageLoadedMsg handler.
//
// Remaining tasks:
// Task 1: Prefetch constants match spec values.
// Task 2: buildSearchPageCmd fires a single API call with correct offset/limit.
// Task 3: buildSearchBatchCmd dispatches the first page (chain removed in story 98).
// Task 4: SearchPageLoadedMsg handler: error toast on failure.
// Task 6: SearchRequestMsg handler uses batch command (offset 0).
// Task 8: Elm purity — buildSearchPageCmd closure does not write to store.

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Task 1: Prefetch constants ---

// TestPrefetchConstants_MatchSpec verifies that the exported prefetch constants
// match the values in the story spec.
func TestPrefetchConstants_MatchSpec(t *testing.T) {
	assert.Equal(t, 10, app.SearchPageSize, "searchPageSize should be 10 (API max per request)")
	assert.Equal(t, 5, app.SearchPrefetchPages, "searchPrefetchPages should be 5 pages per batch")
	assert.Equal(t, 50, app.SearchPrefetchItems, "searchPrefetchItems should be 50 (pageSize * prefetchPages)")
	assert.InDelta(t, 0.6, app.SearchPrefetchThreshold, 0.001, "searchPrefetchThreshold should be 0.6")
	assert.Equal(t, 1000, app.SearchMaxOffset, "searchMaxOffset should be 1000 (Spotify API hard cap)")
}

// --- Task 2: buildSearchPageCmd ---

// searchPageServer returns a test server that captures the query parameters.
// The handler function receives the request so the test can assert on parameters.
func searchPageServer(t *testing.T, handler func(*http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tracks":{"items":[{"id":"t1","name":"Track","uri":"spotify:track:t1","artists":[{"name":"Artist"}]}],"total":50},
			"artists":{"items":[],"total":0},
			"albums":{"items":[],"total":0},
			"playlists":{"items":[],"total":0}
		}`))
	}))
}

// TestBuildSearchPageCmd_CorrectOffsetAndLimit verifies that buildSearchPageCmd sends
// the correct offset and limit parameters to the Spotify search API.
// Story 98: triggered via SearchRequestMsg{Page:4} (offset=(4-1)*10=30).
func TestBuildSearchPageCmd_CorrectOffsetAndLimit(t *testing.T) {
	var capturedOffset, capturedLimit string
	srv := searchPageServer(t, func(r *http.Request) {
		capturedOffset = r.URL.Query().Get("offset")
		capturedLimit = r.URL.Query().Get("limit")
	})
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Trigger search at page=4 so the command uses offset=30 (=(4-1)*10).
	_, cmd := a.Update(panes.SearchRequestMsg{
		Query: "jazz",
		Types: []string{"track"},
		Page:  4,
	})
	require.NotNil(t, cmd, "SearchRequestMsg should return a batch command")

	// Execute the batch: index 0 = loadingCmd (SearchLoadingMsg), index 1 = fetchCmd (API call).
	// Story 100 adds SearchLoadingMsg before the fetch; execute both to trigger the API call.
	executeSequenceCmds(cmd, 2)

	// The page-fetch should have used offset=30 and limit=10.
	assert.Equal(t, "30", capturedOffset, "buildSearchPageCmd should use offset derived from Page")
	assert.Equal(t, "10", capturedLimit, "buildSearchPageCmd should use searchPageSize=10 as limit")
}

// TestBuildSearchPageCmd_CarriesQueryAndPage verifies that the returned
// SearchPageLoadedMsg carries the query and 1-based page number.
func TestBuildSearchPageCmd_CarriesQueryAndPage(t *testing.T) {
	srv := searchPageServer(t, func(_ *http.Request) {})
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Dispatch a search request to get a batch command.
	_, cmd := a.Update(panes.SearchRequestMsg{Query: "rock", Page: 1})
	require.NotNil(t, cmd)

	// Execute the batch: index 0 = SearchLoadingMsg, index 1 = fetchCmd (SearchPageLoadedMsg).
	// Story 100 adds a SearchLoadingMsg before the fetch. executeFetchCmd picks the fetch result.
	msg := executeFetchCmd(cmd)
	pageMsg, ok := msg.(panes.SearchPageLoadedMsg)
	require.True(t, ok, "batch command should return SearchPageLoadedMsg, got %T", msg)

	assert.Equal(t, "rock", pageMsg.Query, "SearchPageLoadedMsg should carry the query")
	assert.Equal(t, 1, pageMsg.Page, "first page should have Page=1 (1-based)")
	assert.NotNil(t, pageMsg.Results, "successful fetch should carry results")
}

// TestBuildSearchPageCmd_NilClient_ReturnsError verifies that a nil search client
// returns a SearchPageLoadedMsg with an error (not a panic).
func TestBuildSearchPageCmd_NilClient_ReturnsError(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// No SetSearch — search client is nil.

	_, cmd := a.Update(panes.SearchRequestMsg{Query: "jazz"})
	require.NotNil(t, cmd, "SearchRequestMsg with nil client should still return a command")

	// Execute fetch cmd: index 1 in the batch (index 0 = loadingCmd, story 100).
	msg := executeFetchCmd(cmd)
	pageMsg, ok := msg.(panes.SearchPageLoadedMsg)
	require.True(t, ok, "nil client should return SearchPageLoadedMsg with error, got %T", msg)
	assert.Error(t, pageMsg.Err, "nil client SearchPageLoadedMsg should carry an error")
	assert.Equal(t, "jazz", pageMsg.Query, "error msg should carry the query for staleness detection")
}

// --- Task 3: buildSearchBatchCmd ---

// TestBuildSearchBatchCmd_DispatchesFirstPage verifies that buildSearchBatchCmd
// dispatches exactly one page fetch command (chain-through-Update removed in story 98;
// story 101 re-introduces single-page fetches with context cancellation).
func TestBuildSearchBatchCmd_DispatchesFirstPage(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tracks":{"items":[],"total":500},
			"artists":{"items":[],"total":0},
			"albums":{"items":[],"total":0},
			"playlists":{"items":[],"total":0}
		}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// A search request triggers buildSearchBatchCmd (dispatches first page only).
	_, cmd := a.Update(panes.SearchRequestMsg{Query: "jazz", Page: 1})
	require.NotNil(t, cmd)

	// Execute the batch: index 0 = loadingCmd, index 1 = fetchCmd (API call). Story 100 change.
	executeSequenceCmds(cmd, 2)

	assert.Equal(t, 1, callCount, "buildSearchBatchCmd should dispatch exactly 1 page command")
}

// TestBuildSearchBatchCmd_PageAtMaxOffset_ReturnsNil verifies that when Page maps to
// an offset >= SearchMaxOffset, no command is created (>= guard).
// Page 101 maps to offset=(101-1)*10=1000 which equals SearchMaxOffset.
func TestBuildSearchBatchCmd_PageAtMaxOffset_ReturnsNil(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":2000},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Page 101 → offset=(101-1)*10=1000 == SearchMaxOffset → guard fires, no cmd created.
	_, cmd := a.Update(panes.SearchRequestMsg{
		Query: "jazz",
		Types: []string{"track"},
		Page:  101,
	})
	assert.Nil(t, cmd, "page 101 (offset=1000) should return nil (Spotify API hard cap)")
	assert.Equal(t, 0, callCount, "no API calls should be made when offset >= searchMaxOffset")
}

// --- Task 4: SearchPageLoadedMsg handler ---

// TestSearchPageLoadedMsg_ErrorTriggersToast verifies that an error on a
// SearchPageLoadedMsg emits a toast notification.
func TestSearchPageLoadedMsg_ErrorTriggersToast(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	_, cmd := a.Update(panes.SearchPageLoadedMsg{
		Query: "jazz",
		Page:  1,
		Err:   errors.New("search timed out"),
	})
	require.NotNil(t, cmd, "error SearchPageLoadedMsg should emit a toast command")

	// Two-pass: feed the alert back to verify it shows in View().
	alertMsg := cmd()
	_, _ = a.Update(alertMsg)
	assert.Contains(t, a.View(), "Search failed", "error toast should mention search failure")
}

// --- Task 6: SearchRequestMsg uses batch command ---

// TestSearchRequestMsg_UsesBatchCommand_DispatchesSinglePage verifies that
// SearchRequestMsg dispatches buildSearchBatchCmd (one page fetch; chain removed in story 98).
func TestSearchRequestMsg_UsesBatchCommand_DispatchesSinglePage(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":500},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	_, cmd := a.Update(panes.SearchRequestMsg{Query: "jazz", Page: 1})
	require.NotNil(t, cmd, "SearchRequestMsg should dispatch a batch command")

	// Execute the batch: index 0 = loadingCmd, index 1 = fetchCmd (API call). Story 100 change.
	executeSequenceCmds(cmd, 2)
	assert.Equal(t, 1, callCount, "SearchRequestMsg should dispatch exactly 1 API call")
}

// TestSearchRequestMsg_UsesTypesWhenSet verifies that tab-cycle SearchRequestMsg
// with specific Types fires with those type strings.
func TestSearchRequestMsg_UsesTypesForTabCycle(t *testing.T) {
	var capturedType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedType = r.URL.Query().Get("type")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":0},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Simulate tab-cycle to Songs: overlay emits SearchRequestMsg with Types=["track"].
	_, cmd := a.Update(panes.SearchRequestMsg{Query: "jazz", Types: []string{"track"}, Page: 1})
	require.NotNil(t, cmd)
	// Execute the batch: index 0 = loadingCmd, index 1 = fetchCmd (API call). Story 100 change.
	executeSequenceCmds(cmd, 2)

	assert.Contains(t, capturedType, "track", "tab-cycle request should send the specified type")
}

// --- Helpers ---

// executeFetchCmd executes a tea.Cmd and returns the payload from the fetch (second)
// command in the batch. Story 100 changed SearchRequestMsg to emit a batch of
// (loadingCmd, fetchCmd); index 0 = SearchLoadingMsg, index 1 = SearchPageLoadedMsg.
func executeFetchCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	switch m := msg.(type) {
	case tea.BatchMsg:
		if len(m) > 1 && m[1] != nil {
			return m[1]()
		}
		// Fallback: if batch has only 1 item, execute it.
		if len(m) == 1 && m[0] != nil {
			return m[0]()
		}
		return nil
	default:
		return firstFromReflect(msg)
	}
}

// firstFromReflect extracts the result of the first command from an unexported
// sequence-style message (slice of tea.Cmd).
func firstFromReflect(msg tea.Msg) tea.Msg {
	v := reflect.ValueOf(msg)
	if v.Kind() != reflect.Slice || v.Len() == 0 {
		return msg
	}
	elem := v.Index(0).Interface()
	if fn, ok := elem.(tea.Cmd); ok {
		return fn()
	}
	return msg
}

// executeSequenceCmds executes the first n commands produced by cmd.
func executeSequenceCmds(cmd tea.Cmd, n int) {
	if cmd == nil || n <= 0 {
		return
	}
	msg := cmd()
	if msg == nil {
		return
	}
	executeNFromMsg(msg, n)
}

// executeNFromMsg executes the first n commands from a sequence/batch message.
func executeNFromMsg(msg tea.Msg, n int) {
	switch m := msg.(type) {
	case tea.BatchMsg:
		for i := 0; i < n && i < len(m); i++ {
			if m[i] != nil {
				_ = m[i]()
			}
		}
	default:
		v := reflect.ValueOf(msg)
		if v.Kind() != reflect.Slice {
			return
		}
		for i := 0; i < n && i < v.Len(); i++ {
			elem := v.Index(i).Interface()
			if fn, ok := elem.(tea.Cmd); ok {
				_ = fn()
			}
		}
	}
}
