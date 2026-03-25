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

- [ ] **`buildSearchCmd` store isolation untested** — No test verifies the command itself does NOT write to store.
- [ ] **`SearchResultsMsg` error path missing from elm_purity_test.go** — Unlike all other message types, SearchResultsMsg error/clear paths are only tested indirectly.
- [ ] **Concurrent stats partial failure untested** — When TopTracks succeeds but TopArtists fails (or vice versa), the behavior is untested.

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

- [ ] **Tests weakened to `cmd != nil`** — Several tests (LikeToggleResultMsg, PlaybackCmdSentMsg, AddToQueueResultMsg, DeviceTransfer) only assert `cmd != nil` instead of verifying toast content and type. Should use the two-pass pattern: `alertMsg := cmd(); a.Update(alertMsg); assert.Contains(a.View(), "expected text")`.

### Robustness

- [ ] **`alerts.Update()` type assertion failure silently ignored** — `app.go` does `if am, ok := updatedAlerts.(bubbleup.AlertModel); ok` but if assertion fails, alert state updates stop silently. Add defensive logging/comment.
- [ ] **`alerts.Init()` return value discarded** — Currently returns nil by design, but a BubbleUp upgrade could break alert auto-dismiss if Init() starts returning a setup command. Batch it into returned commands.
- [ ] **No validation of alert type registration** — `NewNotifications` calls `RegisterNewAlertType` 5 times but never validates success. Invalid theme color strings could cause silent registration failure.

### Consistency

- [x] **`PlaybackStateFetchedMsg` errors still silent** — Fixed in Feature 36: throttled toast after 5 consecutive failures via `consecutivePlaybackErrors` counter.

---

## From PR #37 — Staleness Tracking (2026-03-25)

### Race Conditions

- [ ] **TOCTOU race between staleness check and fetchedAt stamp** — Duplicate fetches possible when staleness check passes but the async fetch hasn't completed yet. Consider adding a "fetching" sentinel set in Update() immediately when staleness gate passes, cleared when loaded message arrives.

### Data Integrity

- [ ] **fetchedAt stamped on nil/empty data from nil-client fallbacks** — All `Set*()` methods unconditionally stamp `fetchedAt = time.Now()` even when data is nil. If API client is nil, the nil-client fallback returns empty success message, which stamps fetchedAt and prevents retries for the full TTL. Consider guarding: only stamp when data is non-nil.
- [ ] **Stats double-stamped** — Both `SetTopTracks` and `SetTopArtists` independently stamp `statsFetchedAt[range]`. If only one is called, the range appears fresh despite incomplete data. Consider stamping once in the `StatsLoadedMsg` handler after both setters.

### Initialization

- [x] **`statsFetchedAt` map not initialized in `New()`** — Fixed in Feature 34: pre-allocated in New(), removed lazy-init nil guards from SetTopTracks, SetTopArtists, StatsFetchedAt, and StatsStale.

### UX

- [ ] **Staleness gate silently drops `FetchPlaylistsRequestMsg`** — When playlists are within TTL, `return a, nil` swallows the request. LibraryPane's `Init()` expects a `LibraryLoadedMsg` for auto-expand. After re-auth, library pane may show collapsed sections. Consider sending synthetic loaded message with cached data.

---

## From PR #38 — Idle Polling Backoff (2026-03-25)

### UX

- [ ] **Only `tea.KeyMsg` resets idle, not `tea.WindowSizeMsg`** — Terminal resize implies user presence but does not reset idle state. User who resizes but doesn't press keys continues at idle polling rates.
- [ ] **Backoff + idle-return interaction** — If user returns from idle during active 429 backoff, tickCount resets to 0 but backoff guard prevents any fetches. No status indicator shown. User sees stale data with no explanation until backoff expires.

### Observability

- [ ] **Nil PlaybackState unlogged** — `pollIntervals()` silently defaults to "paused" when `store.PlaybackState()` returns nil. If this persists beyond startup, it indicates a bug but produces no log/toast. Consider adding observability after N ticks with nil state.

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
