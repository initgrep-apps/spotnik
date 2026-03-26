// Package components — timeutil.go provides time-formatting utilities used
// across panes that display timestamps in human-readable relative form.
package components

import (
	"fmt"
	"time"
)

// FormatRelativeTime returns a human-readable relative timestamp:
//
//	< 1 min    → "just now"
//	1–59 min   → "{n} min ago"
//	1–23 hr    → "{n} hr ago"
//	1–6 days   → "{n} days ago"
//	>= 7 days  → "Jan 2" short date
func FormatRelativeTime(t time.Time) string {
	elapsed := time.Since(t)

	if elapsed < time.Minute {
		return "just now"
	}
	if elapsed < time.Hour {
		mins := int(elapsed.Minutes())
		return fmt.Sprintf("%d min ago", mins)
	}
	if elapsed < 24*time.Hour {
		hours := int(elapsed.Hours())
		return fmt.Sprintf("%d hr ago", hours)
	}
	days := int(elapsed.Hours() / 24)
	if days < 7 {
		return fmt.Sprintf("%d days ago", days)
	}
	return t.Format("Jan 2")
}
