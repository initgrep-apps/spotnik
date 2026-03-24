---
name: Feature 25 (API DRY Refactoring)
description: What was refactored, key patterns established, test update lessons
type: project
---

Feature 25 refactored the API layer to eliminate duplication across 6 clients.

**What was built:**
- `api/base.go`: `BaseClient` struct with `newRequest`, `doJSON`, `doNoContent`. All 6 clients embed it.
- `api/pagination.go`: Generic `fetchAll[T]` helper with `fetchPage` callback pattern.
- `keychain_integration_test.go`: OS keychain tests gated with `//go:build integration`.

**Key decisions:**
- Feature spec referenced `TokenProvider` from Feature 24, but Feature 24 used `accessToken string` directly. `BaseClient` uses `accessToken string` to match actual codebase.
- `BaseClient` fields are lowercase (unexported): `baseURL`, `accessToken`, `http`. Tests in the same package access them directly.
- The `setHTTPClient` method on `BaseClient` is unexported — each client wraps it with its own exported `SetHTTPClient`.
- `DevicesClient` used `token` field name while others used `accessToken` — had to update `devices_test.go` from `client.token` to `client.accessToken`.
- Library tests checked for "429" in error strings, but `BaseClient.doJSON` returns typed `*RateLimitError`. Updated tests to use `errors.As`.
- Task 4 (remove `Get` prefix from getters) was skipped per user instruction — too risky/mechanical.
- Adding `//go:build integration` to the whole `keychain_test.go` would drop coverage below 80%. Solution: split into `keychain_test.go` (InMemory* tests, no tag) and `keychain_integration_test.go` (KeychainTokenStore* tests, tagged).

**Why:** ~150 lines of duplicated HTTP helper code existed across 6 files. Extraction eliminates copy-paste drift and centralizes typed error handling.
