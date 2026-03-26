package panes

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check: AlbumsPane implements layout.Pane.
var _ layout.Pane = &AlbumsPane{}

// newTestAlbumsPane creates an AlbumsPane with a fresh store and black theme.
func newTestAlbumsPane(focused bool) *AlbumsPane {
	s := state.New()
	th := theme.Load("black")
	return NewAlbumsPane(s, th, focused)
}

// newTestAlbumsPaneWithData creates an AlbumsPane pre-loaded with albums.
func newTestAlbumsPaneWithData(focused bool) *AlbumsPane {
	s := state.New()
	s.SetSavedAlbums([]domain.SavedAlbum{
		{
			Album: domain.FullAlbum{
				ID:          "al1",
				Name:        "After Hours",
				URI:         "spotify:album:al1",
				ReleaseDate: "2020-03-20",
				Artists:     []domain.Artist{{Name: "The Weeknd"}},
			},
		},
		{
			Album: domain.FullAlbum{
				ID:          "al2",
				Name:        "OK Computer",
				URI:         "spotify:album:al2",
				ReleaseDate: "1997-05-21",
				Artists:     []domain.Artist{{Name: "Radiohead"}},
			},
		},
		{
			Album: domain.FullAlbum{
				ID:          "al3",
				Name:        "In Rainbows",
				URI:         "spotify:album:al3",
				ReleaseDate: "2007-10-10",
				Artists:     []domain.Artist{{Name: "Radiohead"}},
			},
		},
	})
	th := theme.Load("black")
	pane := NewAlbumsPane(s, th, focused)
	return pane
}

// TestAlbumsPane_ImplementsLayoutPane verifies the interface is satisfied.
func TestAlbumsPane_ImplementsLayoutPane(t *testing.T) {
	pane := newTestAlbumsPane(false)
	assert.NotNil(t, pane)
}

// TestAlbumsPane_ID verifies the pane ID.
func TestAlbumsPane_ID(t *testing.T) {
	pane := newTestAlbumsPane(false)
	assert.Equal(t, layout.PaneAlbums, pane.ID())
}

// TestAlbumsPane_Title returns "Albums".
func TestAlbumsPane_Title(t *testing.T) {
	pane := newTestAlbumsPane(false)
	assert.Equal(t, "Albums", pane.Title())
}

// TestAlbumsPane_ToggleKey returns 4.
func TestAlbumsPane_ToggleKey(t *testing.T) {
	pane := newTestAlbumsPane(false)
	assert.Equal(t, 4, pane.ToggleKey())
}

// TestAlbumsPane_Actions_Default returns filter action by default.
func TestAlbumsPane_Actions_Default(t *testing.T) {
	pane := newTestAlbumsPane(true)
	actions := pane.Actions()
	keys := make([]string, len(actions))
	for i, a := range actions {
		keys[i] = a.Key
	}
	assert.Contains(t, keys, "f", "should have filter action")
}

// TestAlbumsPane_Actions_FilterActive returns close action when filter is active.
func TestAlbumsPane_Actions_FilterActive(t *testing.T) {
	pane := newTestAlbumsPane(true)
	pane.SetSize(80, 20)
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	actions := pane.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "Esc", actions[0].Key)
}

// TestAlbumsPane_View_Empty verifies clean render on empty data.
func TestAlbumsPane_View_Empty(t *testing.T) {
	pane := newTestAlbumsPane(true)
	pane.SetSize(80, 20)
	output := pane.View()
	assert.NotEmpty(t, output, "should return non-empty string for empty albums")
}

// TestAlbumsPane_View_ShowsAlbums verifies album names appear.
func TestAlbumsPane_View_ShowsAlbums(t *testing.T) {
	pane := newTestAlbumsPaneWithData(true)
	pane.SetSize(80, 20)
	output := pane.View()
	assert.Contains(t, output, "After Hours", "first album should appear")
	assert.Contains(t, output, "Radiohead", "artist should appear")
}

// TestAlbumsPane_View_ShowsYear verifies release year appears in the table.
func TestAlbumsPane_View_ShowsYear(t *testing.T) {
	pane := newTestAlbumsPaneWithData(true)
	pane.SetSize(80, 20)
	output := pane.View()
	assert.Contains(t, output, "2020", "release year should appear")
	assert.Contains(t, output, "1997", "release year should appear")
}

// TestAlbumsPane_Enter_EmitsPlayContextMsg verifies Enter emits PlayContextMsg.
func TestAlbumsPane_Enter_EmitsPlayContextMsg(t *testing.T) {
	pane := newTestAlbumsPaneWithData(true)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should return a command")

	msg := cmd()
	playMsg, ok := msg.(PlayContextMsg)
	require.True(t, ok, "command should produce PlayContextMsg")
	assert.Equal(t, "spotify:album:al1", playMsg.ContextURI, "should use album URI")
}

// TestAlbumsPane_Filter_ByAlbumName verifies filter narrows results by album name.
func TestAlbumsPane_Filter_ByAlbumName(t *testing.T) {
	pane := newTestAlbumsPaneWithData(true)
	pane.SetSize(80, 20)

	// Activate filter and type "after"
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "after" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	output := pane.View()
	assert.Contains(t, output, "After Hours", "filter should show matching album")
}

// TestAlbumsPane_Filter_ByArtistName verifies filter matches artist name.
func TestAlbumsPane_Filter_ByArtistName(t *testing.T) {
	pane := newTestAlbumsPaneWithData(true)
	pane.SetSize(80, 20)

	// Activate filter and type "weeknd"
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	for _, r := range "weeknd" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	output := pane.View()
	assert.Contains(t, output, "After Hours", "filter by artist should show matching album")
}

// TestAlbumsPane_IsFocused verifies SetFocused/IsFocused.
func TestAlbumsPane_IsFocused(t *testing.T) {
	pane := newTestAlbumsPane(false)
	assert.False(t, pane.IsFocused())
	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
}

// TestAlbumsPane_IgnoresInputWhenUnfocused verifies pane ignores input when not focused.
func TestAlbumsPane_IgnoresInputWhenUnfocused(t *testing.T) {
	pane := newTestAlbumsPaneWithData(false)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "unfocused pane should not emit commands")
}

// TestAlbumsPane_AlbumsLoadedMsg_RefreshesTable verifies data-load integration.
func TestAlbumsPane_AlbumsLoadedMsg_RefreshesTable(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewAlbumsPane(s, th, true)
	pane.SetSize(80, 20)

	s.SetSavedAlbums([]domain.SavedAlbum{
		{
			Album: domain.FullAlbum{
				ID:      "al1",
				Name:    "Discovery",
				URI:     "spotify:album:al1",
				Artists: []domain.Artist{{Name: "Daft Punk"}},
			},
		},
	})
	pane.Update(AlbumsLoadedMsg{}) //nolint:errcheck

	output := pane.View()
	assert.Contains(t, output, "Discovery", "pane should show newly loaded album after AlbumsLoadedMsg")
}

// TestAlbumsPane_LargeAlbumList verifies no panic with many albums.
func TestAlbumsPane_LargeAlbumList(t *testing.T) {
	s := state.New()
	albums := make([]domain.SavedAlbum, 100)
	for i := range albums {
		albums[i] = domain.SavedAlbum{
			Album: domain.FullAlbum{
				ID:          fmt.Sprintf("al%d", i),
				Name:        fmt.Sprintf("Album %d", i+1),
				URI:         fmt.Sprintf("spotify:album:al%d", i),
				ReleaseDate: "2020-01-01",
				Artists:     []domain.Artist{{Name: "Artist"}},
			},
		}
	}
	s.SetSavedAlbums(albums)
	th := theme.Load("black")
	pane := NewAlbumsPane(s, th, true)
	pane.SetSize(80, 20)

	output := pane.View()
	assert.NotEmpty(t, output, "large album list should render without panic")
}

// TestAlbumsPane_RefreshRows_UpdatesTable verifies the exported RefreshRows method.
func TestAlbumsPane_RefreshRows_UpdatesTable(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewAlbumsPane(s, th, true)
	pane.SetSize(80, 20)

	s.SetSavedAlbums([]domain.SavedAlbum{
		{
			Album: domain.FullAlbum{
				ID:      "al1",
				Name:    "NewAlbum",
				URI:     "spotify:album:al1",
				Artists: []domain.Artist{{Name: "Artist"}},
			},
		},
	})
	pane.RefreshRows()

	output := pane.View()
	assert.Contains(t, output, "NewAlbum", "RefreshRows should update the view")
}

// TestAlbumsPane_Navigation_JK verifies j/k move cursor.
func TestAlbumsPane_Navigation_JK(t *testing.T) {
	pane := newTestAlbumsPaneWithData(true)
	pane.SetSize(80, 20)

	initialCursor := pane.Cursor()
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) //nolint:errcheck
	assert.Greater(t, pane.Cursor(), initialCursor, "cursor should move down after j")
}
