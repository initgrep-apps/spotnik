---
title: "Error Resilience"
status: done
---

## Description
Automatic token refresh on 401, universal rate-limit backoff on 429, and context-aware 403 messaging across all API calls, so the app recovers gracefully from Spotify API errors without user intervention. This feature closes gaps in the original error handling strategies by implementing a RefreshableTokenProvider with proactive 5-minute-before-expiry refresh, extending the existing backoff mechanism so every build*Cmd function emits RateLimitedMsg on 429, and adding context-appropriate 403 error messages for non-playback API calls. The implementation depends on the typed error types and TokenProvider interface established in the typed-errors feature (spec 24).

## Acceptance Criteria
- [ ] RefreshableTokenProvider exists with proactive 5-minute refresh
- [ ] 401 errors trigger token refresh + single retry
- [ ] Failed refresh shows "Session expired. Run: spotnik auth"
- [ ] All build*Cmd functions handle RateLimitError -> RateLimitedMsg
- [ ] 429 from any API call activates the backoff mechanism
- [ ] 403 errors show context-appropriate messages for all features
- [ ] All existing tests pass
- [ ] make ci passes
