# Issues Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Resolve all 33 confirmed issues from `docs/issues.md` that arose during PR reviews of features 28-33.

**Architecture:** Issues are grouped into 6 features (F34-F39) by subsystem. Each feature is a self-contained branch with atomic changes. Features are ordered to minimize merge conflicts: docs first, then type changes, then behavioral fixes.

**Tech Stack:** Go 1.22+, Bubble Tea, BubbleUp, testify

---

## Feature Grouping

| Feature | Name | Issues | Depends On |
|---------|------|--------|------------|
| 34 | Docs, Dead Code & Defensive Init | 5 | — |
| 35 | Type Design Alignment | 4 | 34 |
| 36 | Command Safety & Error Handling | 3 | 35 |
| 37 | Gateway Hardening | 7 | — |
| 38 | Notification & Staleness Hardening | 7 | 36 |
| 39 | Idle Polish & Test Coverage | 8 | 38 |

**Parallelism:** F34 and F37 are independent and can run in parallel. After F34, F35→F36→F38→F39 are sequential. However, per project rules, features must be implemented sequentially (no parallel agents).

**Execution order:** F34 → F35 → F36 → F37 → F38 → F39

---

## Issue → Feature Mapping

### F34: Docs, Dead Code & Defensive Init
- PR#34 #1: store.go package doc stale
- PR#34 #2: Store struct doc stale
- PR#34 #3: Dead unmarshalJSON in api/models.go
- PR#37 #9: statsFetchedAt map not initialized in New()
- PR#34 #5: buildFetchDevicesCmd error fallthrough (FALSE — mark resolved)

### F35: Type Design Alignment
- PR#34 #12: store.go imports api/ for SearchResult
- PR#34 #13: StatsLoadedMsg in stats.go not messages.go
- PR#34 #14: AlbumsLoadedMsg missing Offset field
- PR#34 #11: Inconsistent devicesLoadedMsg encapsulation

### F36: Command Safety & Error Handling
- PR#34 #7: Store reads in buildPlaybackAPICmd goroutine closures (DATA RACE)
- PR#34 #6: Nil-client fallbacks return empty messages with no error
- PR#34 #4 + PR#36 #5: PlaybackStateFetchedMsg.Err never checked / still silent

### F37: Gateway Hardening
- PR#35 #4: SetGateway not thread-safe
- PR#35 #5: time.After timer leaks on context cancellation
- PR#35 #6: nil response from fn() causes panic
- PR#35 #2: doNoContent discards io.ReadAll error
- PR#35 #1: Double 429 parsing with inconsistent error wrapping
- PR#35 #3: Unparseable Retry-After header silently defaults
- PR#35 #7: 429 body clone for dedup waiters

### F38: Notification & Staleness Hardening
- PR#36 #2: alerts.Update() type assertion failure silently ignored
- PR#36 #3: alerts.Init() return value discarded
- PR#36 #4: No validation of alert type registration
- PR#37 #7: fetchedAt stamped on nil/empty data
- PR#37 #8: Stats double-stamped
- PR#37 #6: TOCTOU race — add fetching sentinel
- PR#37 #10: Staleness gate drops FetchPlaylistsRequestMsg

### F39: Idle Polish & Test Coverage
- PR#38 #11: Only tea.KeyMsg resets idle
- PR#38 #12: Backoff + idle-return interaction
- PR#38 #13: Nil PlaybackState unlogged
- PR#36 #1: Tests weakened to cmd != nil (5 tests)
- PR#34 #8: buildSearchCmd store isolation untested
- PR#34 #9: SearchResultsMsg error path missing
- PR#34 #10: Concurrent stats partial failure untested
- PR#34 #4: PlaybackStateFetchedMsg consecutive error tracking test

---

## Workflow Per Feature

1. Feature-implementer agent reads spec from `docs/features/NN-*.md`
2. Creates branch `feat/NN-name`, implements with TDD, conventional commits
3. Runs `make ci`, creates PR
4. Orchestrator runs `pr-review-toolkit:review-pr`
5. Critical issues → re-invoke feature-implementer with fix context
6. Minor issues → log to docs/issues.md
7. Merge PR, update overview

---

*Created: 2026-03-25*
