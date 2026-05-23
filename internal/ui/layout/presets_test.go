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
		layout.PresetStats,
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

// TestPresetStats_HasFivePanes verifies the new 5-pane Stats page preset.
func TestPresetStats_HasFivePanes(t *testing.T) {
	p := layout.PresetStats
	assert.Equal(t, "Stats", p.Name)
	assert.Len(t, p.Visible, 5, "should have 5 visible panes")

	assert.True(t, p.Visible[layout.PaneNowPlaying])
	assert.True(t, p.Visible[layout.PaneGatewayHealth])
	assert.True(t, p.Visible[layout.PanePollingTraffic])
	assert.True(t, p.Visible[layout.PaneGatewayLive])
	assert.True(t, p.Visible[layout.PaneNetworkLog])
}

// TestPresetStats_NowPlayingMinHeight verifies the NowPlaying row carries
// MinHeight: 10 so it always gets enough rows for image rendering.
func TestPresetStats_NowPlayingMinHeight(t *testing.T) {
	p := layout.PresetStats
	require.Len(t, p.Grid, 3)
	assert.Equal(t, 9, p.Grid[0].MinHeight, "NowPlaying row must have MinHeight 9")
}

// TestPresetStats_FlatThreeRows verifies the flat grid structure (story 181):
// NowPlaying strip, single middle row with Health+Traffic+Live (1:1:3),
// NetworkLog full-width row.
func TestPresetStats_FlatThreeRows(t *testing.T) {
	require.Len(t, layout.PresetStats.Grid, 3)

	p := layout.PresetStats

	// Row 0: NowPlaying strip
	assert.Equal(t, 2, p.Grid[0].HeightWeight)
	assert.Len(t, p.Grid[0].Cells, 1)
	assert.Equal(t, layout.PaneNowPlaying, p.Grid[0].Cells[0].PaneID)

	// Row 1: Health + Traffic + Live with 1:1:3 widths
	middle := p.Grid[1]
	require.Len(t, middle.Cells, 3)
	assert.Equal(t, layout.PaneGatewayHealth, middle.Cells[0].PaneID)
	assert.Equal(t, layout.PanePollingTraffic, middle.Cells[1].PaneID)
	assert.Equal(t, layout.PaneGatewayLive, middle.Cells[2].PaneID)
	assert.Equal(t, 1, middle.Cells[0].WidthWeight)
	assert.Equal(t, 1, middle.Cells[1].WidthWeight)
	assert.Equal(t, 3, middle.Cells[2].WidthWeight, "Live keeps weight 3 (~60%)")

	// Row 2: NetworkLog full-width
	assert.Equal(t, 2, p.Grid[2].HeightWeight)
	assert.Len(t, p.Grid[2].Cells, 1)
	assert.Equal(t, layout.PaneNetworkLog, p.Grid[2].Cells[0].PaneID)
}

func TestPagePresets_Counts(t *testing.T) {
	assert.Len(t, layout.PageMusicPresets, 4, "PageMusicPresets should have 4 entries")
	assert.Len(t, layout.PageStatsPresets, 1, "PageStatsPresets should have 1 entry")
}

// TestPreset_MusicPage_NowPlayingMinHeight verifies that Music page presets
// reserve the correct MinHeight for the NowPlaying pane.
func TestPreset_MusicPage_NowPlayingMinHeight(t *testing.T) {
	presets := []struct {
		name      string
		preset    layout.Preset
		minHeight int
	}{
		{"Dashboard", layout.PresetDashboard, 9},
		{"Listening", layout.PresetListening, 9},
		{"Library", layout.PresetLibrary, 9},
		{"Discovery", layout.PresetDiscovery, 9},
	}

	for _, tt := range presets {
		t.Run(tt.name, func(t *testing.T) {
			require.GreaterOrEqual(t, len(tt.preset.Grid), 1, "preset must have at least one row")
			nowPlayingRow := tt.preset.Grid[0]
			assert.Equal(t, layout.PaneNowPlaying, nowPlayingRow.Cells[0].PaneID,
				"first row must contain NowPlaying")
			assert.Equal(t, tt.minHeight, nowPlayingRow.MinHeight,
				"NowPlaying row must have correct MinHeight")
		})
	}
}
