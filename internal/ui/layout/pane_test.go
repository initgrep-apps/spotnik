package layout_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/stretchr/testify/assert"
)

func TestPaneID_IotaValues(t *testing.T) {
	assert.Equal(t, layout.PaneID(0), layout.PaneNowPlaying)
	assert.Equal(t, layout.PaneID(1), layout.PaneQueue)
	assert.Equal(t, layout.PaneID(2), layout.PanePlaylists)
	assert.Equal(t, layout.PaneID(3), layout.PaneAlbums)
	assert.Equal(t, layout.PaneID(4), layout.PaneLikedSongs)
	assert.Equal(t, layout.PaneID(5), layout.PaneRecentlyPlayed)
	assert.Equal(t, layout.PaneID(6), layout.PaneTopTracks)
	assert.Equal(t, layout.PaneID(7), layout.PaneTopArtists)
	assert.Equal(t, layout.PaneID(8), layout.PaneNetworkLog)
	assert.Equal(t, layout.PaneID(9), layout.PaneGatewayHealth)
	assert.Equal(t, layout.PaneID(10), layout.PanePollingTraffic)
	assert.Equal(t, layout.PaneID(11), layout.PaneGatewayLive)
}

// TestPaneIDs_PageBConstants_AreDistinct verifies that the four Page B PaneID constants
// are distinct from each other and from all Page A constants.
func TestPaneIDs_PageBConstants_AreDistinct(t *testing.T) {
	pageB := []layout.PaneID{
		layout.PaneNetworkLog,
		layout.PaneGatewayHealth,
		layout.PanePollingTraffic,
		layout.PaneGatewayLive,
	}
	seen := make(map[layout.PaneID]bool)
	for _, id := range pageB {
		assert.False(t, seen[id], "PaneID %d appears more than once in Page B constants", id)
		seen[id] = true
	}
	// None of the Page B constants should collide with Page A constants.
	pageA := []layout.PaneID{
		layout.PaneNowPlaying,
		layout.PaneQueue,
		layout.PanePlaylists,
		layout.PaneAlbums,
		layout.PaneLikedSongs,
		layout.PaneRecentlyPlayed,
		layout.PaneTopTracks,
		layout.PaneTopArtists,
	}
	for _, id := range pageA {
		assert.False(t, seen[id], "Page A PaneID %d collides with a Page B constant", id)
	}
}

func TestPageID_Constants(t *testing.T) {
	assert.Equal(t, layout.PageID(0), layout.PageA)
	assert.Equal(t, layout.PageID(1), layout.PageB)
}

func TestRect_ContentWidth(t *testing.T) {
	tests := []struct {
		name  string
		rect  layout.Rect
		wantW int
		wantH int
	}{
		{
			name:  "normal rect",
			rect:  layout.Rect{X: 0, Y: 0, Width: 20, Height: 10},
			wantW: 18,
			wantH: 8,
		},
		{
			name:  "width exactly 2",
			rect:  layout.Rect{Width: 2, Height: 2},
			wantW: 0,
			wantH: 0,
		},
		{
			name:  "width 1 (below minimum)",
			rect:  layout.Rect{Width: 1, Height: 1},
			wantW: 0,
			wantH: 0,
		},
		{
			name:  "width 0",
			rect:  layout.Rect{Width: 0, Height: 0},
			wantW: 0,
			wantH: 0,
		},
		{
			name:  "large rect",
			rect:  layout.Rect{Width: 120, Height: 30},
			wantW: 118,
			wantH: 28,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantW, tt.rect.ContentWidth(), "ContentWidth")
			assert.Equal(t, tt.wantH, tt.rect.ContentHeight(), "ContentHeight")
		})
	}
}
