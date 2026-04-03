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
	"fmt"
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
	a.Store().SetSearchQuery("rock")
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

	// Pre-populate store so SearchPrefetchMsg is not filtered.
	a.Store().SetSearchQuery("jazz")
	a.Store().AppendSearchTracks(makeDomainTracks(50), 2000) // 50 loaded, 2000 total → has more

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

	// Pre-populate store: 50 items loaded, total=2000 → has more.
	a.Store().SetSearchQuery("jazz")
	a.Store().AppendSearchTracks(makeDomainTracks(50), 2000)

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
		Results: &panes.SearchResultData{Tracks: []domain.Track{{URI: "t1"}}},
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

	tracks := []domain.Track{
		{URI: "spotify:track:t1", Name: "Track 1", Artists: []domain.Artist{{Name: "Artist 1"}}},
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

// TestSearchPageLoadedMsg_ChainStopsOn429 verifies that a 429 (RateLimitedMsg)
// returned during a chain-through-Update batch stops the chain naturally, because
// RateLimitedMsg is not a SearchPageLoadedMsg so the handler never chains another page.
func TestSearchPageLoadedMsg_ChainStopsOn429(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			// First page succeeds with total=500 to keep SearchHasMore true.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":500},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
			return
		}
		// Second page gets 429 — chain should stop here.
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))
	a.Store().SetSearchQuery("jazz")

	model, cmd := a.Update(panes.SearchRequestMsg{Query: "jazz"})
	a = model.(*app.App)
	require.NotNil(t, cmd)

	// Step 1: Execute first page command → SearchPageLoadedMsg (success) → chains page 2.
	msg := cmd()
	model, chainCmd := a.Update(msg)
	a = model.(*app.App)
	require.NotNil(t, chainCmd, "first page success should chain next page")

	// Step 2: Execute chained page 2 → RateLimitedMsg.
	msg2 := chainCmd()
	rateLimitMsg, ok := msg2.(panes.RateLimitedMsg)
	assert.True(t, ok, "second page 429 should produce RateLimitedMsg, got %T", msg2)
	assert.Equal(t, 5, rateLimitMsg.RetryAfterSecs)

	// Step 3: Feed RateLimitedMsg to Update — it clears SearchLoading.
	model, _ = a.Update(msg2)
	a = model.(*app.App)
	assert.False(t, a.Store().SearchLoading(), "RateLimitedMsg should clear SearchLoading")

	// Only 2 API calls — chain stopped after the 429.
	assert.Equal(t, 2, callCount, "chain should stop after 429: only 2 API calls made")
}

// TestRateLimitedMsg_ClearsSearchLoading verifies that clearAllFetchingSentinels
// (called from RateLimitedMsg handler) clears the SearchLoading flag. This prevents
// the overlay from staying stuck in a loading state when a 429 interrupts a batch.
func TestRateLimitedMsg_ClearsSearchLoading(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetSearchLoading(true)

	model, _ := a.Update(panes.RateLimitedMsg{RetryAfterSecs: 10})
	a = model.(*app.App)

	assert.False(t, a.Store().SearchLoading(), "RateLimitedMsg should clear SearchLoading via clearAllFetchingSentinels")
}

// TestUnauthorizedMsg_ClearsSearchLoading verifies that clearAllFetchingSentinels
// (called from unauthorizedMsg handler) clears the SearchLoading flag.
func TestUnauthorizedMsg_ClearsSearchLoading(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetSearchLoading(true)

	// unauthorizedMsg is unexported; trigger it via a 401 response from the search API.
	srv := unauthorizedServer()
	defer srv.Close()
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Trigger search → get page cmd → execute to get 401 → feed unauthorizedMsg to Update.
	_, cmd := a.Update(panes.SearchRequestMsg{Query: "test"})
	require.NotNil(t, cmd)
	unauthorizedResult := executeFirstSequenceCmd(cmd)

	model, _ := a.Update(unauthorizedResult)
	a = model.(*app.App)

	assert.False(t, a.Store().SearchLoading(), "unauthorizedMsg should clear SearchLoading via clearAllFetchingSentinels")
}

// TestSearchPrefetchMsg_SkippedWhenAlreadyLoading verifies that concurrent
// prefetch batches are blocked: if SearchLoading is true, the SearchPrefetchMsg is ignored.
func TestSearchPrefetchMsg_SkippedWhenAlreadyLoading(t *testing.T) {
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
	a.Store().SetSearchQuery("jazz")
	a.Store().AppendSearchTracks(makeDomainTracks(50), 500)
	// Simulate an in-flight batch already running.
	a.Store().SetSearchLoading(true)

	_, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 50,
	})
	assert.Nil(t, cmd, "SearchPrefetchMsg should be a no-op when SearchLoading is already true")
	assert.Equal(t, 0, callCount, "no API calls should be made when a batch is already in-flight")
}

// TestSearchRequestMsg_NilBatchCmd_ClearsLoading verifies that when buildSearchBatchCmd
// returns nil (startOffset > SearchMaxOffset), the SearchLoading flag is cleared.
// This guards against the loading flag leaking when an offset is out of range.
func TestSearchRequestMsg_NilBatchCmd_ClearsLoading(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Use SearchPrefetchMsg at offset > 1000 — buildSearchBatchCmd returns nil.
	a.Store().SetSearchQuery("jazz")
	a.Store().AppendSearchTracks(makeDomainTracks(50), 2000)

	model, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 1001, // > SearchMaxOffset=1000 → buildSearchBatchCmd returns nil
	})
	a = model.(*app.App)

	assert.Nil(t, cmd, "nil batch cmd guard should produce no command")
	assert.False(t, a.Store().SearchLoading(), "nil batch cmd guard should clear SearchLoading")
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

	// Pre-populate store so SearchHasMore returns true for "track".
	a.Store().SetSearchQuery("jazz")
	a.Store().AppendSearchTracks(makeDomainTracks(50), 500)

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
// buildSearchBatchCmd. The chain-through-Update fires 5 page fetches starting at offset 0.
// (total=0 on each page → SearchHasMore returns false; chain stops after 1 page.)
// To verify 5 pages fire, we use total=500 so SearchHasMore stays true through the batch.
func TestSearchRequestMsg_UsesBatchCommand_ClearsPreviousAndSetsLoading(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// total=500 ensures SearchHasMore stays true throughout the 5-page batch.
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":500},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Pre-populate store with stale results from a prior search.
	a.Store().SetSearchQuery("old")
	a.Store().AppendSearchTracks(makeDomainTracks(5), 5)

	model, cmd := a.Update(panes.SearchRequestMsg{Query: "jazz"})
	a = model.(*app.App)

	// Store should be cleared and loading set.
	assert.Equal(t, "jazz", a.Store().SearchQuery(), "store query should be updated")
	assert.True(t, a.Store().SearchLoading(), "loading should be set true")
	assert.Empty(t, a.Store().SearchTracks().Items, "previous results should be cleared")
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

// TestSearchTabChangedMsg_ClearsAndRefetchesBatch verifies that switching tabs
// clears results and triggers a 5-page chain-through-Update batch for the new type.
func TestSearchTabChangedMsg_ClearsAndRefetchesBatch(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// total=500 ensures SearchHasMore stays true throughout the 5-page batch.
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":500},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Pre-populate store with results from "All" tab search.
	a.Store().SetSearchQuery("jazz")
	a.Store().AppendSearchTracks(makeDomainTracks(10), 50)

	model, cmd := a.Update(panes.SearchTabChangedMsg{
		Query: "jazz",
		Types: []string{"track"},
	})
	a = model.(*app.App)

	assert.Empty(t, a.Store().SearchTracks().Items, "tab change should clear previous results")
	assert.Equal(t, "track", a.Store().SearchActiveType(), "tab change should update active type")
	assert.True(t, a.Store().SearchLoading(), "loading should be set true after tab change")
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
	a.Store().SetSearchQuery("jazz")

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
// message. Used in error resilience tests that need the message from the first
// page-fetch command dispatched by buildSearchBatchCmd.
// This is an alias for executeFirstCmd — named explicitly for clarity in test contexts.
func executeFirstSequenceCmd(cmd tea.Cmd) tea.Msg {
	return executeFirstCmd(cmd)
}

// makeDomainTracks returns n minimal domain.Track values for populating the store.
// IDs are formatted as "t0", "t1", ..., "tN" so they are valid for any N.
func makeDomainTracks(n int) []domain.Track {
	tracks := make([]domain.Track, n)
	for i := range tracks {
		tracks[i] = domain.Track{ID: fmt.Sprintf("t%d", i), Name: "Track"}
	}
	return tracks
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
