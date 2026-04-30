package cliout

import (
	"io"
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"

	"github.com/initgrep-apps/spotnik/internal/uikit"
)

var (
	profileOnce sync.Once
	testMode    bool
	testModeMu  sync.RWMutex
	priorUiMode uikit.GlyphMode
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

// IsTTY returns whether w is an *os.File pointing at a terminal.
// Exported shim over isTTY for use by cmd/.
func IsTTY(w io.Writer) bool { return isTTY(w) }

// checkNoColor honours the NO_COLOR env var (any non-empty value disables colour).
func checkNoColor() bool {
	return os.Getenv("NO_COLOR") != ""
}

// pinASCII forces lipgloss to render without ANSI colour escapes. It controls
// colour profile only — glyph mode is sourced independently from
// uikit.ActiveMode() and is not affected by this call.
// Called once when the first Write/StartSpinner/Ask resolves to a non-TTY or
// NO_COLOR path, or when SetTestMode(true) is called. Uses sync.Once so it is
// permanent once set.
func pinASCII() {
	profileOnce.Do(func() {
		lipgloss.SetColorProfile(termenv.Ascii)
	})
}

// SetTestMode enables or disables test mode. In test mode, pinASCII is called
// immediately, uikit is pinned to GlyphASCII mode so that glyph-role assertions
// are deterministic, and spinner animation is disabled.
// Tests call this in TestMain for deterministic output.
//
// SetTestMode(true) snapshots the current uikit mode before pinning to GlyphASCII.
// SetTestMode(false) restores that snapshot, so nested test helpers that toggle
// test mode do not permanently alter the active glyph mode.
// Note: pinASCII is sync.Once-guarded and is not reversed by SetTestMode(false).
func SetTestMode(enabled bool) {
	testModeMu.Lock()
	defer testModeMu.Unlock()
	testMode = enabled
	if enabled {
		priorUiMode = uikit.ActiveMode()
		pinASCII()
		uikit.SetModeForTest(uikit.GlyphASCII)
	} else {
		uikit.SetModeForTest(priorUiMode)
	}
}

// inTestMode reports whether test mode is active.
func inTestMode() bool {
	testModeMu.RLock()
	defer testModeMu.RUnlock()
	return testMode
}
