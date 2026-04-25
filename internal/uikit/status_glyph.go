package uikit

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// roleGlyph maps an intent Role to its GlyphRole counterpart.
// Only the four status-intent roles (Success, Error, Warning, Info) are mapped;
// unknown roles fall back to GlyphInfo so callers never see an empty glyph.
var roleGlyph = map[Role]GlyphRole{
	RoleSuccess: GlyphSuccess,
	RoleError:   GlyphError,
	RoleWarning: GlyphWarning,
	RoleInfo:    GlyphInfo,
}

// StatusGlyph renders persistent informational state as "<glyph> <text>" where
// the glyph is coloured by Role. Use it for persistent status lines (e.g. Premium
// warnings, config notices) — not for transient notifications (use Toast for those).
//
// Supported roles: RoleSuccess (✓), RoleError (✗), RoleWarning (◬), RoleInfo (→).
// Unknown roles fall back to RoleInfo.
type StatusGlyph struct {
	// Role drives both the glyph character and its foreground colour.
	Role Role
	// Text is the label rendered after the glyph with a single space separator.
	Text string
	// Theme provides the colour token for the role.
	Theme theme.Theme
	// Gap is the number of extra spaces to insert between the glyph and the text,
	// in addition to the mandatory single space. Default 0 produces "◬ text";
	// Gap: 1 produces "◬  text" (two spaces) which aligns with adjacent lines
	// that were rendered with the legacy two-space padding (e.g. "✓  text").
	Gap int
}

// Render returns the ANSI-styled string "<glyph><sep><text>" where sep is one
// space plus Gap additional spaces (Gap == 0 → single space, Gap == 1 → two spaces).
// The glyph is resolved via the frozen glyph catalogue (glyph.go) so the
// active GlyphMode (unicode/ascii) is respected automatically.
func (sg StatusGlyph) Render() string {
	gr, ok := roleGlyph[sg.Role]
	if !ok {
		gr = GlyphInfo
	}
	sep := " " + strings.Repeat(" ", sg.Gap)
	glyph := lipgloss.NewStyle().
		Foreground(ColourFor(sg.Role, sg.Theme)).
		Render(GlyphFor(gr, ActiveMode()))
	text := lipgloss.NewStyle().
		Foreground(ColourFor(sg.Role, sg.Theme)).
		Render(sg.Text)
	return glyph + sep + text
}
