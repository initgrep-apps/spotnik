---
title: "API Infrastructure & Resilience"
status: done
stories: 18–35, 37–39, 65, 126–127, 199–203, 206, 225
---

## Description

Centralized HTTP gateway that all Spotify API calls route through. Implements token-bucket rate limiting (10 req/s, burst 10), in-flight request deduplication for GET requests, priority classification (Interactive vs Background), adaptive idle polling, TTL-based response staleness tracking, and a typed error system (RateLimitError, AuthError, ValidationError). Architecture health stories enforce import boundaries, eliminate dead code, and align domain types. Gateway rate protection rejects Interactive requests during active backoff and applies the token bucket to user-triggered commands to prevent hold-key 429s.

**Error Resilience & Universal Polling (stories 199–203, 206, 225, absorbed from feature 15):** Replaces one-time startup `initialFetchCmds()` with universal tick-driven polling for all data panes. Adds per-pane exponential backoff, first-load retry, and fixes for silent failures (prefs flush, search offset drop, missing HTTP timeout). All API error toasts gain specific recovery hints. Overlay self-sufficiency fixes: profile/devices overlays no longer hang on empty store. Auth error mapping shows user-friendly messages instead of raw Go errors.

## Acceptance Criteria

- [ ] All requests route through Gateway — no direct http.Client.Do calls in API methods
- [ ] Token bucket enforces 10 req/s with burst 10; Interactive requests rejected during backoff
- [ ] In-flight dedup prevents duplicate concurrent GET requests for the same endpoint
- [ ] 429 triggers backoff for Retry-After seconds with ratelimit toast; 401 triggers token refresh + retry
- [ ] Typed errors propagate to toast notifications; no inline error boxes in View()
- [ ] Import boundaries enforced: ui/ never imports api/, api/ never imports ui/
- [ ] All library panes load via polling — no `initialFetchCmds`
- [ ] No network at startup: panes poll every 5s with exponential backoff, first failure emits toast, auto-recovery confirmed
- [ ] Playback polling error toast fires on 3rd consecutive error (not 5th)
- [ ] Preference flush failure emits `ToastWarning` — no silent stderr
- [ ] HTTP calls time out after 30s
- [ ] Profile/devices overlays emit self-fetch when store is empty; never hang indefinitely
