---
name: project_spotnik_feature34_complete
description: Feature 34 (Docs, Dead Code & Defensive Init): store.go doc fixes, unmarshalJSON removal, statsFetchedAt init, issues.md updates
type: project
---

## Feature 34 — Docs, Dead Code & Defensive Init

**What was built:**
- Fixed 2 stale doc comments in `internal/state/store.go` (package doc + Store struct doc)
- Removed dead `unmarshalJSON` helper from `internal/api/models.go`; inlined `json.Unmarshal` in the one remaining caller (`SearchPlaylist.UnmarshalJSON` in `search.go`)
- Pre-allocated `statsFetchedAt: make(map[string]time.Time)` in `New()`; removed 4 lazy-init nil guards from `SetTopTracks`, `SetTopArtists`, `StatsFetchedAt`, and `StatsStale`
- Marked 5 issues resolved in `docs/issues.md` (issues 1, 2, 3, 5, 9 from PR reviews #34 and #37)
- 2 new TDD tests in `internal/state/store_test.go`

**Key files:**
- `internal/state/store.go` — All 4 tasks touched this file
- `internal/api/models.go` — Removed helper + unused `encoding/json` import
- `internal/api/search.go` — Added `encoding/json` import; inlined `json.Unmarshal`
- `internal/state/store_test.go` — 2 new tests for statsFetchedAt init
- `docs/issues.md` — 5 issues marked resolved

**Patterns established:**
- The spec said `unmarshalJSON` was "no longer used" but it still had one caller in `search.go`. Always grep before trusting spec claims about dead code — inline the direct call rather than deleting blindly.
- When removing lazy nil-init from a map field, also clean up ALL readers that had defensive nil checks (StatsFetchedAt, StatsStale) — they become redundant.

**Gotchas:**
- The spec said `unmarshalJSON` was unused but `api/search.go` line 77 still called it. The fix was to inline `json.Unmarshal` in `search.go` and then remove the helper — not just delete the helper (that would have broken the build).
- `topTracks` and `topArtists` maps still use lazy-init guards (nil check before `make`) in `SetTopTracks`/`SetTopArtists` — only `statsFetchedAt` was pre-allocated. Keep the others lazy for now since no spec asked to change them.

**Testing notes:**
- TDD: `TestStore_New_HasInitializedStatsFetchedAt` checked the unexported field directly (package-internal test) — valid since `store_test.go` is in `package state`
- `TestStore_StatsStale_NeverFetched` already existed and tested the nil-map path via the nil guard; after removing the guard both it and the new test pass cleanly
- Coverage: state 99.6%, api 84.5%, total 82.9%
