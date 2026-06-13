package panes

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// prefixState tracks where the user is in the command-prefix input flow.
type prefixState int

const (
	// PrefixNone means the input has no colon prefix — normal search.
	PrefixNone prefixState = iota
	// PrefixTyping means the user is typing a prefix (e.g. ":so") — no space yet.
	PrefixTyping
	// PrefixLocked means a complete valid prefix + space was typed (e.g. ":songs ").
	// When locked, the prefix is promoted to the textinput Prompt field as a styled tag.
	// The input Value() holds only the clean query.
	PrefixLocked
)

// SearchPrefixes is the ordered list of valid command prefixes.
// Exported so tests and the help text can reference them.
var SearchPrefixes = []string{":songs", ":artists", ":albums", ":playlists", ":shows", ":episodes"}

// searchPlaceholder holds a single cycling placeholder entry with a prefix (e.g. ":songs")
// and an action text (e.g. "search tracks") that visually separates the filter command
// from the description text in the search input.
type searchPlaceholder struct {
	Prefix string // e.g. ":songs" — rendered as a styled pill in the Prompt
	Text   string // e.g. "search tracks" — rendered as dim placeholder
}

// searchPlaceholders cycles through animated placeholder entries that demonstrate
// the command prefix syntax. One entry per prefix, shown in order on a 2s tick.
var searchPlaceholders = []searchPlaceholder{
	{":songs", "search tracks"},
	{":artists", "find artists"},
	{":albums", "browse albums"},
	{":playlists", "discover playlists"},
	{":shows", "explore shows"},
	{":episodes", "find episodes"},
}

// SearchPlaceholders exposes the placeholder list for tests.
var SearchPlaceholders = searchPlaceholders

// SearchPlaceholderTexts returns just the action text strings from all placeholders.
// Used by tests that need to check if a Placeholder string is one of the cycling values.
func SearchPlaceholderTexts() []string {
	texts := make([]string, len(searchPlaceholders))
	for i, ph := range searchPlaceholders {
		texts[i] = ph.Text
	}
	return texts
}

// searchPlaceholderTickMsg fires every 2 seconds to advance the cycling placeholder.
type searchPlaceholderTickMsg struct{}

// searchPlaceholderTick returns a tea.Cmd that fires searchPlaceholderTickMsg after 2 seconds.
func searchPlaceholderTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return searchPlaceholderTickMsg{}
	})
}

// prefixToTabMap maps each command prefix to its corresponding SearchTab.
var prefixToTabMap = map[string]SearchTab{
	":songs":     TabSongs,
	":artists":   TabArtists,
	":albums":    TabAlbums,
	":playlists": TabPlaylists,
	":shows":     TabShows,
	":episodes":  TabEpisodes,
}

// tabToPrefixMap maps each non-All SearchTab to its command prefix.
// Used by syncInputToTab() to set the Prompt tag when cycling tabs.
var tabToPrefixMap = map[SearchTab]string{
	TabSongs:     ":songs",
	TabArtists:   ":artists",
	TabAlbums:    ":albums",
	TabPlaylists: ":playlists",
	TabShows:     ":shows",
	TabEpisodes:  ":episodes",
}

// PrefixToTab returns the SearchTab for the given command prefix, and whether it is valid.
// Exported so tests can verify the mapping.
func PrefixToTab(prefix string) (SearchTab, bool) {
	tab, ok := prefixToTabMap[prefix]
	return tab, ok
}

// parsePrefix updates o.prefixState and o.lockedPrefix based on the current input value.
// It is called on every keystroke (typing, backspace) to keep the prefix state in sync.
// When the prefix is already promoted to the Prompt tag (PrefixLocked with lockedPrefix set),
// this function is a no-op so it does not incorrectly reset state from the Prompt.
func (o *SearchOverlay) parsePrefix() {
	// If prefix is already promoted to Prompt tag, skip re-parsing.
	// The Prompt holds the prefix; Value holds only the query.
	if o.prefixState == PrefixLocked && o.lockedPrefix != "" {
		return
	}

	value := o.input.Value()

	if !strings.HasPrefix(value, ":") {
		o.prefixState = PrefixNone
		o.lockedPrefix = ""
		return
	}

	// Find the first space.
	spaceIdx := strings.Index(value, " ")
	if spaceIdx == -1 {
		// Still typing the prefix — no space found yet.
		o.prefixState = PrefixTyping
		o.lockedPrefix = ""
		return
	}

	// Check if the part before the space is a known prefix.
	candidate := value[:spaceIdx]
	if tab, ok := prefixToTabMap[candidate]; ok {
		o.prefixState = PrefixLocked
		o.lockedPrefix = candidate
		// Sync the intent tab to match the locked prefix.
		o.intent.tab = tab
	} else {
		// Unknown prefix — treat as normal search.
		o.prefixState = PrefixNone
		o.lockedPrefix = ""
	}
}

// cleanQuery returns the portion of the input that should be sent to the API.
// When a prefix is locked (Prompt-based), the Value() already holds only the clean query.
// Otherwise it returns the full raw input value.
func (o *SearchOverlay) cleanQuery() string {
	if o.prefixState == PrefixLocked {
		// Value is already clean — prefix is in the Prompt tag.
		return strings.TrimSpace(o.input.Value())
	}
	return o.input.Value()
}

// activeAPITypes returns the Spotify API type strings to use for the current search.
// When a prefix is locked it uses the prefix's mapped tab; otherwise it uses intent.tab.
func (o *SearchOverlay) activeAPITypes() []string {
	if o.prefixState == PrefixLocked {
		if tab, ok := prefixToTabMap[o.lockedPrefix]; ok {
			return TabToAPITypes(tab)
		}
	}
	return TabToAPITypes(o.intent.tab)
}

// showHintLine reports whether the prefix hint row should be rendered inside the
// Search panel. As of story 212, prefix hint pills and the variable-height search
// panel have been removed — the search panel is always 3 lines tall and hints are
// communicated via the cycling placeholder instead.
func (o *SearchOverlay) showHintLine() bool {
	return false
}

// ShowHintLine is the exported accessor for showHintLine — used by tests.
func (o *SearchOverlay) ShowHintLine() bool {
	return o.showHintLine()
}

// renderPrefixHints renders the hint row below the text input as styled pills.
// As of story 212, prefix hint pills have been removed — hints are communicated
// exclusively via the cycling placeholder in the text input.
func (o *SearchOverlay) renderPrefixHints(width int) string {
	return ""
}

// BuildPromptTag returns a styled lipgloss string for the prefix tag shown in the Prompt field.
// Uses SelectedBg()/SelectedFg() colors with bold and padding.
// Exported so it can be called from NewSearchOverlay (before the SearchOverlay struct exists)
// and from tests.
func BuildPromptTag(th theme.Theme, prefix string) string {
	tagStyle := lipgloss.NewStyle().
		Background(th.SelectedBg()).
		Foreground(th.SelectedFg()).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1)
	return tagStyle.Render(prefix) + " "
}

// buildPromptTag returns a styled lipgloss string for the prefix tag shown in the Prompt field.
// Delegates to the exported BuildPromptTag with the overlay's theme.
func (o *SearchOverlay) buildPromptTag(prefix string) string {
	return BuildPromptTag(o.theme, prefix)
}

// promoteToPromptTag moves the locked prefix from the input Value into the textinput
// Prompt field as a styled colored tag. After promotion, Value() holds only the clean query.
// Called when a prefix is locked (either by typing the trailing space or Tab-accepting a suggestion),
// or when SetTheme re-renders the Prompt tag with new colors.
//
// Handles two states:
//   - First promotion: Prompt is still "> ", Value = ":prefix query" → strips prefix from Value
//   - Re-render (e.g. SetTheme): Prompt already has the tag, Value = "query" → preserved as-is
func (o *SearchOverlay) promoteToPromptTag() {
	var query string
	if o.input.Prompt == "> " {
		// First promotion: Value still contains the full ":prefix query".
		// Extract the query portion after the locked prefix.
		raw := o.input.Value()
		if len(raw) > len(o.lockedPrefix) {
			query = strings.TrimSpace(raw[len(o.lockedPrefix):])
		}
		// If raw == lockedPrefix (no space + query yet), query stays "".
	} else {
		// Already promoted: Value is already the clean query.
		query = strings.TrimSpace(o.input.Value())
	}

	// Style the prefix as a tag in the Prompt.
	o.input.Prompt = o.buildPromptTag(o.lockedPrefix)

	// Value now holds only the clean query.
	o.input.SetValue(query)
	o.input.CursorEnd()

	// Replace the cycling placeholder — the prefix tag makes prefix hints redundant
	// and prevents the placeholder from flickering behind a locked prefix.
	o.input.Placeholder = "search..."
}

// demoteFromPromptTag moves the Prompt tag back into the input Value so the user can
// edit the prefix. Called when the user presses Backspace at cursor position 0 while
// a prefix is locked.
func (o *SearchOverlay) demoteFromPromptTag() {
	// Reconstruct the full input with prefix + space + current query.
	query := o.input.Value()
	o.input.Prompt = "> " // reset to default prompt
	o.input.SetValue(o.lockedPrefix + " " + query)
	o.input.CursorEnd()

	// Reset prefix state and intent tab — user is now editing freely.
	// intent.tab must also reset to TabAll so the next search is not filtered
	// by the stale locked-prefix tab value (e.g. TabSongs from a prior :songs lock).
	o.lockedPrefix = ""
	o.prefixState = PrefixNone
	o.intent.tab = TabAll
	// NOTE: We intentionally do NOT call parsePrefix() here. The restored value
	// contains ":prefix query" which would re-lock immediately. Instead we let the
	// next keypress (typically another Backspace to remove the space) drive parsePrefix.

	// Restore the cycling placeholder — the prefix tag is gone so the animated hints
	// should resume when the input is cleared.
	o.input.Placeholder = searchPlaceholders[o.placeholderIdx].Text
}

// syncInputToTab updates the Prompt tag and prefix state to match the newly-selected tab.
// Called from cycleTabForward() and cycleTabBackward() after advancing o.intent.tab.
// Preserves the clean query across tab switches.
func (o *SearchOverlay) syncInputToTab() {
	// Get the clean query so we can preserve it across the tab switch.
	query := o.cleanQuery()

	if o.intent.tab == TabAll {
		// Strip the prefix tag — restore default prompt and cycling placeholder.
		// When the input is empty, show the pill Prompt so the placeholder cycle resumes.
		if query == "" {
			o.input.Prompt = BuildPromptTag(o.theme, searchPlaceholders[o.placeholderIdx].Prefix)
		} else {
			o.input.Prompt = "> "
		}
		o.input.SetValue(query)
		o.lockedPrefix = ""
		o.prefixState = PrefixNone
		// Restore cycling placeholder — we're back to normal input mode.
		o.input.Placeholder = searchPlaceholders[o.placeholderIdx].Text
	} else if prefix, ok := tabToPrefixMap[o.intent.tab]; ok {
		// Set the prefix tag in the Prompt.
		o.lockedPrefix = prefix
		o.prefixState = PrefixLocked
		o.input.Prompt = o.buildPromptTag(prefix)
		o.input.SetValue(query)
		// Replace cycling placeholder — the prefix tag makes prefix hints redundant.
		o.input.Placeholder = "search..."
	}
	o.input.CursorEnd()
}

// --- Exported accessors for tests ---

// PrefixState returns the current prefix parsing state.
// Exported for tests.
func (o *SearchOverlay) PrefixState() prefixState {
	return o.prefixState
}

// LockedPrefix returns the confirmed locked prefix (e.g. ":songs").
// Exported for tests.
func (o *SearchOverlay) LockedPrefix() string {
	return o.lockedPrefix
}

// CleanQuery returns the API-ready query (prefix stripped when locked).
// Exported for tests.
func (o *SearchOverlay) CleanQuery() string {
	return o.cleanQuery()
}

// ActiveAPITypes returns the Spotify API type strings for the current prefix/tab state.
// Exported for tests.
func (o *SearchOverlay) ActiveAPITypes() []string {
	return o.activeAPITypes()
}

// RenderPrefixHints renders the inline autocomplete hint line.
// Exported for tests.
func (o *SearchOverlay) RenderPrefixHints(width int) string {
	return o.renderPrefixHints(width)
}

// PromptTag returns the current textinput Prompt string.
// Exported for tests that verify Prompt-based prefix tag behavior.
func (o *SearchOverlay) PromptTag() string {
	return o.input.Prompt
}
