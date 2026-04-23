package cliout

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAsk_validFirstAttempt_returnsValue verifies that a valid first input is
// returned without error and the label is printed.
func TestAsk_validFirstAttempt_returnsValue(t *testing.T) {
	r := strings.NewReader("hello\n")
	var buf bytes.Buffer
	got, err := Ask(r, &buf, Prompt{Label: "Name"})
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
	assert.Contains(t, buf.String(), "Name:")
}

// TestAsk_trimsWhitespace verifies that surrounding whitespace is stripped.
func TestAsk_trimsWhitespace(t *testing.T) {
	r := strings.NewReader("  hello  \n")
	got, err := Ask(r, &bytes.Buffer{}, Prompt{Label: "Name"})
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

// TestAsk_validatorFails_thenSucceeds verifies that a failing first attempt
// prints a failure step and the second (valid) attempt returns the value.
func TestAsk_validatorFails_thenSucceeds(t *testing.T) {
	r := strings.NewReader("bad\nabc\n")
	var buf bytes.Buffer
	got, err := Ask(r, &buf, Prompt{
		Label: "Code",
		Validate: func(s string) error {
			if s != "abc" {
				return errors.New("must be abc")
			}
			return nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "abc", got)
	assert.Contains(t, buf.String(), "✗")
	assert.Contains(t, buf.String(), "must be abc")
}

// TestAsk_threeValidationFailures_returnsErrAborted verifies that three
// consecutive validation failures exhaust attempts and return ErrAborted.
func TestAsk_threeValidationFailures_returnsErrAborted(t *testing.T) {
	r := strings.NewReader("bad\nbad\nbad\n")
	var buf bytes.Buffer
	_, err := Ask(r, &buf, Prompt{
		Label: "Code",
		Validate: func(s string) error {
			return errors.New("nope")
		},
	})
	require.ErrorIs(t, err, ErrAborted)
	assert.Contains(t, buf.String(), "Giving up after 3 attempts")
}

// TestAsk_EOF_returnsErrAborted verifies that an empty reader (EOF) returns
// ErrAborted and prints an "Aborted" failure step so the caller can safely
// treat the error as already-printed.
func TestAsk_EOF_returnsErrAborted(t *testing.T) {
	r := strings.NewReader("")
	var buf bytes.Buffer
	_, err := Ask(r, &buf, Prompt{Label: "Name"})
	require.ErrorIs(t, err, ErrAborted)
	assert.Contains(t, buf.String(), "Aborted", "EOF must print an abort step")
}

// TestAsk_placeholderShownFirstAttemptOnly verifies that the placeholder text
// appears exactly once (on the first attempt only).
func TestAsk_placeholderShownFirstAttemptOnly(t *testing.T) {
	r := strings.NewReader("bad\nok\n")
	var buf bytes.Buffer
	_, _ = Ask(r, &buf, Prompt{
		Label:       "Code",
		Placeholder: "type ok",
		Validate: func(s string) error {
			if s != "ok" {
				return errors.New("nope")
			}
			return nil
		},
	})
	count := strings.Count(buf.String(), "type ok")
	assert.Equal(t, 1, count, "placeholder must appear exactly once")
}

// TestAsk_capture_recordsPromptMessage verifies that under Capture, the Prompt
// message is recorded.
func TestAsk_capture_recordsPromptMessage(t *testing.T) {
	got := Capture(func(w io.Writer) {
		_, _ = Ask(strings.NewReader(""), w, Prompt{Label: "Name"})
	})
	require.Len(t, got, 1)
	assert.Equal(t, "Name", got[0].(Prompt).Label)
}

// TestAsk_scannerReadError_returnsError verifies that a read error from the
// underlying io.Reader (not just EOF) propagates back from Ask and prints a
// failure step so the caller can safely treat the error as already-printed.
func TestAsk_scannerReadError_returnsError(t *testing.T) {
	sentErr := errors.New("disk error")
	r := &errReader{err: sentErr}
	var buf bytes.Buffer
	_, err := Ask(r, &buf, Prompt{Label: "Name"})
	assert.Error(t, err)
	assert.Contains(t, buf.String(), "read error", "IO error must print a failure step")
}

// errReader is an io.Reader that always returns an error after emitting a
// partial non-EOF read so that bufio.Scanner.Scan returns false and
// bufio.Scanner.Err returns a non-nil error.
type errReader struct {
	err error
}

func (e *errReader) Read(_ []byte) (int, error) {
	return 0, e.err
}
