package uikit

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// FormFieldConfig holds constructor options for a FormField.
type FormFieldConfig struct {
	// Label is rendered above the input in Muted colour, followed by ":".
	Label string

	// Placeholder is displayed in the text input when it is empty.
	Placeholder string

	// Validate is called by Validate() to check the current value.
	// A nil Validate always passes.
	Validate func(string) error

	// Theme provides the colour tokens.
	Theme theme.Theme
}

// FormField is a labelled text input with an intrinsic validator and an error
// slot rendered beneath the input. It wraps bubbles/textinput.Model.
//
// Validation runs on demand via Validate(). On failure the error is cached and
// rendered in Error colour under the input until the next SetValue or Validate
// call clears it.
type FormField struct {
	cfg    FormFieldConfig
	input  textinput.Model
	errMsg string
}

// NewFormField constructs a FormField from cfg. The embedded textinput is ready
// for use; call Focus() to activate cursor blinking.
func NewFormField(cfg FormFieldConfig) *FormField {
	ti := textinput.New()
	ti.Placeholder = cfg.Placeholder
	// Reasonable defaults — callers may adjust via SetWidth if needed.
	ti.Width = 60
	ti.CharLimit = 256

	// Wire theme roles so the colours follow the active theme when rendered.
	// Input.Text = Plain (TextPrimary); Input.Cursor = Accent.
	ti.TextStyle = lipgloss.NewStyle().Foreground(cfg.Theme.TextPrimary())
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(cfg.Theme.Accent())

	return &FormField{cfg: cfg, input: ti}
}

// Focus activates cursor blinking on the underlying textinput.
func (f *FormField) Focus() {
	f.input.Focus()
}

// Blur removes the cursor focus from the underlying textinput.
func (f *FormField) Blur() {
	f.input.Blur()
}

// Update forwards the message to the embedded textinput and returns the updated
// FormField plus any command the textinput needs. The error slot is NOT cleared
// here — callers decide when to clear it (typically on Enter, then re-validate).
func (f *FormField) Update(msg tea.Msg) (*FormField, tea.Cmd) {
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return f, cmd
}

// Render returns the full rendered string: label line, input box, and optional
// error line. No side effects — reads only field state.
func (f *FormField) Render() string {
	t := f.cfg.Theme

	labelStyle := lipgloss.NewStyle().Foreground(t.TextMuted())
	inputBorderStyle := lipgloss.NewStyle().
		Border(RoundedBorder()).
		BorderForeground(t.Accent()).
		Padding(0, 1)

	label := labelStyle.Render(f.cfg.Label + ":")
	inputBox := inputBorderStyle.Render(f.input.View())

	if f.errMsg == "" {
		return lipgloss.JoinVertical(lipgloss.Left, label, inputBox)
	}

	errGlyph := GlyphFor(GlyphError, ActiveMode())
	// ValidationError role: glyph in Error colour, message text in Plain (TextPrimary).
	// The two distinct foreground colours allow visual hierarchy without an icon library.
	glyphStyle := lipgloss.NewStyle().Foreground(t.Error())
	textStyle := lipgloss.NewStyle().Foreground(t.TextPrimary())
	errLine := glyphStyle.Render(errGlyph) + " " + textStyle.Render(f.errMsg)

	return lipgloss.JoinVertical(lipgloss.Left, label, inputBox, errLine)
}

// Value returns the current text from the embedded textinput.
func (f *FormField) Value() string {
	return f.input.Value()
}

// SetValue sets the embedded textinput's value and clears any cached
// validation error so the error slot disappears when the user starts editing.
func (f *FormField) SetValue(v string) {
	f.input.SetValue(v)
	f.errMsg = ""
}

// Validate runs the configured validator against the current value. On failure
// the error message is cached (visible in Render()) and the error is returned.
// On success the cached error is cleared and nil is returned. Validate is a
// no-op validator when cfg.Validate is nil — it always returns nil.
func (f *FormField) Validate() error {
	if f.cfg.Validate == nil {
		f.errMsg = ""
		return nil
	}
	err := f.cfg.Validate(f.input.Value())
	if err != nil {
		f.errMsg = err.Error()
		return err
	}
	f.errMsg = ""
	return nil
}

// ValidationError returns the cached error message from the most recent failed
// Validate() call. Returns "" when no error is cached.
func (f *FormField) ValidationError() string {
	return f.errMsg
}

// InputTextStyle returns the lipgloss.Style applied to typed text inside the
// embedded textinput. Exposed so tests can assert that Input.Text is wired to
// the Plain (TextPrimary) role without relying on rendered ANSI output.
func (f *FormField) InputTextStyle() lipgloss.Style {
	return f.input.TextStyle
}

// InputCursorStyle returns the lipgloss.Style applied to the cursor inside the
// embedded textinput. Exposed so tests can assert that Input.Cursor is wired to
// the Accent role without relying on rendered ANSI output.
func (f *FormField) InputCursorStyle() lipgloss.Style {
	return f.input.Cursor.Style
}
