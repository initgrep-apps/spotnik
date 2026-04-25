package components

import (
	"time"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"go.dalton.dog/bubbleup"
)

// notificationDuration is how long each toast notification stays visible.
const notificationDuration = 4 * time.Second

// NewNotifications creates a BubbleUp AlertModel configured with Spotnik theme
// colors and custom alert types. All five severity levels are registered:
// "success", "error", "warning", "info", and "ratelimit".
//
// Toast notifications are positioned at the bottom-right per DESIGN.md §12.
//
// IMPORTANT: Always call Render(content) in View() — never View(). BubbleUp's
// View() returns an empty string by design; Render() overlays the alert on top
// of the full rendered view string.
func NewNotifications(t theme.Theme) *bubbleup.AlertModel {
	// Width 0 with minWidth causes BubbleUp to size alerts to content width.
	// useNerdFont=false ensures compatibility with standard terminal fonts.
	model := bubbleup.NewAlertModel(60, false, notificationDuration)

	successAlert := bubbleup.AlertDefinition{
		Key:       "success",
		ForeColor: string(t.Success()),
		Prefix:    "✓",
	}
	errorAlert := bubbleup.AlertDefinition{
		Key:       "error",
		ForeColor: string(t.Error()),
		Prefix:    "✗",
	}
	warningAlert := bubbleup.AlertDefinition{
		Key:       "warning",
		ForeColor: string(t.Warning()),
		Prefix:    "◬",
	}
	infoAlert := bubbleup.AlertDefinition{
		Key:       "info",
		ForeColor: string(t.Info()),
		Prefix:    "→",
	}
	rateLimitAlert := bubbleup.AlertDefinition{
		Key:       "ratelimit",
		ForeColor: string(t.Warning()),
		Prefix:    "⧖",
	}

	model.RegisterNewAlertType(successAlert)
	model.RegisterNewAlertType(errorAlert)
	model.RegisterNewAlertType(warningAlert)
	model.RegisterNewAlertType(infoAlert)
	model.RegisterNewAlertType(rateLimitAlert)

	// Reposition toasts to bottom-right per DESIGN.md §12.
	// WithPosition returns an immutable copy with the new position.
	positioned := model.WithPosition(bubbleup.BottomRightPosition)
	return &positioned
}
