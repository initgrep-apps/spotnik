
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

---

## Duplicate truncation helpers: truncateRunes (panes) vs truncateProfileName (render)
**Found:** 2026-04-10 | **Source:** PR #148 Review
**Feature:** 23-user-profile-subscription

`internal/ui/panes/profile.go` has a private `truncateRunes(s string, max int) string` helper and `internal/app/render.go` has a private `truncateProfileName(s string) string` that wraps the same logic with a hardcoded 20-rune cap. Both functions slice `[]rune` and append `"…"`. They can diverge independently. Consider consolidating into a shared utility (e.g., `ui/components/`) or at minimum exporting one and calling it from both sites.

---

## truncateRunes: long-name truncation branch untested
**Found:** 2026-04-10 | **Source:** PR #148 Review
**Feature:** 23-user-profile-subscription

`truncateRunes` in `internal/ui/panes/profile.go` and `truncateProfileName` in `internal/app/render.go` have no test for the truncation branch (name longer than 20 runes). All existing test profiles use short names. The off-by-one behaviour (truncate to max-1 runes + `…`) is unverified. Add a `TestProfileOverlay_View_TruncatesLongDisplayName` test case with a name exceeding 20 runes and assert the output contains `…` and does not contain the full untruncated name.

---

## isPremiumOnlyPlaybackKey: no direct unit test for key enumeration
**Found:** 2026-04-10 | **Source:** PR #149 Review
**Feature:** 23-user-profile-subscription

`isPremiumOnlyPlaybackKey` in `internal/app/routing.go` is tested indirectly via `TestPremiumGate_FreeUser_PlaybackKeyEmitsToast` (8 keys) and `TestPremiumGate_FreeUser_VisualizerKeyNotBlocked`. A direct table-driven unit test listing every key in `isPlaybackKey` and whether it requires premium would serve as a living specification and catch any future addition to `isPlaybackKey` that doesn't get added to `isPremiumOnlyPlaybackKey`.

---

## PlayContextMsg/PlayTrackListMsg/PlayTrackMsg not premium-gated
**Found:** 2026-04-10 | **Source:** PR #149 Review
**Feature:** 23-user-profile-subscription

`PlayContextMsg`, `PlayTrackListMsg`, and `PlayTrackMsg` handlers in `handlers.go` dispatch Premium-only playback API calls but are not gated by `IsPremium()`. Story 116 intentionally scoped only the operations in the design table. These could be a follow-up story to complete gating coverage for free-tier users navigating search results or queue pane.
