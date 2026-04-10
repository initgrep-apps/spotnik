
## StateReader and auth test coverage refinements (PR #146 review findings)
**Found:** 2026-04-10 | **Source:** PR #146 Review
**Feature:** 22-developer-foundations

Items to log:
1. `internal/ui/panes/nowplaying_test.go` — `testStateWriter` embeds full `*state.Store`; a narrower `playbackWriter` interface (just `SetPlaybackState`) would document intent more precisely and prevent future tests from calling unrelated write methods.
2. `internal/api/auth_test.go` — no public-boundary test for nil-client propagation through `ExchangeCode` or `Refresh`; the internal test in `auth_internal_test.go` covers the guard, but a higher-level test would catch any future wrapping regression.

---

## Test coverage gaps in devices.go (PR #145 review findings)
**Found:** 2026-04-10 | **Source:** PR #145 Review
**Feature:** 22-developer-foundations

Items to log:
1. `internal/api/devices_test.go` — `TransferPlayback` with `play: false` is untested; only the `true` path is exercised. Adding a second table row would pin the serialisation contract.
2. `internal/api/devices_test.go` — `{"devices": null}` JSON response not tested; only the `[]` case is covered. A `devices_null.json` fixture and `TestGetDevices_NullDevicesField` would pin the nil-guard on line 49 of devices.go.
3. `internal/testhelpers/fixtures_test.go:22` — `assert.Contains(…, "{")` is a weak JSON validity check; `json.Valid(data)` would be more precise.
