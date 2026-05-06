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
