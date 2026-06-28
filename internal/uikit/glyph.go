package uikit

import "github.com/charmbracelet/lipgloss"

// GlyphRole is a symbolic identifier for a glyph. Primitives never render raw
// runes — they look up the glyph by role, and the active GlyphMode determines
// whether the unicode or ascii form is returned.
type GlyphRole string

// GlyphMode is the active rendering mode for glyphs.
type GlyphMode int

const (
	GlyphUnicode GlyphMode = iota
	GlyphASCII
)

// Glyph roles. Grouped by category. Additions require updating
// docs/system/tui.md §4 (glyph catalogue) in the same PR per CLAUDE.md rule 17.
const (
	// Structural / borders
	GlyphCornerTL       GlyphRole = "corner.tl"
	GlyphCornerTR       GlyphRole = "corner.tr"
	GlyphCornerBL       GlyphRole = "corner.bl"
	GlyphCornerBR       GlyphRole = "corner.br"
	GlyphHRule          GlyphRole = "rule.h"
	GlyphVRule          GlyphRole = "rule.v"
	GlyphOverlayDismiss GlyphRole = "overlay.dismiss"

	// Intent / feedback
	GlyphSuccess   GlyphRole = "intent.success"
	GlyphError     GlyphRole = "intent.error"
	GlyphWarning   GlyphRole = "intent.warning"
	GlyphInfo      GlyphRole = "intent.info"
	GlyphRateLimit GlyphRole = "intent.ratelimit"
	GlyphRunning   GlyphRole = "intent.running"
	GlyphDeadline  GlyphRole = "intent.deadline"
	GlyphPaused    GlyphRole = "intent.paused"
	GlyphBlocked   GlyphRole = "intent.blocked"

	// State / availability
	GlyphActive       GlyphRole = "state.active"
	GlyphInactive     GlyphRole = "state.inactive"
	GlyphAvailable    GlyphRole = "state.available"
	GlyphFilledDot    GlyphRole = "state.filled"
	GlyphEmptySquare  GlyphRole = "state.empty.square"
	GlyphFilledSquare GlyphRole = "state.filled.square"
	GlyphLocked       GlyphRole = "state.locked"
	GlyphPinned       GlyphRole = "state.pinned"
	GlyphUnpinned     GlyphRole = "state.unpinned"
	// GlyphLiked marks a track as liked (heart). Used by NowPlaying and
	// LikedSongs panes to prepend a ♥ prefix on liked track names.
	GlyphLiked        GlyphRole = "state.liked"
	GlyphBullet       GlyphRole = "state.bullet"

	// Navigation / scroll
	GlyphScrollDown  GlyphRole = "nav.down"
	GlyphScrollUp    GlyphRole = "nav.up"
	GlyphScrollRight GlyphRole = "nav.right"
	GlyphScrollLeft  GlyphRole = "nav.left"
	GlyphEllipsis    GlyphRole = "nav.ellipsis"
	GlyphChevronR    GlyphRole = "nav.chevron.r"
	GlyphChevronL    GlyphRole = "nav.chevron.l"
	GlyphArrowLeft   GlyphRole = "nav.arrow.left"
	GlyphArrowRight  GlyphRole = "nav.arrow.right"
	GlyphArrowUp     GlyphRole = "nav.arrow.up"
	GlyphArrowDown   GlyphRole = "nav.arrow.down"
	GlyphArrowLR     GlyphRole = "nav.arrow.lr"

	// Playback controls
	GlyphPlaying   GlyphRole = "play.playing"
	GlyphPausedPB  GlyphRole = "play.paused"
	GlyphStop      GlyphRole = "play.stop"
	GlyphNext      GlyphRole = "play.next"
	GlyphPrev      GlyphRole = "play.prev"
	GlyphFFwd      GlyphRole = "play.ffwd"
	GlyphRewind    GlyphRole = "play.rewind"
	GlyphShuffle   GlyphRole = "play.shuffle"
	GlyphRepeatAll GlyphRole = "play.repeat.all"
	GlyphRepeatOne GlyphRole = "play.repeat.one"
	GlyphRepeatOff GlyphRole = "play.repeat.off"
	GlyphQueue     GlyphRole = "play.queue"
	GlyphEject     GlyphRole = "play.eject"

	// Domain / music / identity
	GlyphMusicNote  GlyphRole = "music.note"
	GlyphDoubleNote GlyphRole = "music.double"
	GlyphEpisode    GlyphRole = "music.episode"
	GlyphPremium    GlyphRole = "music.premium"
	GlyphFreeTier   GlyphRole = "music.free"
	GlyphCloud      GlyphRole = "music.cloud"
	GlyphPlaylist   GlyphRole = "music.playlist"

	// Generic separators
	GlyphSeparator GlyphRole = "sep.bullet"

	// Device-type icons (devices pane)
	GlyphDeviceComputer GlyphRole = "device.computer"
	GlyphDevicePhone    GlyphRole = "device.phone"
	GlyphDeviceSpeaker  GlyphRole = "device.speaker"
	GlyphDeviceTV       GlyphRole = "device.tv"

	// Keyboard chords
	GlyphEnter     GlyphRole = "kbd.enter"
	GlyphEscape    GlyphRole = "kbd.escape"
	GlyphTab       GlyphRole = "kbd.tab"
	GlyphBackspace GlyphRole = "kbd.backspace"
	GlyphSpace     GlyphRole = "kbd.space"

	// Superscripts (used for pane toggle keys)
	GlyphSuperscript0     GlyphRole = "sup.0"
	GlyphSuperscript1     GlyphRole = "sup.1"
	GlyphSuperscript2     GlyphRole = "sup.2"
	GlyphSuperscript3     GlyphRole = "sup.3"
	GlyphSuperscript4     GlyphRole = "sup.4"
	GlyphSuperscript5     GlyphRole = "sup.5"
	GlyphSuperscript6     GlyphRole = "sup.6"
	GlyphSuperscript7     GlyphRole = "sup.7"
	GlyphSuperscript8     GlyphRole = "sup.8"
	GlyphSuperscript9     GlyphRole = "sup.9"
	GlyphSuperscriptPlus  GlyphRole = "sup.plus"
	GlyphSuperscriptMinus GlyphRole = "sup.minus"

	// Graphical fills
	GlyphBarFull          GlyphRole = "bar.full"
	GlyphBarSevenEighths  GlyphRole = "bar.78"
	GlyphBarThreeQuarters GlyphRole = "bar.34"
	GlyphBarFiveEighths   GlyphRole = "bar.58"
	GlyphBarHalf          GlyphRole = "bar.12"
	GlyphBarThreeEighths  GlyphRole = "bar.38"
	GlyphBarQuarter       GlyphRole = "bar.14"
	GlyphBarOneEighth     GlyphRole = "bar.18"
	GlyphBarEmpty         GlyphRole = "bar.empty"
	GlyphBarMedium        GlyphRole = "bar.medium"
	GlyphBarHeavy         GlyphRole = "bar.heavy"
)

// glyphTable holds the unicode + ascii forms for every role. Additions require
// updating docs/system/tui.md §4 (glyph catalogue) in the same PR per CLAUDE.md rule 17.
var glyphTable = map[GlyphRole][2]string{
	// Structural
	GlyphCornerTL:       {"╭", "+"},
	GlyphCornerTR:       {"╮", "+"},
	GlyphCornerBL:       {"╰", "+"},
	GlyphCornerBR:       {"╯", "+"},
	GlyphHRule:          {"─", "-"},
	GlyphVRule:          {"│", "|"},
	GlyphOverlayDismiss: {"×", "x"},

	// Intent
	GlyphSuccess:   {"✓", "+"},
	GlyphError:     {"✗", "x"},
	GlyphWarning:   {"◬", "!"},
	GlyphInfo:      {"→", ">"},
	GlyphRateLimit: {"⧖", "~"},
	GlyphRunning:   {"⚡", "*"},
	GlyphDeadline:  {"◷", "@"},
	GlyphPaused:    {"⏸", "||"},
	GlyphBlocked:   {"⊘", "(-)"},

	// State
	GlyphActive:       {"◉", "(*)"},
	GlyphInactive:     {"◎", "( )"},
	GlyphAvailable:    {"○", "(o)"},
	GlyphFilledDot:    {"●", "(#)"},
	GlyphEmptySquare:  {"□", "[ ]"},
	GlyphFilledSquare: {"■", "[x]"},
	GlyphLocked:       {"◌", "(r)"},
	GlyphPinned:       {"★", "*"},
	GlyphUnpinned:     {"☆", "-"},
	GlyphLiked:        {"♥", "Y"},
	GlyphBullet:       {"•", "*"},

	// Navigation
	GlyphScrollDown:  {"▼", "v"},
	GlyphScrollUp:    {"▲", "^"},
	GlyphScrollRight: {"►", ">"},
	GlyphScrollLeft:  {"◄", "<"},
	GlyphEllipsis:    {"…", "..."},
	GlyphChevronR:    {"›", ">"},
	GlyphChevronL:    {"‹", "<"},
	GlyphArrowLeft:   {"←", "<-"},
	GlyphArrowRight:  {"→", "->"},
	GlyphArrowUp:     {"↑", "^"},
	GlyphArrowDown:   {"↓", "v"},
	GlyphArrowLR:     {"↔", "<>"},

	// Playback
	GlyphPlaying:   {"▶", ">"},
	GlyphPausedPB:  {"▷", "|>"},
	GlyphStop:      {"■", "[]"},
	GlyphNext:      {"⏭", ">>"},
	GlyphPrev:      {"⏮", "<<"},
	GlyphFFwd:      {"⏩", ">>>"},
	GlyphRewind:    {"⏪", "<<<"},
	GlyphShuffle:   {"⇄", "sh"},
	GlyphRepeatAll: {"↻", "rp"},
	GlyphRepeatOne: {"↻¹", "rp1"},
	GlyphRepeatOff: {"⟳", "ro"},
	GlyphQueue:     {"≡", "Q"},
	GlyphEject:     {"⏏", "/E"},

	// Domain
	GlyphMusicNote:  {"♪", "*"},
	GlyphDoubleNote: {"♫", "**"},
	GlyphEpisode:    {"◆", "EP"},
	GlyphPremium:    {"♛", "*P"},
	GlyphFreeTier:   {"○", "(o)"},
	GlyphCloud:      {"☁", "(c)"},
	GlyphPlaylist:   {"▤", "[=]"},

	// Generic separators
	GlyphSeparator: {"·", "|"},

	// Device-type icons
	GlyphDeviceComputer: {"⊡", "[c]"},
	GlyphDevicePhone:    {"⊞", "[p]"},
	GlyphDeviceSpeaker:  {"⊟", "[s]"},
	GlyphDeviceTV:       {"⊠", "[tv]"},

	// Keyboard chords
	GlyphEnter:     {"⏎", "Enter"},
	GlyphEscape:    {"⎋", "Esc"},
	GlyphTab:       {"⇥", "Tab"},
	GlyphBackspace: {"⌫", "BS"},
	GlyphSpace:     {"␣", "Space"},

	// Superscripts
	GlyphSuperscript0:     {"⁰", "0"},
	GlyphSuperscript1:     {"¹", "1"},
	GlyphSuperscript2:     {"²", "2"},
	GlyphSuperscript3:     {"³", "3"},
	GlyphSuperscript4:     {"⁴", "4"},
	GlyphSuperscript5:     {"⁵", "5"},
	GlyphSuperscript6:     {"⁶", "6"},
	GlyphSuperscript7:     {"⁷", "7"},
	GlyphSuperscript8:     {"⁸", "8"},
	GlyphSuperscript9:     {"⁹", "9"},
	GlyphSuperscriptPlus:  {"⁺", "+"},
	GlyphSuperscriptMinus: {"⁻", "-"},

	// Graphical
	GlyphBarFull:          {"█", "#"},
	GlyphBarSevenEighths:  {"▉", "#"},
	GlyphBarThreeQuarters: {"▊", "#"},
	GlyphBarFiveEighths:   {"▋", "="},
	GlyphBarHalf:          {"▌", "="},
	GlyphBarThreeEighths:  {"▍", "-"},
	GlyphBarQuarter:       {"▎", "-"},
	GlyphBarOneEighth:     {"▏", "."},
	GlyphBarEmpty:         {"░", "."},
	GlyphBarMedium:        {"▒", ":"},
	GlyphBarHeavy:         {"▓", "#"},
}

// GlyphFor returns the form of a role in the given mode.
// Unknown role returns empty string — caller is expected to have compiled
// against one of the exported GlyphRole constants.
// Unknown modes fall back to GlyphUnicode.
func GlyphFor(role GlyphRole, mode GlyphMode) string {
	row, ok := glyphTable[role]
	if !ok {
		return ""
	}
	switch mode {
	case GlyphASCII:
		return row[1]
	case GlyphUnicode:
		return row[0]
	default:
		return row[0]
	}
}

// AllGlyphRoles returns every registered role. Used by tests for full-table
// validation; order is not guaranteed.
func AllGlyphRoles() []GlyphRole {
	roles := make([]GlyphRole, 0, len(glyphTable))
	for r := range glyphTable {
		roles = append(roles, r)
	}
	return roles
}

// GlyphWidth returns the terminal-column width of a rendered glyph.
// Thin wrapper around lipgloss.Width for test convenience.
func GlyphWidth(s string) int { return lipgloss.Width(s) }
