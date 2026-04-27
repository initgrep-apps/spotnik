---
name: project_spotnik_feature114_complete
description: Story 114 (UserProfile Data Layer): domain expansion, store replacement, app pipeline update, patterns established
type: project
---

## Story 114 — UserProfile Data Layer Expansion

**Built:**
- Expanded `domain.UserProfile` w/ `DisplayName`, `Product`, `Country` fields + doc comments
- Replaced `userID string` in `Store` w/ `userProfile domain.UserProfile`
- Added `UserProfile()`, `SetUserProfile()`, `IsPremium()` to Store; kept `UserID()` via delegation
- Updated `userProfileLoadedMsg` carry `profile domain.UserProfile` not `userID string`
- Updated `buildFetchCurrentUserCmd` return full profile
- Updated routing.go handler call `SetUserProfile` not `SetUserID`
- Created `testdata/fixtures/me_profile.json`

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/domain/types.go` — UserProfile struct (lines 18-30)
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/state/store.go` — new methods lines 159-187
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/app.go` — userProfileLoadedMsg def (line 215)
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/commands.go` — buildFetchCurrentUserCmd (line 640)
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/routing.go` — handler line 297
- `/Users/irshadsheikh/dev/github/apps/spotnik/testdata/fixtures/me_profile.json` — fixture GET /me

**Patterns:**
- `api.UserProfile` = type alias for `domain.UserProfile` (api/models.go) — assignable direct
- Store methods return UserProfile by value not pointer — matches other store accessors
- IsPremium() defaults false for zero-value (pre-load) — safe default
- `UserID()` kept as delegation shim for call-site compat

**Gotchas:**
- Existing `TestStore_SetGetUserID` test called `SetUserID()` direct — needed update to `SetUserProfile(domain.UserProfile{ID: "..."})`
- `buildFetchCurrentUserCmd` doc still said "carries the user's Spotify ID" post-change — caught review, fixed
- `domain` import needed in app.go when userProfileLoadedMsg type changed

**Testing:**
- Store tests use `domain.UserProfile{}` zero-value compare for initial state
- API test uses `os.ReadFile("../../testdata/fixtures/me_profile.json")` relative path (test runs from package dir)
- IsPremium table test: 4 cases (premium, free, empty, unexpected)
- App routing tests (package `app` not `app_test`) use unexported `userProfileLoadedMsg` direct