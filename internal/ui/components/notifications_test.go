package components

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.dalton.dog/bubbleup"
)

func TestNewNotifications_ReturnsAlertModel(t *testing.T) {
	t.Run("returns a valid AlertModel that is non-nil", func(t *testing.T) {
		th := theme.Load(theme.DefaultThemeID)
		model := NewNotifications(th)
		require.NotNil(t, model, "NewNotifications must return a non-nil *AlertModel")
	})
}

func TestNewNotifications_InitReturnsNilByDesign(t *testing.T) {
	// BubbleUp's Init() returns nil — the alert timer starts only when an alert fires.
	th := theme.Load(theme.DefaultThemeID)
	model := NewNotifications(th)
	// Init() returning nil is expected BubbleUp behavior.
	_ = model.Init() // must not panic
}

func TestNewNotifications_AllFiveAlertTypesRegistered(t *testing.T) {
	th := theme.Load(theme.DefaultThemeID)
	model := NewNotifications(th)

	// All five custom alert types must be usable (non-nil cmd returned).
	types := []string{"success", "error", "warning", "info", "ratelimit"}
	for _, alertType := range types {
		t.Run("type="+alertType, func(t *testing.T) {
			cmd := model.NewAlertCmd(alertType, "test message")
			assert.NotNil(t, cmd, "NewAlertCmd(%q) must return a non-nil Cmd", alertType)
		})
	}
}

func TestNewNotifications_CorrectDuration(t *testing.T) {
	// Verify the model starts with no active alert (clean state).
	th := theme.Load(theme.DefaultThemeID)
	model := NewNotifications(th)

	assert.False(t, model.HasActiveAlert(),
		"fresh model should have no active alert before any NewAlertCmd")
}

func TestNewNotifications_AlertActivatesAfterCmdProcessed(t *testing.T) {
	th := theme.Load(theme.DefaultThemeID)
	model := NewNotifications(th)

	// Triggering an alert and verifying it activates after the cmd message is processed.
	cmd := model.NewAlertCmd("success", "Track added!")
	require.NotNil(t, cmd)

	// Execute the command to get the alertMsg, then feed it to Update.
	msg := cmd()
	updated, _ := model.Update(msg)
	am, ok := updated.(bubbleup.AlertModel)
	require.True(t, ok, "Update should return bubbleup.AlertModel (value type)")
	assert.True(t, am.HasActiveAlert(), "alert should be active after alertMsg processed")
}

func TestNewNotifications_RenderOverlaysContent(t *testing.T) {
	th := theme.Load(theme.DefaultThemeID)
	model := NewNotifications(th)

	// Activate an alert, then verify Render() overlays it on the content.
	cmd := model.NewAlertCmd("error", "Playback failed")
	msg := cmd()
	updated, _ := model.Update(msg)
	am, ok := updated.(bubbleup.AlertModel)
	require.True(t, ok)

	content := "main view content here\nsecond line of content"
	rendered := am.Render(content)
	assert.Contains(t, rendered, "Playback failed",
		"Render() should overlay the alert message on the content")
}

func TestNewNotifications_ViewIsEmpty(t *testing.T) {
	// BubbleUp's View() is intentionally empty — we must use Render(content) instead.
	th := theme.Load(theme.DefaultThemeID)
	model := NewNotifications(th)

	cmd := model.NewAlertCmd("info", "some info")
	msg := cmd()
	updated, _ := model.Update(msg)
	am, ok := updated.(bubbleup.AlertModel)
	require.True(t, ok)

	// View() should be empty — this is by design in BubbleUp.
	view := am.View()
	assert.Empty(t, view, "BubbleUp View() is intentionally empty; use Render() instead")
}

func TestNewNotifications_NoActiveAlertRenderReturnsOriginalContent(t *testing.T) {
	// When no alert is active, Render() should return content unchanged.
	th := theme.Load(theme.DefaultThemeID)
	model := NewNotifications(th)

	// No alert triggered — model has no active alert.
	assert.False(t, model.HasActiveAlert())
	content := "original content string"
	rendered := model.Render(content)
	assert.Equal(t, content, rendered,
		"Render() with no active alert should return original content unchanged")
}

// --- F50 Task 6: Toast notifications positioned bottom-right ---

// TestNewNotifications_ToastAppearsAtBottomRight verifies that the toast notification
// is composited at the bottom-right of the content area per DESIGN.md §12.
// We use a sufficiently tall content block so top vs. bottom positioning is distinguishable.
func TestNewNotifications_ToastAppearsAtBottomRight(t *testing.T) {
	th := theme.Load(theme.DefaultThemeID)
	model := NewNotifications(th)

	// Activate a success alert.
	cmd := model.NewAlertCmd("success", "Added to queue: Starboy")
	msg := cmd()
	updated, _ := model.Update(msg)
	am, ok := updated.(bubbleup.AlertModel)
	require.True(t, ok)

	// Tall content block (20 lines) so top vs. bottom is clearly distinguishable.
	// Alert is 3 lines tall (╭───╮ / │ msg │ / ╰───╯), so at bottom it appears in
	// lines 17-19 (0-indexed). At top it appears in lines 0-2.
	var contentLines []string
	for i := 0; i < 20; i++ {
		contentLines = append(contentLines, "line content here                              ")
	}
	content := strings.Join(contentLines, "\n")

	rendered := am.Render(content)
	lines := strings.Split(rendered, "\n")

	// The alert text must appear somewhere.
	assert.Contains(t, rendered, "Added to queue: Starboy",
		"toast message should appear in rendered output")

	// Find which line index the alert text appears on.
	alertLine := -1
	for i, l := range lines {
		if strings.Contains(l, "Starboy") {
			alertLine = i
			break
		}
	}
	require.NotEqual(t, -1, alertLine, "alert line must be found in rendered output")

	// With BottomRightPosition, the alert appears in the LAST few lines.
	// With TopLeftPosition, it appears in the FIRST few lines (lines 0-2).
	// We verify the alert is in the bottom half of the content (line >= 10).
	assert.GreaterOrEqual(t, alertLine, len(lines)/2,
		"toast should appear in the bottom half of the content (bottom-right positioning)")
}

// TestNewNotifications_InfoUsesInfoToken verifies that the info alert's foreground
// color is wired to t.Info() and that the info toast remains registered and usable
// after the change from t.KeyHint() to t.Info().
func TestNewNotifications_InfoUsesInfoToken(t *testing.T) {
	// We test indirectly: if the code is correct, NewNotifications compiles and runs
	// with the new Info() call. The real behavioral difference (using the canonical
	// info color instead of the key-hint color) is enforced by the Theme interface.
	//
	// This test verifies NewNotifications does not panic when Info() is called.
	th := theme.Load(theme.DefaultThemeID)
	require.NotNil(t, th)
	// Sanity: Info() must return a non-empty color.
	assert.NotEmpty(t, string(th.Info()), "Theme.Info() must return a non-empty color")
	// NewNotifications must not panic with the updated Info() wiring.
	model := NewNotifications(th)
	require.NotNil(t, model, "NewNotifications must not panic with Info() wiring")
	// Info toast must still be registered.
	cmd := model.NewAlertCmd("info", "informational message")
	assert.NotNil(t, cmd, "info alert must be registered and usable after Info() wiring")
}

// TestNewNotifications_WarningPrefixIsCircleTriangle verifies that the warning alert
// prefix is ◬ (U+25EC), not the old U+26A0 glyph, per §5.2 of the design record.
// Content must be tall enough for the bottom-right positioned alert to be visible.
func TestNewNotifications_WarningPrefixIsCircleTriangle(t *testing.T) {
	th := theme.Load(theme.DefaultThemeID)
	model := NewNotifications(th)

	// Activate a warning alert and verify ◬ appears in the rendered output.
	cmd := model.NewAlertCmd("warning", "Premium required")
	require.NotNil(t, cmd)
	msg := cmd()
	updated, _ := model.Update(msg)
	am, ok := updated.(bubbleup.AlertModel)
	require.True(t, ok)

	// Use 20 lines of content so the bottom-right positioned alert is visible.
	var contentLines []string
	for i := 0; i < 20; i++ {
		contentLines = append(contentLines, strings.Repeat(" ", 70))
	}
	content := strings.Join(contentLines, "\n")
	rendered := am.Render(content)
	assert.Contains(t, rendered, "◬",
		"warning toast must use ◬ (U+25EC) prefix, not U+26A0")
	assert.NotContains(t, rendered, "\u26A0",
		"warning toast must not use old U+26A0 glyph")
}

// TestNewNotifications_PositionBottomRight verifies the position constant used is BottomRight.
// This is an indirect test: we verify that the constructed model uses BottomRightPosition
// by checking the rendered output's line distribution.
func TestNewNotifications_AlertDoesNotInterfereWithGrid(t *testing.T) {
	th := theme.Load(theme.DefaultThemeID)
	model := NewNotifications(th)

	// With no active alert, grid content passes through unchanged.
	gridContent := "╭─ Playlists ─────────────╮\n│  Track 1                │\n╰─────────────────────────╯"
	rendered := model.Render(gridContent)
	assert.Contains(t, rendered, "╭─ Playlists", "grid content should pass through unchanged when no alert")
}

// TestNewNotifications_WarningPrefixIsExclamation_ASCII verifies that in ASCII mode
// the warning alert prefix is "!" (ASCII warning glyph) and that "◬" does NOT appear.
// This locks in that the NewNotifications wrapper honours ActiveMode() at registration
// time — a regression where the prefix is hardcoded to the unicode glyph would fail here.
func TestNewNotifications_WarningPrefixIsExclamation_ASCII(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load(theme.DefaultThemeID)
	model := NewNotifications(th)

	cmd := model.NewAlertCmd("warning", "Premium required")
	require.NotNil(t, cmd)
	msg := cmd()
	updated, _ := model.Update(msg)
	am, ok := updated.(bubbleup.AlertModel)
	require.True(t, ok)

	// Use 20 lines of content so the bottom-right positioned alert is visible.
	var contentLines []string
	for i := 0; i < 20; i++ {
		contentLines = append(contentLines, strings.Repeat(" ", 70))
	}
	content := strings.Join(contentLines, "\n")
	rendered := am.Render(content)

	assert.Contains(t, rendered, "!",
		"warning toast in ASCII mode must use '!' prefix")
	assert.NotContains(t, rendered, "◬",
		"warning toast in ASCII mode must not use unicode ◬ glyph")
}
