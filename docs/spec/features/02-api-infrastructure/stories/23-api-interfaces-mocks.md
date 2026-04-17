---
title: "API Client Interfaces & Mocks"
feature: 11-api-gateway
status: done
---

## Background
The architecture spec calls for a SpotifyClient interface but none exists in the codebase. Instead, app.go uses concrete client types and nil-guards in every build*Cmd function. This prevents app-level integration tests with injected fakes, testing error handling at the app level, and detecting missing client initialization at compile time.

## Design

### Per-Domain Interfaces
Define an interface for each of the 6 API clients: PlayerAPI, LibraryAPI, SearchAPI, DevicesAPI, UserAPI, PlaylistsAPI. Match exact method signatures of existing concrete clients. Add compile-time interface satisfaction checks.

### App Struct Update
Change App struct fields from concrete types to interface types. Update Set* methods and cmd/root.go.

### Mock Clients
Create mock implementations in `internal/api/mock_client.go` with result+err pairs per method and Called bools per mutating method.

### Nil-Guard Removal
With interfaces in place, the nil-guard pattern (`if client == nil { return empty }`) is removed from all 18 build*Cmd functions.

## Acceptance Criteria
- [ ] 6 per-domain interfaces defined in `internal/api/`
- [ ] Compile-time interface checks for all 6 concrete clients
- [ ] `App` struct uses interface types, not concrete types
- [ ] `MockPlayer`, `MockLibrary`, etc. exist in `internal/api/mock_client.go`
- [ ] Nil-guard pattern removed from all build*Cmd functions
- [ ] All existing tests pass
- [ ] `make ci` passes

## Tasks
- [ ] Define per-domain interfaces in api/ with compile-time checks
      - test: compile-time checks verify interface satisfaction
- [ ] Update app.go to use interfaces instead of concrete types
      - test: all existing tests pass -- concrete types satisfy interfaces
- [ ] Create mock clients in internal/api/mock_client.go
      - test: compile-time checks for all mocks
- [ ] Remove nil-guard pattern from all build*Cmd functions
      - test: existing tests pass
