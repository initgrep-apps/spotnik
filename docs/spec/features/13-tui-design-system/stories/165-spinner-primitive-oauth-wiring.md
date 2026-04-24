---
title: "Spinner — Done/Fail/Cancel contract; wire onboarding OAuth wait"
feature: 13-tui-design-system
status: open
---

## Background

`Spinner` is the TUI peer of `cliout.Spinner`. Unbounded-wait indicator with
**terminal states**: `Done(text)` resolves to `✓` (Success) and emits
`SpinnerDoneMsg` after a 1.2 s hold; `Fail(text)` resolves to `✗` (Error) and
emits `SpinnerFailMsg{Err}` after a 2 s hold; `Cancel()` clears immediately and
emits `SpinnerCancelledMsg`.

Wraps `bubbles/spinner` internally; exposes a primitive surface that lets
onboarding show a resolvable wait for the OAuth callback. On success the
grid view opens + a `Toast(Success, "Signed in", ...)` is emitted; on failure
the error panel appears + a `Toast(Error, <detail>, "Re-enter Client ID")` is
emitted.

**Depends on:** S1. Design record §7.5 (Spinner contract), §7.1 row 18. Full
step-by-step: Task 16 (S16) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Types

```go
type SpinnerDoneMsg struct{ Text string }
type SpinnerFailMsg struct{ Err string }
type SpinnerCancelledMsg struct{}

type Spinner struct { /* wraps bubbles/spinner.Model + resolution state */ }

func NewSpinner(text string, th theme.Theme) *Spinner
func (s *Spinner) Init() tea.Cmd
func (s *Spinner) Update(msg tea.Msg) (*Spinner, tea.Cmd)
func (s *Spinner) View() string
func (s *Spinner) Done(text string)   (*Spinner, tea.Cmd) // 1.2s hold, then SpinnerDoneMsg
func (s *Spinner) Fail(text string)   (*Spinner, tea.Cmd) // 2s hold, then SpinnerFailMsg
func (s *Spinner) Cancel()            (*Spinner, tea.Cmd) // immediate SpinnerCancelledMsg
```

While running: animated frame (`bubbles/spinner.Dot`) + muted text.
After Done: `✓` in Success colour + muted text.
After Fail: `✗` in Error colour + muted text.
Ascii mode: `|/-\` rotating frames.

### OAuth integration

In `internal/app/auth.go`:

```go
case auth.OAuthSuccessMsg:
    s, cmd := a.onboardingSpinner.Done("Authorized")
    a.onboardingSpinner = s
    return a, cmd

case uikit.SpinnerDoneMsg:
    a.currentView = viewGrid
    return a, a.toasts.Cmd(uikit.Toast{
        Intent: uikit.ToastSuccess,
        Title:  "Signed in",
        Body:   "Welcome back to Spotnik.",
    })

case auth.OAuthFailureMsg:
    s, cmd := a.onboardingSpinner.Fail("Authorization failed")
    a.onboardingSpinner = s
    a.onboardingError = m.Err.Error()
    return a, cmd

case uikit.SpinnerFailMsg:
    a.onboardingStep = stepError
    return a, nil
```

### Roles

| Field | Role |
|---|---|
| Spinner.Frame (animated) | Accent |
| Spinner.Frame (Done hold) | Success |
| Spinner.Frame (Fail hold) | Error |
| Spinner.Text | Muted |

## Acceptance Criteria

- [ ] `internal/uikit/spinner.go` defines `Spinner`, `NewSpinner`, `Init`,
      `Update`, `View`, `Done`, `Fail`, `Cancel`, and the three message types
- [ ] `spinner_test.go` covers:
      - `TestSpinner_Done_EmitsMsgAfterTTL`
      - `TestSpinner_Fail_EmitsMsgWithErr`
      - `TestSpinner_Cancel_ClearsImmediately`
- [ ] `internal/app/auth.go` OAuth success/failure transitions route through
      `uikit.Spinner` → `SpinnerDoneMsg` / `SpinnerFailMsg`
- [ ] Onboarding no longer references the old `onboardingSpinner *spinner.Model`
      directly — it uses `*uikit.Spinner`
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 16 (S16) in plan.

- [ ] Branch: `feat/13-uikit-spinner`
- [ ] Write failing `spinner_test.go` (Step 16.1)
- [ ] Implement `spinner.go` (Step 16.2)
- [ ] Rewire `internal/app/auth.go` OAuth handlers for Done / Fail / Cancel
      (Step 16.3)
- [ ] Update onboarding spinner field type in `app.go`
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 16.4)
