package theme_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Task 0b.1 tests: registry and loader ----

func TestLoad_KnownID(t *testing.T) {
	got := theme.Load("monokai")
	require.NotNil(t, got)
	assert.Equal(t, "monokai", got.ID())
}

func TestLoad_UnknownID_FallsBackToDefault(t *testing.T) {
	got := theme.Load("does-not-exist")
	require.NotNil(t, got)
	assert.Equal(t, theme.DefaultThemeID, got.ID())
}

func TestLoad_DefaultTheme(t *testing.T) {
	got := theme.Load("black")
	require.NotNil(t, got)
	assert.Equal(t, "black", got.ID())
}

func TestAvailable_Returns5Entries(t *testing.T) {
	entries := theme.Available()
	assert.Equal(t, []string{"black", "monokai", "catppuccin", "nord", "light"}, entries)
}

func TestAvailable_StableOrder(t *testing.T) {
	first := theme.Available()
	second := theme.Available()
	assert.Equal(t, first, second)
}

func TestDefaultThemeID_IsBlack(t *testing.T) {
	assert.Equal(t, "black", theme.DefaultThemeID)
	assert.NotEmpty(t, theme.DefaultThemeID)
}

// ---- Task 0b.2 tests: all five themes ----

// allMethodsReturnNonEmpty verifies that every method on a Theme returns a non-empty value.
func allMethodsReturnNonEmpty(t *testing.T, th theme.Theme) {
	t.Helper()
	assert.NotEmpty(t, string(th.Base()), "Base()")
	assert.NotEmpty(t, string(th.Surface()), "Surface()")
	assert.NotEmpty(t, string(th.SurfaceAlt()), "SurfaceAlt()")
	assert.NotEmpty(t, string(th.ActiveBorder()), "ActiveBorder()")
	assert.NotEmpty(t, string(th.InactiveBorder()), "InactiveBorder()")
	assert.NotEmpty(t, string(th.TextPrimary()), "TextPrimary()")
	assert.NotEmpty(t, string(th.TextSecondary()), "TextSecondary()")
	assert.NotEmpty(t, string(th.TextMuted()), "TextMuted()")
	assert.NotEmpty(t, string(th.SelectedBg()), "SelectedBg()")
	assert.NotEmpty(t, string(th.SelectedFg()), "SelectedFg()")
	assert.NotEmpty(t, string(th.SectionHeader()), "SectionHeader()")
	assert.NotEmpty(t, string(th.PlayingIndicator()), "PlayingIndicator()")
	assert.NotEmpty(t, string(th.SeekBar()), "SeekBar()")
	assert.NotEmpty(t, string(th.VolumeBar()), "VolumeBar()")
	assert.NotEmpty(t, string(th.Success()), "Success()")
	assert.NotEmpty(t, string(th.Warning()), "Warning()")
	assert.NotEmpty(t, string(th.Error()), "Error()")
	assert.NotEmpty(t, string(th.DeviceActive()), "DeviceActive()")
	assert.NotEmpty(t, string(th.StatusBarBg()), "StatusBarBg()")
	assert.NotEmpty(t, string(th.StatusBarFg()), "StatusBarFg()")
	assert.NotEmpty(t, string(th.KeyHint()), "KeyHint()")
	assert.NotEmpty(t, th.ID(), "ID()")
	assert.NotEmpty(t, th.Name(), "Name()")
}

func TestAllThemes_ImplementInterface(t *testing.T) {
	for _, id := range theme.Available() {
		id := id // capture
		t.Run(id, func(t *testing.T) {
			th := theme.Load(id)
			allMethodsReturnNonEmpty(t, th)
		})
	}
}

func TestAllThemes_IDMatchesRegistryKey(t *testing.T) {
	for _, id := range theme.Available() {
		id := id
		t.Run(id, func(t *testing.T) {
			th := theme.Load(id)
			assert.Equal(t, id, th.ID())
		})
	}
}

func TestBlackTheme_Base_IsPureBlack(t *testing.T) {
	th := theme.Load("black")
	assert.Equal(t, "#000000", string(th.Base()))
}

func TestMonokaiTheme_Base(t *testing.T) {
	th := theme.Load("monokai")
	assert.Equal(t, "#272822", string(th.Base()))
}

func TestCatppuccinTheme_Base(t *testing.T) {
	th := theme.Load("catppuccin")
	assert.Equal(t, "#1e1e2e", string(th.Base()))
}

func TestNordTheme_Base(t *testing.T) {
	th := theme.Load("nord")
	assert.Equal(t, "#2e3440", string(th.Base()))
}

func TestLightTheme_Base(t *testing.T) {
	th := theme.Load("light")
	assert.Equal(t, "#eff1f5", string(th.Base()))
}
