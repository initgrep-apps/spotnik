package uikit_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpinnerFrames_Unicode verifies that SpinnerFrames(GlyphUnicode) returns the
// 10-frame braille set in the specified order.
func TestSpinnerFrames_Unicode(t *testing.T) {
	frames := uikit.SpinnerFrames(uikit.GlyphUnicode)
	want := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	require.Len(t, frames, 10, "unicode spinner must have 10 frames")
	assert.Equal(t, want, frames, "unicode frames must match the braille sequence")
}

// TestSpinnerFrames_ASCII verifies that SpinnerFrames(GlyphASCII) returns the
// 4-frame rotating-bar set.
func TestSpinnerFrames_ASCII(t *testing.T) {
	frames := uikit.SpinnerFrames(uikit.GlyphASCII)
	want := []string{"|", "/", "-", "\\"}
	require.Len(t, frames, 4, "ascii spinner must have 4 frames")
	assert.Equal(t, want, frames, "ascii frames must match the rotating-bar sequence")
}

// TestSpinnerFrames_ReturnsDifferentSlices verifies that unicode and ascii modes
// return different frame sets (regression guard against accidental same-slice return).
func TestSpinnerFrames_ReturnsDifferentSlices(t *testing.T) {
	uni := uikit.SpinnerFrames(uikit.GlyphUnicode)
	asc := uikit.SpinnerFrames(uikit.GlyphASCII)
	assert.NotEqual(t, uni, asc, "unicode and ascii frame sets must differ")
}

// TestSpinnerFrames_ASCIIContainsOnlyASCIIRunes verifies no non-ASCII rune
// appears in the ASCII frame set.
func TestSpinnerFrames_ASCIIContainsOnlyASCIIRunes(t *testing.T) {
	for _, f := range uikit.SpinnerFrames(uikit.GlyphASCII) {
		for _, r := range f {
			assert.Less(t, int(r), 128, "ascii frame %q must contain only ASCII runes", f)
		}
	}
}
