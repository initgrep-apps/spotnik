package cliout

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// wrap applies the standard 2-char left indent to every line of a block.
var wrap = lipgloss.NewStyle().Padding(0, 2)

// Write renders each message with a leading blank line and standard padding.
// Safe for any io.Writer. If a Recorder is active (see testing.go), writes
// go to the recorder instead of the writer. On first call, pins ASCII profile
// when output is not a TTY or NO_COLOR is set.
func Write(w io.Writer, msgs ...Message) {
	if rec := activeRecorder(); rec != nil {
		rec.append(msgs...)
		return
	}
	if len(msgs) == 0 {
		return
	}
	if !isTTY(w) || checkNoColor() {
		pinASCII()
	}
	block := renderAll(current(), msgs)
	_, _ = fmt.Fprintln(w, "\n"+wrap.Render(block))
}

// WriteInline renders with no leading blank line — for compact step-by-step progress.
// If a Recorder is active, writes go to the recorder instead of the writer. On first
// call, pins ASCII profile when output is not a TTY or NO_COLOR is set.
func WriteInline(w io.Writer, msgs ...Message) {
	if rec := activeRecorder(); rec != nil {
		rec.append(msgs...)
		return
	}
	if len(msgs) == 0 {
		return
	}
	if !isTTY(w) || checkNoColor() {
		pinASCII()
	}
	block := renderAll(current(), msgs)
	_, _ = fmt.Fprintln(w, wrap.Render(block))
}

// renderAll renders all messages and joins non-empty outputs with newlines.
func renderAll(p Palette, msgs []Message) string {
	parts := make([]string, 0, len(msgs))
	for _, m := range msgs {
		s := m.render(p)
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n")
}
