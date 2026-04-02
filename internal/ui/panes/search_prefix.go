package panes

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// prefixState tracks where the user is in the command-prefix input flow.
type prefixState int

const (
	// PrefixNone means the input has no colon prefix — normal search.
	PrefixNone prefixState = iota
	// PrefixTyping means the user is typing a prefix (e.g. ":so") — no space yet.
	PrefixTyping
	// PrefixLocked means a complete valid prefix + space was typed (e.g. ":songs ").
	PrefixLocked
)

// SearchPrefixes is the ordered list of valid command prefixes.
// Exported so tests and the help text can reference them.
var SearchPrefixes = []string{":songs", ":artists", ":albums", ":playlists"}

// prefixToTabMap maps each command prefix to its corresponding SearchTab.
var prefixToTabMap = map[string]SearchTab{
	":songs":     TabSongs,
	":artists":   TabArtists,
	":albums":    TabAlbums,
	":playlists": TabPlaylists,
}

// PrefixToTab returns the SearchTab for the given command prefix, and whether it is valid.
// Exported so tests can verify the mapping.
func PrefixToTab(prefix string) (SearchTab, bool) {
	tab, ok := prefixToTabMap[prefix]
	return tab, ok
}

// parsePrefix updates o.prefixState and o.lockedPrefix based on the current input value.
// It is called on every keystroke (typing, backspace) to keep the prefix state in sync.
func (o *SearchOverlay) parsePrefix() {
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
		// Sync the active tab to match the locked prefix.
		o.activeTab = tab
	} else {
		// Unknown prefix — treat as normal search.
		o.prefixState = PrefixNone
		o.lockedPrefix = ""
	}
}

// cleanQuery returns the portion of the input that should be sent to the API.
// When a prefix is locked, it strips the prefix from the front.
// Otherwise it returns the full raw input value.
func (o *SearchOverlay) cleanQuery() string {
	if o.prefixState == PrefixLocked {
		value := o.input.Value()
		return strings.TrimSpace(value[len(o.lockedPrefix):])
	}
	return o.input.Value()
}

// activeAPITypes returns the Spotify API type strings to use for the current search.
// When a prefix is locked it uses the prefix's mapped tab; otherwise it uses activeTab.
func (o *SearchOverlay) activeAPITypes() []string {
	if o.prefixState == PrefixLocked {
		if tab, ok := prefixToTabMap[o.lockedPrefix]; ok {
			return TabToAPITypes(tab)
		}
	}
	return TabToAPITypes(o.activeTab)
}

// renderPrefixHints renders inline autocomplete hints when the user is typing a prefix.
// Returns an empty string when not in prefixTyping state.
func (o *SearchOverlay) renderPrefixHints(width int) string {
	if o.prefixState != PrefixTyping {
		return ""
	}

	partial := o.input.Value()
	var matches []string
	for _, p := range SearchPrefixes {
		if strings.HasPrefix(p, partial) {
			matches = append(matches, p)
		}
	}

	if len(matches) == 0 {
		return ""
	}

	hintStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())
	hint := hintStyle.Render("  " + strings.Join(matches, "  "))
	// Clamp to width so it never overflows the panel border.
	return lipgloss.NewStyle().MaxWidth(width).Render(hint)
}

// tabCompletePrefix attempts to complete the partial prefix in the input.
// If exactly one prefix matches the current partial, it is completed with a trailing space.
// With zero or more-than-one matches the input is left unchanged.
// Returns (tea.Model, tea.Cmd) — the command is always nil (no async work needed).
func (o *SearchOverlay) tabCompletePrefix() (tea.Model, tea.Cmd) {
	partial := o.input.Value()
	var matches []string
	for _, p := range SearchPrefixes {
		if strings.HasPrefix(p, partial) {
			matches = append(matches, p)
		}
	}

	if len(matches) == 1 {
		// Unique match — insert completed prefix with trailing space and re-parse.
		o.input.SetValue(matches[0] + " ")
		o.input.CursorEnd()
		o.parsePrefix()
	}
	// Zero or multiple matches — do nothing.
	return o, nil
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
