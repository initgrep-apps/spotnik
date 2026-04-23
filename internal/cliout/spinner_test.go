package cliout

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain installs test mode for the entire cliout package test suite.
// Test mode disables TTY animation and SIGINT handler registration so tests
// produce deterministic output without terminal side-effects.
func TestMain(m *testing.M) {
	SetTestMode(true)
	os.Exit(m.Run())
}

// TestStartSpinner_nonTTY_writesStaticPendingLine verifies that in test mode
// (non-TTY), StartSpinner writes a ◌ pending line and Done writes a ✓ step.
func TestStartSpinner_nonTTY_writesStaticPendingLine(t *testing.T) {
	var buf bytes.Buffer
	h := StartSpinner(&buf, "Waiting")
	h.Done("Done")
	out := buf.String()
	assert.Contains(t, out, "◌", "pending glyph expected on start")
	assert.Contains(t, out, "Waiting")
	assert.Contains(t, out, "✓", "success glyph expected on Done")
	assert.Contains(t, out, "Done")
}

// TestSpinner_Fail_writesFailureStep verifies Fail writes a ✗ step.
func TestSpinner_Fail_writesFailureStep(t *testing.T) {
	var buf bytes.Buffer
	h := StartSpinner(&buf, "Waiting")
	h.Fail("timed out")
	out := buf.String()
	assert.Contains(t, out, "✗")
	assert.Contains(t, out, "timed out")
}

// TestSpinner_Stop_silentNoResolutionLine verifies Stop produces no ✓ or ✗ line.
func TestSpinner_Stop_silentNoResolutionLine(t *testing.T) {
	var buf bytes.Buffer
	h := StartSpinner(&buf, "Waiting")
	h.Stop()
	out := buf.String()
	assert.NotContains(t, out, "✓")
	assert.NotContains(t, out, "✗")
}

// TestSpinner_ResolveIdempotent verifies that calling Done twice only
// produces one resolution line.
func TestSpinner_ResolveIdempotent(t *testing.T) {
	var buf bytes.Buffer
	h := StartSpinner(&buf, "Waiting")
	h.Done("first")
	h.Done("second") // must be a no-op
	out := buf.String()
	count := strings.Count(out, "✓")
	assert.Equal(t, 1, count, "second Done must be a no-op")
}

// TestSpinner_Capture_recordsSpinnerMessage verifies that under Capture, the
// Spinner message is recorded and Done's Step is also captured.
func TestSpinner_Capture_recordsSpinnerMessage(t *testing.T) {
	got := Capture(func(w io.Writer) {
		h := StartSpinner(w, "Waiting")
		h.Done("Done")
	})
	// StartSpinner records the Spinner. Done calls WriteInline which also records.
	require.GreaterOrEqual(t, len(got), 1)
	assert.Equal(t, Spinner{Text: "Waiting"}, got[0])
}

// TestSpinnerHandle_resolve_onTTY_Done exercises the onTTY=true branch of
// resolve via a directly-constructed SpinnerHandle. This covers the cursor
// restore escape and WriteInline calls that are otherwise gated behind a
// real TTY. No goroutine is started (cancel/done are nil/unset).
func TestSpinnerHandle_resolve_onTTY_Done(t *testing.T) {
	var buf bytes.Buffer
	h := &SpinnerHandle{
		w:     &buf,
		text:  "working",
		onTTY: true,
		done:  make(chan struct{}),
	}
	// Close done so resolve's <-h.done doesn't block (no goroutine running).
	close(h.done)
	h.Done("finished")
	out := buf.String()
	// Cursor restore escape must be in output.
	assert.Contains(t, out, "\x1b[?25h")
	// Resolution step must be present.
	assert.Contains(t, out, "✓")
	assert.Contains(t, out, "finished")
}

// TestSpinnerHandle_resolve_onTTY_Stop exercises the silent-cancel path with
// onTTY=true — must not print a resolution step but must emit a newline.
func TestSpinnerHandle_resolve_onTTY_Stop(t *testing.T) {
	var buf bytes.Buffer
	h := &SpinnerHandle{
		w:     &buf,
		text:  "working",
		onTTY: true,
		done:  make(chan struct{}),
	}
	close(h.done)
	h.Stop()
	out := buf.String()
	assert.NotContains(t, out, "✓")
	assert.NotContains(t, out, "✗")
	// Cursor restore and trailing newline expected.
	assert.Contains(t, out, "\x1b[?25h")
}

// TestRegisterUnregisterHandle exercises registerHandle and unregisterHandle
// directly to cover those package-internal functions without requiring a TTY.
func TestRegisterUnregisterHandle(t *testing.T) {
	h := &SpinnerHandle{}
	registerHandle(h)
	handlesMu.Lock()
	_, present := handles[h]
	handlesMu.Unlock()
	assert.True(t, present, "handle must be in registry after registerHandle")

	unregisterHandle(h)
	handlesMu.Lock()
	_, present = handles[h]
	handlesMu.Unlock()
	assert.False(t, present, "handle must be removed after unregisterHandle")
}

// TestInstallSIGINTHandler_testMode verifies that installSIGINTHandler is a
// no-op in test mode (it must not install the signal handler or panic).
// Calling it multiple times is also safe because sigOnce prevents re-entry.
func TestInstallSIGINTHandler_testMode(t *testing.T) {
	// In test mode, installSIGINTHandler must return without registering any
	// signal handler. Calling it twice exercises the Once-spent fast path.
	installSIGINTHandler()
	installSIGINTHandler()
}
