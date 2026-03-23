package components

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestRenderError_ContainsMessage(t *testing.T) {
	th := &theme.BlackTheme{}
	got := RenderError(th, 60, 10, "Failed to load devices", "Press d to retry")

	assert.Contains(t, got, "Failed to load devices")
	assert.Contains(t, got, "Press d to retry")
	assert.Contains(t, got, "✗")
}

func TestRenderError_HasRoundedBorder(t *testing.T) {
	th := &theme.BlackTheme{}
	got := RenderError(th, 60, 10, "Error", "Retry")

	// Rounded border corners use ╭ and ╰ per DESIGN.md
	assert.True(t, strings.Contains(got, "╭"), "expected rounded top-left corner")
	assert.True(t, strings.Contains(got, "╰"), "expected rounded bottom-left corner")
}

func TestRenderError_AllThemes(t *testing.T) {
	themes := []struct {
		name string
		th   theme.Theme
	}{
		{"black", &theme.BlackTheme{}},
		{"monokai", &theme.MonokaiTheme{}},
		{"catppuccin", &theme.CatppuccinTheme{}},
		{"nord", &theme.NordTheme{}},
		{"light", &theme.LightTheme{}},
	}

	for _, tt := range themes {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderError(tt.th, 60, 10, "Test error", "Press r to retry")
			assert.Contains(t, got, "Test error")
			assert.Contains(t, got, "Press r to retry")
		})
	}
}

func TestRenderError_EmptyHint(t *testing.T) {
	th := &theme.BlackTheme{}
	got := RenderError(th, 60, 10, "Something failed", "")

	assert.Contains(t, got, "Something failed")
	// Should still render without panicking
	assert.NotEmpty(t, got)
}

func TestRenderError_SmallDimensions(t *testing.T) {
	th := &theme.BlackTheme{}
	// Should not panic even with very small dimensions
	got := RenderError(th, 20, 5, "Error", "Retry")
	assert.NotEmpty(t, got)
}
