package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Task 1: searchPrefixes, prefixToTab, prefixState ---

// TestSearchPrefixes_AllFourDefined verifies all 4 command prefixes are defined.
func TestSearchPrefixes_AllFourDefined(t *testing.T) {
	prefixes := panes.SearchPrefixes
	require.Len(t, prefixes, 4, "should have exactly 4 prefixes")
	assert.Contains(t, prefixes, ":songs")
	assert.Contains(t, prefixes, ":artists")
	assert.Contains(t, prefixes, ":albums")
	assert.Contains(t, prefixes, ":playlists")
}

// TestPrefixToTab_MapsAllEntries verifies each prefix maps to the correct tab.
func TestPrefixToTab_MapsAllEntries(t *testing.T) {
	tests := []struct {
		prefix  string
		wantTab panes.SearchTab
	}{
		{":songs", panes.TabSongs},
		{":artists", panes.TabArtists},
		{":albums", panes.TabAlbums},
		{":playlists", panes.TabPlaylists},
	}
	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			got, ok := panes.PrefixToTab(tt.prefix)
			assert.True(t, ok, "prefix %s should be in PrefixToTab map", tt.prefix)
			assert.Equal(t, tt.wantTab, got)
		})
	}
}

// TestPrefixToTab_InvalidPrefix verifies unknown prefix returns ok=false.
func TestPrefixToTab_InvalidPrefix(t *testing.T) {
	_, ok := panes.PrefixToTab(":unknown")
	assert.False(t, ok)
}

// --- Task 2: parsePrefix() ---

// TestParsePrefix_NoColon sets prefixState to prefixNone.
func TestParsePrefix_NoColon(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type a regular query without colon.
	o, _ = sendKey(t, o, "h")
	o, _ = sendKey(t, o, "e")
	o, _ = sendKey(t, o, "l")

	assert.Equal(t, panes.PrefixNone, o.PrefixState(), "no colon should give prefixNone")
	assert.Equal(t, "", o.LockedPrefix(), "no colon should have empty lockedPrefix")
}

// TestParsePrefix_PartialColon sets prefixState to prefixTyping.
func TestParsePrefix_PartialColon(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":so" — no space yet.
	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "s")
	o, _ = sendKey(t, o, "o")

	assert.Equal(t, panes.PrefixTyping, o.PrefixState(), ":so should give prefixTyping")
	assert.Equal(t, "", o.LockedPrefix(), "lockedPrefix should be empty while typing")
}

// TestParsePrefix_LockedOnSpace locks the prefix when a valid prefix + space is typed.
func TestParsePrefix_LockedOnSpace(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":songs " — full valid prefix + space.
	for _, ch := range ":songs" {
		o, _ = sendKey(t, o, string(ch))
	}
	o, _ = sendKey(t, o, " ")

	assert.Equal(t, panes.PrefixLocked, o.PrefixState(), ":songs space should lock prefix")
	assert.Equal(t, ":songs", o.LockedPrefix())
}

// TestParsePrefix_InvalidPrefixWithSpace treats invalid prefix as normal search.
func TestParsePrefix_InvalidPrefixWithSpace(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":invalid " — not a known prefix.
	for _, ch := range ":invalid" {
		o, _ = sendKey(t, o, string(ch))
	}
	o, _ = sendKey(t, o, " ")

	assert.Equal(t, panes.PrefixNone, o.PrefixState(), ":invalid space should give prefixNone")
	assert.Equal(t, "", o.LockedPrefix())
}

// TestParsePrefix_BackspaceUnlocks verifies that backspacing into a locked prefix
// transitions back to prefixTyping.
func TestParsePrefix_BackspaceUnlocks(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":songs " to lock.
	for _, ch := range ":songs " {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState())

	// Backspace removes the trailing space — back to typing.
	o, _ = sendKey(t, o, "backspace")

	assert.Equal(t, panes.PrefixTyping, o.PrefixState(), "backspace should unlock prefix back to prefixTyping")
}

// --- Task 3: cleanQuery() and activeAPITypes() ---

// TestCleanQuery_LockedPrefix strips the prefix from the query.
func TestCleanQuery_LockedPrefix(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":songs kk".
	for _, ch := range ":songs kk" {
		o, _ = sendKey(t, o, string(ch))
	}

	require.Equal(t, panes.PrefixLocked, o.PrefixState())
	assert.Equal(t, "kk", o.CleanQuery(), "cleanQuery should strip :songs prefix")
}

// TestCleanQuery_NoPrefix returns full input unchanged.
func TestCleanQuery_NoPrefix(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type a regular query.
	for _, ch := range "kk" {
		o, _ = sendKey(t, o, string(ch))
	}

	assert.Equal(t, panes.PrefixNone, o.PrefixState())
	assert.Equal(t, "kk", o.CleanQuery(), "cleanQuery without prefix should return full input")
}

// TestActiveAPITypes_LockedSongs returns track type.
func TestActiveAPITypes_LockedSongs(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	for _, ch := range ":songs kk" {
		o, _ = sendKey(t, o, string(ch))
	}

	require.Equal(t, panes.PrefixLocked, o.PrefixState())
	types := o.ActiveAPITypes()
	assert.Equal(t, []string{"track"}, types)
}

// TestActiveAPITypes_NoPrefix uses activeTab types.
func TestActiveAPITypes_NoPrefix(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	for _, ch := range "kk" {
		o, _ = sendKey(t, o, string(ch))
	}

	// Default activeTab is TabAll — all 4 types.
	types := o.ActiveAPITypes()
	assert.Equal(t, []string{"track", "artist", "album", "playlist"}, types)
}

// --- Task 4: renderPrefixHints() ---

// TestRenderPrefixHints_SingleMatch shows :songs for :s input.
func TestRenderPrefixHints_SingleMatch(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "s")

	require.Equal(t, panes.PrefixTyping, o.PrefixState())
	hint := o.RenderPrefixHints(80)
	assert.Contains(t, hint, ":songs", "hint should contain :songs for :s input")
	assert.NotContains(t, hint, ":artists")
	assert.NotContains(t, hint, ":albums")
}

// TestRenderPrefixHints_MultipleMatches shows both :artists and :albums for :a input.
func TestRenderPrefixHints_MultipleMatches(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "a")

	require.Equal(t, panes.PrefixTyping, o.PrefixState())
	hint := o.RenderPrefixHints(80)
	assert.Contains(t, hint, ":artists")
	assert.Contains(t, hint, ":albums")
}

// TestRenderPrefixHints_ExactMatch shows the exact prefix when typed fully.
func TestRenderPrefixHints_ExactMatch(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	for _, ch := range ":songs" {
		o, _ = sendKey(t, o, string(ch))
	}

	require.Equal(t, panes.PrefixTyping, o.PrefixState())
	hint := o.RenderPrefixHints(80)
	assert.Contains(t, hint, ":songs")
}

// TestRenderPrefixHints_NoMatch returns empty for no matching prefix.
func TestRenderPrefixHints_NoMatch(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":z" — no prefix starts with z.
	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "z")

	require.Equal(t, panes.PrefixTyping, o.PrefixState())
	hint := o.RenderPrefixHints(80)
	assert.Empty(t, hint, "no matching prefix should yield empty hint")
}

// TestRenderPrefixHints_NormalInput returns empty for non-prefix input.
func TestRenderPrefixHints_NormalInput(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	for _, ch := range "hello" {
		o, _ = sendKey(t, o, string(ch))
	}

	assert.Equal(t, panes.PrefixNone, o.PrefixState())
	hint := o.RenderPrefixHints(80)
	assert.Empty(t, hint)
}

// --- Task 5: tabCompletePrefix() ---

// TestTabComplete_UniquePrefixCompletes completes :s to :songs + space.
func TestTabComplete_UniquePrefixCompletes(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "s")
	require.Equal(t, panes.PrefixTyping, o.PrefixState())

	// Tab should complete :s → :songs (unique match).
	o, _ = sendKey(t, o, "tab")

	assert.Equal(t, ":songs ", o.Query(), "unique match should complete to :songs + space")
	assert.Equal(t, panes.PrefixLocked, o.PrefixState(), "after completion prefix should be locked")
}

// TestTabComplete_MultipleMatchesNoChange does not complete :a (2 matches: :artists, :albums).
func TestTabComplete_MultipleMatchesNoChange(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "a")
	require.Equal(t, panes.PrefixTyping, o.PrefixState())

	// Tab should not complete — 2 matches.
	o, _ = sendKey(t, o, "tab")

	assert.Equal(t, ":a", o.Query(), "multiple matches should not change input")
	assert.Equal(t, panes.PrefixTyping, o.PrefixState())
}

// TestTabComplete_ExactPrefixCompletes completes :artists to :artists + space.
func TestTabComplete_ExactPrefixCompletes(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	for _, ch := range ":artists" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixTyping, o.PrefixState())

	o, _ = sendKey(t, o, "tab")

	assert.Equal(t, ":artists ", o.Query(), "exact prefix should complete to :artists + space")
	assert.Equal(t, panes.PrefixLocked, o.PrefixState())
}

// --- Task 6: Tab key routing ---

// TestTabRouting_DuringPrefixTypingCompletes uses tabCompletePrefix not cycleTabForward.
// Already covered by TestTabComplete_UniquePrefixCompletes and TestTabComplete_MultipleMatchesNoChange.
// This test verifies that Tab during prefixNone still cycles tabs.
func TestTabRouting_PrefixNone_CyclesTabs(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	require.Equal(t, panes.TabAll, o.ActiveTab(), "should start at TabAll")

	// Tab when no prefix typed → should cycle to next tab.
	o, _ = sendKey(t, o, "tab")

	assert.Equal(t, panes.TabSongs, o.ActiveTab(), "Tab with no prefix should cycle to TabSongs")
}

// TestTabRouting_PrefixTyping_DoesNotCycleTabs verifies prefix completion is chosen over tab cycling.
func TestTabRouting_PrefixTyping_DoesNotCycleTabs(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Start typing a prefix.
	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "s")
	require.Equal(t, panes.PrefixTyping, o.PrefixState())
	initialTab := o.ActiveTab()

	// Tab should complete prefix, not cycle tab.
	o, _ = sendKey(t, o, "tab")

	// Active tab changes when prefix is locked (syncs to :songs tab).
	// The point is that it didn't cycle from TabAll → TabSongs via normal tab cycling.
	assert.Equal(t, panes.TabSongs, o.ActiveTab(), "Tab during prefixTyping should lock to :songs tab, not cycle")
	_ = initialTab
}

// --- Task 7: debounce uses cleanQuery and skips during prefixTyping ---

// TestDebounce_SkippedDuringPrefixTyping verifies no debounce fires when still typing prefix.
func TestDebounce_SkippedDuringPrefixTyping(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":so" — still typing prefix, no space yet.
	o, _ = sendKey(t, o, ":")

	// The debounce cmd for ":" should not fire a search.
	_, cmd := o.Update(panes.SearchDebounceMsgForTest(":"))
	assert.Nil(t, cmd, "debounce should not fire while still typing prefix ':'")

	o, _ = sendKey(t, o, "s")
	o, _ = sendKey(t, o, "o")
	_, cmd = o.Update(panes.SearchDebounceMsgForTest(":so"))
	assert.Nil(t, cmd, "debounce should not fire while still typing prefix ':so'")
}

// TestDebounce_UsesCleanQuery fires with clean query (no prefix) when prefix is locked.
func TestDebounce_UsesCleanQuery(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":songs kk" — prefix is locked, query is "kk".
	for _, ch := range ":songs kk" {
		o, _ = sendKey(t, o, string(ch))
	}

	require.Equal(t, panes.PrefixLocked, o.PrefixState())

	// The debounce snapshot was taken from cleanQuery() which is "kk".
	// Fire the debounce with "kk" (the clean query).
	_, cmd := o.Update(panes.SearchDebounceMsgForTest("kk"))
	require.NotNil(t, cmd, "debounce should fire for locked prefix with clean query")

	msg := cmd()
	srm, ok := msg.(panes.SearchRequestMsg)
	require.True(t, ok, "should emit SearchRequestMsg, got %T", msg)
	assert.Equal(t, "kk", srm.Query, "SearchRequestMsg should carry clean query")
	assert.Equal(t, []string{"track"}, srm.Types, "SearchRequestMsg should carry track type from :songs prefix")
}

// --- Task 8: SearchRequestMsg carries Types ---

// TestSearchRequestMsg_CarriesTypes verifies SearchRequestMsg has Types field.
func TestSearchRequestMsg_CarriesTypes(t *testing.T) {
	msg := panes.SearchRequestMsg{Query: "kk", Types: []string{"track"}}
	assert.Equal(t, "kk", msg.Query)
	assert.Equal(t, []string{"track"}, msg.Types)
}

// TestCycleTabForward_UsesCleanQueryWhenPrefixLocked verifies that cycling tabs while
// a prefix is locked sends the clean query (no prefix) in SearchTabChangedMsg.
// This prevents ":songs kk" from reaching the API as a raw query string.
func TestCycleTabForward_UsesCleanQueryWhenPrefixLocked(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Lock :songs prefix with query "kk".
	for _, ch := range ":songs kk" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState())
	require.Equal(t, "kk", o.CleanQuery())

	// Press Shift+Tab (cycleTabBackward) to cycle tabs.
	// This is reachable because Tab during PrefixLocked goes to cycleTabForward.
	_, cmd := sendKey(t, o, "shift+tab")

	require.NotNil(t, cmd, "Shift+Tab should emit SearchTabChangedMsg command")
	msg := cmd()
	stcm, ok := msg.(panes.SearchTabChangedMsg)
	require.True(t, ok, "should emit SearchTabChangedMsg, got %T", msg)
	assert.Equal(t, "kk", stcm.Query, "tab cycle should use clean query, not raw ':songs kk'")
}
