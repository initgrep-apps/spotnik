package layout

// Cell represents a pane slot in a row with its relative width.
type Cell struct {
	PaneID      PaneID
	WidthWeight int
}

// Row represents a horizontal strip of cells in the grid with its relative height.
type Row struct {
	HeightWeight int
	Cells        []Cell
}

// Preset is a named grid configuration — a bitmask of visible panes plus the grid layout.
type Preset struct {
	Name    string
	Visible map[PaneID]bool
	Grid    []Row
}

// Page A presets (DESIGN.md §4)

// PresetDashboard shows all 8 Page A panes across 3 rows.
var PresetDashboard = Preset{
	Name: "Full Dashboard",
	Visible: map[PaneID]bool{
		PaneNowPlaying: true, PaneQueue: true, PanePlaylists: true,
		PaneAlbums: true, PaneLikedSongs: true, PaneRecentlyPlayed: true,
		PaneTopTracks: true, PaneTopArtists: true,
	},
	Grid: []Row{
		{HeightWeight: 2, Cells: []Cell{{PaneNowPlaying, 1}}},
		{HeightWeight: 3, Cells: []Cell{{PanePlaylists, 1}, {PaneAlbums, 1}, {PaneLikedSongs, 1}}},
		{HeightWeight: 3, Cells: []Cell{{PaneQueue, 1}, {PaneRecentlyPlayed, 1}, {PaneTopTracks, 1}, {PaneTopArtists, 1}}},
	},
}

// PresetListening shows NowPlaying expanded with Queue and RecentlyPlayed below.
var PresetListening = Preset{
	Name: "Listening",
	Visible: map[PaneID]bool{
		PaneNowPlaying: true, PaneQueue: true, PaneRecentlyPlayed: true,
	},
	Grid: []Row{
		{HeightWeight: 3, Cells: []Cell{{PaneNowPlaying, 1}}},
		{HeightWeight: 2, Cells: []Cell{{PaneQueue, 1}, {PaneRecentlyPlayed, 1}}},
	},
}

// PresetLibrary shows a compact NowPlaying strip with the full library below.
var PresetLibrary = Preset{
	Name: "Library",
	Visible: map[PaneID]bool{
		PaneNowPlaying: true, PanePlaylists: true, PaneAlbums: true, PaneLikedSongs: true,
	},
	Grid: []Row{
		{HeightWeight: 1, Cells: []Cell{{PaneNowPlaying, 1}}},
		{HeightWeight: 4, Cells: []Cell{{PanePlaylists, 1}, {PaneAlbums, 1}, {PaneLikedSongs, 1}}},
	},
}

// PresetDiscovery shows a compact NowPlaying strip with discovery panes below.
var PresetDiscovery = Preset{
	Name: "Discovery",
	Visible: map[PaneID]bool{
		PaneNowPlaying: true, PaneTopTracks: true, PaneTopArtists: true, PaneRecentlyPlayed: true,
	},
	Grid: []Row{
		{HeightWeight: 1, Cells: []Cell{{PaneNowPlaying, 1}}},
		{HeightWeight: 2, Cells: []Cell{{PaneTopTracks, 1}, {PaneTopArtists, 1}}},
		{HeightWeight: 2, Cells: []Cell{{PaneRecentlyPlayed, 1}}},
	},
}

// Page B preset

// PresetNerdStatus shows NowPlaying compact strip + Request Flow + Network Log.
var PresetNerdStatus = Preset{
	Name: "Nerd Status",
	Visible: map[PaneID]bool{
		PaneNowPlaying: true, PaneRequestFlow: true, PaneNetworkLog: true,
	},
	Grid: []Row{
		{HeightWeight: 1, Cells: []Cell{{PaneNowPlaying, 1}}},
		{HeightWeight: 3, Cells: []Cell{{PaneRequestFlow, 1}}},
		{HeightWeight: 2, Cells: []Cell{{PaneNetworkLog, 1}}},
	},
}

// PageAPresets is the ordered list of presets for Page A (Music).
// Index 0 is the default (Full Dashboard).
var PageAPresets = []Preset{PresetDashboard, PresetListening, PresetLibrary, PresetDiscovery}

// PageBPresets is the ordered list of presets for Page B (Nerd Status).
var PageBPresets = []Preset{PresetNerdStatus}
