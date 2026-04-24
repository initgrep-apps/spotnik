---
title: "Onboarding end-to-end rewrite — compose Panel + FormField + URLBox + Spinner + Toast"
feature: 13-tui-design-system
status: open
---

## Background

With Toast, Spinner, URLBox, Panel, FormField, StatusGlyph, and KeyBar all
available, this story rewrites the three onboarding screens —
`renderOnboardingRegister`, `renderOnboardingOAuth`, `renderOnboardingError` —
to compose primitives instead of hand-building lipgloss strings.

Also:
- **Copy-URI feedback becomes a Toast.** The old 2-second `onboardingCopied`
  inline flash, the `clearCopiedMsg` type, and the handler branch are removed;
  pressing `c` emits `a.toasts.Cmd(Toast{Intent: ToastSuccess, Title: "Copied"})`.
- **Panel title absorbs the step header.** No more separate "Step 1 of 2"
  rendered above the panel body; the title lives in the panel's top border.

**Depends on:** S3–S17 (all composing primitives). Design record §7 Panel /
FormField / URLBox / Toast / Spinner contracts; §10 rule 10 (API errors →
Toast). Full step-by-step: Task 18 (S18) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### `renderOnboardingRegister` (Step 1 of 2)

Composes: `Panel` (title in border) wrapping `URLBox` (redirect URI) +
`StatusGlyph` (Premium-required warning, save-path confirmation) + `FormField`
(Client ID input) + `KeyBar` (`c copy URI`, `Enter confirm`, `q quit`).

### `renderOnboardingOAuth` (Step 2 of 2)

Composes: `Panel` + instructional text + `URLBox` (auth URL fallback for headless
users) + `Spinner.View()` + `KeyBar` (`c copy URL`, `q quit`).

### `renderOnboardingError`

Composes: `Panel{Intent: PanelIntentError}` + `StatusGlyph{RoleError, ...}` +
error detail + "Common causes" bullet list + `KeyBar` (`r re-enter`, `l try
again`, `q quit`).

### OAuth flow

- OAuth success → `Spinner.Done("Authorized")` → 1.2s → `SpinnerDoneMsg` →
  grid view + `Toast(Success, "Signed in")`.
- OAuth fail → `Spinner.Fail("Authorization failed")` → 2s → `SpinnerFailMsg` →
  error panel + `Toast(Error, <detail>, "Re-enter Client ID")`.
- User presses `q` during wait → `Spinner.Cancel()` → quit.

### Deletions

- `a.onboardingCopied bool`, `clearCopiedMsg{}`, and its handler branch
- Any inline `✓ Copied!` flash rendering in `render.go`

## Acceptance Criteria

- [ ] `internal/app/render.go:renderOnboardingRegister` renders via
      `uikit.Panel` composing `URLBox` + `StatusGlyph` + `FormField` + `KeyBar`
- [ ] `internal/app/render.go:renderOnboardingOAuth` renders via `uikit.Panel`
      composing instructional text + `URLBox` + `Spinner.View()` + `KeyBar`
- [ ] `internal/app/render.go:renderOnboardingError` uses
      `uikit.Panel{Intent: PanelIntentError}` + `StatusGlyph{RoleError}`
- [ ] `onboardingCopied` state and `clearCopiedMsg` type removed
- [ ] Pressing `c` in any onboarding step emits `a.toasts.Cmd(Toast{Success,
      "Copied"})`
- [ ] Panel title absorbs step header — no separate "Step 1 of 2" line above
      the panel body
- [ ] Visual smoke test passes (see plan Step 18.5):
      - Step 1 shows title in border, URI box, warning glyph, input field, key bar
      - Pressing `c` emits Success toast "Copied"
      - Step 2 shows spinner with animated frame
      - OAuth success: spinner → `✓ Authorized` for 1.2s, then grid + success toast
      - OAuth fail: spinner → `✗ Authorization failed` for 2s, then error panel
- [ ] Existing onboarding tests still PASS (update assertions where needed)
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 18 (S18) in plan.

- [ ] Branch: `feat/13-onboarding-rewrite`
- [ ] Rewrite `renderOnboardingRegister` to compose primitives (Step 18.1)
- [ ] Rewrite `renderOnboardingOAuth` (Step 18.2)
- [ ] Rewrite `renderOnboardingError` (Step 18.3)
- [ ] Replace copy-URI inline flash with Toast; delete `onboardingCopied` /
      `clearCopiedMsg` (Step 18.4)
- [ ] Update `render_test.go` assertions
- [ ] Visual smoke test via `make run` (Step 18.5)
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR
