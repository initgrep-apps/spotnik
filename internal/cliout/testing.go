package cliout

import (
	"bytes"
	"io"
	"sync"
)

// Recorder captures messages written via Write / WriteInline without rendering.
// Used by Capture and by test assertions on structure rather than styled strings.
type Recorder struct {
	mu   sync.Mutex
	msgs []Message
}

// append records messages into the Recorder.
func (r *Recorder) append(msgs ...Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, msgs...)
}

// Messages returns a snapshot copy of the recorded slice.
func (r *Recorder) Messages() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Message, len(r.msgs))
	copy(out, r.msgs)
	return out
}

var (
	recorderMu sync.RWMutex
	recorder   *Recorder
)

// activeRecorder returns the current package-level recorder or nil.
func activeRecorder() *Recorder {
	recorderMu.RLock()
	defer recorderMu.RUnlock()
	return recorder
}

// CurrentForTest returns the currently active palette. Test-only helper that
// allows cmd/ tests to assert on the palette installed by resolveCLIPalette.
func CurrentForTest() Palette { return current() }

// Capture runs fn with a package-level Recorder installed.
// All Write/WriteInline calls during fn are captured and returned.
// Spinner/Prompt dynamic types (Story 149) also append to the recorder
// when active, but their input/animation side effects are skipped —
// tests assert on structure, not on stdin consumption or TTY bytes.
// Not thread-safe — run tests sequentially.
func Capture(fn func(w io.Writer)) []Message {
	r := &Recorder{}
	recorderMu.Lock()
	prev := recorder
	recorder = r
	recorderMu.Unlock()

	defer func() {
		recorderMu.Lock()
		recorder = prev
		recorderMu.Unlock()
	}()

	fn(&bytes.Buffer{}) // writer is discarded; recorder captures everything
	return r.Messages()
}
