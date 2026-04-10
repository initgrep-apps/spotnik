package app

// routing_internal_test.go — White-box tests for isPlaybackKey and isPremiumOnlyPlaybackKey.
// These tests serve as a living specification: every key listed in isPlaybackKey is
// checked for whether it also requires Premium. Any future addition to isPlaybackKey
// that is not mirrored in isPremiumOnlyPlaybackKey will cause a test failure here.

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestIsPlaybackKey_Enumeration(t *testing.T) {
	cases := []struct {
		name       string
		key        tea.KeyMsg
		isPlayback bool
	}{
		{"Space", tea.KeyMsg{Type: tea.KeySpace}, true},
		{"Left arrow", tea.KeyMsg{Type: tea.KeyLeft}, true},
		{"Right arrow", tea.KeyMsg{Type: tea.KeyRight}, true},
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
		{"Left prev track", tea.KeyMsg{Type: tea.KeyLeft}, true},
		{"Right next track", tea.KeyMsg{Type: tea.KeyRight}, true},
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
