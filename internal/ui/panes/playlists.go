// Package panes — PlaylistManager provides the Playlist Manager view that
// temporarily replaces the three-pane layout when the user presses 3.
// It has two sub-panes: a left playlist list and a right track list.
// The pane never imports api/ — all data flows via messages and the central Store.
package panes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// playlistFocus identifies which sub-pane has keyboard focus.
type playlistFocus int

const (
	playlistFocusLeft  playlistFocus = iota // playlist list (left pane)
	playlistFocusRight                      // track list (right pane)
)

// inputMode identifies what the text input is being used for.
type inputMode int

const (
	inputModeNone   inputMode = iota
	inputModeCreate           // creating a new playlist
	inputModeRename           // renaming an existing playlist
)

// PlaylistManager is the Bubble Tea model for the full-screen Playlist Manager view.
// It combines the playlist list (left) and the track list (right) in one model,
// with Tab switching focus between the two halves.
// NOTE: PlaylistManager never imports api/ — it reads data from state.Store only.
type PlaylistManager struct {
	store *state.Store
	theme theme.Theme

	// focus tracks which sub-pane has keyboard focus.
	focus playlistFocus

	// leftCursor is the selected playlist index.
	leftCursor int

	// selectedPlaylistID is the ID of the playlist whose tracks are shown on the right.
	selectedPlaylistID string

	// tracks is the optimistic local track list for the selected playlist.
	// It may differ from the store when an optimistic update is in progress.
	tracks []trackItem

	// prevTracks is a snapshot taken before an optimistic mutation,
	// used to revert on API error.
	prevTracks []trackItem

	// rightCursor is the selected track index in the right pane.
	rightCursor int

	// nameInput is the textinput used for create/rename operations.
	nameInput textinput.Model

	// inputMode tracks what the text input is doing.
	currentInputMode inputMode

	// confirmRemove is true when the "Remove? [y/N]" prompt is showing.
	confirmRemove bool

	// pendingRemoveURI is the track URI staged for removal.
	pendingRemoveURI string

	width  int
	height int
}

// trackItem wraps an api.Track for local pane state (avoids importing api/).
type trackItem struct {
	id         string
	name       string
	uri        string
	artistName string
	durationMs int
}

// NewPlaylistManager constructs a PlaylistManager with the given store and theme.
func NewPlaylistManager(store *state.Store, t theme.Theme) *PlaylistManager {
	ti := textinput.New()
	ti.Placeholder = "Playlist name"
	ti.CharLimit = 100

	return &PlaylistManager{
		store:     store,
		theme:     t,
		nameInput: ti,
	}
}

// SetSize updates the render dimensions.
func (pm *PlaylistManager) SetSize(w, h int) {
	pm.width = w
	pm.height = h
}

// Init satisfies tea.Model. Requests playlists from the API if the store is empty.
// If the store already has playlists (e.g. the user navigated away and back), no fetch
// is triggered to avoid unnecessary duplicate API calls.
func (pm *PlaylistManager) Init() tea.Cmd {
	if len(pm.store.Playlists()) == 0 {
		return func() tea.Msg {
			return FetchPlaylistsRequestMsg{Offset: 0}
		}
	}
	return nil
}

// Cursor returns the current left-pane (playlist) cursor position (exported for tests).
func (pm *PlaylistManager) Cursor() int {
	return pm.leftCursor
}

// RightCursor returns the current right-pane (track) cursor position (exported for tests).
func (pm *PlaylistManager) RightCursor() int {
	return pm.rightCursor
}

// InputOpen returns true while the text input modal is visible (exported for tests).
func (pm *PlaylistManager) InputOpen() bool {
	return pm.currentInputMode != inputModeNone
}

// InputValue returns the current text input value (exported for tests).
func (pm *PlaylistManager) InputValue() string {
	return pm.nameInput.Value()
}

// SetInputValue sets the text input value programmatically (for tests).
func (pm *PlaylistManager) SetInputValue(v string) {
	pm.nameInput.SetValue(v)
}

// ConfirmOpen returns true while the remove confirmation prompt is visible (exported for tests).
func (pm *PlaylistManager) ConfirmOpen() bool {
	return pm.confirmRemove
}

// SelectedPlaylistID returns the ID of the currently selected playlist (exported for tests/app).
func (pm *PlaylistManager) SelectedPlaylistID() string {
	return pm.selectedPlaylistID
}

// Update processes messages for the PlaylistManager.
func (pm *PlaylistManager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case PlaylistTracksLoadedMsg:
		// Tracks fetched — update local track list from store.
		if m.PlaylistID == pm.selectedPlaylistID {
			pm.loadTracksFromStore()
		}
		return pm, nil

	case PlaylistCreatedMsg:
		if m.Err != nil {
			// Creation failed — no optimistic update to revert (we wait for API success).
			return pm, nil
		}
		// Reload playlists (store already updated by root app via re-fetch).
		return pm, nil

	case PlaylistRenamedMsg:
		if m.Err != nil {
			// Rename failed — already reverted optimistically; store was not changed.
			return pm, nil
		}
		return pm, nil

	case PlaylistRemoveResultMsg:
		if m.Err != nil {
			// Remove failed — revert optimistic deletion.
			pm.tracks = make([]trackItem, len(pm.prevTracks))
			copy(pm.tracks, pm.prevTracks)
		}
		return pm, nil

	case PlaylistReorderResultMsg:
		if m.Err != nil {
			// Reorder failed — revert to previous track order.
			pm.tracks = make([]trackItem, len(pm.prevTracks))
			copy(pm.tracks, pm.prevTracks)
		}
		return pm, nil

	case tea.KeyMsg:
		return pm.handleKey(m)
	}
	return pm, nil
}

// handleKey routes key events to the focused sub-pane.
func (pm *PlaylistManager) handleKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When text input is open, route all keys to it.
	if pm.currentInputMode != inputModeNone {
		return pm.handleInputKey(m)
	}

	// When remove confirmation is open, handle y/n/Esc.
	if pm.confirmRemove {
		return pm.handleConfirmKey(m)
	}

	switch pm.focus {
	case playlistFocusLeft:
		return pm.handleLeftKey(m)
	case playlistFocusRight:
		return pm.handleRightKey(m)
	}
	return pm, nil
}

// handleInputKey processes keys while the text input modal is open.
func (pm *PlaylistManager) handleInputKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(pm.nameInput.Value())
		if name == "" {
			// Empty name — cancel.
			pm.closeInput()
			return pm, nil
		}
		mode := pm.currentInputMode
		pm.closeInput()
		switch mode {
		case inputModeCreate:
			return pm, func() tea.Msg {
				return PlaylistCreateRequestMsg{Name: name}
			}
		case inputModeRename:
			playlists := pm.store.Playlists()
			if pm.leftCursor >= len(playlists) {
				return pm, nil
			}
			id := playlists[pm.leftCursor].ID
			return pm, func() tea.Msg {
				return PlaylistRenameRequestMsg{PlaylistID: id, NewName: name}
			}
		}
	case tea.KeyEsc:
		pm.closeInput()
	default:
		// Let the textinput model handle the keystroke.
		var cmd tea.Cmd
		pm.nameInput, cmd = pm.nameInput.Update(m)
		return pm, cmd
	}
	return pm, nil
}

// handleConfirmKey processes keys for the remove confirmation prompt.
func (pm *PlaylistManager) handleConfirmKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.Type != tea.KeyRunes {
		if m.Type == tea.KeyEsc {
			pm.confirmRemove = false
			pm.pendingRemoveURI = ""
		}
		return pm, nil
	}
	switch string(m.Runes) {
	case "y", "Y":
		uri := pm.pendingRemoveURI
		playlistID := pm.selectedPlaylistID
		pm.confirmRemove = false
		pm.pendingRemoveURI = ""
		// Optimistically remove the track.
		pm.optimisticRemoveTrack(uri)
		return pm, func() tea.Msg {
			return PlaylistRemoveRequestMsg{PlaylistID: playlistID, TrackURI: uri}
		}
	default:
		// Any other key cancels.
		pm.confirmRemove = false
		pm.pendingRemoveURI = ""
	}
	return pm, nil
}

// handleLeftKey processes keys for the playlist list (left pane).
func (pm *PlaylistManager) handleLeftKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	playlists := pm.store.Playlists()
	switch m.Type {
	case tea.KeyTab:
		if pm.selectedPlaylistID != "" {
			pm.focus = playlistFocusRight
		}
		return pm, nil
	case tea.KeyEnter:
		if len(playlists) == 0 {
			return pm, nil
		}
		selected := playlists[pm.leftCursor]
		pm.selectedPlaylistID = selected.ID
		pm.rightCursor = 0
		// Load from store if available, otherwise request fetch.
		tracks := pm.store.PlaylistTracks(selected.ID)
		if tracks != nil {
			pm.loadTracksFromStore()
			return pm, nil
		}
		return pm, func() tea.Msg {
			return FetchPlaylistTracksRequestMsg{PlaylistID: selected.ID}
		}
	case tea.KeyRunes:
		switch string(m.Runes) {
		case "j":
			if pm.leftCursor < len(playlists)-1 {
				pm.leftCursor++
			}
		case "k":
			if pm.leftCursor > 0 {
				pm.leftCursor--
			}
		case "n":
			pm.openInput(inputModeCreate, "")
		case "r":
			if len(playlists) > 0 {
				pm.openInput(inputModeRename, playlists[pm.leftCursor].Name)
			}
		}
	case tea.KeyDown:
		if pm.leftCursor < len(playlists)-1 {
			pm.leftCursor++
		}
	case tea.KeyUp:
		if pm.leftCursor > 0 {
			pm.leftCursor--
		}
	}
	return pm, nil
}

// handleRightKey processes keys for the track list (right pane).
func (pm *PlaylistManager) handleRightKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyTab:
		pm.focus = playlistFocusLeft
		return pm, nil

	case tea.KeyRunes:
		switch string(m.Runes) {
		case "j":
			if pm.rightCursor < len(pm.tracks)-1 {
				pm.rightCursor++
			}
		case "k":
			if pm.rightCursor > 0 {
				pm.rightCursor--
			}
		case "x":
			if len(pm.tracks) > 0 {
				pm.confirmRemove = true
				pm.pendingRemoveURI = pm.tracks[pm.rightCursor].uri
			}
		case "a":
			if len(pm.tracks) > 0 {
				uri := pm.tracks[pm.rightCursor].uri
				name := pm.tracks[pm.rightCursor].name
				return pm, func() tea.Msg {
					return AddToQueueMsg{TrackURI: uri, TrackName: name}
				}
			}
		}

	case tea.KeyShiftDown:
		return pm.reorderDown()

	case tea.KeyShiftUp:
		return pm.reorderUp()

	case tea.KeyDown:
		if pm.rightCursor < len(pm.tracks)-1 {
			pm.rightCursor++
		}

	case tea.KeyUp:
		if pm.rightCursor > 0 {
			pm.rightCursor--
		}

	case tea.KeyEnter:
		if len(pm.tracks) > 0 {
			uri := pm.tracks[pm.rightCursor].uri
			return pm, func() tea.Msg {
				return PlayTrackMsg{TrackURI: uri}
			}
		}
	}
	return pm, nil
}

// reorderDown moves the selected track down one position (optimistic).
func (pm *PlaylistManager) reorderDown() (tea.Model, tea.Cmd) {
	if pm.rightCursor >= len(pm.tracks)-1 {
		return pm, nil
	}
	// Snapshot for revert.
	pm.snapshotTracks()

	rangeStart := pm.rightCursor
	insertBefore := pm.rightCursor + 2
	playlistID := pm.selectedPlaylistID

	// Swap.
	pm.tracks[pm.rightCursor], pm.tracks[pm.rightCursor+1] = pm.tracks[pm.rightCursor+1], pm.tracks[pm.rightCursor]
	pm.rightCursor++

	return pm, func() tea.Msg {
		return PlaylistReorderRequestMsg{
			PlaylistID:   playlistID,
			RangeStart:   rangeStart,
			InsertBefore: insertBefore,
			RangeLength:  1,
		}
	}
}

// reorderUp moves the selected track up one position (optimistic).
func (pm *PlaylistManager) reorderUp() (tea.Model, tea.Cmd) {
	if pm.rightCursor <= 0 {
		return pm, nil
	}
	// Snapshot for revert.
	pm.snapshotTracks()

	rangeStart := pm.rightCursor
	insertBefore := pm.rightCursor - 1
	playlistID := pm.selectedPlaylistID

	// Swap.
	pm.tracks[pm.rightCursor], pm.tracks[pm.rightCursor-1] = pm.tracks[pm.rightCursor-1], pm.tracks[pm.rightCursor]
	pm.rightCursor--

	return pm, func() tea.Msg {
		return PlaylistReorderRequestMsg{
			PlaylistID:   playlistID,
			RangeStart:   rangeStart,
			InsertBefore: insertBefore,
			RangeLength:  1,
		}
	}
}

// snapshotTracks saves a copy of the current track list for potential revert.
func (pm *PlaylistManager) snapshotTracks() {
	pm.prevTracks = make([]trackItem, len(pm.tracks))
	copy(pm.prevTracks, pm.tracks)
}

// optimisticRemoveTrack removes a track by URI from the local list.
func (pm *PlaylistManager) optimisticRemoveTrack(uri string) {
	pm.snapshotTracks()
	newTracks := make([]trackItem, 0, len(pm.tracks))
	for _, t := range pm.tracks {
		if t.uri != uri {
			newTracks = append(newTracks, t)
		}
	}
	pm.tracks = newTracks
	if pm.rightCursor >= len(pm.tracks) && pm.rightCursor > 0 {
		pm.rightCursor = len(pm.tracks) - 1
	}
}

// openInput opens the text input in the given mode, optionally pre-filling it.
func (pm *PlaylistManager) openInput(mode inputMode, prefill string) {
	pm.currentInputMode = mode
	pm.nameInput.SetValue(prefill)
	pm.nameInput.Focus()
	if prefill != "" {
		// Move cursor to end of pre-filled text.
		pm.nameInput.CursorEnd()
	}
}

// closeInput hides the text input and resets mode.
func (pm *PlaylistManager) closeInput() {
	pm.currentInputMode = inputModeNone
	pm.nameInput.Blur()
	pm.nameInput.SetValue("")
}

// loadTracksFromStore updates the local track list from the store for the selected playlist.
func (pm *PlaylistManager) loadTracksFromStore() {
	storeTracks := pm.store.PlaylistTracks(pm.selectedPlaylistID)
	pm.tracks = make([]trackItem, len(storeTracks))
	for i, t := range storeTracks {
		artist := ""
		if len(t.Artists) > 0 {
			artist = t.Artists[0].Name
		}
		pm.tracks[i] = trackItem{
			id:         t.ID,
			name:       t.Name,
			uri:        t.URI,
			artistName: artist,
			durationMs: t.DurationMs,
		}
	}
	if pm.rightCursor >= len(pm.tracks) {
		pm.rightCursor = 0
	}
}

// View renders the full Playlist Manager view.
// If the store has a PlaylistsError, renders an error box instead of the playlist layout.
// NOTE: only app.go renders the status bar — no pane-level hint bar here.
func (pm *PlaylistManager) View() string {
	if err := pm.store.PlaylistsError(); err != nil {
		return components.RenderError(
			pm.theme,
			pm.width, pm.height,
			"Failed to load playlists",
			"Press 3 to retry",
		)
	}

	if pm.width <= 0 || pm.height <= 0 {
		return pm.renderSimple()
	}
	leftWidth := pm.width * 30 / 100
	rightWidth := pm.width - leftWidth

	left := pm.renderLeft(leftWidth, pm.height-2)
	right := pm.renderRight(rightWidth, pm.height-2)

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	// Input modal overlays at the bottom if open.
	if pm.currentInputMode != inputModeNone {
		return content + "\n" + pm.renderInputLine()
	}
	return content
}

// renderSimple renders a minimal view for zero-size scenarios.
func (pm *PlaylistManager) renderSimple() string {
	var sb strings.Builder
	sb.WriteString("MY PLAYLISTS\n")
	playlists := pm.store.Playlists()
	for i, pl := range playlists {
		prefix := "  "
		if pm.store.PlayingPlaylistID() == pl.ID {
			prefix = "▶ "
		}
		cursor := "  "
		if i == pm.leftCursor {
			cursor = "> "
		}
		fmt.Fprintf(&sb, "%s%s%s (%d)\n", cursor, prefix, pl.Name, pl.TrackCount)
	}
	if len(pm.tracks) > 0 {
		sb.WriteString("\nTRACKS\n")
		for i, t := range pm.tracks {
			cursor := "  "
			if i == pm.rightCursor {
				cursor = "> "
			}
			fmt.Fprintf(&sb, "%s%d  %s · %s  %s\n", cursor, i+1, t.name, t.artistName, FormatDuration(t.durationMs))
		}
		fmt.Fprintf(&sb, "%d tracks\n", len(pm.tracks))
	}
	if pm.confirmRemove {
		fmt.Fprintf(&sb, "Remove \"%s\"? [y/N]", pm.findTrackName(pm.pendingRemoveURI))
	}
	return sb.String()
}

// findTrackName looks up a track name by URI in the local snapshot or current list.
func (pm *PlaylistManager) findTrackName(uri string) string {
	for _, t := range pm.prevTracks {
		if t.uri == uri {
			return t.name
		}
	}
	for _, t := range pm.tracks {
		if t.uri == uri {
			return t.name
		}
	}
	return "track"
}

// renderLeft renders the playlist list panel.
func (pm *PlaylistManager) renderLeft(width, height int) string {
	playlists := pm.store.Playlists()
	playingID := pm.store.PlayingPlaylistID()

	headerStyle := lipgloss.NewStyle().
		Foreground(pm.theme.SectionHeader()).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Background(pm.theme.SelectedBg()).
		Foreground(pm.theme.TextPrimary())

	normalStyle := lipgloss.NewStyle().
		Foreground(pm.theme.TextPrimary())

	mutedStyle := lipgloss.NewStyle().
		Foreground(pm.theme.TextSecondary())

	playingStyle := lipgloss.NewStyle().
		Foreground(pm.theme.PlayingIndicator()).
		Bold(true)

	var sb strings.Builder
	sb.WriteString(headerStyle.Render("MY PLAYLISTS") + "\n")
	sb.WriteString(strings.Repeat("─", width-2) + "\n")

	for i, pl := range playlists {
		isSelected := i == pm.leftCursor
		isPlaying := pl.ID == playingID

		var indicator string
		if isPlaying {
			indicator = playingStyle.Render("▶") + " "
		} else if isSelected {
			indicator = "> "
		} else {
			indicator = "  "
		}

		countStr := mutedStyle.Render(fmt.Sprintf("(%d)", pl.TrackCount))
		name := pl.Name
		if isSelected && pm.focus == playlistFocusLeft {
			nameStr := selectedStyle.Render(fmt.Sprintf("%-*s", width-8, name))
			sb.WriteString(indicator + nameStr + " " + countStr + "\n")
		} else {
			nameStr := normalStyle.Render(name)
			sb.WriteString(indicator + nameStr + " " + countStr + "\n")
		}
	}

	// Padding to fill height.
	lines := 2 + len(playlists)
	for lines < height {
		sb.WriteString("\n")
		lines++
	}

	borderColor := pm.theme.InactiveBorder()
	if pm.focus == playlistFocusLeft {
		borderColor = pm.theme.ActiveBorder()
	}
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Render(sb.String())
}

// renderRight renders the track list panel.
func (pm *PlaylistManager) renderRight(width, height int) string {
	if pm.selectedPlaylistID == "" {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(pm.theme.InactiveBorder()).
			Foreground(pm.theme.TextMuted()).
			Render("Select a playlist to view tracks")
	}

	playlists := pm.store.Playlists()
	playlistName := pm.selectedPlaylistID
	for _, pl := range playlists {
		if pl.ID == pm.selectedPlaylistID {
			playlistName = pl.Name
			break
		}
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(pm.theme.TextPrimary()).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Background(pm.theme.SelectedBg()).
		Foreground(pm.theme.TextPrimary())

	normalStyle := lipgloss.NewStyle().
		Foreground(pm.theme.TextPrimary())

	mutedStyle := lipgloss.NewStyle().
		Foreground(pm.theme.TextSecondary())

	var sb strings.Builder
	sb.WriteString(headerStyle.Render(playlistName) + "\n")
	sb.WriteString(strings.Repeat("─", width-4) + "\n")

	var totalMs int
	for i, t := range pm.tracks {
		totalMs += t.durationMs
		isSelected := i == pm.rightCursor

		num := mutedStyle.Render(fmt.Sprintf("%3d", i+1))
		dur := FormatDuration(t.durationMs)

		// Right-align duration; calculate available name width.
		durWidth := len(dur) // all durations are "m:ss" = 4 chars
		nameWidth := width - 6 - len(t.artistName) - 3 - durWidth - 4
		if nameWidth < 1 {
			nameWidth = 1
		}
		name := t.name
		if len([]rune(name)) > nameWidth {
			runes := []rune(name)
			name = string(runes[:nameWidth-1]) + "…"
		}

		line := fmt.Sprintf("%s  %-*s · %-*s  %s",
			num,
			nameWidth, name,
			len(t.artistName), t.artistName,
			dur,
		)
		if isSelected && pm.focus == playlistFocusRight {
			sb.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			sb.WriteString(normalStyle.Render(line) + "\n")
		}
	}

	// Footer.
	totalMin := totalMs / 1000 / 60
	totalSec := (totalMs / 1000) % 60
	footer := mutedStyle.Render(
		fmt.Sprintf("%d tracks · %dmin %dsec", len(pm.tracks), totalMin, totalSec),
	)

	if pm.confirmRemove {
		trackName := pm.findTrackName(pm.pendingRemoveURI)
		footer = lipgloss.NewStyle().
			Foreground(pm.theme.Error()).
			Render(fmt.Sprintf("Remove \"%s\"? [y/N]", trackName))
	}

	sb.WriteString("\n" + footer)

	borderColor := pm.theme.InactiveBorder()
	if pm.focus == playlistFocusRight {
		borderColor = pm.theme.ActiveBorder()
	}
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Render(sb.String())
}

// renderInputLine renders the text input prompt at the bottom of the view.
func (pm *PlaylistManager) renderInputLine() string {
	label := "New playlist name: "
	if pm.currentInputMode == inputModeRename {
		label = "Rename: "
	}
	style := lipgloss.NewStyle().
		Foreground(pm.theme.TextPrimary()).
		Background(pm.theme.SurfaceAlt())
	return style.Render(label + pm.nameInput.View())
}

// FormatDuration formats milliseconds as "m:ss" (e.g. 260000 → "4:20").
func FormatDuration(ms int) string {
	totalSec := ms / 1000
	min := totalSec / 60
	sec := totalSec % 60
	return fmt.Sprintf("%d:%02d", min, sec)
}
