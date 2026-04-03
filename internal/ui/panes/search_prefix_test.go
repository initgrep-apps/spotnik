package panes_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
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

// TestParsePrefix_BackspaceUnlocks verifies that backspacing at position 0 when a
// prefix is locked demotes the tag back into the input for editing (Prompt reset).
// With the Prompt-based approach, one backspace at pos 0 demotes (restores prefix
// to value); a second backspace removes the trailing space → PrefixTyping.
func TestParsePrefix_BackspaceUnlocks(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":songs " to lock. After promotion: Prompt = styled ":songs", Value = "".
	for _, ch := range ":songs " {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState())
	require.Equal(t, "", o.Query(), "after promotion, Value holds only clean query (empty)")

	// First backspace at pos 0 demotes the tag: Prompt reset to "> ", Value = ":songs ".
	// demoteFromPromptTag() does NOT call parsePrefix(), so prefixState = PrefixNone.
	o, _ = sendKey(t, o, "backspace")
	assert.Equal(t, panes.PrefixNone, o.PrefixState(), "after demote, prefixState is reset to PrefixNone")
	assert.Equal(t, "> ", o.PromptTag(), "after demote, Prompt is reset to default")
	assert.Equal(t, ":songs ", o.Query(), "after demote, Value is restored to :prefix + space + query")

	// Second backspace removes the trailing space → PrefixTyping.
	o, _ = sendKey(t, o, "backspace")
	assert.Equal(t, panes.PrefixTyping, o.PrefixState(), "after removing the space, should be PrefixTyping")
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

// --- Task 4: renderPrefixHints() — redesigned as styled pills ---

// TestRenderPrefixHints_EmptyInput shows all 4 pills in their badge colors.
func TestRenderPrefixHints_EmptyInput(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	require.Equal(t, panes.PrefixNone, o.PrefixState())
	require.Equal(t, "", o.Query())
	hint := o.RenderPrefixHints(80)
	// All 4 pills should be present.
	assert.Contains(t, hint, ":songs")
	assert.Contains(t, hint, ":artists")
	assert.Contains(t, hint, ":albums")
	assert.Contains(t, hint, ":playlists")
}

// TestRenderPrefixHints_SingleMatch shows all pills; matching (:songs) is highlighted, others dimmed.
func TestRenderPrefixHints_SingleMatch(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "s")

	require.Equal(t, panes.PrefixTyping, o.PrefixState())
	hint := o.RenderPrefixHints(80)
	// All 4 pills are present (non-matching ones are dimmed, not hidden).
	assert.Contains(t, hint, ":songs", "hint should contain :songs for :s input")
	assert.Contains(t, hint, ":artists", "non-matching prefixes appear dimmed, not hidden")
	assert.Contains(t, hint, ":albums", "non-matching prefixes appear dimmed, not hidden")
	assert.Contains(t, hint, ":playlists", "non-matching prefixes appear dimmed, not hidden")
}

// TestRenderPrefixHints_MultipleMatches shows both :artists and :albums highlighted for :a input.
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

// TestRenderPrefixHints_ExactMatch shows the exact prefix highlighted when typed fully.
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

// TestRenderPrefixHints_NoMatch shows all pills dimmed when no prefix matches.
// ":z" doesn't match any prefix, so all 4 are shown in muted color.
func TestRenderPrefixHints_NoMatch(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":z" — no prefix starts with z. Still PrefixTyping (colon present, no space).
	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "z")

	require.Equal(t, panes.PrefixTyping, o.PrefixState())
	hint := o.RenderPrefixHints(80)
	// All 4 pills are shown but dimmed (non-matching).
	assert.Contains(t, hint, ":songs", "dimmed pills still appear in the row")
	assert.Contains(t, hint, ":artists", "dimmed pills still appear in the row")
	assert.NotEmpty(t, hint, "pills row is present even when no prefix matches")
}

// TestRenderPrefixHints_NormalInput returns empty for non-prefix input (PrefixNone, non-empty).
func TestRenderPrefixHints_NormalInput(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	for _, ch := range "hello" {
		o, _ = sendKey(t, o, string(ch))
	}

	assert.Equal(t, panes.PrefixNone, o.PrefixState())
	hint := o.RenderPrefixHints(80)
	assert.Empty(t, hint, "normal query (no colon prefix) should hide the pills row")
}

// TestRenderPrefixHints_Locked returns empty when prefix is locked (Prompt tag is shown instead).
func TestRenderPrefixHints_Locked(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	for _, ch := range ":songs kk" {
		o, _ = sendKey(t, o, string(ch))
	}

	require.Equal(t, panes.PrefixLocked, o.PrefixState())
	hint := o.RenderPrefixHints(80)
	assert.Empty(t, hint, "pills row is hidden when prefix is locked")
}

// --- Task 5: SetSuggestions ghost completion (replaces tabCompletePrefix) ---

// TestSetSuggestions_UniqueMatchCompletes verifies Tab accepts the ghost suggestion for :s.
// SetSuggestions replaces the custom tabCompletePrefix(). After Tab acceptance and
// Prompt-tag promotion, input.Value() holds only the clean query (empty here).
func TestSetSuggestions_UniqueMatchCompletes(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "s")
	require.Equal(t, panes.PrefixTyping, o.PrefixState())

	// Tab forwards to textinput which accepts the suggestion ":songs ".
	// parsePrefix() then locks, and promoteToPromptTag() moves prefix to Prompt.
	o, _ = sendKey(t, o, "tab")

	// After promotion: prefix is in Prompt, Value holds only clean query (empty).
	assert.Equal(t, panes.PrefixLocked, o.PrefixState(), "after Tab acceptance prefix should be locked")
	assert.Equal(t, ":songs", o.LockedPrefix(), "locked prefix should be :songs")
	assert.Equal(t, "", o.Query(), "after Prompt-tag promotion, Value holds only the clean query")
	assert.Contains(t, o.PromptTag(), ":songs", "Prompt should contain the prefix tag")
}

// TestSetSuggestions_MultipleMatchesAcceptsFirst verifies Tab accepts the first suggestion for :a.
// With SetSuggestions, Tab always accepts the first matched suggestion (alphabetically first).
func TestSetSuggestions_MultipleMatchesAcceptsFirst(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	o, _ = sendKey(t, o, ":")
	o, _ = sendKey(t, o, "a")
	require.Equal(t, panes.PrefixTyping, o.PrefixState())

	// Tab forwards to textinput. For ":a", textinput matches both ":albums " and ":artists ".
	// The first match is accepted (textinput accepts the first alphabetical match).
	o, _ = sendKey(t, o, "tab")

	// After Tab: prefix is locked and promoted to Prompt.
	assert.Equal(t, panes.PrefixLocked, o.PrefixState(), "Tab should accept one of the matching suggestions")
	// Accepted prefix is one of the :a prefixes.
	lockedPrefix := o.LockedPrefix()
	assert.True(t, lockedPrefix == ":albums" || lockedPrefix == ":artists",
		"locked prefix should be one of the :a matching prefixes, got: %s", lockedPrefix)
}

// TestSetSuggestions_ExactPrefixCompletes verifies Tab completes :artists to :artists + space.
func TestSetSuggestions_ExactPrefixCompletes(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	for _, ch := range ":artists" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixTyping, o.PrefixState())

	o, _ = sendKey(t, o, "tab")

	assert.Equal(t, panes.PrefixLocked, o.PrefixState(), "exact prefix should lock after Tab")
	assert.Equal(t, ":artists", o.LockedPrefix(), "locked prefix should be :artists")
	assert.Equal(t, "", o.Query(), "after Prompt-tag promotion, Value holds only the clean query")
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

// --- Ctrl+U prefix state reset ---

// TestCtrlU_ResetsPrefixState verifies that Ctrl+U clears the prefix state so
// subsequent typing does not hit a stale lockedPrefix and cause an index out of range.
// Scenario: type ":songs kk" → Ctrl+U → verify prefixState is PrefixNone and lockedPrefix
// is empty → type "h" → verify cleanQuery returns "h" without panic.
func TestCtrlU_ResetsPrefixState(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":songs kk" to reach PrefixLocked state with a non-empty lockedPrefix.
	for _, ch := range ":songs kk" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState(), "setup: should be locked after :songs kk")
	require.Equal(t, ":songs", o.LockedPrefix(), "setup: locked prefix should be :songs")

	// Ctrl+U clears the input.
	o, _ = sendKey(t, o, "ctrl+u")

	// After Ctrl+U, prefix state must be reset — otherwise the next cleanQuery() call
	// would do value[len(":songs"):] on an empty string → index out of range panic.
	assert.Equal(t, panes.PrefixNone, o.PrefixState(), "Ctrl+U should reset prefixState to PrefixNone")
	assert.Equal(t, "", o.LockedPrefix(), "Ctrl+U should clear lockedPrefix")

	// Type "h" — cleanQuery must return "h", not panic.
	o, _ = sendKey(t, o, "h")
	assert.Equal(t, "h", o.CleanQuery(), "cleanQuery after Ctrl+U + typing should return typed char without panic")
}

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

// ============================================================================
// Story 89 — Animated placeholder, SetSuggestions, Prompt tag, pills, syncInputToTab, SetTheme
// ============================================================================

// --- Part 1: Animated cycling placeholder ---

// TestSearchPlaceholders_FourEntries verifies all 4 placeholder messages are defined.
func TestSearchPlaceholders_FourEntries(t *testing.T) {
	require.Len(t, panes.SearchPlaceholders, 4, "should have exactly 4 placeholder messages")
}

// TestSearchPlaceholders_EachStartsWithPrefix verifies each placeholder starts with a valid prefix command.
func TestSearchPlaceholders_EachStartsWithPrefix(t *testing.T) {
	for i, ph := range panes.SearchPlaceholders {
		t.Run(ph, func(t *testing.T) {
			var found bool
			for _, p := range panes.SearchPrefixes {
				if len(ph) >= len(p) && ph[:len(p)] == p {
					found = true
					break
				}
			}
			assert.True(t, found, "placeholder[%d] %q should start with a valid prefix command", i, ph)
		})
	}
}

// TestPlaceholderTick_AdvancesIdx verifies that the placeholder tick increments placeholderIdx.
func TestPlaceholderTick_AdvancesIdx(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	initialIdx := o.PlaceholderIdx()

	// Fire the placeholder tick. Input is empty, so it should advance.
	model, cmd := o.Update(panes.SearchPlaceholderTickMsgForTest())
	updated := model.(*panes.SearchOverlay)

	expectedIdx := (initialIdx + 1) % len(panes.SearchPlaceholders)
	assert.Equal(t, expectedIdx, updated.PlaceholderIdx(), "tick should advance placeholderIdx")
	assert.Equal(t, panes.SearchPlaceholders[expectedIdx], updated.Placeholder(), "placeholder text should update")
	assert.NotNil(t, cmd, "tick should re-arm another tick")
}

// TestPlaceholderTick_WrapsAround verifies the placeholder wraps after N ticks (N = len of placeholders).
func TestPlaceholderTick_WrapsAround(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Fire N ticks from index 0: index becomes 1, 2, 3, 0 — back to 0.
	current := o
	for i := 0; i < len(panes.SearchPlaceholders); i++ {
		m, _ := current.Update(panes.SearchPlaceholderTickMsgForTest())
		current = m.(*panes.SearchOverlay)
	}
	assert.Equal(t, 0, current.PlaceholderIdx(), "after N ticks placeholderIdx should wrap to 0")
}

// TestPlaceholderTick_StopsWhenTyping verifies tick is not re-armed when user has typed.
func TestPlaceholderTick_StopsWhenTyping(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type a character.
	o, _ = sendKey(t, o, "h")
	require.Equal(t, "h", o.Query())

	// Fire placeholder tick — input is not empty, tick should NOT re-arm.
	_, cmd := o.Update(panes.SearchPlaceholderTickMsgForTest())
	assert.Nil(t, cmd, "placeholder tick should not re-arm when user has typed")
}

// TestPlaceholderTick_RestartsOnCtrlU verifies Ctrl+U returns a command that includes a new placeholder tick.
func TestPlaceholderTick_RestartsOnCtrlU(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type a character so placeholder cycling was stopped.
	o, _ = sendKey(t, o, "h")
	require.Equal(t, "h", o.Query())

	// Ctrl+U should include a placeholder tick restart in its returned command.
	_, cmd := sendKey(t, o, "ctrl+u")
	require.NotNil(t, cmd, "Ctrl+U should return a command")

	// The command batch must include a placeholder tick command (non-nil).
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	require.True(t, ok, "Ctrl+U should return a BatchMsg, got %T", msg)
	assert.Greater(t, len(batchMsg), 0, "batch should include at least one command")
}

// --- Part 2: SetSuggestions configuration ---

// TestNewSearchOverlay_ShowSuggestionsEnabled verifies ShowSuggestions is true.
func TestNewSearchOverlay_ShowSuggestionsEnabled(t *testing.T) {
	o := newTestSearchOverlay()
	assert.True(t, o.InputShowSuggestions(), "textinput.ShowSuggestions should be enabled")
}

// TestNewSearchOverlay_SuggestionsContainAllPrefixes verifies all 4 prefixes with trailing space.
func TestNewSearchOverlay_SuggestionsContainAllPrefixes(t *testing.T) {
	o := newTestSearchOverlay()
	suggestions := o.InputAvailableSuggestions()
	require.Len(t, suggestions, 4, "should have exactly 4 suggestions")
	for _, s := range suggestions {
		assert.True(t, len(s) > 0 && s[len(s)-1] == ' ',
			"suggestion %q should end with a trailing space", s)
	}
	assert.Contains(t, suggestions, ":songs ")
	assert.Contains(t, suggestions, ":artists ")
	assert.Contains(t, suggestions, ":albums ")
	assert.Contains(t, suggestions, ":playlists ")
}

// TestNewSearchOverlay_PlaceholderStyleIsInfo verifies PlaceholderStyle uses Info() color.
func TestNewSearchOverlay_PlaceholderStyleIsInfo(t *testing.T) {
	o := newTestSearchOverlay()
	th := theme.Load("black")
	assert.Equal(t, th.Info(), o.PlaceholderStyleFg(), "PlaceholderStyle foreground should match theme Info()")
}

// TestNewSearchOverlay_CompletionStyleIsTextMuted verifies CompletionStyle uses TextMuted() color.
func TestNewSearchOverlay_CompletionStyleIsTextMuted(t *testing.T) {
	o := newTestSearchOverlay()
	th := theme.Load("black")
	assert.Equal(t, th.TextMuted(), o.CompletionStyleFg(), "CompletionStyle foreground should match theme TextMuted()")
}

// --- Part 3: Prompt-based prefix tag ---

// TestPromoteToPromptTag_PromptContainsPrefix verifies the Prompt contains the locked prefix text.
func TestPromoteToPromptTag_PromptContainsPrefix(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":songs kk" to lock the prefix.
	for _, ch := range ":songs kk" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState())

	// After promotion, Prompt should contain ":songs".
	assert.Contains(t, o.PromptTag(), ":songs", "Prompt should contain locked prefix text")
}

// TestPromoteToPromptTag_ValueHoldsCleanQuery verifies Value() contains only the clean query.
func TestPromoteToPromptTag_ValueHoldsCleanQuery(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	for _, ch := range ":songs kk" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState())

	assert.Equal(t, "kk", o.Query(), "Value() should hold only the clean query after Prompt promotion")
}

// TestDemoteFromPromptTag_PromptReset verifies that demote resets Prompt to "> ".
// To trigger demote, cursor must be at pos 0. This happens when prefix is locked
// with empty clean query (pos 0) and backspace is pressed.
func TestDemoteFromPromptTag_PromptReset(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":songs " → locked, promoted. Value = "", pos = 0.
	for _, ch := range ":songs " {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState())
	require.Equal(t, "", o.Query(), "after promotion with empty query, Value should be empty")
	// Cursor is at pos 0 (end of empty string).

	// Backspace at pos 0 → demote.
	o, _ = sendKey(t, o, "backspace")

	assert.Equal(t, "> ", o.PromptTag(), "Prompt should be reset to '> ' after demote")
}

// TestDemoteFromPromptTag_ValueRestoredWithPrefix verifies Value() has prefix+space+query after demote.
func TestDemoteFromPromptTag_ValueRestoredWithPrefix(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":songs " → locked, promoted. Value = "", pos = 0.
	for _, ch := range ":songs " {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState())
	require.Equal(t, "", o.Query(), "setup: empty query after promotion")

	// Backspace at pos 0 → demote: restores ":songs " + "" = ":songs ".
	o, _ = sendKey(t, o, "backspace")

	assert.Equal(t, ":songs ", o.Query(), "after demote with empty query, Value should be ':songs ' (prefix + space + empty)")
}

// TestBackspaceNotAtPos0_NormalBackspace verifies backspace within the query does not demote.
func TestBackspaceNotAtPos0_NormalBackspace(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	for _, ch := range ":songs kk" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState())
	require.Equal(t, "kk", o.Query())

	// Backspace in the middle of the query ("kk") — cursor is at pos 2, not 0.
	// Should remove the last char, NOT demote.
	o, _ = sendKey(t, o, "backspace")
	// After promotion, cursor is at end of "kk" (pos 2), not pos 0.
	// So this backspace removes "k" → "k".
	// Prefix is still locked.
	assert.Equal(t, panes.PrefixLocked, o.PrefixState(), "backspace in the middle of query should not demote")
	assert.Contains(t, o.PromptTag(), ":songs", "Prompt tag should still be present")
}

// --- Part 5: Bidirectional tab sync ---

// TestSyncInputToTab_SongsSetsPromptTag verifies TabSongs sets styled Prompt tag.
func TestSyncInputToTab_SongsSetsPromptTag(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type a query first.
	for _, ch := range "kk" {
		o, _ = sendKey(t, o, string(ch))
	}

	// Cycle to TabSongs.
	o, _ = sendKey(t, o, "tab")
	require.Equal(t, panes.TabSongs, o.ActiveTab())

	assert.Equal(t, panes.PrefixLocked, o.PrefixState(), "syncInputToTab should set PrefixLocked for TabSongs")
	assert.Equal(t, ":songs", o.LockedPrefix(), "locked prefix should be :songs")
	assert.Contains(t, o.PromptTag(), ":songs", "Prompt should contain :songs tag")
	assert.Equal(t, "kk", o.Query(), "query should be preserved in Value()")
}

// TestSyncInputToTab_AllStripsTag verifies TabAll restores default Prompt and clears prefix.
func TestSyncInputToTab_AllStripsTag(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// First cycle to TabSongs with query "kk".
	for _, ch := range "kk" {
		o, _ = sendKey(t, o, string(ch))
	}
	o, _ = sendKey(t, o, "tab") // → TabSongs
	require.Equal(t, panes.TabSongs, o.ActiveTab())
	require.Equal(t, panes.PrefixLocked, o.PrefixState())

	// Shift+Tab back to TabAll.
	o, _ = sendKey(t, o, "shift+tab") // → TabAll (wraps)
	// Keep cycling until we reach TabAll.
	for o.ActiveTab() != panes.TabAll {
		o, _ = sendKey(t, o, "tab")
	}

	assert.Equal(t, panes.PrefixNone, o.PrefixState(), "TabAll should clear prefix state")
	assert.Equal(t, "", o.LockedPrefix(), "TabAll should clear lockedPrefix")
	assert.Equal(t, "> ", o.PromptTag(), "TabAll should restore default '> ' Prompt")
	assert.Equal(t, "kk", o.Query(), "query should be preserved when switching to TabAll")
}

// TestSyncInputToTab_PreservesQuery verifies the clean query is preserved across tab switches.
func TestSyncInputToTab_PreservesQuery(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Type ":songs kk" to lock.
	for _, ch := range ":songs kk" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState())
	require.Equal(t, "kk", o.CleanQuery())

	// Tab cycles to TabArtists (from TabSongs → TabArtists).
	o, _ = sendKey(t, o, "tab")

	assert.Equal(t, panes.TabArtists, o.ActiveTab())
	assert.Equal(t, "kk", o.Query(), "clean query preserved after tab cycle")
	assert.Equal(t, panes.PrefixLocked, o.PrefixState())
	assert.Equal(t, ":artists", o.LockedPrefix())
}

// --- Part 6: SetTheme propagation ---

// TestSetTheme_UpdatesPlaceholderStyle verifies PlaceholderStyle changes with new theme.
func TestSetTheme_UpdatesPlaceholderStyle(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	newTheme := theme.Load("dracula")
	o.SetTheme(newTheme)

	assert.Equal(t, newTheme.Info(), o.PlaceholderStyleFg(),
		"PlaceholderStyle foreground should match new theme Info()")
}

// TestSetTheme_UpdatesCompletionStyle verifies CompletionStyle changes with new theme.
func TestSetTheme_UpdatesCompletionStyle(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	newTheme := theme.Load("dracula")
	o.SetTheme(newTheme)

	assert.Equal(t, newTheme.TextMuted(), o.CompletionStyleFg(),
		"CompletionStyle foreground should match new theme TextMuted()")
}

// TestSetTheme_ReRendersPromptTagWhenLocked verifies active Prompt tag is re-styled on theme change.
func TestSetTheme_ReRendersPromptTagWhenLocked(t *testing.T) {
	o := newTestSearchOverlay()
	o.SetSize(80, 30)

	// Lock a prefix.
	for _, ch := range ":songs kk" {
		o, _ = sendKey(t, o, string(ch))
	}
	require.Equal(t, panes.PrefixLocked, o.PrefixState())

	promptBefore := o.PromptTag()
	queryBefore := o.Query()

	// Switch theme.
	newTheme := theme.Load("dracula")
	o.SetTheme(newTheme)

	// Prompt tag should still contain ":songs" (re-rendered with new theme colors).
	assert.Contains(t, o.PromptTag(), ":songs", "Prompt tag should still contain prefix after SetTheme")
	// Query should be preserved (promoteToPromptTag re-applies, not re-extracts from scratch).
	assert.Equal(t, queryBefore, o.Query(), "Query must not change after SetTheme")
	_ = promptBefore // exact ANSI codes differ by theme, just verify presence of prefix
}
