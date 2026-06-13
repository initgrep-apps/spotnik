package layout_test

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain forces TrueColor profile so ANSI codes are emitted in tests,
// regardless of whether the test runner has a TTY attached.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}

// stripANSI removes ANSI escape sequences so we can test plain text content.
func stripANSI(s string) string {
	// lipgloss.Width strips ANSI for width calculation — we use a simple state machine.
	result := make([]byte, 0, len(s))
	inEsc := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			// End of escape sequence: 'm' for SGR, 'K' for EL, etc.
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEsc = false
			}
			continue
		}
		result = append(result, s[i])
	}
	return string(result)
}

// ── Task 1 Tests: RenderPaneBorder basic behaviour ────────────────────────────

func TestRenderPaneBorder_BasicCornerCharacters(t *testing.T) {
	th := theme.Load("black")
	cfg := layout.BorderConfig{
		Width:       20,
		Height:      5,
		Title:       "Test",
		ToggleKey:   0,
		AccentColor: th.PaneBorderNowPlaying(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	require.GreaterOrEqual(t, len(lines), 1)

	plain := stripANSI(lines[0])
	assert.True(t, strings.HasPrefix(plain, "╭"), "top-left corner should be ╭")
	assert.True(t, strings.HasSuffix(plain, "╮"), "top-right corner should be ╮")

	// Check bottom border
	lastLine := stripANSI(lines[len(lines)-1])
	assert.True(t, strings.HasPrefix(lastLine, "╰"), "bottom-left corner should be ╰")
	assert.True(t, strings.HasSuffix(lastLine, "╯"), "bottom-right corner should be ╯")
}

func TestRenderPaneBorder_WithToggleKey(t *testing.T) {
	th := theme.Load("black")
	cfg := layout.BorderConfig{
		Width:       30,
		Height:      5,
		Title:       "Playlists",
		ToggleKey:   3,
		AccentColor: th.PaneBorderPlaylists(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)

	topLine := stripANSI(lines[0])
	assert.Contains(t, topLine, "³", "superscript 3 should appear in top border")
	assert.Contains(t, topLine, "Playlists", "title should appear in top border")
}

func TestRenderPaneBorder_WithActions(t *testing.T) {
	th := theme.Load("black")
	actions := []layout.Action{
		{Key: "f", Label: "filter"},
		{Key: "n", Label: "new"},
	}
	cfg := layout.BorderConfig{
		Width:       60,
		Height:      5,
		Title:       "Playlists",
		ToggleKey:   3,
		Actions:     actions,
		AccentColor: th.PaneBorderPlaylists(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)

	topLine := stripANSI(lines[0])
	// Story 75: actions now use corner-notch format (╮ key label ╭) — no ᐅ prefix.
	assert.NotContains(t, topLine, "ᐅ", "action prefix ᐅ should NOT appear in corner-notch format")
	assert.Contains(t, topLine, "filter", "action label 'filter' should appear")
	assert.Contains(t, topLine, "new", "action label 'new' should appear")
}

func TestRenderPaneBorder_WidthMatchesRequested(t *testing.T) {
	th := theme.Load("black")
	const wantWidth = 40
	cfg := layout.BorderConfig{
		Width:       wantWidth,
		Height:      5,
		Title:       "Queue",
		ToggleKey:   2,
		AccentColor: th.PaneBorderQueue(),
		Focused:     false,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		assert.Equal(t, wantWidth, w, "line %d width should be %d, got %d: %q", i, wantWidth, w, stripANSI(line))
	}
}

func TestRenderPaneBorder_HeightMatchesRequested(t *testing.T) {
	th := theme.Load("black")
	const wantHeight = 7
	cfg := layout.BorderConfig{
		Width:       30,
		Height:      wantHeight,
		Title:       "Albums",
		AccentColor: th.PaneBorderAlbums(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	assert.Equal(t, wantHeight, len(lines), "should produce exactly %d lines", wantHeight)
}

func TestRenderPaneBorder_ContentLinesPaddedToFit(t *testing.T) {
	th := theme.Load("black")
	const w = 30
	content := "hello"
	cfg := layout.BorderConfig{
		Width:       w,
		Height:      3,
		Title:       "Test",
		AccentColor: th.PaneBorderNowPlaying(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder(content, cfg)
	lines := strings.Split(result, "\n")
	// Content line is lines[1] — should be padded to width w
	require.Len(t, lines, 3)
	contentLine := lines[1]
	assert.Equal(t, w, lipgloss.Width(contentLine), "content line should be width %d", w)
}

func TestRenderPaneBorder_FilterMode(t *testing.T) {
	th := theme.Load("black")
	actions := []layout.Action{
		{Key: "f", Label: "filter"},
		{Key: "n", Label: "new"},
	}
	cfg := layout.BorderConfig{
		Width:       60,
		Height:      5,
		Title:       "Queue",
		ToggleKey:   2,
		Actions:     actions,
		FilterQuery: "rock",
		AccentColor: th.PaneBorderQueue(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)

	topLine := stripANSI(lines[0])
	// Filter mode now renders as f(query) — graded shrink label, no close-notch.
	assert.Contains(t, topLine, "f(rock)", "filter mode must render f(query) label")
	assert.Contains(t, topLine, "rock", "filter query should appear")
	// Actions should NOT appear in filter mode
	assert.NotContains(t, topLine, "new", "action labels should not appear in filter mode")
	// Close-notch retired — no "Esc close" in filter mode
	assert.NotContains(t, topLine, "Esc close", "filter mode must not render close-notch")
}

func TestFormatFilterLabel(t *testing.T) {
	cases := []struct {
		name   string
		query  string
		budget int
		want   string
	}{
		// Variant 1 — full unquoted form fits
		{"full at large budget", "rock", 20, "f(rock)"},
		{"full at exact 7-col budget (rock)", "rock", 7, "f(rock)"},
		{"full at exact 12-col budget (rocknroll)", "rocknroll", 12, "f(rocknroll)"},
		// Variant 2 — truncation. "rocknroll" trimmed: f(rocknro…) w11, …,
		// f(roc…) w7, f(ro…) w6, f(r…) w5.
		{"truncated to f(rocknro…) at 11-col budget", "rocknroll", 11, "f(rocknro…)"},
		{"truncated to f(roc…) at 7-col budget", "rocknroll", 7, "f(roc…)"},
		{"truncated to f(ro…) at 6-col budget", "rocknroll", 6, "f(ro…)"},
		{"truncated to f(r…) at 5-col budget", "rocknroll", 5, "f(r…)"},
		// "rock" can also truncate when budget < 7
		{"rock truncated to f(ro…) at 6-col budget", "rock", 6, "f(ro…)"},
		{"rock truncated to f(r…) at 5-col budget", "rock", 5, "f(r…)"},
		// Variant 3 — minimal indicator
		{"falls back to f(…) at 4-col budget", "rocknroll", 4, "f(…)"},
		// Single-rune query: variant 2 loop is skipped (i starts at 0, condition i>=1 false)
		{"single-rune query fits f(x) at 4-col budget", "x", 4, "f(x)"},
		{"single-rune query drops at 3-col budget (no truncation possible)", "x", 3, ""},
		// Variant 4 — drop
		{"too narrow drops to empty", "rocknroll", 3, ""},
		{"empty query returns empty", "", 100, ""},
		{"zero budget returns empty", "rock", 0, ""},
		{"negative budget returns empty", "rock", -1, ""},
		// Wide-rune (CJK) queries: lipgloss.Width counts each as 2 columns;
		// formatFilterLabel relies on that for correct fit calculations.
		// Example: "字" → "f(字)" width 5. At budget 5 → returns "f(字)".
		// At budget 4 → variant 1 fails (5>4), variant 2 loop skipped
		// (1 rune), variant 3 "f(…)" width 4 → returns "f(…)".
		{"wide-rune fits f(字) at 5-col budget", "字", 5, "f(字)"},
		{"wide-rune falls back to f(…) at 4-col budget", "字", 4, "f(…)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := layout.FormatFilterLabel(tc.query, tc.budget)
			assert.Equal(t, tc.want, got)
			if got != "" {
				assert.LessOrEqual(t, lipgloss.Width(got), tc.budget,
					"label width must fit within budget")
			}
		})
	}
}

func TestRenderPaneBorder_NarrowPane_FilterShrinksGracefully(t *testing.T) {
	cfg := layout.BorderConfig{
		Width: 30, Height: 5,
		Title: "Queue", ToggleKey: 2,
		AccentColor: lipgloss.Color("#888"),
		FilterQuery: "rock",
		Theme:       theme.Load("black"),
	}
	out := stripANSI(layout.RenderPaneBorder("", cfg))

	// Positive: filter indicator must be present in some form.
	assert.Contains(t, out, "f(", "narrow pane must still show filter indicator")

	// Negative: the close-notch is fully retired in filter mode. Verify by
	// asserting that no segment of the rendered output contains the literal
	// close-notch sequence "Esc close" (case-sensitive). The action-mode
	// notch with "filter" is also absent because cfg.FilterQuery is non-empty.
	assert.NotContains(t, out, "Esc close",
		"filter mode must not render the ╮ Esc close ╭ notch (Esc is global, see help overlay)")
}

func TestRenderPaneBorder_FilterModeRendersOnlyLabel(t *testing.T) {
	// Wide pane so f("rock") fits comfortably; assert the right segment is
	// exactly the label, no notch, no separator dash bar before "╮" except
	// the top-border fill dashes.
	cfg := layout.BorderConfig{
		Width: 80, Height: 5,
		Title: "Queue", ToggleKey: 2,
		AccentColor: lipgloss.Color("#888"),
		FilterQuery: "rock",
		Theme:       theme.Load("black"),
	}
	topLine := strings.Split(stripANSI(layout.RenderPaneBorder("", cfg)), "\n")[0]
	assert.Contains(t, topLine, "f(rock)")
	assert.NotContains(t, topLine, "Esc close")
	// The top-right notch corners are part of the border itself (╭...╮); the
	// CLOSE-NOTCH was an inner ╮...╭ pair around "Esc close". Confirm we have
	// exactly one ╮ on the line (the top-right border corner).
	assert.Equal(t, 1, strings.Count(topLine, "╮"),
		"filter mode should have exactly one ╮ (the top-right border corner)")
}

func TestRenderPaneBorder_NoToggleKey(t *testing.T) {
	th := theme.Load("black")
	cfg := layout.BorderConfig{
		Width:       30,
		Height:      5,
		Title:       "Request Flow",
		ToggleKey:   0, // no toggle key
		AccentColor: th.PaneBorderRequestFlow(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)

	topLine := stripANSI(lines[0])
	// No superscript digits should appear
	for _, sup := range []string{"¹", "²", "³", "⁴", "⁵", "⁶", "⁷", "⁸"} {
		assert.NotContains(t, topLine, sup, "no superscript should appear when ToggleKey=0")
	}
	assert.Contains(t, topLine, "Request Flow", "title should still appear")
}

func TestRenderPaneBorder_EmptyActionsOnlyTitleAndDashes(t *testing.T) {
	th := theme.Load("black")
	cfg := layout.BorderConfig{
		Width:       40,
		Height:      5,
		Title:       "Albums",
		ToggleKey:   4,
		Actions:     nil, // empty actions
		AccentColor: th.PaneBorderAlbums(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)

	topLine := stripANSI(lines[0])
	// Only dashes after title
	assert.NotContains(t, topLine, "ᐅ", "no action prefix without actions")
	assert.Contains(t, topLine, "─", "dashes should appear")
	assert.Contains(t, topLine, "Albums", "title should appear")
}

func TestRenderPaneBorder_FocusedStyleApplied(t *testing.T) {
	th := theme.Load("black")
	cfg := layout.BorderConfig{
		Width:       30,
		Height:      5,
		Title:       "Test",
		AccentColor: th.PaneBorderNowPlaying(),
		Focused:     true,
		Theme:       th,
	}
	focusedResult := layout.RenderPaneBorder("", cfg)

	cfg.Focused = false
	unfocusedResult := layout.RenderPaneBorder("", cfg)

	// The outputs should differ — focused: full AccentColor + bold title;
	// unfocused: AccentColor + Faint (dimmed but still colored).
	assert.NotEqual(t, focusedResult, unfocusedResult, "focused and unfocused renders should differ")
}

func TestRenderPaneBorder_UnfocusedFaintStyle(t *testing.T) {
	th := theme.Load("black")
	cfg := layout.BorderConfig{
		Width:       30,
		Height:      5,
		Title:       "Test",
		AccentColor: th.PaneBorderNowPlaying(),
		Focused:     false,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	// Unfocused border: AccentColor + Faint (dim). Both color escape and dim code are present.
	assert.Contains(t, result, "\x1b[", "unfocused border should contain ANSI escape codes")
}

// ── Story 71 Task 1: Unfocused accent color ───────────────────────────────────

// TestRenderPaneBorder_Unfocused_UsesAccentColor verifies that an unfocused border
// still emits the accent color escape (Foreground) along with the faint modifier,
// rather than only the faint grey produced by lipgloss.Faint(true) alone.
//
// The black theme NowPlaying accent is #00ff88 → TrueColor RGB(0, 255, 136) →
// escape sequence "38;2;0;255;136". This escape must appear in the unfocused render
// (the border chars), not only in the key-hint or title segments.
func TestRenderPaneBorder_Unfocused_UsesAccentColor(t *testing.T) {
	th := theme.Load("black")
	// Use a config with ToggleKey=0 and no actions so the only styled elements
	// are the border characters themselves and the title.
	accentColor := th.PaneBorderNowPlaying() // #00ff88 → 38;2;0;255;136
	cfg := layout.BorderConfig{
		Width:       40,
		Height:      5,
		Title:       "Test",
		ToggleKey:   0, // no superscript — eliminates KeyHint color from output
		Actions:     nil,
		AccentColor: accentColor,
		Focused:     false,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)

	// With ToggleKey=0 and no Actions, the only colored elements are borders and title.
	// The accent color #00ff88 encodes as "38;2;0;255;136" in TrueColor mode.
	// If the border uses Faint(true) alone, this sequence will NOT appear.
	// If it uses Foreground(AccentColor)+Faint(true), this sequence WILL appear.
	const accentEscape = "38;2;0;255;136"
	assert.Contains(t, result, accentEscape,
		"unfocused border must emit the accent foreground color (not just faint grey)")
	// Faint modifier must also be present.
	assert.Contains(t, result, "\x1b[2", "unfocused border must include the faint (dim) modifier")
}

// TestRenderPaneBorder_Focused_NoBoldRegression verifies that a focused border
// emits both the accent color and bold, and that both focused vs unfocused renders
// differ (focus is distinguished by brightness/bold, not just presence of color).
func TestRenderPaneBorder_Focused_NoBoldRegression(t *testing.T) {
	th := theme.Load("black")
	accentColor := th.PaneBorderNowPlaying()
	cfgFocused := layout.BorderConfig{
		Width:       40,
		Height:      5,
		Title:       "Now Playing",
		ToggleKey:   1,
		AccentColor: accentColor,
		Focused:     true,
		Theme:       th,
	}
	cfgUnfocused := cfgFocused
	cfgUnfocused.Focused = false

	focused := layout.RenderPaneBorder("", cfgFocused)
	unfocused := layout.RenderPaneBorder("", cfgUnfocused)

	// Both must contain the accent color sequence.
	assert.Contains(t, focused, "38;2;", "focused border must contain foreground color escape")
	assert.Contains(t, unfocused, "38;2;", "unfocused border must contain foreground color escape")
	// The two renders must differ — focus is visually distinguishable.
	assert.NotEqual(t, focused, unfocused, "focused and unfocused renders must differ")
	// Focused title must use bold (ANSI code 1). Lipgloss emits bold as "[1;" prefix.
	assert.Contains(t, focused, "\x1b[1;", "focused title must be bold")
}

// ── Task 1: PaneBorderColor helper ───────────────────────────────────────────

func TestPaneBorderColor_ReturnsCorrectColorPerPane(t *testing.T) {
	th := theme.Load("black")
	tests := []struct {
		id   layout.PaneID
		want lipgloss.Color
	}{
		{layout.PaneNowPlaying, th.PaneBorderNowPlaying()},
		{layout.PaneQueue, th.PaneBorderQueue()},
		{layout.PanePlaylists, th.PaneBorderPlaylists()},
		{layout.PaneAlbums, th.PaneBorderAlbums()},
		{layout.PaneLikedSongs, th.PaneBorderLikedSongs()},
		{layout.PaneRecentlyPlayed, th.PaneBorderRecentlyPlayed()},
		{layout.PaneTopTracks, th.PaneBorderTopTracks()},
		{layout.PaneTopArtists, th.PaneBorderTopArtists()},
		{layout.PaneGatewayHealth, th.PaneBorderRequestFlow()},
		{layout.PanePollingTraffic, th.PaneBorderRequestFlow()},
		{layout.PaneGatewayLive, th.PaneBorderRequestFlow()},
		{layout.PaneNetworkLog, th.PaneBorderNetworkLog()},
		{layout.PanePodcastPlayback, th.PaneBorderNowPlaying()},
		{layout.PaneShowEpisodes, th.PaneBorderShowEpisodes()},
		{layout.PaneFollowedShows, th.PaneBorderFollowedShows()},
		{layout.PaneSavedEpisodes, th.PaneBorderSavedEpisodes()},
	}
	for _, tt := range tests {
		got := layout.PaneBorderColor(tt.id, th)
		assert.Equal(t, tt.want, got, "pane %d color mismatch", tt.id)
	}
}

func TestPaneBorderColor_UnknownIDFallback(t *testing.T) {
	th := theme.Load("black")
	// Unknown pane ID should not panic — returns some color
	got := layout.PaneBorderColor(layout.PaneID(99), th)
	assert.NotEmpty(t, string(got), "unknown ID should return non-empty fallback color")
}

// ── Task 2 Tests: Edge cases and content truncation ───────────────────────────

func TestRenderPaneBorder_NarrowBorderDropsActions(t *testing.T) {
	th := theme.Load("black")
	actions := []layout.Action{
		{Key: "f", Label: "filter"},
		{Key: "n", Label: "new"},
	}
	cfg := layout.BorderConfig{
		Width:   15,
		Height:  5,
		Title:   "Queue",
		Actions: actions,
		Focused: true,
		Theme:   th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)

	topLine := stripANSI(lines[0])
	// At width 15, actions should be dropped to fit
	assert.NotContains(t, topLine, "filter", "actions should be dropped on narrow border")
}

func TestRenderPaneBorder_VeryNarrowTruncatesTitle(t *testing.T) {
	th := theme.Load("black")
	cfg := layout.BorderConfig{
		Width:   10,
		Height:  5,
		Title:   "A Very Long Title",
		Focused: true,
		Theme:   th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)

	topLine := stripANSI(lines[0])
	assert.True(t, strings.HasPrefix(topLine, "╭"), "should still start with ╭")
	assert.True(t, strings.HasSuffix(topLine, "╮"), "should still end with ╮")
	// Width must still be exact
	for _, line := range lines {
		w := lipgloss.Width(line)
		assert.Equal(t, 10, w, "line width should still be 10")
	}
}

func TestRenderPaneBorder_ContentShorterThanHeightPadded(t *testing.T) {
	th := theme.Load("black")
	// Content is 1 line but height=5 means we need 3 content lines (5-2 borders)
	cfg := layout.BorderConfig{
		Width:       30,
		Height:      5,
		Title:       "Test",
		AccentColor: th.PaneBorderNowPlaying(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("one line", cfg)
	lines := strings.Split(result, "\n")
	assert.Equal(t, 5, len(lines), "should be exactly 5 lines")
}

func TestRenderPaneBorder_ContentWiderThanWidthTruncated(t *testing.T) {
	th := theme.Load("black")
	const w = 20
	// Content wider than the interior (w-2 = 18 chars)
	longContent := strings.Repeat("X", 50)
	cfg := layout.BorderConfig{
		Width:       w,
		Height:      3,
		Title:       "T",
		AccentColor: th.PaneBorderNowPlaying(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder(longContent, cfg)
	lines := strings.Split(result, "\n")
	require.Len(t, lines, 3)

	// All lines must be exactly w columns wide
	for i, line := range lines {
		assert.Equal(t, w, lipgloss.Width(line), "line %d should be width %d", i, w)
	}
	// Content line should contain truncation marker
	contentLine := stripANSI(lines[1])
	assert.Contains(t, contentLine, "…", "truncated content should end with ellipsis")
}

func TestRenderPaneBorder_UnicodeContentMeasuredCorrectly(t *testing.T) {
	th := theme.Load("black")
	const w = 20
	// CJK characters are 2 columns wide each
	cjkContent := "日本語テスト" // 6 chars × 2 columns = 12 columns
	cfg := layout.BorderConfig{
		Width:       w,
		Height:      3,
		Title:       "T",
		AccentColor: th.PaneBorderNowPlaying(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder(cjkContent, cfg)
	lines := strings.Split(result, "\n")
	require.Len(t, lines, 3)

	for i, line := range lines {
		w := lipgloss.Width(line)
		assert.Equal(t, 20, w, "line %d with CJK content should be exactly 20 wide", i)
	}
}

func TestRenderPaneBorder_ActionPrefixCharacterWidth(t *testing.T) {
	// Verify that ᐅ (U+1405) measures as 1 column wide
	width := lipgloss.Width("ᐅ")
	assert.Equal(t, 1, width, "ᐅ (U+1405) should be 1 column wide")
}

func TestRenderPaneBorder_SuperscriptCharacterWidth(t *testing.T) {
	// Verify superscripts measure as 1 column wide
	for _, sup := range []string{"¹", "²", "³", "⁴", "⁵", "⁶", "⁷", "⁸"} {
		width := lipgloss.Width(sup)
		assert.Equal(t, 1, width, "superscript %q should be 1 column wide", sup)
	}
}

// ── Task 3 Tests: Integration scenarios ──────────────────────────────────────

func TestRenderPaneBorder_NowPlayingBorder(t *testing.T) {
	th := theme.Load("black")
	actions := []layout.Action{
		{Key: "s", Label: "shuffle"},
		{Key: "r", Label: "repeat"},
	}
	cfg := layout.BorderConfig{
		Width:       60,
		Height:      10,
		Title:       "Now Playing",
		ToggleKey:   1,
		Actions:     actions,
		AccentColor: th.PaneBorderNowPlaying(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	assert.Equal(t, 10, len(lines), "NowPlaying border should be 10 lines tall")

	topLine := stripANSI(lines[0])
	assert.Contains(t, topLine, "¹", "toggle key 1 superscript")
	assert.Contains(t, topLine, "Now Playing", "title")
	assert.Contains(t, topLine, "shuffle", "shuffle action")
	assert.Contains(t, topLine, "repeat", "repeat action")

	for i, line := range lines {
		assert.Equal(t, 60, lipgloss.Width(line), "line %d should be exactly 60 wide", i)
	}
}

func TestRenderPaneBorder_PlaylistsBorder(t *testing.T) {
	th := theme.Load("black")
	actions := []layout.Action{
		{Key: "f", Label: "filter"},
		{Key: "n", Label: "new"},
		{Key: "r", Label: "rename"},
		{Key: "x", Label: "delete"},
	}
	cfg := layout.BorderConfig{
		Width:       80,
		Height:      15,
		Title:       "Playlists",
		ToggleKey:   3,
		Actions:     actions,
		AccentColor: th.PaneBorderPlaylists(),
		Focused:     false,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	assert.Equal(t, 15, len(lines))

	topLine := stripANSI(lines[0])
	assert.Contains(t, topLine, "³")
	assert.Contains(t, topLine, "Playlists")

	for i, line := range lines {
		assert.Equal(t, 80, lipgloss.Width(line), "line %d width", i)
	}
}

func TestRenderPaneBorder_QueueWithActiveFilter(t *testing.T) {
	th := theme.Load("black")
	actions := []layout.Action{
		{Key: "f", Label: "filter"},
		{Key: "x", Label: "clear"},
	}
	cfg := layout.BorderConfig{
		Width:       60,
		Height:      8,
		Title:       "Queue",
		ToggleKey:   2,
		Actions:     actions,
		FilterQuery: "rock",
		AccentColor: th.PaneBorderQueue(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	assert.Equal(t, 8, len(lines))

	topLine := stripANSI(lines[0])
	// Filter mode now renders as f(query) label — no legacy "filtering:" prefix.
	assert.Contains(t, topLine, "f(rock)", "filter mode must render f(query) label")
	assert.Contains(t, topLine, "rock")
	assert.NotContains(t, topLine, "filtering:", "old filtering: prefix retired")
	// Original actions should not appear
	assert.NotContains(t, topLine, "clear")

	for i, line := range lines {
		assert.Equal(t, 60, lipgloss.Width(line), "line %d width", i)
	}
}

func TestRenderPaneBorder_StatsPageRequestFlowNoToggleKey(t *testing.T) {
	th := theme.Load("black")
	cfg := layout.BorderConfig{
		Width:       70,
		Height:      12,
		Title:       "Request Flow",
		ToggleKey:   0, // Stats page — no toggle key
		AccentColor: th.PaneBorderRequestFlow(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	assert.Equal(t, 12, len(lines))

	topLine := stripANSI(lines[0])
	assert.Contains(t, topLine, "Request Flow")
	for _, sup := range []string{"¹", "²", "³", "⁴", "⁵", "⁶", "⁷", "⁸"} {
		assert.NotContains(t, topLine, sup)
	}

	for i, line := range lines {
		assert.Equal(t, 70, lipgloss.Width(line), "line %d width", i)
	}
}

func TestRenderPaneBorder_ExactDimensions(t *testing.T) {
	th := theme.Load("black")
	tests := []struct {
		w, h int
	}{
		{20, 4},
		{40, 8},
		{80, 20},
		{100, 30},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			cfg := layout.BorderConfig{
				Width:       tt.w,
				Height:      tt.h,
				Title:       "Test",
				AccentColor: th.PaneBorderNowPlaying(),
				Focused:     true,
				Theme:       th,
			}
			result := layout.RenderPaneBorder("", cfg)
			lines := strings.Split(result, "\n")
			assert.Equal(t, tt.h, len(lines), "height mismatch for %dx%d", tt.w, tt.h)
			for i, line := range lines {
				assert.Equal(t, tt.w, lipgloss.Width(line), "line %d width mismatch for %dx%d", i, tt.w, tt.h)
			}
		})
	}
}

func TestRenderPaneBorder_SideBySideNoOverlap(t *testing.T) {
	th := theme.Load("black")
	makePane := func(title string, w, h int) string {
		cfg := layout.BorderConfig{
			Width:       w,
			Height:      h,
			Title:       title,
			AccentColor: th.PaneBorderNowPlaying(),
			Focused:     false,
			Theme:       th,
		}
		return layout.RenderPaneBorder("", cfg)
	}

	left := makePane("Left", 40, 10)
	right := makePane("Right", 40, 10)
	combined := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	// Combined width should be 80 (40+40)
	combinedLines := strings.Split(combined, "\n")
	for i, line := range combinedLines {
		assert.Equal(t, 80, lipgloss.Width(line), "side-by-side line %d should be 80 wide", i)
	}
}

// ── Story 75 Task 1: Corner-notch border actions ──────────────────────────────

// TestBuildRightSegment_CornerNotchFormat verifies that the corner-notch format uses
// ╮ and ╭ characters and does NOT use the old ᐅ prefix style for actions.
func TestBuildRightSegment_CornerNotchFormat(t *testing.T) {
	th := theme.Load("black")
	actions := []layout.Action{
		{Key: "s", Label: "shfl"},
		{Key: "r", Label: "rpt"},
	}
	cfg := layout.BorderConfig{
		Width:       60,
		Height:      5,
		Title:       "Now Playing",
		ToggleKey:   1,
		Actions:     actions,
		AccentColor: th.PaneBorderNowPlaying(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)

	topLine := stripANSI(lines[0])
	// The top line should contain multiple ╭ characters: the action notches embed them
	// between each action. Count of ╭ must be at least len(actions) (one per action).
	notchCount := strings.Count(topLine, "╭")
	assert.GreaterOrEqual(t, notchCount, len(actions),
		"corner-notch format should have at least one ╭ per action, got %d in: %q", notchCount, topLine)
	assert.Contains(t, topLine, "shfl", "action label 'shfl' should appear")
	assert.Contains(t, topLine, "rpt", "action label 'rpt' should appear")
	// ᐅ must NOT appear in the action segment (corner-notch replaces it).
	assert.NotContains(t, topLine, "ᐅ", "corner-notch format must not use ᐅ prefix for actions")
}

// TestRenderPaneBorder_FilterMode_UsesLabelNotNotch verifies that filter mode
// renders the graded f(query) label and does NOT render the retired close-notch
// or the banned ᐅ prefix.
func TestRenderPaneBorder_FilterMode_UsesLabelNotNotch(t *testing.T) {
	th := theme.Load("black")
	cfg := layout.BorderConfig{
		Width:       60,
		Height:      5,
		Title:       "Queue",
		ToggleKey:   2,
		FilterQuery: "rock",
		AccentColor: th.PaneBorderQueue(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)

	topLine := stripANSI(lines[0])
	assert.NotContains(t, topLine, "ᐅ", "filter mode must not use ᐅ prefix")
	assert.NotContains(t, topLine, `filtering: "rock"`, "old filtering: prefix retired")
	assert.NotContains(t, topLine, "Esc close", "close-notch retired — Esc is global")
	assert.Contains(t, topLine, "f(rock)", "filter mode must render f(query) label")
}

// TestRenderPaneBorder_ASCIIMode_SwapsCorners verifies that when ascii glyph
// overrides are passed in BorderConfig the border renderer emits + instead of
// ╭╮╰╯. Border.go does not call uikit.ActiveMode(); callers (e.g. PaneChrome)
// resolve glyphs and pass them via the Glyph fields.
func TestRenderPaneBorder_ASCIIMode_SwapsCorners(t *testing.T) {
	cfg := layout.BorderConfig{
		Width:       40,
		Height:      3,
		Title:       "Test",
		AccentColor: lipgloss.Color("#ffffff"),
		Focused:     true,
		Theme:       theme.Load("black"),
		// ASCII glyph overrides — normally set by uikit.PaneChrome.Render.
		CornerTL: "+",
		CornerTR: "+",
		CornerBL: "+",
		CornerBR: "+",
		HRule:    "-",
		VRule:    "|",
	}
	out := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(out, "\n")

	topLine := stripANSI(lines[0])
	bottomLine := stripANSI(lines[len(lines)-1])

	assert.Contains(t, topLine, "+", "ascii top-left corner must be +")
	assert.NotContains(t, topLine, "╭", "no unicode top-left corner in ascii mode")
	assert.NotContains(t, topLine, "╮", "no unicode top-right corner in ascii mode")
	assert.Contains(t, bottomLine, "+", "ascii bottom corners must be +")
	assert.NotContains(t, bottomLine, "╰", "no unicode bottom-left corner in ascii mode")
	assert.NotContains(t, bottomLine, "╯", "no unicode bottom-right corner in ascii mode")
}

// TestRenderPaneBorder_NotchActions_FitsWidth verifies that the total rendered
// width matches config width exactly when actions use corner-notch format.
func TestRenderPaneBorder_NotchActions_FitsWidth(t *testing.T) {
	th := theme.Load("black")
	actions := []layout.Action{
		{Key: "s", Label: "shfl"},
		{Key: "r", Label: "rpt"},
		{Key: "space", Label: "play"},
	}
	widths := []int{60, 80, 100}
	for _, w := range widths {
		cfg := layout.BorderConfig{
			Width:       w,
			Height:      5,
			Title:       "Now Playing",
			ToggleKey:   1,
			Actions:     actions,
			AccentColor: th.PaneBorderNowPlaying(),
			Focused:     true,
			Theme:       th,
		}
		result := layout.RenderPaneBorder("", cfg)
		lines := strings.Split(result, "\n")
		for i, line := range lines {
			assert.Equal(t, w, lipgloss.Width(line),
				"line %d should be exactly %d wide with corner-notch actions", i, w)
		}
	}
}

func TestRenderPaneBorder_AllThemesAccentColorsChange(t *testing.T) {
	themeIDs := theme.Available()
	results := make(map[string]string)

	for _, id := range themeIDs {
		th := theme.Load(id)
		cfg := layout.BorderConfig{
			Width:       30,
			Height:      5,
			Title:       "Test",
			AccentColor: th.PaneBorderNowPlaying(),
			Focused:     true,
			Theme:       th,
		}
		results[id] = layout.RenderPaneBorder("", cfg)
	}

	// All theme renders should produce the correct dimensions
	for _, id := range themeIDs {
		lines := strings.Split(results[id], "\n")
		assert.Equal(t, 5, len(lines), "theme %s should produce 5 lines", id)
		for i, line := range lines {
			assert.Equal(t, 30, lipgloss.Width(line), "theme %s line %d width", id, i)
		}
	}
}

// ── Story 77 Task 3: rightSuffix conditional — flush corner when no actions ───

// TestRenderPaneBorder_NoActions_FlushCorner verifies that when there are no actions
// the top border ends with ─╮ (dash immediately before corner) with no space gap.
func TestRenderPaneBorder_NoActions_FlushCorner(t *testing.T) {
	th := theme.Load("black")
	cfg := layout.BorderConfig{
		Width:       40,
		Height:      5,
		Title:       "Test",
		ToggleKey:   0,
		Actions:     nil, // no actions
		AccentColor: th.PaneBorderNowPlaying(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)

	topLine := stripANSI(lines[0])
	// The top border must end with "─╮" — a dash immediately before the corner, no space.
	assert.True(t, strings.HasSuffix(topLine, "─╮"),
		"top border with no actions should end with ─╮ (no space before corner), got: %q", topLine)
}

// TestRenderPaneBorder_WithActions_FlushCorner verifies that when actions are
// present the last notch's ╭ is immediately followed by the corner ╮ with no space.
// The notch ╭ already provides visual separation from ╮.
func TestRenderPaneBorder_WithActions_FlushCorner(t *testing.T) {
	th := theme.Load("black")
	cfg := layout.BorderConfig{
		Width:       60,
		Height:      5,
		Title:       "Test",
		ToggleKey:   0,
		Actions:     []layout.Action{{Key: "f", Label: "filter"}},
		AccentColor: th.PaneBorderNowPlaying(),
		Focused:     true,
		Theme:       th,
	}
	result := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)

	topLine := stripANSI(lines[0])
	// The action segment ends with ╭, immediately followed by the corner ╮.
	assert.True(t, strings.HasSuffix(topLine, "╭╮"),
		"top border with actions should end with '╭╮' (no space before corner), got: %q", topLine)
}
