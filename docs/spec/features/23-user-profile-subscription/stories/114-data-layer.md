---
title: "Data Layer — UserProfile Expansion and Profile Pipeline"
feature: 23-user-profile-subscription
status: done
---

## Background

`domain.UserProfile` currently holds only `ID string`. The `GET /me` Spotify endpoint returns
`display_name`, `product`, and `country` in addition to `id`, but these fields are discarded.

The Store holds `userID string` directly instead of the full profile. As a consequence there is
no way to check subscription tier without a second API call.

This story expands the struct, updates the Store, and wires the existing startup command so the
full profile reaches the Store — without adding any new API calls or changing any existing
call site that uses `store.UserID()`.

## Design

### `internal/domain/types.go`

Replace the `UserProfile` struct with:

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

### `testdata/fixtures/me_profile.json`

Create a fixture for the expanded `GET /me` response:

```json
{
  "id": "user123",
  "display_name": "Test User",
  "product": "premium",
  "country": "DE",
  "email": "test@example.com"
}
```

### `internal/api/user_test.go`

Extend `TestUserClient_Profile_Success` to assert `DisplayName`, `Product`, and `Country` from
the fixture. Load the fixture via `os.ReadFile("../../testdata/fixtures/me_profile.json")`.

### `internal/state/store.go`

Replace the `userID string` field with `userProfile domain.UserProfile`. Replace `SetUserID` /
`UserID` with the following four methods:

```go
// UserID returns the Spotify user ID. Returns "" before profile is loaded.
// Preserved for call-site compatibility — delegates to userProfile.ID.
func (s *Store) UserID() string

// UserProfile returns the full authenticated user profile.
// Returns a zero-value UserProfile before profile is loaded.
func (s *Store) UserProfile() domain.UserProfile

// SetUserProfile stores the authenticated user's full Spotify profile.
// Called once at startup after GET /v1/me succeeds.
func (s *Store) SetUserProfile(p domain.UserProfile)

// IsPremium returns true only when Product == "premium".
// Returns false for free users, unknown tier, or when profile not yet loaded.
func (s *Store) IsPremium() bool
```

All four methods must lock/unlock `s.mu` appropriately.

### `internal/app/app.go`

Replace `userProfileLoadedMsg`:

```go
// userProfileLoadedMsg is sent when the initial GET /v1/me fetch completes.
type userProfileLoadedMsg struct {
    profile domain.UserProfile
    err     error
}
```

### `internal/app/commands.go`

Update `buildFetchCurrentUserCmd` to return the full `domain.UserProfile` in the message payload
instead of only the user ID:

```go
return userProfileLoadedMsg{profile: profile}
```

No change to the method signature or the `UserAPI` interface.

### `internal/app/routing.go` — `userProfileLoadedMsg` handler

Replace `a.store.SetUserID(m.userID)` with `a.store.SetUserProfile(m.profile)`. All other
handler logic (error toast, forwarding `UserProfileReadyMsg` to `PanePlaylists`) is unchanged.

## Acceptance Criteria

- [ ] `domain.UserProfile` has `DisplayName`, `Product`, `Country` with correct JSON tags
- [ ] `testdata/fixtures/me_profile.json` exists with `id`, `display_name`, `product`, `country`
- [ ] `TestUserClient_Profile_Success` asserts all four fields
- [ ] `store.SetUserProfile` / `store.UserProfile()` round-trip correctly
- [ ] `store.IsPremium()` returns `true` for `"premium"`, `false` for `"free"`, `false` for `""`
- [ ] `store.UserID()` still returns the profile ID (no existing callers break)
- [ ] `userProfileLoadedMsg` carries `profile domain.UserProfile`, not `userID string`
- [ ] `buildFetchCurrentUserCmd` passes the full profile in the message
- [ ] `routing.go` handler calls `SetUserProfile` instead of `SetUserID`
- [ ] All existing tests that reference `userProfileLoadedMsg{userID: ...}` are updated
- [ ] `go build ./...` clean; `make ci` passes

## Tasks

- [ ] Create `testdata/fixtures/me_profile.json`
      - test: file parseable as JSON with all four fields
- [ ] Extend `TestUserClient_Profile_Success` in `internal/api/user_test.go` to assert the new
      fields; run `go test ./internal/api/... -run TestUserClient_Profile_Success -v` → FAIL
- [ ] Add `DisplayName`, `Product`, `Country` to `domain.UserProfile` in `types.go`
      - test: same test now → PASS
- [ ] Add `TestStore_SetGetUserProfile` and `TestStore_IsPremium` (table-driven) to
      `internal/state/store_test.go`; run `go test ./internal/state/... -run "TestStore_SetGet|TestStore_IsPremium" -v` → FAIL
- [ ] Replace `userID string` with `userProfile domain.UserProfile` in `store.go`; add
      `UserProfile()`, `SetUserProfile()`, `IsPremium()`; update `UserID()` to delegate
      - test: store tests → PASS; `go build ./...` clean (no `SetUserID` call sites break — `UserID()` is the only external method used elsewhere)
- [ ] Update `userProfileLoadedMsg` in `app.go` to carry `profile domain.UserProfile`
- [ ] Update `buildFetchCurrentUserCmd` in `commands.go` to return full profile
- [ ] Update `userProfileLoadedMsg` handler in `routing.go` to call `SetUserProfile`
- [ ] Fix any remaining compile errors (existing test files referencing old `userID` field)
      - test: `go test ./internal/app/... -run TestUserProfileLoaded -v` → PASS
- [ ] `make ci` passes
