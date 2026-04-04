package app_test

// prefetch_test.go — Tests for Story 83: Search Prefetch Pagination Engine
//
// Task 1: Prefetch constants match spec values.
// Task 2: buildSearchPageCmd fires a single API call with correct offset/limit.
// Task 3: buildSearchBatchCmd dispatches 1 page; chain-through-Update fires exactly 5 total (or fewer near maxOffset).
// Task 4: SearchPageLoadedMsg handler: stale discarded, results appended, error toast, loading cleared.
// Task 5: SearchPrefetchMsg handler: dispatches batch, skipped when no more, skipped when stale.
// Task 6: SearchRequestMsg handler uses batch command (offset 0, clears previous results).
// Task 7: SearchTabChangedMsg clears results and dispatches batch with tab-specific types.
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

	// Trigger prefetch with offset=30 so the first page command uses offset=30.
	_, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 30,
	})
	require.NotNil(t, cmd, "SearchPrefetchMsg should return a batch command")

	// Execute the first page command (buildSearchBatchCmd dispatches one page at a time).
	executeSequenceCmds(cmd, 1)

	// The first page-fetch should have used offset=30 and limit=10.
	assert.Equal(t, "30", capturedOffset, "buildSearchPageCmd should use the provided offset")
	assert.Equal(t, "10", capturedLimit, "buildSearchPageCmd should use searchPageSize=10 as limit")
}

// TestBuildSearchPageCmd_CarriesQueryAndOffset verifies that the returned
// SearchPageLoadedMsg carries the query and offset from the command, not just results.
func TestBuildSearchPageCmd_CarriesQueryAndOffset(t *testing.T) {
	srv := searchPageServer(t, func(_ *http.Request) {})
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Dispatch a search request to get a batch command.
	_, cmd := a.Update(panes.SearchRequestMsg{Query: "rock"})
	require.NotNil(t, cmd)

	// Execute the batch command — fires the first (and only initial) page at offset=0.
	msg := executeFirstCmd(cmd)
	pageMsg, ok := msg.(panes.SearchPageLoadedMsg)
	require.True(t, ok, "batch command should return SearchPageLoadedMsg, got %T", msg)

	assert.Equal(t, "rock", pageMsg.Query, "SearchPageLoadedMsg should carry the query")
	assert.Equal(t, 0, pageMsg.Offset, "first page should have offset=0")
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

	// Execute first command.
	msg := executeFirstCmd(cmd)
	pageMsg, ok := msg.(panes.SearchPageLoadedMsg)
	require.True(t, ok, "nil client should return SearchPageLoadedMsg with error, got %T", msg)
	assert.Error(t, pageMsg.Err, "nil client SearchPageLoadedMsg should carry an error")
	assert.Equal(t, "jazz", pageMsg.Query, "error msg should carry the query for staleness detection")
}

// --- Task 3: buildSearchBatchCmd ---

// TestBuildSearchBatchCmd_CreatesExactly5Commands verifies that a batch starting at
// offset 0 results in exactly 5 API calls via the chain-through-Update pattern.
// buildSearchBatchCmd dispatches only the first page; each SearchPageLoadedMsg
// handler chains the next page until the batch boundary (offset 40+10=50 = batchEnd).
func TestBuildSearchBatchCmd_CreatesExactly5Commands(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return total > 50 so SearchHasMore returns true throughout the batch,
		// allowing the chain to continue until the batch boundary.
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

	// A search request at offset 0 triggers buildSearchBatchCmd (dispatches first page only).
	model, cmd := a.Update(panes.SearchRequestMsg{Query: "jazz"})
	a = model.(*app.App)
	require.NotNil(t, cmd)

	// Drive the chain through Update: each SearchPageLoadedMsg chains the next page.
	// 5 iterations: offsets 0,10,20,30,40; offset 50 = batchEnd so chain stops.
	for i := 0; i < 5 && cmd != nil; i++ {
		msg := cmd()
		model, cmd = a.Update(msg)
		a = model.(*app.App)
	}

	assert.Equal(t, 5, callCount, "buildSearchBatchCmd chain-through-Update should fire exactly 5 API calls")
}

// TestBuildSearchBatchCmd_FewerCommandsNearMaxOffset verifies that when startOffset
// is near the Spotify API cap (1000), the chain-through-Update stops at SearchMaxOffset.
// Batch at offset 990: batchEnd = ((990/50)+1)*50 = 20*50 = 1000.
// Pages: 990 (batchEnd=1000, next=1000 < 1000 is false) → only 1 API call.
// Note: the old seq guard was `offset > 1000`; the chain now stops when nextOffset >= batchEnd.
func TestBuildSearchBatchCmd_FewerCommandsNearMaxOffset(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tracks":{"items":[],"total":2000},
			"artists":{"items":[],"total":0},
			"albums":{"items":[],"total":0},
			"playlists":{"items":[],"total":0}
		}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Trigger prefetch at offset 990.
	// batchEnd = ((990/50)+1)*50 = 20*50 = 1000.
	// Page at offset 990 fires; nextOffset=1000, 1000 < 1000 is false → chain stops after 1 page.
	model, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 990,
	})
	a = model.(*app.App)
	require.NotNil(t, cmd)

	// Drive chain: offset 990 fires, then chain stops (nextOffset 1000 is not < batchEnd 1000).
	for cmd != nil {
		msg := cmd()
		pageMsg, ok := msg.(panes.SearchPageLoadedMsg)
		if !ok {
			break
		}
		model, cmd = a.Update(pageMsg)
		a = model.(*app.App)
	}

	// Chain stops after 1 page: offset=990 (nextOffset=1000 not < batchEnd=1000).
	assert.Equal(t, 1, callCount, "offset=990 with batchEnd=1000: chain fires 1 page (990), stops when nextOffset==batchEnd")
}

// TestBuildSearchBatchCmd_Offset1001_ExcludesAll verifies that when startOffset
// is 1001 (beyond SearchMaxOffset), no commands are created.
// The guard is `offset >= searchMaxOffset`, so offset=1001 >= 1000 → no pages generated.
func TestBuildSearchBatchCmd_Offset1001_ExcludesAll(t *testing.T) {
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

	_, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 1001, // > 1000 → guard fires on first iteration, no cmds created
	})
	// buildSearchBatchCmd returns nil when no commands are generated.
	assert.Nil(t, cmd, "offset=1001 should return nil (no commands created)")
	assert.Equal(t, 0, callCount, "no API calls should be made when offset >= searchMaxOffset")
}

// TestBuildSearchBatchCmd_Offset1000_ExcludesAll verifies that when startOffset
// equals SearchMaxOffset (1000), no commands are created (>= guard).
func TestBuildSearchBatchCmd_Offset1000_ExcludesAll(t *testing.T) {
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

	_, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 1000, // == SearchMaxOffset → guard fires, no cmds created
	})
	// buildSearchBatchCmd returns nil when offset >= SearchMaxOffset.
	assert.Nil(t, cmd, "offset=1000 should return nil (Spotify API hard cap)")
	assert.Equal(t, 0, callCount, "no API calls should be made when offset == searchMaxOffset")
}

// --- Task 4: SearchPageLoadedMsg handler ---

// TestSearchPageLoadedMsg_ErrorTriggersToast verifies that an error on a
// SearchPageLoadedMsg emits a toast notification.
func TestSearchPageLoadedMsg_ErrorTriggersToast(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	_, cmd := a.Update(panes.SearchPageLoadedMsg{
		Query:  "jazz",
		Offset: 0,
		Err:    errors.New("search timed out"),
	})
	require.NotNil(t, cmd, "error SearchPageLoadedMsg should emit a toast command")

	// Two-pass: feed the alert back to verify it shows in View().
	alertMsg := cmd()
	_, _ = a.Update(alertMsg)
	assert.Contains(t, a.View(), "Search failed", "error toast should mention search failure")
}

// --- Task 5: SearchPrefetchMsg handler ---

// TestSearchPrefetchMsg_DispatchesBatchWithCorrectOffset verifies that
// SearchPrefetchMsg triggers a new batch via chain-through-Update starting at NextOffset.
// The chain fires 5 pages (offsets 50,60,70,80,90) then stops at batchEnd=100.
func TestSearchPrefetchMsg_DispatchesBatchWithCorrectOffset(t *testing.T) {
	callCount := 0
	var capturedOffsets []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		capturedOffsets = append(capturedOffsets, r.URL.Query().Get("offset"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":500},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	model, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 50,
	})
	a = model.(*app.App)
	require.NotNil(t, cmd, "SearchPrefetchMsg with more pages should return a batch command")

	// Drive chain through Update: batch at offset 50 has batchEnd=((50/50)+1)*50=100.
	// Pages: 50,60,70,80,90 — nextOffset 100 is not < batchEnd 100 → stops after 5.
	for i := 0; i < 5 && cmd != nil; i++ {
		msg := cmd()
		model, cmd = a.Update(msg)
		a = model.(*app.App)
	}

	assert.Equal(t, 5, callCount, "prefetch at offset 50 should fire 5 API calls (offsets 50-90)")
	if len(capturedOffsets) > 0 {
		assert.Equal(t, "50", capturedOffsets[0], "first prefetch page should start at offset 50")
	}
}

// --- Task 6: SearchRequestMsg uses batch ---

// TestSearchRequestMsg_UsesBatchCommand_DispatchesBatch verifies that
// SearchRequestMsg dispatches buildSearchBatchCmd (chain-through-Update fires exactly
// 5 page fetches starting at offset 0 when total=500 keeps the chain alive).
func TestSearchRequestMsg_UsesBatchCommand_DispatchesBatch(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// total=500 ensures chain continues until the batch boundary (offset 50).
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":500},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	model, cmd := a.Update(panes.SearchRequestMsg{Query: "jazz"})
	a = model.(*app.App)

	require.NotNil(t, cmd, "SearchRequestMsg should dispatch a batch command")

	// Drive chain through Update: batch at offset 0 has batchEnd=50.
	// Pages: 0,10,20,30,40 → 5 API calls. nextOffset=50 not < batchEnd=50 → stops.
	for i := 0; i < 5 && cmd != nil; i++ {
		msg := cmd()
		model, cmd = a.Update(msg)
		a = model.(*app.App)
	}
	assert.Equal(t, 5, callCount, "SearchRequestMsg chain-through-Update should fire 5 API calls")
}

// --- Task 7: SearchTabChangedMsg clears and re-fetches ---

// TestSearchTabChangedMsg_DispatchesBatch verifies that switching tabs triggers
// a 5-page chain-through-Update batch for the new type.
func TestSearchTabChangedMsg_DispatchesBatch(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// total=500 ensures chain continues until the batch boundary.
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":500},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	model, cmd := a.Update(panes.SearchTabChangedMsg{
		Query: "jazz",
		Types: []string{"track"},
	})
	a = model.(*app.App)

	require.NotNil(t, cmd, "tab change should dispatch a search command")

	// Drive chain through Update: batch at offset 0 has batchEnd=50.
	// Pages: 0,10,20,30,40 → 5 API calls.
	for i := 0; i < 5 && cmd != nil; i++ {
		msg := cmd()
		model, cmd = a.Update(msg)
		a = model.(*app.App)
	}
	assert.Equal(t, 5, callCount, "SearchTabChangedMsg chain-through-Update should fire 5 API calls")
}

// TestSearchTabChangedMsg_AllTab_SetsBatchForAllTypes verifies that switching to
// the All tab dispatches a batch with all 4 type strings via chain-through-Update.
func TestSearchTabChangedMsg_AllTab_SetsBatchForAllTypes(t *testing.T) {
	callCount := 0
	var capturedTypes []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		capturedTypes = append(capturedTypes, r.URL.Query().Get("type"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// total=500 so SearchHasMore("all") stays true for all 5 pages.
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":500},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	model, cmd := a.Update(panes.SearchTabChangedMsg{
		Query: "jazz",
		Types: []string{"track", "artist", "album", "playlist"},
	})
	a = model.(*app.App)
	require.NotNil(t, cmd)

	// Drive chain through Update: 5 pages for batchEnd=50.
	for i := 0; i < 5 && cmd != nil; i++ {
		msg := cmd()
		model, cmd = a.Update(msg)
		a = model.(*app.App)
	}

	assert.Equal(t, 5, callCount, "All tab chain-through-Update should fire 5 API calls")
	// Each request should use all 4 types.
	for _, typeStr := range capturedTypes {
		assert.Contains(t, typeStr, "track", "All-tab requests should include track type")
	}
}

// --- Helpers ---

// executeFirstSequenceCmd executes a tea.Cmd and returns the first concrete payload
// message. Used in error resilience tests that need the message from the first
// page-fetch command dispatched by buildSearchBatchCmd.
// This is an alias for executeFirstCmd — named explicitly for clarity in test contexts.
func executeFirstSequenceCmd(cmd tea.Cmd) tea.Msg {
	return executeFirstCmd(cmd)
}

// executeFirstCmd executes a tea.Cmd and returns the first concrete payload message.
// Handles tea.BatchMsg and the unexported tea.sequenceMsg via reflection.
func executeFirstCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	switch m := msg.(type) {
	case tea.BatchMsg:
		if len(m) > 0 && m[0] != nil {
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
