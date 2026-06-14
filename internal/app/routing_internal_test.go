package app

// routing_internal_test.go — White-box tests for toggle key routing helpers.
// Covers isPlaybackKey, isPremiumOnlyPlaybackKey, currentToggleKeyMap,
// and podcastPresetNames.

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPlaybackKey_Enumeration(t *testing.T) {
	cases := []struct {
		name       string
		key        tea.KeyMsg
		isPlayback bool
	}{
		{"Space", tea.KeyMsg{Type: tea.KeySpace}, true},
		{"Left arrow (seek)", tea.KeyMsg{Type: tea.KeyLeft}, true},
		{"Right arrow (seek)", tea.KeyMsg{Type: tea.KeyRight}, true},
		{"Shift+Left (prev track)", tea.KeyMsg{Type: tea.KeyShiftLeft}, true},
		{"Shift+Right (next track)", tea.KeyMsg{Type: tea.KeyShiftRight}, true},
		{"+ volume up", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}, true},
		{"- volume down", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}, true},
		{"s shuffle", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}, true},
		{"r repeat", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}, true},
		{"v visualizer", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}, true},
		// Non-playback keys
		{"n (removed)", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}, false},
		{"q quit", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, false},
		{"Tab", tea.KeyMsg{Type: tea.KeyTab}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.isPlayback, isPlaybackKey(tc.key),
				"isPlaybackKey(%q)", tc.name)
		})
	}
}

func TestIsPremiumOnlyPlaybackKey_Enumeration(t *testing.T) {
	cases := []struct {
		name        string
		key         tea.KeyMsg
		needPremium bool
	}{
		// Premium-required keys
		{"Space play/pause", tea.KeyMsg{Type: tea.KeySpace}, true},
		{"Left seek back", tea.KeyMsg{Type: tea.KeyLeft}, true},
		{"Right seek forward", tea.KeyMsg{Type: tea.KeyRight}, true},
		{"Shift+Left prev track", tea.KeyMsg{Type: tea.KeyShiftLeft}, true},
		{"Shift+Right next track", tea.KeyMsg{Type: tea.KeyShiftRight}, true},
		{"+ volume up", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}, true},
		{"- volume down", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}, true},
		{"s shuffle", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}, true},
		{"r repeat", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}, true},
		// v (visualizer) is local UI — no API call, no Premium gate
		{"v visualizer (local)", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}, false},
		// Non-playback keys are never premium-gated
		{"n (removed)", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}, false},
		{"q quit", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.needPremium, isPremiumOnlyPlaybackKey(tc.key),
				"isPremiumOnlyPlaybackKey(%q)", tc.name)
		})
	}
}

// ── currentToggleKeyMap tests ──────────────────────────────────────────────────

func TestCurrentToggleKeyMap_PodcastPreset(t *testing.T) {
	a := New(&config.Config{}, AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a.layout.SetPreset(2)
	require.Equal(t, layout.PresetNamePodcast, a.layout.ActivePresetName())

	km := a.currentToggleKeyMap()

	assert.Equal(t, layout.PaneNowPlaying, km['1'])
	assert.Equal(t, layout.PaneQueue, km['2'])
	assert.Equal(t, layout.PaneFollowedShows, km['3'])
	assert.Equal(t, layout.PaneSavedEpisodes, km['4'])

	for _, k := range []rune{'5', '6', '7', '8'} {
		_, exists := km[k]
		assert.False(t, exists, "key '%c' should not exist in podcast key map", k)
	}
}

func TestCurrentToggleKeyMap_MusicPreset(t *testing.T) {
	a := New(&config.Config{}, AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Equal(t, layout.PresetNameDashboard, a.layout.ActivePresetName())

	km := a.currentToggleKeyMap()

	assert.Equal(t, layout.PaneNowPlaying, km['1'])
	assert.Equal(t, layout.PaneQueue, km['2'])
	assert.Equal(t, layout.PanePlaylists, km['3'])
	assert.Equal(t, layout.PaneAlbums, km['4'])
	assert.Equal(t, layout.PaneLikedSongs, km['5'])
	assert.Equal(t, layout.PaneRecentlyPlayed, km['6'])
	assert.Equal(t, layout.PaneTopTracks, km['7'])
	assert.Equal(t, layout.PaneTopArtists, km['8'])
}

func TestCurrentToggleKeyMap_StatsPreset(t *testing.T) {
	a := New(&config.Config{}, AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a.layout.TogglePage()
	require.Equal(t, layout.PageStats, a.layout.ActivePage())

	km := a.currentToggleKeyMap()

	assert.Equal(t, layout.PaneNowPlaying, km['1'])
	assert.Equal(t, layout.PaneGatewayHealth, km['2'])
	assert.Equal(t, layout.PanePollingTraffic, km['3'])
	assert.Equal(t, layout.PaneGatewayLive, km['4'])
	assert.Equal(t, layout.PaneNetworkLog, km['5'])

	for _, k := range []rune{'6', '7', '8'} {
		_, exists := km[k]
		assert.False(t, exists, "key '%c' should not exist in stats key map", k)
	}
}

func TestCurrentToggleKeyMap_PodcastDashboardPreset(t *testing.T) {
	a := New(&config.Config{}, AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a.layout.SetPreset(5)
	require.Equal(t, layout.PresetNamePodcastDashboard, a.layout.ActivePresetName())

	km := a.currentToggleKeyMap()

	assert.Equal(t, layout.PaneNowPlaying, km['1'])
	assert.Equal(t, layout.PaneQueue, km['2'])
	assert.Equal(t, layout.PaneFollowedShows, km['3'])
	assert.Equal(t, layout.PaneSavedEpisodes, km['4'])
}

func TestPodcastPresetNames_UsesConstants(t *testing.T) {
	assert.True(t, podcastPresetNames[layout.PresetNamePodcast])
	assert.True(t, podcastPresetNames[layout.PresetNamePodcastDashboard])
	assert.False(t, podcastPresetNames[layout.PresetNameDashboard])
}
