package cliout

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// stripAnsi removes ANSI escape sequences so tests can assert on visible text only.
func stripAnsi(s string) string {
	re := regexp.MustCompile("\x1b\\[[0-9;]*m")
	return re.ReplaceAllString(s, "")
}

func TestStatusColor_allBranches(t *testing.T) {
	cases := []struct {
		s    Status
		want lipgloss.TerminalColor
	}{
		{Active, Fixed.Accent},
		{Inactive, Fixed.Muted},
		{StatusSuccess, Fixed.Success},
		{StatusFailure, Fixed.Error},
		{StatusWarning, Fixed.Warning},
		{Pending, Fixed.Muted},
		{Status(99), Fixed.Plain},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, statusColor(c.s, Fixed), "statusColor(%v)", c.s)
	}
}

func TestStatusGlyph(t *testing.T) {
	cases := []struct {
		s    Status
		want string
	}{
		{Active, "◉"},
		{Inactive, "◎"},
		{StatusSuccess, "✓"},
		{StatusFailure, "✗"},
		{StatusWarning, "◬"},
		{Pending, "◌"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, statusGlyph(c.s), "statusGlyph(%v)", c.s)
	}
}

func TestStatusGlyph_unknown(t *testing.T) {
	// Any unknown Status value must return "?" without panicking.
	assert.Equal(t, "?", statusGlyph(Status(99)))
}

// TestStatusGlyph_AsciiMode verifies that every Status value returns its ASCII
// form when uikit is pinned to ASCII mode.
func TestStatusGlyph_AsciiMode(t *testing.T) {
	prev := uikit.ActiveMode()
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(prev)

	cases := []struct {
		s    Status
		want string
	}{
		{Active, uikit.GlyphFor(uikit.GlyphActive, uikit.GlyphASCII)},
		{Inactive, uikit.GlyphFor(uikit.GlyphInactive, uikit.GlyphASCII)},
		{StatusSuccess, uikit.GlyphFor(uikit.GlyphSuccess, uikit.GlyphASCII)},
		{StatusFailure, uikit.GlyphFor(uikit.GlyphError, uikit.GlyphASCII)},
		{StatusWarning, uikit.GlyphFor(uikit.GlyphWarning, uikit.GlyphASCII)},
		{Pending, uikit.GlyphFor(uikit.GlyphLocked, uikit.GlyphASCII)},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, statusGlyph(c.s), "statusGlyph(%v) in ASCII mode", c.s)
	}
}

func TestHeader_renderActive(t *testing.T) {
	h := Header{Status: Active, Subject: "Spotnik", State: "authenticated"}
	out := h.render(Fixed)
	assert.Contains(t, out, "◉")
	assert.Contains(t, out, "Spotnik")
	assert.Contains(t, out, "authenticated")
}

func TestHeader_renderSuccess(t *testing.T) {
	h := Header{Status: StatusSuccess, Subject: "Auth", State: "complete"}
	out := h.render(Fixed)
	assert.Contains(t, out, "✓")
	assert.Contains(t, out, "Auth")
	assert.Contains(t, out, "complete")
}

func TestStep_render(t *testing.T) {
	s := Step{Status: StatusSuccess, Text: "Authorization received"}
	out := s.render(Fixed)
	assert.Contains(t, out, "✓")
	assert.Contains(t, out, "Authorization received")
}

func TestKV_alignsLabels(t *testing.T) {
	kv := KV{Pairs: []KVPair{
		{Label: "Client ID", Value: "present"},
		{Label: "Expires", Value: "Wed, 23 Apr"},
	}}
	out := kv.render(Fixed)
	lines := strings.Split(stripAnsi(out), "\n")
	require.Len(t, lines, 2)
	// "Client ID" is 9 chars, "Expires" is 7 — pad "Expires" by 2.
	assert.True(t, strings.HasPrefix(lines[0], "Client ID  "), "first line must start with 'Client ID  ', got: %q", lines[0])
	assert.True(t, strings.HasPrefix(lines[1], "Expires    "), "second line must start with 'Expires    ', got: %q", lines[1])
}

func TestKV_empty_returnsEmptyString(t *testing.T) {
	kv := KV{Pairs: nil}
	assert.Equal(t, "", kv.render(Fixed))
}

func TestKV_withCaption(t *testing.T) {
	kv := KV{Pairs: []KVPair{
		{Label: "Token", Value: "abc123", Caption: "expires soon"},
	}}
	out := stripAnsi(kv.render(Fixed))
	assert.Contains(t, out, "Token")
	assert.Contains(t, out, "abc123")
	assert.Contains(t, out, "expires soon")
}

func TestSteps_numbersFrom1(t *testing.T) {
	s := Steps{Items: []string{"first", "second", "third"}}
	out := stripAnsi(s.render(Fixed))
	lines := strings.Split(out, "\n")
	require.Len(t, lines, 3)
	assert.True(t, strings.HasPrefix(lines[0], "1"), "first line starts with 1, got: %q", lines[0])
	assert.True(t, strings.HasPrefix(lines[1], "2"), "second line starts with 2, got: %q", lines[1])
	assert.True(t, strings.HasPrefix(lines[2], "3"), "third line starts with 3, got: %q", lines[2])
}

func TestSteps_empty_returnsEmptyString(t *testing.T) {
	s := Steps{Items: nil}
	assert.Equal(t, "", s.render(Fixed))
}

func TestHint_renderWithAllFields(t *testing.T) {
	h := Hint{Verb: "Run", Cmd: "spotnik auth login", Tail: "to reconnect"}
	out := stripAnsi(h.render(Fixed))
	assert.Equal(t, "→ Run spotnik auth login to reconnect", out)
}

func TestHint_omitsEmptyFields(t *testing.T) {
	h := Hint{Cmd: "spotnik auth register"}
	out := stripAnsi(h.render(Fixed))
	assert.Equal(t, "→ spotnik auth register", out)
}

func TestHint_arrowOnlyWhenAllEmpty(t *testing.T) {
	h := Hint{}
	out := stripAnsi(h.render(Fixed))
	assert.Equal(t, "→", out)
}

func TestURL_noLabel_rendersHrefOnly(t *testing.T) {
	u := URL{Href: "https://example.com"}
	out := stripAnsi(u.render(Fixed))
	assert.Equal(t, "https://example.com", out)
}

func TestURL_withLabel_rendersTwoLines(t *testing.T) {
	u := URL{Label: "Visit:", Href: "https://example.com"}
	out := stripAnsi(u.render(Fixed))
	assert.Equal(t, "Visit:\nhttps://example.com", out)
}

func TestParagraph_plain(t *testing.T) {
	p := Paragraph{Text: "plain line"}
	out := stripAnsi(p.render(Fixed))
	assert.Equal(t, "plain line", out)
}

func TestParagraph_dim(t *testing.T) {
	p := Paragraph{Text: "dim line", Dim: true}
	out := stripAnsi(p.render(Fixed))
	assert.Equal(t, "dim line", out)
}

func TestSpinner_renderPanicsUntilStory149(t *testing.T) {
	assert.Panics(t, func() { Spinner{Text: "x"}.render(Fixed) })
}

func TestPrompt_renderPanicsUntilStory149(t *testing.T) {
	assert.Panics(t, func() { Prompt{Label: "x"}.render(Fixed) })
}

// Verify all static message types satisfy the Message interface at compile time.
var _ Message = Header{}
var _ Message = Step{}
var _ Message = KV{}
var _ Message = Steps{}
var _ Message = Hint{}
var _ Message = URL{}
var _ Message = Paragraph{}
var _ Message = Spinner{}
var _ Message = Prompt{}

// TestIsMessage_markers calls every isMessage() method so that coverage counts
// the no-op marker methods rather than leaving them at 0%. The methods exist
// only to enforce the sealed Message interface within the package.
func TestIsMessage_markers(t *testing.T) {
	Header{}.isMessage()
	Step{}.isMessage()
	KV{}.isMessage()
	Steps{}.isMessage()
	Hint{}.isMessage()
	URL{}.isMessage()
	Paragraph{}.isMessage()
	Spinner{}.isMessage()
	Prompt{}.isMessage()
}
