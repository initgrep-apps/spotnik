package uikit

var spinnerFramesUnicode = []string{
	"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
}

var spinnerFramesASCII = []string{"|", "/", "-", "\\"}

// SpinnerFrames returns a defensive copy of the animation frames for the given
// mode. Each call returns a fresh slice so callers may not corrupt the
// package-level backing slices (both uikit.Spinner and cliout.Spinner share
// those backing slices).
// Unknown modes fall back to GlyphUnicode frames.
func SpinnerFrames(m GlyphMode) []string {
	switch m {
	case GlyphASCII:
		return append([]string(nil), spinnerFramesASCII...)
	case GlyphUnicode:
		return append([]string(nil), spinnerFramesUnicode...)
	default:
		return append([]string(nil), spinnerFramesUnicode...)
	}
}
