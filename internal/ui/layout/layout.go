package layout

// Manager computes pane positions from a grid definition and terminal size.
// It manages page switching, preset cycling, pane toggling, and focus rotation.
// The Manager is purely a layout engine — it does not render anything.
type Manager struct {
	activePage   PageID
	presets      map[PageID][]Preset
	activePreset map[PageID]int  // index into presets slice per page
	hidden       map[PaneID]bool // manual toggles (Page A only)
	rects        map[PaneID]Rect // computed positions
	focusOrder   []PaneID        // visible panes in grid order (row-by-row, left-to-right)
	focusIndex   int
	width        int
	height       int
	headerHeight int // 1 line
	statusHeight int // 3 lines (bubbles/help bar: border + 1 content row + border)
}

// NewManager creates a Manager with default presets and Page A active.
func NewManager() *Manager {
	m := &Manager{
		activePage: PageA,
		presets: map[PageID][]Preset{
			PageA: PageAPresets,
			PageB: PageBPresets,
		},
		activePreset: map[PageID]int{
			PageA: 0,
			PageB: 0,
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
func (m *Manager) recompute() {
	// Clear previous rects
	m.rects = make(map[PaneID]Rect)

	preset := m.activePreset[m.activePage]
	presets := m.presets[m.activePage]
	if preset >= len(presets) {
		return
	}
	grid := presets[preset]

	// Build active grid: filter hidden cells and empty rows.
	type activeCell struct {
		paneID      PaneID
		widthWeight int
	}
	type activeRow struct {
		heightWeight int
		cells        []activeCell
	}

	var activeRows []activeRow
	for _, row := range grid.Grid {
		var cells []activeCell
		for _, cell := range row.Cells {
			// A cell is visible if:
			// 1. The pane is in the preset's Visible map
			// 2. The pane is not manually hidden
			if grid.Visible[cell.PaneID] && !m.hidden[cell.PaneID] {
				cells = append(cells, activeCell{cell.PaneID, cell.WidthWeight})
			}
		}
		if len(cells) > 0 {
			activeRows = append(activeRows, activeRow{row.HeightWeight, cells})
		}
	}

	if len(activeRows) == 0 {
		m.focusOrder = nil
		m.clampFocusIndex()
		return
	}

	// Content area dimensions
	contentH := m.height - m.headerHeight - m.statusHeight
	if contentH < 0 {
		contentH = 0
	}

	// Compute total height weight
	totalHWeight := 0
	for _, row := range activeRows {
		totalHWeight += row.heightWeight
	}

	// Distribute height among rows
	type rowLayout struct {
		y      int
		height int
		cells  []activeCell
	}
	var rowLayouts []rowLayout

	y := 0
	for i, row := range activeRows {
		var h int
		if totalHWeight == 0 {
			h = 0
		} else if i == len(activeRows)-1 {
			// Last row absorbs rounding remainder
			h = contentH - y
		} else {
			h = contentH * row.heightWeight / totalHWeight
		}
		if h < 0 {
			h = 0
		}
		rowLayouts = append(rowLayouts, rowLayout{y, h, row.cells})
		y += h
	}

	// Distribute width per row and assign Rects
	var newFocusOrder []PaneID
	for _, rl := range rowLayouts {
		totalWWeight := 0
		for _, cell := range rl.cells {
			totalWWeight += cell.widthWeight
		}

		x := 0
		for j, cell := range rl.cells {
			var w int
			if totalWWeight == 0 {
				w = 0
			} else if j == len(rl.cells)-1 {
				// Last cell absorbs rounding remainder
				w = m.width - x
			} else {
				w = m.width * cell.widthWeight / totalWWeight
			}
			if w < 0 {
				w = 0
			}
			m.rects[cell.paneID] = Rect{X: x, Y: rl.y, Width: w, Height: rl.height}
			newFocusOrder = append(newFocusOrder, cell.paneID)
			x += w
		}
	}

	// Update focus order and clamp index
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

// TogglePage switches between PageA and PageB.
// Resets hidden map and recomputes layout.
func (m *Manager) TogglePage() {
	if m.activePage == PageA {
		m.activePage = PageB
	} else {
		m.activePage = PageA
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

// TogglePane toggles visibility of a pane (keys 1-8 on Page A, 1-5 on Page B).
// Does nothing if the pane belongs to a different page or is not part of the current preset.
// If toggling would hide ALL panes, the toggle is rejected.
func (m *Manager) TogglePane(id PaneID) {
	// Each page can only toggle its own panes.
	if m.activePage == PageA && id >= PaneNetworkLog {
		return
	}
	if m.activePage == PageB && id < PaneNetworkLog {
		return
	}

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
