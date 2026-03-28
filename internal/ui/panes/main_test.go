package panes_test

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain forces TrueColor profile so ANSI escape codes are emitted in tests,
// regardless of whether the test runner has a TTY attached. Without this,
// lipgloss silently strips all colour sequences and tests that check for ANSI
// output will fail.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}
