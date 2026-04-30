---
title: "API Gateway & Reliability"
status: done
---

## Description

Centralized HTTP gateway that all Spotify API calls route through. Implements token-bucket rate limiting (10 req/s, burst 10), in-flight request deduplication for GET requests, priority classification (Interactive vs Background), adaptive idle polling, TTL-based response staleness tracking, and a typed error system (RateLimitError, AuthError, ValidationError). The error resilience stories establish token refresh on 401, rate-limit backoff on 429, and typed errors throughout. Architecture health stories enforce import boundaries, eliminate dead code, and align domain types. Gateway rate protection rejects Interactive requests during active backoff and applies the token bucket to user-triggered commands to prevent hold-key 429s.

## Acceptance Criteria

- [ ] All requests route through Gateway — no direct http.Client.Do calls in API methods
- [ ] Token bucket enforces 10 req/s with burst 10; Interactive requests rejected during backoff
- [ ] In-flight dedup prevents duplicate concurrent GET requests for the same endpoint
- [ ] 429 triggers backoff for Retry-After seconds with ratelimit toast; 401 triggers token refresh + retry
- [ ] Typed errors propagate to toast notifications; no inline error boxes in View()
- [ ] Import boundaries enforced: ui/ never imports api/, api/ never imports ui/
- [ ] Open: stories 21, 22, 34, 35 (architecture health gaps)
