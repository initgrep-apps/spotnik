package uikit

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"go.dalton.dog/bubbleup"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// ToastIntent classifies the semantic intent of a Toast notification.
// It maps to a bubbleup alert key, a theme colour, and a glyph.
type ToastIntent int

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
// toast intents. Glyph prefixes are resolved via GlyphFor at call time so the
// result honours ActiveMode(). Call this after uikit.Use() to ensure the correct
// mode is active.
func RegisterBubbleupAlerts(t theme.Theme) []bubbleup.AlertDefinition {
	m := ActiveMode()
	return []bubbleup.AlertDefinition{
		{
			Key:       "success",
			ForeColor: string(t.Success()),
			Prefix:    GlyphFor(GlyphSuccess, m),
		},
		{
			Key:       "error",
			ForeColor: string(t.Error()),
			Prefix:    GlyphFor(GlyphError, m),
		},
		{
			Key:       "warning",
			ForeColor: string(t.Warning()),
			Prefix:    GlyphFor(GlyphWarning, m),
		},
		{
			Key:       "info",
			ForeColor: string(t.Info()),
			Prefix:    GlyphFor(GlyphInfo, m),
		},
		{
			Key:       "ratelimit",
			ForeColor: string(t.Warning()),
			Prefix:    GlyphFor(GlyphRateLimit, m),
		},
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

// truncateRunes truncates s to at most max runes. If len([]rune(s)) > max,
// the last N runes (where N = len(ellipsis runes)) are replaced with the
// ellipsis glyph (unicode "…" = 1 rune, ascii "..." = 3 runes) so the result
// is exactly max runes. max must be >= len(ellipsis runes); Normalize always
// calls this with max >= 48 so this invariant is guaranteed.
func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	runes = runes[:max]
	ell := []rune(GlyphFor(GlyphEllipsis, ActiveMode()))
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
	key := intentKey[t.Intent]
	msg := t.Title
	if t.Body != "" {
		msg = t.Title + "\n" + t.Body
	}
	return tm.model.NewAlertCmd(key, msg)
}
