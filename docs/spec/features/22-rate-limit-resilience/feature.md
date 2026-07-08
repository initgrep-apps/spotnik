---
title: "Rate Limit Resilience & Empty State Context"
status: open
---

## Description

Two tightly-coupled problems discovered during long-running sessions with the Dashboard preset + NowPlaying active:

**Problem 1 — Wrong empty state messages:** When rate limiting hits before a preset's panes ever fetch data (e.g., Podcasts preset never active before 429), panes show "No followed shows" / "No saved episodes" — misleading because the data was never fetched, not genuinely empty. Panes must distinguish between never-fetched, fetching, fetch-failed, rate-limited, and genuinely-empty states.

**Problem 2 — Gateway is purely reactive:** The gateway only reacts to 429s with global backoff. It does nothing to prevent them. When Dashboard preset has 8 panes + NowPlaying + Queue, all fire fetch commands simultaneously on the same tick. The token bucket (10 req/s) and semaphore (5 concurrent) provide backpressure but don't prevent burst patterns that trigger Spotify's rate limits. The gateway must proactively shape traffic.

## Acceptance Criteria

- [ ] Panes show context-aware messages: "Unable to load X — rate limited" vs "No X" vs "Loading X..."
- [ ] `StateReader` exposes podcast error/fetching accessors so panes can read fetch status
- [ ] `EmptyState` supports status-driven text (never-fetched, fetching, error, rate-limited, empty)
- [ ] All 9 table panes updated to check fetch status before rendering empty state
- [x] Gateway exposes admission control: `CanAdmit(priority) bool` — app checks before dispatching
- [x] Fetch commands are staggered across ticks — no two panes fire on the same tick
- [x] Gateway adaptively reduces its internal rate limit after consecutive 429s
- [x] Gateway recovers rate limit gradually after 429s stop (not instant reset)
- [x] `make ci` passes

## Stories

| # | Story | File | Depends on |
|---|-------|------|------------|
| 272 | Context-aware empty states | `stories/272-context-aware-empty-states.md` | — |
| 273 | Gateway proactive traffic shaping | `stories/273-gateway-traffic-shaping.md` | — |
