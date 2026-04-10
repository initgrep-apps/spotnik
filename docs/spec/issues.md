
## Test coverage gaps in devices.go (PR #145 review findings)
**Found:** 2026-04-10 | **Source:** PR #145 Review
**Feature:** 22-developer-foundations

Items to log:
1. `internal/api/devices_test.go` — `TransferPlayback` with `play: false` is untested; only the `true` path is exercised. Adding a second table row would pin the serialisation contract.
2. `internal/api/devices_test.go` — `{"devices": null}` JSON response not tested; only the `[]` case is covered. A `devices_null.json` fixture and `TestGetDevices_NullDevicesField` would pin the nil-guard on line 49 of devices.go.
3. `internal/testhelpers/fixtures_test.go:22` — `assert.Contains(…, "{")` is a weak JSON validity check; `json.Valid(data)` would be more precise.
