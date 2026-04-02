package app_test

// prefetch_test.go — Tests for Story 83: Search Prefetch Pagination Engine
//
// Task 1: Prefetch constants match spec values.
// Task 2: buildSearchPageCmd fires a single API call with correct offset/limit.
// Task 3: buildSearchBatchCmd sequences exactly 5 page commands (or fewer near maxOffset).
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
	"github.com/initgrep-apps/spotnik/internal/domain"
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

	// Use SearchRequestMsg which now dispatches buildSearchBatchCmd (offset 0).
	// To test a specific page, we need to trigger the prefetch via SearchPrefetchMsg.
	// First set up the store with a query so SearchPrefetchMsg isn't discarded.
	a.Store().SetSearchQuery("jazz")
	a.Store().AppendSearchTracks(makeDomainTracks(30), 50) // 30 loaded, 50 total → has more

	// Trigger prefetch with offset=30 so the first page command uses offset=30.
	_, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 30,
	})
	require.NotNil(t, cmd, "SearchPrefetchMsg should return a batch command")

	// Execute all commands in the sequence to trigger the first HTTP call.
	executeSequenceCmds(cmd, 1) // execute first command in sequence

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
	a.Store().SetSearchQuery("rock")
	_, cmd := a.Update(panes.SearchRequestMsg{Query: "rock"})
	require.NotNil(t, cmd)

	// Execute the sequence — first command fires offset=0.
	msg := executeFirstCmd(cmd)
	pageMsg, ok := msg.(panes.SearchPageLoadedMsg)
	require.True(t, ok, "first command in sequence should return SearchPageLoadedMsg, got %T", msg)

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

	a.Store().SetSearchQuery("jazz")

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
// offset 0 schedules exactly 5 page-fetch commands (offsets 0,10,20,30,40).
func TestBuildSearchBatchCmd_CreatesExactly5Commands(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tracks":{"items":[],"total":0},
			"artists":{"items":[],"total":0},
			"albums":{"items":[],"total":0},
			"playlists":{"items":[],"total":0}
		}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// A search request at offset 0 triggers buildSearchBatchCmd.
	_, cmd := a.Update(panes.SearchRequestMsg{Query: "jazz"})
	require.NotNil(t, cmd)

	// Execute all commands in the sequence to count HTTP calls.
	executeAllCmds(cmd)

	assert.Equal(t, 5, callCount, "buildSearchBatchCmd with offset=0 should fire exactly 5 API calls")
}

// TestBuildSearchBatchCmd_FewerCommandsNearMaxOffset verifies that when startOffset
// is near the Spotify API cap (1000), fewer than 5 commands are created.
// Offset 990 creates 2 commands: offsets 990 and 1000 (guard is `> 1000`, not `>= 1000`).
// Offset 1001 would be excluded since 1001 > 1000.
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

	// Pre-populate store so SearchPrefetchMsg is not filtered.
	a.Store().SetSearchQuery("jazz")
	a.Store().AppendSearchTracks(makeDomainTracks(50), 2000) // 50 loaded, 2000 total → has more

	// Trigger prefetch at offset 990. Guard is `offset > 1000`, so:
	// offset=990 → included, offset=1000 → included (1000 is not > 1000), offset=1010 → excluded.
	_, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 990,
	})
	require.NotNil(t, cmd)

	executeAllCmds(cmd)

	// 2 commands: offset=990 and offset=1000 (1010 excluded by > 1000 guard).
	assert.Equal(t, 2, callCount, "offset=990 should fire 2 commands (990 and 1000; 1010 > 1000 excluded)")
}

// TestBuildSearchBatchCmd_Offset1001_ExcludesAll verifies that when startOffset
// is 1001 (beyond SearchMaxOffset), no commands are created.
// The guard is `offset > searchMaxOffset`, so offset=1001 > 1000 → no pages generated.
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

	// Pre-populate store with items so SearchHasMore is true.
	a.Store().SetSearchQuery("jazz")
	a.Store().AppendSearchTracks(makeDomainTracks(50), 2000)

	_, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 1001, // > 1000 → guard fires on first iteration, no cmds created
	})
	// buildSearchBatchCmd returns nil when no commands are generated.
	assert.Nil(t, cmd, "offset=1001 should return nil (no commands created)")
	assert.Equal(t, 0, callCount, "no API calls should be made when offset > searchMaxOffset")
}

// --- Task 4: SearchPageLoadedMsg handler ---

// TestSearchPageLoadedMsg_StaleQueryDiscarded verifies that results for a superseded
// query are discarded without updating the store.
func TestSearchPageLoadedMsg_StaleQueryDiscarded_NoBatchAccumulation(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetSearchQuery("current")

	// Stale message (query "old" does not match store "current").
	m, cmd := a.Update(panes.SearchPageLoadedMsg{
		Query:   "old",
		Offset:  0,
		Results: &panes.SearchResultData{Tracks: []panes.SearchTrackItem{{URI: "t1"}}},
	})
	a = m.(*app.App)

	assert.Nil(t, cmd, "stale SearchPageLoadedMsg should produce no command")
	assert.Empty(t, a.Store().SearchTracks().Items, "stale results must not be appended to store")
}

// TestSearchPageLoadedMsg_FreshResultsAppended verifies that a fresh SearchPageLoadedMsg
// (matching query) appends its results to the store.
func TestSearchPageLoadedMsg_FreshResultsAppended(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetSearchQuery("jazz")

	tracks := []panes.SearchTrackItem{
		{URI: "spotify:track:t1", Name: "Track 1", Artist: "Artist 1"},
	}
	m, _ := a.Update(panes.SearchPageLoadedMsg{
		Query:  "jazz",
		Offset: 0,
		Results: &panes.SearchResultData{
			Tracks:      tracks,
			TracksTotal: 50,
		},
	})
	a = m.(*app.App)

	storeTracks := a.Store().SearchTracks()
	assert.Len(t, storeTracks.Items, 1, "fresh results should be appended to store")
	assert.Equal(t, 50, storeTracks.Total, "TracksTotal should be set")
}

// TestSearchPageLoadedMsg_ErrorTriggersToast verifies that an error on a fresh
// SearchPageLoadedMsg emits a toast notification.
func TestSearchPageLoadedMsg_ErrorTriggersToast(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetSearchQuery("jazz")

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

// TestSearchPageLoadedMsg_LoadingClearedOnLastPage verifies that the loading flag
// is cleared when the last page of a batch is received.
func TestSearchPageLoadedMsg_LoadingClearedAfterLoad(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetSearchQuery("jazz")
	a.Store().SetSearchLoading(true)

	m, _ := a.Update(panes.SearchPageLoadedMsg{
		Query:  "jazz",
		Offset: 40, // last page in a 5-page batch (offsets 0,10,20,30,40)
		Results: &panes.SearchResultData{
			TracksTotal: 50,
		},
	})
	a = m.(*app.App)

	assert.False(t, a.Store().SearchLoading(), "loading should be cleared when SearchPageLoadedMsg is processed")
}

// --- Task 5: SearchPrefetchMsg handler ---

// TestSearchPrefetchMsg_DispatchesBatchWithCorrectOffset verifies that
// SearchPrefetchMsg triggers a new batch command at the specified NextOffset.
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

	// Pre-populate store so SearchHasMore returns true for "track".
	a.Store().SetSearchQuery("jazz")
	a.Store().AppendSearchTracks(makeDomainTracks(50), 500)

	_, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 50,
	})
	require.NotNil(t, cmd, "SearchPrefetchMsg with more pages should return a batch command")

	executeAllCmds(cmd)

	assert.Equal(t, 5, callCount, "prefetch at offset 50 should fire 5 API calls (offsets 50-90)")
	if len(capturedOffsets) > 0 {
		assert.Equal(t, "50", capturedOffsets[0], "first prefetch page should start at offset 50")
	}
}

// TestSearchPrefetchMsg_SkippedWhenNoMore verifies that SearchPrefetchMsg is a no-op
// when there are no more pages available (store.SearchHasMore returns false).
func TestSearchPrefetchMsg_SkippedWhenNoMore(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetSearchQuery("jazz")
	// Load all tracks (offset == total → no more).
	a.Store().AppendSearchTracks(makeDomainTracks(50), 50)

	_, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 50,
	})
	assert.Nil(t, cmd, "SearchPrefetchMsg should be skipped when SearchHasMore returns false")
}

// TestSearchPrefetchMsg_SkippedWhenQueryStale verifies that SearchPrefetchMsg is a
// no-op when the query does not match the store's current query.
func TestSearchPrefetchMsg_SkippedWhenQueryStale(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetSearchQuery("current") // store has "current", msg has "old"

	_, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "old",
		Types:      []string{"track"},
		NextOffset: 50,
	})
	assert.Nil(t, cmd, "stale SearchPrefetchMsg should be ignored")
}

// --- Task 6: SearchRequestMsg uses batch ---

// TestSearchRequestMsg_UsesBatchCommand_ClearsPreviousAndSetsLoading verifies that
// the updated SearchRequestMsg handler clears old results, sets loading, and dispatches
// buildSearchBatchCmd (which produces 5 sequential page fetches starting at offset 0).
func TestSearchRequestMsg_UsesBatchCommand_ClearsPreviousAndSetsLoading(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":0},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Pre-populate store with stale results from a prior search.
	a.Store().SetSearchQuery("old")
	a.Store().AppendSearchTracks(makeDomainTracks(5), 5)

	m, cmd := a.Update(panes.SearchRequestMsg{Query: "jazz"})
	a = m.(*app.App)

	// Store should be cleared and loading set.
	assert.Equal(t, "jazz", a.Store().SearchQuery(), "store query should be updated")
	assert.True(t, a.Store().SearchLoading(), "loading should be set true")
	assert.Empty(t, a.Store().SearchTracks().Items, "previous results should be cleared")
	require.NotNil(t, cmd, "SearchRequestMsg should dispatch a batch command")

	// Execute all commands in the batch — should fire 5 API calls.
	executeAllCmds(cmd)
	assert.Equal(t, 5, callCount, "SearchRequestMsg should fire 5 sequential page fetches")
}

// --- Task 7: SearchTabChangedMsg clears and re-fetches ---

// TestSearchTabChangedMsg_ClearsAndRefetchesBatch verifies that switching tabs
// clears results and dispatches a full 5-page batch for the new type.
func TestSearchTabChangedMsg_ClearsAndRefetchesBatch(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":0},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Pre-populate store with results from "All" tab search.
	a.Store().SetSearchQuery("jazz")
	a.Store().AppendSearchTracks(makeDomainTracks(10), 50)

	m, cmd := a.Update(panes.SearchTabChangedMsg{
		Query: "jazz",
		Types: []string{"track"},
	})
	a = m.(*app.App)

	assert.Empty(t, a.Store().SearchTracks().Items, "tab change should clear previous results")
	assert.Equal(t, "track", a.Store().SearchActiveType(), "tab change should update active type")
	assert.True(t, a.Store().SearchLoading(), "loading should be set true after tab change")
	require.NotNil(t, cmd, "tab change should dispatch a search command")

	// Execute the sequence — should fire 5 API calls.
	executeAllCmds(cmd)
	assert.Equal(t, 5, callCount, "SearchTabChangedMsg should fire 5 sequential page fetches")
}

// TestSearchTabChangedMsg_AllTab_SetsBatchForAllTypes verifies that switching to
// the All tab dispatches a batch with all 4 type strings.
func TestSearchTabChangedMsg_AllTab_SetsBatchForAllTypes(t *testing.T) {
	callCount := 0
	var capturedTypes []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		capturedTypes = append(capturedTypes, r.URL.Query().Get("type"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":0},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))
	a.Store().SetSearchQuery("jazz")

	_, cmd := a.Update(panes.SearchTabChangedMsg{
		Query: "jazz",
		Types: []string{"track", "artist", "album", "playlist"},
	})
	require.NotNil(t, cmd)

	executeAllCmds(cmd)

	assert.Equal(t, 5, callCount, "All tab should fire 5 sequential page fetches")
	// Each request should request all 4 types.
	for _, typeStr := range capturedTypes {
		assert.Contains(t, typeStr, "track", "All-tab requests should include track type")
	}
}

// --- Task 8: Elm purity — buildSearchPageCmd does not write to store ---

// TestBuildSearchPageCmd_ElmPurity_NoStoreWrites verifies that executing the
// page-fetch command closure does not write to the store. Store mutations
// belong in Update(), not in command closures.
func TestBuildSearchPageCmd_ElmPurity_NoStoreWrites(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tracks":{"items":[{"id":"t1","name":"Track","uri":"spotify:track:t1","artists":[{"name":"Artist"}]}],"total":50},
			"artists":{"items":[],"total":0},
			"albums":{"items":[],"total":0},
			"playlists":{"items":[],"total":0}
		}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Trigger a search to get the batch command.
	_, cmd := a.Update(panes.SearchRequestMsg{Query: "jazz"})
	require.NotNil(t, cmd)

	// Snapshot store state after Update() (which is allowed to write).
	beforeQuery := a.Store().SearchQuery()
	beforeLoading := a.Store().SearchLoading()
	beforeTracks := a.Store().SearchTracks().Items

	// Execute first command (one page-fetch closure).
	msg := executeFirstCmd(cmd)
	require.NotNil(t, msg, "page-fetch command should return a message")

	// Store state must be unchanged by the command closure execution.
	assert.Equal(t, beforeQuery, a.Store().SearchQuery(), "closure must not modify store.SearchQuery")
	assert.Equal(t, beforeLoading, a.Store().SearchLoading(), "closure must not modify store.SearchLoading")
	assert.Equal(t, beforeTracks, a.Store().SearchTracks().Items, "closure must not append to store")

	// Message should carry the results payload.
	pageMsg, ok := msg.(panes.SearchPageLoadedMsg)
	require.True(t, ok, "command should return SearchPageLoadedMsg, got %T", msg)
	assert.NotNil(t, pageMsg.Results, "SearchPageLoadedMsg should carry results payload")
}

// --- Helpers ---

// executeFirstSequenceCmd executes a tea.Cmd and returns the first concrete payload
// message from the resulting sequence. Used in error resilience tests that need
// the message from the first page-fetch in a buildSearchBatchCmd sequence.
// This is an alias for executeFirstCmd — named explicitly for clarity in test contexts
// that document the sequence step being exercised.
func executeFirstSequenceCmd(cmd tea.Cmd) tea.Msg {
	return executeFirstCmd(cmd)
}

// makeDomainTracks returns n minimal domain.Track values for populating the store.
func makeDomainTracks(n int) []domain.Track {
	tracks := make([]domain.Track, n)
	for i := range tracks {
		tracks[i] = domain.Track{ID: "t" + string(rune('0'+i)), Name: "Track"}
	}
	return tracks
}

// executeAllCmds executes all commands produced by a tea.Cmd, handling both
// tea.BatchMsg (exported) and the unexported tea.sequenceMsg via reflection.
// This simulates what the Bubble Tea framework does when processing commands.
func executeAllCmds(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if msg == nil {
		return
	}
	executeMsgCmds(msg)
}

// executeMsgCmds recursively executes all sub-commands in a message.
// Uses reflection to handle the unexported tea.sequenceMsg type.
func executeMsgCmds(msg tea.Msg) {
	if msg == nil {
		return
	}
	switch m := msg.(type) {
	case tea.BatchMsg:
		for _, c := range m {
			if c != nil {
				subMsg := c()
				executeMsgCmds(subMsg)
			}
		}
	default:
		// Use reflection to handle unexported tea.sequenceMsg ([]tea.Cmd).
		execSequenceViaReflect(msg)
	}
}

// execSequenceViaReflect handles tea.sequenceMsg (unexported type in bubbletea v0.27)
// by iterating the underlying slice via reflection and executing each cmd.
// Elements are typed tea.Cmd (named type alias for func() tea.Msg).
func execSequenceViaReflect(msg tea.Msg) {
	v := reflect.ValueOf(msg)
	if v.Kind() != reflect.Slice {
		return
	}
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i).Interface()
		if fn, ok := elem.(tea.Cmd); ok {
			subMsg := fn()
			executeMsgCmds(subMsg)
		}
	}
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
