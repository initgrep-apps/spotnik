package cliout

import (
	"io"
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

var (
	profileOnce sync.Once
	testMode    bool
	testModeMu  sync.RWMutex
)

// isTTY returns whether w is an *os.File pointing at a terminal.
// Returns false for any non-file writer (pipes, buffers, io.Discard).
func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// checkNoColor honours the NO_COLOR env var (any non-empty value disables colour).
func checkNoColor() bool {
	return os.Getenv("NO_COLOR") != ""
}

// pinASCII forces lipgloss to render without ANSI escapes. Called once when
// the first Write/StartSpinner/Ask resolves to a non-TTY or NO_COLOR path,
// or when SetTestMode(true) is called. Uses sync.Once so it is permanent once set.
func pinASCII() {
	profileOnce.Do(func() {
		lipgloss.SetColorProfile(termenv.Ascii)
	})
}

// SetTestMode enables or disables test mode. In test mode, pinASCII is called
// immediately and spinner animation is disabled (Story 149 uses this flag).
// Tests call this in TestMain for deterministic output.
func SetTestMode(enabled bool) {
	testModeMu.Lock()
	defer testModeMu.Unlock()
	testMode = enabled
	if enabled {
		pinASCII()
	}
}

// inTestMode reports whether test mode is active.
func inTestMode() bool {
	testModeMu.RLock()
	defer testModeMu.RUnlock()
	return testMode
}
