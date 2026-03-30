---
title: "Adaptive Idle Polling"
feature: 11-api-gateway
status: done
---

## Background
The tick loop polled at full speed (3s playback, 9s queue) regardless of user activity or playback state. When music was paused or the user was on the stats/playlists view, polling wasted bandwidth and risked rate limiting. This story is the proactive layer (Layer 1) of rate management -- it reduces the number of requests entering the gateway. Feature 30 (API Gateway) is the reactive layer (Layer 2) -- it limits the rate of requests that pass through. They are independent but complementary.

Gap reference: G6 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

## Design

### Polling Schedule

| State | Playback Interval | Queue Interval |
|---|---|---|
| Active + Playing | 3s (current) | 9s (current) |
| Active + Paused | 10s | 30s |
| Idle (60s no input) + Playing | 10s | 30s |
| Idle + Paused | 30s | 60s |

"Active" = user interacted within the last 60 seconds. "Idle" = no tea.KeyMsg received for 60+ seconds.

### Implementation
```go
lastInteraction time.Time
idleThreshold   time.Duration  // 60 * time.Second

func (a *App) isIdle() bool {
    return time.Since(a.lastInteraction) > a.idleThreshold
}

func (a *App) pollIntervals() (playbackInterval, queueInterval int) {
    idle := a.isIdle()
    playing := false
    if ps := a.store.PlaybackState(); ps != nil {
        playing = ps.IsPlaying
    }
    switch {
    case !idle && playing:
        return 3, 9
    case !idle && !playing:
        return 10, 30
    case idle && playing:
        return 10, 30
    default: // idle && !playing
        return 30, 60
    }
}
```

### Idle-to-Active Reset
When tea.KeyMsg arrives after app was idle, reset `a.tickCount = 0` to force immediate fetch on next tick.

### Verification
```bash
grep -r 'playbackFetchInterval\b\|queueFetchInterval\b' internal/app/app.go
# Expected: ZERO matches
grep -r 'pollIntervals\|isIdle\|lastInteraction' internal/app/app.go
# Expected: multiple matches
make ci
```

## Acceptance Criteria
- [ ] Polling intervals adapt based on a 4-state matrix (active/idle x playing/paused)
- [ ] User idle state detected after 60 seconds of no `tea.KeyMsg`
- [ ] Returning from idle resets `tickCount` to 0 for immediate data refresh
- [ ] Old hardcoded `playbackFetchInterval` and `queueFetchInterval` constants removed
- [ ] `make ci` passes

## Tasks
- [ ] Track lastInteraction time -- add fields to App struct, update on KeyMsg
      - test: isIdle() returns false immediately after creation; true after threshold; KeyMsg resets
- [ ] Adaptive pollIntervals() method -- replace hardcoded constants with dynamic calculation
      - test: active + playing -> 3s/9s; active + paused -> 10s/30s; idle + playing -> 10s/30s; idle + paused -> 30s/60s
- [ ] Wire into tick handler + idle-to-active reset + docs
      - test: tick handler uses dynamic intervals; KeyMsg after idle resets tickCount to 0
      - test: polling interval changes when playback state changes
