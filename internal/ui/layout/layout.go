package layout

// Manager computes pane positions from a grid definition and terminal size.
// It manages page switching, preset cycling, pane toggling, and focus rotation.
// The Manager is purely a layout engine — it does not render anything.
type Manager struct {
	activePage   PageID
	presets      map[PageID][]Preset
	activePreset map[PageID]int  // index into presets slice per page
	hidden       map[PaneID]bool // manual toggles (per-page; reset on page switch)
	rects        map[PaneID]Rect // computed positions
	focusOrder   []PaneID        // visible panes in grid order (row-by-row, left-to-right)
	focusIndex   int
	width        int
	height       int
	headerHeight int // 1 line
	statusHeight int // 3 lines (bubbles/help bar: border + 1 content row + border)
}

// NewManager creates a Manager with default presets and Music page active.
func NewManager() *Manager {
	m := &Manager{
		activePage: PageMusic,
		presets: map[PageID][]Preset{
			PageMusic: PageMusicPresets,
			PageStats: PageStatsPresets,
		},
		activePreset: map[PageID]int{
			PageMusic: 0,
			PageStats: 0,
		},
		hidden:       make(map[PaneID]bool),
		rects:        make(map[PaneID]Rect),
		headerHeight: 1,
		statusHeight: 3, // 1 content row for help bar + top/bottom border
	}
	return m
}

// Resize updates terminal dimensions and recomputes all pane rects.
func (m *Manager) Resize(width, height int) {
	m.width = width
	m.height = height
	m.recompute()
}

// recompute recalculates all Rects from the active preset + hidden state.
// Called after Resize, SetPreset, CyclePreset, TogglePage, TogglePane.
//
// The layout engine is a simple flat two-pass algorithm:
//  1. Filter rows/cells to only those that are visible (preset.Visible && !hidden).
//  2. Distribute height proportionally by HeightWeight; last row gets remainder.
//  3. Within each row distribute width proportionally by WidthWeight; last cell gets remainder.
//
// Toggle redistribution is automatic: removing a cell from a row expands surviving
// cells in that row; removing all cells from a row collapses the row itself.
func (m *Manager) recompute() {
	m.rects = make(map[PaneID]Rect)

	presets := m.presets[m.activePage]
	presetIdx := m.activePreset[m.activePage]
	if presetIdx >= len(presets) {
		return
	}
	grid := presets[presetIdx]

	isVisible := func(id PaneID) bool {
		return grid.Visible[id] && !m.hidden[id]
	}

	type liveCell struct {
		paneID      PaneID
		widthWeight int
	}
	type liveRow struct {
		heightWeight int
		cells        []liveCell
	}
	var liveRows []liveRow
	for _, row := range grid.Grid {
		var cells []liveCell
		for _, c := range row.Cells {
			if isVisible(c.PaneID) {
				cells = append(cells, liveCell{c.PaneID, c.WidthWeight})
			}
		}
		if len(cells) > 0 {
			liveRows = append(liveRows, liveRow{row.HeightWeight, cells})
		}
	}

	if len(liveRows) == 0 {
		m.focusOrder = nil
		m.clampFocusIndex()
		return
	}

	contentH := m.height - m.headerHeight - m.statusHeight
	if contentH < 0 {
		contentH = 0
	}

	totalHWeight := 0
	for _, r := range liveRows {
		totalHWeight += r.heightWeight
	}

	var newFocusOrder []PaneID
	y := 0
	for i, row := range liveRows {
		var h int
		switch {
		case totalHWeight == 0:
			h = 0
		case i == len(liveRows)-1:
			h = contentH - y
		default:
			h = contentH * row.heightWeight / totalHWeight
		}
		if h < 0 {
			h = 0
		}

		totalWWeight := 0
		for _, c := range row.cells {
			totalWWeight += c.widthWeight
		}

		x := 0
		for j, c := range row.cells {
			var w int
			switch {
			case totalWWeight == 0:
				w = 0
			case j == len(row.cells)-1:
				w = m.width - x
			default:
				w = m.width * c.widthWeight / totalWWeight
			}
			if w < 0 {
				w = 0
			}
			m.rects[c.paneID] = Rect{X: x, Y: y, Width: w, Height: h}
			newFocusOrder = append(newFocusOrder, c.paneID)
			x += w
		}
		y += h
	}

	prevFocused := m.currentFocusedPane()
	m.focusOrder = newFocusOrder
	m.restoreFocus(prevFocused)
}

// currentFocusedPane returns the pane that currently has focus, or -1 if none.
func (m *Manager) currentFocusedPane() PaneID {
	if len(m.focusOrder) == 0 {
		return PaneID(-1)
	}
	if m.focusIndex < 0 || m.focusIndex >= len(m.focusOrder) {
		return PaneID(-1)
	}
	return m.focusOrder[m.focusIndex]
}

// restoreFocus attempts to keep focus on prevFocused after a recompute.
// If prevFocused is no longer visible, focus moves to index 0.
func (m *Manager) restoreFocus(prevFocused PaneID) {
	if len(m.focusOrder) == 0 {
		m.focusIndex = 0
		return
	}
	for i, id := range m.focusOrder {
		if id == prevFocused {
			m.focusIndex = i
			return
		}
	}
	// Previously focused pane is no longer visible — reset to first
	m.focusIndex = 0
}

// clampFocusIndex ensures focusIndex is within bounds.
func (m *Manager) clampFocusIndex() {
	if len(m.focusOrder) == 0 {
		m.focusIndex = 0
		return
	}
	if m.focusIndex >= len(m.focusOrder) {
		m.focusIndex = 0
	}
}

// PaneRect returns the computed Rect for a pane. Returns zero Rect if hidden.
func (m *Manager) PaneRect(id PaneID) Rect {
	return m.rects[id]
}

// VisiblePanes returns all visible PaneIDs in layout order (top-left to bottom-right).
func (m *Manager) VisiblePanes() []PaneID {
	result := make([]PaneID, len(m.focusOrder))
	copy(result, m.focusOrder)
	return result
}

// ActivePage returns the current page.
func (m *Manager) ActivePage() PageID {
	return m.activePage
}

// ActivePresetIndex returns the current preset index for the active page.
func (m *Manager) ActivePresetIndex() int {
	return m.activePreset[m.activePage]
}

// ActivePresetName returns the name of the current preset.
func (m *Manager) ActivePresetName() string {
	idx := m.activePreset[m.activePage]
	presets := m.presets[m.activePage]
	if idx >= len(presets) {
		return ""
	}
	return presets[idx].Name
}

// TogglePage switches between PageMusic and PageStats.
// Resets hidden map and recomputes layout.
func (m *Manager) TogglePage() {
	if m.activePage == PageMusic {
		m.activePage = PageStats
	} else {
		m.activePage = PageMusic
	}
	m.hidden = make(map[PaneID]bool)
	m.focusIndex = 0
	m.recompute()
}

// CyclePreset advances to the next preset on the active page.
// Wraps to first preset after the last. Resets manual toggles.
func (m *Manager) CyclePreset() {
	page := m.activePage
	next := (m.activePreset[page] + 1) % len(m.presets[page])
	m.activePreset[page] = next
	m.hidden = make(map[PaneID]bool)
	m.focusIndex = 0
	m.recompute()
}

// SetPreset sets a specific preset index. Resets manual toggles.
func (m *Manager) SetPreset(index int) {
	page := m.activePage
	presets := m.presets[page]
	if index < 0 || index >= len(presets) {
		return
	}
	m.activePreset[page] = index
	m.hidden = make(map[PaneID]bool)
	m.focusIndex = 0
	m.recompute()
}

// TogglePane toggles visibility of a pane (keys 1-8 on Music page, 1-5 on Stats page).
// Does nothing if the pane is not part of the current preset.
// If toggling would hide ALL panes, the toggle is rejected.
// NOTE: Preset membership is the sole authority for whether a pane is toggleable.
// This naturally handles cross-page safety: panes not in the active preset are rejected,
// including Music page panes on Stats page and vice versa (with the exception of PaneNowPlaying
// which appears in both pages' presets and is intentionally toggleable on both).
func (m *Manager) TogglePane(id PaneID) {
	// Check that the pane is in the current preset
	preset := m.presets[m.activePage][m.activePreset[m.activePage]]
	if !preset.Visible[id] {
		return
	}

	// If currently visible, check if hiding would leave no panes visible
	if !m.hidden[id] {
		// Count panes that would remain visible
		visibleAfter := 0
		for _, row := range preset.Grid {
			for _, cell := range row.Cells {
				if cell.PaneID != id && preset.Visible[cell.PaneID] && !m.hidden[cell.PaneID] {
					visibleAfter++
				}
			}
		}
		if visibleAfter == 0 {
			// Reject — cannot hide the last pane
			return
		}
		m.hidden[id] = true
	} else {
		delete(m.hidden, id)
	}

	m.recompute()
}

// IsPaneVisible returns whether a pane is currently visible.
func (m *Manager) IsPaneVisible(id PaneID) bool {
	_, hasRect := m.rects[id]
	return hasRect
}

// RotateFocus moves focus to the next (forward=true) or previous visible pane.
// Wraps around. Uses focusOrder built during recompute().
func (m *Manager) RotateFocus(forward bool) {
	if len(m.focusOrder) == 0 {
		return
	}
	if forward {
		m.focusIndex = (m.focusIndex + 1) % len(m.focusOrder)
	} else {
		m.focusIndex = (m.focusIndex - 1 + len(m.focusOrder)) % len(m.focusOrder)
	}
}

// FocusedPane returns the PaneID that currently has keyboard focus.
func (m *Manager) FocusedPane() PaneID {
	if len(m.focusOrder) == 0 {
		return PaneID(-1)
	}
	return m.focusOrder[m.focusIndex]
}

// SetFocus sets focus to a specific pane. No-op if pane is not visible.
func (m *Manager) SetFocus(id PaneID) {
	for i, pane := range m.focusOrder {
		if pane == id {
			m.focusIndex = i
			return
		}
	}
	// Pane not visible — no-op
}

// PaneAt returns the PaneID at terminal coordinates (x, y).
// Returns PaneID(-1) if no pane is at that position.
// Coordinates are 0-based from top-left of terminal.
// The header occupies y=0 and the status bar occupies y=height-1.
func (m *Manager) PaneAt(x, y int) PaneID {
	// Header and status bar areas contain no panes
	if y < m.headerHeight || y >= m.height-m.statusHeight {
		return PaneID(-1)
	}
	// Adjust y to content-area-relative coordinates
	contentY := y - m.headerHeight

	for id, r := range m.rects {
		if x >= r.X && x < r.X+r.Width &&
			contentY >= r.Y && contentY < r.Y+r.Height {
			return id
		}
	}
	return PaneID(-1)
}
