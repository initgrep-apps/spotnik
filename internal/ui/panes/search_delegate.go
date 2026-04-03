package panes

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// SearchListItem represents a single search result in the bubbles/list.
// It implements list.Item so the list component can render and navigate it.
type SearchListItem struct {
	// Category is the result type: "track", "artist", "album", or "playlist".
	Category string
	// Name is the primary display name.
	Name string
	// Subtitle is the secondary info line (artist for tracks/albums, owner for playlists).
	Subtitle string
	// URI is the Spotify URI used for playback and queue commands.
	URI string
	// IsTrack is true for tracks (played as individual track vs. context URI).
	IsTrack bool
}

// Title returns the item's primary display name, satisfying list.Item.
func (i SearchListItem) Title() string { return i.Name }

// Description returns the item's secondary info line, satisfying list.Item.
func (i SearchListItem) Description() string { return i.Subtitle }

// FilterValue returns the searchable string for list filtering, satisfying list.Item.
func (i SearchListItem) FilterValue() string { return i.Name }

// categorySymbol returns the single-character badge for the given category.
// Symbols are BMP Unicode glyphs (not emoji) so lipgloss Foreground() coloring works reliably.
func categorySymbol(category string) string {
	switch category {
	case "track":
		return "♪" // U+266A Eighth Note — musical, single-width
	case "artist":
		return "★" // U+2605 Black Star — fame/artist, single-width
	case "album":
		return "◎" // U+25CE Bullseye — disc-like, single-width
	case "playlist":
		return "▤" // U+25A4 Square with horizontal fill — list, single-width
	default:
		return "·"
	}
}

// SearchItemDelegate is a bubbles/list ItemDelegate that renders each search
// result with a type badge, name line, and optional subtitle line.
type SearchItemDelegate struct {
	theme theme.Theme
}

// NewSearchItemDelegate creates a SearchItemDelegate wired to the given theme.
// Exported so tests can construct it directly.
func NewSearchItemDelegate(t theme.Theme) SearchItemDelegate {
	return SearchItemDelegate{theme: t}
}

// Height returns the number of lines each item occupies (2: name + subtitle).
func (d SearchItemDelegate) Height() int { return 2 }

// Spacing returns the number of empty lines between items (0 = flush).
func (d SearchItemDelegate) Spacing() int { return 0 }

// Update handles messages for list items (no-op — items are stateless).
func (d SearchItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render writes the item's two-line representation to w.
// Line 1: badge symbol + item name (highlighted when selected).
// Line 2: indented subtitle (only when non-empty).
func (d SearchItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(SearchListItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	// Type badge colored by category.
	badgeStyle := lipgloss.NewStyle().
		Foreground(d.badgeColor(si.Category)).
		Bold(true)
	badge := badgeStyle.Render(categorySymbol(si.Category))

	// Title line: highlighted when selected.
	titleStyle := lipgloss.NewStyle().Foreground(d.theme.TextPrimary())
	if isSelected {
		titleStyle = titleStyle.
			Background(d.theme.SelectedBg()).
			Foreground(d.theme.SelectedFg())
	}
	_, _ = fmt.Fprintf(w, "%s %s\n", badge, titleStyle.Render(si.Name))

	// Subtitle line: always secondary color, indented.
	subtitleStyle := lipgloss.NewStyle().Foreground(d.theme.TextSecondary())
	if si.Subtitle != "" {
		_, _ = fmt.Fprintf(w, "  %s\n", subtitleStyle.Render(si.Subtitle))
	} else {
		_, _ = fmt.Fprintln(w, "")
	}
}

// badgeColor returns the lipgloss color for the given category badge.
func (d SearchItemDelegate) badgeColor(category string) lipgloss.Color {
	switch category {
	case "track":
		return d.theme.Success()
	case "artist":
		return d.theme.KeyHint()
	case "album":
		return d.theme.SeekBar()
	case "playlist":
		return d.theme.SectionHeader()
	default:
		return d.theme.TextMuted()
	}
}

// --- Conversion helpers: domain types → SearchListItem ---

// tracksToListItems converts domain.Track slices to SearchListItem slices.
func tracksToListItems(tracks []domain.Track) []list.Item {
	items := make([]list.Item, len(tracks))
	for i, t := range tracks {
		artist := ""
		if len(t.Artists) > 0 {
			artist = t.Artists[0].Name
		}
		items[i] = SearchListItem{
			Category: "track",
			Name:     t.Name,
			Subtitle: artist,
			URI:      t.URI,
			IsTrack:  true,
		}
	}
	return items
}

// artistsToListItems converts domain.SearchArtist slices to SearchListItem slices.
func artistsToListItems(artists []domain.SearchArtist) []list.Item {
	items := make([]list.Item, len(artists))
	for i, a := range artists {
		items[i] = SearchListItem{
			Category: "artist",
			Name:     a.Name,
			Subtitle: "",
			URI:      a.URI,
			IsTrack:  false,
		}
	}
	return items
}

// albumsToListItems converts domain.SearchAlbum slices to SearchListItem slices.
func albumsToListItems(albums []domain.SearchAlbum) []list.Item {
	items := make([]list.Item, len(albums))
	for i, al := range albums {
		artist := ""
		if len(al.Artists) > 0 {
			artist = al.Artists[0].Name
		}
		items[i] = SearchListItem{
			Category: "album",
			Name:     al.Name,
			Subtitle: artist,
			URI:      al.URI,
			IsTrack:  false,
		}
	}
	return items
}

// playlistsToListItems converts domain.SearchPlaylist slices to SearchListItem slices.
func playlistsToListItems(playlists []domain.SearchPlaylist) []list.Item {
	items := make([]list.Item, len(playlists))
	for i, p := range playlists {
		items[i] = SearchListItem{
			Category: "playlist",
			Name:     p.Name,
			Subtitle: p.Owner.DisplayName,
			URI:      p.URI,
			IsTrack:  false,
		}
	}
	return items
}

// --- SearchResultData → SearchListItem conversion helpers ---
// Used as fallback when store TypePages are empty (overlay-standalone / test path).

// searchTrackItemsToListItems converts SearchTrackItem slices to SearchListItem slices.
func searchTrackItemsToListItems(tracks []SearchTrackItem) []list.Item {
	items := make([]list.Item, len(tracks))
	for i, t := range tracks {
		items[i] = SearchListItem{
			Category: "track",
			Name:     t.Name,
			Subtitle: t.Artist,
			URI:      t.URI,
			IsTrack:  true,
		}
	}
	return items
}

// searchArtistItemsToListItems converts SearchArtistItem slices to SearchListItem slices.
func searchArtistItemsToListItems(artists []SearchArtistItem) []list.Item {
	items := make([]list.Item, len(artists))
	for i, a := range artists {
		items[i] = SearchListItem{
			Category: "artist",
			Name:     a.Name,
			Subtitle: "",
			URI:      a.URI,
			IsTrack:  false,
		}
	}
	return items
}

// searchAlbumItemsToListItems converts SearchAlbumItem slices to SearchListItem slices.
func searchAlbumItemsToListItems(albums []SearchAlbumItem) []list.Item {
	items := make([]list.Item, len(albums))
	for i, al := range albums {
		items[i] = SearchListItem{
			Category: "album",
			Name:     al.Name,
			Subtitle: al.Artist,
			URI:      al.URI,
			IsTrack:  false,
		}
	}
	return items
}

// searchPlaylistItemsToListItems converts SearchPlaylistItem slices to SearchListItem slices.
func searchPlaylistItemsToListItems(playlists []SearchPlaylistItem) []list.Item {
	items := make([]list.Item, len(playlists))
	for i, p := range playlists {
		items[i] = SearchListItem{
			Category: "playlist",
			Name:     p.Name,
			Subtitle: p.Owner,
			URI:      p.URI,
			IsTrack:  false,
		}
	}
	return items
}
