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
	"github.com/initgrep-apps/spotnik/internal/uikit"
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

// categorySymbol returns the badge glyph for the given category via the uikit catalogue.
// In ASCII mode ActiveMode() returns GlyphASCII and GlyphFor returns safe ASCII forms.
func categorySymbol(category string) string {
	m := uikit.ActiveMode()
	switch category {
	case "track":
		return uikit.GlyphFor(uikit.GlyphMusicNote, m)
	case "artist":
		return uikit.GlyphFor(uikit.GlyphPinned, m)
	case "album":
		return uikit.GlyphFor(uikit.GlyphInactive, m)
	case "playlist":
		return uikit.GlyphFor(uikit.GlyphPlaylist, m)
	default:
		return uikit.GlyphFor(uikit.GlyphSeparator, m)
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

// Height returns the number of lines each item occupies (3: title + subtitle + description).
func (d SearchItemDelegate) Height() int { return 3 }

// wrapLine applies layout to a single content line.
// content must be a single line; multi-line input only gets the bar on the first line.
// Selected items get a left accent bar (GlyphVRule in ActiveBorder colour) followed by a space;
// in ASCII mode GlyphVRule renders as "|". Normal items get 2-space left padding.
// No background fill is used — the border glyph is the sole selection indicator.
func (d SearchItemDelegate) wrapLine(content string, selected bool) string {
	if selected {
		bar := lipgloss.NewStyle().
			Foreground(d.theme.ActiveBorder()).
			Render(uikit.GlyphFor(uikit.GlyphVRule, uikit.ActiveMode()))
		return bar + " " + content
	}
	return lipgloss.NewStyle().
		Padding(0, 0, 0, 2).
		Render(content)
}

// Spacing returns the number of empty lines between items (0 = flush).
func (d SearchItemDelegate) Spacing() int { return 0 }

// Update handles messages for list items (no-op — items are stateless).
func (d SearchItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render writes the item's 3-line representation to w.
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
// Line 1: ♪ bold name ......... [E] duration
// Line 2: artists
// Line 3: album
func (d SearchItemDelegate) renderTrack(w io.Writer, si SearchListItem, selected bool, width int) {
	badge := d.styledBadge(si.Category)
	innerW := width - 2
	if innerW < 1 {
		innerW = 1
	}

	explicitStr := ""
	if si.Explicit {
		explicitStr = lipgloss.NewStyle().Foreground(d.theme.Warning()).Bold(true).Render("[E]") + " "
	}
	durationStr := lipgloss.NewStyle().Foreground(d.theme.TextMuted()).Render(si.Duration)
	rightMeta := explicitStr + durationStr

	rightMetaPlain := plainLen(explicitStr) + plainLen(durationStr)
	nameMaxW := innerW - 2 - rightMetaPlain - 1
	if nameMaxW < 1 {
		nameMaxW = 1
	}
	name := d.styledName(truncateString(si.Name, nameMaxW), selected, nameMaxW)
	line1Content := d.rightAlign(badge+" "+name, rightMeta, innerW)

	line2Content := d.line2Style(selected, d.theme.ColumnSecondary()).Render(si.ArtistNames)
	line3Content := d.line3Style(selected, d.theme.ColumnTertiary()).Render(si.AlbumName)

	_, _ = fmt.Fprintf(w, "%s\n%s\n%s\n",
		d.wrapLine(line1Content, selected),
		d.wrapLine(line2Content, selected),
		d.wrapLine(line3Content, selected))
}

// renderArtist renders an artist item:
// Line 1: ★ bold name
// Line 2: genres
// Line 3: followers · popularity
func (d SearchItemDelegate) renderArtist(w io.Writer, si SearchListItem, selected bool, width int) {
	badge := d.styledBadge(si.Category)
	innerW := width - 2
	if innerW < 1 {
		innerW = 1
	}

	name := d.styledName(truncateString(si.Name, innerW-2), selected, innerW-2)
	line1Content := badge + " " + name

	line2Content := d.line2Style(selected, d.theme.ColumnSecondary()).Render(si.Genres)

	l3st := d.line3Style(selected, d.theme.TextMuted())
	var line3Parts []string
	if si.Followers != "" {
		line3Parts = append(line3Parts, l3st.Render(si.Followers))
	}
	if si.Popularity > 0 {
		line3Parts = append(line3Parts, l3st.Render(fmt.Sprintf("Pop: %d", si.Popularity)))
	}
	line3Content := strings.Join(line3Parts, d.styledDot())

	_, _ = fmt.Fprintf(w, "%s\n%s\n%s\n",
		d.wrapLine(line1Content, selected),
		d.wrapLine(line2Content, selected),
		d.wrapLine(line3Content, selected))
}

// renderAlbum renders an album item:
// Line 1: ◎ bold name ......... albumType · year
// Line 2: artists
// Line 3: trackCount
func (d SearchItemDelegate) renderAlbum(w io.Writer, si SearchListItem, selected bool, width int) {
	badge := d.styledBadge(si.Category)
	innerW := width - 2
	if innerW < 1 {
		innerW = 1
	}

	rightMeta := lipgloss.NewStyle().Foreground(d.theme.Info()).Render(si.AlbumType) +
		d.styledDot() +
		lipgloss.NewStyle().Foreground(d.theme.TextMuted()).Render(si.ReleaseYear)

	rightMetaPlain := len(si.AlbumType) + len(" · ") + len(si.ReleaseYear)
	nameMaxW := innerW - 2 - rightMetaPlain - 1
	if nameMaxW < 1 {
		nameMaxW = 1
	}
	name := d.styledName(truncateString(si.Name, nameMaxW), selected, nameMaxW)
	line1Content := d.rightAlign(badge+" "+name, rightMeta, innerW)

	line2Content := d.line2Style(selected, d.theme.ColumnSecondary()).Render(si.AlbumArtists)
	line3Content := d.line3Style(selected, d.theme.TextMuted()).Render(si.TrackCount)

	_, _ = fmt.Fprintf(w, "%s\n%s\n%s\n",
		d.wrapLine(line1Content, selected),
		d.wrapLine(line2Content, selected),
		d.wrapLine(line3Content, selected))
}

// renderPlaylist renders a playlist item:
// Line 1: ▤ bold name ......... trackCount
// Line 2: by owner
// Line 3: description (italic)
func (d SearchItemDelegate) renderPlaylist(w io.Writer, si SearchListItem, selected bool, width int) {
	badge := d.styledBadge(si.Category)
	innerW := width - 2
	if innerW < 1 {
		innerW = 1
	}

	rightMeta := lipgloss.NewStyle().Foreground(d.theme.TextMuted()).Render(si.PlaylistTracks)

	nameMaxW := innerW - 2 - len(si.PlaylistTracks) - 1
	if nameMaxW < 1 {
		nameMaxW = 1
	}
	name := d.styledName(truncateString(si.Name, nameMaxW), selected, nameMaxW)
	line1Content := d.rightAlign(badge+" "+name, rightMeta, innerW)

	line2Content := d.line2Style(selected, d.theme.ColumnSecondary()).Render("by " + si.Owner)

	desc := truncateString(si.PlaylistDesc, innerW)
	line3Content := d.line3Style(selected, d.theme.TextMuted()).Italic(true).Render(desc)

	_, _ = fmt.Fprintf(w, "%s\n%s\n%s\n",
		d.wrapLine(line1Content, selected),
		d.wrapLine(line2Content, selected),
		d.wrapLine(line3Content, selected))
}

// renderDefault renders items with unknown category using a simple 3-line layout.
func (d SearchItemDelegate) renderDefault(w io.Writer, si SearchListItem, selected bool, width int) {
	badge := d.styledBadge(si.Category)
	innerW := width - 2
	if innerW < 1 {
		innerW = 1
	}

	name := d.styledName(truncateString(si.Name, innerW-2), selected, innerW-2)
	line1Content := badge + " " + name

	line2Content := d.line2Style(selected, d.theme.TextSecondary()).Render(si.Subtitle)
	line3Content := ""

	_, _ = fmt.Fprintf(w, "%s\n%s\n%s\n",
		d.wrapLine(line1Content, selected),
		d.wrapLine(line2Content, selected),
		d.wrapLine(line3Content, selected))
}

// --- Shared delegate helpers ---

// styledBadge returns the category badge symbol styled with the category color.
func (d SearchItemDelegate) styledBadge(category string) string {
	return lipgloss.NewStyle().
		Foreground(d.badgeColor(category)).
		Bold(true).
		Render(categorySymbol(category))
}

// styledName renders the item name in bold. When selected, the foreground switches to
// SelectedFg — a dedicated selection accent guaranteed to differ from ColumnSecondary,
// ColumnTertiary, and TextPrimary in every theme.
func (d SearchItemDelegate) styledName(name string, selected bool, _ int) string {
	fg := d.theme.TextPrimary()
	if selected {
		fg = d.theme.SelectedFg()
	}
	return lipgloss.NewStyle().
		Foreground(fg).
		Bold(true).
		Render(name)
}

// line2Style returns a style for the second (subtitle) line of a list item.
// When selected, uses the SelectedFg accent at normal weight.
func (d SearchItemDelegate) line2Style(selected bool, normal lipgloss.Color) lipgloss.Style {
	if selected {
		return lipgloss.NewStyle().Foreground(d.theme.SelectedFg())
	}
	return lipgloss.NewStyle().Foreground(normal)
}

// line3Style returns a style for the third (detail) line of a list item.
// When selected, uses the SelectedFg accent with italic for a graduated look.
func (d SearchItemDelegate) line3Style(selected bool, normal lipgloss.Color) lipgloss.Style {
	if selected {
		return lipgloss.NewStyle().Foreground(d.theme.SelectedFg()).Italic(true)
	}
	return lipgloss.NewStyle().Foreground(normal)
}

// styledDot returns a separator glyph (GlyphSeparator) surrounded by spaces,
// rendered in TextMuted color. In ASCII mode GlyphSeparator renders as "|".
func (d SearchItemDelegate) styledDot() string {
	sep := uikit.GlyphFor(uikit.GlyphSeparator, uikit.ActiveMode())
	return lipgloss.NewStyle().Foreground(d.theme.TextMuted()).Render(" " + sep + " ")
}

// rightAlign composes left and right parts so right is flush to the given width,
// padding the gap with plain spaces.
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

// --- Exported converters: domain types → []SearchListItem (for commands.go) ---

// TracksToSearchListItems converts domain.Track slices to SearchListItem slices.
// Used by commands.go to build SearchPageLoadedMsg.Results from the API response.
func TracksToSearchListItems(tracks []domain.Track) []SearchListItem {
	items := make([]SearchListItem, len(tracks))
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

// ArtistsToSearchListItems converts domain.SearchArtist slices to SearchListItem slices.
// Used by commands.go to build SearchPageLoadedMsg.Results from the API response.
func ArtistsToSearchListItems(artists []domain.SearchArtist) []SearchListItem {
	items := make([]SearchListItem, len(artists))
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

// AlbumsToSearchListItems converts domain.SearchAlbum slices to SearchListItem slices.
// Used by commands.go to build SearchPageLoadedMsg.Results from the API response.
func AlbumsToSearchListItems(albums []domain.SearchAlbum) []SearchListItem {
	items := make([]SearchListItem, len(albums))
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

// PlaylistsToSearchListItems converts domain.SearchPlaylist slices to SearchListItem slices.
// Used by commands.go to build SearchPageLoadedMsg.Results from the API response.
func PlaylistsToSearchListItems(playlists []domain.SearchPlaylist) []SearchListItem {
	items := make([]SearchListItem, len(playlists))
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

// --- Internal helpers: domain types → []list.Item ---
// These delegate to the exported converters to avoid duplicating conversion logic.
// The bubbles/list component requires []list.Item; SearchListItem satisfies list.Item.

// tracksToListItems converts domain.Track slices to []list.Item via TracksToSearchListItems.
func tracksToListItems(tracks []domain.Track) []list.Item {
	src := TracksToSearchListItems(tracks)
	items := make([]list.Item, len(src))
	for i, s := range src {
		items[i] = s
	}
	return items
}

// artistsToListItems converts domain.SearchArtist slices to []list.Item via ArtistsToSearchListItems.
func artistsToListItems(artists []domain.SearchArtist) []list.Item {
	src := ArtistsToSearchListItems(artists)
	items := make([]list.Item, len(src))
	for i, s := range src {
		items[i] = s
	}
	return items
}

// albumsToListItems converts domain.SearchAlbum slices to []list.Item via AlbumsToSearchListItems.
func albumsToListItems(albums []domain.SearchAlbum) []list.Item {
	src := AlbumsToSearchListItems(albums)
	items := make([]list.Item, len(src))
	for i, s := range src {
		items[i] = s
	}
	return items
}

// playlistsToListItems converts domain.SearchPlaylist slices to []list.Item via PlaylistsToSearchListItems.
func playlistsToListItems(playlists []domain.SearchPlaylist) []list.Item {
	src := PlaylistsToSearchListItems(playlists)
	items := make([]list.Item, len(src))
	for i, s := range src {
		items[i] = s
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

// truncateString truncates s to max runes, appending the ellipsis glyph if truncated.
// The ellipsis rune count is accounted for so the result never exceeds max runes.
// When max is too small to fit even the ellipsis, it returns the ellipsis itself.
func truncateString(s string, max int) string {
	runes := []rune(s)
	if max <= 0 {
		return ""
	}
	if len(runes) > max {
		ellipsis := uikit.GlyphFor(uikit.GlyphEllipsis, uikit.ActiveMode())
		ellipsisLen := len([]rune(ellipsis))
		keep := max - ellipsisLen
		if keep <= 0 {
			return ellipsis
		}
		return string(runes[:keep]) + ellipsis
	}
	return s
}
