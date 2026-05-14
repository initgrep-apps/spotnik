package panes

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestProfileOverlay creates a ProfileOverlay wired to a fresh store and black theme.
func newTestProfileOverlay() (*ProfileOverlay, *state.Store) {
	s := state.New()
	t := theme.Load("black")
	return NewProfileOverlay(s, t), s
}

// TestProfileOverlay_View_ShowsDisplayName verifies that View() renders the user's display name.
func TestProfileOverlay_View_ShowsDisplayName(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})

	view := overlay.View()

	assert.Contains(t, view, "Irshad Sheikh", "View should render the user's display name")
}

// TestProfileOverlay_View_PremiumBadge verifies that a premium user sees the ♛ Premium badge.
func TestProfileOverlay_View_PremiumBadge(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})

	view := overlay.View()

	assert.Contains(t, view, "♛", "View should render ♛ for Premium users")
	assert.Contains(t, view, "Premium", "View should render 'Premium' text for premium users")
}

// TestProfileOverlay_View_FreeBadge verifies that a free-tier user sees the ○ Free badge.
func TestProfileOverlay_View_FreeBadge(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user456",
		DisplayName: "Free User",
		Product:     "free",
		Country:     "US",
	})

	view := overlay.View()

	assert.Contains(t, view, "○", "View should render ○ for Free users")
	assert.Contains(t, view, "Free", "View should render 'Free' text for free users")
}

// TestProfileOverlay_View_ShowsCountry verifies that View() renders the country code.
func TestProfileOverlay_View_ShowsCountry(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})

	view := overlay.View()

	assert.Contains(t, view, "DE", "View should render the country code")
	assert.Contains(t, view, "◎", "View should render ◎ before country code")
}

// TestProfileOverlay_View_LoadingState verifies that View() renders "Loading profile..."
// when the profile has not yet been loaded (ID is empty).
func TestProfileOverlay_View_LoadingState(t *testing.T) {
	overlay, _ := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	// Store has zero-value UserProfile (ID == "")

	view := overlay.View()

	assert.Contains(t, view, "Loading profile...", "View should show loading placeholder when profile not loaded")
}

// TestProfileOverlay_View_NoEscHint verifies that View() does NOT contain any "esc" or
// "close" text. Esc-to-close is a universal convention; the hint is redundant noise.
func TestProfileOverlay_View_NoEscHint(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})

	view := overlay.View()

	assert.NotContains(t, view, "esc", "View should NOT contain 'esc' hint — universal convention")
	assert.NotContains(t, view, "close", "View should NOT contain 'close' hint — redundant noise")
}

// TestProfileOverlay_View_HasBorderCorners verifies that the overlay uses rounded border corners.
func TestProfileOverlay_View_HasBorderCorners(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})

	view := overlay.View()

	assert.Contains(t, view, "╭", "overlay should use rounded corner ╭")
	assert.Contains(t, view, "╰", "overlay should use rounded corner ╰")
}

// TestProfileOverlay_EscEmitsClosedMsg verifies that Esc key produces ProfileOverlayClosedMsg.
func TestProfileOverlay_EscEmitsClosedMsg(t *testing.T) {
	overlay, _ := newTestProfileOverlay()

	_, cmd := overlay.Update(tea.KeyMsg{Type: tea.KeyEsc})

	require.NotNil(t, cmd, "Esc should return a command")
	msg := cmd()
	_, ok := msg.(ProfileOverlayClosedMsg)
	require.True(t, ok, "Esc should produce ProfileOverlayClosedMsg, got %T", msg)
}

// TestProfileOverlay_OtherKeysIgnored verifies that non-Esc keys do not close the overlay.
func TestProfileOverlay_OtherKeysIgnored(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"rune q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}},
		{"tab", tea.KeyMsg{Type: tea.KeyTab}},
		{"rune j", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlay, _ := newTestProfileOverlay()
			_, cmd := overlay.Update(tt.key)

			// Either cmd is nil, or the command does NOT produce ProfileOverlayClosedMsg.
			if cmd != nil {
				msg := cmd()
				_, isClose := msg.(ProfileOverlayClosedMsg)
				assert.False(t, isClose, "key %q should not produce ProfileOverlayClosedMsg", tt.name)
			}
		})
	}
}

// keyMsg constructs a tea.KeyMsg for a single rune character.
func keyMsg(r string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)}
}

// TestProfileOverlay_logoutFirstPress_showsConfirmation verifies that the first press of 'l'
// emits a confirmation command and does NOT render any inline confirmation text in View()
// (confirmation is delivered via toast — see ProfileConfirmToastMsg).
func TestProfileOverlay_logoutFirstPress_showsConfirmation(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user1",
		DisplayName: "Test User",
		Product:     "premium",
		Country:     "US",
	})

	updated, cmd := overlay.Update(keyMsg("l"))
	model := updated.(*ProfileOverlay)

	assert.NotNil(t, cmd, "first 'l' press should emit a ProfileConfirmToastMsg command")
	view := model.View()
	assert.NotContains(t, view, "Press l again", "confirmation must not be rendered inline; toast handles it")
	assert.Contains(t, view, "logout", "logout label should still render")
}

// TestProfileOverlay_logoutSecondPress_emitsLogoutMsg verifies that two consecutive 'l' presses
// emit ProfileLogoutMsg.
func TestProfileOverlay_logoutSecondPress_emitsLogoutMsg(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user1",
		DisplayName: "Test User",
		Product:     "premium",
		Country:     "US",
	})

	overlay.Update(keyMsg("l"))
	_, cmd := overlay.Update(keyMsg("l"))

	require.NotNil(t, cmd, "second 'l' press should emit a command")
	msg := cmd()
	_, ok := msg.(ProfileLogoutMsg)
	require.True(t, ok, "second 'l' press should produce ProfileLogoutMsg, got %T", msg)
}

// TestProfileOverlay_forgetFirstPress_showsConfirmation verifies that the first press of 'f'
// emits a confirmation command and does NOT render any inline confirmation text in View()
// (confirmation is delivered via toast — see ProfileConfirmToastMsg).
func TestProfileOverlay_forgetFirstPress_showsConfirmation(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user1",
		DisplayName: "Test User",
		Product:     "premium",
		Country:     "US",
	})

	updated, cmd := overlay.Update(keyMsg("f"))
	model := updated.(*ProfileOverlay)

	assert.NotNil(t, cmd, "first 'f' press should emit a ProfileConfirmToastMsg command")
	view := model.View()
	assert.NotContains(t, view, "Press f again", "confirmation must not be rendered inline; toast handles it")
	assert.Contains(t, view, "forget", "forget label should still render")
}

// TestProfileOverlay_forgetSecondPress_emitsForgetMsg verifies that two consecutive 'f' presses
// emit ProfileForgetMsg.
func TestProfileOverlay_forgetSecondPress_emitsForgetMsg(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user1",
		DisplayName: "Test User",
		Product:     "premium",
		Country:     "US",
	})

	overlay.Update(keyMsg("f"))
	_, cmd := overlay.Update(keyMsg("f"))

	require.NotNil(t, cmd, "second 'f' press should emit a command")
	msg := cmd()
	_, ok := msg.(ProfileForgetMsg)
	require.True(t, ok, "second 'f' press should produce ProfileForgetMsg, got %T", msg)
}

// TestProfileOverlay_differentKeyAfterFirstPress_cancelsAndArmsNew verifies that pressing
// 'l' then 'f' cancels the logout confirmation and arms the forget confirmation instead.
func TestProfileOverlay_differentKeyAfterFirstPress_cancelsAndArmsNew(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user1",
		DisplayName: "Test User",
		Product:     "premium",
		Country:     "US",
	})

	overlay.Update(keyMsg("l"))                 // arm logout
	updated, cmd := overlay.Update(keyMsg("f")) // different key: cancel + arm forget
	model := updated.(*ProfileOverlay)

	// 'f' arms forget — now emits a ProfileConfirmToastMsg.
	assert.NotNil(t, cmd, "pressing 'f' after 'l' should emit a ProfileConfirmToastMsg")
	assert.Equal(t, profileActionForget, model.pendingAction, "pendingAction should be armed for forget")

	// Confirmation no longer renders inline — verify toast text from cmd.
	toast, ok := cmd().(ProfileConfirmToastMsg)
	require.True(t, ok, "cmd should produce ProfileConfirmToastMsg, got %T", cmd())
	assert.Contains(t, toast.Text, "confirm forget")
}

// TestFetchCurrentUserRequestMsg_Exists verifies that FetchCurrentUserRequestMsg
// is defined and can be instantiated as a tea.Msg.
func TestFetchCurrentUserRequestMsg_Exists(t *testing.T) {
	msg := FetchCurrentUserRequestMsg{}
	var _ tea.Msg = msg // must satisfy the tea.Msg interface
}

// TestUserProfileLoadedMsg_Exists verifies that UserProfileLoadedMsg
// is defined and its Err field can be set.
func TestUserProfileLoadedMsg_Exists(t *testing.T) {
	msg := UserProfileLoadedMsg{Err: nil}
	var _ tea.Msg = msg
	msg = UserProfileLoadedMsg{Err: errors.New("fail")}
	assert.NotNil(t, msg.Err)
}

// TestProfileOverlay_Init_EmitsFetchWhenStoreEmpty verifies that Init() returns
// a FetchCurrentUserRequestMsg command when the store has no user profile loaded.
func TestProfileOverlay_Init_EmitsFetchWhenStoreEmpty(t *testing.T) {
	overlay, _ := newTestProfileOverlay()
	// Store has zero-value UserProfile (ID == "")
	cmd := overlay.Init()
	require.NotNil(t, cmd, "Init() should return a non-nil command when profile is empty")
	msg := cmd()
	_, ok := msg.(FetchCurrentUserRequestMsg)
	require.True(t, ok, "Init() command should produce FetchCurrentUserRequestMsg, got %T", msg)
}

// TestProfileOverlay_Init_NilWhenProfilePresent verifies that Init() returns nil
// when the store already has a user profile loaded.
func TestProfileOverlay_Init_NilWhenProfilePresent(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	store.SetUserProfile(domain.UserProfile{ID: "user1", DisplayName: "Test"})
	cmd := overlay.Init()
	assert.Nil(t, cmd, "Init() should return nil when profile is already loaded")
}

// TestProfileOverlay_Update_StoresError verifies that Update() handles
// UserProfileLoadedMsg by storing the error on the overlay.
func TestProfileOverlay_Update_StoresError(t *testing.T) {
	overlay, _ := newTestProfileOverlay()
	overlay.SetSize(40, 12)

	testErr := errors.New("network error")
	updated, cmd := overlay.Update(UserProfileLoadedMsg{Err: testErr})
	p, ok := updated.(*ProfileOverlay)
	require.True(t, ok, "Update should return *ProfileOverlay")
	assert.NotNil(t, p.Err(), "Err() should be non-nil after error message")
	assert.Equal(t, testErr, p.Err(), "Err() should match the error from the message")
	assert.Nil(t, cmd, "Update should return nil command for UserProfileLoadedMsg")
}

// TestProfileOverlay_Update_ClearsErrorOnSuccess verifies that receiving a
// UserProfileLoadedMsg with nil error clears a previously stored error.
func TestProfileOverlay_Update_ClearsErrorOnSuccess(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)

	// Set error first.
	updated, _ := overlay.Update(UserProfileLoadedMsg{Err: errors.New("fail")})
	overlay = updated.(*ProfileOverlay)
	assert.NotNil(t, overlay.Err(), "should have error after failed load")

	// Now store the profile and send success.
	store.SetUserProfile(domain.UserProfile{ID: "user1", DisplayName: "Test"})
	updated, _ = overlay.Update(UserProfileLoadedMsg{Err: nil})
	overlay = updated.(*ProfileOverlay)
	assert.Nil(t, overlay.Err(), "Err() should be nil after successful load")
}

// TestProfileOverlay_View_ErrorState verifies that View() renders
// "Profile unavailable" and "Check your connection." when err is set.
func TestProfileOverlay_View_ErrorState(t *testing.T) {
	overlay, _ := newTestProfileOverlay()
	overlay.SetSize(40, 12)

	updated, _ := overlay.Update(UserProfileLoadedMsg{Err: errors.New("network error")})
	p := updated.(*ProfileOverlay)

	view := p.View()
	plain := stripANSI(view)
	assert.Contains(t, plain, "Profile unavailable", "View should show 'Profile unavailable' on error")
	assert.Contains(t, plain, "Check your connection", "View should show hint on error")
}

// TestProfileOverlay_View_NoInfiniteLoading verifies that View() does NOT show
// "Loading profile..." when err is set — the error state takes precedence.
func TestProfileOverlay_View_NoInfiniteLoading(t *testing.T) {
	overlay, _ := newTestProfileOverlay()
	overlay.SetSize(40, 12)

	updated, _ := overlay.Update(UserProfileLoadedMsg{Err: errors.New("network error")})
	p := updated.(*ProfileOverlay)

	view := p.View()
	assert.NotContains(t, view, "Loading profile", "View must not show 'Loading profile' when err is set")
	assert.NotContains(t, view, "Loading", "View must not show any 'Loading' text when err is set")
}

// TestProfileOverlay_SetSize verifies that SetSize stores the dimensions.
func TestProfileOverlay_SetSize(t *testing.T) {
	overlay, _ := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	assert.Equal(t, 40, overlay.width, "SetSize should store width")
	assert.Equal(t, 12, overlay.height, "SetSize should store height")
}

// TestProfileOverlay_SetTheme verifies that after calling SetTheme the overlay renders
// without panicking and reflects the new theme (smoke test).
func TestProfileOverlay_SetTheme(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user1",
		DisplayName: "Theme Switcher",
		Product:     "premium",
		Country:     "US",
	})

	// Switch to a different theme.
	newTheme := theme.Load("dracula")
	overlay.SetTheme(newTheme)

	// View must not panic and must still render content.
	view := overlay.View()
	assert.NotEmpty(t, view, "View should return non-empty content after SetTheme")
	assert.Contains(t, view, "Theme Switcher", "overlay should render profile name after SetTheme")
}

// TestProfileOverlay_logoutFirstPress_emitsToastMsg verifies that the first 'l' press
// emits a ProfileConfirmToastMsg containing "confirm logout".
func TestProfileOverlay_logoutFirstPress_emitsToastMsg(t *testing.T) {
	store := state.New()
	pane := NewProfileOverlay(store, theme.Load("black"))

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	require.NotNil(t, cmd)
	msg := cmd()
	toast, ok := msg.(ProfileConfirmToastMsg)
	require.True(t, ok, "first 'l' press should produce ProfileConfirmToastMsg, got %T", msg)
	assert.Contains(t, toast.Text, "confirm logout")
}

// TestProfileOverlay_forgetFirstPress_emitsToastMsg verifies that the first 'f' press
// emits a ProfileConfirmToastMsg containing "confirm forget".
func TestProfileOverlay_forgetFirstPress_emitsToastMsg(t *testing.T) {
	store := state.New()
	pane := NewProfileOverlay(store, theme.Load("black"))

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	require.NotNil(t, cmd)
	msg := cmd()
	toast, ok := msg.(ProfileConfirmToastMsg)
	require.True(t, ok, "first 'f' press should produce ProfileConfirmToastMsg, got %T", msg)
	assert.Contains(t, toast.Text, "confirm forget")
}

// TestProfileOverlay_confirmationView_noInlineText verifies that the overlay does not
// render any inline confirmation prompt after first 'l' press — the toast handles it.
func TestProfileOverlay_confirmationView_noInlineText(t *testing.T) {
	store := state.New()
	store.SetUserProfile(domain.UserProfile{
		ID:          "user1",
		DisplayName: "Test User",
		Product:     "premium",
		Country:     "US",
	})
	pane := NewProfileOverlay(store, theme.Load("black"))
	pane.SetSize(40, 12)

	updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	view := updated.(*ProfileOverlay).View()
	assert.NotContains(t, view, "Press l again", "overlay must not render inline confirmation; toast handles it")
	assert.NotContains(t, view, "!!")
}

// newPremiumProfileStore creates a state.Store pre-loaded with a premium user profile
// (DisplayName "Irshad", Country "IN", Product "premium"). Reused by table-layout tests.
func newPremiumProfileStore() *state.Store {
	st := state.New()
	st.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Irshad",
		Product:     "premium",
		Country:     "IN",
	})
	return st
}

// TestProfileOverlay_View_TableLayout verifies the three icon+value rows
// render in order (Name, Plan, Region), each as glyph + two-space gap + value,
// inside the single bordered card.
func TestProfileOverlay_View_TableLayout(t *testing.T) {
	st := newPremiumProfileStore()
	p := NewProfileOverlay(st, theme.Load("black"))
	p.SetSize(80, 40)

	plain := stripANSI(p.View())

	// Three rows in order. Glyphs use unicode mode (default).
	assert.Regexp(t, `◉\s+Irshad`, plain, "name row: ◉ + Irshad")
	assert.Regexp(t, `♛\s+Premium`, plain, "plan row: ♛ + Premium")
	assert.Regexp(t, `◎\s+IN`, plain, "region row: ◎ + IN")

	// Order check: Name appears before Plan appears before Region.
	idxName := strings.Index(plain, "Irshad")
	idxPlan := strings.Index(plain, "Premium")
	idxRegion := strings.Index(plain, "IN")
	require.True(t, idxName >= 0 && idxPlan >= 0 && idxRegion >= 0, "all three rows must render")
	assert.Less(t, idxName, idxPlan, "Name above Plan")
	assert.Less(t, idxPlan, idxRegion, "Plan above Region")
}

// TestProfileOverlay_View_KeyBarLine verifies the action area renders as a
// single uikit.KeyBar line, not as two stacked rows.
func TestProfileOverlay_View_KeyBarLine(t *testing.T) {
	st := newPremiumProfileStore()
	p := NewProfileOverlay(st, theme.Load("black"))
	p.SetSize(80, 40)

	plain := stripANSI(p.View())

	// Both bindings on the same logical line, separated by the KeyBar middot.
	assert.Regexp(t, `l\s+logout\s*[·|]\s*f\s+forget`, plain,
		"l logout and f forget must appear on one line separated by the KeyBar separator")

	// Old stacked-row format must be gone.
	assert.NotContains(t, plain, "l  Logout", "old stacked Logout row must be removed")
	assert.NotContains(t, plain, "f  Forget", "old stacked Forget row must be removed")
}

// TestProfileOverlay_View_ASCIIGlyphs validates the glyph system wiring by
// verifying ASCII-mode renders the ASCII variants (per uikit.GlyphFor).
func TestProfileOverlay_View_ASCIIGlyphs(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	st := newPremiumProfileStore() // "Irshad", "IN", premium
	p := NewProfileOverlay(st, theme.Load("black"))
	p.SetSize(80, 40)

	plain := stripANSI(p.View())

	assert.Regexp(t, `\(\*\)\s+Irshad`, plain, "name row: (*) in ASCII mode")
	assert.Regexp(t, `\*P\s+Premium`, plain, "plan row: *P in ASCII mode")
	assert.Regexp(t, `\( \)\s+IN`, plain, "region row: ( ) in ASCII mode")
}

// TestProfileOverlay_View_NoSeparators verifies the horizontal rule lines
// used as section separators in the old implementation are gone.
func TestProfileOverlay_View_NoSeparators(t *testing.T) {
	st := newPremiumProfileStore()
	p := NewProfileOverlay(st, theme.Load("black"))
	p.SetSize(80, 40)

	plain := stripANSI(p.View())

	// The old implementation rendered "│────────────────────" content separator
	// lines (20 dashes framed by the border │). The new layout has none.
	// We check the framed form to distinguish content lines from border corners
	// (the bottom border ╰──────────────────────╯ has more than 20 dashes and
	// no leading │).
	assert.NotContains(t, plain, "│────────────────────",
		"the 20-char content separator lines must be removed")
}

// TestProfilePane_AsciiBorder verifies that the ProfileOverlay border in ASCII mode
// uses ASCII corner characters ('+') and '|' verticals, with no unicode box-drawing
// characters (╭╮╰╯─│) present anywhere in the output.
func TestProfilePane_AsciiBorder(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	st := newPremiumProfileStore()
	p := NewProfileOverlay(st, theme.Load("black"))
	p.SetSize(40, 12)

	out := p.View()
	require.NotEmpty(t, out, "ProfileOverlay.View must produce output")

	// ASCII mode: all six unicode box-drawing characters must be absent.
	for _, ch := range []string{"╭", "╮", "╰", "╯", "─", "│"} {
		assert.NotContains(t, out, ch, "unicode glyph %q must not appear in ASCII mode", ch)
	}
	// ASCII mode: '+' corners must be present.
	assert.Contains(t, out, "+", "'+' corners must appear in ASCII mode border")
	// ASCII mode: '|' vertical rules must be present.
	assert.Contains(t, out, "|", "'|' vertical rules must appear in ASCII mode")
}
