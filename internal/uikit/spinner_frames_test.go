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

// TestSpinnerFrames_MutationSafe verifies that mutating the returned slice does
// not affect a subsequent call. The package-level backing slices must not be
// handed out directly.
func TestSpinnerFrames_MutationSafe(t *testing.T) {
	for _, mode := range []uikit.GlyphMode{uikit.GlyphUnicode, uikit.GlyphASCII} {
		original := uikit.SpinnerFrames(mode)
		// Mutate the first element of the returned slice.
		original[0] = "MUTATED"
		// A second call must return the original, unmodified values.
		fresh := uikit.SpinnerFrames(mode)
		assert.NotEqual(t, "MUTATED", fresh[0],
			"mutating SpinnerFrames result must not affect the package-level backing slice (mode=%v)", mode)
	}
}

// TestSpinnerFrames_DefaultModeReturnsUnicode verifies that any unrecognised
// GlyphMode falls back to unicode frames rather than silently returning nil.
func TestSpinnerFrames_DefaultModeReturnsUnicode(t *testing.T) {
	want := uikit.SpinnerFrames(uikit.GlyphUnicode)
	got := uikit.SpinnerFrames(uikit.GlyphMode(999))
	assert.Equal(t, want, got, "unknown mode should fall back to unicode frames")
}
