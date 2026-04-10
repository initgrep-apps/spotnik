
## buildFetchCurrentUserCmd uses non-cancellable context
**Found:** 2026-04-10 | **Source:** PR #147 Review
**Feature:** 23-user-profile-subscription

`buildFetchCurrentUserCmd` in `internal/app/commands.go` calls `userAPI.Profile(api.WithPriority(context.Background(), api.Interactive))` using `context.Background()` rather than a cancellable context tied to the app lifecycle. If the user quits while the fetch is in-flight, the HTTP request will not be cancelled. Pre-existing pattern gap — `buildSearchPageCmd` and others handle this correctly.

---

## No concurrent store test for UserProfile methods
**Found:** 2026-04-10 | **Source:** PR #147 Review
**Feature:** 23-user-profile-subscription

`UserProfile()`, `SetUserProfile()`, and `IsPremium()` have correct mutex locking but `store_test.go` has no goroutine-concurrent test to act as a race-detection regression guard. A short concurrent test calling `SetUserProfile` and `IsPremium` simultaneously would protect against future locking mistakes.

---

## TestUserClient_Profile_Success not table-driven
**Found:** 2026-04-10 | **Source:** PR #147 Review
**Feature:** 23-user-profile-subscription

`internal/api/user_test.go` `TestUserClient_Profile_Success` was extended with new field assertions but remains a flat single-case test. Project convention (CLAUDE.md) requires table-driven style. Acceptable now with one scenario; should be refactored if a second fixture path is added.

---

## StateReader and auth test coverage refinements (PR #146 review findings)
**Found:** 2026-04-10 | **Source:** PR #146 Review
**Feature:** 22-developer-foundations

Items to log:
1. `internal/ui/panes/nowplaying_test.go` — `testStateWriter` embeds full `*state.Store`; a narrower `playbackWriter` interface (just `SetPlaybackState`) would document intent more precisely and prevent future tests from calling unrelated write methods.
2. `internal/api/auth_test.go` — no public-boundary test for nil-client propagation through `ExchangeCode` or `Refresh`; the internal test in `auth_internal_test.go` covers the guard, but a higher-level test would catch any future wrapping regression.

---

## Test coverage gaps in devices.go (PR #145 review findings)
**Found:** 2026-04-10 | **Source:** PR #145 Review
**Feature:** 22-developer-foundations

Items to log:
1. `internal/api/devices_test.go` — `TransferPlayback` with `play: false` is untested; only the `true` path is exercised. Adding a second table row would pin the serialisation contract.
2. `internal/api/devices_test.go` — `{"devices": null}` JSON response not tested; only the `[]` case is covered. A `devices_null.json` fixture and `TestGetDevices_NullDevicesField` would pin the nil-guard on line 49 of devices.go.
3. `internal/testhelpers/fixtures_test.go:22` — `assert.Contains(…, "{")` is a weak JSON validity check; `json.Valid(data)` would be more precise.
