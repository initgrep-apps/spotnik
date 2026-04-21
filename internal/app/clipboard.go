package app

import (
	"fmt"
	"os/exec"
	"strings"
)

// copyToClipboard attempts to copy text to the system clipboard.
// Tries pbcopy (macOS), xclip -selection clipboard (Linux X11), wl-copy (Wayland).
// Returns an error if all methods fail.
// Callers treat failure silently — the URL remains visible for manual selection.
// Key handler (Story 138) calls this on the 'c' key during viewAuth and stepOAuth.
func copyToClipboard(text string) error {
	commands := [][]string{
		{"pbcopy"},
		{"xclip", "-selection", "clipboard"},
		{"wl-copy"},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no clipboard command available (tried pbcopy, xclip, wl-copy)")
}
