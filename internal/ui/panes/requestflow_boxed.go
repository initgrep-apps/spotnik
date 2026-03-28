package panes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
func (p *RequestFlowPane) renderSubBox(title string, lines []string, width int) string {
	if width < 8 {
		return ""
	}

	borderColor := p.theme.TextSecondary()
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	borderChar := borderStyle.Render("│")

	innerW := width - 2 // subtract left/right border chars

	// Build top border: ╭─ TITLE ──────╮
	titleStyled := lipgloss.NewStyle().Foreground(p.theme.TextSecondary()).Bold(true).Render(title)
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

// renderRightArrow renders the connecting arrow between GATEWAY and SPOTIFY columns.
// The arrow style reflects the HTTP response outcome:
//   - 2xx: animated flowing arrow (Success color)
//   - 429: "── ╳ ──" (Warning color)
//   - 5xx: animated arrow (Error color)
//   - 0:   "── ╳ ──" (TextMuted — blocked, no HTTP call made)
func (p *RequestFlowPane) renderRightArrow(r reqDisplay, colWidth int) string {
	frames := []string{"──→──", "───→─", "────→"}

	var arrow string
	var style lipgloss.Style

	switch {
	case r.statusCode == 0:
		arrow = "── ╳ ──"
		style = lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	case r.statusCode == 429:
		arrow = "── ╳ ──"
		style = lipgloss.NewStyle().Foreground(p.theme.Warning())
	case r.statusCode >= 500:
		arrow = frames[p.frameIndex%3]
		style = lipgloss.NewStyle().Foreground(p.theme.Error())
	case r.statusCode >= 200 && r.statusCode < 300:
		arrow = frames[p.frameIndex%3]
		style = lipgloss.NewStyle().Foreground(p.theme.Success())
	default:
		arrow = frames[p.frameIndex%3]
		style = lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	}

	return padRightVisible(style.Render(arrow), colWidth)
}

// gatewayStateLines returns the GATEWAY metrics as individual styled lines.
// This is used by both renderGatewayState() (flat layout) and
// buildGatewayBoxLines() (boxed layout) for consistent output.
func (p *RequestFlowPane) gatewayStateLines() []string {
	snap := p.lastSnapshot

	successStyle := lipgloss.NewStyle().Foreground(p.theme.Success())
	warnStyle := lipgloss.NewStyle().Foreground(p.theme.Warning())
	errorStyle := lipgloss.NewStyle().Foreground(p.theme.Error())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())

	// Token bucket bar: ● (Success) for available, ○ (muted) for consumed.
	tokenBar := p.renderColoredDotBar(snap.TokensAvailable, snap.TokensMax, '●', '○', successStyle, mutedStyle)
	tokenLine := fmt.Sprintf("tokens  %s %d/%d", tokenBar, snap.TokensAvailable, snap.TokensMax)
	// Show "(min: N)" annotation when the gateway observed a token dip this window.
	// Gateway tracks this internally at the moment of consumption (not by sampling).
	if snap.MinTokens < snap.TokensAvailable {
		tokenLine += mutedStyle.Render(fmt.Sprintf(" (min: %d)", snap.MinTokens))
	}

	// Semaphore bar: ■ (Warning) for in-use, □ (muted) for available.
	semBar := p.renderColoredDotBar(snap.ConcurrentActive, snap.ConcurrentMax, '■', '□', warnStyle, mutedStyle)
	semLine := fmt.Sprintf("conc    %s %d/%d", semBar, snap.ConcurrentActive, snap.ConcurrentMax)
	// Show "(peak: N)" annotation when the gateway observed a concurrency spike this window.
	if snap.PeakConcurrent > snap.ConcurrentActive {
		semLine += mutedStyle.Render(fmt.Sprintf(" (peak: %d)", snap.PeakConcurrent))
	}

	lines := []string{tokenLine, semLine}

	// Backoff timer: only show when store is throttled.
	if p.store != nil && p.store.IsThrottled() {
		remaining := snap.BackoffRemaining
		if remaining <= 0 {
			remaining = float64(p.store.ThrottleRetryAfterSecs())
		}
		lines = append(lines, errorStyle.Render(fmt.Sprintf("⏳ backoff %.1fs", remaining)))
	}

	// Dedup waiters: only show when active.
	if snap.DedupWaiters > 0 {
		lines = append(lines, secondaryStyle.Render(fmt.Sprintf("dedup  %d in-flight", snap.DedupWaiters)))
	}

	// InFlightKeys: render up to 3 with truncation.
	if len(snap.InFlightKeys) > 0 {
		const maxKeys = 3
		shown := len(snap.InFlightKeys)
		if shown > maxKeys {
			shown = maxKeys
		}
		for i := 0; i < shown; i++ {
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("  → %s", snap.InFlightKeys[i])))
		}
		if len(snap.InFlightKeys) > maxKeys {
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("  … +%d more", len(snap.InFlightKeys)-maxKeys)))
		}
	}

	return lines
}

// buildAppBoxLines returns styled content lines for the APP sub-box.
// Lines show endpoint paths for recent requests (newest first), up to maxRows.
// Padded with empty strings if fewer requests exist.
func (p *RequestFlowPane) buildAppBoxLines(maxRows int) []string {
	if maxRows <= 0 {
		return nil
	}
	lines := make([]string, 0, maxRows)
	for i, r := range p.recentReqs {
		if i >= maxRows {
			break
		}
		// Reuse renderAppEntry but strip the padding to get the raw styled text.
		// We pass a large colWidth so padding doesn't interfere; box renders it.
		lines = append(lines, strings.TrimRight(p.renderAppEntry(r, 200), " "))
	}
	// Pad with empty lines to fill maxRows.
	for len(lines) < maxRows {
		lines = append(lines, "")
	}
	return lines
}

// buildGatewayBoxLines returns styled content lines for the GATEWAY sub-box.
// Lines show gateway metrics (token bucket, semaphore, backoff, dedup, in-flight),
// up to maxRows. Padded with empty strings if fewer metric lines exist.
func (p *RequestFlowPane) buildGatewayBoxLines(maxRows int) []string {
	if maxRows <= 0 {
		return nil
	}
	raw := p.gatewayStateLines()
	lines := make([]string, 0, maxRows)
	for i, l := range raw {
		if i >= maxRows {
			break
		}
		lines = append(lines, l)
	}
	for len(lines) < maxRows {
		lines = append(lines, "")
	}
	return lines
}

// buildSpotifyBoxLines returns styled content lines for the SPOTIFY sub-box.
// Lines show HTTP status + latency for recent requests (newest first), up to maxRows.
// Padded with empty strings if fewer requests exist.
func (p *RequestFlowPane) buildSpotifyBoxLines(maxRows int) []string {
	if maxRows <= 0 {
		return nil
	}
	lines := make([]string, 0, maxRows)
	for i, r := range p.recentReqs {
		if i >= maxRows {
			break
		}
		lines = append(lines, p.renderSpotifyEntry(r))
	}
	for len(lines) < maxRows {
		lines = append(lines, "")
	}
	return lines
}

// buildLeftArrowLines builds arrow strings for APP→GATEWAY (one per row).
// Rows beyond request count are space-padded to colWidth.
func (p *RequestFlowPane) buildLeftArrowLines(maxRows, colWidth int) []string {
	if maxRows <= 0 {
		return nil
	}
	lines := make([]string, maxRows)
	for i := 0; i < maxRows; i++ {
		if i < len(p.recentReqs) {
			lines[i] = p.renderArrow(p.recentReqs[i], colWidth)
		} else {
			lines[i] = strings.Repeat(" ", colWidth)
		}
	}
	return lines
}

// buildRightArrowLines builds arrow strings for GATEWAY→SPOTIFY (one per row).
// Rows beyond request count are space-padded to colWidth.
func (p *RequestFlowPane) buildRightArrowLines(maxRows, colWidth int) []string {
	if maxRows <= 0 {
		return nil
	}
	lines := make([]string, maxRows)
	for i := 0; i < maxRows; i++ {
		if i < len(p.recentReqs) {
			lines[i] = p.renderRightArrow(p.recentReqs[i], colWidth)
		} else {
			lines[i] = strings.Repeat(" ", colWidth)
		}
	}
	return lines
}
