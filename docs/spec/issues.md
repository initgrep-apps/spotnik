## Stale test comments — fetchPlaybackStateCmd signature
**Found:** 2026-04-15 | **Source:** PR #163 Review
**Feature:** 26-playback-correctness

Two test comments reference the old `fetchPlaybackStateCmd` signature:
- `internal/app/command_safety_test.go:373` — comment says `fetchPlaybackStateCmd(nil)` (old 1-arg form); correct is `fetchPlaybackStateCmd(nil, api.Background)`
- `internal/app/app_test.go:458` — comment says `fetchPlaybackStateCmd(nil, store)` (wrong second arg); correct is `fetchPlaybackStateCmd(nil, api.Background)`

Both are comment-only (non-executable), harmless but misleading to future readers.
