package uikit

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// EmptyStatus classifies why a pane has no data to display.
type EmptyStatus int

const (
	// EmptyStatusNone means data is genuinely empty (API returned 200, zero items).
	EmptyStatusNone EmptyStatus = iota
	// EmptyStatusNeverFetched means no API call has ever completed for this data.
	EmptyStatusNeverFetched
	// EmptyStatusFetching means the first API call is in-flight.
	EmptyStatusFetching
	// EmptyStatusError means the last API call failed with a non-rate-limit error.
	EmptyStatusError
	// EmptyStatusRateLimited means the gateway is in 429 backoff.
	EmptyStatusRateLimited
)

// EmptyState is shown when a pane has nothing to display. Text is centered
// vertically and horizontally in the provided rectangle; an optional Hint
// renders below Text. Both Text and Hint are rendered in the Muted role.
//
// When Status is set (non-zero), Render() derives the display text from the
// status instead of using Text/Hint directly. Text is still used as the
// category source (e.g. "No followed shows" → category "followed shows").
//
// Render() returns exactly Height newline-separated lines.
type EmptyState struct {
	// Text is the primary no-data message (e.g. "Empty queue").
	// When Status is set, Text is used to derive the category name by
	// stripping the "No " prefix.
	Text string
	// Hint is the optional secondary help text rendered below Text.
	// When Status is EmptyStatusRateLimited, the caller should set Hint
	// with the retry-after info (e.g. "Rate limited — retrying in 5s").
	Hint string
	// Status, when set (non-zero), overrides Text/Hint with status-driven
	// messages. EmptyStatusNone uses Text/Hint as-is.
	Status EmptyStatus
	// Width is the column width of the rendered output.
	Width int
	// Height is the number of lines in the rendered output.
	Height int
	// Theme provides colour tokens.
	Theme theme.Theme
}

// Render centers Text (and Hint below it) both horizontally and vertically
// within the Height×Width rectangle. Both are styled in the Muted role.
// When Status is set, derives display text from the status.
// Returns exactly Height newline-joined lines.
func (e EmptyState) Render() string {
	if e.Height <= 0 {
		return ""
	}

	text := e.Text
	hint := e.Hint

	switch e.Status {
	case EmptyStatusNeverFetched, EmptyStatusFetching:
		category := strings.TrimPrefix(e.Text, "No ")
		text = "Loading " + category + "..."
		hint = ""
	case EmptyStatusError:
		category := strings.TrimPrefix(e.Text, "No ")
		text = "Unable to load " + category
		if hint == "" {
			hint = "Check your connection"
		}
	case EmptyStatusRateLimited:
		category := strings.TrimPrefix(e.Text, "No ")
		text = "Unable to load " + category
		// hint is set by caller with retry-after info
	}

	mutedStyle := lipgloss.NewStyle().Foreground(e.Theme.TextMuted())

	// Build the body lines (text + optional hint).
	body := mutedStyle.Render(text)
	if hint != "" {
		body = body + "\n" + mutedStyle.Render(hint)
	}

	bodyLines := strings.Split(body, "\n")
	bodyHeight := len(bodyLines)

	// Calculate top padding for vertical centering.
	// Clamp to zero so a body larger than Height still renders without blank top lines.
	topPad := max(0, (e.Height-bodyHeight)/2)

	lines := make([]string, 0, e.Height)

	// Pad above.
	blankLine := strings.Repeat(" ", e.Width)
	for i := 0; i < topPad; i++ {
		lines = append(lines, blankLine)
	}

	// Append body lines, each horizontally centered.
	// Clamp to Height so overflowing body lines do not exceed the rectangle.
	for _, bl := range bodyLines {
		if len(lines) >= e.Height {
			break
		}
		lines = append(lines, centerLine(bl, e.Width))
	}

	// Pad below to reach exactly Height.
	for len(lines) < e.Height {
		lines = append(lines, blankLine)
	}

	return strings.Join(lines, "\n")
}

// PaneEmptyStatus determines the EmptyState for a pane based on store state.
// category is the display name (e.g. "followed shows", "saved episodes").
// hasData is true when len(store.Data()) > 0.
// isFetching is the store's *Fetching() accessor result.
// fetchErr is the store's *FetchErr() accessor result.
// neverFetched is true when fetchedAt is zero time.
// isThrottled is store.IsThrottled().
// retryAfterSecs is store.ThrottleRetryAfterSecs().
func PaneEmptyStatus(category string, hasData, isFetching bool, fetchErr error,
	neverFetched, isThrottled bool, retryAfterSecs int) EmptyState {
	if isThrottled {
		return EmptyState{
			Status: EmptyStatusRateLimited,
			Text:   "No " + category,
			Hint:   fmt.Sprintf("Rate limited — retrying in %ds", retryAfterSecs),
		}
	}
	if isFetching {
		return EmptyState{Status: EmptyStatusFetching, Text: "No " + category}
	}
	if neverFetched {
		return EmptyState{Status: EmptyStatusNeverFetched, Text: "No " + category}
	}
	if fetchErr != nil {
		return EmptyState{Status: EmptyStatusError, Text: "No " + category}
	}
	return EmptyState{Status: EmptyStatusNone, Text: "No " + category}
}

// centerLine pads a single rendered line with spaces so the visible content
// is horizontally centered in a column of width w. ANSI escape codes are
// not counted toward the visible width (lipgloss.Width handles this).
func centerLine(s string, w int) string {
	cur := lipgloss.Width(s)
	if cur >= w {
		return s
	}
	left := (w - cur) / 2
	right := w - cur - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
