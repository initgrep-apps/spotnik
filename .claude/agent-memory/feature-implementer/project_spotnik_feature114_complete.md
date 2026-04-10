---
name: project_spotnik_feature114_complete
description: Story 114 (UserProfile Data Layer): domain expansion, store replacement, app pipeline update, patterns established
type: project
---

## Story 114 — UserProfile Data Layer Expansion

**What was built:**
- Expanded `domain.UserProfile` with `DisplayName`, `Product`, `Country` fields + doc comments
- Replaced `userID string` in `Store` with `userProfile domain.UserProfile`
- Added `UserProfile()`, `SetUserProfile()`, `IsPremium()` to Store; preserved `UserID()` via delegation
- Updated `userProfileLoadedMsg` to carry `profile domain.UserProfile` instead of `userID string`
- Updated `buildFetchCurrentUserCmd` to return full profile
- Updated routing.go handler to call `SetUserProfile` instead of `SetUserID`
- Created `testdata/fixtures/me_profile.json`

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/domain/types.go` — UserProfile struct (lines 18-30)
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/state/store.go` — new methods at lines 159-187
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/app.go` — userProfileLoadedMsg definition (line 215)
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/commands.go` — buildFetchCurrentUserCmd (line 640)
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/routing.go` — handler at line 297
- `/Users/irshadsheikh/dev/github/apps/spotnik/testdata/fixtures/me_profile.json` — fixture for GET /me

**Patterns established:**
- `api.UserProfile` is a type alias for `domain.UserProfile` (in api/models.go) — assignable directly
- Store methods return UserProfile by value (not pointer) — consistent with other store accessors
- IsPremium() defaults to false for zero-value (before profile loads) — safe default pattern
- The `UserID()` method is preserved as a delegation shim for call-site compatibility

**Gotchas:**
- The existing `TestStore_SetGetUserID` test called `SetUserID()` directly — needed updating to `SetUserProfile(domain.UserProfile{ID: "..."})`
- `buildFetchCurrentUserCmd` doc comment still said "carries the user's Spotify ID" after the change — caught in review, fixed
- `domain` import needed to be added to app.go when userProfileLoadedMsg type changed

**Testing notes:**
- Store tests use `domain.UserProfile{}` zero-value comparison for initial state check
- API test uses `os.ReadFile("../../testdata/fixtures/me_profile.json")` relative path (test runs from package dir)
- IsPremium table test: 4 cases (premium, free, empty, unexpected)
- App routing tests (in package `app` not `app_test`) use unexported `userProfileLoadedMsg` directly
