---
title: "API Gateway & Data Architecture"
status: done
---

## Description
Centralizes all outbound Spotify HTTP traffic through a prioritized, rate-limited gateway while enforcing Elm Architecture purity, toast-based error notifications, TTL-based cache staleness, and adaptive idle polling to deliver a resilient, responsive playback experience.

Spotnik's interaction with the Spotify API spans playback polling, library browsing, search, queue management, device switching, and stats dashboards. As these features accumulated, several architectural gaps emerged: command functions mutated the Store directly inside goroutine closures (violating the Elm Architecture contract), HTTP requests fired with no throttling or deduplication, error feedback was limited to a single status string with no severity distinction, cached data had no concept of freshness, and polling ran at full speed regardless of user activity or playback state.

This feature consolidates five coordinated efforts that together establish a robust data architecture. First, all data-fetching commands were refactored to carry their results in typed message payloads, restoring the Elm purity guarantee that only Update() may write to the Store. Second, a centralized API Gateway was introduced to control all outbound HTTP traffic with token-bucket rate limiting, concurrency capping, request deduplication, priority classification, and 429 backoff. Third, the primitive statusMsg string was replaced with a BubbleUp-based toast notification system that routes all API errors through severity-typed overlays. Fourth, TTL-based staleness tracking was added to the Store so that Update() can make informed decisions about when to re-fetch versus reuse cached data. Fifth, an adaptive polling system was built to reduce API traffic when the user is idle or playback is paused, resuming full-speed polling on interaction.

## Acceptance Criteria
- [ ] Zero store.Set* or store.Clear* calls remain in internal/app/commands.go
- [ ] All Msg types carry Data + Err error fields
- [ ] All API calls route through the gateway
- [ ] Token bucket limits requests to 10/second with burst of 10
- [ ] Max 5 concurrent in-flight requests
- [ ] Duplicate in-flight requests are deduplicated
- [ ] statusMsg field completely removed from codebase
- [ ] All 16 former statusMsg sites emit typed toast commands
- [ ] Every data domain has a fetchedAt timestamp
- [ ] Boolean sentinels albumsLoaded and likedLoaded are removed
- [ ] Polling intervals adapt based on a 4-state matrix (active/idle x playing/paused)
- [ ] make ci passes
