---
title: "Gateway Proactive Traffic Shaping"
feature: 22-rate-limit-resilience
status: open
---

## Background

Spotify's rate limit is a **rolling 30-second window** — not a per-second cap. If the app makes too many requests in any 30-second window, Spotify returns 429 with a `Retry-After` header. The exact limit depends on whether the app is in development mode or extended quota mode. There is no published absolute number; the app must detect 429s and adapt.

Spotify's own recommendations for building with rate limits:
1. **Backoff-retry strategy** — respect `Retry-After`, slow down after 429
2. **Batch APIs** — use endpoints like `Get Multiple Albums` to fetch N items in 1 request
3. **snapshot_id** — for playlists, skip re-fetching unchanged data
4. **Lazy loading** — only fetch data the user is actively viewing

What Spotnik already does well:
- Visibility-gated polling (lazy loading) — only visible panes are polled
- `Retry-After` backoff with global fetch suspension
- Inflight dedup for identical Background GETs
- Per-pane exponential backoff on errors

Two gaps remain:

**Gap 1 — Simultaneous burst dispatch:** When Dashboard preset has 8 panes + NowPlaying + Queue, all eligible panes fire fetch commands on the same tick via `tea.Batch`. The gateway's semaphore (5) and token bucket (10 req/s) provide backpressure, but 8 simultaneous goroutines still consume tokens rapidly. When intervals align (e.g., tick 60: playlists@60, albums@120, liked@60, recent@30 all fire), the burst can trigger Spotify's 30-second window limit.

**Gap 2 — Token bucket too permissive for sustained load:** The token bucket allows 10 req/s sustained = 300 requests per 30-second window. This is tuned for burst (initial preset load) but too aggressive for steady-state polling. The bucket has no mechanism to self-adjust when 429s indicate the current rate exceeds Spotify's limit.

**What exists today:**
- `Gateway.Do()` — 4-phase pipeline: backoff check → token bucket → inflight dedup → semaphore
- `tokenBucket` — 10 req/s, burst 10, lazy refill, blocking `wait()`
- `checkNewlyVisiblePanes()` — fires all stale panes at once on preset switch (correct for initial load)
- Tick handler — all eligible panes dispatched in one `tea.Batch` (the problem)
- Global 429 backoff — stops all fetches, clears sentinels, schedules `throttleExpiredMsg`

**What must change:**
1. Gateway adaptively reduces its internal rate limit after consecutive 429s
2. Gateway recovers rate limit gradually (not instant reset)
3. Gateway exposes `CanAdmit(priority Priority) bool` for pre-flight checks
4. App-level fetch scheduler dispatches at most 1 library fetch per tick
5. Token bucket rate is dynamically adjustable (not hardcoded 10)

**Future optimizations (out of scope for this story):**
- Batch API usage: `Get Multiple Albums`, `Get Several Shows`, `Get Several Episodes` could reduce request count for initial loads
- Playlist `snapshot_id`: skip re-fetching playlist tracks when snapshot hasn't changed

## Design

### 1. `internal/api/gateway.go` — adaptive rate limit

Add fields to `Gateway`:

```go
type Gateway struct {
    // ... existing fields ...

    // Adaptive rate limiting — reduces internal rate after 429s, recovers gradually.
    adaptiveRate     float64       // current effective rate (starts at initialRate)
    initialRate      float64       // rate to recover to (10 req/s)
    minRate          float64       // floor (1 req/s)
    consecutive429s  int           // count of consecutive 429 responses
    last429Time      time.Time     // when the last 429 occurred
    recoveryInterval time.Duration // how long before attempting rate increase (30s)
}
```

**Constants:**

```go
const (
    // defaultInitialRate is 6 req/s = 180 requests per 30-second rolling window.
    // Spotify's rate limit is a rolling 30-second window (not per-second).
    // 6 req/s is a conservative default that stays well under typical limits
    // while still allowing fast initial preset population.
    // The adaptive mechanism will reduce this further if 429s occur.
    defaultInitialRate      = 6.0   // req/s
    defaultMinRate          = 1.0   // req/s — floor, never drop below this
    defaultRecoveryInterval = 30 * time.Second
)
```

**Constructor update (`NewGateway`):**

```go
func NewGateway() *Gateway {
    g := &Gateway{
        bucket:           newTokenBucket(defaultInitialRate, defaultInitialRate),
        semaphore:        make(chan struct{}, 5),
        inflight:         make(map[RequestKey]*inflightEntry),
        adaptiveRate:     defaultInitialRate,
        initialRate:      defaultInitialRate,
        minRate:          defaultMinRate,
        recoveryInterval: defaultRecoveryInterval,
    }
    g.lastEmittedTokens = int(g.bucket.max)
    return g
}
```

**On 429 — reduce rate (`Do()` method, inside the 429 handling block at `gateway.go:494`):**

After setting `g.backoffUntil` and `g.retryAfter`:

```go
// Adaptive rate reduction: halve the rate on each consecutive 429.
g.consecutive429s++
g.last429Time = time.Now()
newRate := g.initialRate / float64(int(1)<<uint(g.consecutive429s-1))
if newRate < g.minRate {
    newRate = g.minRate
}
g.adaptiveRate = newRate
g.bucket.setRate(g.adaptiveRate)
```

Sequence: 6 → 3 → 1.5 → 1.0 (floor)

**On successful request — reset consecutive counter (`Do()` method, after a non-429 response):**

After the HTTP call completes with a non-429 status:

```go
if resp != nil && resp.StatusCode != http.StatusTooManyRequests {
    g.mu.Lock()
    if g.consecutive429s > 0 {
        g.consecutive429s = 0
        // Don't immediately restore rate — recovery is gradual via tryRecover()
    }
    g.mu.Unlock()
}
```

**Recovery — gradually increase rate:**

```go
// tryRecover checks if enough time has passed since the last 429 to increase the rate.
// Called from CanAdmit() on each pre-flight check (every tick).
func (g *Gateway) tryRecover() {
    g.mu.Lock()
    defer g.mu.Unlock()
    if g.consecutive429s > 0 {
        return // still in active 429 sequence, don't recover
    }
    if g.adaptiveRate >= g.initialRate {
        return // already at max
    }
    if time.Since(g.last429Time) < g.recoveryInterval {
        return // not enough time since last 429
    }
    // Increase by one step: double the rate, cap at initialRate.
    newRate := g.adaptiveRate * 2
    if newRate > g.initialRate {
        newRate = g.initialRate
    }
    g.adaptiveRate = newRate
    g.last429Time = time.Now() // reset timer for next recovery step
    g.bucket.setRate(g.adaptiveRate)
}
```

Recovery sequence: 1.0 → 2.0 → 4.0 → 6.0 (30s between each step, ~90 seconds full recovery)

### 2. `internal/api/gateway_bucket.go` — dynamic rate

Add `setRate` method to `tokenBucket`:

```go
// setRate updates the token bucket's refill rate and max capacity.
// Thread-safe — acquires tb.mu.
func (tb *tokenBucket) setRate(rate float64) {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    tb.rate = rate
    tb.max = rate // burst = rate (1 second of burst)
}
```

When rate changes, existing tokens are preserved. The new rate takes effect on the next `wait()` call.

### 3. `internal/api/gateway.go` — `CanAdmit()`

Non-blocking pre-flight check. The app calls this before dispatching a fetch command to avoid creating goroutines that will immediately block or fail.

```go
// CanAdmit returns true if the gateway is likely to accept a request of the given
// priority without blocking or returning a RateLimitError. This is a pre-flight
// check — the actual Do() call still goes through all enforcement phases.
//
// Returns false when:
//   - Gateway is in 429 backoff
//   - Semaphore is at capacity (5 concurrent requests already in-flight)
func (g *Gateway) CanAdmit(priority Priority) bool {
    // Check backoff.
    g.mu.Lock()
    throttled := time.Now().Before(g.backoffUntil)
    g.mu.Unlock()
    if throttled {
        return false
    }

    // Attempt recovery before checking — this is the periodic recovery tick.
    g.tryRecover()

    // Check semaphore capacity (non-blocking).
    if len(g.semaphore) >= cap(g.semaphore) {
        return false
    }

    return true
}
```

Note: `CanAdmit` does NOT check the token bucket. The token bucket is a rate-smoothing mechanism — it may cause a short wait but won't reject. The semaphore and backoff are the hard gates.

### 4. `internal/app/handlers.go` — fetch scheduler

Replace the current "all eligible panes in one Batch" dispatch with a round-robin scheduler that picks the single most overdue pane per tick.

**New struct:**

```go
// pollEntry describes one pane's polling configuration for the fetch scheduler.
type pollEntry struct {
    paneID     layout.PaneID
    p          *pollState
    intervalFn func() int  // returns current interval based on playback/idle state
    fetching   func() bool
    setFetch   func(bool)
    cmd        func() tea.Cmd
    lastTick   int  // tickCount when this pane was last dispatched
}
```

**Scheduler logic** — replaces the `for _, entry := range []struct{...}` loop at `handlers.go:518-555`:

```go
// Build the poll table once (in App struct or as a method).
entries := a.pollEntries()

// Find the most overdue visible pane.
var best *pollEntry
var bestRatio float64
for i := range entries {
    e := &entries[i]
    if !a.layout.IsPaneVisible(e.paneID) {
        continue
    }
    // TopTracks/TopArtists share stats — visible if either is.
    if e.paneID == layout.PaneTopTracks {
        if !a.layout.IsPaneVisible(layout.PaneTopTracks) && !a.layout.IsPaneVisible(layout.PaneTopArtists) {
            continue
        }
    }
    if e.p.backoffTicks > 0 {
        e.p.backoffTicks--
        continue
    }
    if e.fetching() {
        continue
    }
    interval := e.intervalFn()
    if interval <= 0 {
        continue
    }
    since := a.tickCount - e.lastTick
    if since < interval {
        continue
    }
    ratio := float64(since) / float64(interval)
    if ratio > bestRatio {
        bestRatio = ratio
        best = e
    }
}

// Dispatch the most overdue pane if gateway admits it.
if best != nil && a.gateway.CanAdmit(api.Background) {
    best.setFetch(true)
    best.lastTick = a.tickCount
    cmds = append(cmds, best.cmd())
}
```

**`pollEntries()` method on `*App`:**

```go
func (a *App) pollEntries() []pollEntry {
    return []pollEntry{
        {layout.PanePlaylists, &a.playlistsPoll, func() int { return a.libraryInterval(&a.playlistsPoll, playlistsIntervals) }, a.store.PlaylistsFetching, a.store.SetPlaylistsFetching, func() tea.Cmd { return a.buildFetchPlaylistsCmd(0) }, 0},
        {layout.PaneAlbums, &a.albumsPoll, func() int { return a.libraryInterval(&a.albumsPoll, albumsIntervals) }, a.store.AlbumsFetching, a.store.SetAlbumsFetching, func() tea.Cmd { return a.buildFetchAlbumsCmd(0) }, 0},
        {layout.PaneLikedSongs, &a.likedSongsPoll, func() int { return a.libraryInterval(&a.likedSongsPoll, likedSongsIntervals) }, a.store.LikedFetching, a.store.SetLikedFetching, func() tea.Cmd { return a.buildFetchLikedTracksCmd(0) }, 0},
        {layout.PaneRecentlyPlayed, &a.recentPlayedPoll, func() int { return a.libraryInterval(&a.recentPlayedPoll, recentPlayedIntervals) }, a.store.RecentFetching, a.store.SetRecentFetching, func() tea.Cmd { return a.buildFetchRecentlyPlayedCmd() }, 0},
        {layout.PaneTopTracks, &a.statsPoll, func() int { return a.libraryInterval(&a.statsPoll, statsIntervals) }, func() bool { return a.store.StatsFetching("short_term") }, func(b bool) { a.store.SetStatsFetching("short_term", b) }, func() tea.Cmd { return a.buildFetchStatsCmd("short_term") }, 0},
        {layout.PaneFollowedShows, &a.followedShowsPoll, func() int { return a.libraryInterval(&a.followedShowsPoll, podcastIntervals) }, a.store.FollowedShowsFetching, a.store.SetFollowedShowsFetching, func() tea.Cmd { return a.buildFetchFollowedShowsCmd() }, 0},
        {layout.PaneSavedEpisodes, &a.savedEpisodesPoll, func() int { return a.libraryInterval(&a.savedEpisodesPoll, podcastIntervals) }, a.store.SavedEpisodesFetching, a.store.SetSavedEpisodesFetching, func() tea.Cmd { return a.buildFetchSavedEpisodesCmd() }, 0},
    }
}
```

**Key properties of the scheduler:**
- At most 1 library fetch per tick (playback + queue are separate, dispatched unconditionally on their own intervals)
- Most overdue pane wins — ensures fairness, no pane starves
- `CanAdmit` gate prevents dispatch when gateway is saturated
- `lastTick` tracks when each pane was last dispatched (not when it last succeeded)
- Per-pane backoff still applies (`p.backoffTicks > 0` skips)
- Fetching sentinel still applies (`e.fetching()` skips)

**Initial load (preset switch) is unchanged:** `checkNewlyVisiblePanes()` still fires all stale panes at once via `tea.Batch`. The scheduler only governs steady-state polling. This preserves fast initial population.

### 5. `internal/app/app.go` — add `pollEntries` field or method

The `pollEntry` slice is static (same 7 entries always). Store as a method that returns a new slice each call (so `lastTick` is reset per call — actually no, `lastTick` must persist across calls).

**Option A:** Store `lastTick` on `pollState` (already exists). Add `lastDispatchedTick int` to `pollState`.

```go
type pollState struct {
    errorCount         int
    backoffTicks       int
    hasData            bool
    lastDispatchedTick int  // NEW: tickCount when last dispatched
}
```

Then the scheduler reads `p.lastDispatchedTick` instead of `e.lastTick`.

**Option B:** Store entries as a field on `*App` with pointers to `pollState`.

Option A is simpler and keeps state with the pollState. Use Option A.

### 6. Gateway state observability

The `GatewayHealthPane` and `GatewayLivePane` already read gateway events. The adaptive rate should be visible in the `GatewayStateSnapshot` so these panes can display it.

Add to `domain.GatewayStateSnapshot`:

```go
type GatewayStateSnapshot struct {
    // ... existing fields ...
    AdaptiveRate     float64 // current adaptive rate limit (req/s)
    Consecutive429s  int     // count of consecutive 429 responses
}
```

Update `captureSnapshot()` and `captureSnapshotLocked()` to include these fields.

### 7. `checkNewlyVisiblePanes` — add `CanAdmit` gate

When switching presets, `checkNewlyVisiblePanes` fires all stale panes at once. Add a `CanAdmit` check to avoid dispatching into a saturated gateway:

```go
if !a.gateway.CanAdmit(api.Background) {
    break // stop dispatching — remaining panes will be picked up by the scheduler
}
```

This prevents the initial burst from overwhelming the gateway when it's already under backoff or at capacity.

## Acceptance Criteria

- [ ] Gateway reduces token bucket rate on consecutive 429s: 6→3→1 req/s
- [ ] Gateway recovers rate gradually: doubles every 30s after 429s stop, up to 6 req/s
- [ ] `Gateway.CanAdmit(priority)` returns false during backoff and when semaphore is full
- [ ] `CanAdmit` triggers `tryRecover()` on each call (periodic recovery check)
- [ ] `tokenBucket.setRate(rate)` updates both rate and max dynamically
- [ ] Fetch scheduler dispatches at most 1 library pane per tick
- [ ] Most overdue pane wins (highest `since/interval` ratio)
- [ ] `pollState.lastDispatchedTick` tracks when each pane was last dispatched
- [ ] `checkNewlyVisiblePanes` stops dispatching when `CanAdmit` returns false
- [ ] `GatewayStateSnapshot` includes `AdaptiveRate` and `Consecutive429s`
- [ ] Playback + Queue polling unchanged (dispatched on their own intervals, not through scheduler)
- [ ] Devices overlay polling unchanged (dispatched on its own interval, not through scheduler)
- [ ] Existing per-pane backoff and fetching sentinels still work correctly
- [ ] `make ci` passes

## Tasks

- [ ] Add `setRate()` method to `tokenBucket` in `internal/api/gateway_bucket.go`
      - test: `TestTokenBucket_SetRate`, `TestTokenBucket_SetRate_PreservesTokens`

- [ ] Add adaptive rate fields to `Gateway`; update `NewGateway()` in `internal/api/gateway.go`
      - test: `TestNewGateway_AdaptiveRateDefaults`

- [ ] Add adaptive rate reduction in `Do()` 429 handling block in `internal/api/gateway.go`
      - test: `TestGateway_AdaptiveRate_ReducesOnConsecutive429`, `TestGateway_AdaptiveRate_FloorsAtMin`

- [ ] Add consecutive-429 reset on successful requests in `Do()` in `internal/api/gateway.go`
      - test: `TestGateway_AdaptiveRate_ResetsConsecutiveOnSuccess`

- [ ] Add `tryRecover()` method in `internal/api/gateway.go`
      - test: `TestGateway_AdaptiveRate_RecoversAfterInterval`, `TestGateway_AdaptiveRate_RecoversToInitial`

- [ ] Add `CanAdmit()` method in `internal/api/gateway.go`
      - test: `TestGateway_CanAdmit_RejectsDuringBackoff`, `TestGateway_CanAdmit_RejectsWhenSemaphoreFull`, `TestGateway_CanAdmit_AllowsWhenClear`, `TestGateway_CanAdmit_CallsTryRecover`

- [ ] Add `AdaptiveRate` + `Consecutive429s` to `domain.GatewayStateSnapshot` in `internal/domain/types.go`
      - test: compile-time — existing gateway tests will fail if fields are missing from snapshot methods

- [ ] Update `captureSnapshot()` and `captureSnapshotLocked()` in `internal/api/gateway.go`
      - test: `TestGateway_StateSnapshot_AdaptiveRate`, `TestGateway_StateSnapshot_Consecutive429s`

- [ ] Add `lastDispatchedTick` to `pollState` in `internal/app/app.go`
      - test: compile-time — field addition, no behavior change yet

- [ ] Add `pollEntries()` method to `*App` in `internal/app/app.go`
      - test: `TestApp_PollEntries_AllSevenPanes`, `TestApp_PollEntries_IntervalsNonZero`

- [ ] Replace batch dispatch with fetch scheduler in TickMsg handler in `internal/app/handlers.go`
      - test: `TestApp_FetchScheduler_DispatchesMostOverdue`, `TestApp_FetchScheduler_SkipsWhenCanAdmitFalse`, `TestApp_FetchScheduler_SkipsHiddenPanes`, `TestApp_FetchScheduler_SkipsWhenFetching`, `TestApp_FetchScheduler_RespectsPerPaneBackoff`

- [ ] Add `CanAdmit` gate to `checkNewlyVisiblePanes()` in `internal/app/app.go`
      - test: `TestApp_CheckNewlyVisiblePanes_StopsWhenCanAdmitFalse`

- [ ] `make ci` passes
