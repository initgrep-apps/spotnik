---
title: "Onboarding — Header Centering, Copy Feedback, Subtitle Text"
feature: 09-auth-and-profile
status: done
---

## Background

Post-launch testing on the onboarding screens (stories 138–139) found four issues:

1. **Header not centered** — `onboardingTitle()` uses `JoinVertical(lipgloss.Center, ...)` to
   center its two text lines *relative to each other*, but in both `renderOnboardingRegister`
   and `renderOnboardingOAuth` the full body is composed with `lipgloss.Left`. The result is
   that "♪  spotnik" and the subtitle appear left-aligned inside the panel. The user sees the
   title floating near the left edge instead of visually centered.

2. **No copy feedback on either screen** — Pressing `c` on screen 2 (stepOAuth) calls
   `copyToClipboard()` silently; the user gets no confirmation. Screen 1 (stepRegister) has no
   `c` shortcut at all for the redirect URI — the user must manually select text from the box.

3. **Key conflict on screen 1** — the text input in stepRegister accepts all printable chars
   including `c` (valid in 32-char hex client IDs like `abc123...`). The `c` shortcut must
   therefore be guarded: active **only when the input is empty**. Once the user starts typing,
   `c` is passed to the input as normal. The `c  copy URI` hint is hidden once typing begins.

4. **Subtitle says "for developers"** — `onboardingTitle()` renders "A terminal Spotify client
   for developers". `renderAuthPanel` in `auth.go` may also include this string. Per the agreed
   tagline, drop "for developers".

**Depends on:** Stories 138 (onboarding handlers/routing), 139 (onboarding render screens).

## Design

### `internal/app/app.go` — new field and message type

Add `onboardingCopied bool` alongside existing onboarding fields:

```go
// onboardingCopied is set true briefly after 'c' copies a URL or URI.
// Cleared by copiedFeedbackMsg after 2 seconds.
onboardingCopied bool
```

Add the clear message type (alongside other onboarding message types in `app.go` or `auth.go`):

```go
// copiedFeedbackMsg clears the copy-confirmation indicator after a short delay.
type copiedFeedbackMsg struct{}
```

### `internal/app/handlers.go` — copiedFeedbackMsg handler

```go
case copiedFeedbackMsg:
    a.onboardingCopied = false
    return a, nil
```

### `internal/app/routing.go` — `handleOnboardingKey` updates

**`stepRegister` case** — add `c` shortcut guarded by empty input:

```go
case stepRegister:
    // 'c' copies the redirect URI — only when the input field is empty.
    // Once the user starts typing, 'c' is a valid hex character and must pass through.
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

**`stepOAuth` case** — add copy feedback:

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

Add `"time"` to imports in `routing.go` if not already present.

### `internal/app/render.go` — centering and copy feedback

**`onboardingTitle()`** — subtitle text change:

```go
func (a *App) onboardingTitle() string {
    titleStyle := lipgloss.NewStyle().
        Foreground(a.theme.TextPrimary()).
        Bold(true)
    subtitleStyle := lipgloss.NewStyle().
        Foreground(a.theme.TextMuted())

    return lipgloss.JoinVertical(lipgloss.Center,
        titleStyle.Render("♪  spotnik"),
        subtitleStyle.Render("A terminal Spotify client"),  // drop "for developers"
    )
}
```

**`renderOnboardingRegister()`** — center title and add copy feedback to hint:

To center the title inside the panel, compute `panelInnerWidth` (based on `a.width`) and wrap
the title block in a fixed-width style before joining:

```go
// Determine panel inner width (same calculation used for outerBorder).
panelInnerWidth := a.width - 8
if panelInnerWidth < 72 {
    panelInnerWidth = 72
}

centeredTitle := lipgloss.NewStyle().
    Width(panelInnerWidth).
    Align(lipgloss.Center).
    Render(a.onboardingTitle())
```

Replace the hint line with feedback-aware version:

```go
var hintLine string
if a.onboardingCopied {
    hintLine = lipgloss.NewStyle().Foreground(a.theme.Success()).Render("✓  Copied!")
} else if a.onboardingInput.Value() == "" {
    hintLine = hintStyle.Render("c  copy URI  ·  Enter  confirm  ·  q  quit")
} else {
    hintLine = hintStyle.Render("Enter  confirm  ·  q  quit")
}
```

Replace `a.onboardingTitle()` in `body` with `centeredTitle`.

**`renderOnboardingOAuth()`** — center title and add copy feedback to hint:

Apply the same centering pattern for `panelInnerWidth`. Replace hint line:

```go
var hintLine string
if a.onboardingCopied {
    hintLine = lipgloss.NewStyle().Foreground(a.theme.Success()).Render("✓  Copied!")
} else {
    hintLine = hintStyle.Render("c  copy URL  ·  q  quit")
}
```

### `internal/app/auth.go` — subtitle in `renderAuthPanel`

In `renderAuthPanel`, if the subtitle "A terminal Spotify client for developers" appears, change
to "A terminal Spotify client". (Verify presence — the function may not include a subtitle, but
check and fix if it does.)

### Tests — `internal/app/render_test.go`

```go
func TestOnboardingTitle_noForDevelopers(t *testing.T) {
    a := &App{theme: theme.NewBlack()}
    title := a.onboardingTitle()
    assert.NotContains(t, title, "for developers")
    assert.Contains(t, title, "A terminal Spotify client")
}

func TestRenderOnboardingRegister_copyHint_shownWhenEmpty(t *testing.T) {
    a := newTestApp()
    // Input is empty — copy hint should appear.
    view := a.renderOnboardingRegister()
    assert.Contains(t, view, "copy URI")
}

func TestRenderOnboardingRegister_copyHint_hiddenWhenTyping(t *testing.T) {
    a := newTestApp()
    a.onboardingInput.SetValue("abc")
    view := a.renderOnboardingRegister()
    assert.NotContains(t, view, "copy URI")
}

func TestRenderOnboardingRegister_copiedFeedback(t *testing.T) {
    a := newTestApp()
    a.onboardingCopied = true
    view := a.renderOnboardingRegister()
    assert.Contains(t, view, "✓")
    assert.Contains(t, view, "Copied")
}

func TestRenderOnboardingOAuth_copiedFeedback(t *testing.T) {
    a := newTestApp()
    a.onboardingCopied = true
    view := a.renderOnboardingOAuth()
    assert.Contains(t, view, "✓")
    assert.Contains(t, view, "Copied")
}
```

`newTestApp()` is a helper returning a minimal `*App` with theme, dimensions, and
`onboardingInput` initialised — add it to `render_test.go` if not already present.

## Acceptance Criteria

- [ ] `onboardingTitle()` subtitle reads "A terminal Spotify client" (no "for developers")
- [ ] `renderAuthPanel` subtitle (if present) also reads "A terminal Spotify client"
- [ ] Title block is visually centered inside both screen 1 and screen 2 panels
- [ ] Screen 1 (stepRegister): `c` copies redirect URI when input is empty; `c` passes through to
      input when typing has started
- [ ] Screen 1: `c  copy URI` hint shown only when input is empty
- [ ] Screen 2 (stepOAuth): pressing `c` sets `onboardingCopied = true`
- [ ] Both screens: hint line shows `✓  Copied!` in `Success()` colour for 2 seconds after copy
- [ ] `copiedFeedbackMsg` clears `onboardingCopied` after 2 seconds
- [ ] All `TestRenderOnboarding*` tests pass
- [ ] `make ci` passes

## Tasks

- [ ] Add `onboardingCopied bool` field and `copiedFeedbackMsg` type to `internal/app/app.go`
      - test: `go build ./...` → clean
- [ ] Add `copiedFeedbackMsg` handler to `internal/app/handlers.go`
      - test: `go build ./...` → clean
- [ ] Update `handleOnboardingKey` in `internal/app/routing.go`: `c` guard for stepRegister,
      copy + feedback tick for both stepRegister and stepOAuth
      - test: `go build ./...` → clean
- [ ] Write render tests in `internal/app/render_test.go`
      - test: `go test ./internal/app/... -run "TestOnboarding|TestRenderOnboarding" -v` → FAIL (before render changes)
- [ ] Update `onboardingTitle()` subtitle in `internal/app/render.go`; add centering for title
      in both render functions; update hint lines with `onboardingCopied` feedback
      - test: `TestOnboardingTitle_noForDevelopers` → PASS; `TestRenderOnboarding*` → PASS
- [ ] Check and fix subtitle in `renderAuthPanel` in `internal/app/auth.go`
      - test: `go build ./...` → clean
- [ ] `make ci` → PASS
