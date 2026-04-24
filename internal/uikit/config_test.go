package uikit_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

func TestResolve_Auto_WithUTF8Lang_ReturnsUnicode(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LANG", "en_US.UTF-8")
	assert.Equal(t, uikit.GlyphUnicode, uikit.ResolveMode("auto"))
}

func TestResolve_Auto_WithCLang_ReturnsASCII(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	t.Setenv("LANG", "")
	assert.Equal(t, uikit.GlyphASCII, uikit.ResolveMode("auto"))
}

func TestResolve_ExplicitUnicode_Honoured(t *testing.T) {
	t.Setenv("LANG", "C")
	assert.Equal(t, uikit.GlyphUnicode, uikit.ResolveMode("unicode"))
}

func TestResolve_ExplicitASCII_Honoured(t *testing.T) {
	t.Setenv("LANG", "en_US.UTF-8")
	assert.Equal(t, uikit.GlyphASCII, uikit.ResolveMode("ascii"))
}

func TestResolve_NO_COLOR_IsOrthogonal(t *testing.T) {
	// NO_COLOR strips colour, not glyphs. Explicitly assert the same LANG
	// still resolves to unicode when NO_COLOR is set.
	// LC_ALL and LC_CTYPE are cleared so the test is not affected by the
	// caller's shell locale (common CI hardening sets LC_ALL=C).
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("LANG", "en_US.UTF-8")
	assert.Equal(t, uikit.GlyphUnicode, uikit.ResolveMode("auto"))
}

func TestResolve_Auto_WithLCCTYPE_ReturnsUnicode(t *testing.T) {
	// LC_CTYPE is consulted when LC_ALL is empty.
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "en_US.UTF-8")
	t.Setenv("LANG", "")
	assert.Equal(t, uikit.GlyphUnicode, uikit.ResolveMode("auto"))
	t.Setenv("LC_CTYPE", "") // restore for parallel tests
}

func TestUse_And_ActiveMode(t *testing.T) {
	// SetModeForTest resets state so Use can be called from tests.
	uikit.SetModeForTest(uikit.GlyphASCII)
	uikit.Use("unicode")
	// After Use("unicode"), ActiveMode should return GlyphUnicode.
	assert.Equal(t, uikit.GlyphUnicode, uikit.ActiveMode())
}

func TestSetModeForTest_OverridesMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	assert.Equal(t, uikit.GlyphASCII, uikit.ActiveMode())
	uikit.SetModeForTest(uikit.GlyphUnicode)
	assert.Equal(t, uikit.GlyphUnicode, uikit.ActiveMode())
}

func TestResolve_Auto_AllLocaleVarsEmpty_ReturnsASCII(t *testing.T) {
	// When all three locale variables are empty, auto-detect falls through to
	// ASCII (the safe fallback for unknown terminals).
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "")
	assert.Equal(t, uikit.GlyphASCII, uikit.ResolveMode("auto"))
}

func TestResolve_UnknownValue_FallsThroughToAutoDetect(t *testing.T) {
	// Unknown config values (e.g. typos) fall through to auto-detect rather
	// than silently defaulting to ASCII, so that users in UTF-8 locales still
	// get unicode glyphs despite a misconfigured config value.
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "en_US.UTF-8")
	assert.Equal(t, uikit.GlyphUnicode, uikit.ResolveMode("garbage"))

	t.Setenv("LANG", "C")
	assert.Equal(t, uikit.GlyphASCII, uikit.ResolveMode("garbage"))
}
