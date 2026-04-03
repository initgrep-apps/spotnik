package panes

import (
	"fmt"
	"io"
	"strings"

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
	// Subtitle is the rich description line. Retained to satisfy list.Item.Description()
	// and to support list's built-in filtering.
	Subtitle string
	// URI is the Spotify URI used for playback and queue commands.
	URI string
	// IsTrack is true for tracks (played as individual track vs. context URI).
	IsTrack bool

	// Track-specific metadata.
	ArtistNames string // All artists joined: "Artist1, Artist2"
	AlbumName   string // Album name
	Duration    string // Formatted "3:42"
	Explicit    bool   // Explicit content flag

	// Artist-specific metadata.
	Genres     string // Top 2-3 genres joined: "art rock, grunge"
	Followers  string // Formatted: "12.4M followers" or "847 followers"
	Popularity int    // 0-100

	// Album-specific metadata.
	AlbumType    string // "Album", "Single", "Compilation"
	ReleaseYear  string // "2020" (extracted from release_date)
	TrackCount   string // "13 tracks"
	AlbumArtists string // All album artists joined

	// Playlist-specific metadata.
	Owner          string // Owner display name
	PlaylistDesc   string // Playlist description (truncated)
	PlaylistTracks string // "245 tracks"
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
// Dispatches to per-category render helpers for rich metadata display.
func (d SearchItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(SearchListItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	width := m.Width()

	switch si.Category {
	case "track":
		d.renderTrack(w, si, isSelected, width)
	case "artist":
		d.renderArtist(w, si, isSelected, width)
	case "album":
		d.renderAlbum(w, si, isSelected, width)
	case "playlist":
		d.renderPlaylist(w, si, isSelected, width)
	default:
		d.renderDefault(w, si, isSelected, width)
	}
}

// renderTrack renders a track item:
// Line 1: ♪ name ......... [E] duration
// Line 2:   artists · album
func (d SearchItemDelegate) renderTrack(w io.Writer, si SearchListItem, selected bool, width int) {
	badge := d.styledBadge(si.Category)

	// Right-side metadata: optional [E] + duration.
	explicitStr := ""
	if si.Explicit {
		explicitStr = lipgloss.NewStyle().Foreground(d.theme.Warning()).Bold(true).Render("[E]") + " "
	}
	durationStr := lipgloss.NewStyle().Foreground(d.theme.TextMuted()).Render(si.Duration)
	rightMeta := explicitStr + durationStr

	// Calculate max width for name: badge(1) + space(1) + name + padding + rightMeta
	rightMetaPlain := plainLen(explicitStr) + plainLen(durationStr)
	nameMaxW := width - 2 - rightMetaPlain - 1
	if nameMaxW < 1 {
		nameMaxW = 1
	}
	name := d.styledName(truncateString(si.Name, nameMaxW), selected, nameMaxW)
	line1 := d.rightAlign(badge+" "+name, rightMeta, width)

	// Line 2: artists · album
	artistStyle := lipgloss.NewStyle().Foreground(d.theme.ColumnSecondary())
	albumStyle := lipgloss.NewStyle().Foreground(d.theme.ColumnTertiary())
	line2 := "  " + artistStyle.Render(si.ArtistNames) + d.styledDot() + albumStyle.Render(si.AlbumName)

	_, _ = fmt.Fprintf(w, "%s\n%s\n", line1, line2)
}

// renderArtist renders an artist item:
// Line 1: ★ name
// Line 2:   genres · followers
func (d SearchItemDelegate) renderArtist(w io.Writer, si SearchListItem, selected bool, width int) {
	badge := d.styledBadge(si.Category)
	name := d.styledName(si.Name, selected, width-2)
	line1 := badge + " " + name

	// Line 2: genres · followers (omit dot if genres empty)
	genreStyle := lipgloss.NewStyle().Foreground(d.theme.ColumnSecondary())
	followerStyle := lipgloss.NewStyle().Foreground(d.theme.TextMuted())

	var line2Parts []string
	if si.Genres != "" {
		line2Parts = append(line2Parts, genreStyle.Render(si.Genres))
	}
	if si.Followers != "" {
		line2Parts = append(line2Parts, followerStyle.Render(si.Followers))
	}
	line2 := "  " + strings.Join(line2Parts, d.styledDot())

	_, _ = fmt.Fprintf(w, "%s\n%s\n", line1, line2)
}

// renderAlbum renders an album item:
// Line 1: ◎ name ......... albumType · year
// Line 2:   artists · trackCount
func (d SearchItemDelegate) renderAlbum(w io.Writer, si SearchListItem, selected bool, width int) {
	badge := d.styledBadge(si.Category)

	// Right meta: albumType · year
	typeStyle := lipgloss.NewStyle().Foreground(d.theme.Info())
	yearStyle := lipgloss.NewStyle().Foreground(d.theme.TextMuted())
	rightMeta := typeStyle.Render(si.AlbumType) + d.styledDot() + yearStyle.Render(si.ReleaseYear)

	rightMetaPlain := len(si.AlbumType) + len(" · ") + len(si.ReleaseYear)
	nameMaxW := width - 2 - rightMetaPlain - 1
	if nameMaxW < 1 {
		nameMaxW = 1
	}
	name := d.styledName(truncateString(si.Name, nameMaxW), selected, nameMaxW)
	line1 := d.rightAlign(badge+" "+name, rightMeta, width)

	// Line 2: artists · trackCount
	artistStyle := lipgloss.NewStyle().Foreground(d.theme.ColumnSecondary())
	tcStyle := lipgloss.NewStyle().Foreground(d.theme.TextMuted())
	line2 := "  " + artistStyle.Render(si.AlbumArtists) + d.styledDot() + tcStyle.Render(si.TrackCount)

	_, _ = fmt.Fprintf(w, "%s\n%s\n", line1, line2)
}

// renderPlaylist renders a playlist item:
// Line 1: ▤ name ......... trackCount
// Line 2:   by owner [· description]
func (d SearchItemDelegate) renderPlaylist(w io.Writer, si SearchListItem, selected bool, width int) {
	badge := d.styledBadge(si.Category)

	// Right meta: trackCount
	tcStyle := lipgloss.NewStyle().Foreground(d.theme.TextMuted())
	rightMeta := tcStyle.Render(si.PlaylistTracks)

	nameMaxW := width - 2 - len(si.PlaylistTracks) - 1
	if nameMaxW < 1 {
		nameMaxW = 1
	}
	name := d.styledName(truncateString(si.Name, nameMaxW), selected, nameMaxW)
	line1 := d.rightAlign(badge+" "+name, rightMeta, width)

	// Line 2: by owner [· description]
	ownerStyle := lipgloss.NewStyle().Foreground(d.theme.ColumnSecondary())
	descStyle := lipgloss.NewStyle().Foreground(d.theme.TextMuted()).Italic(true)

	line2 := "  " + ownerStyle.Render("by "+si.Owner)
	if si.PlaylistDesc != "" {
		// Truncate description to fit available width.
		availW := width - 2 - len("by "+si.Owner) - len(" · ") - 4
		desc := truncateString(si.PlaylistDesc, availW)
		line2 += d.styledDot() + descStyle.Render(desc)
	}

	_, _ = fmt.Fprintf(w, "%s\n%s\n", line1, line2)
}

// renderDefault renders items with unknown category using a simple two-line layout.
func (d SearchItemDelegate) renderDefault(w io.Writer, si SearchListItem, selected bool, _ int) {
	badge := d.styledBadge(si.Category)
	name := d.styledName(si.Name, selected, 0)
	_, _ = fmt.Fprintf(w, "%s %s\n", badge, name)
	subtitleStyle := lipgloss.NewStyle().Foreground(d.theme.TextSecondary())
	if si.Subtitle != "" {
		_, _ = fmt.Fprintf(w, "  %s\n", subtitleStyle.Render(si.Subtitle))
	} else {
		_, _ = fmt.Fprintln(w, "")
	}
}

// --- Shared delegate helpers ---

// styledBadge returns the category badge symbol styled with the category color.
func (d SearchItemDelegate) styledBadge(category string) string {
	return lipgloss.NewStyle().
		Foreground(d.badgeColor(category)).
		Bold(true).
		Render(categorySymbol(category))
}

// styledName renders the item name with appropriate color.
// When selected, applies SelectedBg/SelectedFg. The name should be pre-truncated by the caller.
func (d SearchItemDelegate) styledName(name string, selected bool, _ int) string {
	style := lipgloss.NewStyle().Foreground(d.theme.TextPrimary())
	if selected {
		style = style.
			Background(d.theme.SelectedBg()).
			Foreground(d.theme.SelectedFg())
	}
	return style.Render(name)
}

// styledDot returns " · " rendered in TextMuted color.
func (d SearchItemDelegate) styledDot() string {
	return lipgloss.NewStyle().Foreground(d.theme.TextMuted()).Render(" · ")
}

// rightAlign composes left and right parts so right is flush to the given width.
// Uses ANSI-aware lipgloss width measurement to handle styled strings.
func (d SearchItemDelegate) rightAlign(left, right string, width int) string {
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	pad := width - leftW - rightW
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
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

// plainLen returns the display width of a string, stripping ANSI codes.
// NOTE: Uses lipgloss.Width which correctly handles ANSI escape sequences.
func plainLen(s string) int {
	return lipgloss.Width(s)
}

// --- Conversion helpers: domain types → SearchListItem ---

// tracksToListItems converts domain.Track slices to SearchListItem slices.
func tracksToListItems(tracks []domain.Track) []list.Item {
	items := make([]list.Item, len(tracks))
	for i, t := range tracks {
		artists := joinArtistNames(t.Artists)
		dur := formatSearchDuration(t.DurationMs)
		subtitle := artists + " · " + t.Album.Name + " · " + dur
		items[i] = SearchListItem{
			Category:    "track",
			Name:        t.Name,
			Subtitle:    subtitle,
			URI:         t.URI,
			IsTrack:     true,
			ArtistNames: artists,
			AlbumName:   t.Album.Name,
			Duration:    dur,
			Explicit:    t.Explicit,
		}
	}
	return items
}

// artistsToListItems converts domain.SearchArtist slices to SearchListItem slices.
func artistsToListItems(artists []domain.SearchArtist) []list.Item {
	items := make([]list.Item, len(artists))
	for i, a := range artists {
		genres := joinGenres(a.Genres, 3)
		followers := formatFollowers(a.Followers)
		subtitle := genres
		if genres != "" && followers != "" {
			subtitle += " · " + followers
		} else if followers != "" {
			subtitle = followers
		}
		items[i] = SearchListItem{
			Category:   "artist",
			Name:       a.Name,
			Subtitle:   subtitle,
			URI:        a.URI,
			IsTrack:    false,
			Genres:     genres,
			Followers:  followers,
			Popularity: a.Popularity,
		}
	}
	return items
}

// albumsToListItems converts domain.SearchAlbum slices to SearchListItem slices.
func albumsToListItems(albums []domain.SearchAlbum) []list.Item {
	items := make([]list.Item, len(albums))
	for i, al := range albums {
		artists := joinArtistNames(al.Artists)
		year := extractYear(al.ReleaseDate)
		tc := fmt.Sprintf("%d tracks", al.TotalTracks)
		subtitle := artists + " · " + year + " · " + tc
		items[i] = SearchListItem{
			Category:     "album",
			Name:         al.Name,
			Subtitle:     subtitle,
			URI:          al.URI,
			IsTrack:      false,
			AlbumType:    formatAlbumType(al.AlbumType),
			ReleaseYear:  year,
			TrackCount:   tc,
			AlbumArtists: artists,
		}
	}
	return items
}

// playlistsToListItems converts domain.SearchPlaylist slices to SearchListItem slices.
func playlistsToListItems(playlists []domain.SearchPlaylist) []list.Item {
	items := make([]list.Item, len(playlists))
	for i, p := range playlists {
		tc := fmt.Sprintf("%d tracks", p.TrackCount)
		subtitle := "by " + p.Owner.DisplayName + " · " + tc
		desc := truncateString(p.Description, 60)
		if desc != "" {
			subtitle += " · " + desc
		}
		items[i] = SearchListItem{
			Category:       "playlist",
			Name:           p.Name,
			Subtitle:       subtitle,
			URI:            p.URI,
			IsTrack:        false,
			Owner:          p.Owner.DisplayName,
			PlaylistTracks: tc,
			PlaylistDesc:   desc,
		}
	}
	return items
}

// --- Formatting helpers ---

// joinArtistNames joins artist names with ", ".
func joinArtistNames(artists []domain.Artist) string {
	names := make([]string, len(artists))
	for i, a := range artists {
		names[i] = a.Name
	}
	return strings.Join(names, ", ")
}

// joinGenres joins up to max genres with ", ".
func joinGenres(genres []string, max int) string {
	if len(genres) > max {
		genres = genres[:max]
	}
	return strings.Join(genres, ", ")
}

// formatFollowers returns a human-readable follower count: "12.4M followers", "3.2K followers", "847 followers".
func formatFollowers(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM followers", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK followers", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d followers", n)
	}
}

// formatSearchDuration converts milliseconds to "m:ss" format.
func formatSearchDuration(ms int) string {
	totalSec := ms / 1000
	return fmt.Sprintf("%d:%02d", totalSec/60, totalSec%60)
}

// formatAlbumType capitalizes the album type for display.
func formatAlbumType(t string) string {
	switch t {
	case "album":
		return "Album"
	case "single":
		return "Single"
	case "compilation":
		return "Compilation"
	default:
		return t
	}
}

// truncateString truncates s to max runes, appending "…" if truncated.
// When max <= 1, returns "…" for non-empty strings to avoid a slice panic.
func truncateString(s string, max int) string {
	runes := []rune(s)
	if max <= 0 {
		return ""
	}
	if len(runes) > max {
		if max == 1 {
			return "…"
		}
		return string(runes[:max-1]) + "…"
	}
	return s
}
