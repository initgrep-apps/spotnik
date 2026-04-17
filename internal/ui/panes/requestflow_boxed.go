package panes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
)

// renderSubBox renders a small bordered box with a title label.
// lines are the content lines (already styled). The box is sized to exactly
// width columns × (len(lines) + 2) rows (content + top/bottom border).
// Inner content is padded with one space on each side.
// If width < 8, returns empty string (too narrow for a meaningful box).
// NOTE: viewBoxed() guarantees width >= 10 for all boxes via minimum clamps
// and falls back to viewFlat() if totals exceed pane width, so the empty-string
// path is a safety net rather than a normal render path.
func (p *RequestFlowPane) renderSubBox(title string, lines []string, width int, borderColor lipgloss.Color) string {
	if width < 8 {
		return ""
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	borderChar := borderStyle.Render("│")

	innerW := width - 2 // subtract left/right border chars

	// Build top border: ╭─ TITLE ──────╮
	titleStyled := lipgloss.NewStyle().Foreground(borderColor).Bold(true).Render(title)
	titleVisible := lipgloss.Width(titleStyled)

	// Fill pattern: "─ <title> " then pad to innerW
	prefixPlain := "─ "
	suffixPlain := " "
	prefixWidth := 2 + titleVisible + 1 // "─ " + title + " "
	remaining := innerW - prefixWidth
	if remaining < 0 {
		remaining = 0
	}
	topBorder := borderStyle.Render("╭"+prefixPlain) +
		titleStyled +
		borderStyle.Render(suffixPlain+strings.Repeat("─", remaining)+"╮")

	// Build bottom border: ╰──────────╯
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", innerW) + "╯")

	if len(lines) == 0 {
		return topBorder + "\n" + bottomBorder
	}

	var sb strings.Builder
	sb.WriteString(topBorder)
	sb.WriteString("\n")

	for _, line := range lines {
		// Pad/truncate each line to innerW - 2 (1 space padding each side).
		cell := layout.TruncateOrPad(line, innerW-2)
		sb.WriteString(borderChar)
		sb.WriteString(" ")
		sb.WriteString(cell)
		sb.WriteString(" ")
		sb.WriteString(borderChar)
		sb.WriteString("\n")
	}

	sb.WriteString(bottomBorder)
	return sb.String()
}

// buildAppBoxLines returns styled content lines for the APP sub-box.
// Lines show endpoint paths for active requests (newest first), up to maxRows.
// Padded with empty strings if fewer requests exist.
func (p *RequestFlowPane) buildAppBoxLines(maxRows int) []string {
	if maxRows <= 0 {
		return nil
	}
	anims := p.sortedAnimations()
	lines := make([]string, 0, maxRows)
	for i, a := range anims {
		if i >= maxRows {
			break
		}
		ep := truncateStr(a.method+" "+a.path, 200)
		var style lipgloss.Style
		if a.phase >= phaseCompleted {
			style = lipgloss.NewStyle().Foreground(p.theme.TextMuted())
		} else if a.priority == domain.PriorityInteractive {
			style = lipgloss.NewStyle().Foreground(p.theme.TextPrimary())
		} else {
			style = lipgloss.NewStyle().Foreground(p.theme.TextMuted())
		}
		lines = append(lines, strings.TrimRight(style.Render(ep), " "))
	}
	// Pad with empty lines to fill maxRows.
	for len(lines) < maxRows {
		lines = append(lines, "")
	}
	return lines
}

// buildGatewayBoxLines returns styled content lines for the GATEWAY LOG sub-box.
// Pure event stream — no state metric bars. State is shown in the GATEWAY banner.
// Events are newest-first (most recent at top). Padded to maxRows with empty strings.
func (p *RequestFlowPane) buildGatewayBoxLines(maxRows int) []string {
	if maxRows <= 0 {
		return nil
	}

	successStyle := lipgloss.NewStyle().Foreground(p.theme.Success())
	errorStyle := lipgloss.NewStyle().Foreground(p.theme.Error())
	warnStyle := lipgloss.NewStyle().Foreground(p.theme.Warning())
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	primaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary())

	lines := make([]string, 0, maxRows)
	for i := len(p.displayState.decisions) - 1; i >= 0 && len(lines) < maxRows; i-- {
		d := p.displayState.decisions[i]
		var style lipgloss.Style
		switch d.kind {
		case domain.EventRequestEntered:
			if d.priority == domain.PriorityInteractive {
				style = primaryStyle
			} else {
				style = mutedStyle
			}
		case domain.EventRequestAllowed, domain.EventBackoffExpired,
			domain.EventDedupResolved:
			style = successStyle
		case domain.EventHttpCompleted:
			switch {
			case d.statusCode >= 200 && d.statusCode < 300:
				style = successStyle
			case d.statusCode == 429:
				style = warnStyle
			case d.statusCode >= 500:
				style = errorStyle
			default:
				style = secondaryStyle
			}
		case domain.EventRequestBlocked, domain.EventBackoffStarted:
			style = errorStyle
		case domain.EventDedupJoined:
			style = warnStyle
		case domain.EventTokenConsumed, domain.EventSemaphoreAcquired,
			domain.EventSemaphoreReleased:
			style = secondaryStyle
		case domain.EventTokenRefilled:
			style = mutedStyle
		default:
			style = mutedStyle
		}
		lines = append(lines, style.Render(d.label))
	}

	for len(lines) < maxRows {
		lines = append(lines, "")
	}
	return lines
}

// buildSpotifyBoxLines returns styled content lines for the SPOTIFY sub-box.
// Format per row: [status]  [method] [path]  [latency]
// Only requests that reached Spotify are included — blocked and dedup-joined
// requests are omitted. No padding: the box height reflects actual HTTP traffic.
func (p *RequestFlowPane) buildSpotifyBoxLines(maxRows int) []string {
	if maxRows <= 0 {
		return nil
	}
	anims := p.sortedAnimations()
	lines := make([]string, 0, len(anims))

	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	for _, a := range anims {
		if len(lines) >= maxRows {
			break
		}
		// Skip requests that never reached Spotify.
		if a.decision == domain.EventDedupJoined || a.decision == domain.EventRequestBlocked {
			continue
		}

		// Skip requests that haven't yet reached the HTTP call phase.
		if a.phase < phaseInFlight {
			continue
		}

		path := stripAPIPrefix(a.path)
		methodStr := secondaryStyle.Render(a.method)

		if a.statusCode == 0 {
			// In-flight — HTTP call in progress, no response yet.
			placeholder := mutedStyle.Render("···")
			lines = append(lines, fmt.Sprintf("%s  %s %s  %s",
				placeholder, methodStr, mutedStyle.Render(path), placeholder))
			continue
		}

		var statusStyle lipgloss.Style
		switch {
		case a.statusCode >= 200 && a.statusCode < 300:
			statusStyle = lipgloss.NewStyle().Foreground(p.theme.Success())
		case a.statusCode == 429:
			statusStyle = lipgloss.NewStyle().Foreground(p.theme.Warning())
		case a.statusCode >= 500:
			statusStyle = lipgloss.NewStyle().Foreground(p.theme.Error())
		default:
			statusStyle = lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
		}

		statusStr := statusStyle.Render(fmt.Sprintf("%d", a.statusCode))
		pathStr := statusStyle.Render(path)
		latStr := secondaryStyle.Render(fmt.Sprintf("%dms", a.durationMs))
		lines = append(lines, fmt.Sprintf("%s  %s %s  %s", statusStr, methodStr, pathStr, latStr))
	}
	return lines
}
