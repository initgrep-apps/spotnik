package uikit

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// RepeatMode is the typed enum for the transport strip's repeat state.
type RepeatMode int

const (
	// RepeatOff renders the ⟳ (ro) glyph in inactive colour.
	RepeatOff RepeatMode = iota
	// RepeatAll renders the ↻ (rp) glyph in active colour.
	RepeatAll
	// RepeatOne renders the ↻¹ (rp1) glyph in active colour.
	RepeatOne
)

// PlaybackControls renders the three-position transport strip:
// shuffle  play/pause  repeat
//
// All glyphs are resolved through GlyphFor so the active GlyphMode
// (unicode/ascii) is respected automatically. Active positions use
// Theme.PlayingIndicator(); inactive positions use Theme.TextSecondary().
type PlaybackControls struct {
	// Playing is true when a track is currently playing (shows pause icon),
	// false when paused (shows play icon).
	Playing bool
	// Shuffle is true when shuffle is enabled.
	Shuffle bool
	// RepeatMode controls which repeat glyph and colour are used.
	RepeatMode RepeatMode
	// Theme provides colour tokens.
	Theme theme.Theme
}

// Render returns the transport strip as an ANSI-styled string.
// The strip is formatted as: <shuffle>  <play/pause>  <repeat>
// with two-space gaps between icons.
func (c PlaybackControls) Render() string {
	m := ActiveMode()
	activeStyle := lipgloss.NewStyle().Foreground(c.Theme.PlayingIndicator())
	inactiveStyle := lipgloss.NewStyle().Foreground(c.Theme.TextSecondary())

	pickStyle := func(active bool) lipgloss.Style {
		if active {
			return activeStyle
		}
		return inactiveStyle
	}

	shuffle := pickStyle(c.Shuffle).Render(GlyphFor(GlyphShuffle, m))

	var playPause string
	if c.Playing {
		playPause = activeStyle.Render(GlyphFor(GlyphPaused, m))
	} else {
		playPause = inactiveStyle.Render(GlyphFor(GlyphPausedPB, m))
	}

	var repeat string
	switch c.RepeatMode {
	case RepeatOne:
		repeat = activeStyle.Render(GlyphFor(GlyphRepeatOne, m))
	case RepeatAll:
		repeat = activeStyle.Render(GlyphFor(GlyphRepeatAll, m))
	default:
		repeat = inactiveStyle.Render(GlyphFor(GlyphRepeatOff, m))
	}

	return shuffle + "  " + playPause + "  " + repeat
}
