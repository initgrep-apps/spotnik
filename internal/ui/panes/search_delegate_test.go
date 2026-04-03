package panes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCategorySymbol verifies the badge character returned for each category.
func TestCategorySymbol(t *testing.T) {
	tests := []struct {
		category string
		want     string
	}{
		{category: "track", want: "♪"},
		{category: "artist", want: "★"},
		{category: "album", want: "◎"},
		{category: "playlist", want: "▤"},
		{category: "unknown", want: "·"},
		{category: "", want: "·"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := categorySymbol(tt.category)
			assert.Equal(t, tt.want, got, "categorySymbol(%q) should return %q", tt.category, tt.want)
		})
	}
}
