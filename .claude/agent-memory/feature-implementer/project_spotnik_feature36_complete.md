---
name: project_spotnik_feature36_complete
description: Feature 36 (Command Safety & Error Handling): data race fix, errNilClient sentinel, playback error counter, DevicesLoadErrorMsg removal
type: project
---

## Feature 36 — Command Safety & Error Handling

**What was built:**
- Fixed data race in `buildPlaybackAPICmd`: store values (`VolumePercent`, `ShuffleState`, `RepeatState`) were read inside goroutine closures. Now snapshotted in function body (Update() context, thread-safe) before returning the closure.
- Added `errNilClient` sentinel in `commands.go`; all 7 specified nil-client fallbacks now set `Err: errNilClient`.
- Update() handlers for all 6 affected message types check `errors.Is(m.Err, errNilClient)` and skip silently (no toast) — nil client is an expected startup condition.
- Added `consecutivePlaybackErrors int` field to App struct; warning toast emitted on exactly the 5th consecutive `PlaybackStateFetchedMsg` error; counter resets to 0 on success.
- Removed `DevicesLoadErrorMsg` dead code: type from messages.go, handler from app.go, test from toast_routing_test.go.
- Fixed stale `SetDevicesFetchedAt` comment in store.go (now calls root app.Update()).
- Fixed stale `devicesLoadedMsg` reference in ARCHITECTURE.md (now `DevicesLoadedMsg`).

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/commands.go` — errNilClient sentinel (line 29), nil-client fix in buildPlaybackAPICmd (early return with errNilClient), 6 other nil-client returns updated
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/app.go` — consecutivePlaybackErrors field (line 151), PlaybackStateFetchedMsg handler updated (line 715), 6 errNilClient guards in handlers
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/command_safety_test.go` — 17 new tests

**Patterns established:**
- The 7 nil-client sites from spec: buildPlaybackAPICmd, buildFetchPlaylistsCmd, buildFetchAlbumsCmd, buildFetchLikedTracksCmd, buildFetchRecentlyPlayedCmd, buildSearchCmd, fetchQueueCmd. Other nil-client checks (buildPlayContextCmd, buildPlayTrackCmd etc.) were intentionally NOT in scope per spec.
- `errors.Is(m.Err, errNilClient)` guard pattern for skipping toasts on startup errors.
- `consecutivePlaybackErrors == 5` (exact equality, not >=) so only the 5th fires the toast, not subsequent ones.

**Gotchas:**
- Two test stubs (`TestNilClientFallback_SearchCmd` and `TestNilClientFallback_QueueCmd`) were written as `_ = a` placeholders during initial TDD. Caught in self-review and replaced with substantive tests verifying real errors still toast.
- `buildPlayContextCmd` and `buildPlayTrackCmd` still have nil-player checks returning empty `PlaybackCmdSentMsg{}` without errNilClient — this is intentional (not in spec's 7 sites) and low-risk since the handler only acts on non-nil Err.
- `errNilClient` is unexported so tests use the two-pass approach: trigger the cmd, execute it, feed the result message back through Update(), then assert no cmd returned.
- `fetchQueueCmd` is a package-level function (not a method), so the nil-client path is tested via the Update() handler rather than directly through message dispatch.

**Testing notes:**
- 17 tests added in `command_safety_test.go`
- `go test -race ./internal/app/...` passes (no races)
- Coverage: 82.5% (threshold: 80%)
- PR: https://github.com/initgrep-apps/spotnik/pull/41
