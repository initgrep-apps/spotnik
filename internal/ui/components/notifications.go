package components

import (
	"time"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"go.dalton.dog/bubbleup"
)

// notificationDuration is how long each toast notification stays visible.
const notificationDuration = 4 * time.Second

// NewNotifications creates a BubbleUp AlertModel configured with Spotnik theme
// colors and glyph prefixes resolved via uikit.RegisterBubbleupAlerts. Glyph
// prefixes honour uikit.ActiveMode() so ASCII mode produces ASCII prefixes.
//
// Toast notifications are positioned at the bottom-right per DESIGN.md §12.
//
// IMPORTANT: Always call Render(content) in View() — never View(). BubbleUp's
// View() returns an empty string by design; Render() overlays the alert on top
// of the full rendered view string.
func NewNotifications(t theme.Theme) *bubbleup.AlertModel {
	defs := uikit.RegisterBubbleupAlerts(t)

	// Width 60 with useNerdFont=false ensures compatibility with standard terminal fonts.
	model := bubbleup.NewAlertModel(60, false, notificationDuration)
	for _, def := range defs {
		model.RegisterNewAlertType(def)
	}

	// Reposition toasts to bottom-right per DESIGN.md §12.
	// WithPosition returns an immutable copy with the new position.
	positioned := model.WithPosition(bubbleup.BottomRightPosition)
	return &positioned
}
