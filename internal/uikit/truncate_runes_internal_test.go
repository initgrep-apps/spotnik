package uikit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTruncateRunes_GuardMaxLessThanEllipsisLen exercises the defensive guard
// added in story 193: when max is smaller than the ellipsis rune length,
// truncateRunes returns the original string unmodified rather than slicing
// out of bounds.
//
// truncateRunes is unexported; this is the only call site that exercises the
// max < len(ell) branch directly. Normalize (the only public caller) uses
// max=48 and max=160, both well above the 1- or 3-rune ellipsis lengths, so
// the branch is unreachable from production code today. The guard stays as
// defensive insurance against a future caller passing a smaller cap.
func TestTruncateRunes_GuardMaxLessThanEllipsisLen(t *testing.T) {
	cases := []struct {
		name string
		mode GlyphMode
		s    string
		max  int
	}{
		// ASCII ellipsis is "..." (3 runes). max=2 is below the guard.
		{"ascii max=0", GlyphASCII, "hello", 0},
		{"ascii max=1", GlyphASCII, "hello", 1},
		{"ascii max=2", GlyphASCII, "hello", 2},
		// Unicode ellipsis is "…" (1 rune). max=0 is below the guard.
		{"unicode max=0", GlyphUnicode, "hello", 0},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			SetModeForTest(tt.mode)
			defer SetModeForTest(GlyphUnicode)

			// Must not panic and must return the original string unchanged.
			assert.NotPanics(t, func() {
				got := truncateRunes(tt.s, tt.max)
				assert.Equal(t, tt.s, got, "guard branch must return s unmodified")
			})
		})
	}
}
