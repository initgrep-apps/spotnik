package layout

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// superscripts maps toggle key numbers 1-8 to their Unicode superscript equivalents.
var superscripts = map[int]string{
	1: "¹", 2: "²", 3: "³", 4: "⁴",
	5: "⁵", 6: "⁶", 7: "⁷", 8: "⁸",
}

// BorderConfig holds all data needed to render a btop-style pane border.
type BorderConfig struct {
	// Width is the total border width in terminal columns (includes the 2 border columns).
	Width int
	// Height is the total border height in terminal rows (includes the 2 border rows).
	Height int
	// Title is the pane title shown in the top border (e.g., "Playlists").
	Title string
	// ToggleKey is the number key (1-8) shown as a superscript before the title.
	// Pass 0 for panes that have no toggle key (e.g. Page B panes).
	ToggleKey int
	// Actions are pane-specific shortcuts shown in the top-right of the border.
	// Displayed as: ᐅkey label, separated by ─── .
	Actions []Action
	// AccentColor is the per-pane border accent color (from Theme.PaneBorder*()).
	AccentColor lipgloss.Color
	// Focused controls whether the pane has keyboard focus.
	// Focused: full AccentColor + Bold title; unfocused: AccentColor + Faint (dimmed but colored).
	Focused bool
	// FilterQuery is non-empty when filter mode is active.
	// When set, replaces the action shortcuts with: filtering: "query" ─── ᐅEsc close
	FilterQuery string
	// Theme provides KeyHint() and TextMuted() colors for action rendering.
	Theme theme.Theme
}

// RenderPaneBorder wraps content in a btop-style border.
//
// The top border line contains the toggle key superscript, title, dash fill,
// and action shortcuts (or filter query when active). Border characters always
// use the pane's AccentColor: focused = full brightness + bold title; unfocused
// = dimmed (Faint on top of AccentColor) so each pane retains its identity color.
//
// Content should be pre-sized to Width-2 × Height-2 (the interior dimensions).
// Lines are padded or truncated to fit exactly inside the border.
func RenderPaneBorder(content string, cfg BorderConfig) string {
	if cfg.Width < 2 {
		cfg.Width = 2
	}
	if cfg.Height < 2 {
		cfg.Height = 2
	}

	// Build styled helper: always applies the accent Foreground color; unfocused
	// adds Faint(true) on top so the color is dimmed but still visible as the
	// pane's identity color (not flat grey).
	borderStyle := func(s string) string {
		style := lipgloss.NewStyle().Foreground(cfg.AccentColor)
		if !cfg.Focused {
			style = style.Faint(true)
		}
		return style.Render(s)
	}

	keyHintStyle := func(s string) string {
		if cfg.Theme != nil {
			return lipgloss.NewStyle().Foreground(cfg.Theme.KeyHint()).Render(s)
		}
		return s
	}

	mutedStyle := func(s string) string {
		if cfg.Theme != nil {
			return lipgloss.NewStyle().Foreground(cfg.Theme.TextMuted()).Render(s)
		}
		return s
	}

	// titleStyle: focused renders with AccentColor + Bold; unfocused renders with
	// AccentColor + Faint so each pane title retains its identity color when not focused.
	titleStyle := func(s string) string {
		style := lipgloss.NewStyle().Foreground(cfg.AccentColor)
		if cfg.Focused {
			style = style.Bold(true)
		} else {
			style = style.Faint(true)
		}
		return style.Render(s)
	}

	// ── Build top border ─────────────────────────────────────────────────────

	// Fixed prefix: ╭─ (corner + dash + space = 3 columns)
	const (
		cornerTL = "╭"
		cornerTR = "╮"
		cornerBL = "╰"
		cornerBR = "╯"
		hBar     = "─"
		vBar     = "│"
	)

	// Build the right-side segment (actions or filter).
	rightSegment := buildRightSegment(cfg, keyHintStyle, mutedStyle)

	// Build the left-side prefix (toggle key + title).
	leftInner := buildLeftInner(cfg, keyHintStyle, titleStyle)

	// Compute available space for the dash fill.
	// Total inner width = Width - 2 (subtracting ╭ and ╮).
	// Left prefix: "─ " (2) + leftInner content.
	// Right suffix: rightSegment + " " (1).
	//
	// We use lipgloss.Width() for accurate terminal-column counting.
	leftPrefix := "─ "
	rightSuffix := " "

	// Total non-dash columns in the top border:
	// 1 (╭) + len(leftPrefix) + w(leftInner) + w(dashes) + w(rightSegment) + len(rightSuffix) + 1 (╮) = Width
	// => w(dashes) = Width - 2 - len(leftPrefix) - w(leftInner) - w(rightSegment) - len(rightSuffix)

	outerWidth := cfg.Width // includes ╭ and ╮
	fixedWidth := 1 +       // ╭
		lipgloss.Width(leftPrefix) + // "─ " = 2
		lipgloss.Width(leftInner) + // superscript + title (variable)
		lipgloss.Width(rightSegment) + // actions or filter
		lipgloss.Width(rightSuffix) + // " " = 1
		1 // ╮

	dashCount := outerWidth - fixedWidth
	if dashCount < 0 {
		// Border too narrow to fit title + actions — drop actions and retry.
		rightSegment = ""
		fixedWidth = 1 +
			lipgloss.Width(leftPrefix) +
			lipgloss.Width(leftInner) +
			lipgloss.Width(rightSuffix) +
			1
		dashCount = outerWidth - fixedWidth
	}
	if dashCount < 0 {
		// Still too narrow — truncate title with ellipsis.
		leftInner = truncateToColumns(leftInner, outerWidth-1-lipgloss.Width(leftPrefix)-lipgloss.Width(rightSuffix)-1-1)
		fixedWidth = 1 +
			lipgloss.Width(leftPrefix) +
			lipgloss.Width(leftInner) +
			lipgloss.Width(rightSuffix) +
			1
		dashCount = outerWidth - fixedWidth
	}
	if dashCount < 0 {
		dashCount = 0
	}

	dashes := strings.Repeat(hBar, dashCount)

	topBorder := borderStyle(cornerTL) +
		borderStyle(leftPrefix) +
		leftInner +
		borderStyle(dashes) +
		rightSegment +
		borderStyle(rightSuffix) +
		borderStyle(cornerTR)

	// ── Build content lines ───────────────────────────────────────────────────

	contentWidth := cfg.Width - 2 // interior columns
	contentHeight := cfg.Height - 2
	if contentWidth < 0 {
		contentWidth = 0
	}
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Split provided content into lines, pad/truncate each to contentWidth.
	rawLines := strings.Split(content, "\n")
	contentLines := make([]string, contentHeight)
	for i := range contentLines {
		var line string
		if i < len(rawLines) {
			line = rawLines[i]
		}
		contentLines[i] = padOrTruncate(line, contentWidth)
	}

	// ── Build bottom border ───────────────────────────────────────────────────
	bottomBorder := borderStyle(cornerBL) + borderStyle(strings.Repeat(hBar, cfg.Width-2)) + borderStyle(cornerBR)

	// ── Assemble all rows ─────────────────────────────────────────────────────
	rows := make([]string, 0, cfg.Height)
	rows = append(rows, topBorder)
	for _, cl := range contentLines {
		rows = append(rows, borderStyle(vBar)+cl+borderStyle(vBar))
	}
	rows = append(rows, bottomBorder)

	return strings.Join(rows, "\n")
}

// buildRightSegment builds the right-side content of the top border:
// either filter mode text or action shortcuts.
// Returns an empty string if there are no actions and no filter.
//
// Filter mode is unchanged: "filtering: "query" ─── ᐅEsc close".
// Action mode uses the corner-notch format: each action is rendered as
// "╮ key label ╭" with a single "─" between consecutive notches.
// The ╮ and ╭ characters use the pane's accent color (faint when unfocused)
// so they visually blend into the border dashes as notch cutouts.
func buildRightSegment(cfg BorderConfig, keyHintStyle, mutedStyle func(string) string) string {
	if cfg.FilterQuery != "" {
		// Filter mode: "filtering: "query" ─── ᐅEsc close"
		filtering := mutedStyle(`filtering: "` + cfg.FilterQuery + `"`)
		sep := mutedStyle(" ─── ")
		escAction := keyHintStyle("Esc") + " " + mutedStyle("close")
		prefix := mutedStyle("ᐅ")
		return filtering + sep + prefix + escAction
	}
	if len(cfg.Actions) == 0 {
		return ""
	}

	// borderChar renders a single character in the pane accent color (faint if unfocused).
	// Used for the ╮ and ╭ notch characters so they blend into the border line.
	borderChar := func(s string) string {
		style := lipgloss.NewStyle().Foreground(cfg.AccentColor)
		if !cfg.Focused {
			style = style.Faint(true)
		}
		return style.Render(s)
	}

	// Corner-notch format: ╮ key label ╭ with ─ between consecutive notches.
	parts := make([]string, len(cfg.Actions))
	for i, a := range cfg.Actions {
		parts[i] = borderChar("╮") + " " +
			keyHintStyle(a.Key) + " " + mutedStyle(a.Label) + " " +
			borderChar("╭")
	}
	return strings.Join(parts, borderChar("─"))
}

// buildLeftInner builds the inner-left content of the top border:
// superscript toggle key (if ToggleKey > 0) + title.
// Colors are applied via the provided style functions.
func buildLeftInner(cfg BorderConfig, keyHintStyle, titleStyle func(string) string) string {
	var b strings.Builder
	if sup, ok := superscripts[cfg.ToggleKey]; ok {
		b.WriteString(keyHintStyle(sup))
	}
	b.WriteString(titleStyle(cfg.Title))
	return b.String()
}

// padOrTruncate ensures s is exactly width terminal columns.
// If s is shorter, it is right-padded with spaces.
// If s is longer, it is truncated and an ellipsis (…) is appended.
func padOrTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w == width {
		return s
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	// Truncate: reduce rune-by-rune until we fit, then append …
	runes := []rune(s)
	for len(runes) > 0 {
		candidate := string(runes) + "…"
		if lipgloss.Width(candidate) <= width {
			return candidate + strings.Repeat(" ", width-lipgloss.Width(candidate))
		}
		runes = runes[:len(runes)-1]
	}
	return strings.Repeat(" ", width)
}

// truncateToColumns truncates s to at most maxCols terminal columns,
// appending … if truncation occurred.
func truncateToColumns(s string, maxCols int) string {
	if maxCols <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxCols {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		candidate := string(runes) + "…"
		if lipgloss.Width(candidate) <= maxCols {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}
	return ""
}

// PaneBorderColor returns the accent color for a given PaneID from the Theme.
// This maps PaneID constants to the corresponding PaneBorder*() Theme method.
// Falls back to Theme.ActiveBorder() for unknown PaneIDs.
func PaneBorderColor(id PaneID, t theme.Theme) lipgloss.Color {
	switch id {
	case PaneNowPlaying:
		return t.PaneBorderNowPlaying()
	case PaneQueue:
		return t.PaneBorderQueue()
	case PanePlaylists:
		return t.PaneBorderPlaylists()
	case PaneAlbums:
		return t.PaneBorderAlbums()
	case PaneLikedSongs:
		return t.PaneBorderLikedSongs()
	case PaneRecentlyPlayed:
		return t.PaneBorderRecentlyPlayed()
	case PaneTopTracks:
		return t.PaneBorderTopTracks()
	case PaneTopArtists:
		return t.PaneBorderTopArtists()
	case PaneRequestFlow:
		return t.PaneBorderRequestFlow()
	case PaneNetworkLog:
		return t.PaneBorderNetworkLog()
	default:
		return t.ActiveBorder()
	}
}
