package app_test

// search_cancellation_test.go — Tests for Story 100: App context cancellation and staleness keys.
//
// Tests verify:
//   - NewApp() searchCancel is safe to call immediately (no panic)
//   - SearchRequestMsg handler cancels prior ctx, records staleness keys, sets loading=true
//   - SearchPageLoadedMsg: staleness check discards mismatched query/page; error branch toasts
//   - SearchPageLoadedMsg: stale error (including context.Canceled) is silently discarded
//   - closeSearch() cancels, clears all four fields
//   - openSearch() resets cancel func, calls Reset()+Init()

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewApp_SearchCancelIsNotNil verifies that a freshly constructed App has a
// non-nil searchCancel func and calling it does not panic.
func TestNewApp_SearchCancelIsNotNil(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	require.NotNil(t, a)

	// SearchLoading should be false initially.
	assert.False(t, a.SearchLoading(), "searchLoading should be false after construction")
	// SearchQuery should be empty initially.
	assert.Empty(t, a.SearchQuery(), "searchQuery should be empty after construction")
	// SearchPage should be 0 initially.
	assert.Equal(t, 0, a.SearchPage(), "searchPage should be 0 after construction")
	// Calling searchCancel should not panic.
	assert.NotPanics(t, func() { a.CallSearchCancel() }, "calling searchCancel right after construction must not panic")
}

// TestSearchRequestMsg_CancelsPriorCtx verifies that sending a second SearchRequestMsg
// cancels the context from the first one before dispatching the new fetch.
func TestSearchRequestMsg_CancelsPriorCtx(t *testing.T) {
	// A server that blocks until the test signals it — lets us hold an in-flight request.
	unblock := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wait until unblocked or context cancelled.
		select {
		case <-unblock:
		case <-r.Context().Done():
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":0},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()
	defer close(unblock)

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	// Send first request — get the context for the first in-flight request.
	model, cmd1 := a.Update(panes.SearchRequestMsg{Query: "jazz", Types: []string{"track"}, Page: 1})
	a = model.(*app.App)
	require.NotNil(t, cmd1, "first SearchRequestMsg should produce a command")

	// Verify staleness keys were set after the first request.
	assert.Equal(t, "jazz", a.SearchQuery(), "searchQuery should be set after first SearchRequestMsg")
	assert.Equal(t, 1, a.SearchPage(), "searchPage should be 1 after first SearchRequestMsg")
	assert.True(t, a.SearchLoading(), "searchLoading should be true after SearchRequestMsg")

	// Capture the context from the first request.
	firstCtx := a.SearchCancelCtx()
	require.NotNil(t, firstCtx, "first context should be non-nil")

	// The first context should NOT be cancelled yet.
	select {
	case <-firstCtx.Done():
		t.Fatal("first context should NOT be cancelled before second request arrives")
	default:
		// good
	}

	// Send second request — this should cancel the first context.
	model2, cmd2 := a.Update(panes.SearchRequestMsg{Query: "rock", Types: []string{"track"}, Page: 1})
	a = model2.(*app.App)
	require.NotNil(t, cmd2, "second SearchRequestMsg should produce a command")

	// The first context should now be cancelled.
	select {
	case <-firstCtx.Done():
		// good — context was cancelled
	default:
		t.Fatal("first context should be cancelled after second SearchRequestMsg")
	}

	// The staleness keys should now reflect the second query.
	assert.Equal(t, "rock", a.SearchQuery(), "searchQuery should be updated to second query")
	assert.True(t, a.SearchLoading(), "searchLoading should remain true after second SearchRequestMsg")
}

// TestSearchPageLoadedMsg_StalenessTable verifies the four staleness scenarios.
func TestSearchPageLoadedMsg_StalenessTable(t *testing.T) {
	someErr := errors.New("network error")

	tests := []struct {
		name        string
		msgQuery    string
		appQuery    string
		msgPage     int
		appPage     int
		msgErr      error
		wantDiscard bool // true = handler returns nil cmd (stale result discarded)
		wantToast   bool // true = toast command should be dispatched
		wantLoading bool // expected a.searchLoading after handler
	}{
		{
			name:        "matching query and page — forward to overlay",
			msgQuery:    "jazz",
			appQuery:    "jazz",
			msgPage:     1,
			appPage:     1,
			msgErr:      nil,
			wantDiscard: false,
			wantToast:   false,
			wantLoading: false,
		},
		{
			name:        "query mismatch — discard",
			msgQuery:    "jazz",
			appQuery:    "rock",
			msgPage:     1,
			appPage:     1,
			msgErr:      nil,
			wantDiscard: true,
			wantToast:   false,
			wantLoading: true, // loading remains true (stale result ignored)
		},
		{
			name:        "page mismatch — discard",
			msgQuery:    "jazz",
			appQuery:    "jazz",
			msgPage:     2,
			appPage:     1,
			msgErr:      nil,
			wantDiscard: true,
			wantToast:   false,
			wantLoading: true, // loading remains true
		},
		{
			name:        "error — toast + forward to overlay to clear spinners",
			msgQuery:    "jazz",
			appQuery:    "jazz",
			msgPage:     1,
			appPage:     1,
			msgErr:      someErr,
			wantDiscard: false,
			wantToast:   true,
			wantLoading: false, // error clears loading flag
		},
		{
			// Stale error: the user typed a new query while the old request was in-flight.
			// The old request may return context.Canceled or any network error — it must be
			// silently discarded rather than showing a spurious toast.
			name:        "stale error with query mismatch — discard silently, no toast",
			msgQuery:    "jazz",
			appQuery:    "rock", // user already moved on to "rock"
			msgPage:     1,
			appPage:     1,
			msgErr:      someErr,
			wantDiscard: true,
			wantToast:   false,
			wantLoading: true, // loading remains true for the current "rock" request
		},
		{
			// Context-cancelled error from a prior query: the most common stale-error
			// scenario triggered by Story 100's cancellation mechanism. Must NOT toast.
			name:        "stale context-cancelled error — discard silently, no toast",
			msgQuery:    "jazz",
			appQuery:    "rock",
			msgPage:     1,
			appPage:     1,
			msgErr:      context.Canceled,
			wantDiscard: true,
			wantToast:   false,
			wantLoading: true, // loading remains true for the ongoing "rock" request
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			a := app.New(cfg, app.AppOptions{})

			// Set up the app's internal staleness keys to simulate an in-flight search.
			a.SetSearchSession(tt.appQuery, tt.appPage, true)

			msg := panes.SearchPageLoadedMsg{
				Query: tt.msgQuery,
				Page:  tt.msgPage,
				Err:   tt.msgErr,
			}
			model, cmd := a.Update(msg)
			a = model.(*app.App)

			assert.Equal(t, tt.wantLoading, a.SearchLoading(),
				"searchLoading mismatch for %q", tt.name)

			if tt.wantDiscard {
				assert.Nil(t, cmd, "stale result should produce nil cmd")
			}
			if tt.wantToast {
				assert.NotNil(t, cmd, "error case should produce a cmd (toast)")
			}
		})
	}
}

// TestCloseSearch_CancelsAndClearsFields verifies that closeSearch calls
// searchCancel and resets all four fields.
func TestCloseSearch_CancelsAndClearsFields(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open search to set searchOpen=true.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = model.(*app.App)
	require.True(t, a.SearchOpen(), "search should be open after '/' key")

	// Simulate an in-flight search by setting fields and capturing the context.
	a.SetSearchSession("jazz", 1, true)

	// Now send a SearchRequestMsg to get a real cancellable context on the app.
	// This replaces the no-op cancel with a real one.
	model2, _ := a.Update(panes.SearchRequestMsg{Query: "jazz", Types: []string{"track"}, Page: 1})
	a = model2.(*app.App)
	ctx := a.SearchCancelCtx()
	require.NotNil(t, ctx)

	// Close the search overlay via SearchClosedMsg.
	model3, _ := a.Update(panes.SearchClosedMsg{})
	a = model3.(*app.App)

	assert.False(t, a.SearchOpen(), "searchOpen should be false after close")
	assert.Empty(t, a.SearchQuery(), "searchQuery should be cleared after close")
	assert.Equal(t, 0, a.SearchPage(), "searchPage should be 0 after close")
	assert.False(t, a.SearchLoading(), "searchLoading should be false after close")

	// The context captured before close should now be cancelled.
	select {
	case <-ctx.Done():
		// good — context was cancelled
	default:
		t.Fatal("context should be cancelled after closeSearch")
	}
}

// TestOpenSearch_ResetsCancelFunc verifies that openSearch sets a fresh no-op cancel func.
func TestOpenSearch_ResetsCancelFunc(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Step 1: open search.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = model.(*app.App)
	require.True(t, a.SearchOpen())

	// Step 2: send a real request to get a cancellable context.
	model2, _ := a.Update(panes.SearchRequestMsg{Query: "jazz", Types: []string{"track"}, Page: 1})
	a = model2.(*app.App)

	// Step 3: close search (clears state, cancels ctx).
	model3, _ := a.Update(panes.SearchClosedMsg{})
	a = model3.(*app.App)
	require.False(t, a.SearchOpen())

	// Step 4: reopen search.
	model4, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	a = model4.(*app.App)
	require.True(t, a.SearchOpen())

	// After reopen: searchLoading should be false and calling cancel should not panic.
	assert.False(t, a.SearchLoading(), "searchLoading should be false after reopen")
	assert.Empty(t, a.SearchQuery(), "searchQuery should be empty after reopen")
	assert.NotPanics(t, func() { a.CallSearchCancel() }, "searchCancel should be a safe no-op after reopen")
}

// Verify context is a valid cancellable context (not context.Background directly).
func TestSearchRequestMsg_ContextIsCancellable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tracks":{"items":[],"total":0},"artists":{"items":[],"total":0},"albums":{"items":[],"total":0},"playlists":{"items":[],"total":0}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetSearch(api.NewSearchClient(srv.URL, "test-token"))

	model, _ := a.Update(panes.SearchRequestMsg{Query: "test", Types: []string{"track"}, Page: 1})
	a = model.(*app.App)

	ctx := a.SearchCancelCtx()
	require.NotNil(t, ctx)

	// The context should be a cancellable context (not already done).
	select {
	case <-ctx.Done():
		t.Fatal("context should NOT be done immediately after SearchRequestMsg")
	default:
		// good — context is live
	}

	// Calling cancel should cancel the context.
	a.CallSearchCancel()
	select {
	case <-ctx.Done():
		// good
	default:
		t.Fatal("context should be done after CallSearchCancel")
	}
}

// TestSearchPageLoadedMsg_StaleContextCancelledDoesNotToastOrClearLoading verifies
// the exact bug described in Story 100 PR #126:
//
// When the user types a new query ("rock") while the prior request ("jazz") is
// still in-flight, the app cancels the "jazz" context. The "jazz" goroutine then
// delivers a SearchPageLoadedMsg with Err=context.Canceled and Query="jazz".
//
// This message is stale (Query "jazz" != app's current searchQuery "rock").
// The handler must:
//   - NOT emit a toast ("Search failed: context canceled")
//   - NOT clear searchLoading (the "rock" request is still in-flight)
//   - return a nil command
func TestSearchPageLoadedMsg_StaleContextCancelledDoesNotToastOrClearLoading(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Simulate the user having moved on to "rock" (page 1) as the current query.
	a.SetSearchSession("rock", 1, true)

	// The old "jazz" request arrives back with context.Canceled.
	msg := panes.SearchPageLoadedMsg{
		Query: "jazz",
		Page:  1,
		Err:   context.Canceled,
	}
	model, cmd := a.Update(msg)
	a = model.(*app.App)

	// searchLoading must remain true — the "rock" request is still in-flight.
	assert.True(t, a.SearchLoading(),
		"searchLoading must remain true when a stale context.Canceled error arrives")

	// No command should be produced — no toast, no overlay update.
	assert.Nil(t, cmd,
		"stale context.Canceled error must produce nil cmd (no toast emitted)")
}
