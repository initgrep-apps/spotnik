# Unresolved Issues

Quick dump ground for issues found during implementation or review.
Triage into feature stories when ready to fix.

---

## Unbounded Retry-After accepted
**Found:** 2026-03-25 | **Source:** PR #42 Review
**Feature:** 11-api-gateway

`parseRetryAfter` in gateway.go accepts any integer including negative or very large values. A malicious proxy sending `Retry-After: 999999` would cause ~11.5 day backoff. Add bounds: `v > 0 && v <= 300`.

---

## entry.resp set on 429 path
**Found:** 2026-03-25 | **Source:** PR #42 Review
**Feature:** 11-api-gateway

gateway.go stores both resp and err for dedup waiters on 429 path. Currently safe because waiters check err first, but fragile. Consider setting `entry.resp = nil` when err != nil.

---

## Synthetic cached messages re-stamp fetchedAt
**Found:** 2026-03-25 | **Source:** PR #43 Review
**Feature:** 11-api-gateway

Cached data flows through the normal loaded-message handler and calls Set*() which re-stamps fetchedAt. This extends TTL indefinitely if panes periodically re-fire Init(). Consider adding `FromCache: true` flag or stamping only in Update() handler.

---

## fetchedAt len>0 guard blocks empty collections
**Found:** 2026-03-25 | **Source:** PR #43 Review
**Feature:** 04-library

Users with genuinely empty libraries (0 playlists, 0 albums) will never get fetchedAt stamped, causing repeated API calls. Distinguish "empty because error" from "empty because user has no data."

---

## Hardcoded time range strings in clearAllFetchingSentinels
**Found:** 2026-03-25 | **Source:** PR #43 Review
**Feature:** 08-stats

`app.go` iterates `{"short_term", "medium_term", "long_term"}` as literals. Extract to constants to prevent silent sentinel leak on drift.

---

## Pagination response can clear Offset=0 sentinel
**Found:** 2026-03-25 | **Source:** PR #43 Review
**Feature:** 04-library

A paginated loaded message (Offset>0) unconditionally clears the fetching sentinel. Narrow window for duplicate Offset=0 fetches during active pagination.

---

## PlaylistsPane `n` key creates with hardcoded "New Playlist"
**Found:** 2026-03-26 | **Source:** PR #52 Review
**Feature:** 09-playlists

Needs textinput integration to collect user-specified name before emitting `PlaylistCreateRequestMsg`. The old `PlaylistManager` had a `textinput.Model` for this.

---

## PlaylistsPane `r` key sends current name as NewName
**Found:** 2026-03-26 | **Source:** PR #52 Review
**Feature:** 09-playlists

`PlaylistRenameRequestMsg` gets `pl.Name` (current name) instead of a new name. Needs textinput integration to collect the new name.

---

## PlaylistsPane Title() calls store.PlaylistTracks() on every render
**Found:** 2026-03-26 | **Source:** PR #52 Review
**Feature:** 09-playlists

Could cache the track count in a field updated in `refreshTrackRows()` instead of reading from store on every `Title()` call.

---

## Playlist deletion (x key) removed
**Found:** 2026-03-26 | **Source:** PR #52 Review
**Feature:** 09-playlists

The `x` key was using `PlaylistRemoveRequestMsg` (track removal) for playlist deletion. Removed since playlist unfollow requires a different message type (`PlaylistUnfollowRequestMsg`). Add proper playlist deletion support when needed.

---

## TopTracksPane "Pop" column always shows "--"
**Found:** 2026-03-26 | **Source:** PR #53 Review
**Feature:** 08-stats

`domain.Track` lacks a `Popularity` field. The Spotify top-tracks API returns popularity, but it's not captured in the domain model. Either add `Popularity int` to `domain.Track` and populate the column, or replace the column with extra width for Track/Artist.

---

## Gateway.Snapshot() is best-effort, not atomic
**Found:** 2026-03-26 | **Source:** PR #56 Review
**Feature:** 11-api-gateway

Token bucket and gateway mutex are acquired separately. Snapshot fields may be from slightly different points in time. Acceptable for display purposes but worth documenting.

---

## PollingSnapshotMsg.TickIntervalMs is misleading
**Found:** 2026-03-26 | **Source:** PR #56 Review
**Feature:** 14-nerd-status

Shows the polling decision interval (3000ms, 10000ms) but the actual tea.Tick fires every 1000ms. Consider renaming to `PollIntervalMs` or displaying the actual tick interval separately.

---

## ARCHITECTURE.md references deleted pane names
**Found:** 2026-03-26 | **Source:** PR #58 Review
**Feature:** 00-architecture

The ASCII diagram at line 33 still shows `LibraryPane`, `PlayerPane`, and `QueuePane`. Test examples at lines 621/628 reference `PlayerPane`. These types no longer exist. Update to reflect the 10-pane grid layout.

---

## formatDuration duplication
**Found:** 2026-03-26 | **Source:** PR #58 Review
**Feature:** 13-nowplaying

`formatDuration` in `gradient.go` and `formatDurationMs` in `nowplaying.go` are duplicate implementations. Extract to a shared utility in `components/`.

---

## Unstyled space characters in device overlay cursor rows
**Found:** 2026-03-31 | **Source:** PR #94 Review
**Feature:** 16-vivid-themes

In `devices.go renderDevice()`, literal `" "` space characters concatenated between styled cursor-row elements carry no `Background(SelectedBg())`. Unlike `themes.go` which wraps the entire row in a `rowStyle` with background, `devices.go` returns raw concatenation. This can create 1-column highlight gaps on cursor rows depending on terminal rendering.

---

## Custom theme silent fallback when missing new fields
**Found:** 2026-04-01 | **Source:** PR #96 Review
**Feature:** 16-vivid-themes

`theme.Load(id)` silently falls back to the default theme if a user-provided TOML fails validation (e.g., missing the `info` field added in story 79). No toast or user-visible feedback explains why their custom theme was rejected. Consider logging when a user theme fails validation.

---

## persistThemeChoice error silently discarded
**Found:** 2026-04-01 | **Source:** PR #96 Review
**Feature:** 16-vivid-themes

In `app.go` ThemeSwitchMsg handler, `a.persistThemeChoice(m.ThemeID)` is called in an anonymous Cmd that returns nil regardless of success. If config file write fails, the theme switch appears successful but the choice is lost on restart.

---

## PersistTheme writes zero-valued Preset/Visualizer fields
**Found:** 2026-04-01 | **Source:** PR #97 Review
**Feature:** 17-bootstrap

When `PersistTheme` updates the config file, it writes `preset = 0` and `visualizer = 0` even if those fields were never set by the user. This pollutes the config with values the user didn't choose. Consider using `omitempty` in the raw TOML struct or a different approach. Note: PersistTheme is being replaced by PreferenceStore in story 79, so this may resolve itself.

---

## ThemeValidator package-level var has no concurrency protection
**Found:** 2026-04-01 | **Source:** PR #97 Review
**Feature:** 17-bootstrap

`config.ThemeValidator` is a mutable `func(string) bool` set in `cmd/root.go` init(). No mutex or sync.Once protects it. Safe today since it's set before any Load() call, but a latent data race if tests use `t.Parallel()` or if Load() is called from multiple goroutines.

---

## DefaultConfigPath silently falls back to CWD
**Found:** 2026-04-01 | **Source:** PR #97 Review
**Feature:** 17-bootstrap

When `os.UserHomeDir()` fails, `DefaultConfigPath()` returns `"config.toml"` (relative to CWD) with no warning. Could cause config to be read/written from unexpected locations in containers or CI.

---

## Visualizer range hardcoded in comment
**Found:** 2026-04-01 | **Source:** PR #97 Review
**Feature:** 17-bootstrap

The `Visualizer` field comment in `PreferencesConfig` says "0-6" which will rot if patterns are added/removed. Should reference the viz engine instead of hardcoding the count.

---

## PreferenceStore exported test-only accessors on App
**Found:** 2026-04-01 | **Source:** PR #98 Review
**Feature:** 17-bootstrap

`Prefs()`, `PrefsDirtyGen()`, and `SchedulePrefsFlush()` are exported on the App struct solely for testing. Should use `export_test.go` pattern to keep the public API clean.

---

## No flush-on-quit for PreferenceStore
**Found:** 2026-04-01 | **Source:** PR #98 Review
**Feature:** 17-bootstrap

If the user changes a preference and quits within the 500ms debounce window, the change is lost. Consider adding a synchronous flush in the quit handler.

---

## CyclePattern lacks empty-patterns guard
**Found:** 2026-04-01 | **Source:** PR #99 Review
**Feature:** 13-nowplaying

`Engine.CyclePattern()` does `(e.patternIdx + 1) % len(e.patterns)` without checking for an empty patterns slice, while the new `SetPattern()` correctly guards against it. Add the same `len(e.patterns) == 0` guard for consistency.

---

## Unbounded prefs flush retry with no user notification
**Found:** 2026-04-01 | **Source:** PR #99 Review
**Feature:** 17-bootstrap

When a prefs flush fails, `handlePrefsMsg` retries via `schedulePrefsFlush()` with no retry limit. On a permanently unwritable config file, this retries every 500ms indefinitely with only stderr logging (invisible in TUI). Consider capping retries and emitting a toast after N failures.

---

## Search Store: Type safety and convenience improvements
**Found:** 2026-04-02 | **Source:** PR #105 Review
**Feature:** 19-search-redesign

1. Introduce `SearchType` string constants (e.g. `SearchTypeTrack`, `SearchTypeAll`) to replace bare strings in `SearchHasMore`, `SearchPageLoadedMsg.Type`, and `SearchState.ActiveType` — prevents typo bugs
2. Consider removing redundant `TypePage.Offset` field — derive from `len(Items)` instead to eliminate consistency hazard
3. Add `Store.ResetForNewQuery(query string)` method that atomically sets query, clears pages, and sets loading — prevents callers from forgetting a step
4. `SearchHasMore` silently returns false for unknown type names — add logging for debugging
5. `convertSearch*Items` functions in commands.go have no direct unit tests — add table-driven tests

---

## Search Prefetch: Dead Type field and toast dedup
**Found:** 2026-04-02 | **Source:** PR #107 Review
**Feature:** 19-search-redesign

1. `SearchPageLoadedMsg.Type` field is always empty string after prefetch engine rewrite — either remove the field or add proper `TODO(84-search-list-delegate):` tag
2. Consider debounce/dedup for search error toasts — rapid failures from batch pages could stack multiple identical toasts

---

## Search Redesign: Post-merge review minor items
**Found:** 2026-04-03 | **Source:** Post-merge review of PRs #108-110
**Feature:** 19-search-redesign

1. `checkPrefetch()` fires on every arrow keypress past threshold — consider tracking last-requested prefetch offset to suppress duplicates (currently deduplicated by SearchLoading guard in app.go)
2. `nextOffsetForTab()` returns 0 instead of -1 when Total == 0 — harmless in production but could cause redundant offset-0 fetch in tests
3. Unknown prefix (`:foo `) sends colon-prefixed string to Spotify API with no user feedback that the prefix was not recognized

---

## Search spinner tick comment: 130ms vs actual 100ms interval
**Found:** 2026-04-04 | **Source:** PR #121 Review
**Feature:** 19-search-redesign

The comment on `searchSpinnerTick()` in `search.go` states "The 130ms interval matches the spinner.Dot tick rate from bubbles/spinner" but `spinner.Dot` actually uses `FPS: time.Second / 10` = 100ms. The 130ms value works fine functionally (just slightly slower animation) but the comment is inaccurate.

---

## Story 105: buildPlayContextCmd nil-check pattern deviation
**Found:** 2026-04-05 | **Source:** PR #132 Review
**Feature:** 20-playback-context

`buildPlayContextCmd` and `buildPlayTrackListCmd` in `internal/app/commands.go` capture `a.player` into a local variable and check for nil inside the closure, whereas the established `buildPlaybackAPICmd` pattern checks nil before creating the closure. Functionally equivalent today but deviates from the project pattern.

---

## Story 105: 403 ForbiddenError toast message for context playback
**Found:** 2026-04-05 | **Source:** PR #132 Review
**Feature:** 20-playback-context

The `PlaybackCmdSentMsg` handler shows "Playback control not available on this device" for 403 errors, which is accurate for device-control actions but misleading when context/list playback fails due to account tier. A more specific message for context play 403s ("Spotify Premium required") would improve UX.

---

## Story 106: `checkPrefetch` integer underflow latent risk
**Found:** 2026-04-05 | **Source:** PR #133 Review
**Feature:** 20-playback-context

`checkPrefetch` in `playlists_pane.go` computed `len(p.loadedTracks)-10` before checking for an empty slice. With `len=0` this evaluates to `-10`, and any cursor position ≥ -10 would pass the guard. Fixed in the PR by adding an empty-slice early return, but worth noting as a pattern to watch in similar paginated panes.

---

## Story 106: `TestBuildFetchPlaylistTracksCmd_CancelledCtx_ReturnsNil` assertion weakness
**Found:** 2026-04-05 | **Source:** PR #133 Review
**Feature:** 20-playback-context

The cancelled-context test wraps its inner assertion in `if msg != nil`, which means the test cannot fail if the cancellation guard is broken and a non-nil msg is returned with incorrect content. Consider hardening to an unconditional `assert.Nil` in a future pass.

---

## Story 107: `TestBuildFetchAlbumTracksCmd_CancelledCtx_ReturnsNil` conditional assertion
**Found:** 2026-04-05 | **Source:** PR #134 Review
**Feature:** 20-playback-context

The cancelled-context test uses `if msg != nil { assert.Equal(...) }` instead of an unconditional `assert.Nil`. If the cancellation guard were broken and returned a non-nil message, the test would pass silently. Consider hardening to `assert.Nil(t, msg)` in a future pass.
