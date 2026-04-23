package cliout

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapture_recordsMessagesWithoutRendering(t *testing.T) {
	got := Capture(func(w io.Writer) {
		Write(w,
			Header{Status: Active, Subject: "Spotnik", State: "authenticated"},
			Paragraph{Text: "body"},
		)
	})

	require.Len(t, got, 2)
	assert.Equal(t,
		Header{Status: Active, Subject: "Spotnik", State: "authenticated"},
		got[0])
	assert.Equal(t, Paragraph{Text: "body"}, got[1])
}

func TestCapture_nested_restoresPreviousRecorder(t *testing.T) {
	outer := Capture(func(w io.Writer) {
		Write(w, Paragraph{Text: "outer before"})
		inner := Capture(func(w2 io.Writer) {
			Write(w2, Paragraph{Text: "inner"})
		})
		assert.Len(t, inner, 1)
		Write(w, Paragraph{Text: "outer after"})
	})

	// Outer recorder must only contain outer's writes; inner was captured separately.
	require.Len(t, outer, 2)
	assert.Equal(t, "outer before", outer[0].(Paragraph).Text)
	assert.Equal(t, "outer after", outer[1].(Paragraph).Text)
}

func TestSetTestMode_pinsASCII(t *testing.T) {
	SetTestMode(true)
	t.Cleanup(func() { SetTestMode(false) })
	// lipgloss profile is global — just verify SetTestMode(true) doesn't panic
	// and inTestMode reports true.
	assert.True(t, inTestMode())
}

func TestCapture_WriteInline_captured(t *testing.T) {
	got := Capture(func(w io.Writer) {
		WriteInline(w, Step{Status: StatusSuccess, Text: "done"})
	})
	require.Len(t, got, 1)
	assert.Equal(t, Step{Status: StatusSuccess, Text: "done"}, got[0])
}
