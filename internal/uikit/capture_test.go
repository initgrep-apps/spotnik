package uikit_test

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

func TestCapture_StripsANSI_ReturnsPlainLines(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).
		Render("hello") + "\n" + "world"
	got := uikit.Capture(styled)
	assert.Equal(t, []string{"hello", "world"}, got)
}

func TestCapture_PreservesLeadingSpaces(t *testing.T) {
	got := uikit.Capture("  indented\n    more")
	assert.Equal(t, []string{"  indented", "    more"}, got)
}

func TestCapture_EmptyString_ReturnsEmptySlice(t *testing.T) {
	got := uikit.Capture("")
	assert.Empty(t, got)
}
