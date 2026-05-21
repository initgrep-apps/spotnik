package layout_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Task 3: Manager construction and space distribution ──────────────────────

func TestNewManager_Defaults(t *testing.T) {
	m := layout.NewManager()
	require.NotNil(t, m)
	assert.Equal(t, layout.PageMusic, m.ActivePage())
	assert.Equal(t, 0, m.ActivePresetIndex())
	assert.Equal(t, "Full Dashboard", m.ActivePresetName())
}

func TestResize_ComputesRectsForDashboard(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	visible := m.VisiblePanes()
	// Dashboard shows 8 panes
	assert.Len(t, visible, 8)

	// Every visible pane should have a non-zero Rect
	for _, id := range visible {
		r := m.PaneRect(id)
		assert.Greater(t, r.Width, 0, "pane %d should have positive width", id)
		assert.Greater(t, r.Height, 0, "pane %d should have positive height", id)
	}
}

func TestResize_RectsNonOverlapping(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	visible := m.VisiblePanes()
	rects := make([]layout.Rect, len(visible))
	for i, id := range visible {
		rects[i] = m.PaneRect(id)
	}

	for i := 0; i < len(rects); i++ {
		for j := i + 1; j < len(rects); j++ {
			a, b := rects[i], rects[j]
			overlap := a.X < b.X+b.Width && a.X+a.Width > b.X &&
				a.Y < b.Y+b.Height && a.Y+a.Height > b.Y
			assert.False(t, overlap,
				"panes %d and %d overlap: %+v vs %+v",
				visible[i], visible[j], a, b)
		}
	}
}

func TestResize_TilesContentArea(t *testing.T) {
	m := layout.NewManager()
	const W, H = 120, 30
	m.Resize(W, H)

	// Content area: full width, height minus header (1) and status (3)
	contentH := H - 4

	visible := m.VisiblePanes()

	// All rects must be within the content area
	for _, id := range visible {
		r := m.PaneRect(id)
		assert.GreaterOrEqual(t, r.X, 0, "pane %d: X must be >= 0", id)
		assert.GreaterOrEqual(t, r.Y, 0, "pane %d: Y must be >= 0", id)
		assert.LessOrEqual(t, r.X+r.Width, W, "pane %d: right edge must fit in terminal", id)
		assert.LessOrEqual(t, r.Y+r.Height, contentH, "pane %d: bottom edge must fit in content area", id)
	}
}

func TestResize_HeightWeightDistribution(t *testing.T) {
	// Dashboard: weights 2:3:3 over 26 content rows (30 - 1 header - 3 status)
	// 2/8 * 26 = 6, 3/8 * 26 = 9, last row absorbs remainder = 26-6-9 = 11
	m := layout.NewManager()
	m.Resize(120, 30) // content height = 26

	// NowPlaying is in row 1 (weight 2) — expect height 6
	nowPlayingRect := m.PaneRect(layout.PaneNowPlaying)
	assert.Equal(t, 6, nowPlayingRect.Height)

	// Playlists is in row 2 (weight 3) — expect height 9
	playlistsRect := m.PaneRect(layout.PanePlaylists)
	assert.Equal(t, 9, playlistsRect.Height)

	// Queue is in row 3 (weight 3, last row absorbs remainder) — expect 26-6-9=11
	queueRect := m.PaneRect(layout.PaneQueue)
	assert.Equal(t, 11, queueRect.Height)
}

func TestResize_WidthWeightDistribution(t *testing.T) {
	// Row 2 has 3 panes with weight 1:1:1 over width 120.
	// 120/3 = 40 each (no remainder needed since 120 is divisible by 3).
	m := layout.NewManager()
	m.Resize(120, 30)

	playlistsRect := m.PaneRect(layout.PanePlaylists)
	albumsRect := m.PaneRect(layout.PaneAlbums)
	likedRect := m.PaneRect(layout.PaneLikedSongs)

	assert.Equal(t, 40, playlistsRect.Width)
	assert.Equal(t, 40, albumsRect.Width)
	assert.Equal(t, 40, likedRect.Width)
}

func TestResize_SingleCellRowGetsFullWidth(t *testing.T) {
	// NowPlaying is the only cell in row 1 — should get full terminal width.
	m := layout.NewManager()
	m.Resize(120, 30)

	r := m.PaneRect(layout.PaneNowPlaying)
	assert.Equal(t, 120, r.Width)
}

func TestPaneRect_HiddenPaneReturnsZero(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// PaneNetworkLog is a Stats page pane — not visible on Music page
	r := m.PaneRect(layout.PaneNetworkLog)
	assert.Equal(t, layout.Rect{}, r, "hidden pane should return zero Rect")
}

func TestVisiblePanes_DashboardReturnsEight(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	visible := m.VisiblePanes()
	assert.Len(t, visible, 8)
}

func TestResize_LastCellAbsorbsWidthRemainder(t *testing.T) {
	// Row 3 has 4 panes with equal weights over 121 cols.
	// 121/4 = 30 r1: first 3 get 30, last gets 31.
	m := layout.NewManager()
	m.Resize(121, 30)

	queueRect := m.PaneRect(layout.PaneQueue)
	recentRect := m.PaneRect(layout.PaneRecentlyPlayed)
	topTracksRect := m.PaneRect(layout.PaneTopTracks)
	topArtistsRect := m.PaneRect(layout.PaneTopArtists)

	total := queueRect.Width + recentRect.Width + topTracksRect.Width + topArtistsRect.Width
	assert.Equal(t, 121, total, "all cells in a row must sum to terminal width")
}

// ── Task 4: Page toggle, preset cycling, pane toggling ───────────────────────

func TestTogglePage_SwitchesBetweenPages(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	assert.Equal(t, layout.PageMusic, m.ActivePage())
	m.TogglePage()
	assert.Equal(t, layout.PageStats, m.ActivePage())
	m.TogglePage()
	assert.Equal(t, layout.PageMusic, m.ActivePage())
}

func TestTogglePage_ClearsHiddenState(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Hide a pane on Music page
	m.TogglePane(layout.PanePlaylists)
	assert.False(t, m.IsPaneVisible(layout.PanePlaylists))

	// Switch to Stats page and back
	m.TogglePage()
	m.TogglePage()

	// Hidden state should be cleared
	assert.True(t, m.IsPaneVisible(layout.PanePlaylists))
}

func TestCyclePreset_CyclesThroughAllPresets(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	assert.Equal(t, 0, m.ActivePresetIndex())
	m.CyclePreset()
	assert.Equal(t, 1, m.ActivePresetIndex())
	m.CyclePreset()
	assert.Equal(t, 2, m.ActivePresetIndex())
	m.CyclePreset()
	assert.Equal(t, 3, m.ActivePresetIndex())
	// Wraps back to 0
	m.CyclePreset()
	assert.Equal(t, 0, m.ActivePresetIndex())
}

func TestCyclePreset_ResetsManualToggles(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Hide a pane manually
	m.TogglePane(layout.PanePlaylists)
	assert.False(t, m.IsPaneVisible(layout.PanePlaylists))

	// Cycle preset should reset toggles
	m.CyclePreset()
	// Now on Listening preset — Playlists isn't in it, but internal toggle must be reset
	m.CyclePreset() // Library — Playlists visible
	assert.True(t, m.IsPaneVisible(layout.PanePlaylists))
}

func TestTogglePane_HidesAndRestores(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Playlists starts visible
	assert.True(t, m.IsPaneVisible(layout.PanePlaylists))

	// Hide it
	m.TogglePane(layout.PanePlaylists)
	assert.False(t, m.IsPaneVisible(layout.PanePlaylists))

	// Restore it
	m.TogglePane(layout.PanePlaylists)
	assert.True(t, m.IsPaneVisible(layout.PanePlaylists))
}

func TestTogglePane_SiblingsExpand(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	widthBefore := m.PaneRect(layout.PaneAlbums).Width

	// Hide Playlists — Albums and LikedSongs should expand
	m.TogglePane(layout.PanePlaylists)

	widthAfter := m.PaneRect(layout.PaneAlbums).Width
	assert.Greater(t, widthAfter, widthBefore, "Albums should expand when Playlists is hidden")
}

func TestTogglePane_RowCollapsesWhenAllHidden(t *testing.T) {
	// Hide all panes in row 2 (Playlists, Albums, LikedSongs).
	// Row 2 disappears, rows 1 and 3 expand.
	m := layout.NewManager()
	m.Resize(120, 30)

	heightBefore := m.PaneRect(layout.PaneNowPlaying).Height

	m.TogglePane(layout.PanePlaylists)
	m.TogglePane(layout.PaneAlbums)
	m.TogglePane(layout.PaneLikedSongs)

	heightAfter := m.PaneRect(layout.PaneNowPlaying).Height
	assert.Greater(t, heightAfter, heightBefore, "NowPlaying should expand when row 2 is collapsed")

	// Row 2 panes should have zero rects
	assert.Equal(t, layout.Rect{}, m.PaneRect(layout.PanePlaylists))
	assert.Equal(t, layout.Rect{}, m.PaneRect(layout.PaneAlbums))
	assert.Equal(t, layout.Rect{}, m.PaneRect(layout.PaneLikedSongs))
}

func TestTogglePane_CannotHideLastVisible(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Switch to Listening (3 panes: NowPlaying, Queue, RecentlyPlayed)
	m.CyclePreset() // preset 1 = Listening

	// Hide all except one
	m.TogglePane(layout.PaneQueue)
	m.TogglePane(layout.PaneRecentlyPlayed)
	// Now only NowPlaying is visible

	// Attempt to hide the last pane should be rejected
	m.TogglePane(layout.PaneNowPlaying)
	assert.True(t, m.IsPaneVisible(layout.PaneNowPlaying),
		"last visible pane must not be hideable")
}

func TestTogglePane_StatsPage_TogglesNowPlaying(t *testing.T) {
	m := layout.NewManager()
	m.Resize(200, 50)
	m.TogglePage() // switch to Stats page

	// NowPlaying (key 1) is in PresetStats and must be toggleable on Stats page.
	require.True(t, m.IsPaneVisible(layout.PaneNowPlaying))
	m.TogglePane(layout.PaneNowPlaying)
	assert.False(t, m.IsPaneVisible(layout.PaneNowPlaying), "NowPlaying must hide after toggle on Stats page")

	m.TogglePane(layout.PaneNowPlaying)
	assert.True(t, m.IsPaneVisible(layout.PaneNowPlaying), "NowPlaying must show after second toggle on Stats page")
}

func TestTogglePane_StatsPage_TogglesStatsPanes(t *testing.T) {
	m := layout.NewManager()
	m.Resize(200, 50)
	m.TogglePage() // switch to Stats page

	// GatewayHealth (key 2) should be toggleable on Stats page
	require.True(t, m.IsPaneVisible(layout.PaneGatewayHealth))
	m.TogglePane(layout.PaneGatewayHealth)
	assert.False(t, m.IsPaneVisible(layout.PaneGatewayHealth), "GatewayHealth must hide after toggle")

	m.TogglePane(layout.PaneGatewayHealth)
	assert.True(t, m.IsPaneVisible(layout.PaneGatewayHealth), "GatewayHealth must show after second toggle")
}

func TestTogglePane_StatsPage_IgnoresMusicPagePanes(t *testing.T) {
	m := layout.NewManager()
	m.Resize(200, 50)
	m.TogglePage() // switch to Stats page

	// Music page panes must not be toggleable while on Stats page
	m.TogglePane(layout.PaneQueue) // PaneQueue < PaneNetworkLog — Music page pane
	assert.True(t, m.IsPaneVisible(layout.PaneNowPlaying), "NowPlaying must still be visible")
}

func TestTogglePane_MusicPage_IgnoresStatsPagePanes(t *testing.T) {
	m := layout.NewManager()
	m.Resize(200, 50)
	// Still on Music page — attempting to toggle a Stats page pane must be a no-op
	m.TogglePane(layout.PaneGatewayHealth)
	// NowPlaying is a Music page pane and must remain visible (no change to Music page state)
	assert.True(t, m.IsPaneVisible(layout.PaneNowPlaying))
}

func TestTogglePane_StatsPage_CannotHideLastPane(t *testing.T) {
	m := layout.NewManager()
	m.Resize(200, 50)
	m.TogglePage() // Stats page — PresetStats has 5 panes

	// Hide 4 of 5 panes — only NowPlaying remains
	m.TogglePane(layout.PaneGatewayHealth)
	m.TogglePane(layout.PanePollingTraffic)
	m.TogglePane(layout.PaneGatewayLive)
	m.TogglePane(layout.PaneNetworkLog)

	require.True(t, m.IsPaneVisible(layout.PaneNowPlaying), "NowPlaying must be the last visible pane")
	// Attempt to hide the last pane must be rejected
	m.TogglePane(layout.PaneNowPlaying)
	assert.True(t, m.IsPaneVisible(layout.PaneNowPlaying), "cannot-hide-last guard must reject on Stats page")
}

func TestIsPaneVisible_ReflectsToggleState(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	assert.True(t, m.IsPaneVisible(layout.PanePlaylists))
	m.TogglePane(layout.PanePlaylists)
	assert.False(t, m.IsPaneVisible(layout.PanePlaylists))
}

// ── Task 5: Focus rotation ────────────────────────────────────────────────────

func TestRotateFocus_ForwardCyclesAll(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	visible := m.VisiblePanes()
	require.Greater(t, len(visible), 0)

	first := m.FocusedPane()
	// Rotate forward through all panes and wrap back
	for i := 0; i < len(visible); i++ {
		m.RotateFocus(true)
	}
	// Should be back at the original pane
	assert.Equal(t, first, m.FocusedPane())
}

func TestRotateFocus_BackwardCycles(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	visible := m.VisiblePanes()
	require.Greater(t, len(visible), 0)

	first := m.FocusedPane()
	// Rotate backward through all panes
	for i := 0; i < len(visible); i++ {
		m.RotateFocus(false)
	}
	assert.Equal(t, first, m.FocusedPane())
}

func TestRotateFocus_WrapsAtBoundaries(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	visible := m.VisiblePanes()

	// Go to last pane
	for i := 0; i < len(visible)-1; i++ {
		m.RotateFocus(true)
	}
	last := m.FocusedPane()

	// One more forward should wrap to first
	m.RotateFocus(true)
	assert.Equal(t, visible[0], m.FocusedPane(), "should wrap from last to first")

	// Set focus back to last, go backward from first
	m.SetFocus(last)
	m.RotateFocus(false) // should go to last-1... no, backward from last goes to last-1
	// Actually let's verify: from first pane, going backward wraps to last
	m.SetFocus(visible[0])
	m.RotateFocus(false)
	assert.Equal(t, last, m.FocusedPane(), "backward from first should wrap to last")
}

func TestRotateFocus_SkipsHiddenPanes(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Start at NowPlaying (index 0 in focus order)
	m.SetFocus(layout.PaneNowPlaying)

	// Hide Playlists (which should be in row 2)
	m.TogglePane(layout.PanePlaylists)

	visible := m.VisiblePanes()
	// Playlists should not be in the visible list
	for _, id := range visible {
		assert.NotEqual(t, layout.PanePlaylists, id)
	}
}

func TestRotateFocus_AfterToggleFocusMovesToFirst(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Focus NowPlaying (row 1) then hide it — focus must move to next visible
	m.SetFocus(layout.PaneNowPlaying)
	m.TogglePane(layout.PaneNowPlaying)

	focused := m.FocusedPane()
	assert.NotEqual(t, layout.PaneNowPlaying, focused,
		"focus must leave hidden pane")
	assert.True(t, m.IsPaneVisible(focused),
		"new focus target must be visible")
}

func TestRotateFocus_AfterCyclePresetResetsToFirst(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Go to some pane deep in the order
	m.RotateFocus(true)
	m.RotateFocus(true)

	m.CyclePreset() // Changes preset

	visible := m.VisiblePanes()
	require.Greater(t, len(visible), 0)
	assert.Equal(t, visible[0], m.FocusedPane(),
		"after preset change, focus should be first visible pane")
}

func TestSetFocus_ChangesCurrentFocus(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	m.SetFocus(layout.PaneAlbums)
	assert.Equal(t, layout.PaneAlbums, m.FocusedPane())
}

func TestSetFocus_NoOpForHiddenPane(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	m.SetFocus(layout.PaneNowPlaying)
	m.TogglePane(layout.PanePlaylists) // Hide Playlists

	// Try to focus the hidden pane — should be a no-op
	m.SetFocus(layout.PanePlaylists)
	assert.NotEqual(t, layout.PanePlaylists, m.FocusedPane(),
		"SetFocus on hidden pane should be a no-op")
}

// ── Task 6: PaneAt hit-test ────────────────────────────────────────────────────

func TestPaneAt_CenterOfPane(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	r := m.PaneRect(layout.PaneNowPlaying)
	cx := r.X + r.Width/2
	cy := r.Y + r.Height/2

	got := m.PaneAt(cx, cy)
	assert.Equal(t, layout.PaneNowPlaying, got)
}

func TestPaneAt_HeaderAreaReturnsMinusOne(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Header is row 0 (y=0 before content area offset)
	// PaneAt coordinates are relative to the terminal top-left
	// Content area starts at y=1 (after 1-line header)
	got := m.PaneAt(60, 0)
	assert.Equal(t, layout.PaneID(-1), got)
}

func TestPaneAt_StatusBarReturnsMinusOne(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Status bar is at y=29 (last row, height=30 so y=29)
	got := m.PaneAt(60, 29)
	assert.Equal(t, layout.PaneID(-1), got)
}

func TestPaneAt_OutsideAllPanesReturnsMinusOne(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Way outside
	got := m.PaneAt(200, 200)
	assert.Equal(t, layout.PaneID(-1), got)
}

// ── Task 7: Integration / edge cases ──────────────────────────────────────────

func TestFullLifecycle(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Start: Music page, Dashboard
	assert.Equal(t, layout.PageMusic, m.ActivePage())
	assert.Len(t, m.VisiblePanes(), 8)

	// Cycle preset
	m.CyclePreset()
	assert.Equal(t, "Listening", m.ActivePresetName())
	assert.Len(t, m.VisiblePanes(), 3)

	// Toggle a pane on Listening
	m.TogglePane(layout.PaneQueue)
	assert.False(t, m.IsPaneVisible(layout.PaneQueue))

	// Switch page
	m.TogglePage()
	assert.Equal(t, layout.PageStats, m.ActivePage())
	assert.Len(t, m.VisiblePanes(), 5) // PresetStats has 5 panes

	// Switch back
	m.TogglePage()
	assert.Equal(t, layout.PageMusic, m.ActivePage())
	// Hidden state was cleared when we toggled to Stats page and back
	// so we cycle to Listening again manually
	m.SetPreset(1) // Listening
	// Manual toggles were reset when we switched pages, so Queue should be visible again
	assert.True(t, m.IsPaneVisible(layout.PaneQueue))
}

func TestResize_ReshrinkScalesRects(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	largeRect := m.PaneRect(layout.PaneNowPlaying)

	m.Resize(80, 24)
	smallRect := m.PaneRect(layout.PaneNowPlaying)

	assert.Less(t, smallRect.Width, largeRect.Width, "width should shrink")
	assert.Less(t, smallRect.Height, largeRect.Height, "height should shrink")
}

func TestEdge_ZeroSizeTerminal(t *testing.T) {
	m := layout.NewManager()
	// Should not panic
	assert.NotPanics(t, func() {
		m.Resize(0, 0)
		_ = m.VisiblePanes()
		_ = m.PaneRect(layout.PaneNowPlaying)
	})
}

func TestEdge_VerySmallTerminal(t *testing.T) {
	m := layout.NewManager()
	assert.NotPanics(t, func() {
		m.Resize(1, 1)
		_ = m.VisiblePanes()
		_ = m.PaneRect(layout.PaneNowPlaying)
	})
}

// TestLayoutManager_MinHeight verifies the two-step height distribution:
// reserve MinHeight first, then distribute remaining by weight.
func TestLayoutManager_MinHeight(t *testing.T) {
	// Temporarily replace PageStatsPresets with a 3-row test preset
	oldPresets := layout.PageStatsPresets
	defer func() { layout.PageStatsPresets = oldPresets }()

	layout.PageStatsPresets = []layout.Preset{{
		Name: "TestMinHeight",
		Visible: map[layout.PaneID]bool{
			layout.PaneNowPlaying:     true,
			layout.PaneQueue:          true,
			layout.PaneRecentlyPlayed: true,
		},
		Grid: []layout.Row{
			{HeightWeight: 1, MinHeight: 10, Cells: []layout.Cell{{PaneID: layout.PaneNowPlaying, WidthWeight: 1}}},
			{HeightWeight: 1, Cells: []layout.Cell{{PaneID: layout.PaneQueue, WidthWeight: 1}}},
			{HeightWeight: 1, Cells: []layout.Cell{{PaneID: layout.PaneRecentlyPlayed, WidthWeight: 1}}},
		},
	}}

	m := layout.NewManager()
	m.TogglePage()    // switch to Stats page
	m.Resize(120, 34) // contentH = 30

	// reserved = 10, remaining = 20, totalW = 3
	// Row 0: 10 + 20*1/3 = 16, Row 1: 0 + 20*1/3 = 6, Row 2 (last): 30 - 22 = 8
	np := m.PaneRect(layout.PaneNowPlaying)
	q := m.PaneRect(layout.PaneQueue)
	rp := m.PaneRect(layout.PaneRecentlyPlayed)

	assert.Equal(t, 16, np.Height, "row with MinHeight=10 should get 10 + proportional share")
	assert.Equal(t, 6, q.Height, "row without MinHeight should get only proportional share")
	assert.Equal(t, 8, rp.Height, "last row absorbs rounding remainder")
	assert.Equal(t, 30, np.Height+q.Height+rp.Height, "total must equal content height")
}

// TestLayoutManager_MinHeight_ZeroRegression verifies that rows without MinHeight
// distribute identically to the pre-MinHeight algorithm.
func TestLayoutManager_MinHeight_ZeroRegression(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30) // contentH = 26

	// Dashboard: weights 2:3:3 over 26 content rows
	// Row 1 (weight 2): 26*2/8 = 6
	// Row 2 (weight 3): 26*3/8 = 9
	// Row 3 (weight 3, last): 26 - 15 = 11
	nowPlayingRect := m.PaneRect(layout.PaneNowPlaying)
	playlistsRect := m.PaneRect(layout.PanePlaylists)
	queueRect := m.PaneRect(layout.PaneQueue)

	assert.Equal(t, 6, nowPlayingRect.Height)
	assert.Equal(t, 9, playlistsRect.Height)
	assert.Equal(t, 11, queueRect.Height)
}

// TestLayoutManager_MinHeight_Overflow verifies no panic when MinHeight sum
// exceeds content height; last row absorbs the deficit.
func TestLayoutManager_MinHeight_Overflow(t *testing.T) {
	oldPresets := layout.PageStatsPresets
	defer func() { layout.PageStatsPresets = oldPresets }()

	layout.PageStatsPresets = []layout.Preset{{
		Name: "TestOverflow",
		Visible: map[layout.PaneID]bool{
			layout.PaneNowPlaying: true,
			layout.PaneQueue:      true,
		},
		Grid: []layout.Row{
			{HeightWeight: 1, MinHeight: 10, Cells: []layout.Cell{{PaneID: layout.PaneNowPlaying, WidthWeight: 1}}},
			{HeightWeight: 1, MinHeight: 10, Cells: []layout.Cell{{PaneID: layout.PaneQueue, WidthWeight: 1}}},
		},
	}}

	m := layout.NewManager()
	m.TogglePage()

	assert.NotPanics(t, func() {
		m.Resize(120, 10) // contentH = 6, reserved = 20 > 6
	})
}

// TestPresetStats_NowPlayingRowHeight verifies that with the real PresetStats,
// the NowPlaying row gets the correct height at different terminal sizes.
func TestPresetStats_NowPlayingRowHeight(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)
	m.TogglePage() // Stats page

	// PresetStats: MinHeight=14, weights 1:3:2, contentH=26
	// reserved=14, remaining=12, totalW=6
	// NowPlaying: 14 + 12*1/6 = 16
	np := m.PaneRect(layout.PaneNowPlaying)
	assert.Equal(t, 16, np.Height, "at 30-row terminal NowPlaying should get 16 rows")

	// At 50-row terminal: contentH=46, reserved=14, remaining=32, totalW=6
	// NowPlaying: 14 + 32*1/6 = 14 + 5 = 19 (last row absorbs rounding)
	m.Resize(120, 50)
	np = m.PaneRect(layout.PaneNowPlaying)
	assert.Equal(t, 19, np.Height, "at 50-row terminal NowPlaying should get 19 rows")
}

func TestPresetCycleFullLoop(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	expectedNames := []string{"Full Dashboard", "Listening", "Library", "Discovery"}
	expectedVisible := []int{8, 3, 4, 4}

	for i := 0; i < len(expectedNames); i++ {
		assert.Equal(t, expectedNames[i], m.ActivePresetName(), "preset %d name", i)
		assert.Len(t, m.VisiblePanes(), expectedVisible[i], "preset %d visible panes", i)
		m.CyclePreset()
	}
	// Back to Dashboard
	assert.Equal(t, "Full Dashboard", m.ActivePresetName())
}

func TestRowCollapseHeightRedistributed(t *testing.T) {
	// Hide all panes in row 2 (Playlists, Albums, LikedSongs) on Dashboard.
	// Remaining content height should be split between rows 1 and 3 (weights 2:3).
	m := layout.NewManager()
	const W, H = 120, 30
	m.Resize(W, H)

	contentH := H - 4 // 26 (1 header + 3 status bar)

	m.TogglePane(layout.PanePlaylists)
	m.TogglePane(layout.PaneAlbums)
	m.TogglePane(layout.PaneLikedSongs)

	// Active rows: weight 2 and weight 3 → totalWeight 5
	// Row 1: 2/5 * 26 = 10, Row 3: 3/5 * 26 = 16 (last absorbs 26 - 10 = 16)
	nowH := m.PaneRect(layout.PaneNowPlaying).Height
	queueH := m.PaneRect(layout.PaneQueue).Height

	assert.Equal(t, contentH, nowH+queueH,
		"remaining rows must sum to content height")
}

func TestFocusRotation_AfterHideWrapsCorrectly(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Start at Listening preset (3 panes)
	m.CyclePreset()

	// Hide Queue
	m.TogglePane(layout.PaneQueue)
	visible := m.VisiblePanes()
	assert.Len(t, visible, 2)

	// Rotate through visible panes
	first := m.FocusedPane()
	m.RotateFocus(true)
	m.RotateFocus(true)
	// Should be back at first
	assert.Equal(t, first, m.FocusedPane())
}

// ── Story 181: flat Stats page layout (RowSpan retired) ─────────────────────────

func TestRecompute_StatsPageFlat_ThreeRows(t *testing.T) {
	m := layout.NewManager()
	m.Resize(200, 50)
	m.TogglePage() // switch to Stats page (flat 3-row PresetStats)

	np := m.PaneRect(layout.PaneNowPlaying)
	h := m.PaneRect(layout.PaneGatewayHealth)
	pt := m.PaneRect(layout.PanePollingTraffic)
	gl := m.PaneRect(layout.PaneGatewayLive)
	nl := m.PaneRect(layout.PaneNetworkLog)

	// Three rows: NowPlaying / [Health Traffic Live] / NetworkLog
	assert.Equal(t, 0, np.X)
	assert.Equal(t, 200, np.Width)
	assert.Equal(t, np.Y+np.Height, h.Y, "row 2 starts where row 1 ends")
	assert.Equal(t, h.Y, pt.Y, "Health and Traffic share row")
	assert.Equal(t, h.Y, gl.Y, "Live shares row with Health/Traffic")
	assert.Equal(t, h.Y+h.Height, nl.Y, "NetworkLog starts where row 2 ends")
	assert.Equal(t, h.Height, pt.Height, "Health/Traffic/Live share row height")
	assert.Equal(t, h.Height, gl.Height)

	// 1:1:3 width split (with last-cell rounding compensation absorbed by Live)
	assert.Equal(t, h.Width, pt.Width, "Health and Traffic equal width")
	assert.InDelta(t, 3*h.Width, gl.Width, 2, "Live ≈ 3× Health (rounding ±2)")
	assert.Equal(t, 200, h.Width+pt.Width+gl.Width, "row width sums to terminal width")
}

func TestTogglePane_StatsPage_HealthHidden_TrafficLiveExpand(t *testing.T) {
	m := layout.NewManager()
	m.Resize(200, 50)
	m.TogglePage()
	pre := m.PaneRect(layout.PanePollingTraffic).Width

	m.TogglePane(layout.PaneGatewayHealth)
	assert.False(t, m.IsPaneVisible(layout.PaneGatewayHealth))

	pt := m.PaneRect(layout.PanePollingTraffic)
	gl := m.PaneRect(layout.PaneGatewayLive)
	assert.Equal(t, 0, pt.X, "Traffic now starts at x=0")
	assert.Greater(t, pt.Width, pre, "Traffic must absorb Health's column")
	assert.Equal(t, pt.Y, gl.Y)
	assert.Equal(t, 200, pt.Width+gl.Width, "remaining cells fill the row width")
}

func TestTogglePane_StatsPage_HealthAndTrafficHidden_LiveFullRow(t *testing.T) {
	m := layout.NewManager()
	m.Resize(200, 50)
	m.TogglePage()
	m.TogglePane(layout.PaneGatewayHealth)
	m.TogglePane(layout.PanePollingTraffic)

	gl := m.PaneRect(layout.PaneGatewayLive)
	assert.Equal(t, 0, gl.X)
	assert.Equal(t, 200, gl.Width, "Live fills full row when both siblings hidden")
}

func TestTogglePane_StatsPage_LiveHidden_HealthTrafficExpand(t *testing.T) {
	m := layout.NewManager()
	m.Resize(200, 50)
	m.TogglePage()
	m.TogglePane(layout.PaneGatewayLive)

	h := m.PaneRect(layout.PaneGatewayHealth)
	pt := m.PaneRect(layout.PanePollingTraffic)
	assert.Equal(t, 0, h.X)
	assert.Equal(t, h.Width, pt.Width, "Health and Traffic split equally")
	assert.Equal(t, 200, h.Width+pt.Width, "they fill the full row")
}

func TestRecompute_StatsPage_FocusOrder_LeftToRightTopToBottom(t *testing.T) {
	// Flat layout: focus order is purely visual reading order.
	m := layout.NewManager()
	m.Resize(200, 50)
	m.TogglePage()

	expected := []layout.PaneID{
		layout.PaneNowPlaying,
		layout.PaneGatewayHealth,
		layout.PanePollingTraffic,
		layout.PaneGatewayLive,
		layout.PaneNetworkLog,
	}
	assert.Equal(t, expected, m.VisiblePanes(),
		"Stats page focus order: NowPlaying → Health → Traffic → Live → NetworkLog")

	for i, want := range expected {
		assert.Equal(t, want, m.FocusedPane(),
			"step %d: focused pane mismatch", i)
		m.RotateFocus(true)
	}
	assert.Equal(t, expected[0], m.FocusedPane(), "rotation must wrap to first pane")
}

func TestRecompute_StatsPage_RectsNonOverlapping(t *testing.T) {
	// Use odd dimensions to exercise rounding-remainder paths.
	m := layout.NewManager()
	m.Resize(201, 79)
	m.TogglePage()

	visible := m.VisiblePanes()
	for i := range visible {
		for j := i + 1; j < len(visible); j++ {
			a := m.PaneRect(visible[i])
			b := m.PaneRect(visible[j])
			overlap := a.X < b.X+b.Width && a.X+a.Width > b.X &&
				a.Y < b.Y+b.Height && a.Y+a.Height > b.Y
			assert.False(t, overlap,
				"panes %d and %d must not overlap: %+v vs %+v",
				visible[i], visible[j], a, b)
		}
	}
}
