package layout_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
	}{
		{
			name:     "short string unchanged",
			input:    "hello",
			maxWidth: 10,
			want:     "hello",
		},
		{
			name:     "exact width string unchanged",
			input:    "hello",
			maxWidth: 5,
			want:     "hello",
		},
		{
			name:     "long string truncated with ellipsis",
			input:    "hello world",
			maxWidth: 8,
			want:     "hello w…",
		},
		{
			name:     "empty string unchanged",
			input:    "",
			maxWidth: 10,
			want:     "",
		},
		{
			name:     "maxWidth zero returns empty",
			input:    "hello",
			maxWidth: 0,
			want:     "",
		},
		{
			name:     "maxWidth one returns ellipsis only",
			input:    "hello",
			maxWidth: 1,
			want:     "…",
		},
		{
			name:     "unicode CJK truncated at correct column boundary",
			input:    "日本語テスト",
			maxWidth: 5,
			want:     "日本…",
		},
		{
			name:     "string with ANSI escape codes counted by rendered width",
			input:    "\x1b[32mgreen\x1b[0m text",
			maxWidth: 8,
			// "green text" = 10 visible cols; maxWidth 8 → keep 7 + "…" = 8
			want: "\x1b[32mgreen\x1b[0m t…",
		},
		{
			name:     "single char wider than maxWidth returns ellipsis",
			input:    "日",
			maxWidth: 1,
			want:     "…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layout.Truncate(tt.input, tt.maxWidth)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{
			name:  "short string padded to width",
			input: "hi",
			width: 5,
			want:  "hi   ",
		},
		{
			name:  "exact width string unchanged",
			input: "hello",
			width: 5,
			want:  "hello",
		},
		{
			name:  "wider string returned unchanged",
			input: "hello world",
			width: 5,
			want:  "hello world",
		},
		{
			name:  "empty string padded to width",
			input: "",
			width: 3,
			want:  "   ",
		},
		{
			name:  "zero width returns string unchanged",
			input: "hi",
			width: 0,
			want:  "hi",
		},
		{
			name:  "CJK characters padded correctly",
			input: "日",
			width: 5,
			want:  "日   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layout.PadRight(tt.input, tt.width)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncateOrPad(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{
			name:  "long string is truncated",
			input: "hello world",
			width: 8,
			want:  "hello w…",
		},
		{
			name:  "short string is padded",
			input: "hi",
			width: 5,
			want:  "hi   ",
		},
		{
			name:  "exact width string unchanged",
			input: "hello",
			width: 5,
			want:  "hello",
		},
		{
			name:  "empty string padded",
			input: "",
			width: 4,
			want:  "    ",
		},
		{
			name:  "zero width returns empty",
			input: "hello",
			width: 0,
			want:  "",
		},
		{
			name:  "CJK: short padded correctly",
			input: "日",
			width: 5,
			want:  "日   ",
		},
		{
			name:  "CJK: long truncated correctly",
			input: "日本語テスト",
			width: 5,
			want:  "日本…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layout.TruncateOrPad(tt.input, tt.width)
			assert.Equal(t, tt.want, got)
		})
	}
}
