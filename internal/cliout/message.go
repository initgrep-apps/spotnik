package cliout

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// Message is implemented by every CLI message type. External packages cannot
// implement this interface — the isMessage marker is unexported.
type Message interface {
	render(p Palette) string
	isMessage()
}

// Status is used by Header and Step to indicate the current state of an operation.
type Status int

const (
	// Active indicates the operation is currently running.
	Active Status = iota
	// Inactive indicates the operation is not running.
	Inactive
	// StatusSuccess indicates a successful outcome.
	StatusSuccess
	// StatusFailure indicates a failed outcome.
	StatusFailure
	// StatusWarning indicates a warning state.
	StatusWarning
	// Pending indicates the operation is queued but not yet started.
	Pending
)

// statusGlyphRole maps each Status value to its uikit GlyphRole.
// Glyphs are resolved at render time via uikit.GlyphFor so that ASCII mode
// and unicode mode are both honoured.
var statusGlyphRole = map[Status]uikit.GlyphRole{
	Active:        uikit.GlyphActive,
	Inactive:      uikit.GlyphInactive,
	StatusSuccess: uikit.GlyphSuccess,
	StatusFailure: uikit.GlyphError,
	StatusWarning: uikit.GlyphWarning,
	Pending:       uikit.GlyphLocked,
}

// statusGlyph returns the glyph for a status value in the current uikit mode.
func statusGlyph(s Status) string {
	role, ok := statusGlyphRole[s]
	if !ok {
		return "?"
	}
	return uikit.GlyphFor(role, uikit.ActiveMode())
}

// statusColor maps a status to a palette role colour.
func statusColor(s Status, p Palette) lipgloss.TerminalColor {
	switch s {
	case Active:
		return p.Accent
	case Inactive:
		return p.Muted
	case StatusSuccess:
		return p.Success
	case StatusFailure:
		return p.Error
	case StatusWarning:
		return p.Warning
	case Pending:
		return p.Muted
	default:
		return p.Plain
	}
}

// Header is a primary status line rendered as "◉ Subject  State".
type Header struct {
	Status  Status
	Subject string
	State   string
}

func (Header) isMessage() {}

func (h Header) render(p Palette) string {
	glyph := lipgloss.NewStyle().Foreground(statusColor(h.Status, p)).Bold(true).Render(statusGlyph(h.Status))
	subject := lipgloss.NewStyle().Foreground(p.Plain).Bold(true).Render(h.Subject)
	state := lipgloss.NewStyle().Foreground(p.Plain).Render(h.State)
	return glyph + "  " + subject + "  " + state
}

// Step is an inline progress event rendered as "✓ Text".
type Step struct {
	Status Status
	Text   string
}

func (Step) isMessage() {}

func (s Step) render(p Palette) string {
	glyph := lipgloss.NewStyle().Foreground(statusColor(s.Status, p)).Bold(true).Render(statusGlyph(s.Status))
	text := lipgloss.NewStyle().Foreground(p.Plain).Render(s.Text)
	return glyph + " " + text
}

// KVPair is one row of a KV block.
type KVPair struct {
	Label   string
	Value   string
	Caption string // optional trailing muted context
}

// KV is an aligned key/value block. Labels are right-padded so values align.
type KV struct {
	Pairs []KVPair
}

func (KV) isMessage() {}

func (kv KV) render(p Palette) string {
	if len(kv.Pairs) == 0 {
		return ""
	}
	maxLabel := 0
	for _, pair := range kv.Pairs {
		if len(pair.Label) > maxLabel {
			maxLabel = len(pair.Label)
		}
	}
	labelStyle := lipgloss.NewStyle().Foreground(p.Muted)
	valueStyle := lipgloss.NewStyle().Foreground(p.Plain)
	captionStyle := lipgloss.NewStyle().Foreground(p.Muted)

	lines := make([]string, len(kv.Pairs))
	for i, pair := range kv.Pairs {
		pad := strings.Repeat(" ", maxLabel-len(pair.Label))
		line := labelStyle.Render(pair.Label+pad) + "  " + valueStyle.Render(pair.Value)
		if pair.Caption != "" {
			line += "  " + captionStyle.Render("·  "+pair.Caption)
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

// Steps is a numbered instruction list. Numbers are 1..N automatic.
type Steps struct {
	Items []string
}

func (Steps) isMessage() {}

func (s Steps) render(p Palette) string {
	if len(s.Items) == 0 {
		return ""
	}
	numStyle := lipgloss.NewStyle().Foreground(p.Muted)
	textStyle := lipgloss.NewStyle().Foreground(p.Plain)
	lines := make([]string, len(s.Items))
	for i, item := range s.Items {
		num := fmt.Sprintf("%d", i+1)
		lines[i] = numStyle.Render(num) + "  " + textStyle.Render(item)
	}
	return strings.Join(lines, "\n")
}

// Hint is an action suggestion rendered as "→ Verb Cmd Tail".
type Hint struct {
	Verb string
	Cmd  string
	Tail string
}

func (Hint) isMessage() {}

func (h Hint) render(p Palette) string {
	arrow := lipgloss.NewStyle().Foreground(p.Accent).Bold(true).
		Render(uikit.GlyphFor(uikit.GlyphInfo, uikit.ActiveMode()))
	parts := []string{arrow}
	if h.Verb != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(p.Plain).Render(h.Verb))
	}
	if h.Cmd != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Render(h.Cmd))
	}
	if h.Tail != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(p.Plain).Render(h.Tail))
	}
	return strings.Join(parts, " ")
}

// URL is a bare URL with an optional preceding label.
type URL struct {
	Label string // optional muted precursor, e.g. "Visit this URL to authorize:"
	Href  string
}

func (URL) isMessage() {}

func (u URL) render(p Palette) string {
	href := lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Render(u.Href)
	if u.Label == "" {
		return href
	}
	label := lipgloss.NewStyle().Foreground(p.Muted).Render(u.Label)
	return label + "\n" + href
}

// Paragraph is free prose. Dim=true renders in Muted; otherwise Plain.
type Paragraph struct {
	Text string
	Dim  bool
}

func (Paragraph) isMessage() {}

func (pg Paragraph) render(p Palette) string {
	if pg.Dim {
		return lipgloss.NewStyle().Foreground(p.Muted).Render(pg.Text)
	}
	return lipgloss.NewStyle().Foreground(p.Plain).Render(pg.Text)
}

// Spinner and Prompt are implemented in Story 149. They satisfy Message so the
// taxonomy is stable, but render() panics if called before Story 149.

// Spinner displays an animated progress line during long-running operations.
type Spinner struct {
	Text string
}

func (Spinner) isMessage() {}

// render panics because Spinner must be passed to StartSpinner, not Write.
func (Spinner) render(_ Palette) string {
	panic("cliout.Spinner: pass to cliout.StartSpinner, not cliout.Write")
}

// Prompt displays an interactive input prompt with validation.
type Prompt struct {
	Label       string
	Placeholder string
	Validate    func(string) error
}

func (Prompt) isMessage() {}

// render panics because Prompt must be passed to Ask, not Write.
func (Prompt) render(_ Palette) string {
	panic("cliout.Prompt: pass to cliout.Ask, not cliout.Write")
}
