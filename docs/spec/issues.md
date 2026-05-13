# Spotnik — Issues / Follow-ups

> Placeholder for unresolved items captured during PR reviews and triage.
> Triage into feature stories when ready to fix.

---

## Clipboard OSC 52 — review polish

**Found:** 2026-05-07 | **Source:** PR #267 Review
**Feature:** 09-auth-and-profile

Suggestion-tier follow-ups from the PR #267 multi-agent review. None
block merge; bundle into a single small story when convenient.

Items to log:

1. `internal/app/routing.go:510` — comment claims `c` "is a valid hex
   character" so must pass through to the input. Hex framing is rot-prone
   (the FormField doesn't actually key on hex). Reword to "once the input
   is non-empty, treat 'c' as ordinary input so the user can edit freely."
2. `internal/app/routing.go:533` — `c → copy auth URL to clipboard via OSC 52`
   leaks transport detail that already lives on `copyToClipboardCmd`'s doc.
   Trim to `c → copy auth URL; toast emitted by the clipboardCopiedMsg
   handler, not here.` Same callsite-comment inconsistency exists at the
   `viewAuth` site (no comment) — pick one or neither.
3. `internal/app/clipboard_internal_test.go:74` — `// Reset form per upstream:`
   is vague. Either cite "XTerm Control Sequences spec: empty payload is
   the documented reset form" or drop the rationale and let the byte
   equality assertion speak for itself.
4. `internal/app/clipboard_internal_test.go` `captureStderr` — if `fn()`
   panics, `w.Close()` is skipped and the reader goroutine blocks
   forever. Wrap the close + restore in `defer` to make the helper
   panic-safe.
5. `internal/app/clipboard_internal_test.go` `TestCopyToClipboardCmd_brokenStderr_returnsError`
   — assert `strings.Contains(copied.Err.Error(), "emitting OSC 52")` so
   the wrap prefix is locked in (CLAUDE.md says wrap errors with context).
6. Missing edge-case tests:
   - `stepError` with `c` key — pin "no clipboard cmd, no panic" so a
     future story that adds copy-error-text doesn't slip in silently.
   - `stepRegister` with `c` after the user typed then deleted everything
     (`onboardingField.Value() == ""` again) — should re-dispatch the copy.
7. `clipboardCopiedMsg` payload — only carries `Err`. Adding `Text string`
   would (a) match the project's `Data + Err` convention used by every
   other Msg, (b) let routing tests assert the URL without redirecting
   stderr, and (c) document the "what was copied" semantics in the type
   itself.

---

## Volume bar snap-back — review polish

**Found:** 2026-05-08 | **Source:** PR #271 Review
**Feature:** 03-playback

Suggestion-tier follow-ups from the PR #271 multi-agent review. None
block merge; bundle into a single small story when convenient.

Items to log:

1. Missing app-level test for `VolumeAppliedMsg` with 401 Unauthorized —
   verify the handler routes to `unauthorizedMsg{}` token-refresh path.
2. Missing app-level test for `VolumeAppliedMsg` with generic (non-429,
   non-401) error — verify the `tea.Batch` dispatches both
   `fetchPlaybackStateCmd` and the "Volume change failed" toast.
3. `TestApp_VolumeDebounce_FiveRapidPresses_SendsOneCall` — `MockPlayer`
   uses a boolean `SetVolumeCalled` flag, so the test proves "at least one
   call" not "exactly one call". Add `SetVolumeCallCount int` to the mock
   and assert equality.
4. `TestApp_VolumeAppliedMsg_429_ClearsPendingAndBacksOff` — executes the
   final command but never inspects the `NowPlayingPane` to confirm
   `hasPending` was cleared. Assert via `View()` that the bar is no longer
   pending after the message is handled.
5. Missing `NowPlayingPane` test for stale `VolumeAppliedMsg` (seq mismatch)
   — prime a burst, start a second burst, feed the first burst's
   `VolumeAppliedMsg`, assert the bar stays in the second burst's pending
   state.
6. `TestApp_VolumeAppliedMsg_Success_DispatchesInteractivePoll` — only
   asserts `NotNil`. Execute the returned command and verify it produces
   a `PlaybackStateFetchedMsg` (or at least is not a no-op).
7. `VolumeAppliedMsg` handler in `handlers.go` discards the pane command
   with `updated, _ := np.Update(m)`. Today harmless (pane returns nil),
   but a latent bug if a future refactor adds a command to the pane's
   `VolumeAppliedMsg` handler. Capture `cmd` and batch it with downstream
   effects.
8. `VolumeAppliedMsg` zero-value ambiguity — `VolumeAppliedMsg{}` reads as
   "volume successfully set to 0%". Consider adding constructor functions
   `NewVolumeAppliedMsg(vol, seq int)` and `NewVolumeAppliedError(seq int,
   err error)` to make the success/failure duality explicit at call sites.

---

## Story 199 polling tests — review polish

**Found:** 2026-05-13 | **Source:** PR #278 Review
**Feature:** 15-error-resilience

Suggestion-tier follow-ups from the PR #278 multi-agent review. None
block merge; bundle into a future story when convenient.

Items to log:

1. `internal/app/poll_test.go` `collectAllMsgs` only resolves one level of
   batch nesting. Currently safe (TickMsg handler returns a flat batch), but
   fragile if a future story wraps fetch commands in an inner batch. Make it
   recursive like `collectInitMsgs` in `app_test.go`.
2. `internal/app/poll_test.go` `TestApp_TickMsg_LibraryPollDispatchesAtTick0`
   uses a disjunctive `hasLibraryMsg` helper that passes if any one of the
   five library message types appears — four panes could silently fail. Change
   to count distinct dispatched message types (or check all five explicitly)
   to prove the full dispatch table fires.
