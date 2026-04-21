---
title: "Onboarding — Header Centering, Copy Feedback, Subtitle Text"
feature: 09-auth-and-profile
status: open
---

## Background

Post-launch testing on the onboarding screens (stories 138–139) found four issues:

1. **Header not centered** — `onboardingTitle()` uses `JoinVertical(lipgloss.Center, ...)` to
   center its two text lines relative to each other, but the `body` in both
   `renderOnboardingRegister` and `renderOnboardingOAuth` is joined with `lipgloss.Left`, so the
   title block appears left-aligned inside the outer panel. The user sees "♪  spotnik" starting
   from the left edge instead of being centered in the panel.

2. **No copy feedback on screen 2 (stepOAuth)** — pressing `c` calls `copyToClipboard()`
   silently. The user gets no visual confirmation that the URL was copied. Feedback should be a
   brief `✓ Copied!` in `Success()` colour, shown inline in the hint area.

3. **No `c` shortcut on screen 1 (stepRegister)** — the redirect URI box shows the callback URL
   the user needs to add to Spotify, but there is no keyboard shortcut to copy it. The user must
   manually select the text. Add `c` as a copy shortcut for the redirect URI with the same
   feedback pattern as screen 2.
   
   **Key conflict handling**: The text input in step 1 is focused and accepts all printable
   characters. Client IDs are 32-character hex strings (`a-f`, `0-9`) — `c` is a valid input
   character. The `c` shortcut is therefore only active when the text input is **empty**
   (no characters typed yet). Once the user starts typing, `c` is passed to the input as normal.
   Show the hint `c  copy URI` only when the input is empty; hide it once typing begins.

4. **Subtitle text** — `onboardingTitle()` currently renders "A terminal Spotify client for
   developers". Drop "for developers" to match the agreed tagline. This also applies to the
   `viewAuth` screen's `renderAuthPanel` which may include the same subtitle.

## Design

### `internal/app/app.go` — new field

Add a `onboardingCopied bool` field to `App` struct, placed alongside the other onboarding fields:

```go
onboardingCopied bool // true briefly after 'c' copies URL — triggers ✓ feedback in View()
```

Add a `copiedFeedbackMsg` type to clear the flag after a short duration:

```go
// copiedFeedbackMsg is sent after a short delay to clear the copy-feedback indicator.
type copiedFeedbackMsg struct{}
```

### `internal/app/routing.go` — key handlers

**`handleOnboardingKey` — `stepRegister` case**:

```go
case stepRegister:
    // 'c' copies redirect URI only when input is empty.
    if m.Type == tea.KeyRunes && string(m.Runes) == "c" && a.onboardingInput.Value() == "" {
        _ = copyToClipboard(fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort))
        a.onboardingCopied = true
        return a, tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
            return copiedFeedbackMsg{}
        })
    }
    if m.Type == tea.KeyEnter {
        clientID := strings.TrimSpace(a.onboardingInput.Value())
        if clientID == "" {
            return a, nil
        }
        return a, saveClientIDCmd(config.DefaultConfigPath(), clientID)
    }
    var cmd tea.Cmd
    a.onboardingInput, cmd = a.onboardingInput.Update(m)
    return a, cmd
```

**`handleOnboardingKey` — `stepOAuth` case**:

```go
case stepOAuth:
    if m.Type == tea.KeyRunes && string(m.Runes) == "c" {
        _ = copyToClipboard(a.onboardingAuthURL)
        a.onboardingCopied = true
        return a, tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
            return copiedFeedbackMsg{}
        })
    }
    return a, nil
```

**`handleMsg` — `copiedFeedbackMsg` handler** (in `handlers.go`):

```go
case copiedFeedbackMsg:
    a.onboardingCopied = false
    return a, nil
```

### `internal/app/render.go` — visual changes

**`onboardingTitle()`** — update subtitle text:

```go
subtitleStyle.Render("A terminal Spotify client")
```

Also change `JoinVertical` alignment to `lipgloss.Center` to ensure centering within the panel
(it already uses Center — the fix is in the outer body join below).

**`renderOnboardingRegister()`** — three changes:

1. Change `lipgloss.Left` to `lipgloss.Center` for the outer `body` join so the title is
   centered within the outer panel width:

```go
body := lipgloss.JoinVertical(lipgloss.Center,
    a.onboardingTitle(),
    ...
)
```

2. Add `c  copy URI` hint — visible only when input is empty:

```go
var hintLine string
if a.onboardingInput.Value() == "" {
    hintLine = hintStyle.Render("c  copy URI  ·  Enter  confirm  ·  q  quit")
} else {
    hintLine = hintStyle.Render("Enter  confirm  ·  q  quit")
}
```

Use `hintLine` in the body join (replacing the existing `"Enter  confirm  ·  q  quit"` line).

3. Show copy feedback below the URI box when `a.onboardingCopied == true`:

```go
var uriSection string
if a.onboardingCopied {
    feedbackStyle := lipgloss.NewStyle().Foreground(t.Success())
    uriSection = lipgloss.JoinVertical(lipgloss.Left,
        uriBox,
        feedbackStyle.Render("  ✓  Copied!"),
    )
} else {
    uriSection = uriBox
}
```

**`renderOnboardingOAuth()`** — two changes:

1. Change `lipgloss.Left` to `lipgloss.Center` for the outer `body` join.

2. Show copy feedback in the hint line:

```go
var hintLine string
if a.onboardingCopied {
    feedbackStyle := lipgloss.NewStyle().Foreground(t.Success())
    hintLine = feedbackStyle.Render("✓  Copied!") + hintStyle.Render("  ·  q  quit")
} else {
    hintLine = hintStyle.Render("c  copy URL  ·  q  quit")
}
```

**`renderAuthPanel`** (for `viewAuth` returning user) — update the subtitle in the title section
from "A terminal Spotify client for developers" if it appears there. Check the current
`renderAuthPanel` implementation and update any subtitle string found.

### Tests — `internal/app/render_test.go` and `internal/app/routing_test.go`

```go
func TestRenderOnboardingRegister_titleCentered(t *testing.T) {
    // Build a minimal App with known width.
    // Call renderOnboardingRegister().
    // Assert output contains "spotnik" (title renders).
    // Assert alignment is Center (approximate: title is not at column 0 of output).
}

func TestRenderOnboardingRegister_showsCopyHintWhenInputEmpty(t *testing.T) {
    // Arrange: onboardingInput is empty.
    // Assert: View() contains "c  copy URI".
}

func TestRenderOnboardingRegister_hidesCopyHintWhenInputNotEmpty(t *testing.T) {
    // Arrange: onboardingInput has some text.
    // Assert: View() does NOT contain "c  copy URI".
}

func TestRenderOnboardingOAuth_showsCopiedFeedback(t *testing.T) {
    // Arrange: a.onboardingCopied = true.
    // Assert: renderOnboardingOAuth() contains "✓".
}

func TestHandleOnboardingKey_stepRegister_c_withEmptyInput_setsFlag(t *testing.T) {
    // Arrange: App in stepRegister, empty input, known onboardingPort.
    // Act: Update(keyMsg("c")).
    // Assert: a.onboardingCopied == true; cmd fires copiedFeedbackMsg after delay.
}

func TestHandleOnboardingKey_stepRegister_c_withNonEmptyInput_passesToInput(t *testing.T) {
    // Arrange: App in stepRegister, input has text "abc".
    // Act: Update(keyMsg("c")).
    // Assert: a.onboardingCopied == false; input.Value() contains "c".
}

func TestHandleOnboardingKey_stepOAuth_c_setsFlag(t *testing.T) {
    // Arrange: App in stepOAuth, onboardingAuthURL = "https://example.com".
    // Act: Update(keyMsg("c")).
    // Assert: a.onboardingCopied == true.
}

func TestOnboardingTitle_noForDevelopers(t *testing.T) {
    // Create minimal App; call onboardingTitle().
    // Assert: does not contain "for developers".
    // Assert: contains "A terminal Spotify client".
}
```

## Acceptance Criteria

- [ ] `onboardingTitle()` renders "A terminal Spotify client" (no "for developers")
- [ ] `renderOnboardingRegister` body joined with `lipgloss.Center` — title is centered in panel
- [ ] `renderOnboardingOAuth` body joined with `lipgloss.Center` — title is centered in panel
- [ ] `c  copy URI` hint shown on step 1 when text input is empty; hidden when input has content
- [ ] Pressing `c` on step 1 with empty input: copies redirect URI, sets `onboardingCopied = true`
- [ ] Pressing `c` on step 1 with non-empty input: 'c' typed into input, `onboardingCopied` unchanged
- [ ] Step 1 shows `✓  Copied!` in `Success()` colour after 'c' press; clears after 2 seconds
- [ ] Pressing `c` on step 2: copies auth URL, shows `✓  Copied!` in hint area
- [ ] `copiedFeedbackMsg` handler resets `onboardingCopied = false`
- [ ] No hardcoded hex colour values in modified render functions
- [ ] All new `TestHandleOnboardingKey_*` and `TestRenderOnboarding*` tests pass
- [ ] `make ci` passes

## Tasks

- [ ] Add `onboardingCopied bool` field to `App` struct; add `copiedFeedbackMsg` type in
      `internal/app/app.go`
      - test: `go build ./...` → clean
- [ ] Write failing tests in `internal/app/render_test.go` and a routing test file for the 8
      test cases above
      - test: compile errors expected
- [ ] Update `onboardingTitle()` subtitle in `internal/app/render.go`; also update any
      subtitle in `renderAuthPanel` if present
      - test: `TestOnboardingTitle_noForDevelopers` → PASS
- [ ] Update `renderOnboardingRegister` and `renderOnboardingOAuth` body join alignment and copy
      feedback display in `internal/app/render.go`
      - test: `TestRenderOnboardingRegister_showsCopyHintWhenInputEmpty`,
              `TestRenderOnboardingOAuth_showsCopiedFeedback` → PASS
- [ ] Update `handleOnboardingKey` `stepRegister` and `stepOAuth` cases in
      `internal/app/routing.go` to set `onboardingCopied` and return timer command
      - test: `TestHandleOnboardingKey_*` → PASS
- [ ] Add `copiedFeedbackMsg` handler in `internal/app/handlers.go`
      - test: `go build ./...` → clean
- [ ] `make ci` passes
