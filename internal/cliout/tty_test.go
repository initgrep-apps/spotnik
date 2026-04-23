package cliout

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

// nopWriter is an io.Writer that discards all bytes, used to test isTTY.
type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }
