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

- [ ] **`PlaybackStateFetchedMsg.Err` never checked** — The most frequently fired message (every 1-3s). When Err is non-nil, the handler silently skips. Consider tracking consecutive errors and showing a transient notification after N failures.
- [x] **`buildFetchDevicesCmd` error fallthrough** — Verified in Feature 34: already handles errors correctly with early return. No fix needed.
- [ ] **Nil-client fallbacks return empty messages with no error** — Seven command builders return zero-value messages when their API client is nil. Consider returning an error message instead of silent empty data.
- [ ] **Store reads in `buildPlaybackAPICmd` goroutine closures** — Lines 48, 59, 70, 77 in commands.go read store inside goroutines. This is a data race — snapshot values in Update() and pass as parameters.

### Test Coverage Gaps

- [ ] **`buildSearchCmd` store isolation untested** — No test verifies the command itself does NOT write to store.
- [ ] **`SearchResultsMsg` error path missing from elm_purity_test.go** — Unlike all other message types, SearchResultsMsg error/clear paths are only tested indirectly.
- [ ] **Concurrent stats partial failure untested** — When TopTracks succeeds but TopArtists fails (or vice versa), the behavior is untested.

### Type Design

- [ ] **Inconsistent message encapsulation** — `devicesLoadedMsg` uses unexported type + constructor pattern. Other data-carrying messages are fully exported. Consider aligning over time.
- [ ] **store.go still imports `api/` for `SearchResult`** — Breaks the clean domain boundary. Track migration of `SearchResult` to `domain/` package.
- [ ] **`StatsLoadedMsg` defined in stats.go, not messages.go** — Breaks convention that all shared message types live in messages.go.
- [ ] **`AlbumsLoadedMsg` missing Offset field** — Unlike `LibraryLoadedMsg` and `LikedTracksLoadedMsg`, albums messages don't carry pagination offset.

---

## From PR #35 — API Gateway (2026-03-25)

### Error Handling

- [ ] **Double 429 parsing with inconsistent error wrapping** — Gateway converts 429 to `RateLimitError`, then `doJSON` wraps it with "sending request:" prefix. Dedup waiters get unwrapped error. Consider having gateway set backoff only, let `checkResponseStatus` handle the error.
- [ ] **`doNoContent` discards `io.ReadAll` error** (pre-existing) — Line 139 in base.go: `body, _ := io.ReadAll(resp.Body)` silently drops read errors.
- [ ] **Unparseable `Retry-After` header silently defaults** — Non-integer values (e.g., HTTP-date format per RFC 7231) are silently ignored with 5s default. Consider logging parse failures.

### Thread Safety

- [ ] **`SetGateway` not thread-safe** — `base.go` line 57-59 writes to `b.gateway` without synchronization. Could race with concurrent `doJSON` calls during token refresh. Consider `atomic.Pointer[Gateway]`.

### Robustness

- [ ] **`time.After` timer leaks on context cancellation** — `tokenBucket.wait()` and `waitForBackoff()` use `time.After` which leaks timers when ctx is cancelled. Use `time.NewTimer` + explicit `Stop()`.
- [ ] **nil response from `fn()` causes panic** — If HTTP transport returns `(nil, nil)`, gateway will nil-pointer dereference on `resp.Body`. Add nil check after `fn()`.
- [ ] **429 path leaves `resp.Body` unreadable for dedup waiters** — On 429, `entry.resp.Body` is consumed but not replaced with a clone. Currently safe because waiters check `entry.err` first, but fragile for future maintenance.

---

## From PR #36 — Notifications + Error Routing (2026-03-25)

### Test Quality

- [ ] **Tests weakened to `cmd != nil`** — Several tests (LikeToggleResultMsg, PlaybackCmdSentMsg, AddToQueueResultMsg, DeviceTransfer) only assert `cmd != nil` instead of verifying toast content and type. Should use the two-pass pattern: `alertMsg := cmd(); a.Update(alertMsg); assert.Contains(a.View(), "expected text")`.

### Robustness

- [ ] **`alerts.Update()` type assertion failure silently ignored** — `app.go` does `if am, ok := updatedAlerts.(bubbleup.AlertModel); ok` but if assertion fails, alert state updates stop silently. Add defensive logging/comment.
- [ ] **`alerts.Init()` return value discarded** — Currently returns nil by design, but a BubbleUp upgrade could break alert auto-dismiss if Init() starts returning a setup command. Batch it into returned commands.
- [ ] **No validation of alert type registration** — `NewNotifications` calls `RegisterNewAlertType` 5 times but never validates success. Invalid theme color strings could cause silent registration failure.

### Consistency

- [ ] **`PlaybackStateFetchedMsg` errors still silent** — Despite "all errors via toast" rule, playback errors are silently skipped. Consider throttled toast after N consecutive failures, or document as intentional exception.

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
