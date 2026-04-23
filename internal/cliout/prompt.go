package cliout

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ErrAborted is returned from Ask when the user aborts (EOF, Ctrl+C, or
// exhausts all retry attempts).
var ErrAborted = errors.New("prompt aborted")

const maxPromptAttempts = 3

// Ask renders a Prompt and reads a validated value from r (typically os.Stdin).
// On validation failure it retries up to maxPromptAttempts times, printing a
// failure step after each bad attempt. After maxPromptAttempts failures it
// prints a "Giving up" step and returns ErrAborted.
//
// If a Recorder is active (Capture mode), the Prompt message is recorded and
// Ask returns immediately with ("", nil) — no input is consumed.
func Ask(r io.Reader, w io.Writer, p Prompt) (string, error) {
	if rec := activeRecorder(); rec != nil {
		rec.append(p)
		return "", nil
	}

	scanner := bufio.NewScanner(r)
	palette := current()
	labelStyle := lipgloss.NewStyle().Foreground(palette.Muted)
	placeholderStyle := lipgloss.NewStyle().Foreground(palette.Muted)

	for attempt := 0; attempt < maxPromptAttempts; attempt++ {
		// Render "  <Label>: " with optional placeholder on first attempt.
		line := "  " + labelStyle.Render(p.Label+":") + " "
		_, _ = fmt.Fprint(w, line)
		if p.Placeholder != "" && attempt == 0 {
			_, _ = fmt.Fprint(w, placeholderStyle.Render("("+p.Placeholder+") "))
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", err
			}
			// EOF reached.
			return "", ErrAborted
		}
		value := strings.TrimSpace(scanner.Text())

		if p.Validate == nil {
			return value, nil
		}
		if err := p.Validate(value); err == nil {
			return value, nil
		} else {
			WriteInline(w, Step{Status: StatusFailure, Text: err.Error()})
		}
	}

	Write(w, Step{Status: StatusFailure, Text: fmt.Sprintf("Giving up after %d attempts", maxPromptAttempts)})
	return "", ErrAborted
}
