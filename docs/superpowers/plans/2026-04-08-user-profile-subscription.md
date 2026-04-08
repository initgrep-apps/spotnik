# User Profile & Subscription Awareness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fetch the full `GET /me` response at startup, display user name and subscription tier in the header and a `u`-triggered overlay, and gate Premium-only playback keys before any API call is made.

**Architecture:** Expand `domain.UserProfile` with `DisplayName`, `Product`, `Country`; replace the Store's bare `userID string` with `userProfile domain.UserProfile`; wire `IsPremium()` into the key handler as a soft gate; add a `ProfileOverlay` pane composed via bubbletea-overlay following the same pattern as `DeviceOverlay`.

**Tech Stack:** Go 1.22, Bubble Tea v0.27+, Lip Gloss, bubbletea-overlay (`github.com/rmhubbert/bubbletea-overlay`), testify, httptest.

**Spec:** `docs/superpowers/specs/2026-04-08-user-profile-subscription-design.md`

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `internal/domain/types.go` | Add `DisplayName`, `Product`, `Country` to `UserProfile` |
| Create | `testdata/fixtures/me_profile.json` | `GET /me` response fixture |
| Modify | `internal/api/user_test.go` | Extend `TestUserClient_Profile_Success` to assert new fields |
| Modify | `internal/state/store.go` | Replace `userID string` with `userProfile domain.UserProfile`; add `IsPremium()`, `UserProfile()`, `SetUserProfile()`; preserve `UserID()` |
| Modify | `internal/state/store_test.go` | Tests for new store methods |
| Modify | `internal/app/app.go` | Expand `userProfileLoadedMsg`; add `profileOverlayOpen bool`, `profilePane *panes.ProfileOverlay` |
| Modify | `internal/app/commands.go` | `buildFetchCurrentUserCmd` returns full profile |
| Modify | `internal/app/routing.go` | Update `userProfileLoadedMsg` handler; add `u` key; add profile overlay guard; add premium gate; update 403 message; add `TransferPlaybackMsg` premium check |
| Create | `internal/ui/panes/profile.go` | `ProfileOverlay` struct + `View()`, `Update()`, `Init()` |
| Modify | `internal/ui/panes/messages.go` | Add `ProfileOverlayClosedMsg{}` |
| Create | `internal/ui/panes/profile_test.go` | `ProfileOverlay` rendering and behaviour tests |
| Modify | `internal/app/render.go` | `renderProfileChip()`; update `renderHeader()` right side; `renderWithProfileOverlay()`; update `buildView()` compositing |
| Modify | `internal/app/splash.go` | Add static Premium notice to `renderSplashView()` |
| Modify | `internal/app/splash_test.go` | Assert Premium notice present |
| Modify | `internal/app/user_profile_test.go` | Extend for full profile; add premium gate tests |
| Modify | `docs/DESIGN.md` | Add `u` to keybinding table |

---

## Task 1: Feature Branch

- [ ] **Step 1: Check the next available feature number**

```bash
ls docs/spec/features/ | sort
```

The last directory is `20-playback-context`. Use feature **21**.

- [ ] **Step 2: Create the feature branch**

```bash
git checkout main && git pull origin main
git checkout -b feat/21-user-profile-subscription
```

- [ ] **Step 3: Create the feature spec skeleton**

```bash
mkdir -p docs/spec/features/21-user-profile-subscription/stories
```

Create `docs/spec/features/21-user-profile-subscription/feature.md`:

```markdown
---
title: User Profile & Subscription Awareness
status: in-progress
---

# User Profile & Subscription Awareness

Fetch the authenticated user's full profile at startup.
Display name and tier in the header and a profile overlay.
Gate Premium-only operations before any API call.
```

Create `docs/spec/features/21-user-profile-subscription/stories/01-user-profile-subscription.md`:

```markdown
---
title: User Profile & Subscription Awareness
feature: 21-user-profile-subscription
status: in-progress
---

See design spec: docs/superpowers/specs/2026-04-08-user-profile-subscription-design.md
```

- [ ] **Step 4: Commit skeleton**

```bash
git add docs/spec/features/21-user-profile-subscription/
git commit -m "chore(spec): scaffold feature 21 user profile & subscription"
```

---

## Task 2: Expand UserProfile Domain Type + Fixture + API Test

**Files:**
- Modify: `internal/domain/types.go:18-21`
- Create: `testdata/fixtures/me_profile.json`
- Modify: `internal/api/user_test.go:177-192`

- [ ] **Step 1: Write the failing test — extend `TestUserClient_Profile_Success`**

In `internal/api/user_test.go`, replace the existing `TestUserClient_Profile_Success` (lines 177-192) with:

```go
// TestUserClient_Profile_Success verifies that Profile returns all fields from GET /me.
func TestUserClient_Profile_Success(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/me_profile.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := newUserClient(srv.URL)
	profile, err := client.Profile(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "user123", profile.ID)
	assert.Equal(t, "Test User", profile.DisplayName)
	assert.Equal(t, "premium", profile.Product)
	assert.Equal(t, "DE", profile.Country)
}
```

- [ ] **Step 2: Create the fixture**

Create `testdata/fixtures/me_profile.json`:

```json
{
  "id": "user123",
  "display_name": "Test User",
  "product": "premium",
  "country": "DE",
  "email": "test@example.com"
}
```

- [ ] **Step 3: Run the test — expect failure**

```bash
cd /path/to/spotnik && go test ./internal/api/... -run TestUserClient_Profile_Success -v
```

Expected: FAIL — `profile.DisplayName` is empty (field not yet on struct).

- [ ] **Step 4: Add fields to `domain.UserProfile`**

In `internal/domain/types.go`, replace the `UserProfile` struct (lines 16-21):

```go
// UserProfile represents the authenticated user's Spotify profile,
// as returned by GET /v1/me.
type UserProfile struct {
	// ID is the Spotify user ID. Used to distinguish owned vs followed playlists.
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Product     string `json:"product"` // "premium" or "free"
	Country     string `json:"country"` // ISO 3166-1 alpha-2, e.g. "DE"
}
```

- [ ] **Step 5: Run the test — expect pass**

```bash
go test ./internal/api/... -run TestUserClient_Profile_Success -v
```

Expected: PASS.

- [ ] **Step 6: Run full test suite to confirm no regressions**

```bash
go test ./internal/api/... -v 2>&1 | tail -20
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/domain/types.go testdata/fixtures/me_profile.json internal/api/user_test.go
git commit -m "feat(profile): expand UserProfile with DisplayName, Product, Country"
```

---

## Task 3: Expand Store

**Files:**
- Modify: `internal/state/store.go:107-171`
- Modify: `internal/state/store_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/state/store_test.go` (after existing tests):

```go
func TestStore_SetGetUserProfile(t *testing.T) {
	s := New()

	// Initially empty.
	assert.Equal(t, "", s.UserID(), "UserID should be empty before profile is set")
	assert.Equal(t, domain.UserProfile{}, s.UserProfile())

	p := domain.UserProfile{
		ID:          "user-abc",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	}
	s.SetUserProfile(p)

	assert.Equal(t, "user-abc", s.UserID(), "UserID() must return the profile's ID")
	got := s.UserProfile()
	assert.Equal(t, "user-abc", got.ID)
	assert.Equal(t, "Irshad Sheikh", got.DisplayName)
	assert.Equal(t, "premium", got.Product)
	assert.Equal(t, "DE", got.Country)
}

func TestStore_IsPremium(t *testing.T) {
	tests := []struct {
		name    string
		product string
		want    bool
	}{
		{"premium product", "premium", true},
		{"free product", "free", false},
		{"empty product", "", false},
		{"unexpected value", "student", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			s.SetUserProfile(domain.UserProfile{ID: "u1", Product: tt.product})
			assert.Equal(t, tt.want, s.IsPremium())
		})
	}
}
```

- [ ] **Step 2: Run the tests — expect failure**

```bash
go test ./internal/state/... -run "TestStore_SetGetUserProfile|TestStore_IsPremium" -v
```

Expected: FAIL — `SetUserProfile`, `UserProfile()`, `IsPremium()` undefined.

- [ ] **Step 3: Update `store.go`**

In `internal/state/store.go`, replace lines 107-110 (the `userID` field):

```go
	// userProfile holds the authenticated user's Spotify profile.
	// Set once at startup via GET /v1/me.
	userProfile domain.UserProfile
```

Replace lines 157-171 (the `UserID()` and `SetUserID()` methods) with:

```go
// UserID returns the Spotify user ID of the authenticated user.
// Returns "" if the profile has not yet been fetched.
// Preserved for call-site compatibility — delegates to UserProfile().ID.
func (s *Store) UserID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userProfile.ID
}

// UserProfile returns the full authenticated user profile.
// Returns a zero-value UserProfile if not yet fetched.
func (s *Store) UserProfile() domain.UserProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userProfile
}

// SetUserProfile stores the authenticated user's full Spotify profile.
// Called once at startup after fetching GET /v1/me.
func (s *Store) SetUserProfile(p domain.UserProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userProfile = p
}

// IsPremium returns true if the authenticated user has a Spotify Premium subscription.
// Returns false for free users, unknown tier, or when profile is not yet loaded.
func (s *Store) IsPremium() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userProfile.Product == "premium"
}
```

- [ ] **Step 4: Run the new tests — expect pass**

```bash
go test ./internal/state/... -run "TestStore_SetGetUserProfile|TestStore_IsPremium" -v
```

Expected: PASS.

- [ ] **Step 5: Run full state tests to confirm no regressions**

```bash
go test ./internal/state/... -v 2>&1 | tail -20
```

Expected: all PASS. If any test calls `SetUserID` or references `userID` directly, update those call sites now. Run `grep -r "SetUserID" .` to find them.

- [ ] **Step 6: Commit**

```bash
git add internal/state/store.go internal/state/store_test.go
git commit -m "feat(store): replace userID with full UserProfile; add IsPremium()"
```

---

## Task 4: Wire Full Profile Through Command → Message → Routing

**Files:**
- Modify: `internal/app/app.go:217-222` (`userProfileLoadedMsg`)
- Modify: `internal/app/commands.go:639-657` (`buildFetchCurrentUserCmd`)
- Modify: `internal/app/routing.go:280-295` (`userProfileLoadedMsg` handler)
- Modify: `internal/app/user_profile_test.go`

- [ ] **Step 1: Write the failing test**

In `internal/app/user_profile_test.go`, add (inside `package app`):

```go
// TestUserProfileLoadedMsg_StoresFullProfile verifies all profile fields are written to store.
func TestUserProfileLoadedMsg_StoresFullProfile(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})

	p := domain.UserProfile{
		ID:          "user-xyz",
		DisplayName: "Test Name",
		Product:     "premium",
		Country:     "US",
	}
	a.Update(userProfileLoadedMsg{profile: p})

	stored := a.store.UserProfile()
	assert.Equal(t, "user-xyz", stored.ID)
	assert.Equal(t, "Test Name", stored.DisplayName)
	assert.Equal(t, "premium", stored.Product)
	assert.Equal(t, "US", stored.Country)
}
```

Also ensure the existing tests in that file still compile — they reference `userProfileLoadedMsg{userID: ...}`. You will fix those in Step 3.

- [ ] **Step 2: Run the test — expect compile failure**

```bash
go test ./internal/app/... -run TestUserProfileLoadedMsg_StoresFullProfile -v
```

Expected: compile error — `userProfileLoadedMsg` field mismatch.

- [ ] **Step 3: Update `userProfileLoadedMsg` in `app.go`**

In `internal/app/app.go`, replace lines 217-222:

```go
// userProfileLoadedMsg is sent when the initial GET /v1/me fetch completes.
// profile holds all parsed fields; err is non-nil on failure.
type userProfileLoadedMsg struct {
	profile domain.UserProfile
	err     error
}
```

Make sure `domain` is imported in `app.go`. Check existing imports — it likely already is since `domain.PlaybackState` is used.

- [ ] **Step 4: Update `buildFetchCurrentUserCmd` in `commands.go`**

In `internal/app/commands.go`, replace lines 635-657:

```go
// buildFetchCurrentUserCmd fetches the authenticated user's Spotify profile via
// GET /v1/me. The returned userProfileLoadedMsg carries the full profile so the
// store can serve IsPremium(), UserProfile(), and UserID().
func (a *App) buildFetchCurrentUserCmd() tea.Cmd {
	userAPI := a.userAPI
	return func() tea.Msg {
		if userAPI == nil {
			return userProfileLoadedMsg{err: errNilClient}
		}
		profile, err := userAPI.Profile(context.Background())
		if err != nil {
			if isUnauthorizedError(err) {
				return unauthorizedMsg{}
			}
			return userProfileLoadedMsg{err: err}
		}
		return userProfileLoadedMsg{profile: profile}
	}
}
```

- [ ] **Step 5: Update the `userProfileLoadedMsg` routing handler in `routing.go`**

In `internal/app/routing.go`, replace lines 280-295 (`case userProfileLoadedMsg:`):

```go
case userProfileLoadedMsg:
	if m.err != nil {
		if errors.Is(m.err, errNilClient) {
			// Programming error — userAPI was nil at startup; no toast.
			return a, nil, true
		}
		return a, a.alerts.NewAlertCmd("warning", "Could not load your Spotify profile."), true
	}
	if m.profile.ID != "" {
		a.store.SetUserProfile(m.profile)
		// Refresh playlist rows so the ~ prefix appears immediately.
		return a, a.forwardToPane(layout.PanePlaylists, panes.UserProfileReadyMsg{}), true
	}
	return a, nil, true
```

- [ ] **Step 6: Fix remaining compile errors**

Run `go build ./...` to find any remaining call sites referencing the old `userProfileLoadedMsg{userID: ...}` or `store.SetUserID(...)`. Fix them:
- In `user_profile_test.go`: update existing tests to use `userProfileLoadedMsg{profile: domain.UserProfile{ID: "..."}}`.

- [ ] **Step 7: Run the new test — expect pass**

```bash
go test ./internal/app/... -run TestUserProfileLoadedMsg -v
```

Expected: all PASS.

- [ ] **Step 8: Run full app tests**

```bash
go test ./internal/app/... 2>&1 | tail -20
```

Expected: all PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/app/app.go internal/app/commands.go internal/app/routing.go internal/app/user_profile_test.go
git commit -m "feat(profile): wire full UserProfile through command → message → store"
```

---

## Task 5: ProfileOverlay Pane

**Files:**
- Modify: `internal/ui/panes/messages.go`
- Create: `internal/ui/panes/profile.go`
- Create: `internal/ui/panes/profile_test.go`

- [ ] **Step 1: Add `ProfileOverlayClosedMsg` to `messages.go`**

Open `internal/ui/panes/messages.go`. Find `DeviceOverlayClosedMsg` for reference — add after it:

```go
// ProfileOverlayClosedMsg is emitted by ProfileOverlay when the user presses Esc.
// The root app handles this by setting profileOverlayOpen = false.
type ProfileOverlayClosedMsg struct{}
```

- [ ] **Step 2: Write failing tests for `ProfileOverlay`**

Create `internal/ui/panes/profile_test.go`:

```go
package panes_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newProfileTestStore(p domain.UserProfile) *state.Store {
	s := state.New()
	s.SetUserProfile(p)
	return s
}

func newProfileOverlay(s *state.Store) *panes.ProfileOverlay {
	t := theme.Load("black")
	o := panes.NewProfileOverlay(s, t)
	o.SetSize(40, 12)
	return o
}

func TestProfileOverlay_View_ShowsDisplayName(t *testing.T) {
	s := newProfileTestStore(domain.UserProfile{
		ID:          "u1",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})
	o := newProfileOverlay(s)
	view := o.View()
	assert.Contains(t, view, "Irshad Sheikh")
}

func TestProfileOverlay_View_PremiumBadge(t *testing.T) {
	s := newProfileTestStore(domain.UserProfile{ID: "u1", Product: "premium"})
	o := newProfileOverlay(s)
	view := o.View()
	assert.Contains(t, view, "♛")
	assert.Contains(t, view, "Premium")
}

func TestProfileOverlay_View_FreeBadge(t *testing.T) {
	s := newProfileTestStore(domain.UserProfile{ID: "u1", Product: "free"})
	o := newProfileOverlay(s)
	view := o.View()
	assert.Contains(t, view, "○")
	assert.Contains(t, view, "Free")
}

func TestProfileOverlay_View_ShowsCountry(t *testing.T) {
	s := newProfileTestStore(domain.UserProfile{ID: "u1", Country: "DE"})
	o := newProfileOverlay(s)
	view := o.View()
	assert.Contains(t, view, "DE")
}

func TestProfileOverlay_View_LoadingState(t *testing.T) {
	s := state.New() // no profile set
	o := newProfileOverlay(s)
	view := o.View()
	assert.Contains(t, view, "Loading")
}

func TestProfileOverlay_EscEmitsClosedMsg(t *testing.T) {
	s := newProfileTestStore(domain.UserProfile{ID: "u1"})
	o := newProfileOverlay(s)
	updated, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd, "Esc must return a close command")
	msg := cmd()
	_, ok := msg.(panes.ProfileOverlayClosedMsg)
	assert.True(t, ok, "Esc must emit ProfileOverlayClosedMsg, got %T", msg)
	_ = updated
}
```

- [ ] **Step 3: Run the tests — expect compile failure**

```bash
go test ./internal/ui/panes/... -run TestProfileOverlay -v
```

Expected: compile error — `panes.ProfileOverlay`, `panes.NewProfileOverlay` undefined.

- [ ] **Step 4: Create `internal/ui/panes/profile.go`**

```go
package panes

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// ProfileOverlay renders the authenticated user's profile as a floating overlay.
// It reads directly from the Store — no local data copy needed.
// Triggered by the 'u' key; closed by Esc.
type ProfileOverlay struct {
	store  *state.Store
	theme  theme.Theme
	width  int
	height int
}

// NewProfileOverlay constructs a ProfileOverlay with the given store and theme.
func NewProfileOverlay(store *state.Store, t theme.Theme) *ProfileOverlay {
	return &ProfileOverlay{store: store, theme: t}
}

// SetSize sets the overlay dimensions. Called by the root app on resize.
func (p *ProfileOverlay) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Init satisfies tea.Model. No commands needed — data comes from the store.
func (p *ProfileOverlay) Init() tea.Cmd { return nil }

// Update handles Esc to close the overlay.
func (p *ProfileOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m, ok := msg.(tea.KeyMsg); ok && m.Type == tea.KeyEsc {
		return p, func() tea.Msg { return ProfileOverlayClosedMsg{} }
	}
	return p, nil
}

// View renders the profile card. Reads from store — pure function.
func (p *ProfileOverlay) View() string {
	t := p.theme
	w := p.width
	if w < 36 {
		w = 36
	}
	innerW := w - 4 // 2 padding each side

	profile := p.store.UserProfile()

	if profile.ID == "" {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderFocused()).
			Padding(1, 2).
			Width(w).
			Render(lipgloss.NewStyle().Foreground(t.TextMuted()).Render("Loading profile..."))
	}

	nameStr := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.TextPrimary()).
		Render(profile.DisplayName)

	sep := lipgloss.NewStyle().
		Foreground(t.TextMuted()).
		Render(strings.Repeat("─", innerW))

	var tierStr string
	if p.store.IsPremium() {
		icon := lipgloss.NewStyle().Foreground(t.Info()).Render("♛")
		label := lipgloss.NewStyle().Foreground(t.Info()).Render("Premium")
		tierStr = icon + "  " + label
	} else {
		icon := lipgloss.NewStyle().Foreground(t.TextMuted()).Render("○")
		label := lipgloss.NewStyle().Foreground(t.TextMuted()).Render("Free")
		tierStr = icon + "  " + label
	}

	countryIcon := lipgloss.NewStyle().Foreground(t.TextMuted()).Render("◎")
	countryLabel := lipgloss.NewStyle().Foreground(t.TextFg()).Render(profile.Country)
	countryStr := countryIcon + "  " + countryLabel

	hint := lipgloss.NewStyle().Foreground(t.TextMuted()).Render("esc  close")

	body := lipgloss.JoinVertical(lipgloss.Left,
		nameStr,
		sep,
		tierStr,
		countryStr,
		"",
		hint,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocused()).
		Padding(1, 2).
		Width(w).
		Render(body)
}
```

- [ ] **Step 5: Run the tests — expect pass**

```bash
go test ./internal/ui/panes/... -run TestProfileOverlay -v
```

Expected: all PASS.

- [ ] **Step 6: Run full pane tests to confirm no regressions**

```bash
go test ./internal/ui/panes/... 2>&1 | tail -20
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/ui/panes/profile.go internal/ui/panes/profile_test.go internal/ui/panes/messages.go
git commit -m "feat(profile): add ProfileOverlay pane with Premium/Free badge"
```

---

## Task 6: Wire ProfileOverlay Into App

**Files:**
- Modify: `internal/app/app.go` (struct fields, constructor)
- Modify: `internal/app/routing.go` (`u` key, overlay guard, `ProfileOverlayClosedMsg` handler, mouse guard)
- Modify: `internal/app/render.go` (`renderWithProfileOverlay`, `buildView`)

- [ ] **Step 1: Add fields to `App` struct and constructor**

In `internal/app/app.go`, find the `App` struct. Add after `deviceOverlayOpen bool`:

```go
profileOverlayOpen bool
profilePane        *panes.ProfileOverlay
```

In the `New()` constructor, find where `devicePane` is initialised. After it, add:

```go
profilePane: panes.NewProfileOverlay(store, t),
```

(Where `store` and `t` are the store and theme already in scope in the constructor. Check adjacent `devicePane` initialisation for the exact variable names.)

- [ ] **Step 2: Add `u` key handler in `routing.go`**

In `internal/app/routing.go` `handleKeyMsg`, find the `'d'` global shortcut block (around line 101):

```go
// 'd' opens the device switcher overlay from any pane.
if m.Type == tea.KeyRunes && string(m.Runes) == "d" {
    return a.openDeviceOverlay()
}
```

Add immediately after:

```go
// 'u' opens the user profile overlay.
if m.Type == tea.KeyRunes && string(m.Runes) == "u" {
    a.profileOverlayOpen = true
    return a, nil
}
```

- [ ] **Step 3: Add overlay routing guard in `handleKeyMsg`**

In `routing.go` `handleKeyMsg`, find the device overlay guard (around line 53):

```go
// When device overlay is open, route all keys to the device pane.
if a.deviceOverlayOpen {
    updated, cmd := a.devicePane.Update(m)
    ...
}
```

Add immediately after that block:

```go
// When profile overlay is open, route all keys to it.
if a.profileOverlayOpen {
    updated, cmd := a.profilePane.Update(m)
    if pp, ok := updated.(*panes.ProfileOverlay); ok {
        a.profilePane = pp
    }
    return a, cmd
}
```

- [ ] **Step 4: Handle `ProfileOverlayClosedMsg` in routing**

In `routing.go`, in the message dispatch switch (where `DeviceOverlayClosedMsg` is handled), add:

```go
case panes.ProfileOverlayClosedMsg:
    a.profileOverlayOpen = false
    return a, nil, true
```

- [ ] **Step 5: Suppress mouse events when profile overlay is open**

In `routing.go`, `handleMouseMsg`, find:

```go
if a.deviceOverlayOpen || a.searchOpen {
    return nil
}
```

Replace with:

```go
if a.deviceOverlayOpen || a.searchOpen || a.profileOverlayOpen {
    return nil
}
```

- [ ] **Step 6: Add `renderWithProfileOverlay` to `render.go`**

In `internal/app/render.go`, add after `renderWithDeviceOverlay`:

```go
// renderWithProfileOverlay composites the profile overlay over the dimmed background.
// Positioned in the top-right corner to mirror the header chip location.
func (a *App) renderWithProfileOverlay(background string) string {
	fg := a.profilePane.View()
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return background
	}
	// btoverlay.Composite(fg, bg, horizAlign, vertAlign, offsetY, offsetX)
	// Same constants as renderWithDeviceOverlay — top-right positioning.
	return btoverlay.Composite(fg, dimmed, btoverlay.Right, btoverlay.Top, 0, 0)
}
```

- [ ] **Step 7: Wire into `buildView` in `render.go`**

In `render.go` `buildView()` (the method that assembles the full view string), find:

```go
if a.deviceOverlayOpen {
    return a.renderWithDeviceOverlay(body)
}
```

Add immediately after:

```go
if a.profileOverlayOpen {
    return a.renderWithProfileOverlay(body)
}
```

- [ ] **Step 8: Propagate size to profile pane on resize**

In `app.go`, find the `tea.WindowSizeMsg` handler where `a.devicePane.SetSize(...)` is called. Add the same call for profile pane:

```go
a.profilePane.SetSize(40, 12) // fixed size — profile card is not resizable
```

- [ ] **Step 9: Build and confirm no compile errors**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 10: Run app tests**

```bash
go test ./internal/app/... 2>&1 | tail -20
```

Expected: all PASS.

- [ ] **Step 11: Commit**

```bash
git add internal/app/app.go internal/app/routing.go internal/app/render.go
git commit -m "feat(profile): wire ProfileOverlay into App — u key, routing guard, render"
```

---

## Task 7: Header Profile Chip

**Files:**
- Modify: `internal/app/render.go` (`renderProfileChip`, `renderHeader`)

- [ ] **Step 1: Add `renderProfileChip()` to `render.go`**

In `internal/app/render.go`, add after `renderHeader`:

```go
// renderProfileChip renders the user name + tier badge for the header right side.
// Returns "" if the profile has not yet been loaded (graceful startup).
// Name is truncated to 20 chars to avoid overflowing narrow terminals.
func (a *App) renderProfileChip() string {
	p := a.store.UserProfile()
	if p.ID == "" {
		return ""
	}
	name := p.DisplayName
	if len([]rune(name)) > 20 {
		name = string([]rune(name)[:19]) + "…"
	}
	nameStr := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.TextFg()).
		Render(name)
	var badge string
	if a.store.IsPremium() {
		badge = lipgloss.NewStyle().
			Background(a.theme.StatusBarBg()).
			Foreground(a.theme.Info()).
			Render("♛")
	} else {
		badge = lipgloss.NewStyle().
			Background(a.theme.StatusBarBg()).
			Foreground(a.theme.TextMuted()).
			Render("○")
	}
	return nameStr + " " + badge + " "
}
```

- [ ] **Step 2: Update `renderHeader()` right side**

In `render.go` `renderHeader()`, find the right-side block (around line 367):

```go
// Right side: device indicator.
device := a.store.ActiveDevice()
var right string
if device != nil {
    name := truncateDeviceName(device.Name)
    activeStyle := lipgloss.NewStyle().
        Background(a.theme.StatusBarBg()).
        Foreground(a.theme.DeviceActive())
    right = activeStyle.Render("◉ " + name + " ")
} else {
    right = bgStyle.Render("○ No device ")
}
```

Replace with:

```go
// Right side: device chip then profile chip (profile is rightmost).
device := a.store.ActiveDevice()
var deviceChip string
if device != nil {
    name := truncateDeviceName(device.Name)
    activeStyle := lipgloss.NewStyle().
        Background(a.theme.StatusBarBg()).
        Foreground(a.theme.DeviceActive())
    deviceChip = activeStyle.Render("◉ " + name + " ")
} else {
    deviceChip = bgStyle.Render("○ No device ")
}
right := deviceChip + a.renderProfileChip()
```

- [ ] **Step 3: Build and smoke-test visually**

```bash
go build -o bin/spotnik . && ./bin/spotnik
```

Log in; header right side should show `◉ DeviceName   YourName ♛` (or `○` for Free).

- [ ] **Step 4: Run render tests to confirm no regressions**

```bash
go test ./internal/app/... -run "TestRender\|TestHeader\|TestSplash" -v 2>&1 | tail -30
```

Expected: all PASS. If any test checks exact header output, update the expected string to include the new chip format (when profile is empty the chip returns "" so existing tests may be unaffected).

- [ ] **Step 5: Commit**

```bash
git add internal/app/render.go
git commit -m "feat(profile): add profile chip to header — name and tier badge"
```

---

## Task 8: Subscription Gate — Playback Keys

**Files:**
- Modify: `internal/app/routing.go` (`isPlaybackKey` branch)
- Modify: `internal/app/user_profile_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/app/user_profile_test.go` (inside `package app`):

```go
// TestPremiumGate_FreeUser_PlaybackKeyEmitsToast verifies that a free-tier user
// pressing a playback key gets a warning toast instead of a playback command.
func TestPremiumGate_FreeUser_PlaybackKeyEmitsToast(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})
	// Set free user profile.
	a.store.SetUserProfile(domain.UserProfile{ID: "u1", Product: "free"})

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	require.NotNil(t, cmd, "free user pressing Space must return a cmd (toast)")
	// Execute the cmd; it should produce an alert message, not a PlaybackCmdSentMsg.
	msg := cmd()
	_, isPlayback := msg.(panes.PlaybackCmdSentMsg)
	assert.False(t, isPlayback, "free user Space must not dispatch playback command, got %T", msg)
}

// TestPremiumGate_PremiumUser_PlaybackKeyDispatches verifies that a premium user
// pressing a playback key dispatches normally (no gate intercept).
func TestPremiumGate_PremiumUser_PlaybackKeyDispatches(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})
	// Set premium user profile.
	a.store.SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	require.NotNil(t, cmd, "premium user pressing Space must return a cmd")
}
```

Add necessary imports at top of file if not present:
```go
import (
    "testing"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/initgrep-apps/spotnik/internal/config"
    "github.com/initgrep-apps/spotnik/internal/domain"
    "github.com/initgrep-apps/spotnik/internal/ui/panes"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)
```

- [ ] **Step 2: Run the tests — expect failure**

```bash
go test ./internal/app/... -run TestPremiumGate -v
```

Expected: FAIL — free user dispatches playback command (gate not yet implemented).

- [ ] **Step 3: Add premium gate in `routing.go`**

In `routing.go` `handleKeyMsg`, find the `isPlaybackKey` block (around line 156):

```go
// Playback keys always go to the NowPlaying pane regardless of focus.
if isPlaybackKey(m) {
    np := a.nowPlayingPane()
    if np == nil {
        return a, nil
    }
    ...
    return a, cmd
}
```

Replace with:

```go
// Playback keys always go to the NowPlaying pane regardless of focus.
// Gate: free-tier users cannot use playback controls.
if isPlaybackKey(m) {
    if !a.store.IsPremium() {
        return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required")
    }
    np := a.nowPlayingPane()
    if np == nil {
        return a, nil
    }
    wasFocused := np.IsFocused()
    if !wasFocused {
        np.SetFocused(true)
    }
    updatedPane, cmd := np.Update(m)
    if pp, ok := updatedPane.(*panes.NowPlayingPane); ok {
        a.panes[layout.PaneNowPlaying] = pp
        np = pp
    }
    if !wasFocused {
        np.SetFocused(false)
    }
    return a, cmd
}
```

- [ ] **Step 4: Run the tests — expect pass**

```bash
go test ./internal/app/... -run TestPremiumGate -v
```

Expected: all PASS.

- [ ] **Step 5: Run full app tests to confirm no regressions**

```bash
go test ./internal/app/... 2>&1 | tail -20
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/app/routing.go internal/app/user_profile_test.go
git commit -m "feat(subscription): gate playback keys for free-tier users"
```

---

## Task 9: Subscription Gate — Transfer Playback + 403 Safety Net

**Files:**
- Modify: `internal/app/app.go` (`TransferPlaybackMsg` handler at line ~1709, `PlaybackCmdSentMsg` handler at line ~1351)
- Modify: `internal/app/user_profile_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/app/user_profile_test.go`:

```go
// TestPremiumGate_FreeUser_TransferPlaybackEmitsToast verifies that a free-tier user
// selecting a device gets a warning toast instead of a transfer command.
func TestPremiumGate_FreeUser_TransferPlaybackEmitsToast(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{})
	a.store.SetUserProfile(domain.UserProfile{ID: "u1", Product: "free"})

	_, cmd := a.Update(panes.TransferPlaybackMsg{DeviceID: "dev1", DeviceName: "Speaker"})
	require.NotNil(t, cmd, "free user TransferPlaybackMsg must return a cmd")
	msg := cmd()
	// Must not be a device transfer result — that would mean the API call went out.
	_, isTransfer := msg.(panes.DeviceTransferredMsg)
	assert.False(t, isTransfer, "free user must not dispatch device transfer, got %T", msg)
}
```

- [ ] **Step 2: Run the test — expect failure**

```bash
go test ./internal/app/... -run TestPremiumGate_FreeUser_TransferPlaybackEmitsToast -v
```

Expected: FAIL — transfer dispatches API call for free users.

- [ ] **Step 3: Add premium gate in `TransferPlaybackMsg` handler in `app.go`**

In `internal/app/app.go`, find the `TransferPlaybackMsg` handler (around line 1709):

```go
case panes.TransferPlaybackMsg:
    // User selected a device; show info toast and dispatch transfer API call.
    a.deviceOverlayOpen = false
    return a, tea.Batch(
        a.buildTransferPlaybackCmd(m.DeviceID),
        a.alerts.NewAlertCmd("info", fmt.Sprintf("Switching to %s...", m.DeviceName)),
    )
```

Replace with:

```go
case panes.TransferPlaybackMsg:
    a.deviceOverlayOpen = false
    if !a.store.IsPremium() {
        return a, a.alerts.NewAlertCmd("warning", "Spotify Premium required")
    }
    // User selected a device; show info toast and dispatch transfer API call.
    return a, tea.Batch(
        a.buildTransferPlaybackCmd(m.DeviceID),
        a.alerts.NewAlertCmd("info", fmt.Sprintf("Switching to %s...", m.DeviceName)),
    )
```

- [ ] **Step 4: Improve 403 message in `PlaybackCmdSentMsg` handler in `app.go`**

In `internal/app/app.go`, find the `PlaybackCmdSentMsg` handler (around line 1351):

```go
var forbiddenErr *api.ForbiddenError
if errors.As(m.Err, &forbiddenErr) {
    return a, tea.Batch(
        fetchPlaybackStateCmd(a.player),
        a.alerts.NewAlertCmd("warning", "Playback control not available on this device"),
    )
}
```

Replace the toast message:

```go
var forbiddenErr *api.ForbiddenError
if errors.As(m.Err, &forbiddenErr) {
    return a, tea.Batch(
        fetchPlaybackStateCmd(a.player),
        a.alerts.NewAlertCmd("warning", "Spotify Premium required"),
    )
}
```

- [ ] **Step 5: Run all premium gate tests — expect pass**

```bash
go test ./internal/app/... -run TestPremiumGate -v
```

Expected: all PASS.

- [ ] **Step 6: Run full app tests**

```bash
go test ./internal/app/... 2>&1 | tail -20
```

Expected: all PASS. If `TestApp_PlaybackCmdSentMsg_ForbiddenEmitsWarningToast` or similar test checks the exact toast string, update it to `"Spotify Premium required"`.

- [ ] **Step 7: Commit**

```bash
git add internal/app/app.go internal/app/user_profile_test.go
git commit -m "feat(subscription): gate TransferPlayback; improve 403 toast message"
```

---

## Task 10: Splash Screen Premium Notice

**Files:**
- Modify: `internal/app/splash.go`
- Modify: `internal/app/splash_test.go`

- [ ] **Step 1: Write the failing test**

In `internal/app/splash_test.go`, add:

```go
func TestRenderSplash_ContainsPremiumNotice(t *testing.T) {
	th := theme.Load("black")
	view := renderSplashView(th, 120, 40)
	assert.Contains(t, view, "Playback controls require")
	assert.Contains(t, view, "Spotify Premium")
}
```

- [ ] **Step 2: Run the test — expect failure**

```bash
go test ./internal/app/... -run TestRenderSplash_ContainsPremiumNotice -v
```

Expected: FAIL — notice not yet in splash.

- [ ] **Step 3: Update `renderSplashView` in `splash.go`**

In `internal/app/splash.go`, find `renderSplashView`. Replace the `content` assembly:

```go
notice := lipgloss.NewStyle().
    Foreground(t.TextMuted()).
    Render("♛ Playback controls require Spotify Premium")

content := lipgloss.JoinVertical(lipgloss.Center,
    bannerStyle.Render(banner),
    "",
    tagline,
    version,
    "",
    notice,
)
```

- [ ] **Step 4: Run the test — expect pass**

```bash
go test ./internal/app/... -run TestRenderSplash -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/splash.go internal/app/splash_test.go
git commit -m "feat(splash): add static Spotify Premium notice"
```

---

## Task 11: Update DESIGN.md + CI Gate

**Files:**
- Modify: `docs/DESIGN.md`

- [ ] **Step 1: Add `u` to the keybinding table in `docs/DESIGN.md`**

Find the global keybindings section (§17 or wherever the keybinding table lives). Add:

```
| u    | Open user profile overlay (global) |
```

Place it near `d` (devices) since both are overlay triggers.

- [ ] **Step 2: Run `make ci` — must pass fully**

```bash
make ci
```

Expected: lint PASS, tests PASS, coverage ≥ 80%. Fix any failures before proceeding.

- [ ] **Step 3: Commit DESIGN.md**

```bash
git add docs/DESIGN.md
git commit -m "docs(design): add u keybinding for profile overlay"
```

- [ ] **Step 4: Push and open PR**

```bash
git push origin feat/21-user-profile-subscription
```

Open a PR titled: `feat(profile): user profile display & subscription awareness`

PR body should include:
- Summary: what the feature does (profile overlay, header chip, premium gate, splash notice)
- Test plan: which commands to run and what to verify manually
- Note: `make ci` passed

---

## Self-Review Against Spec

| Spec Requirement | Task |
|-----------------|------|
| `UserProfile` gains `DisplayName`, `Product`, `Country` | Task 2 |
| Store replaces `userID` with `userProfile`; adds `IsPremium()`, `UserProfile()`, `SetUserProfile()` | Task 3 |
| `buildFetchCurrentUserCmd` returns full profile | Task 4 |
| `userProfileLoadedMsg` carries `domain.UserProfile` | Task 4 |
| `UserID()` call-site compatibility preserved | Task 3 |
| `ProfileOverlayClosedMsg` added to `messages.go` | Task 5 |
| `ProfileOverlay` shows name, tier badge, country, hint | Task 5 |
| Loading state when profile empty | Task 5 |
| `u` key opens overlay; `Esc` closes it | Tasks 5, 6 |
| Overlay composited via bubbletea-overlay top-right | Task 6 |
| Mouse events suppressed when overlay open | Task 6 |
| Header: device chip then profile chip (rightmost) | Task 7 |
| Profile chip omitted if not yet loaded | Task 7 |
| `♛`/`○` badges with correct theme colors | Tasks 5, 7 |
| Premium gate in `isPlaybackKey` branch | Task 8 |
| Premium gate in `TransferPlaybackMsg` handler | Task 9 |
| 403 message improved to "Spotify Premium required" | Task 9 |
| Splash static Premium notice | Task 10 |
| `u` added to `docs/DESIGN.md` keybinding table | Task 11 |
| `make ci` passes (lint + tests + 80% coverage) | Task 11 |
