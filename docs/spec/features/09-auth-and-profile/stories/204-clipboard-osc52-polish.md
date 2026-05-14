---
title: "Fix: Clipboard OSC 52 review polish"
feature: 09-auth-and-profile
status: done
---

## Background

Seven follow-up items from the PR #267 multi-agent review of story 196.
None blocked merge; bundled here as a single small story.

Root causes / observations:
- `routing.go:510` comment incorrectly frames `c` as a "valid hex character" — the
  `FormField` does not key on hex; the real invariant is "pass through once the user
  has started typing so they can edit freely."
- `routing.go:547` leaks the OSC 52 transport detail that already lives in
  `copyToClipboardCmd`'s own doc comment.
- `clipboard_internal_test.go:74` comment "Reset form per upstream:" cites nothing
  concrete; the byte-equality assertion already speaks for itself.
- `captureStderr` is not panic-safe: if `fn()` panics, `w.Close()` is skipped and
  the reader goroutine blocks forever.
- `TestCopyToClipboardCmd_brokenStderr_returnsError` doesn't lock in the
  `"emitting OSC 52"` wrap prefix, so a rename would silently pass.
- Two routing scenarios have no test coverage:
  - `stepError` with `c` — no clipboard cmd, no panic.
  - `stepRegister` with `c` after the user typed then deleted all input —
    `Value() == ""` again, so the copy should re-dispatch.
- `clipboardCopiedMsg` carries only `Err`; adding `Text string` aligns it with
  the project's `Data + Err` convention, lets routing tests assert the URL without
  capturing stderr, and documents "what was copied" in the type.

## Design

### `internal/app/routing.go` — comment fixes

**Line 510 (`stepRegister` guard):**
```go
// Before:
// 'c' copies the redirect URI — only when the input field is empty.
// Once the user starts typing, 'c' is a valid hex character and must pass through.

// After:
// 'c' copies the redirect URI — only when the input field is empty.
// Once the user starts typing, treat 'c' as ordinary input so they can edit freely.
```

**Line 547 (`stepOAuth` c-key):**
```go
// Before:
// c → copy auth URL to clipboard via OSC 52; toast is emitted by the clipboardCopiedMsg handler.

// After:
// c → copy auth URL to clipboard; toast is emitted by the clipboardCopiedMsg handler.
```

### `internal/app/clipboard_internal_test.go` — helper + test fixes

**`captureStderr` — panic-safe:**
Wrap the `w.Close()` + `os.Stderr = orig` restore in a single `defer` so the pipe
is cleaned up even if `fn()` panics:
```go
defer func() {
    w.Close()
    os.Stderr = orig
}()
fn()
return <-done
```

**`TestCopyToClipboardCmd_emptyText_emitsResetSequence` — drop vague comment:**
Remove the `// Reset form per upstream:` line (line 74). The byte-equality
assertion is self-documenting.

**`TestCopyToClipboardCmd_brokenStderr_returnsError` — lock in wrap prefix:**
```go
assert.Error(t, copied.Err)
assert.True(t, strings.Contains(copied.Err.Error(), "emitting OSC 52"),
    "Err must include wrap prefix; got: %v", copied.Err)
```
Add `"strings"` to imports.

### `internal/app/clipboard_routing_internal_test.go` — two new tests

```go
// TestHandleOnboardingKey_stepError_c_noCopyAndNoPanic confirms that 'c' in
// stepError dispatches no command and does not panic.
func TestHandleOnboardingKey_stepError_c_noCopyAndNoPanic(t *testing.T) {
    a := newTestApp(false)
    a.currentView = viewOnboarding
    a.onboardingStep = stepError

    _, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
    assert.Nil(t, cmd)
}

// TestHandleOnboardingKey_stepRegister_c_afterDeleteAll_dispatchesCopy confirms that
// 'c' still copies after the user typed and then deleted all input.
func TestHandleOnboardingKey_stepRegister_c_afterDeleteAll_dispatchesCopy(t *testing.T) {
    a := newTestApp(false)
    a.currentView = viewOnboarding
    a.onboardingStep = stepRegister
    a.onboardingPort = 9090
    a.onboardingField.SetValue("abc123")
    a.onboardingField.SetValue("")

    _, cmd := a.handleOnboardingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
    require.NotNil(t, cmd, "stepRegister 'c' on empty field (after delete-all) must dispatch copy")
    msg := cmd()
    copied, ok := msg.(clipboardCopiedMsg)
    require.True(t, ok)
    assert.NoError(t, copied.Err)
}
```

### `internal/app/clipboard.go` — add `Text` to `clipboardCopiedMsg`

```go
type clipboardCopiedMsg struct {
    Text string // populated with the copied string; empty for clipboard-reset sequences
    Err  error
}

func copyToClipboardCmd(text string) tea.Cmd {
    return func() tea.Msg {
        _, err := fmt.Fprint(os.Stderr, ansi.SetSystemClipboard(text))
        if err != nil {
            return clipboardCopiedMsg{Err: fmt.Errorf("emitting OSC 52: %w", err)}
        }
        return clipboardCopiedMsg{Text: text}
    }
}
```

Update any existing test assertions that use the zero-value `clipboardCopiedMsg{}`
on success paths to use `clipboardCopiedMsg{Text: ""}` (empty-payload / reset case)
or simply drop the `Text` assertion where `Err` is already the focus.

## Acceptance Criteria

- [ ] `routing.go:510` comment says "edit freely", not "hex character"
- [ ] `routing.go:547` comment does not mention "OSC 52"
- [ ] `captureStderr` uses defer — no goroutine leak on panic
- [ ] Vague `// Reset form per upstream:` comment removed from test line 74
- [ ] `TestCopyToClipboardCmd_brokenStderr_returnsError` asserts `"emitting OSC 52"` in error
- [ ] `TestHandleOnboardingKey_stepError_c_noCopyAndNoPanic` passes
- [ ] `TestHandleOnboardingKey_stepRegister_c_afterDeleteAll_dispatchesCopy` passes
- [ ] `clipboardCopiedMsg.Text` holds the copied string on success paths
- [ ] `make ci` passes

## Tasks

- [ ] Reword routing.go:510 and routing.go:547 comments
      - test: `make lint` passes

- [ ] Make `captureStderr` panic-safe; remove vague comment at test line 74
      - test: `go test ./internal/app/ -run TestCopyToClipboard -v` → all PASS

- [ ] Add `strings.Contains` assertion to `TestCopyToClipboardCmd_brokenStderr_returnsError`
      - test: `go test ./internal/app/ -run TestCopyToClipboardCmd_brokenStderr -v` → PASS

- [ ] Add two routing tests to `clipboard_routing_internal_test.go`
      - test: `go test ./internal/app/ -run TestHandleOnboardingKey -v` → all PASS

- [ ] Add `Text string` to `clipboardCopiedMsg`; populate in `copyToClipboardCmd`;
      update affected zero-value assertions in existing tests
      - test: `go test ./internal/app/ -v` → all PASS

- [ ] `make ci` passes
