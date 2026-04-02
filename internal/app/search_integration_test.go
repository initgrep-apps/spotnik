package app_test

// search_integration_test.go — Integration tests for Story 86: Search Cleanup and Integration
//
// Scenario 1: Basic search flow — open overlay → type query → debounce → batch → store populated.
// Scenario 2: Tab switching — tab change clears results and re-fires with type filter.
// Scenario 3: Prefix flow — ':songs kk' → prefix locks → search fires with query="kk", type=track.
// Scenario 4: Prefetch trigger — 50 items loaded, scroll past 60% → SearchPrefetchMsg fires.
// Scenario 5: Stale result discard — 'kk' results arrive after 'jazz' query set → discarded.
// Scenario 6: Close and reopen — closing then reopening clears results and resets input.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// searchJSONResponse builds a Spotify search JSON response with n tracks and the given total.
func searchJSONResponse(n, total int) []byte {
	type artistObj struct {
		Name string `json:"name"`
	}
	type trackObj struct {
		ID      string      `json:"id"`
		Name    string      `json:"name"`
		URI     string      `json:"uri"`
		Artists []artistObj `json:"artists"`
	}
	type tracksPage struct {
		Items []trackObj `json:"items"`
		Total int        `json:"total"`
	}
	type emptyPage struct {
		Items []struct{} `json:"items"`
		Total int        `json:"total"`
	}
	type response struct {
		Tracks    tracksPage `json:"tracks"`
		Artists   emptyPage  `json:"artists"`
		Albums    emptyPage  `json:"albums"`
		Playlists emptyPage  `json:"playlists"`
	}

	tracks := make([]trackObj, n)
	for i := range tracks {
		tracks[i] = trackObj{
			ID:      fmt.Sprintf("t%d", i),
			Name:    fmt.Sprintf("Track %d", i),
			URI:     fmt.Sprintf("spotify:track:t%d", i),
			Artists: []artistObj{{Name: "Artist"}},
		}
	}

	resp := response{
		Tracks:    tracksPage{Items: tracks, Total: total},
		Artists:   emptyPage{Total: 0},
		Albums:    emptyPage{Total: 0},
		Playlists: emptyPage{Total: 0},
	}
	data, _ := json.Marshal(resp)
	return data
}

// searchServer creates a test server that returns searchJSONResponse(n, total) for all requests.
func searchServer(t *testing.T, n, total int) *httptest.Server {
	t.Helper()
	data := searchJSONResponse(n, total)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
}

// searchServerCapture creates a test server that captures query params and returns a response.
func searchServerCapture(t *testing.T, n, total int, capture func(r *http.Request)) *httptest.Server {
	t.Helper()
	data := searchJSONResponse(n, total)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
}

// newSearchApp creates a test *app.App wired to the given search server URL.
func newSearchApp(t *testing.T, serverURL string) *app.App {
	t.Helper()
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(serverURL, "test-token"))
	return a
}

// driveSearchBatch drives the chain-through-Update batch. It fires the initial batch
// command, then feeds each resulting SearchPageLoadedMsg back into app.Update until
// no more commands are produced (or the batch ends). Returns the final app state.
// maxIter is an upper bound to prevent infinite loops.
func driveSearchBatch(t *testing.T, a *app.App, initCmd tea.Cmd, maxIter int) *app.App {
	t.Helper()
	cmd := initCmd
	for i := 0; i < maxIter && cmd != nil; i++ {
		msg := executeFirstCmd(cmd)
		if msg == nil {
			break
		}
		pageMsg, ok := msg.(panes.SearchPageLoadedMsg)
		if !ok {
			break
		}
		var nextCmd tea.Cmd
		model, c := a.Update(pageMsg)
		a = model.(*app.App)
		nextCmd = c
		cmd = nextCmd
	}
	return a
}

// --- Scenario 1: Basic search flow ---

// TestSearchIntegration_BasicFlow verifies the full search lifecycle:
// open overlay → SearchRequestMsg → batch fires → SearchPageLoadedMsg → store populated.
func TestSearchIntegration_BasicFlow(t *testing.T) {
	srv := searchServer(t, 10, 50) // 10 items per page, total=50
	defer srv.Close()

	a := newSearchApp(t, srv.URL)

	// Step 1: Open search overlay (simulates pressing '/').
	model, initCmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = model.(*app.App)
	require.True(t, a.SearchOpen(), "pressing '/' should open the search overlay")

	// Step 2: The SearchClearedMsg from Init() should clear the store.
	// Drive the init batch to process SearchClearedMsg.
	if initCmd != nil {
		msg := initCmd()
		if batchMsg, ok := msg.(tea.BatchMsg); ok {
			for _, subCmd := range batchMsg {
				if subCmd == nil {
					continue
				}
				subMsg := subCmd()
				if _, cleared := subMsg.(panes.SearchClearedMsg); cleared {
					a.Update(subMsg) //nolint:errcheck
				}
			}
		}
	}

	// Step 3: Simulate a debounce-fired SearchRequestMsg (the overlay emits this).
	model2, batchCmd := a.Update(panes.SearchRequestMsg{Query: "rock"})
	a = model2.(*app.App)
	require.NotNil(t, batchCmd, "SearchRequestMsg should return a batch command")

	// Verify store reflects the new query.
	assert.Equal(t, "rock", a.Store().SearchQuery(), "store query should be updated to 'rock'")
	assert.True(t, a.Store().SearchLoading(), "store should show loading after SearchRequestMsg")

	// Step 4: Drive one page of results.
	a = driveSearchBatch(t, a, batchCmd, 1)

	// Step 5: Store should now have tracks from the first page.
	tracks := a.Store().SearchTracks().Items
	assert.NotEmpty(t, tracks, "store should have tracks after page loaded")
	assert.Equal(t, 50, a.Store().SearchTracks().Total, "store total should match API response")
}

// --- Scenario 2: Tab switching ---

// TestSearchIntegration_TabSwitching verifies that a tab change clears current results
// and re-fires the search with the new type filter.
func TestSearchIntegration_TabSwitching(t *testing.T) {
	var lastTypeParam string
	srv := searchServerCapture(t, 5, 20, func(r *http.Request) {
		lastTypeParam = r.URL.Query().Get("type")
	})
	defer srv.Close()

	a := newSearchApp(t, srv.URL)

	// Set up an initial search for "kk" on "All" tab.
	a.Store().SetSearchQuery("kk")
	a.Store().AppendSearchTracks([]domain.Track{
		{ID: "t1", Name: "KK Track", URI: "spotify:track:t1"},
	}, 10)

	// Simulate tab switch to "Songs" — overlay emits SearchTabChangedMsg.
	model, cmd := a.Update(panes.SearchTabChangedMsg{
		Types: []string{"track"},
		Query: "kk",
	})
	a = model.(*app.App)

	// Store results should be cleared.
	assert.Empty(t, a.Store().SearchTracks().Items, "tab switch should clear previous results")
	assert.True(t, a.Store().SearchLoading(), "tab switch should set loading")
	require.NotNil(t, cmd, "tab switch should fire a new search batch command")

	// Drive the first page.
	a = driveSearchBatch(t, a, cmd, 1)

	// The API should have been called with type=track.
	assert.Equal(t, "track", lastTypeParam, "tab switch should fire API with type=track")
}

// --- Scenario 3: Prefix flow ---

// TestSearchIntegration_PrefixFlow verifies that when the overlay emits SearchRequestMsg
// with Types=["track"] (from ":songs kk" prefix), the app fires the API with type=track
// and only "kk" (not ":songs kk") is sent as the search query.
func TestSearchIntegration_PrefixFlow(t *testing.T) {
	var capturedQuery, capturedType string
	srv := searchServerCapture(t, 5, 10, func(r *http.Request) {
		capturedQuery = r.URL.Query().Get("q")
		capturedType = r.URL.Query().Get("type")
	})
	defer srv.Close()

	a := newSearchApp(t, srv.URL)

	// Simulate the overlay sending SearchRequestMsg with prefix-derived Types.
	// The overlay strips ":songs " and sends query="kk", Types=["track"].
	model, cmd := a.Update(panes.SearchRequestMsg{
		Query: "kk",
		Types: []string{"track"},
	})
	a = model.(*app.App)
	require.NotNil(t, cmd, "prefix SearchRequestMsg should dispatch a batch command")

	// Drive the first page.
	a = driveSearchBatch(t, a, cmd, 1)

	// Verify the API was called with the clean query and correct type.
	assert.Equal(t, "kk", capturedQuery, "API should receive clean query 'kk', not ':songs kk'")
	assert.Equal(t, "track", capturedType, "API should receive type=track from :songs prefix")

	// Store active type should reflect the locked prefix type.
	// (App sets this in the SearchTabChangedMsg handler, not SearchRequestMsg,
	// so for a direct SearchRequestMsg the store type is unchanged. Verify store has tracks.)
	assert.NotEmpty(t, a.Store().SearchTracks().Items, "store should have tracks from prefix search")
}

// --- Scenario 4: Prefetch trigger ---

// TestSearchIntegration_PrefetchTrigger verifies that when the overlay scrolls past
// 60% of loaded results, SearchPrefetchMsg fires and the app dispatches the next batch.
func TestSearchIntegration_PrefetchTrigger(t *testing.T) {
	srv := searchServer(t, 10, 200) // 10 items per page, lots more available
	defer srv.Close()

	a := newSearchApp(t, srv.URL)

	// Pre-populate store: 50 items loaded, total=200 (plenty more).
	a.Store().SetSearchQuery("jazz")
	a.Store().AppendSearchTracks(makeDomainTracks(50), 200)

	// Simulate the overlay emitting SearchPrefetchMsg when cursor is at item 31
	// (past the 60% threshold of 50 items = item 30).
	model, cmd := a.Update(panes.SearchPrefetchMsg{
		Query:      "jazz",
		Types:      []string{"track"},
		NextOffset: 50, // the next offset to fetch
	})
	a = model.(*app.App)

	require.NotNil(t, cmd, "SearchPrefetchMsg should dispatch a batch command for next page")
	assert.True(t, a.Store().SearchLoading(), "store should show loading after prefetch triggered")

	// Drive one page from the prefetch.
	a = driveSearchBatch(t, a, cmd, 1)

	// Store should have more tracks now (50 original + 10 from prefetch).
	tracks := a.Store().SearchTracks().Items
	assert.Greater(t, len(tracks), 50, "prefetch should append more tracks to the store")
}

// --- Scenario 5: Stale result discard ---

// TestSearchIntegration_StaleResultDiscard verifies that results for a superseded
// query are discarded without updating the store.
func TestSearchIntegration_StaleResultDiscard(t *testing.T) {
	srv := searchServer(t, 5, 5) // small response
	defer srv.Close()

	a := newSearchApp(t, srv.URL)

	// Set current query to "jazz" in the store.
	a.Store().SetSearchQuery("jazz")

	// Simulate results arriving for the OLD query "kk".
	staleMsg := panes.SearchPageLoadedMsg{
		Query:  "kk",  // stale — doesn't match store's "jazz"
		Offset: 0,
		Results: &panes.SearchResultData{
			Tracks: []panes.SearchTrackItem{
				{URI: "spotify:track:old", Name: "Old Track", Artist: "Old Artist"},
			},
			TracksTotal: 1,
		},
	}

	model, _ := a.Update(staleMsg)
	a = model.(*app.App)

	// Stale results should NOT be written to the store.
	assert.Empty(t, a.Store().SearchTracks().Items,
		"stale results (query='kk') should be discarded when store query is 'jazz'")

	// Loading should be cleared after stale discard.
	assert.False(t, a.Store().SearchLoading(), "loading should be cleared after stale discard")
}

// --- Scenario 6: Close and reopen ---

// TestSearchIntegration_CloseAndReopen verifies that closing and reopening the search
// overlay clears the previous session's results and resets the overlay's input.
func TestSearchIntegration_CloseAndReopen(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Populate the store with a previous session's results.
	a.Store().SetSearchQuery("previous query")
	a.Store().AppendSearchTracks([]domain.Track{
		{ID: "t1", Name: "Previous Track", URI: "spotify:track:prev"},
	}, 1)

	// Open the search overlay.
	model, initCmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = model.(*app.App)
	require.True(t, a.SearchOpen(), "pressing '/' should open the search overlay")

	// Process SearchClearedMsg from Init() — it should clear the store.
	if initCmd != nil {
		msg := initCmd()
		if batchMsg, ok := msg.(tea.BatchMsg); ok {
			for _, subCmd := range batchMsg {
				if subCmd == nil {
					continue
				}
				subMsg := subCmd()
				if _, cleared := subMsg.(panes.SearchClearedMsg); cleared {
					model, _ = a.Update(subMsg)
					a = model.(*app.App)
				}
			}
		}
	}

	// After opening and processing clear, store should be reset.
	assert.Equal(t, "", a.Store().SearchQuery(),
		"store query should be empty after overlay opens (clear-on-open)")
	assert.Empty(t, a.Store().SearchTracks().Items,
		"store tracks should be cleared after overlay opens")

	// Close the overlay.
	model, _ = a.Update(panes.SearchClosedMsg{})
	a = model.(*app.App)
	assert.False(t, a.SearchOpen(), "SearchClosedMsg should close the overlay")

	// Reopen.
	model, initCmd2 := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = model.(*app.App)
	require.True(t, a.SearchOpen(), "pressing '/' again should reopen the overlay")
	require.NotNil(t, initCmd2, "reopening should return an init command (with clear)")

	// The search overlay should be in a clean state — confirmed by Init() emitting
	// SearchClearedMsg again (already tested in unit test, confirmed here by presence).
	msg := initCmd2()
	batchMsg, ok := msg.(tea.BatchMsg)
	require.True(t, ok, "Init() should return a BatchMsg on reopen")

	var foundClear bool
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		if _, cleared := subCmd().(panes.SearchClearedMsg); cleared {
			foundClear = true
		}
	}
	assert.True(t, foundClear, "reopening the overlay should emit SearchClearedMsg again")
}
