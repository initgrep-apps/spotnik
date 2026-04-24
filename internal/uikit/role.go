package uikit

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Role identifies an emphasis role in the design system. Primitives declare
// which fields map to which roles; the role-to-colour resolution happens via
// ColourFor(role, theme). Call sites never pass raw colours — they set a role.
type Role string

const (
	RoleAccent          Role = "accent"
	RoleStrong          Role = "strong"
	RolePlain           Role = "plain"
	RoleMuted           Role = "muted"
	RoleSuccess         Role = "success"
	RoleError           Role = "error"
	RoleWarning         Role = "warning"
	RoleInfo            Role = "info"
	RoleSelection       Role = "selection"
	RoleColumnIndex     Role = "column.index"
	RoleColumnPrimary   Role = "column.primary"
	RoleColumnSecondary Role = "column.secondary"
	RoleColumnTertiary  Role = "column.tertiary"
)

// AllRoles returns every registered role.
func AllRoles() []Role {
	return []Role{
		RoleAccent, RoleStrong, RolePlain, RoleMuted,
		RoleSuccess, RoleError, RoleWarning, RoleInfo,
		RoleSelection,
		RoleColumnIndex, RoleColumnPrimary, RoleColumnSecondary, RoleColumnTertiary,
	}
}

// ColourFor resolves a role to a lipgloss.Color on the given theme.
// Strong uses TextPrimary (bold is applied at the primitive level, not here).
func ColourFor(r Role, th theme.Theme) lipgloss.Color {
	switch r {
	case RoleAccent:
		return th.Accent()
	case RoleStrong, RolePlain:
		return th.TextPrimary()
	case RoleMuted:
		return th.TextMuted()
	case RoleSuccess:
		return th.Success()
	case RoleError:
		return th.Error()
	case RoleWarning:
		return th.Warning()
	case RoleInfo:
		return th.Info()
	case RoleSelection:
		return th.SelectedFg()
	case RoleColumnIndex:
		return th.ColumnIndex()
	case RoleColumnPrimary:
		return th.ColumnPrimary()
	case RoleColumnSecondary:
		return th.ColumnSecondary()
	case RoleColumnTertiary:
		return th.ColumnTertiary()
	default:
		return th.TextPrimary()
	}
}

// Apply returns a lipgloss.Style foreground-coloured by the role, with Bold
// set when role is Strong. All primitives compose their own additional style
// attributes (width, alignment) on top of this.
func Apply(r Role, th theme.Theme) lipgloss.Style {
	s := lipgloss.NewStyle().Foreground(ColourFor(r, th))
	if r == RoleStrong {
		s = s.Bold(true)
	}
	return s
}
