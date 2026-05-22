package layout

// Cell represents a pane slot in a row with its relative width.
type Cell struct {
	PaneID      PaneID
	WidthWeight int
}

// Row represents a horizontal strip of cells in the grid with its relative height.
type Row struct {
	HeightWeight int
	MinHeight    int // if > 0, this row is guaranteed at least MinHeight rows
	Cells        []Cell
}

// Preset is a named grid configuration — a bitmask of visible panes plus the grid layout.
type Preset struct {
	Name    string
	Visible map[PaneID]bool
	Grid    []Row
}

// Music page presets (DESIGN.md §4)

// PresetDashboard shows all 8 Music page panes across 3 rows.
var PresetDashboard = Preset{
	Name: "Full Dashboard",
	Visible: map[PaneID]bool{
		PaneNowPlaying: true, PaneQueue: true, PanePlaylists: true,
		PaneAlbums: true, PaneLikedSongs: true, PaneRecentlyPlayed: true,
		PaneTopTracks: true, PaneTopArtists: true,
	},
	Grid: []Row{
		{HeightWeight: 2, MinHeight: 14, Cells: []Cell{{PaneID: PaneNowPlaying, WidthWeight: 1}}},
		{HeightWeight: 3, Cells: []Cell{
			{PaneID: PanePlaylists, WidthWeight: 1},
			{PaneID: PaneAlbums, WidthWeight: 1},
			{PaneID: PaneLikedSongs, WidthWeight: 1},
		}},
		{HeightWeight: 3, Cells: []Cell{
			{PaneID: PaneQueue, WidthWeight: 1},
			{PaneID: PaneRecentlyPlayed, WidthWeight: 1},
			{PaneID: PaneTopTracks, WidthWeight: 1},
			{PaneID: PaneTopArtists, WidthWeight: 1},
		}},
	},
}

// PresetListening shows NowPlaying expanded with Queue and RecentlyPlayed below.
var PresetListening = Preset{
	Name: "Listening",
	Visible: map[PaneID]bool{
		PaneNowPlaying: true, PaneQueue: true, PaneRecentlyPlayed: true,
	},
	Grid: []Row{
		{HeightWeight: 3, MinHeight: 14, Cells: []Cell{{PaneID: PaneNowPlaying, WidthWeight: 1}}},
		{HeightWeight: 2, Cells: []Cell{
			{PaneID: PaneQueue, WidthWeight: 1},
			{PaneID: PaneRecentlyPlayed, WidthWeight: 1},
		}},
	},
}

// PresetLibrary shows a compact NowPlaying strip with the full library below.
var PresetLibrary = Preset{
	Name: "Library",
	Visible: map[PaneID]bool{
		PaneNowPlaying: true, PanePlaylists: true, PaneAlbums: true, PaneLikedSongs: true,
	},
	Grid: []Row{
		{HeightWeight: 1, MinHeight: 14, Cells: []Cell{{PaneID: PaneNowPlaying, WidthWeight: 1}}},
		{HeightWeight: 4, Cells: []Cell{
			{PaneID: PanePlaylists, WidthWeight: 1},
			{PaneID: PaneAlbums, WidthWeight: 1},
			{PaneID: PaneLikedSongs, WidthWeight: 1},
		}},
	},
}

// PresetDiscovery shows a compact NowPlaying strip with discovery panes below.
var PresetDiscovery = Preset{
	Name: "Discovery",
	Visible: map[PaneID]bool{
		PaneNowPlaying: true, PaneTopTracks: true, PaneTopArtists: true, PaneRecentlyPlayed: true,
	},
	Grid: []Row{
		{HeightWeight: 1, MinHeight: 14, Cells: []Cell{{PaneID: PaneNowPlaying, WidthWeight: 1}}},
		{HeightWeight: 2, Cells: []Cell{
			{PaneID: PaneTopTracks, WidthWeight: 1},
			{PaneID: PaneTopArtists, WidthWeight: 1},
		}},
		{HeightWeight: 2, Cells: []Cell{{PaneID: PaneRecentlyPlayed, WidthWeight: 1}}},
	},
}

// Stats page preset

// PresetStats shows NowPlaying strip, three diagnostic panes side-by-side
// (Health, Traffic, Live with weights 1:1:3 → ~20%/20%/60%), and NetworkLog
// full-width below. All five panes are individually toggleable via keys 1-5.
var PresetStats = Preset{
	Name: "Stats",
	Visible: map[PaneID]bool{
		PaneNowPlaying:     true,
		PaneGatewayHealth:  true,
		PanePollingTraffic: true,
		PaneGatewayLive:    true,
		PaneNetworkLog:     true,
	},
	Grid: []Row{
		{HeightWeight: 1, MinHeight: 14, Cells: []Cell{
			{PaneID: PaneNowPlaying, WidthWeight: 1},
		}},
		{HeightWeight: 3, Cells: []Cell{
			{PaneID: PaneGatewayHealth, WidthWeight: 1},
			{PaneID: PanePollingTraffic, WidthWeight: 1},
			{PaneID: PaneGatewayLive, WidthWeight: 3},
		}},
		{HeightWeight: 2, Cells: []Cell{
			{PaneID: PaneNetworkLog, WidthWeight: 1},
		}},
	},
}

// PageMusicPresets is the ordered list of presets for the Music page.
// Index 0 is the default (Full Dashboard).
var PageMusicPresets = []Preset{PresetDashboard, PresetListening, PresetLibrary, PresetDiscovery}

// PageStatsPresets is the ordered list of presets for the Stats page.
var PageStatsPresets = []Preset{PresetStats}
