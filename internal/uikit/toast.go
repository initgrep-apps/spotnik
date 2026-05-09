package uikit

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"go.dalton.dog/bubbleup"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// ToastIntent classifies the semantic intent of a Toast notification.
// It maps to a bubbleup alert key, a theme colour, and a glyph.
type ToastIntent int

const (
	// ToastNone is the sentinel value returned by ErrorMapper.Map when no toast
	// should be dispatched (nil error or UnauthorizedError). It is outside the
	// valid range of dispatchable intents so ToastManager.Cmd silently drops it.
	ToastNone ToastIntent = -1
)

const (
	// ToastSuccess indicates a successful completion (e.g. "Saved", "Added").
	ToastSuccess ToastIntent = iota
	// ToastError indicates a failure that requires user attention.
	ToastError
	// ToastWarning indicates a non-fatal issue or advisory state.
	ToastWarning
	// ToastInfo indicates a neutral informational event.
	ToastInfo
	// ToastRateLimit indicates the Spotify API has returned a 429 rate-limit
	// response. Default TTL is Retry-After seconds (default 30s if unspecified).
	ToastRateLimit
)

// RegisterBubbleupAlerts builds the bubbleup AlertDefinition slice for the five
// toast intents. Prefix is set to "" because ToastManager.Cmd renders the glyph
// inline via renderToastMessage as a vertically-centred left column.
// Call this after uikit.Use() to ensure the correct mode is active.
func RegisterBubbleupAlerts(t theme.Theme) []bubbleup.AlertDefinition {
	return []bubbleup.AlertDefinition{
		{Key: "success", ForeColor: string(t.Success()), Prefix: ""},
		{Key: "error", ForeColor: string(t.Error()), Prefix: ""},
		{Key: "warning", ForeColor: string(t.Warning()), Prefix: ""},
		{Key: "info", ForeColor: string(t.Info()), Prefix: ""},
		{Key: "ratelimit", ForeColor: string(t.Warning()), Prefix: ""},
	}
}

// intentKey maps ToastIntent to the bubbleup alert key used by RegisterBubbleupAlerts
// and consumed by ToastManager.Cmd when dispatching bubbleup commands.
var intentKey = [...]string{
	ToastSuccess:   "success",
	ToastError:     "error",
	ToastWarning:   "warning",
	ToastInfo:      "info",
	ToastRateLimit: "ratelimit",
}

// intentGlyphRole maps ToastIntent to the GlyphRole used to look up the glyph.
var intentGlyphRole = [...]GlyphRole{
	ToastSuccess:   GlyphSuccess,
	ToastError:     GlyphError,
	ToastWarning:   GlyphWarning,
	ToastInfo:      GlyphInfo,
	ToastRateLimit: GlyphRateLimit,
}

// defaultTTLs holds the canonical TTL for each intent (spec §7.4).
var defaultTTLs = [...]time.Duration{
	ToastSuccess:   4 * time.Second,
	ToastError:     6 * time.Second,
	ToastWarning:   5 * time.Second,
	ToastInfo:      4 * time.Second,
	ToastRateLimit: 30 * time.Second,
}

// DefaultTTL returns the default display duration for the given intent.
// Success/Info: 4s. Warning: 5s. Error: 6s. RateLimit: 30s (Retry-After default).
func DefaultTTL(i ToastIntent) time.Duration {
	if int(i) < 0 || int(i) >= len(defaultTTLs) {
		return 4 * time.Second
	}
	return defaultTTLs[i]
}

// ToastGlyph returns the glyph string for the given intent in the given mode.
// The glyph is looked up from the frozen catalogue in glyph.go so the catalogue
// remains the single source of truth for all glyph assignments.
func ToastGlyph(i ToastIntent, m GlyphMode) string {
	if int(i) < 0 || int(i) >= len(intentGlyphRole) {
		return GlyphFor(GlyphInfo, m)
	}
	return GlyphFor(intentGlyphRole[i], m)
}

// Toast is the canonical notification value. Callers set Intent + Title and
// optionally Body and TTL. Normalize fills defaults before dispatch.
//
// Content rules (spec §7.4):
//   - Title: ≤ 48 runes, sentence case, no trailing ".", past-participle for
//     completions ("Saved"), noun + state for async events ("Device disconnected").
//   - Body: ≤ 160 runes, single sentence with trailing ".", optional for
//     Success/Info, required for Error (must name the next action).
//   - No emoji. Sentence case throughout.
type Toast struct {
	Intent ToastIntent
	Title  string
	Body   string
	TTL    time.Duration
}

// Normalize returns a copy of the Toast with:
//   - Title hard-truncated to 48 runes (last rune replaced with "…" if > 48).
//   - Body hard-truncated to 160 runes (last rune replaced with "…" if > 160).
//   - TTL defaulted to DefaultTTL(Intent) when zero.
func (t Toast) Normalize() Toast {
	t.Title = truncateRunes(t.Title, 48)
	t.Body = truncateRunes(t.Body, 160)
	if t.TTL == 0 {
		t.TTL = DefaultTTL(t.Intent)
	}
	return t
}

// renderToastMessage builds the multi-line interior string for bubbleup with a
// two-column layout: a 4-char left column containing the glyph on the
// vertically-centred row, and the right column containing title + optional body.
func renderToastMessage(glyph, title, body string) string {
	right := []string{title}
	if body != "" {
		right = append(right, body)
	}
	lines := len(right)

	// Determine glyph row (0-indexed). For even line counts this places the glyph
	// on the upper-middle line (e.g. line 0 of 2, line 1 of 4). True centre across
	// an even number of rows is impossible in a character grid; this convention is
	// the standard TUI approximation.
	glyphRow := (lines - 1) / 2

	const leftWidth = 4 // " " + glyph(1) + " " + gap(1)
	var sb strings.Builder
	for i := 0; i < lines; i++ {
		if i == glyphRow {
			sb.WriteString(" " + glyph + "  ")
		} else {
			sb.WriteString(strings.Repeat(" ", leftWidth))
		}
		sb.WriteString(right[i])
		if i < lines-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// truncateRunes truncates s to at most max runes. If len([]rune(s)) > max,
// the last N runes (where N = len(ellipsis runes)) are replaced with the
// ellipsis glyph (unicode "…" = 1 rune, ascii "..." = 3 runes) so the result
// is exactly max runes.
//
// Runtime guard: if max < len(ellipsis runes) the function returns s unchanged
// rather than panicking via an out-of-bounds slice. Normalize always calls this
// with max >= 48 so the guard path is defensive — it protects future callers
// that may pass a smaller max.
func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	ell := []rune(GlyphFor(GlyphEllipsis, ActiveMode()))
	if max < len(ell) {
		// Guard: cannot fit even the ellipsis — return original unmodified.
		return s
	}
	runes = runes[:max]
	copy(runes[max-len(ell):max], ell)
	return string(runes)
}

// ToastManager wraps a *bubbleup.AlertModel and provides a typed Cmd factory.
// Construct it once in App.New() and store as a.toasts.
type ToastManager struct {
	model *bubbleup.AlertModel
}

// NewToastManager wraps the given AlertModel in a ToastManager.
// model must be the same pointer as a.alerts in app.go so that theme
// changes (which reassign a.alerts) are reflected immediately: pass &a.alerts.
func NewToastManager(model *bubbleup.AlertModel) *ToastManager {
	return &ToastManager{model: model}
}

// Cmd normalises the toast, maps its intent to the registered bubbleup alert
// key, composes the message string (Title + optional "\n" + Body), and
// returns a tea.Cmd via model.NewAlertCmd.
//
// Returns nil if model is nil or if the intent is out of range.
func (tm *ToastManager) Cmd(t Toast) tea.Cmd {
	if tm.model == nil {
		return nil
	}
	if int(t.Intent) < 0 || int(t.Intent) >= len(intentKey) {
		return nil
	}
	t = t.Normalize()
	glyph := ToastGlyph(t.Intent, ActiveMode())
	msg := renderToastMessage(glyph, t.Title, t.Body)
	key := intentKey[t.Intent]
	return tm.model.NewAlertCmd(key, msg)
}
