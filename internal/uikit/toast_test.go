package uikit_test

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.dalton.dog/bubbleup"
)

// TestToast_DefaultTTL_ByIntent verifies default TTLs match spec §7.4:
// Success 4s, Info 4s, Warning 5s, Error 6s.
func TestToast_DefaultTTL_ByIntent(t *testing.T) {
	tests := []struct {
		name   string
		intent uikit.ToastIntent
		want   time.Duration
	}{
		{name: "Success 4s", intent: uikit.ToastSuccess, want: 4 * time.Second},
		{name: "Info 4s", intent: uikit.ToastInfo, want: 4 * time.Second},
		{name: "Warning 5s", intent: uikit.ToastWarning, want: 5 * time.Second},
		{name: "Error 6s", intent: uikit.ToastError, want: 6 * time.Second},
		{name: "RateLimit 30s default", intent: uikit.ToastRateLimit, want: 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uikit.DefaultTTL(tt.intent)
			assert.Equal(t, tt.want, got, "DefaultTTL(%v) should be %v", tt.intent, tt.want)
		})
	}
}

// TestToast_TruncatesTitle48Runes verifies Normalize hard-truncates Title at 48 runes with "…".
func TestToast_TruncatesTitle48Runes(t *testing.T) {
	// Exactly 48 runes — should not truncate.
	exactly48 := "A title that is exactly forty-eight runes long!!" // verified below
	for len([]rune(exactly48)) < 48 {
		exactly48 += "x"
	}
	exactly48 = string([]rune(exactly48)[:48])
	require.Equal(t, 48, len([]rune(exactly48)), "precondition: exactly48 is 48 runes")
	toast := uikit.Toast{Intent: uikit.ToastError, Title: exactly48}.Normalize()
	assert.Equal(t, exactly48, toast.Title, "exactly 48 runes: no truncation")

	// 49 runes — should truncate to 47 chars + "…".
	long49 := exactly48 + "x"
	require.Equal(t, 49, len([]rune(long49)), "precondition: long49 is 49 runes")
	toast2 := uikit.Toast{Intent: uikit.ToastError, Title: long49}.Normalize()
	runes := []rune(toast2.Title)
	assert.Equal(t, 48, len(runes), "49-rune title truncated to 48 runes")
	assert.Equal(t, '…', runes[47], "last rune should be ellipsis")
}

// TestToast_TruncatesBody160Runes verifies Normalize hard-truncates Body at 160 runes with "…".
func TestToast_TruncatesBody160Runes(t *testing.T) {
	// Exactly 160 runes — should not truncate.
	exactly160 := string(make([]rune, 160))
	for i := range exactly160 {
		exactly160 = exactly160[:i] + "a" + exactly160[i+1:]
	}
	require.Equal(t, 160, len([]rune(exactly160)), "precondition: exactly160 is 160 runes")
	toast := uikit.Toast{Intent: uikit.ToastError, Body: exactly160}.Normalize()
	assert.Equal(t, exactly160, toast.Body, "exactly 160 runes: no truncation")

	// 161 runes — should truncate to 159 + "…".
	long161 := exactly160 + "x" // 161 runes
	require.Equal(t, 161, len([]rune(long161)), "precondition: long161 is 161 runes")
	toast2 := uikit.Toast{Intent: uikit.ToastError, Body: long161}.Normalize()
	runes := []rune(toast2.Body)
	assert.Equal(t, 160, len(runes), "161-rune body truncated to 160 runes")
	assert.Equal(t, '…', runes[159], "last rune should be ellipsis")
}

// TestToast_GlyphByIntent verifies ToastGlyph returns correct unicode glyph per intent.
func TestToast_GlyphByIntent(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	tests := []struct {
		name   string
		intent uikit.ToastIntent
		want   string
	}{
		{name: "Success ✓", intent: uikit.ToastSuccess, want: "✓"},
		{name: "Error ✗", intent: uikit.ToastError, want: "✗"},
		{name: "Warning ◬", intent: uikit.ToastWarning, want: "◬"},
		{name: "Info →", intent: uikit.ToastInfo, want: "→"},
		{name: "RateLimit ⧖", intent: uikit.ToastRateLimit, want: "⧖"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uikit.ToastGlyph(tt.intent, uikit.GlyphUnicode)
			assert.Equal(t, tt.want, got, "ToastGlyph(%v, Unicode) should return %q", tt.intent, tt.want)
		})
	}
}

// TestToast_GlyphByIntent_ASCII verifies ToastGlyph returns correct ASCII glyph per intent.
func TestToast_GlyphByIntent_ASCII(t *testing.T) {
	tests := []struct {
		name   string
		intent uikit.ToastIntent
		want   string
	}{
		{name: "Success +", intent: uikit.ToastSuccess, want: "+"},
		{name: "Error x", intent: uikit.ToastError, want: "x"},
		{name: "Warning !", intent: uikit.ToastWarning, want: "!"},
		{name: "Info >", intent: uikit.ToastInfo, want: ">"},
		{name: "RateLimit ~", intent: uikit.ToastRateLimit, want: "~"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uikit.ToastGlyph(tt.intent, uikit.GlyphASCII)
			assert.Equal(t, tt.want, got, "ToastGlyph(%v, ASCII) should return %q", tt.intent, tt.want)
		})
	}
}

// TestToast_Normalize_DefaultsTTL verifies Normalize fills zero TTL with DefaultTTL.
func TestToast_Normalize_DefaultsTTL(t *testing.T) {
	toast := uikit.Toast{Intent: uikit.ToastError, Title: "Something broke"}.Normalize()
	assert.Equal(t, 6*time.Second, toast.TTL, "zero TTL should be defaulted to Error TTL (6s)")
}

// TestToast_Normalize_KeepsExplicitTTL verifies Normalize does not override an explicit TTL.
func TestToast_Normalize_KeepsExplicitTTL(t *testing.T) {
	toast := uikit.Toast{Intent: uikit.ToastError, Title: "Something broke", TTL: 30 * time.Second}.Normalize()
	assert.Equal(t, 30*time.Second, toast.TTL, "explicit TTL should be preserved")
}

// TestToast_Normalize_NoTruncationNeeded verifies Normalize does not change short Title/Body.
func TestToast_Normalize_NoTruncationNeeded(t *testing.T) {
	toast := uikit.Toast{Intent: uikit.ToastSuccess, Title: "Saved", Body: "Preference stored."}.Normalize()
	assert.Equal(t, "Saved", toast.Title)
	assert.Equal(t, "Preference stored.", toast.Body)
}

// TestToastManager_Cmd_NilModel verifies ToastManager.Cmd returns nil when AlertModel is nil.
func TestToastManager_Cmd_NilModel(t *testing.T) {
	mgr := uikit.NewToastManager(nil)
	cmd := mgr.Cmd(uikit.Toast{Intent: uikit.ToastInfo, Title: "Hello"})
	assert.Nil(t, cmd, "nil AlertModel should return nil cmd (guard path)")
}

// TestToastManager_Cmd_ReturnsNonNilCmd verifies ToastManager.Cmd returns a non-nil tea.Cmd
// when the AlertModel is properly initialised.
func TestToastManager_Cmd_ReturnsNonNilCmd(t *testing.T) {
	model := makeTestAlertModel()
	mgr := uikit.NewToastManager(model)

	allIntents := []uikit.ToastIntent{
		uikit.ToastSuccess,
		uikit.ToastError,
		uikit.ToastWarning,
		uikit.ToastInfo,
		uikit.ToastRateLimit,
	}
	for _, intent := range allIntents {
		cmd := mgr.Cmd(uikit.Toast{Intent: intent, Title: "Test title", Body: "Body text."})
		assert.NotNil(t, cmd, "Cmd should return non-nil for intent %v", intent)
	}
}

// TestToastManager_Cmd_BodyComposition verifies that Title-only and Title+Body
// toasts produce non-nil cmds.
func TestToastManager_Cmd_BodyComposition(t *testing.T) {
	model := makeTestAlertModel()
	mgr := uikit.NewToastManager(model)

	// Title only
	cmdTitleOnly := mgr.Cmd(uikit.Toast{Intent: uikit.ToastSuccess, Title: "Saved"})
	assert.NotNil(t, cmdTitleOnly, "title-only toast should return non-nil cmd")

	// Title + Body
	cmdBoth := mgr.Cmd(uikit.Toast{Intent: uikit.ToastError, Title: "Save failed", Body: "Try again."})
	assert.NotNil(t, cmdBoth, "title+body toast should return non-nil cmd")
}

// TestToast_DefaultTTL_InvalidIntent verifies DefaultTTL falls back to 4s for out-of-range intent.
func TestToast_DefaultTTL_InvalidIntent(t *testing.T) {
	got := uikit.DefaultTTL(uikit.ToastIntent(999))
	assert.Equal(t, 4*time.Second, got, "out-of-range intent should fall back to 4s")
}

// TestToast_GlyphByIntent_InvalidIntent verifies ToastGlyph falls back to GlyphInfo for out-of-range intent.
func TestToast_GlyphByIntent_InvalidIntent(t *testing.T) {
	got := uikit.ToastGlyph(uikit.ToastIntent(999), uikit.GlyphUnicode)
	expected := uikit.ToastGlyph(uikit.ToastInfo, uikit.GlyphUnicode)
	assert.Equal(t, expected, got, "out-of-range intent should fall back to info glyph")
}

// TestToastManager_Cmd_InvalidIntent verifies Cmd returns nil for out-of-range ToastIntent.
func TestToastManager_Cmd_InvalidIntent(t *testing.T) {
	model := makeTestAlertModel()
	mgr := uikit.NewToastManager(model)
	cmd := mgr.Cmd(uikit.Toast{Intent: uikit.ToastIntent(999), Title: "oops"})
	assert.Nil(t, cmd, "out-of-range intent should return nil cmd (bounds guard)")
}

// makeTestAlertModel creates a minimal bubbleup.AlertModel with all five Spotnik
// alert types registered so ToastManager.Cmd can be exercised without importing
// the components package (which imports theme, etc.).
func makeTestAlertModel() *bubbleup.AlertModel {
	model := bubbleup.NewAlertModel(60, false, 4*time.Second)
	model.RegisterNewAlertType(bubbleup.AlertDefinition{Key: "success", ForeColor: "#1db954"})
	model.RegisterNewAlertType(bubbleup.AlertDefinition{Key: "error", ForeColor: "#e05252"})
	model.RegisterNewAlertType(bubbleup.AlertDefinition{Key: "warning", ForeColor: "#e0a000"})
	model.RegisterNewAlertType(bubbleup.AlertDefinition{Key: "info", ForeColor: "#6ec7e0"})
	model.RegisterNewAlertType(bubbleup.AlertDefinition{Key: "ratelimit", ForeColor: "#e0a000"})
	positioned := model.WithPosition(bubbleup.BottomRightPosition)
	return &positioned
}
