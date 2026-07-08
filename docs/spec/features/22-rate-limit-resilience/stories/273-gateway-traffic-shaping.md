---
title: "Gateway Proactive Traffic Shaping"
feature: 22-rate-limit-resilience
status: open
---

## Background

Spotify's rate limit is a **rolling 30-second window**. There is no published absolute number; the limit depends on the app's mode (development vs. extended quota). The app must therefore keep its own request volume conservative and self-correct when Spotify signals it is too aggressive.

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

What it does **not** do well enough yet:

**Gap 1 — Default token bucket is too permissive for sustained load.**  
The gateway currently allows **10 requests per second** with a burst of 10. Over a 30-second rolling window that is **up to 300 requests**. In practice this exceeds the limit Spotify enforces for many apps, so the 429s the user observed are expected consequences, not edge cases. The gateway reacts to 429s after they happen but never lowers its own steady-state ceiling.

**Gap 2 — Burst amplification on preset switch and aligned ticks.**  
When the Dashboard preset is active, 5–7 library panes are visible. On each tick all eligible panes fire at once via `tea.Batch`. When their intervals align (e.g. tick 60: playlists@60, liked@60, recent@30, followed shows@60) a burst hits Spotify in the same second. Preset switches make this worse: `checkNewlyVisiblePanes()` dispatches every newly visible stale pane immediately.

**Gap 3 — Polling never pauses when the user is away.**  
Idle detection currently shortens intervals after 60 seconds of no input, but it does not stop library polling. If the app is left running overnight with music paused, it still polls the same visible panes every 2–10 minutes forever. Over hours this still consumes the 30-second budget when a burst occurs.

**Gap 4 — No pre-flight admission gate.**  
Commands are dispatched into goroutines that only learn they are blocked when they reach the gateway. This wastes goroutines and makes it hard for the app to avoid piling up work when the gateway is already saturated.

## Problem statement

The user left the app running with the Dashboard preset selected and music playing. After hours/days away, the app showed a "rate limited" toast and the gateway blocked requests. The gateway's job is to avoid letting Spotify rate-limit Spotnik in the first place. It failed because its internal throttle was set too high, it had no mechanism to self-adjust downward, and it kept issuing background requests even when there was no human present.

## Design

### Goal

Reduce the steady-state request rate to something Spotify's rolling 30-second window tolerates, smooth bursts, and stop all library polling when the user has been away for a long time. Keep playback/queue polling responsive.

### 1. `internal/api/gateway_bucket.go` — decouple rate and burst

Add a configurable burst cap that is independent of the refill rate. A slow refill rate should not kill initial preset load completely.

```go
type tokenBucket struct {
    mu       sync.Mutex
    tokens   float64
    max      float64 // burst capacity (not the same as rate)
    rate     float64 // tokens per second
    lastFill time.Time
}

func newTokenBucket(max, rate float64) *tokenBucket

// setRate updates the refill rate without changing the burst capacity.
// Existing tokens are preserved, capped to the (unchanged) max.
func (tb *tokenBucket) setRate(rate float64)
```

**Lock ordering note:** `setRate` only acquires `tb.mu`. `Gateway` code must never hold both `g.mu` and `tb.mu` at the same time in opposite order to `captureSnapshot()`. Specifically, `captureSnapshot()` acquires `tb.mu` then `g.mu`; therefore any code path that holds `g.mu` and needs to adjust the bucket must release `g.mu` before calling `setRate`, or use a helper that acquires `tb.mu` after releasing `g.mu`.

### 2. `internal/api/gateway.go` — adaptive rate limiting

New constants:

```go
const (
    // defaultBackgroundRate is the sustained rate for Background requests.
    // 2 req/s = 60 requests per 30-second rolling window.
    // This is intentionally conservative; the adaptive mechanism can raise
    // it temporarily after a sustained clean period.
    defaultBackgroundRate = 2.0 // req/s

    // defaultBurst is the maximum tokens the bucket can hold.
    // Allows short bursts (e.g. initial preset load) without raising the
    // long-term average above Spotify's rolling window.
    defaultBurst = 5

    // minBackgroundRate is the floor. Never drop below this.
    minBackgroundRate = 0.5 // req/s

    // rateReductionStep is how much the rate is lowered on each 429.
    rateReductionStep = 1.0 // req/s

    // rateRecoveryStep is how much the rate is raised on each recovery tick.
    rateRecoveryStep = 0.5 // req/s

    // recoveryInterval is how long must pass without a 429 before rate rises.
    recoveryInterval = 30 * time.Second

    // longIdleThreshold is how long without input before library polling pauses.
    longIdleThreshold = 15 * time.Minute
)
```

New Gateway fields:

```go
type Gateway struct {
    // ... existing fields ...

    // backgroundRate is the current effective Background request rate.
    backgroundRate float64
    // burst is the token bucket burst capacity (kept independent of rate).
    burst float64
    // last429Time is when the most recent 429 response was received.
    last429Time time.Time
    // lastRecoveryTick is when the rate was last raised toward defaultBackgroundRate.
    lastRecoveryTick time.Time
}
```

Constructor:

```go
func NewGateway() *Gateway {
    g := &Gateway{
        bucket:           newTokenBucket(defaultBurst, defaultBackgroundRate),
        semaphore:        make(chan struct{}, 5),
        inflight:         make(map[RequestKey]*inflightEntry),
        backgroundRate:   defaultBackgroundRate,
        burst:            defaultBurst,
        last429Time:      time.Time{},
        lastRecoveryTick: time.Now(),
    }
    g.lastEmittedTokens = int(g.bucket.max)
    return g
}
```

On 429 — inside the `resp.StatusCode == 429` block in `Do()`:

```go
now := time.Now()
g.mu.Lock()
g.retryAfter = retryAfter
g.backoffUntil = now.Add(time.Duration(retryAfter) * time.Second)
g.last429Time = now

// Reduce rate by one step, floored at minBackgroundRate.
newRate := g.backgroundRate - rateReductionStep
if newRate < minBackgroundRate {
    newRate = minBackgroundRate
}
g.backgroundRate = newRate

// Reset recovery timer so we do not start climbing again immediately.
g.lastRecoveryTick = now

// Snapshot the values we need for setRate while still under g.mu, then
// release g.mu before touching the bucket to preserve lock order.
bucket := g.bucket
rate := g.backgroundRate
g.mu.Unlock()

bucket.setRate(rate)
```

On recovery:

```go
// tryRecover raises the Background rate slowly when no 429 has occurred
// for recoveryInterval and at least recoveryInterval has passed since the
// last rate change. Called from CanAdmit and from App.TickMsg.
func (g *Gateway) tryRecover() {
    g.mu.Lock()

    if g.backgroundRate >= defaultBackgroundRate {
        g.mu.Unlock()
        return
    }
    if time.Since(g.last429Time) < recoveryInterval {
        g.mu.Unlock()
        return
    }
    if time.Since(g.lastRecoveryTick) < recoveryInterval {
        g.mu.Unlock()
        return
    }

    newRate := g.backgroundRate + rateRecoveryStep
    if newRate > defaultBackgroundRate {
        newRate = defaultBackgroundRate
    }
    g.backgroundRate = newRate
    g.lastRecoveryTick = time.Now()

    // Snapshot the bucket pointer and new rate, then release g.mu before
    // touching the bucket. captureSnapshot() locks bucket.mu then g.mu, so we
    // must never hold g.mu while acquiring bucket.mu.
    bucket := g.bucket
    g.mu.Unlock()

    bucket.setRate(newRate)
}
```

Sequence after 429s: 2 → 1 → 0.5 req/s. Recovery: 0.5 → 1 → 1.5 → 2 (one step every 30s).

**Why no "consecutive 429" counter:** Once the gateway enters backoff after the first 429, all subsequent Background requests are rejected at phase 1 before they reach Spotify. They therefore cannot return additional 429s. Counting consecutive 429 responses would stay at 1 and reset on the first success, giving the same behavior as tracking the most recent 429. Using `last429Time` is simpler and matches the actual execution model.

### 3. `internal/api/gateway.go` — `CanAdmit`

Pre-flight admission that prevents the app from dispatching commands that will immediately fail or block for a long time.

```go
// CanAdmit returns true if the gateway is currently willing to accept a
// request of the given priority. It is called by the app before dispatching
// a fetch command so the app can avoid creating work that will be rejected.
//
// It returns false when:
//   - the gateway is in 429 backoff
//   - the token bucket has no tokens right now (Background only)
//   - the semaphore is full (Background only)
//
// Interactive requests are only gated by backoff; they are never queued by the
// app and the caller decides whether to dispatch them.
func (g *Gateway) CanAdmit(priority Priority) bool {
    g.mu.Lock()
    throttled := time.Now().Before(g.backoffUntil)
    g.mu.Unlock()

    // Periodic recovery tick. Runs even during backoff so the rate can climb
    // while the door is closed and be ready when it reopens.
    g.tryRecover()

    if throttled {
        return false
    }

    if priority == Interactive {
        // Interactive requests are never queued by the app; only reject during backoff.
        return true
    }

    if len(g.semaphore) >= cap(g.semaphore) {
        return false
    }

    // Non-blocking token check.
    g.bucket.mu.Lock()
    now := time.Now()
    elapsed := now.Sub(g.bucket.lastFill).Seconds()
    tokens := g.bucket.tokens + elapsed*g.bucket.rate
    if tokens > g.bucket.max {
        tokens = g.bucket.max
    }
    g.bucket.lastFill = now
    g.bucket.tokens = tokens
    hasToken := tokens >= 1
    g.bucket.mu.Unlock()

    return hasToken
}
```

### 4. `internal/app/handlers.go` — fetch scheduler

Replace the batch loop with a round-robin scheduler that dispatches at most one library pane per tick.

Add `lastDispatchedTick` and `lastSuccessTick` to `pollState` in `internal/app/app.go`:

```go
type pollState struct {
    backoffTicks       int
    errorCount         int
    hasData            bool
    lastDispatchedTick int
    lastSuccessTick    int
}
```

Polling table method on `*App`:

```go
func (a *App) libraryPollEntries() []libraryPollEntry {
    return []libraryPollEntry{
        {layout.PanePlaylists, &a.playlistsPoll, playlistsIntervals, a.store.PlaylistsFetching, a.store.SetPlaylistsFetching, func() tea.Cmd { return a.buildFetchPlaylistsCmd(0) }},
        {layout.PaneAlbums, &a.albumsPoll, albumsIntervals, a.store.AlbumsFetching, a.store.SetAlbumsFetching, func() tea.Cmd { return a.buildFetchAlbumsCmd(0) }},
        {layout.PaneLikedSongs, &a.likedSongsPoll, likedSongsIntervals, a.store.LikedFetching, a.store.SetLikedFetching, func() tea.Cmd { return a.buildFetchLikedTracksCmd(0) }},
        {layout.PaneRecentlyPlayed, &a.recentPlayedPoll, recentPlayedIntervals, a.store.RecentFetching, a.store.SetRecentFetching, func() tea.Cmd { return a.buildFetchRecentlyPlayedCmd() }},
        {layout.PaneTopTracks, &a.statsPoll, statsIntervals,
            func() bool { return a.store.StatsFetching("short_term") },
            func(b bool) { a.store.SetStatsFetching("short_term", b) },
            func() tea.Cmd { return a.buildFetchStatsCmd("short_term") }},
        {layout.PaneFollowedShows, &a.followedShowsPoll, podcastIntervals, a.store.FollowedShowsFetching, a.store.SetFollowedShowsFetching, func() tea.Cmd { return a.buildFetchFollowedShowsCmd() }},
        {layout.PaneSavedEpisodes, &a.savedEpisodesPoll, podcastIntervals, a.store.SavedEpisodesFetching, a.store.SetSavedEpisodesFetching, func() tea.Cmd { return a.buildFetchSavedEpisodesCmd() }},
    }
}
```

Tick handler scheduler logic:

```go
// Playback and queue keep their own unconditional intervals.
playbackInterval, queueInterval := a.pollIntervals()
if a.tickCount%playbackInterval == 0 {
    cmds = append(cmds, fetchPlaybackStateCmd(a.player, api.Background))
}
if a.tickCount%queueInterval == 0 {
    cmds = append(cmds, fetchQueueCmd(a.player))
}

// Long-idle guard: pause all library polling after 15 minutes of inactivity.
// Playback and queue still run (so the app can resume when the user returns).
if !a.isLongIdle() {
    if best := a.pickMostOverdueLibraryPane(); best != nil && a.gateway.CanAdmit(api.Background) {
        p := best.p
        p.lastDispatchedTick = a.tickCount
        best.setFetch(true)
        cmds = append(cmds, best.cmd())
    }
}

// Recovery tick: even if no pane was admitted, try to raise the gateway rate.
a.gateway.tryRecover()
```

`pickMostOverdueLibraryPane`:

```go
func (a *App) pickMostOverdueLibraryPane() *libraryPollEntry {
    entries := a.libraryPollEntries()
    var best *libraryPollEntry
    var bestRatio float64
    for i := range entries {
        e := &entries[i]
        paneID := e.paneID
        if e.paneID == layout.PaneTopTracks {
            if !a.layout.IsPaneVisible(layout.PaneTopTracks) && !a.layout.IsPaneVisible(layout.PaneTopArtists) {
                continue
            }
        } else if !a.layout.IsPaneVisible(paneID) {
            continue
        }

        p := e.p
        if p.backoffTicks > 0 {
            p.backoffTicks--
            continue
        }
        if e.fetching() {
            continue
        }

        interval := a.libraryInterval(p, e.iv)
        if interval <= 0 {
            continue
        }

        // Use last success, not last dispatch, so a pane whose previous fetch
        // failed is not treated as freshly updated.
        since := a.tickCount - p.lastSuccessTick
        if since < interval {
            continue
        }

        ratio := float64(since) / float64(interval)
        if ratio > bestRatio {
            bestRatio = ratio
            best = e
        }
    }
    return best
}
```

`lastSuccessTick` is set by each library loaded-message handler on success, alongside resetting `errorCount` and setting `hasData = true`.

### 5. `internal/app/app.go` — `checkNewlyVisiblePanes` gating

Initial preset load is still batched, but each dispatch is gated by `CanAdmit` so the batch stops if the gateway is saturated.

```go
// Iterate visible panes in deterministic order so CanAdmit gating produces
// consistent results across runs and tests.
visibleIDs := make([]layout.PaneID, 0, len(cur.Visible))
for id := range cur.Visible {
    visibleIDs = append(visibleIDs, id)
}
sort.Slice(visibleIDs, func(i, j int) bool { return visibleIDs[i] < visibleIDs[j] })

for _, id := range visibleIDs {
    if oldVisible[id] {
        continue
    }
    gate, ok := gates[id]
    if !ok {
        continue
    }
    if gate.fetching() {
        continue
    }
    if !gate.stale() {
        continue
    }
    if !a.gateway.CanAdmit(api.Background) {
        break
    }
    gate.setFetch(true)
    cmds = append(cmds, gate.cmd())
}
```

### 6. `internal/app/app.go` — long-idle helper

```go
// isLongIdle returns true when the user has been inactive longer than longIdleThreshold.
// When true, library polling pauses. Playback/queue continue so the UI stays current.
// NOTE: this is measured from lastInteraction, independent of idleThreshold.
func (a *App) isLongIdle() bool {
    return time.Since(a.lastInteraction) > longIdleThreshold
}
```

`lastInteraction` is updated on `tea.KeyMsg`, `tea.MouseMsg`, and `tea.WindowSizeMsg` (a resize implies the user is at the machine).

### 7. `internal/domain/gateway.go` — GatewayStateSnapshot observability

```go
type GatewayStateSnapshot struct {
    // ... existing fields ...
    BackgroundRate   float64 // current adaptive Background rate limit (req/s)
    BurstCapacity    float64 // token bucket burst capacity
    Last429AgoSecs   float64 // seconds since last 429 (0 if never)
}
```

`captureSnapshot()` and `captureSnapshotLocked()` populate these. The fields are added to `internal/domain/gateway.go`, not `internal/domain/types.go`.

### 8. UI panes

`GatewayHealthPane` and `GatewayLivePane` already read `GatewayStateSnapshot`. They will display `BackgroundRate` without additional data plumbing. If layout/golden output changes, golden files must be regenerated with `go test ./... -update` and sanity tests updated if behavior is critical.

## Acceptance Criteria

- [ ] `tokenBucket` exposes `setRate(rate)` and keeps `max` (burst) independent of `rate`
- [ ] `NewGateway()` defaults to Background rate 2 req/s, burst 5
- [ ] `Gateway` tracks `backgroundRate`, `burst`, `last429Time`, `lastRecoveryTick`
- [ ] Each 429 reduces `backgroundRate` by 1 req/s, floored at 0.5 req/s
- [ ] `tryRecover()` raises `backgroundRate` by 0.5 req/s every 30s with no 429s, capped at 2 req/s
- [ ] `CanAdmit(Background)` returns false during backoff, when semaphore is full, or when token bucket has no tokens
- [ ] `CanAdmit(Interactive)` returns false only during backoff
- [ ] `CanAdmit` triggers `tryRecover()` on every call, even during backoff
- [ ] `TickMsg` handler calls `gateway.tryRecover()` even when no pane is dispatched
- [ ] Scheduler dispatches at most 1 library pane per tick
- [ ] Scheduler picks the most overdue pane using `lastSuccessTick / interval`
- [ ] `checkNewlyVisiblePanes()` stops dispatching when `CanAdmit` returns false
- [ ] Library polling pauses when `isLongIdle()` is true
- [ ] Playback and queue polling remain unchanged
- [ ] Devices overlay polling remains unchanged
- [ ] Per-pane backoff and fetching sentinels continue to work
- [ ] `GatewayStateSnapshot` includes `BackgroundRate`, `BurstCapacity`, `Last429AgoSecs`
- [ ] `make ci` passes

## Tasks

- [ ] Decouple `max` and `rate` in `tokenBucket`; add `setRate()`; add burst parameter to `NewGateway()`
      - test: `TestTokenBucket_SetRate_PreservesBurst`, `TestTokenBucket_SetRate_PreservesTokens`, `TestTokenBucket_BurstIndependentFromRate`
- [ ] Update `NewGateway()` with default rate 2/s, burst 5 and adaptive fields
      - test: `TestNewGateway_AdaptiveDefaults`
- [ ] Add 429 rate-reduction logic in `Do()` with safe lock ordering
      - test: `TestGateway_429_ReducesRate`, `TestGateway_429_FloorsAtMin`
- [ ] Add `tryRecover()` and wire it into `CanAdmit()` and `TickMsg`
      - test: `TestGateway_RecoversRateAfterInterval`, `TestGateway_RecoveryCappedAtDefault`
- [ ] Implement `CanAdmit(priority)` with backoff, semaphore, and token checks
      - test: `TestGateway_CanAdmit_BackoffFalse`, `TestGateway_CanAdmit_SemaphoreFullFalse`, `TestGateway_CanAdmit_NoTokenFalse`, `TestGateway_CanAdmit_CallsTryRecover`
- [ ] Add `BackgroundRate`, `BurstCapacity`, `Last429AgoSecs` to `GatewayStateSnapshot` in `internal/domain/gateway.go` and update capture methods
      - test: `TestGateway_Snapshot_IncludesAdaptiveFields`
- [ ] Add `lastSuccessTick` to `pollState` in `internal/app/app.go`
      - test: compile-time / existing poll tests
- [ ] Add `libraryPollEntries()` and `pickMostOverdueLibraryPane()` to `*App`
      - test: `TestApp_PickMostOverdue_OverdueWins`, `TestApp_PickMostOverdue_SkipsHidden`, `TestApp_PickMostOverdue_SkipsBackoff`, `TestApp_PickMostOverdue_UsesSuccessTick`
- [ ] Replace library batch dispatch with scheduler in `TickMsg` handler
      - test: `TestApp_Tick_DispatchesAtMostOneLibraryPane`, `TestApp_Tick_SchedulerRespectsCanAdmit`, `TestApp_Tick_PausesLibraryOnLongIdle`
- [ ] Gate `checkNewlyVisiblePanes()` with `CanAdmit`
      - test: `TestApp_CheckNewlyVisiblePanes_StopsWhenCanAdmitFalse`
- [ ] Add `isLongIdle()` and long-idle pause to library scheduler; update `lastInteraction` on `WindowSizeMsg`
      - test: `TestApp_IsLongIdle`, `TestApp_LongIdlePausesLibraryPolling`
- [ ] Update loaded-message handlers to set `lastSuccessTick` on success
      - test: existing handler tests extended
- [ ] Regenerate golden files if `GatewayHealthPane`/`GatewayLivePane` output changes
      - test: `go test ./... -update`, review diff, commit
- [ ] `make ci` passes

## Out of scope

- Batch API usage (`Get Multiple Albums`, `Get Several Shows`, `Get Several Episodes`)
- Playlist `snapshot_id` short-circuit
- These remain valid future optimizations but are not required to stop the observed 429s.
