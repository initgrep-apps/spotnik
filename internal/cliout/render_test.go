package cliout

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrite_emptySlice_writesNothing(t *testing.T) {
	var buf bytes.Buffer
	Write(&buf)
	assert.Empty(t, buf.String())
}

func TestWrite_addsLeadingBlankAndIndent(t *testing.T) {
	var buf bytes.Buffer
	Write(&buf, Paragraph{Text: "hello"})
	out := buf.String()
	assert.True(t, strings.HasPrefix(out, "\n"), "expected leading blank line, got: %q", out)
	assert.Contains(t, out, "  hello") // 2-char left indent
}

func TestWriteInline_noLeadingBlank(t *testing.T) {
	var buf bytes.Buffer
	WriteInline(&buf, Paragraph{Text: "hello"})
	out := buf.String()
	assert.False(t, strings.HasPrefix(out, "\n"), "WriteInline must not add leading blank, got: %q", out)
	assert.Contains(t, out, "  hello") // 2-char left indent still present
}

func TestWrite_joinsMessagesWithNewline(t *testing.T) {
	var buf bytes.Buffer
	Write(&buf,
		Header{Status: Active, Subject: "Spotnik", State: "authenticated"},
		Paragraph{Text: "body"},
	)
	out := buf.String()
	assert.Contains(t, out, "Spotnik")
	assert.Contains(t, out, "body")
	// Both messages are present — joined by newline within the rendered block.
	assert.Contains(t, out, "\n")
}

func TestWriteInline_emptySlice_writesNothing(t *testing.T) {
	var buf bytes.Buffer
	WriteInline(&buf)
	assert.Empty(t, buf.String())
}
