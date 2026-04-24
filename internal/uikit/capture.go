package uikit

import (
	"regexp"
	"strings"
)

// ansiEscape matches any ANSI CSI escape sequence. Used by Capture to produce
// plain-text lines for structural assertions in primitive tests.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// Capture returns the rendered string with all ANSI colour/style codes stripped
// and split into individual lines. Used in primitive tests to assert on
// structural content without colour noise.
func Capture(rendered string) []string {
	plain := ansiEscape.ReplaceAllString(rendered, "")
	return strings.Split(plain, "\n")
}
