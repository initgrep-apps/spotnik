package cliout

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/initgrep-apps/spotnik/internal/uikit"
)

func TestCheckNoColor_withEnvVar(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	assert.True(t, checkNoColor())
}

func TestCheckNoColor_withoutEnvVar(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	assert.False(t, checkNoColor())
}

func TestIsTTY_nonFileWriter_returnsFalse(t *testing.T) {
	// bytes.Buffer is not an *os.File — must return false.
	var buf nopWriter
	assert.False(t, isTTY(buf))
}

func TestIsTTY_exported_matchesInternal(t *testing.T) {
	var buf nopWriter
	assert.Equal(t, isTTY(buf), IsTTY(buf))
}

// TestSetTestMode_RestoresPriorUikitMode verifies that SetTestMode(false)
// restores the uikit GlyphMode that was active before SetTestMode(true).
// Previously the restore was comment-only and SetTestMode(false) left uikit
// pinned to GlyphASCII.
func TestSetTestMode_RestoresPriorUikitMode(t *testing.T) {
	// Arrange: record the current uikit mode, then set it to a known value.
	originalMode := uikit.ActiveMode()
	defer uikit.SetModeForTest(originalMode) // always restore after the test

	// Pin uikit to unicode (the "prior" mode) so we can detect restore.
	uikit.SetModeForTest(uikit.GlyphUnicode)

	// Act: enable test mode (pins uikit to ASCII) then disable it.
	SetTestMode(true)
	assert.Equal(t, uikit.GlyphASCII, uikit.ActiveMode(), "SetTestMode(true) must pin uikit to GlyphASCII")

	SetTestMode(false)

	// Assert: uikit must return to the mode that was active before SetTestMode(true).
	assert.Equal(t, uikit.GlyphUnicode, uikit.ActiveMode(),
		"SetTestMode(false) must restore the prior uikit mode")
}

// nopWriter is an io.Writer that discards all bytes, used to test isTTY.
type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }
