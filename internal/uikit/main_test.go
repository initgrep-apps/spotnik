package uikit_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain forces TrueColor profile so ANSI escape codes are emitted during
// tests regardless of whether the test runner has a TTY attached. This is
// necessary for assertions that compare colour-distinguished renders (e.g.
// PanelIntentDefault vs PanelIntentError border colour).
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}

// stripANSI removes ANSI escape sequences so tests can assert on visible text only.
var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}
