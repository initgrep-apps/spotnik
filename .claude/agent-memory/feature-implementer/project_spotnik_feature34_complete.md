Inline text — no file to process. Apply compression rules directly.

---
name: project_spotnik_feature34_complete
description: Feature 34 (Docs, Dead Code & Defensive Init): store.go doc fixes, unmarshalJSON removal, statsFetchedAt init, issues.md updates
type: project
---

## Feature 34 — Docs, Dead Code & Defensive Init

**What was built:**
- Fixed 2 stale doc comments in `internal/state/store.go` (package doc + Store struct doc)
- Removed dead `unmarshalJSON` helper from `internal/api/models.go`; inlined `json.Unmarshal` in remaining caller (`SearchPlaylist.UnmarshalJSON` in `search.go`)
- Pre-allocated `statsFetchedAt: make(map[string]time.Time)` in `New()`; removed 4 lazy-init nil guards from `SetTopTracks`, `SetTopArtists`, `StatsFetchedAt`, `StatsStale`
- Marked 5 issues resolved in `docs/issues.md` (issues 1, 2, 3, 5, 9 from PR reviews #34, #37)
- 2 new TDD tests in `internal/state/store_test.go`

**Key files:**
- `internal/state/store.go` — all 4 tasks touched this file
- `internal/api/models.go` — removed helper + unused `encoding/json` import
- `internal/api/search.go` — added `encoding/json` import; inlined `json.Unmarshal`
- `internal/state/store_test.go` — 2 new tests for statsFetchedAt init
- `docs/issues.md` — 5 issues marked resolved

**Patterns established:**
- Spec said `unmarshalJSON` "no longer used" but still had caller in `search.go`. Grep before trusting spec dead-code claims — inline direct call, don't delete blindly.
- Removing lazy nil-init from map field? Clean up ALL readers with defensive nil checks (StatsFetchedAt, StatsStale) — now redundant.

**Gotchas:**
- Spec said `unmarshalJSON` unused but `api/search.go` line 77 called it. Fix: inline `json.Unmarshal` in `search.go`, then remove helper — not just delete (would break build).
- `topTracks`, `topArtists` maps still use lazy-init guards (nil check before `make`) in `SetTopTracks`/`SetTopArtists` — only `statsFetchedAt` pre-allocated. Keep others lazy; no spec asked to change.

**Testing notes:**
- TDD: `TestStore_New_HasInitializedStatsFetchedAt` checks unexported field directly (package-internal test) — valid since `store_test.go` in `package state`
- `TestStore_StatsStale_NeverFetched` existed, tested nil-map path via nil guard; after removing guard, both it + new test pass cleanly
- Coverage: state 99.6%, api 84.5%, total 82.9%