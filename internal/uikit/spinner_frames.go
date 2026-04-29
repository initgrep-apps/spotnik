package uikit

var spinnerFramesUnicode = []string{
	"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
}

var spinnerFramesASCII = []string{"|", "/", "-", "\\"}

// SpinnerFrames returns the animation frames for the given mode.
// The returned slice must NOT be mutated by the caller — it is a stable
// reference shared between uikit.Spinner and cliout.Spinner.
func SpinnerFrames(m GlyphMode) []string {
	if m == GlyphASCII {
		return spinnerFramesASCII
	}
	return spinnerFramesUnicode
}
