---
title: "FormField — labelled input with intrinsic validation; migrate onboarding Client ID"
feature: 13-tui-design-system
status: done
---

## Background

`FormField` wraps `bubbles/textinput.Model` with an intrinsic validator and an
error slot rendered beneath the input. Migrates the onboarding Client ID input
currently built inline in `internal/app/app.go` + `render.go`.

Validation runs on demand (typically Enter key). On failure, the error message
is cached and rendered in `Error` colour under the input until the next
`SetValue` or `Validate` call clears it.

**Depends on:** S13 (Toast for success feedback after save), S15 (ProgressBar —
not used directly; listed in plan dependency tree because onboarding uses it),
S16 (Spinner for OAuth wait — `FormField` + `Spinner` compose the onboarding
screen). Design record §7.1 row 14, §7.6 FormField stub. Full step-by-step:
Task 17 (S17) in `docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Types

```go
type FormFieldConfig struct {
    Label       string
    Placeholder string
    Validate    func(string) error
    Theme       theme.Theme
}

type FormField struct { /* cfg, input (textinput.Model), errMsg */ }

func NewFormField(cfg FormFieldConfig) *FormField
func (f *FormField) Update(msg tea.Msg) (*FormField, tea.Cmd)
func (f *FormField) Render() string
func (f *FormField) Value() string
func (f *FormField) SetValue(v string)
func (f *FormField) Validate() error
func (f *FormField) ValidationError() string
```

Render output: `<Label>:` (muted) on its own line, input box below (rounded
border, accent foreground, `Padding(0,1)`), optional error line with `✗ <msg>`
in Error colour beneath.

### Onboarding migration

`internal/app/app.go`: replace `a.onboardingInput textinput.Model` with
`a.onboardingField *uikit.FormField` constructed with a Spotify Client ID
validator (32 chars, hex).

`internal/app/render.go:renderOnboardingRegister`: replace the inline
`<label>: <input>` composition with `a.onboardingField.Render()` — the field
now renders its own label + input + error.

### Roles

| Field | Role |
|---|---|
| FormField.Label | Muted |
| FormField.Input.Text | Plain |
| FormField.Input.Cursor | Accent |
| FormField.ValidationError | Error glyph + Plain text |

## Acceptance Criteria

- [ ] `internal/uikit/form_field.go` defines `FormField`, `FormFieldConfig`,
      `NewFormField`, `Update`, `Render`, `Value`, `SetValue`, `Validate`,
      `ValidationError`
- [ ] `form_field_test.go` covers:
      - `TestFormField_NoErrorBeforeValidation`
      - `TestFormField_ReportsErrorAfterValidate`
      - `TestFormField_AcceptsValidValue`
- [ ] `internal/app/app.go` uses `*uikit.FormField` for the onboarding Client ID
      input; old `textinput.Model` field removed
- [ ] `internal/app/render.go:renderOnboardingRegister` renders the input via
      `a.onboardingField.Render()`
- [ ] Existing onboarding tests still PASS
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 17 (S17) in plan.

- [ ] Branch: `feat/13-uikit-form-field`
- [ ] Write failing `form_field_test.go` (Step 17.1)
- [ ] Implement `form_field.go` (Step 17.2)
- [ ] Wire `a.onboardingField` in `app.go` (Step 17.3)
- [ ] Migrate `renderOnboardingRegister` to use the primitive (Step 17.3)
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 17.4)
