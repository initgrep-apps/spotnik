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
//
// RowSpan support: a Cell with RowSpan ≥ 2 spans its own grid row plus the next N-1 rows.
// The spanning cell occupies a fixed horizontal interval in all covered rows. Continuation
// rows (those covered by a spanner from an earlier row) reserve that interval and place their
// own cells proportionally in the remaining horizontal space.
func (m *Manager) recompute() {
	m.rects = make(map[PaneID]Rect)

	preset := m.activePreset[m.activePage]
	presets := m.presets[m.activePage]
	if preset >= len(presets) {
		return
	}
	grid := presets[preset]

	// isVisible reports whether a cell should be shown.
	isVisible := func(id PaneID) bool {
		return grid.Visible[id] && !m.hidden[id]
	}

	// ── Step 1: determine which original grid rows are "live" ──────────────────
	//
	// A row is live if:
	//   (a) it has at least one own visible cell, OR
	//   (b) a spanner from an earlier live row still covers this row.
	//
	// We track, for each original row index, which spanner panes cover it
	// (so the row layout step knows what horizontal space to reserve).

	nRows := len(grid.Grid)
	// ownCellsByRow: for each original row index, the set of visible own cells.
	type cellSpec struct {
		paneID      PaneID
		widthWeight int
		rowSpan     int // effective (≥1)
	}
	ownCellsByRow := make([][]cellSpec, nRows)
	for ri, row := range grid.Grid {
		for _, cell := range row.Cells {
			if isVisible(cell.PaneID) {
				ownCellsByRow[ri] = append(ownCellsByRow[ri], cellSpec{
					paneID:      cell.PaneID,
					widthWeight: cell.WidthWeight,
					rowSpan:     cell.rowSpan(),
				})
			}
		}
	}

	// spannerCoverageByRow: for each original row index, which spanner panes cover it.
	// A spanner originating in row i with rowSpan s covers rows i, i+1, ..., i+s-1.
	// This slice tracks coverage from "above" (rows i+1 .. i+s-1).
	spannerCoverageByRow := make([][]PaneID, nRows)
	for ri, cells := range ownCellsByRow {
		for _, c := range cells {
			if c.rowSpan > 1 {
				for k := 1; k < c.rowSpan && ri+k < nRows; k++ {
					spannerCoverageByRow[ri+k] = append(spannerCoverageByRow[ri+k], c.paneID)
				}
			}
		}
	}

	// liveRows: list of original row indices that are live.
	type liveRow struct {
		origIdx         int
		heightWeight    int
		ownCells        []cellSpec
		spannerCoverage []PaneID
	}
	var liveRows []liveRow
	for ri, origRow := range grid.Grid {
		hasOwn := len(ownCellsByRow[ri]) > 0
		hasCoverage := len(spannerCoverageByRow[ri]) > 0
		if hasOwn || hasCoverage {
			liveRows = append(liveRows, liveRow{
				origIdx:         ri,
				heightWeight:    origRow.HeightWeight,
				ownCells:        ownCellsByRow[ri],
				spannerCoverage: spannerCoverageByRow[ri],
			})
		}
	}

	if len(liveRows) == 0 {
		m.focusOrder = nil
		m.clampFocusIndex()
		return
	}

	// ── Step 2: distribute height among live rows ──────────────────────────────

	contentH := m.height - m.headerHeight - m.statusHeight
	if contentH < 0 {
		contentH = 0
	}

	totalHWeight := 0
	for _, row := range liveRows {
		totalHWeight += row.heightWeight
	}

	type rowLayout struct {
		origIdx         int
		y               int
		height          int
		ownCells        []cellSpec
		spannerCoverage []PaneID
	}
	rowLayouts := make([]rowLayout, len(liveRows))
	y := 0
	for i, row := range liveRows {
		var h int
		if totalHWeight == 0 {
			h = 0
		} else if i == len(liveRows)-1 {
			h = contentH - y
		} else {
			h = contentH * row.heightWeight / totalHWeight
		}
		if h < 0 {
			h = 0
		}
		rowLayouts[i] = rowLayout{
			origIdx:         row.origIdx,
			y:               y,
			height:          h,
			ownCells:        row.ownCells,
			spannerCoverage: row.spannerCoverage,
		}
		y += h
	}

	// Build lookup: origIdx → rowLayout index (for spanner height accumulation).
	rowIdxByOrig := make(map[int]int, len(rowLayouts))
	for i, rl := range rowLayouts {
		rowIdxByOrig[rl.origIdx] = i
	}

	// ── Step 3: spanning pass — compute spanner Rects ─────────────────────────
	//
	// For each origin row, assign X/W to all cells in declaration order,
	// then for spanning cells accumulate H across covered rows.

	type spannerRect struct {
		x, w, y, h int
	}
	spannerRects := make(map[PaneID]spannerRect)

	for _, rl := range rowLayouts {
		// Only process rows that contain a spanner at origin.
		hasSpanner := false
		for _, c := range rl.ownCells {
			if c.rowSpan > 1 {
				hasSpanner = true
				break
			}
		}
		if !hasSpanner {
			continue
		}

		// Total weight for this row (all own cells — both span and non-span).
		totalW := 0
		for _, c := range rl.ownCells {
			totalW += c.widthWeight
		}

		// Assign X/W in declaration order; record spanners.
		cx := 0
		for j, c := range rl.ownCells {
			var w int
			if totalW == 0 {
				w = 0
			} else if j == len(rl.ownCells)-1 {
				w = m.width - cx
			} else {
				w = m.width * c.widthWeight / totalW
			}
			if c.rowSpan > 1 {
				// Accumulate height across covered rows.
				totalH := 0
				for k := 0; k < c.rowSpan; k++ {
					targetOrig := rl.origIdx + k
					if rli, ok := rowIdxByOrig[targetOrig]; ok {
						totalH += rowLayouts[rli].height
					}
				}
				spannerRects[c.paneID] = spannerRect{x: cx, w: w, y: rl.y, h: totalH}
			}
			cx += w
		}
	}

	// Assign Rects for all spanning cells.
	for id, sr := range spannerRects {
		m.rects[id] = Rect{X: sr.x, Y: sr.y, Width: sr.w, Height: sr.h}
	}

	// ── Step 4: per-row placement — place non-spanner cells ───────────────────
	//
	// For each live row:
	//   1. Compute reserved horizontal intervals from spanners covering this row.
	//   2. Compute available width = totalWidth − sum(reserved widths).
	//   3. Place non-spanner cells left-to-right using nextFreeX to skip reserved.

	var newFocusOrder []PaneID
	spannerInFocusOrder := make(map[PaneID]bool)

	for _, rl := range rowLayouts {
		// Build reserved interval list for this row.
		// Reserve space for spanners that cover this row:
		//   (a) spanners originating in an earlier row (continuation coverage), AND
		//   (b) spanners originating in this row itself (so non-spanner own cells
		//       are placed in the remaining horizontal space, not the full width).
		type interval struct{ x, w int }
		var reserved []interval
		for _, covID := range rl.spannerCoverage {
			if sr, ok := spannerRects[covID]; ok {
				reserved = append(reserved, interval{sr.x, sr.w})
			}
		}
		// Also reserve space for spanners originating in this row.
		for _, c := range rl.ownCells {
			if c.rowSpan > 1 {
				if sr, ok := spannerRects[c.paneID]; ok {
					reserved = append(reserved, interval{sr.x, sr.w})
				}
			}
		}

		// nextFreeX advances cx past any reserved interval that overlaps it.
		nextFreeX := func(cx int) int {
			for {
				moved := false
				for _, iv := range reserved {
					if cx >= iv.x && cx < iv.x+iv.w {
						cx = iv.x + iv.w
						moved = true
					}
				}
				if !moved {
					break
				}
			}
			return cx
		}

		// Compute available width for own non-spanner cells.
		available := m.width
		for _, iv := range reserved {
			available -= iv.w
		}

		// Separate own cells: add spanners to focus order at origin, non-spanners get placed.
		var nonSpanCells []cellSpec
		for _, c := range rl.ownCells {
			if c.rowSpan > 1 {
				// Spanner — add to focus order at origin row only.
				if !spannerInFocusOrder[c.paneID] {
					newFocusOrder = append(newFocusOrder, c.paneID)
					spannerInFocusOrder[c.paneID] = true
				}
			} else {
				nonSpanCells = append(nonSpanCells, c)
			}
		}

		// Total weight for non-spanner cells (proportional width in available space).
		totalWWeight := 0
		for _, c := range nonSpanCells {
			totalWWeight += c.widthWeight
		}

		cx := 0
		for j, c := range nonSpanCells {
			cx = nextFreeX(cx)
			var w int
			if totalWWeight == 0 {
				w = 0
			} else if j == len(nonSpanCells)-1 {
				// Last non-spanner cell: fill to next reserved boundary or terminal edge.
				nextBoundary := m.width
				for _, iv := range reserved {
					if iv.x > cx && iv.x < nextBoundary {
						nextBoundary = iv.x
					}
				}
				w = nextBoundary - cx
			} else {
				w = available * c.widthWeight / totalWWeight
			}
			if w < 0 {
				w = 0
			}
			m.rects[c.paneID] = Rect{X: cx, Y: rl.y, Width: w, Height: rl.height}
			newFocusOrder = append(newFocusOrder, c.paneID)
			cx += w
		}
	}

	// ── Step 5: update focus order ─────────────────────────────────────────────

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
// Does nothing if the pane is not part of the current preset.
// If toggling would hide ALL panes, the toggle is rejected.
// NOTE: Preset membership is the sole authority for whether a pane is toggleable.
// This naturally handles cross-page safety: panes not in the active preset are rejected,
// including Page A panes on Page B and vice versa (with the exception of PaneNowPlaying
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
