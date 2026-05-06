package app

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// clipboardCopiedMsg is emitted after copyToClipboardCmd attempts a write.
// Err is non-nil only when emitting the OSC 52 sequence to stderr failed
// (broken stderr pipe). A nil Err does NOT guarantee the terminal actually
// wrote to the system clipboard — OSC 52 has no acknowledgement protocol.
type clipboardCopiedMsg struct {
	Err error
}

// copyToClipboardCmd returns a tea.Cmd that copies text to the user's
// terminal clipboard via OSC 52 (ESC ] 52 ; c ; <base64> BEL).
//
// The escape sequence is consumed by the terminal emulator — not the
// host OS — so it targets the clipboard on the screen the user is
// looking at, regardless of whether spotnik is running locally, in
// Docker, over SSH, or inside tmux passthrough.
//
// Writes to os.Stderr because os.Stdout is owned by the Bubble Tea
// renderer; stderr is connected to the same TTY in interactive
// sessions and terminals process escape sequences from either stream.
//
// Terminals without OSC 52 support silently ignore the sequence; we
// cannot detect that. Callers should keep the source URL/text visible
// on screen as a manual-copy fallback.
func copyToClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		_, err := fmt.Fprint(os.Stderr, ansi.SetSystemClipboard(text))
		if err != nil {
			return clipboardCopiedMsg{Err: fmt.Errorf("emitting OSC 52: %w", err)}
		}
		return clipboardCopiedMsg{}
	}
}
