// preference persistence — theme, preset, and visualizer changes are
// debounced and flushed to disk via a generation-counter pattern to avoid excessive
// writes during rapid key presses.
package app

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/prefs"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
)

// prefsFlushTickMsg is sent by the debounce timer started in schedulePrefsFlush.
// Only the tick whose generation matches the current prefsDirtyGen triggers a flush;
// stale ticks from superseded changes are ignored.
type prefsFlushTickMsg struct{ generation int }

// schedulePrefsFlush marks preferences dirty and starts a 500ms debounce timer.
// Returns the tea.Cmd for the timer. Call this after every prefs.Set().
// The generation counter ensures only the latest change triggers a disk write.
func (a *App) schedulePrefsFlush() tea.Cmd {
	a.prefsDirtyGen++
	gen := a.prefsDirtyGen
	return tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
		return prefsFlushTickMsg{generation: gen}
	})
}

// SchedulePrefsFlush is the exported test accessor for schedulePrefsFlush.
// It increments the generation counter and returns the debounce tick Cmd.
func (a *App) SchedulePrefsFlush() tea.Cmd {
	return a.schedulePrefsFlush()
}

// Prefs returns the underlying PreferenceStore for inspection in tests.
func (a *App) Prefs() *prefs.PreferenceStore {
	return a.prefs
}

// PrefsDirtyGen returns the current preference dirty generation counter.
// Used in tests to verify that schedulePrefsFlush increments it correctly.
func (a *App) PrefsDirtyGen() int {
	return a.prefsDirtyGen
}

// handlePrefsMsg routes preference-related messages. Called from handleMsg switch
// for prefsFlushTickMsg, prefs.FlushedMsg, and panes.VisualizerPatternChangedMsg.
//
// prefsFlushTickMsg: debounce timer fired — flush only if generation matches.
// prefs.FlushedMsg: log error to stderr on failure (non-critical, no toast).
// panes.VisualizerPatternChangedMsg: persist new visualizer index via PreferenceStore.
func (a *App) handlePrefsMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch m := msg.(type) {
	case prefsFlushTickMsg:
		// Only flush if no newer preference change has been made (debounce).
		if m.generation == a.prefsDirtyGen {
			return a, a.prefs.FlushCmd(), true
		}
		return a, nil, true

	case prefs.FlushedMsg:
		if m.Err != nil {
			// Non-critical — log to stderr; no user toast (a failed flush is invisible).
			// Re-queue retry: re-queued changes sit in pending map (done by FlushCmd on
			// error), and schedulePrefsFlush arms a new debounce timer to flush them.
			fmt.Fprintf(os.Stderr, "spotnik: prefs flush failed: %v\n", m.Err)
			return a, a.schedulePrefsFlush(), true
		}
		return a, nil, true

	case panes.VisualizerPatternChangedMsg:
		// User cycled the visualizer pattern — persist via PreferenceStore.
		a.prefs.Set("visualizer", m.PatternIndex)
		return a, a.schedulePrefsFlush(), true
	}
	return a, nil, false
}
