package app

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStderr redirects os.Stderr through a pipe for the duration of fn,
// returns whatever fn wrote. Mutates a process global — callers must not
// run with t.Parallel().
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	require.NoError(t, w.Close())
	os.Stderr = orig
	return <-done
}

func TestCopyToClipboardCmd_emitsOSC52ToStderr(t *testing.T) {
	const payload = "https://accounts.spotify.com/authorize?client_id=test"

	var msg interface{}
	out := captureStderr(t, func() {
		msg = copyToClipboardCmd(payload)()
	})

	copied, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok, "expected clipboardCopiedMsg, got %T", msg)
	assert.NoError(t, copied.Err)

	// OSC 52 frame: ESC ] 52 ; c ; <base64> BEL
	assert.True(t, strings.HasPrefix(out, "\x1b]52;c;"),
		"stderr must start with OSC 52 prefix; got %q", out)
	assert.True(t, strings.HasSuffix(out, "\x07"),
		"stderr must end with BEL terminator; got %q", out)

	body := strings.TrimSuffix(strings.TrimPrefix(out, "\x1b]52;c;"), "\x07")
	decoded, err := base64.StdEncoding.DecodeString(body)
	require.NoError(t, err, "OSC 52 payload must be valid base64")
	assert.Equal(t, payload, string(decoded))
}

func TestCopyToClipboardCmd_emptyText_emitsResetSequence(t *testing.T) {
	var msg interface{}
	out := captureStderr(t, func() {
		msg = copyToClipboardCmd("")()
	})

	copied, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok)
	assert.NoError(t, copied.Err)

	// Reset form per upstream: ESC ] 52 ; c ; BEL with empty payload.
	assert.Equal(t, "\x1b]52;c;\x07", out)
}

func TestCopyToClipboardCmd_brokenStderr_returnsError(t *testing.T) {
	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, r.Close()) // close read end so the write fails
	os.Stderr = w
	defer func() {
		_ = w.Close()
		os.Stderr = orig
	}()

	msg := copyToClipboardCmd("anything")()
	copied, ok := msg.(clipboardCopiedMsg)
	require.True(t, ok)
	assert.Error(t, copied.Err)
}

func TestUpdate_clipboardCopiedMsg_success_returnsToastCmd(t *testing.T) {
	a := newTestApp(false)
	_, cmd := a.Update(clipboardCopiedMsg{})
	require.NotNil(t, cmd, "success path must enqueue a toast cmd")
}

func TestUpdate_clipboardCopiedMsg_error_returnsToastCmd(t *testing.T) {
	a := newTestApp(false)
	_, cmd := a.Update(clipboardCopiedMsg{Err: errors.New("boom")})
	require.NotNil(t, cmd, "error path must enqueue a toast cmd")
}
