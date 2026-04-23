package cliout

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const spinnerInterval = 100 * time.Millisecond

// SpinnerHandle controls a running spinner. Obtain one via StartSpinner.
// Done, Fail, and Stop are safe to call from any goroutine and are idempotent.
type SpinnerHandle struct {
	w        io.Writer
	text     string
	cancel   context.CancelFunc
	done     chan struct{} // closed when goroutine exits
	onTTY    bool
	resolved bool
	mu       sync.Mutex
}

// StartSpinner starts an animated spinner on TTY or writes a static pending
// line on non-TTY / test mode. Returns a SpinnerHandle to resolve via
// Done, Fail, or Stop.
//
// If a Recorder is active (test capture via Capture()), the Spinner message is
// recorded and a no-op handle is returned.
func StartSpinner(w io.Writer, text string) *SpinnerHandle {
	if rec := activeRecorder(); rec != nil {
		rec.append(Spinner{Text: text})
		return &SpinnerHandle{w: w, text: text}
	}

	onTTY := isTTY(w) && !checkNoColor() && !inTestMode()
	h := &SpinnerHandle{w: w, text: text, onTTY: onTTY, done: make(chan struct{})}

	if !onTTY {
		// Non-TTY / test mode: write a single static pending line.
		Write(w, Step{Status: Pending, Text: text})
		close(h.done)
		return h
	}

	installSIGINTHandler()
	registerHandle(h)

	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel

	// Hide cursor.
	_, _ = fmt.Fprint(w, "\x1b[?25l")

	go h.run(ctx)
	return h
}

// run redraws the spinner on every tick until ctx is cancelled.
func (h *SpinnerHandle) run(ctx context.Context) {
	defer close(h.done)

	p := current()
	frameStyle := lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(p.Muted)
	const padding = "  " // 2-char indent, matches Write/WriteInline

	ticker := time.NewTicker(spinnerInterval)
	defer ticker.Stop()

	i := 0
	render := func() {
		line := padding + frameStyle.Render(spinnerFrames[i%len(spinnerFrames)]) + " " + textStyle.Render(h.text)
		_, _ = fmt.Fprint(h.w, "\r\x1b[K"+line)
		i++
	}
	render() // draw first frame immediately

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			render()
		}
	}
}

// Done resolves the spinner with a success step.
func (h *SpinnerHandle) Done(text string) { h.resolve(StatusSuccess, text) }

// Fail resolves the spinner with a failure step.
func (h *SpinnerHandle) Fail(text string) { h.resolve(StatusFailure, text) }

// Stop cancels the spinner silently. Idempotent.
func (h *SpinnerHandle) Stop() { h.resolve(-1, "") }

// resolve is the internal idempotent shutdown. s == -1 means silent cancel.
func (h *SpinnerHandle) resolve(s Status, text string) {
	h.mu.Lock()
	if h.resolved {
		h.mu.Unlock()
		return
	}
	h.resolved = true
	h.mu.Unlock()

	if h.cancel != nil {
		h.cancel()
		<-h.done // wait for goroutine to exit before printing the resolution line
	}

	if h.onTTY {
		// Clear the spinner line and restore cursor.
		_, _ = fmt.Fprint(h.w, "\r\x1b[K\x1b[?25h")
	}

	unregisterHandle(h)

	if s == -1 {
		// Silent cancel — no resolution line.
		if h.onTTY {
			_, _ = fmt.Fprintln(h.w) // leave cursor on a fresh line
		}
		return
	}

	// Print resolution as a standard Step.
	WriteInline(h.w, Step{Status: s, Text: text})
}

// Package-level registry of active handles for SIGINT cleanup.
var (
	handlesMu sync.Mutex
	handles   = map[*SpinnerHandle]struct{}{}
	sigOnce   sync.Once
)

func registerHandle(h *SpinnerHandle) {
	handlesMu.Lock()
	defer handlesMu.Unlock()
	handles[h] = struct{}{}
}

func unregisterHandle(h *SpinnerHandle) {
	handlesMu.Lock()
	defer handlesMu.Unlock()
	delete(handles, h)
}

// installSIGINTHandler registers a one-time SIGINT/SIGTERM handler that
// stops all active spinners, restores the cursor, and exits 130.
// Skipped entirely in test mode.
func installSIGINTHandler() {
	sigOnce.Do(func() {
		if inTestMode() {
			return
		}
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-ch
			handlesMu.Lock()
			for h := range handles {
				if h.cancel != nil {
					h.cancel()
				}
				if h.onTTY {
					_, _ = fmt.Fprint(h.w, "\r\x1b[K\x1b[?25h")
				}
			}
			handlesMu.Unlock()
			os.Exit(130) // standard SIGINT exit code
		}()
	})
}
