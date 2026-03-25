# Known Issues & Technical Debt

> Tracked issues from PR reviews. Items here are non-blocking but should be
> addressed in future features or cleanup passes.

---

## From PR #34 — Elm Purity: Data-Carrying Messages (2026-03-25)

### Documentation

- [x] **store.go package doc stale** — Fixed in Feature 34: updated package doc and Store struct doc to reflect Elm purity rule.
- [x] **Store struct doc stale** — Fixed in Feature 34: updated to "the root app.Update() writes to it via Msg payloads."

### Dead Code

- [x] **Dead `unmarshalJSON` in api/models.go** — Fixed in Feature 34: removed helper; inlined json.Unmarshal in the one remaining caller (SearchPlaylist.UnmarshalJSON in search.go).

### Error Handling (pre-existing)

- [x] **`PlaybackStateFetchedMsg.Err` never checked** — Fixed in Feature 36: `consecutivePlaybackErrors` counter added to App struct. Toast emitted on exactly the 5th consecutive error; counter resets on success.
- [x] **`buildFetchDevicesCmd` error fallthrough** — Verified in Feature 34: already handles errors correctly with early return. No fix needed.
- [x] **Nil-client fallbacks return empty messages with no error** — Fixed in Feature 36: `errNilClient` sentinel added; all 7 nil-client fallbacks now set `Err: errNilClient`. Update() handlers skip silently (no toast) for this sentinel — it is an expected startup condition.
- [x] **Store reads in `buildPlaybackAPICmd` goroutine closures** — Fixed in Feature 36: store values snapshotted in `buildPlaybackAPICmd` body (Update() context, thread-safe). Closures now use captured values only.

### Test Coverage Gaps

- [x] **`buildSearchCmd` store isolation untested** — Fixed in Feature 39: `TestBuildSearchCmd_DoesNotWriteToStore` added to elm_purity_test.go.
- [x] **`SearchResultsMsg` error path missing from elm_purity_test.go** — Fixed in Feature 39: `TestSearchResultsMsg_ErrorPath` and `TestSearchResultsMsg_ClearPath` added.
- [x] **Concurrent stats partial failure untested** — Fixed in Feature 39: `TestStatsLoadedMsg_PartialFailure` added.

### Type Design

- [x] **Inconsistent message encapsulation** — Fixed in Feature 35: `devicesLoadedMsg` exported to `DevicesLoadedMsg`, moved to messages.go, constructor removed. Store mutations moved from DeviceOverlay.Update() to root app.Update().
- [x] **store.go still imports `api/` for `SearchResult`** — Fixed in Feature 35: `SearchResult` and supporting types moved to `internal/domain/search.go`. Type aliases in `api/models.go` for backward compat. `state/store.go` no longer imports `api/`.
- [x] **`StatsLoadedMsg` defined in stats.go, not messages.go** — Fixed in Feature 35: moved to messages.go alongside all other shared message types.
- [x] **`AlbumsLoadedMsg` missing Offset field** — Fixed in Feature 35: `Offset int` field added, handler updated to append vs replace like LibraryLoadedMsg and LikedTracksLoadedMsg.

---

## From PR #35 — API Gateway (2026-03-25)

### Error Handling

- [x] **Double 429 parsing with inconsistent error wrapping** — Fixed in Feature 37: extracted `parseRetryAfter` helper shared by gateway.go and errors.go. Gateway sets backoff and creates `RateLimitError` directly so dedup waiters receive consistent errors. Body always cloned for all responses.
- [x] **`doNoContent` discards `io.ReadAll` error** (pre-existing) — Fixed in Feature 37: `body, readErr := io.ReadAll(resp.Body)` now checked; returns `fmt.Errorf("reading response body: %w", readErr)` on failure.
- [x] **Unparseable `Retry-After` header silently defaults** — Fixed in Feature 37: `parseRetryAfter` documents the intentional behaviour with a comment explaining HTTP-date format is not supported and the 5s default is used.

### Thread Safety

- [x] **`SetGateway` not thread-safe** — Fixed in Feature 37: `gateway *Gateway` field changed to `gateway atomic.Pointer[Gateway]`. `SetGateway` uses `.Store()`, all reads use `.Load()`.

### Robustness

- [x] **`time.After` timer leaks on context cancellation** — Fixed in Feature 37: `tokenBucket.wait()` and `waitForBackoff()` now use `time.NewTimer` with explicit `Stop()` on cancellation.
- [x] **nil response from `fn()` causes panic** — Fixed in Feature 37: nil guard added after `fn()` call in `Gateway.Do()`.
- [x] **429 path leaves `resp.Body` unreadable for dedup waiters** — Fixed in Feature 37 as part of Task 5: body is now always cloned for all responses (not just non-429), so dedup waiters always get a readable body.

---

## From PR #36 — Notifications + Error Routing (2026-03-25)

### Test Quality

- [x] **Tests weakened to `cmd != nil`** — Fixed in Feature 39: five tests (LikeToggleResultMsg, PlaybackCmdSentMsg, AddToQueueResultMsg, AddToQueueResultMsg error, DeviceTransfer) now use the two-pass pattern to verify toast content and type.

### Robustness

- [x] **`alerts.Update()` type assertion failure silently ignored** — Fixed in Feature 38: added a defensive comment explaining why the assertion is safe (BubbleUp.AlertModel.Update always returns AlertModel). Assertion failure indicates a BubbleUp library bug — app continues without crashing.
- [x] **`alerts.Init()` return value discarded** — Fixed in Feature 38: `alertsInitCmd := a.alerts.Init()` is now batched into the returned commands in both authenticated and unauthenticated Init() paths.
- [x] **No validation of alert type registration** — Fixed in Feature 38: `TestNewNotifications_AllFiveAlertTypesRegistered` verifies all 5 alert types produce non-nil commands after registration. BubbleUp's `RegisterNewAlertType` is void, so the test is the validation point.

### Consistency

- [x] **`PlaybackStateFetchedMsg` errors still silent** — Fixed in Feature 36: throttled toast after 5 consecutive failures via `consecutivePlaybackErrors` counter.

---

## From PR #37 — Staleness Tracking (2026-03-25)

### Race Conditions

- [x] **TOCTOU race between staleness check and fetchedAt stamp** — Fixed in Feature 38: fetching sentinel fields (`playlistsFetching`, `albumsFetching`, `likedFetching`, `recentFetching`, `statsFetching`, `devicesFetching`) added to Store. Sentinels set before dispatch, cleared in loaded-message handlers. Paginated requests (Offset > 0) bypass sentinels.

### Data Integrity

- [x] **fetchedAt stamped on nil/empty data from nil-client fallbacks** — Fixed in Feature 38: `SetPlaylists`, `SetSavedAlbums`, `SetLikedTracks`, `SetRecentlyPlayed` now only stamp `fetchedAt` when the slice is non-empty. Exception: `SetPlaybackState` where nil is valid (204 = nothing playing).
- [x] **Stats double-stamped** — Fixed in Feature 38: removed `statsFetchedAt` stamping from `SetTopTracks` and `SetTopArtists`. Added `StampStatsFetchedAt(timeRange)` method, called once in the `StatsLoadedMsg` handler after both setters succeed.

### Initialization

- [x] **`statsFetchedAt` map not initialized in `New()`** — Fixed in Feature 34: pre-allocated in New(), removed lazy-init nil guards from SetTopTracks, SetTopArtists, StatsFetchedAt, and StatsStale.

### UX

- [x] **Staleness gate silently drops `FetchPlaylistsRequestMsg`** — Fixed in Feature 38: when playlists (albums, liked tracks, recently-played) are within TTL, a synthetic loaded message carrying cached store data is returned instead of `nil`, so the pane can initialize its list without a redundant API call.

---

## From PR #38 — Idle Polling Backoff (2026-03-25)

### UX

- [x] **Only `tea.KeyMsg` resets idle, not `tea.WindowSizeMsg`** — Fixed in Feature 39: WindowSizeMsg handler now resets lastInteraction and tickCount (when idle) identically to KeyMsg.
- [x] **Backoff + idle-return interaction** — Fixed in Feature 39: KeyMsg handler now emits a "ratelimit" toast with remaining seconds when user returns from idle during an active 429 backoff.

### Observability

- [x] **Nil PlaybackState unlogged** — Fixed in Feature 39: `nilPlaybackStateTicks` counter added. A "warning" toast fires on the 30th consecutive nil-state, reset to 0 on any non-nil state.

---

## From PR #39 Review — Docs & Init Cleanup (2026-03-25)

### Documentation (pre-existing, surfaced by review)

- [x] **DeviceOverlay.Update() writes to Store directly** — Fixed in Feature 35: moved SetDevicesError, ClearDevicesError, SetDevicesFetchedAt from DeviceOverlay.Update() to root app.Update() via DevicesLoadedMsg handler.
- [x] **Store error state comment stale** — Fixed in Feature 35: updated comment to "Set by Update() handlers on failure".
- [x] **Orphaned "After Task 3" TODO** — Fixed in Feature 35: removed the stale comment from store.go.

---

## From PR #40 Review — Type Design Alignment (2026-03-25)

### Documentation

- [x] **SetDevicesFetchedAt comment stale** — Fixed in Feature 36: comment updated to "Called by root app.Update() after a successful DevicesLoadedMsg."
- [x] **ARCHITECTURE.md stale devicesLoadedMsg reference** — Fixed in Feature 36: updated to `DevicesLoadedMsg` with exported type signature.

### Dead Code

- [x] **DevicesLoadErrorMsg now dead** — Fixed in Feature 36: type removed from messages.go, handler removed from app.go, test removed from toast_routing_test.go.

---

## From PR #42 Review — Gateway Hardening (2026-03-25)

### Robustness

- [ ] **Unbounded Retry-After accepted** — `parseRetryAfter` in gateway.go accepts any integer including negative or very large values. A malicious proxy sending `Retry-After: 999999` would cause ~11.5 day backoff. Add bounds: `v > 0 && v <= 300`.
- [ ] **entry.resp set on 429 path** — gateway.go stores both resp and err for dedup waiters on 429 path. Currently safe because waiters check err first, but fragile. Consider setting `entry.resp = nil` when err != nil.

---

## From PR #43 Review — Notification & Staleness Hardening (2026-03-25)

### Design

- [ ] **Synthetic cached messages re-stamp fetchedAt** — Cached data flows through the normal loaded-message handler and calls Set*() which re-stamps fetchedAt. This extends TTL indefinitely if panes periodically re-fire Init(). Consider adding `FromCache: true` flag or stamping only in Update() handler.
- [ ] **fetchedAt len>0 guard blocks empty collections** — Users with genuinely empty libraries (0 playlists, 0 albums) will never get fetchedAt stamped, causing repeated API calls. Distinguish "empty because error" from "empty because user has no data."
- [ ] **Hardcoded time range strings in clearAllFetchingSentinels** — `app.go` iterates `{"short_term", "medium_term", "long_term"}` as literals. Extract to constants to prevent silent sentinel leak on drift.
- [ ] **Pagination response can clear Offset=0 sentinel** — A paginated loaded message (Offset>0) unconditionally clears the fetching sentinel. Narrow window for duplicate Offset=0 fetches during active pagination.
