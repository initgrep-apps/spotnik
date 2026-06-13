package panes

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTMLToMarkdown_Paragraphs(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "single paragraph",
			html: "<p>Hello world</p>",
			want: "Hello world",
		},
		{
			name: "two paragraphs",
			html: "<p>First para</p><p>Second para</p>",
			want: "First para\n\nSecond para",
		},
		{
			name: "nested paragraph with bold",
			html: "<p>This is <b>bold</b> text</p>",
			want: "This is **bold** text",
		},
		{
			name: "paragraph with italic",
			html: "<p>This is <i>italic</i> text</p>",
			want: "This is *italic* text",
		},
		{
			name: "paragraph with link",
			html: "<p>Visit <a href=\"https://example.com\">here</a></p>",
			want: "Visit [here]",
		},
		{
			name: "line break",
			html: "<p>Line1<br/>Line2</p>",
			want: "Line1  \nLine2",
		},
		{
			name: "unordered list",
			html: "<ul><li>Item 1</li><li>Item 2</li></ul>",
			want: "- Item 1\n\n- Item 2",
		},
		{
			name: "html entities",
			html: "<p>&#x1f44b; Hello &amp; goodbye</p>",
			want: "\U0001f44b Hello & goodbye",
		},
		{
			name: "empty html",
			html: "",
			want: "",
		},
		{
			name: "strong and em",
			html: "<p><strong>Warning</strong> with <em>emphasis</em></p>",
			want: "**Warning** with *emphasis*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := htmlToMarkdown(tt.html)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHTMLToMarkdown_ComplexSpotifyExample(t *testing.T) {
	html := `<p>&#x1f44b; Donate here or Subscribe Here.</p><p><a href="https://podcasters.spotify.com/pod/show/lofihealingquran/subscribe" rel="nofollow">Subscribe on this Playlist</a>&#x1f3a7;</p><p><a href="https://paypal.me/AkzMedia" rel="nofollow">Donate via PayPal</a> &#x2764;&#xfe0f;</p>`
	got := htmlToMarkdown(html)
	assert.Contains(t, got, "Donate here or Subscribe Here")
	assert.Contains(t, got, "Subscribe on this Playlist")
	assert.Contains(t, got, "Donate via PayPal")
	assert.NotContains(t, got, "<p>")
	assert.NotContains(t, got, "<a")
	assert.NotContains(t, got, "rel=")
	assert.NotContains(t, got, "</")
}

func TestRenderMarkdown_Basic(t *testing.T) {
	md := "# Hello\n\nThis is **bold** and *italic*."
	result, err := renderMarkdown(md, 40)
	require.NoError(t, err)
	assert.Contains(t, result, "Hello")
	assert.NotEmpty(t, result)
}

func TestRenderMarkdown_EmptyInput(t *testing.T) {
	result, err := renderMarkdown("", 40)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestRenderMarkdown_WordWrap(t *testing.T) {
	long := "This is a very long line that should definitely be wrapped by glamour to fit within the specified width"
	result, err := renderMarkdown(long, 20)
	require.NoError(t, err)
	lines := strings.Split(result, "\n")
	assert.GreaterOrEqual(t, len(lines), 2, "long text at width 20 should wrap to multiple lines")
}

func TestRenderMarkdown_BoldAndItalic(t *testing.T) {
	md := "**bold** and *italic*"
	result, err := renderMarkdown(md, 80)
	require.NoError(t, err)
	assert.Contains(t, result, "bold")
	assert.Contains(t, result, "italic")
}

func TestRenderHTMLDescription_Empty(t *testing.T) {
	result, err := renderMarkdown(htmlToMarkdown(""), 40)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestRenderHTMLDescription_Simple(t *testing.T) {
	md := htmlToMarkdown("<p>Simple description</p>")
	result, err := renderMarkdown(md, 40)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Simple")
	assert.Contains(t, result, "description")
}

func TestRenderHTMLDescription_WithEntities(t *testing.T) {
	md := htmlToMarkdown("<p>&#x1f44b; Hello world</p>")
	result, err := renderMarkdown(md, 40)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotContains(t, result, "&#x")
}
