// Package layout provides the grid-based layout engine for Spotnik's btop-inspired UI.
// It computes pane positions (Rect values) from preset definitions and terminal dimensions,
// and manages page switching, preset cycling, pane toggling, and focus rotation.
// The Manager does not render anything — rendering is handled by Feature 42 (border renderer).
package layout

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// PaneID uniquely identifies a pane slot in the grid.
type PaneID int

const (
	PaneNowPlaying     PaneID = iota // Page A pane 1 (toggle key 1)
	PaneQueue                        // Page A pane 2 (toggle key 2)
	PanePlaylists                    // Page A pane 3 (toggle key 3)
	PaneAlbums                       // Page A pane 4 (toggle key 4)
	PaneLikedSongs                   // Page A pane 5 (toggle key 5)
	PaneRecentlyPlayed               // Page A pane 6 (toggle key 6)
	PaneTopTracks                    // Page A pane 7 (toggle key 7)
	PaneTopArtists                   // Page A pane 8 (toggle key 8)
	PaneNetworkLog                   // Page B — not toggleable via number keys
	PaneGatewayHealth                // Page B — not toggleable via number keys
	PanePollingTraffic               // Page B — not toggleable via number keys
	PaneGatewayLive                  // Page B — not toggleable via number keys
)

// PageID identifies a page (group of panes).
type PageID int

const (
	PageA PageID = iota // Music (8 panes)
	PageB               // Nerd Status (2 panes)
)

// Rect describes a pane's position and size in terminal cells.
type Rect struct {
	X, Y          int // Top-left corner (relative to content area)
	Width, Height int // Dimensions including borders
}

// ContentWidth returns the usable width inside borders.
// Returns 0 if width is less than 2 (cannot fit borders).
func (r Rect) ContentWidth() int {
	if r.Width < 2 {
		return 0
	}
	return r.Width - 2
}

// ContentHeight returns the usable height inside borders.
// Returns 0 if height is less than 2 (cannot fit borders).
func (r Rect) ContentHeight() int {
	if r.Height < 2 {
		return 0
	}
	return r.Height - 2
}

// Action describes a pane-specific shortcut shown in the border.
type Action struct {
	Key   string // e.g., "f"
	Label string // e.g., "filter"
}

// Pane is the interface every grid pane must implement.
// It extends tea.Model with layout and focus management methods.
type Pane interface {
	tea.Model
	// SetSize sets the content area dimensions (inside borders).
	SetSize(width, height int)
	// SetFocused updates the keyboard focus state.
	SetFocused(focused bool)
	// IsFocused returns whether this pane currently has keyboard focus.
	IsFocused() bool
	// ID returns the PaneID for this slot in the grid.
	ID() PaneID
	// Title returns the display title shown in the pane border.
	Title() string
	// ToggleKey returns the number key (1-8) for btop-style pane toggling on Page A.
	// Returns 0 for panes that are not individually toggleable (Page B panes).
	ToggleKey() int
	// Actions returns pane-specific shortcut hints displayed in the border.
	Actions() []Action
	// SetTheme updates the pane's theme reference for runtime theme switching.
	// Table-based panes must rebuild their tables with new column colors.
	SetTheme(th theme.Theme)
}

// FilterablePane is implemented by panes that support in-pane text filtering.
// When HasActiveFilter() returns true, the routing layer sends all key events
// directly to the pane, bypassing global shortcuts.
type FilterablePane interface {
	HasActiveFilter() bool
}

// FilterQueryPane is implemented by panes that expose a committed filter query
// for display in the pane border. When ActiveFilterQuery() returns a non-empty
// string, the border renderer shows filtering: "query" in the top-right corner.
type FilterQueryPane interface {
	ActiveFilterQuery() string
}
