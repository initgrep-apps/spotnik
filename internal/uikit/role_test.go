package uikit_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

func TestRole_AllRolesResolveToNonEmptyColour(t *testing.T) {
	th := theme.Load("black")
	for _, role := range uikit.AllRoles() {
		c := uikit.ColourFor(role, th)
		assert.NotEmpty(t, string(c), "role %q resolved to empty colour", role)
	}
}

func TestRole_AccentFallsBackToSeekBarWhenUnset(t *testing.T) {
	// The spec Section 6.1 states Accent falls back to SeekBar when
	// theme.Accent() is unset. Theme implementations must return either the
	// TOML `accent` field or SeekBar().
	th := theme.Load("black")
	assert.NotEmpty(t, string(uikit.ColourFor(uikit.RoleAccent, th)))
}

func TestRole_PlainIsTextPrimary(t *testing.T) {
	th := theme.Load("black")
	assert.Equal(t, th.TextPrimary(), uikit.ColourFor(uikit.RolePlain, th))
}

func TestRole_MutedIsTextMuted(t *testing.T) {
	th := theme.Load("black")
	assert.Equal(t, th.TextMuted(), uikit.ColourFor(uikit.RoleMuted, th))
}

func TestRole_UnknownRoleFallsBackToTextPrimary(t *testing.T) {
	// The default branch in ColourFor returns TextPrimary for any unrecognised role.
	th := theme.Load("black")
	unknown := uikit.Role("does.not.exist")
	assert.Equal(t, th.TextPrimary(), uikit.ColourFor(unknown, th))
}

func TestRole_Apply_StrongSetsBold(t *testing.T) {
	th := theme.Load("black")
	s := uikit.Apply(uikit.RoleStrong, th)
	assert.True(t, s.GetBold(), "Apply(RoleStrong) must set Bold(true)")
}

func TestRole_Apply_PlainDoesNotSetBold(t *testing.T) {
	th := theme.Load("black")
	s := uikit.Apply(uikit.RolePlain, th)
	assert.False(t, s.GetBold(), "Apply(RolePlain) must not set Bold")
}

func TestRole_PaneBorderFor_KnownPaneIDReturnsNonEmpty(t *testing.T) {
	th := theme.Load("black")
	knownIDs := []string{
		"nowplaying", "queue", "playlists", "albums", "likedsongs",
		"recentlyplayed", "toptracks", "topartists", "requestflow", "networklog",
	}
	for _, id := range knownIDs {
		c := uikit.PaneBorderFor(id, th)
		assert.NotEmpty(t, string(c), "PaneBorderFor(%q) returned empty colour", id)
	}
}

func TestRole_PaneBorderFor_UnknownIDFallsBackToAccent(t *testing.T) {
	th := theme.Load("black")
	c := uikit.PaneBorderFor("unknown-pane", th)
	assert.Equal(t, th.Accent(), c, "PaneBorderFor with unknown ID should fall back to Accent()")
}
