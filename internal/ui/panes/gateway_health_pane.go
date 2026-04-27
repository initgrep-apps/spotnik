package panes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// NOTE: GatewayHealthPane.View() returns raw content without a border.
// The outer btop-style border is added by render.go via layout.RenderPaneBorder,
// which reads Title() and ToggleKey() directly from the pane interface.
// Do NOT wrap View() output in uikit.PaneChrome — that causes a double border.

// Compile-time check: GatewayHealthPane implements layout.Pane.
var _ layout.Pane = &GatewayHealthPane{}

// GatewayHealthPane displays a 4-row fixed grid showing token bucket fill,
// concurrent semaphore state, backoff countdown, and dedup waiter count.
// Data is read from gateway events via cursor-based reads from the store's event journal.
type GatewayHealthPane struct {
	store       state.StateReader
	theme       theme.Theme
	focused     bool
	width       int
	height      int
	eventCursor uint64
	snapshot    domain.GatewayStateSnapshot
}

// NewGatewayHealthPane creates a GatewayHealthPane with the given store and theme.
func NewGatewayHealthPane(s state.StateReader, th theme.Theme) *GatewayHealthPane {
	return &GatewayHealthPane{
		store:    s,
		theme:    th,
		snapshot: domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
	}
}

// ID returns the PaneGatewayHealth identifier.
func (p *GatewayHealthPane) ID() layout.PaneID { return layout.PaneGatewayHealth }

// Title returns the display title shown in the pane border.
func (p *GatewayHealthPane) Title() string { return "Gateway Health" }

// ToggleKey returns 2 — the Page B toggle key for this pane.
func (p *GatewayHealthPane) ToggleKey() int { return 2 }

// Actions returns nil — this pane has no interactive shortcuts.
func (p *GatewayHealthPane) Actions() []layout.Action { return nil }

// IsFocused returns whether this pane has keyboard focus.
func (p *GatewayHealthPane) IsFocused() bool { return p.focused }

// SetFocused updates the keyboard focus state.
func (p *GatewayHealthPane) SetFocused(f bool) { p.focused = f }

// Init returns nil — GatewayHealthPane reacts to TickMsg from the app.
func (p *GatewayHealthPane) Init() tea.Cmd { return nil }

// SetSize updates the content area dimensions.
func (p *GatewayHealthPane) SetSize(w, h int) { p.width = w; p.height = h }

// SetTheme updates the theme reference for runtime theme switching.
func (p *GatewayHealthPane) SetTheme(th theme.Theme) { p.theme = th }

// Update handles TickMsg to drain gateway events and update the snapshot.
func (p *GatewayHealthPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(TickMsg); ok {
		p.drainEvents()
	}
	return p, nil
}

// drainEvents reads new gateway events from the store and updates the snapshot
// to the newest event's Snapshot field.
func (p *GatewayHealthPane) drainEvents() {
	if p.store == nil {
		return
	}
	newCursor, events := p.store.ReadEventsFrom(p.eventCursor)
	p.eventCursor = newCursor
	if len(events) > 0 {
		p.snapshot = events[len(events)-1].Snapshot
	}
}

// View renders the 4-row health grid. Pure — no side effects.
func (p *GatewayHealthPane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	th := p.theme
	snap := p.snapshot
	mode := uikit.ActiveMode()
	const labelWidth = 8

	mutedStyle := lipgloss.NewStyle().Foreground(th.TextMuted())

	// Token row
	tokenColor := th.TextSecondary()
	if snap.TokensMax > 0 && snap.TokensAvailable <= 2 {
		tokenColor = th.Warning()
	}
	tokenStyle := lipgloss.NewStyle().Foreground(tokenColor)
	tokenIcon := tokenStyle.Render(uikit.GlyphFor(uikit.GlyphFilledDot, mode))
	tokenBar := p.renderDotBar(snap.TokensAvailable, snap.TokensMax,
		uikit.GlyphFilledDot, uikit.GlyphAvailable, tokenStyle, mutedStyle)
	tokenRow := p.renderRow(tokenIcon, "Tokens", tokenBar, labelWidth, mutedStyle)

	// Slot row
	slotColor := th.TextSecondary()
	if snap.ConcurrentMax > 0 && snap.ConcurrentActive >= snap.ConcurrentMax {
		slotColor = th.Warning()
	}
	slotStyle := lipgloss.NewStyle().Foreground(slotColor)
	slotIcon := slotStyle.Render(uikit.GlyphFor(uikit.GlyphFilledSquare, mode))
	slotBar := p.renderDotBar(snap.ConcurrentActive, snap.ConcurrentMax,
		uikit.GlyphFilledSquare, uikit.GlyphEmptySquare, slotStyle, mutedStyle)
	slotRow := p.renderRow(slotIcon, "Slots", slotBar, labelWidth, mutedStyle)

	// Backoff row
	backoffColor := th.TextMuted()
	backoffData := "none"
	if snap.BackoffRemaining > 0 {
		backoffColor = th.Error()
		backoffData = fmt.Sprintf("%.1fs", snap.BackoffRemaining)
	}
	backoffStyle := lipgloss.NewStyle().Foreground(backoffColor)
	backoffRow := p.renderRow(
		backoffStyle.Render(uikit.GlyphFor(uikit.GlyphDeadline, mode)),
		"Backoff", backoffStyle.Render(backoffData), labelWidth, mutedStyle)

	// Dedup row
	dedupColor := th.TextMuted()
	dedupData := "none"
	if snap.DedupWaiters > 0 {
		dedupColor = th.TextSecondary()
		dedupData = fmt.Sprintf("%d waiters", snap.DedupWaiters)
	}
	dedupStyle := lipgloss.NewStyle().Foreground(dedupColor)
	dedupRow := p.renderRow(
		dedupStyle.Render(uikit.GlyphFor(uikit.GlyphRateLimit, mode)),
		"Dedup", dedupStyle.Render(dedupData), labelWidth, mutedStyle)

	return strings.Join([]string{tokenRow, slotRow, backoffRow, dedupRow}, "\n")
}

// renderRow composes a single grid row: icon  label  data.
func (p *GatewayHealthPane) renderRow(icon, label, data string, labelWidth int, labelStyle lipgloss.Style) string {
	return icon + "  " + labelStyle.Render(uikit.PadOrTruncate(label, labelWidth)) + "  " + data
}

// renderDotBar renders a horizontal bar of filled/empty glyphs.
// filled is the current value; total is the maximum. Each position shows the
// filledRole glyph when its index < filled, emptyRole otherwise.
func (p *GatewayHealthPane) renderDotBar(filled, total int,
	filledRole, emptyRole uikit.GlyphRole,
	filledStyle, emptyStyle lipgloss.Style) string {
	if total <= 0 {
		return ""
	}
	mode := uikit.ActiveMode()
	var sb strings.Builder
	for i := 0; i < total; i++ {
		if i < filled {
			sb.WriteString(filledStyle.Render(uikit.GlyphFor(filledRole, mode)))
		} else {
			sb.WriteString(emptyStyle.Render(uikit.GlyphFor(emptyRole, mode)))
		}
	}
	return sb.String()
}
