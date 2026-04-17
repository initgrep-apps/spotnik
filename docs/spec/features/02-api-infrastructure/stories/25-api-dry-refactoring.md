---
title: "API DRY Refactoring"
feature: 11-api-gateway
status: done
---

## Background
The 6 API client files share ~150 lines of duplicated HTTP helper code (newRequest, doJSON, doNoContent). The architecture spec defines a generic fetchAll[T] pagination helper that was never implemented. Go convention (Effective Go) says getters should not have a Get prefix. Keychain tests need integration build tags.

## Design

### BaseClient Extraction
Create a shared `BaseClient` struct in `internal/api/base.go` with `NewRequest()`, `DoJSON()`, `DoNoContent()` methods. Refactor all 6 clients to embed `BaseClient`.

### fetchAll[T] Pagination
```go
func fetchAll[T any](ctx context.Context, maxItems int, fetchPage func(ctx context.Context, offset int) ([]T, int, error)) ([]T, error)
```
Fetches all pages of a paginated endpoint with a safety cap.

### Get Prefix Removal
Rename all getter methods: GetPlaybackState -> PlaybackState, GetPlaylists -> Playlists, etc. Mechanical rename across 17 files, ~64 edits.

### Build Tags
Add `//go:build integration` to keychain tests.

## Acceptance Criteria
- [ ] `BaseClient` struct in `api/base.go` with shared HTTP helpers
- [ ] All 6 clients embed `BaseClient`, zero duplicated HTTP code
- [ ] `fetchAll[T]` generic helper in `api/pagination.go`
- [ ] Zero `Get` prefixed getter methods on API clients
- [ ] All callers updated with new method names
- [ ] `keychain_test.go` has `//go:build integration` tag
- [ ] All existing tests pass
- [ ] `make ci` passes

## Tasks
- [ ] Extract shared BaseClient in internal/api/base.go
      - test: BaseClient.NewRequest sets correct auth header
      - test: BaseClient.DoJSON returns typed errors for 401/403/429
- [ ] Implement fetchAll[T] pagination helper in internal/api/pagination.go
      - test: fetchAll with 3 pages returns all items; stops at maxItems; handles empty first page; propagates errors
- [ ] Remove Get prefix from API getters across all clients and callers
      - test: all existing tests pass with updated method names; go build clean
- [ ] Add //go:build integration to keychain tests
      - test: go test without -tags integration skips keychain tests
