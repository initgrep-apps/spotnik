package components

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// gradientVolumeBarWidth is the default number of fill characters in the volume bar.
const gradientVolumeBarWidth = 14

// GradientSeekBar renders a seek bar with a gradient fill interpolated from
// Gradient1() (left) to Gradient2() (right), with an empty portion in Surface().
type GradientSeekBar struct {
	th    theme.Theme
	width int
}

// NewGradientSeekBar creates a gradient seek bar using theme tokens.
func NewGradientSeekBar(t theme.Theme) *GradientSeekBar {
	return &GradientSeekBar{th: t}
}

// SetWidth updates the bar width.
func (b *GradientSeekBar) SetWidth(width int) {
	b.width = width
}

// Render returns the seek bar string for the given progress.
// progressMs and durationMs are in milliseconds.
// Format: "1:41  ████████████████░░░░░░░░░░░░░░  5:30"
func (b *GradientSeekBar) Render(progressMs, durationMs int) string {
	elapsed := formatDuration(progressMs)
	total := formatDuration(durationMs)

	// Reserve space for time labels and padding: "m:ss  " + "  m:ss" = 6+len each side.
	// Use 2 spaces padding each side.
	labelPad := 2
	timeWidth := len(elapsed) + len(total) + labelPad*2
	barWidth := b.width - timeWidth
	if barWidth < 1 {
		barWidth = 1
	}

	var ratio float64
	if durationMs > 0 {
		ratio = float64(progressMs) / float64(durationMs)
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1.0 {
		ratio = 1.0
	}

	m := uikit.ActiveMode()
	fullGlyph := uikit.GlyphFor(uikit.GlyphBarFull, m)
	emptyGlyph := uikit.GlyphFor(uikit.GlyphBarEmpty, m)

	g1 := string(b.th.Gradient1())
	g2 := string(b.th.Gradient2())
	emptyStyle := lipgloss.NewStyle().Foreground(b.th.Surface())

	// Compute fill using the §5.7 partial-block algorithm.
	filledFloat := ratio * float64(barWidth)
	fillCount := int(filledFloat)
	remainder := filledFloat - float64(fillCount)

	// Build gradient fill: each full character gets an interpolated color.
	var sb strings.Builder
	for i := 0; i < fillCount; i++ {
		var t float64
		if fillCount > 1 {
			t = float64(i) / float64(fillCount-1)
		}
		col := interpolateHex(g1, g2, t)
		sb.WriteString(lipgloss.NewStyle().Foreground(col).Render(fullGlyph))
	}
	// Partial block at the fill boundary — coloured with the end-of-gradient tone.
	emptyCount := barWidth - fillCount
	if remainder > 0 && fillCount < barWidth {
		col := interpolateHex(g1, g2, 1.0)
		sb.WriteString(lipgloss.NewStyle().Foreground(col).Render(uikit.PartialGlyph(remainder, m)))
		emptyCount--
	}
	sb.WriteString(emptyStyle.Render(strings.Repeat(emptyGlyph, emptyCount)))

	return elapsed + strings.Repeat(" ", labelPad) + sb.String() + strings.Repeat(" ", labelPad) + total
}

// VolumeDebounceTickMsg is the timer payload fired by GradientVolumeBar.HandleKey
// after the 300 ms debounce window. Seq is a monotonically increasing counter;
// NowPlayingPane discards the tick when Seq does not match the bar's current seq,
// meaning a newer keypress superseded this one.
type VolumeDebounceTickMsg struct {
	TargetVol int
	Seq       int
}

// GradientVolumeBar renders a volume bar with color bands and a music note icon.
// Format: "♪ ████▎░░░░░░░░░ 31%"
//
// Full cells use █; the fractional last cell uses the §5.7 partial-block
// algorithm (▏▎▍▌▋▊▉) giving sub-character resolution on every 1% step.
// Empty cells use ░ (GlyphBarEmpty).
//
// Color bands:
//   - 0-33%:  Gradient1() (green/cool)
//   - 34-66%: Gradient2() (yellow/warm)
//   - 67-100%: Gradient3() (red/hot)
//
// Icon color:
//   - volume > 0: ♪ in Gradient1() color
//   - volume = 0: ♪ in TextMuted() color
//
// The bar is a smart component: call HandleKey on volume keypresses, HandleDebounce
// on VolumeDebounceTickMsg, and SetConfirmed when a Spotify poll resolves.
type GradientVolumeBar struct {
	th         theme.Theme
	width      int  // total component width; 0 uses default
	currentVol int  // displayed volume: pending value or last confirmed value
	hasPending bool // true while a debounce tick is in flight
	seq        int  // monotonically increasing; stale ticks have a lower seq
}

// NewGradientVolumeBar creates a gradient volume bar using theme tokens.
func NewGradientVolumeBar(t theme.Theme) *GradientVolumeBar {
	return &GradientVolumeBar{th: t}
}

// SetWidth updates the total component width (including icon and percentage).
func (b *GradientVolumeBar) SetWidth(width int) {
	b.width = width
}

// SetConfirmed updates currentVol from the authoritative Spotify poll value.
// It is a no-op when hasPending is true — the optimistic pending value is shown
// until the debounce resolves, at which point the next poll re-syncs.
func (b *GradientVolumeBar) SetConfirmed(vol int) {
	if !b.hasPending {
		b.currentVol = vol
	}
}

// HandleKey computes the new pending volume, updates currentVol immediately so
// the bar renders the new value on the next frame, increments seq to invalidate
// any in-flight debounce tick, and returns a 300 ms debounce cmd.
//
// delta must be +1 or -1. confirmedVol is the store's current device volume,
// used as the starting base only when no pending intent exists (hasPending == false).
func (b *GradientVolumeBar) HandleKey(delta, confirmedVol int) tea.Cmd {
	base := confirmedVol
	if b.hasPending {
		base = b.currentVol
	}
	newVol := base + delta
	if newVol > 100 {
		newVol = 100
	}
	if newVol < 0 {
		newVol = 0
	}
	b.currentVol = newVol
	b.hasPending = true
	b.seq++
	target, seq := newVol, b.seq
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
		return VolumeDebounceTickMsg{TargetVol: target, Seq: seq}
	})
}

// HandleDebounce checks whether the debounce tick is current.
// Returns (true, targetVol) when seq matches — the caller should emit
// VolumeIntentMsg{TargetVol: targetVol} to trigger the API call.
// Returns (false, 0) when the tick is stale (a newer keypress superseded it).
// Increments seq on a successful match so duplicate ticks with the same seq
// are discarded on subsequent calls.
func (b *GradientVolumeBar) HandleDebounce(m VolumeDebounceTickMsg) (matched bool, targetVol int) {
	if m.Seq != b.seq {
		return false, 0
	}
	b.hasPending = false
	b.seq++ // prevent double-fire: any future tick with the same seq is now stale
	return true, m.TargetVol
}

// Render returns the volume bar string for the given volume level.
// Volume is clamped to [0, 100].
// Format: "♪ ████▎░░░░░░░░░ 31%"
//
// Full cells use █ (U+2588). The fractional last cell uses the §5.7 partial-block
// thresholds (▏▎▍▌▋▊▉) so the bar moves smoothly on every 1% step.
// Empty cells use ░ (GlyphBarEmpty).
func (b *GradientVolumeBar) Render(volume int) string {
	if volume > 100 {
		volume = 100
	}
	if volume < 0 {
		volume = 0
	}

	// Determine the fill bar width from the component width, or use the default.
	barWidth := gradientVolumeBarWidth
	if b.width > 0 {
		// "♪ " = 2 chars, "  XX%" = up to 5 chars → reserve 7
		reserved := 7
		computed := b.width - reserved
		if computed > 0 {
			barWidth = computed
		}
	}
	if barWidth < 1 {
		barWidth = 1
	}

	// Pick fill color from the volume band.
	var fillColor lipgloss.Color
	switch {
	case volume <= 33:
		fillColor = b.th.Gradient1()
	case volume <= 66:
		fillColor = b.th.Gradient2()
	default:
		fillColor = b.th.Gradient3()
	}

	m := uikit.ActiveMode()
	fullGlyph := uikit.GlyphFor(uikit.GlyphBarFull, m)
	emptyGlyph := uikit.GlyphFor(uikit.GlyphBarEmpty, m)

	fillStyle := lipgloss.NewStyle().Foreground(fillColor)
	emptyStyle := lipgloss.NewStyle().Foreground(b.th.Surface())

	filledFloat := float64(volume) / 100.0 * float64(barWidth)
	fullBlocks := int(filledFloat)
	remainder := filledFloat - float64(fullBlocks)

	var sb strings.Builder
	// Full blocks.
	for i := 0; i < fullBlocks; i++ {
		sb.WriteString(fillStyle.Render(fullGlyph))
	}
	// Partial block: emit for any non-zero remainder (§5.7 algorithm — no dead zone).
	emptyCount := barWidth - fullBlocks
	if remainder > 0 && fullBlocks < barWidth {
		sb.WriteString(fillStyle.Render(uikit.PartialGlyph(remainder, m)))
		emptyCount--
	}
	// Empty cells.
	sb.WriteString(emptyStyle.Render(strings.Repeat(emptyGlyph, emptyCount)))

	bar := sb.String()

	// Music note icon: gradient color when volume > 0, muted when 0.
	note := uikit.GlyphFor(uikit.GlyphMusicNote, m)
	var icon string
	if volume > 0 {
		iconStyle := lipgloss.NewStyle().Foreground(b.th.Gradient1())
		icon = iconStyle.Render(note)
	} else {
		iconStyle := lipgloss.NewStyle().Foreground(b.th.TextMuted())
		icon = iconStyle.Render(note)
	}

	return fmt.Sprintf("%s %s  %d%%", icon, bar, volume)
}

// interpolateHex interpolates between two hex color strings.
// t ranges from 0.0 (returns hex1) to 1.0 (returns hex2). Values outside [0,1] are clamped.
// Hex strings should be in the format "#rrggbb".
func interpolateHex(hex1, hex2 string, t float64) lipgloss.Color {
	if t <= 0 {
		return lipgloss.Color(hex1)
	}
	if t >= 1 {
		return lipgloss.Color(hex2)
	}

	r1, g1, b1 := parseHex(hex1)
	r2, g2, b2 := parseHex(hex2)

	r := lerp(r1, r2, t)
	g := lerp(g1, g2, t)
	b := lerp(b1, b2, t)

	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

// parseHex parses a "#rrggbb" hex string into r, g, b uint8 components.
// Returns (0, 0, 0) on parse failure.
func parseHex(hex string) (r, g, b uint8) {
	s := strings.TrimPrefix(hex, "#")
	if len(s) != 6 {
		return 0, 0, 0
	}
	rv, _ := strconv.ParseUint(s[0:2], 16, 8)
	gv, _ := strconv.ParseUint(s[2:4], 16, 8)
	bv, _ := strconv.ParseUint(s[4:6], 16, 8)
	return uint8(rv), uint8(gv), uint8(bv)
}

// lerp linearly interpolates between two uint8 values using signed arithmetic to
// avoid uint8 underflow when b < a.
func lerp(a, b uint8, t float64) uint8 {
	return uint8(math.Round(float64(a) + t*(float64(b)-float64(a))))
}

// formatDuration converts milliseconds to "m:ss" string (e.g. 154000 → "2:34").
func formatDuration(ms int) string {
	totalSeconds := ms / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}
