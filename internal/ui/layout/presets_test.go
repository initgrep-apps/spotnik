package layout_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collectGridPanes collects all unique PaneIDs referenced in a preset's grid.
func collectGridPanes(p layout.Preset) map[layout.PaneID]bool {
	panes := make(map[layout.PaneID]bool)
	for _, row := range p.Grid {
		for _, cell := range row.Cells {
			panes[cell.PaneID] = true
		}
	}
	return panes
}

func TestPreset_GridConsistency(t *testing.T) {
	// Every pane referenced in a preset's Grid must be in Visible.
	// (Visible may contain panes that appear in Grid — they form the canonical set.)
	presets := []layout.Preset{
		layout.PresetDashboard,
		layout.PresetListening,
		layout.PresetLibrary,
		layout.PresetDiscovery,
		layout.PresetNerdStatus,
	}

	for _, p := range presets {
		t.Run(p.Name, func(t *testing.T) {
			gridPanes := collectGridPanes(p)
			for id := range gridPanes {
				assert.True(t, p.Visible[id],
					"pane %d is in Grid but not in Visible map", id)
			}
		})
	}
}

func TestPresetDashboard(t *testing.T) {
	p := layout.PresetDashboard
	assert.Equal(t, "Full Dashboard", p.Name)
	assert.Len(t, p.Visible, 8, "should have 8 visible panes")
	assert.Len(t, p.Grid, 3, "should have 3 rows")

	// Row 1: NowPlaying full-width (weight 2)
	require.Len(t, p.Grid[0].Cells, 1)
	assert.Equal(t, 2, p.Grid[0].HeightWeight)
	assert.Equal(t, layout.PaneNowPlaying, p.Grid[0].Cells[0].PaneID)

	// Row 2: Playlists, Albums, LikedSongs (weight 3)
	require.Len(t, p.Grid[1].Cells, 3)
	assert.Equal(t, 3, p.Grid[1].HeightWeight)

	// Row 3: Queue, RecentlyPlayed, TopTracks, TopArtists (weight 3)
	require.Len(t, p.Grid[2].Cells, 4)
	assert.Equal(t, 3, p.Grid[2].HeightWeight)
}

func TestPresetListening(t *testing.T) {
	p := layout.PresetListening
	assert.Equal(t, "Listening", p.Name)
	assert.Len(t, p.Visible, 3, "should have 3 visible panes")
	assert.Len(t, p.Grid, 2, "should have 2 rows")

	assert.True(t, p.Visible[layout.PaneNowPlaying])
	assert.True(t, p.Visible[layout.PaneQueue])
	assert.True(t, p.Visible[layout.PaneRecentlyPlayed])
}

func TestPresetLibrary(t *testing.T) {
	p := layout.PresetLibrary
	assert.Equal(t, "Library", p.Name)
	assert.Len(t, p.Visible, 4, "should have 4 visible panes")
	assert.Len(t, p.Grid, 2, "should have 2 rows")

	assert.True(t, p.Visible[layout.PaneNowPlaying])
	assert.True(t, p.Visible[layout.PanePlaylists])
	assert.True(t, p.Visible[layout.PaneAlbums])
	assert.True(t, p.Visible[layout.PaneLikedSongs])
}

func TestPresetDiscovery(t *testing.T) {
	p := layout.PresetDiscovery
	assert.Equal(t, "Discovery", p.Name)
	assert.Len(t, p.Visible, 4, "should have 4 visible panes")
	assert.Len(t, p.Grid, 3, "should have 3 rows")

	assert.True(t, p.Visible[layout.PaneNowPlaying])
	assert.True(t, p.Visible[layout.PaneTopTracks])
	assert.True(t, p.Visible[layout.PaneTopArtists])
	assert.True(t, p.Visible[layout.PaneRecentlyPlayed])
}

func TestPresetNerdStatus(t *testing.T) {
	p := layout.PresetNerdStatus
	assert.Equal(t, "Nerd Status", p.Name)
	assert.Len(t, p.Visible, 3, "should have 3 visible panes")
	assert.Len(t, p.Grid, 3, "should have 3 rows")

	assert.True(t, p.Visible[layout.PaneNowPlaying])
	assert.True(t, p.Visible[layout.PaneRequestFlow])
	assert.True(t, p.Visible[layout.PaneNetworkLog])

	// Row weights: 1, 3, 2
	assert.Equal(t, 1, p.Grid[0].HeightWeight)
	assert.Equal(t, 3, p.Grid[1].HeightWeight)
	assert.Equal(t, 2, p.Grid[2].HeightWeight)
}

func TestPagePresets_Counts(t *testing.T) {
	assert.Len(t, layout.PageAPresets, 4, "PageAPresets should have 4 entries")
	assert.Len(t, layout.PageBPresets, 1, "PageBPresets should have 1 entry")
}
