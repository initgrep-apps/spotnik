# Feature Implementer Agent Memory

- [project_spotnik_feature22_complete.md](project_spotnik_feature22_complete.md) — Feature 22 (app.go Decomposition): what moved where, routing.go rationale, test collision fix, fmt-check worktree gotcha
- [project_spotnik_feature25_complete.md](project_spotnik_feature25_complete.md) — Feature 25 (API DRY Refactoring): BaseClient embedding, pagination helper, integration build tag split, test update lessons
- [project_spotnik_feature27_complete.md](project_spotnik_feature27_complete.md) — Feature 27 (Error Resilience): 429/403/401 handling extended to all API calls, unauthorizedMsg/tokenRefreshedMsg pattern, test chain stepping
- [project_spotnik_feature28_complete.md](project_spotnik_feature28_complete.md) — Feature 28 (API Cleanup Follow-up): Get prefix removal, TokenProvider interface, search.go import boundary fix
- [project_spotnik_feature32_complete.md](project_spotnik_feature32_complete.md) — Feature 32 (Staleness Tracking): fetchedAt timestamps, IsStale helper, TTL constants, boolean sentinel removal, staleness-gated fetches in app.go
- [project_spotnik_feature33_complete.md](project_spotnik_feature33_complete.md) — Feature 33 (Idle Polling Backoff): 4-state polling matrix, isIdle helper, pollIntervals method, tick handler wiring, test time-control pattern
- [project_spotnik_feature34_complete.md](project_spotnik_feature34_complete.md) — Feature 34 (Docs/Dead Code/Init): store.go doc fix, unmarshalJSON removal (inlined in search.go), statsFetchedAt pre-alloc in New(), issues.md cleanup
- [project_spotnik_feature35_complete.md](project_spotnik_feature35_complete.md) — Feature 35 (Type Design Alignment): message exports, AlbumsLoadedMsg Offset, SearchResult to domain, Elm purity for DevicesLoadedMsg, package doc shadowing gotcha
- [project_spotnik_feature36_complete.md](project_spotnik_feature36_complete.md) — Feature 36 (Command Safety & Error Handling): data race fix, errNilClient sentinel, consecutivePlaybackErrors counter, DevicesLoadErrorMsg removal
- [project_spotnik_feature37_complete.md](project_spotnik_feature37_complete.md) — Feature 37 (Gateway Hardening): atomic.Pointer, timer leaks fix (explicit Stop() not defer), nil guard, doNoContent error, parseRetryAfter shared helper, always-clone body
- [project_spotnik_feature39_complete.md](project_spotnik_feature39_complete.md) — Feature 39 (Idle Polish & Test Coverage): WindowSizeMsg idle reset, backoff toast on idle-return, nilPlaybackStateTicks counter, two-pass toast pattern, elm purity coverage gap tests
