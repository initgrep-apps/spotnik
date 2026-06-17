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
	assert.Equal(t, layout.PagePlayer, m.ActivePage())
	assert.Equal(t, 0, m.ActivePresetIndex())
	assert.Equal(t, "Dashboard", m.ActivePresetName())
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
	// Dashboard: weights 1:3:3 over 26 content rows (30 - 1 header - 3 status)
	// MinHeight sum = 6 (NowPlaying row only, story 223).
	// totalW = 7, remaining = 26-6 = 20
	// Row 0: 6 + 20*1/7 = 6 + 2 = 8
	// Row 1: 0 + 20*3/7 = 8
	// Row 2 (last): 26 - 8 - 8 = 10
	m := layout.NewManager()
	m.Resize(120, 30) // content height = 26

	nowPlayingRect := m.PaneRect(layout.PaneNowPlaying)
	assert.Equal(t, 8, nowPlayingRect.Height)

	playlistsRect := m.PaneRect(layout.PanePlaylists)
	assert.Equal(t, 8, playlistsRect.Height)

	queueRect := m.PaneRect(layout.PaneQueue)
	assert.Equal(t, 10, queueRect.Height)
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

func TestTogglePage_TwoCycle(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	assert.Equal(t, layout.PagePlayer, m.ActivePage())
	m.TogglePage()
	assert.Equal(t, layout.PageStats, m.ActivePage())
	m.TogglePage()
	assert.Equal(t, layout.PagePlayer, m.ActivePage())
}

func TestTogglePage_ClearsHiddenState(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Hide a pane on Music page
	m.TogglePane(layout.PanePlaylists)
	assert.False(t, m.IsPaneVisible(layout.PanePlaylists))

	// Switch page (Player → Stats → Player)
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
	m.CyclePreset()
	assert.Equal(t, 4, m.ActivePresetIndex())
	m.CyclePreset()
	assert.Equal(t, 5, m.ActivePresetIndex())
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
	m.CyclePreset() // Podcast — Playlists not visible
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
	m.TogglePage() // Player → Stats

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
	m.TogglePage() // Player → Stats

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
	m.TogglePage() // Player → Stats

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

	m.TogglePage() // Player → Stats page — PresetStats has 5 panes

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
	assert.Equal(t, layout.PagePlayer, m.ActivePage())
	assert.Len(t, m.VisiblePanes(), 8)

	// Cycle preset
	m.CyclePreset()
	assert.Equal(t, "Listening", m.ActivePresetName())
	assert.Len(t, m.VisiblePanes(), 3)

	// Toggle a pane on Listening
	m.TogglePane(layout.PaneQueue)
	assert.False(t, m.IsPaneVisible(layout.PaneQueue))

	// Switch page (Player → Stats)
	m.TogglePage()
	assert.Equal(t, layout.PageStats, m.ActivePage())
	assert.Len(t, m.VisiblePanes(), 5) // PresetStats has 5 panes

	// Switch back (Stats → Player)
	m.TogglePage()
	assert.Equal(t, layout.PagePlayer, m.ActivePage())
	// Hidden state was cleared when we toggled through pages and back
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
	// Height: at 120x30 NowPlaying gets MinHeight=6 (story 223). At 80x24,
	// contentH=20, MinHeight sum=6, remaining=14, so NowPlaying gets max(6, 2)=6.
	// Heights are equal (both 6), so assert.LessOrEqual instead of Less.
	assert.LessOrEqual(t, smallRect.Height, largeRect.Height, "height should not grow")
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
	m.TogglePage()    // Player → Stats
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
	// Use a custom preset with no MinHeight so the pure weight-based
	// distribution can be verified independently of the Dashboard preset.
	oldPresets := layout.PagePlayerPresets
	defer func() { layout.PagePlayerPresets = oldPresets }()

	layout.PagePlayerPresets = []layout.Preset{{
		Name: "TestNoMinHeight",
		Visible: map[layout.PaneID]bool{
			layout.PaneNowPlaying: true,
			layout.PaneQueue:      true,
			layout.PanePlaylists:  true,
		},
		Grid: []layout.Row{
			{HeightWeight: 2, Cells: []layout.Cell{{PaneID: layout.PaneNowPlaying, WidthWeight: 1}}},
			{HeightWeight: 3, Cells: []layout.Cell{{PaneID: layout.PaneQueue, WidthWeight: 1}}},
			{HeightWeight: 3, Cells: []layout.Cell{{PaneID: layout.PanePlaylists, WidthWeight: 1}}},
		},
	}}

	m := layout.NewManager()
	m.Resize(120, 30) // contentH = 26

	// weights 2:3:3 over 26 content rows
	// Row 0 (weight 2): 26*2/8 = 6
	// Row 1 (weight 3): 26*3/8 = 9
	// Row 2 (weight 3, last): 26 - 15 = 11
	nowPlayingRect := m.PaneRect(layout.PaneNowPlaying)
	queueRect := m.PaneRect(layout.PaneQueue)
	playlistsRect := m.PaneRect(layout.PanePlaylists)

	assert.Equal(t, 6, nowPlayingRect.Height)
	assert.Equal(t, 9, queueRect.Height)
	assert.Equal(t, 11, playlistsRect.Height)
}

// TestLayoutManager_MinHeight_Overflow verifies no panic when MinHeight sum
// exceeds content height; earlier rows get their MinHeight, last row clamped to 0.
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
	m.TogglePage() // Player → Stats

	assert.NotPanics(t, func() {
		m.Resize(120, 10) // contentH = 6, reserved = 20 > 6
	})

	// earlier row gets its MinHeight; last row clamped to 0
	npRect := m.PaneRect(layout.PaneNowPlaying)
	queueRect := m.PaneRect(layout.PaneQueue)
	assert.Equal(t, 10, npRect.Height)
	assert.Equal(t, 0, queueRect.Height)
}

// TestPresetStats_NowPlayingRowHeight verifies that with the real PresetStats,
// the NowPlaying row gets the correct height at different terminal sizes.
func TestPresetStats_NowPlayingRowHeight(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)
	m.TogglePage() // Player → Stats

	// PresetStats: weights 1:3:3, contentH=26, MinHeight sum=6
	// totalW=7, remaining=26-6=20
	// NowPlaying: 6 + 20*1/7 = 6 + 2 = 8
	np := m.PaneRect(layout.PaneNowPlaying)
	assert.Equal(t, 8, np.Height, "at 30-row terminal NowPlaying should get 8 rows")

	// At 50-row terminal: contentH=46, MinHeight sum=6
	// totalW=7, remaining=46-6=40
	// NowPlaying: 6 + 40*1/7 = 6 + 5 = 11
	m.Resize(120, 50)
	np = m.PaneRect(layout.PaneNowPlaying)
	assert.Equal(t, 11, np.Height, "at 50-row terminal NowPlaying should get 11 rows")
}

func TestPresetCycleFullLoop(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	expectedNames := []string{"Dashboard", "Listening", "Podcast", "Library", "Discovery", "Podcast Dashboard"}
	expectedVisible := []int{8, 3, 3, 4, 4, 4}

	for i := 0; i < len(expectedNames); i++ {
		assert.Equal(t, expectedNames[i], m.ActivePresetName(), "preset %d name", i)
		assert.Len(t, m.VisiblePanes(), expectedVisible[i], "preset %d visible panes", i)
		m.CyclePreset()
	}
	// Back to Dashboard
	assert.Equal(t, "Dashboard", m.ActivePresetName())
}

func TestRowCollapseHeightRedistributed(t *testing.T) {
	// Hide all panes in row 2 (Playlists, Albums, LikedSongs) on Dashboard.
	// Remaining content height should be split between rows 1 and 3 (weights 1:3).
	m := layout.NewManager()
	const W, H = 120, 30
	m.Resize(W, H)

	contentH := H - 4 // 26 (1 header + 3 status bar)

	m.TogglePane(layout.PanePlaylists)
	m.TogglePane(layout.PaneAlbums)
	m.TogglePane(layout.PaneLikedSongs)

	// Active rows: weight 1 and weight 3 → totalWeight 4
	// Row 1: 1/4 * 26 = 6, Row 3: 3/4 * 26 = 20 (last absorbs 26 - 6 = 20)
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

	m.TogglePage() // Player → Stats page (flat 3-row PresetStats)

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
	m.TogglePage() // Player → Stats
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
	m.TogglePage() // Player → Stats
	m.TogglePane(layout.PaneGatewayHealth)
	m.TogglePane(layout.PanePollingTraffic)

	gl := m.PaneRect(layout.PaneGatewayLive)
	assert.Equal(t, 0, gl.X)
	assert.Equal(t, 200, gl.Width, "Live fills full row when both siblings hidden")
}

func TestTogglePane_StatsPage_LiveHidden_HealthTrafficExpand(t *testing.T) {
	m := layout.NewManager()
	m.Resize(200, 50)
	m.TogglePage() // Player → Stats
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
	m.TogglePage() // Player → Stats

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
	m.TogglePage() // Player → Stats

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

// ── Story 233: Player page unification ─────────────────────────────────────────

// ── SwitchToPage tests ──────────────────────────────────────────────────────────

func TestSwitchToPage_SwitchesAndResets(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	require.Equal(t, layout.PagePlayer, m.ActivePage())
	require.Equal(t, 0, m.ActivePresetIndex())

	m.SwitchToPage(layout.PageStats)
	assert.Equal(t, layout.PageStats, m.ActivePage())
	assert.Equal(t, 0, m.ActivePresetIndex(), "preset index should reset to 0")
}

func TestSwitchToPage_NoOpIfAlreadyOnPage(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	require.Equal(t, layout.PagePlayer, m.ActivePage())
	presetBefore := m.ActivePresetIndex()

	m.SwitchToPage(layout.PagePlayer)
	assert.Equal(t, layout.PagePlayer, m.ActivePage())
	assert.Equal(t, presetBefore, m.ActivePresetIndex(), "preset index should not change")
}

func TestSwitchToPage_ResetsHiddenPanes(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Hide a pane on Player page
	m.TogglePane(layout.PanePlaylists)
	require.False(t, m.IsPaneVisible(layout.PanePlaylists))

	// Switch to Stats — hidden state must be cleared
	m.SwitchToPage(layout.PageStats)
	assert.Equal(t, layout.PageStats, m.ActivePage())

	// Switch back to Player — hidden state must be cleared
	m.SwitchToPage(layout.PagePlayer)
	assert.True(t, m.IsPaneVisible(layout.PanePlaylists), "hidden panes should be cleared after SwitchToPage")
}

func TestSwitchToPage_ResetsFocusIndex(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Move focus past the first pane
	m.RotateFocus(true)
	m.RotateFocus(true)
	focusBefore := m.FocusedPane()
	require.NotEqual(t, layout.PaneNowPlaying, focusBefore)

	// Switching pages resets focus to first visible pane
	m.SwitchToPage(layout.PageStats)
	visible := m.VisiblePanes()
	assert.Equal(t, visible[0], m.FocusedPane(), "focus should reset to first visible pane after SwitchToPage")
}

func TestPagePlayer_Value(t *testing.T) {
	assert.Equal(t, layout.PageID(0), layout.PagePlayer)
}

func TestPageStats_Value(t *testing.T) {
	assert.Equal(t, layout.PageID(1), layout.PageStats)
}

func TestTogglePage_PlayerStatsTwoCycle(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	assert.Equal(t, layout.PagePlayer, m.ActivePage())
	m.TogglePage()
	assert.Equal(t, layout.PageStats, m.ActivePage())
	m.TogglePage()
	assert.Equal(t, layout.PagePlayer, m.ActivePage())
}

func TestSetPreset_DirectSwitch(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	assert.Equal(t, 0, m.ActivePresetIndex())
	assert.Equal(t, "Dashboard", m.ActivePresetName())

	m.SetPreset(2) // Podcast
	assert.Equal(t, 2, m.ActivePresetIndex())
	assert.Equal(t, "Podcast", m.ActivePresetName())

	m.SetPreset(-1) // out of range, no-op
	assert.Equal(t, 2, m.ActivePresetIndex())

	m.SetPreset(99) // out of range, no-op
	assert.Equal(t, 2, m.ActivePresetIndex())
}

func TestActivePreset_ReturnsCorrectPreset(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	tests := []struct {
		presetIdx int
		wantName  string
	}{
		{0, "Dashboard"},
		{1, "Listening"},
		{2, "Podcast"},
		{3, "Library"},
		{4, "Discovery"},
		{5, "Podcast Dashboard"},
	}

	for _, tt := range tests {
		m.SetPreset(tt.presetIdx)
		got := m.ActivePreset()
		assert.Equal(t, tt.wantName, got.Name)
		assert.NotEmpty(t, got.Visible)
		assert.NotEmpty(t, got.Grid)
	}
}

func TestActivePreset_OutOfBounds(t *testing.T) {
	// Switch to Stats page (1 preset) and set preset index beyond the single entry.
	m := layout.NewManager()
	m.Resize(120, 30)
	m.TogglePage() // Player → Stats
	require.Equal(t, 0, m.ActivePresetIndex())
	m.CyclePreset() // wraps to 0, still valid
	require.Equal(t, 0, m.ActivePresetIndex())

	// ActivePreset on Stats page with index 0 returns the correct preset.
	got := m.ActivePreset()
	assert.Equal(t, "Stats", got.Name)
	assert.NotEmpty(t, got.Visible)
}

func TestActivePreset_ReturnsZeroValueForOutOfBounds(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Valid preset returns non-zero Preset.
	got := m.ActivePreset()
	assert.Equal(t, "Dashboard", got.Name)
	assert.NotEmpty(t, got.Grid)

	// Force out-of-bounds index on Player page (6 presets: 0-5).
	m.SetActivePresetIndex(99)
	got = m.ActivePreset()
	assert.Equal(t, layout.Preset{}, got)
	assert.Empty(t, got.Name)
	assert.Empty(t, got.Grid)
	assert.Empty(t, got.Visible)
}

func TestIsPaneVisible_CurrentPreset(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)

	// Dashboard: FollowedShows not visible
	assert.False(t, m.IsPaneVisible(layout.PaneFollowedShows))

	// Switch to Podcast preset
	m.SetPreset(2) // Podcast
	assert.True(t, m.IsPaneVisible(layout.PaneFollowedShows))
}

func TestPagePlayerPresets_HasSixEntries(t *testing.T) {
	assert.Len(t, layout.PagePlayerPresets, 6)
	assert.Equal(t, "Dashboard", layout.PagePlayerPresets[0].Name)
	assert.Equal(t, "Listening", layout.PagePlayerPresets[1].Name)
	assert.Equal(t, "Podcast", layout.PagePlayerPresets[2].Name)
	assert.Equal(t, "Library", layout.PagePlayerPresets[3].Name)
	assert.Equal(t, "Discovery", layout.PagePlayerPresets[4].Name)
	assert.Equal(t, "Podcast Dashboard", layout.PagePlayerPresets[5].Name)
}

func TestPresetPodcast_PlayerGrid(t *testing.T) {
	assert.Equal(t, "Podcast", layout.PresetPodcast.Name)
	require.Len(t, layout.PresetPodcast.Grid, 2)
	require.Len(t, layout.PresetPodcast.Grid[1].Cells, 2)
	assert.Equal(t, layout.PaneQueue, layout.PresetPodcast.Grid[1].Cells[0].PaneID)
	assert.Equal(t, layout.PaneFollowedShows, layout.PresetPodcast.Grid[1].Cells[1].PaneID)
	assert.Equal(t, 45, layout.PresetPodcast.Grid[1].Cells[0].WidthWeight)
	assert.Equal(t, 55, layout.PresetPodcast.Grid[1].Cells[1].WidthWeight)
}

func TestPresetPodcastDashboard_PlayerGrid(t *testing.T) {
	assert.Equal(t, "Podcast Dashboard", layout.PresetPodcastDashboard.Name)
	require.Len(t, layout.PresetPodcastDashboard.Grid, 2)
	require.Len(t, layout.PresetPodcastDashboard.Grid[1].Cells, 3)
	assert.Equal(t, layout.PaneQueue, layout.PresetPodcastDashboard.Grid[1].Cells[0].PaneID)
	assert.Equal(t, layout.PaneFollowedShows, layout.PresetPodcastDashboard.Grid[1].Cells[1].PaneID)
	assert.Equal(t, layout.PaneSavedEpisodes, layout.PresetPodcastDashboard.Grid[1].Cells[2].PaneID)
}

func TestPaneIDs_NoPodcastPlaybackOrShowEpisodes(t *testing.T) {
	// PanePodcastPlayback and PaneShowEpisodes were removed in the player page unification.
	// Verify that follow-on PaneIDs (FollowedShows, SavedEpisodes) still have valid values.
	assert.GreaterOrEqual(t, int(layout.PaneFollowedShows), 0)
	assert.GreaterOrEqual(t, int(layout.PaneSavedEpisodes), 0)
}

func TestPlayerPage_PodcastPresetPaneRects(t *testing.T) {
	m := layout.NewManager()
	m.Resize(120, 30)
	m.SetPreset(2) // Podcast preset

	visible := m.VisiblePanes()
	require.Greater(t, len(visible), 0, "podcast preset should have visible panes")

	for _, id := range visible {
		r := m.PaneRect(id)
		assert.Greater(t, r.Width, 0, "pane %d should have positive width", id)
		assert.Greater(t, r.Height, 0, "pane %d should have positive height", id)
	}
}
