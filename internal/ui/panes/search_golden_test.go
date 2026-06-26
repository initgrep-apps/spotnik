package panes_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// sampleSearchResults returns a set of 5 SearchListItems covering all 5 searchable types.
func sampleSearchResults() []panes.SearchListItem {
	return []panes.SearchListItem{
		{
			Category:    "track",
			Name:        "Blinding Lights",
			URI:         "spotify:track:1",
			ArtistNames: "The Weeknd",
			AlbumName:   "After Hours",
			Duration:    "3:22",
			IsTrack:     true,
		},
		{
			Category:  "artist",
			Name:      "The Weeknd",
			URI:       "spotify:artist:1",
			Genres:    "Pop, R&B",
			Followers: "42M followers",
		},
		{
			Category:    "album",
			Name:        "After Hours",
			URI:         "spotify:album:1",
			ArtistNames: "The Weeknd",
			AlbumType:   "Album",
			ReleaseYear: "2020",
			TrackCount:  "14 tracks",
		},
		{
			Category: "playlist",
			Name:     "Today's Top Hits",
			URI:      "spotify:playlist:1",
			Subtitle: "By Spotify",
		},
		{
			Category: "show",
			Name:     "Tech Weekly",
			URI:      "spotify:show:1",
			IsShow:   true,
			Subtitle: "Technology podcast",
		},
	}
}

// TestSearchOverlay_Golden_Idle verifies the golden snapshot of the search overlay
// in its initial idle state — no query, placeholder cycling active, two panels visible.
func TestSearchOverlay_Golden_Idle(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 24)

	tm := goldentest.NewPaneTest(t, o, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestSearchOverlay_Golden_WithQuery verifies the golden snapshot when the user
// has typed a query but no results have been delivered yet.
func TestSearchOverlay_Golden_WithQuery(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 24)

	// Type "testing" into the input.
	for _, r := range "testing" {
		o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	tm := goldentest.NewPaneTest(t, o, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestSearchOverlay_Golden_WithResults verifies the golden snapshot after results
// have been loaded — all tabs visible, result items displayed with selection highlight.
func TestSearchOverlay_Golden_WithResults(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 24)

	// Deliver results the same way the root app model does.
	o.Update(panes.SearchPageLoadedMsg{
		Results: sampleSearchResults(),
		Total:   5,
		Query:   "testing",
		Page:    1,
	})

	tm := goldentest.NewPaneTest(t, o, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestSearchOverlay_Golden_PrefixLocked verifies the golden snapshot when
// the ":songs" prefix is locked — prompt tag shows "Search Songs".
func TestSearchOverlay_Golden_PrefixLocked(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 24)

	// Type ":songs " to lock the prefix.
	for _, r := range ":songs " {
		o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	tm := goldentest.NewPaneTest(t, o, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestSearchOverlay_Golden_Page2 verifies the golden snapshot when pagination
// shows page 2 — prev arrow active, next arrow dimmed when no more pages.
func TestSearchOverlay_Golden_Page2(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 24)

	// Set up results with total=20 so page 2 is valid but no next page.
	o.Update(panes.SearchPageLoadedMsg{
		Results: sampleSearchResults(),
		Total:   20,
		Query:   "testing",
		Page:    1,
	})

	// Send PgDn to advance to page 2.
	o.Update(tea.KeyMsg{Type: tea.KeyPgDown})

	// Deliver page 2 results.
	o.Update(panes.SearchPageLoadedMsg{
		Results: []panes.SearchListItem{
			{
				Category:    "track",
				Name:        "Save Your Tears",
				URI:         "spotify:track:2",
				ArtistNames: "The Weeknd",
				AlbumName:   "After Hours",
				Duration:    "3:36",
				IsTrack:     true,
			},
		},
		Total: 20,
		Query: "testing",
		Page:  2,
	})

	tm := goldentest.NewPaneTest(t, o, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestSearchOverlay_Golden_NoResults verifies the golden snapshot when a search
// returned zero results — empty state hint shown in the results panel.
func TestSearchOverlay_Golden_NoResults(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(80, 24)

	// Type query first so the overlay is in active-search mode.
	for _, r := range "zzzznomatch" {
		o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Deliver empty results.
	o.Update(panes.SearchPageLoadedMsg{
		Results: nil,
		Total:   0,
		Query:   "zzzznomatch",
		Page:    1,
	})

	tm := goldentest.NewPaneTest(t, o, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestSearchOverlay_Golden_Narrow verifies the golden snapshot at narrow
// terminal dimensions (40×24) — panel sizing adapted to the smaller width.
func TestSearchOverlay_Golden_Narrow(t *testing.T) {
	th := theme.Load("black")
	o := panes.NewSearchOverlay(th)
	o.SetSize(40, 24)

	// Deliver results so the panels have content.
	o.Update(panes.SearchPageLoadedMsg{
		Results: []panes.SearchListItem{
			{
				Category:    "track",
				Name:        "Short Line",
				URI:         "spotify:track:1",
				ArtistNames: "Artist",
				AlbumName:   "Album",
				Duration:    "3:22",
				IsTrack:     true,
			},
		},
		Total: 1,
		Query: "short",
		Page:  1,
	})

	tm := goldentest.NewPaneTest(t, o, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
