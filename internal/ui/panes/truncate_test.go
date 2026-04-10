package panes

// truncate_test.go — White-box tests for truncateRunes.
// The production code in profile.go uses truncateRunes to cap display names.
// This file pins the truncation contract (rune count, ellipsis, unicode safety).

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateRunes_ShortString(t *testing.T) {
	assert.Equal(t, "Alice", truncateRunes("Alice", 20))
}

func TestTruncateRunes_ExactLength(t *testing.T) {
	s := strings.Repeat("a", 20)
	assert.Equal(t, s, truncateRunes(s, 20))
}

func TestTruncateRunes_LongString(t *testing.T) {
	s := strings.Repeat("a", 25)
	result := truncateRunes(s, 20)
	assert.True(t, len([]rune(result)) <= 20, "result must not exceed max runes")
	assert.True(t, strings.HasSuffix(result, "…"), "result must end with ellipsis")
	assert.NotContains(t, result, s, "result must not contain the full original string")
}

func TestTruncateRunes_UnicodeRunes(t *testing.T) {
	// 21 multi-byte runes — must count runes not bytes.
	s := strings.Repeat("é", 21)
	result := truncateRunes(s, 20)
	assert.True(t, len([]rune(result)) <= 20)
	assert.True(t, strings.HasSuffix(result, "…"))
}

func TestTruncateRunes_EmptyString(t *testing.T) {
	assert.Equal(t, "", truncateRunes("", 20))
}
