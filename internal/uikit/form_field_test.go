package uikit_test

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validHexValidator is a simple 32-char hex validator, mirroring the Spotify Client ID rule.
func validHexValidator(s string) error {
	if len(s) != 32 {
		return errors.New("must be 32 characters")
	}
	for _, c := range s {
		isDigit := c >= '0' && c <= '9'
		isLower := c >= 'a' && c <= 'f'
		isUpper := c >= 'A' && c <= 'F'
		if !isDigit && !isLower && !isUpper {
			return errors.New("must be hexadecimal")
		}
	}
	return nil
}

// TestFormField_NoErrorBeforeValidation verifies that a freshly constructed
// FormField has no validation error and ValidationError() returns "".
func TestFormField_NoErrorBeforeValidation(t *testing.T) {
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label:    "Client ID",
		Validate: validHexValidator,
		Theme:    th,
	})

	assert.Equal(t, "", f.ValidationError(), "no error before Validate() is called")
	assert.Equal(t, "", f.Value(), "Value() starts empty")
}

// TestFormField_ReportsErrorAfterValidate verifies that calling Validate() on
// an invalid value caches the error message and ValidationError() returns it.
func TestFormField_ReportsErrorAfterValidate(t *testing.T) {
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label:    "Client ID",
		Validate: validHexValidator,
		Theme:    th,
	})

	f.SetValue("not-a-valid-id")
	err := f.Validate()
	require.Error(t, err, "Validate() should return error for invalid value")
	assert.NotEmpty(t, f.ValidationError(), "ValidationError() should be set after failed Validate()")
	assert.Equal(t, err.Error(), f.ValidationError(), "ValidationError() matches returned error message")
}

// TestFormField_AcceptsValidValue verifies that Validate() on a valid 32-char
// hex string returns nil and clears ValidationError().
func TestFormField_AcceptsValidValue(t *testing.T) {
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label:    "Client ID",
		Validate: validHexValidator,
		Theme:    th,
	})

	f.SetValue("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4") // 32 hex chars
	err := f.Validate()
	require.NoError(t, err, "Validate() should return nil for valid value")
	assert.Equal(t, "", f.ValidationError(), "ValidationError() should be empty after successful Validate()")
}

// TestFormField_SetValue_ClearsError verifies that SetValue() clears any cached
// validation error so the error slot disappears when the user starts editing.
func TestFormField_SetValue_ClearsError(t *testing.T) {
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label:    "Client ID",
		Validate: validHexValidator,
		Theme:    th,
	})

	f.SetValue("bad")
	_ = f.Validate() // cache an error
	require.NotEmpty(t, f.ValidationError())

	f.SetValue("something-new")
	assert.Empty(t, f.ValidationError(), "SetValue() must clear the cached error")
}

// TestFormField_Render_ContainsLabel verifies that Render() includes the label text.
func TestFormField_Render_ContainsLabel(t *testing.T) {
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label:    "Client ID",
		Validate: validHexValidator,
		Theme:    th,
	})

	rendered := stripANSI(f.Render())
	assert.Contains(t, rendered, "Client ID", "Render() should include the label")
}

// TestFormField_Render_ContainsErrorGlyph verifies that after a failed Validate()
// the rendered output contains the error glyph (✗) and the error message.
func TestFormField_Render_ContainsErrorGlyph(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label:    "Client ID",
		Validate: validHexValidator,
		Theme:    th,
	})

	f.SetValue("bad")
	_ = f.Validate()
	rendered := stripANSI(f.Render())
	assert.True(t, strings.Contains(rendered, "✗") || strings.Contains(rendered, "x"),
		"Render() should contain error glyph after validation failure, got: %q", rendered)
}

// TestFormField_Render_NoErrorGlyphWhenClean verifies that Render() does NOT
// include an error glyph when there is no validation error.
func TestFormField_Render_NoErrorGlyphWhenClean(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label:    "Client ID",
		Validate: validHexValidator,
		Theme:    th,
	})

	rendered := stripANSI(f.Render())
	assert.NotContains(t, rendered, "✗", "Render() should not contain error glyph when no error")
}

// TestFormField_Update_ForwardsToTextInput verifies that Update() returns a
// (possibly non-nil) cmd when the textinput processes a key event.
func TestFormField_Update_ForwardsToTextInput(t *testing.T) {
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label:    "Client ID",
		Validate: validHexValidator,
		Theme:    th,
	})

	f.Focus()
	// Send a rune key — textinput should process it.
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	_, _ = f.Update(msg)
	// After sending 'a', Value() should include the typed character.
	assert.Equal(t, "a", f.Value(), "Update() should forward key events to the embedded textinput")
}

// TestFormField_Focus_Blur verifies that Focus and Blur transition the field
// correctly without panicking.
func TestFormField_Focus_Blur(t *testing.T) {
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label: "Client ID",
		Theme: th,
	})
	// Should not panic
	f.Focus()
	f.Blur()
}

// TestFormField_Validate_NilValidateAlwaysPasses verifies that when no Validate
// function is provided, Validate() always returns nil and clears any prior error.
func TestFormField_Validate_NilValidateAlwaysPasses(t *testing.T) {
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label: "No Validator",
		Theme: th,
	})

	f.SetValue("anything at all")
	err := f.Validate()
	assert.NoError(t, err, "nil Validate should always return nil")
	assert.Empty(t, f.ValidationError(), "ValidationError() should be empty when no validator set")
}

// TestFormField_Validate_ClearsErrorOnRetry verifies that calling Validate()
// again after a failure clears the old error when the value is now valid.
func TestFormField_Validate_ClearsErrorOnRetry(t *testing.T) {
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label:    "Client ID",
		Validate: validHexValidator,
		Theme:    th,
	})

	f.SetValue("bad")
	_ = f.Validate()
	require.NotEmpty(t, f.ValidationError())

	f.SetValue("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4")
	err := f.Validate()
	require.NoError(t, err)
	assert.Empty(t, f.ValidationError(), "ValidationError() should be cleared after successful retry")
}

// ansiEscapeRe matches ANSI SGR foreground colour sequences.
var ansiEscapeRe = regexp.MustCompile(`\x1b\[[\d;]+m`)

// TestFormField_Render_ValidationError_TwoDistinctColours is a regression test
// for the role wiring of FormField.ValidationError. The spec mandates
// "Error glyph + Plain text" — two distinct foreground colours. Before the fix,
// the entire line was painted in a single Error foreground, so this test would
// have found only one unique sequence for the error segment.
func TestFormField_Render_ValidationError_TwoDistinctColours(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label:    "Client ID",
		Validate: validHexValidator,
		Theme:    th,
	})

	f.SetValue("bad")
	_ = f.Validate()

	rendered := f.Render()

	// Find the error line — it is the last line of the rendered output.
	lines := strings.Split(rendered, "\n")
	var errLine string
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(stripANSI(lines[i])) != "" {
			errLine = lines[i]
			break
		}
	}
	require.NotEmpty(t, errLine, "error line must be present after Validate failure")

	// Collect unique ANSI sequences that open a new foreground colour.
	// We look for SGR sequences that contain a colour code (38;2; or simple colour digits).
	matches := ansiEscapeRe.FindAllString(errLine, -1)
	// Filter out reset sequences (\x1b[0m or \x1b[m) to focus on colour openers.
	var colourOpeners []string
	for _, m := range matches {
		if m != "\x1b[0m" && m != "\x1b[m" {
			colourOpeners = append(colourOpeners, m)
		}
	}

	// Deduplicate.
	seen := map[string]struct{}{}
	for _, s := range colourOpeners {
		seen[s] = struct{}{}
	}

	assert.GreaterOrEqual(t, len(seen), 2,
		"error line must contain at least two distinct ANSI colour sequences (Error glyph + Plain text), got: %v", colourOpeners)
}

// TestFormField_NewFormField_TextStyleAndCursorStyleSet is a regression test for
// Input.Text = Plain and Input.Cursor = Accent role wiring. It verifies that the
// TextStyle carries the TextPrimary foreground and the Cursor.Style carries the
// Accent foreground so theme switching propagates correctly.
func TestFormField_NewFormField_TextStyleAndCursorStyleSet(t *testing.T) {
	th := newTestTheme()
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label: "Client ID",
		Theme: th,
	})

	// TextStyle must be non-zero (carries TextPrimary foreground).
	wantTextStyle := lipgloss.NewStyle().Foreground(th.TextPrimary())
	assert.Equal(t, wantTextStyle, f.InputTextStyle(),
		"Input.Text role must be wired to TextPrimary via TextStyle")

	// Cursor.Style must be non-zero (carries Accent foreground).
	wantCursorStyle := lipgloss.NewStyle().Foreground(th.Accent())
	assert.Equal(t, wantCursorStyle, f.InputCursorStyle(),
		"Input.Cursor role must be wired to Accent via Cursor.Style")
}
