package uikit

import (
	"os"
	"strings"
	"sync"
)

// ResolveMode converts the user-facing config value ("auto", "unicode",
// "ascii") to a concrete GlyphMode. Auto inspects LC_ALL then LANG for a
// UTF-8 marker; anything else resolves to ASCII.
//
// NO_COLOR is orthogonal — it strips colour, not glyphs — and is not
// consulted here.
func ResolveMode(cfg string) GlyphMode {
	switch strings.ToLower(strings.TrimSpace(cfg)) {
	case "unicode":
		return GlyphUnicode
	case "ascii":
		return GlyphASCII
	default: // "auto" or empty
		if isUTF8Locale() {
			return GlyphUnicode
		}
		return GlyphASCII
	}
}

func isUTF8Locale() bool {
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		v := os.Getenv(key)
		if v == "" {
			continue
		}
		lower := strings.ToLower(v)
		if strings.Contains(lower, "utf-8") || strings.Contains(lower, "utf8") {
			return true
		}
		// First non-empty locale variable wins; a non-UTF-8 value rules out
		// unicode mode.
		return false
	}
	return false
}

// activeMode stores the resolved mode after first call to Use.
var (
	activeModeOnce sync.Once
	activeMode     GlyphMode
)

// Use resolves and caches the active GlyphMode. Called once at app startup
// after config is loaded. Subsequent calls are no-ops (mode is frozen for
// the lifetime of the process).
func Use(cfg string) {
	activeModeOnce.Do(func() {
		activeMode = ResolveMode(cfg)
	})
}

// ActiveMode returns the cached mode. Tests use SetModeForTest instead.
func ActiveMode() GlyphMode { return activeMode }

// SetModeForTest overrides the active mode in tests. Resets the sync.Once so
// subsequent Use calls can reinitialise.
func SetModeForTest(m GlyphMode) {
	activeMode = m
	activeModeOnce = sync.Once{}
}
