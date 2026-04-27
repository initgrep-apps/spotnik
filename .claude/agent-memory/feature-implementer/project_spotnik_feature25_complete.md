---
name: Feature 25 (API DRY Refactoring)
description: What was refactored, key patterns established, test update lessons
type: project
---

Feature 25 refactored API layer, eliminated duplication across 6 clients.

**Built:**
- `api/base.go`: `BaseClient` struct w/ `newRequest`, `doJSON`, `doNoContent`. All 6 clients embed.
- `api/pagination.go`: Generic `fetchAll[T]` helper, `fetchPage` callback pattern.
- `keychain_integration_test.go`: OS keychain tests gated `//go:build integration`.

**Decisions:**
- Spec referenced `TokenProvider` from Feature 24, but Feature 24 used `accessToken string` directly. `BaseClient` uses `accessToken string` to match codebase.
- `BaseClient` fields lowercase (unexported): `baseURL`, `accessToken`, `http`. Same-package tests access directly.
- `setHTTPClient` on `BaseClient` unexported — each client wraps w/ own exported `SetHTTPClient`.
- `DevicesClient` used `token` field, others `accessToken` — updated `devices_test.go` from `client.token` to `client.accessToken`.
- Library tests checked "429" in error strings, but `BaseClient.doJSON` returns typed `*RateLimitError`. Updated tests to `errors.As`.
- Task 4 (drop `Get` prefix from getters) skipped per user — too risky/mechanical.
- `//go:build integration` on whole `keychain_test.go` drops coverage <80%. Fix: split into `keychain_test.go` (InMemory* tests, untagged) + `keychain_integration_test.go` (KeychainTokenStore* tests, tagged).

**Why:** ~150 lines duplicated HTTP helper code across 6 files. Extraction kills copy-paste drift, centralizes typed error handling.