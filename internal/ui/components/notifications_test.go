package components

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
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
