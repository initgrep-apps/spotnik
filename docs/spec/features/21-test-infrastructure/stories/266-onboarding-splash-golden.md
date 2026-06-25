---
title: "Onboarding + Splash golden tests"
feature: 21-test-infrastructure
status: open
---

## Background

The onboarding, splash, and "too small" screens are the first things a user sees. They are
rendered by five functions in `internal/app/render.go` with zero golden test coverage. A
change to `uikit.Panel`, `FormField`, `URLBox`, or `Spinner` primitives can break these
screens silently. Current unit tests (`render_test.go`) verify string content but never
snapshot the full visual output at known dimensions.

### Screens requiring coverage

| Screen | Render function | When shown |
|--------|----------------|------------|
| Splash | `renderSplashView()` | App startup (3s) |
| Too small | `renderTooSmall()` | Terminal < 80×24 |
| Registration | `renderOnboardingRegister()` | Step 1: enter Client ID |
| OAuth wait | `renderOnboardingOAuth()` | Step 2: browser auth + spinner |
| OAuth error | `renderOnboardingError()` | Step 2 error: retry options |
| Permissions overlay | `OnboardingPermissionsOverlay` | 'v' during Step 2 |

## Design

### Golden tests: `internal/app/splash_golden_test.go`

- `TestSplashView_Default` — 120×40, version string, branding, centered
- `TestSplashView_Narrow` — 40×10, scaled down splash

### Golden tests: `internal/app/onboarding_golden_test.go`

Registration (Step 1):
- `TestOnboardingRegister_View_EmptyField` — client ID field empty, redirect URI shown in URLBox, FormField focused
- `TestOnboardingRegister_View_WithInput` — "abc123" typed in field, validation not yet triggered
- `TestOnboardingRegister_View_ValidationError` — invalid input submitted, error glyph + message below FormField

OAuth (Step 2):
- `TestOnboardingOAuth_View_SpinnerRunning` — auth URL displayed in URLBox, spinner animating, 'c' copy hint
- `TestOnboardingOAuth_View_PermissionsOverlay` — 'v' pressed, permissions overlay visible on top

Error (Step 2):
- `TestOnboardingError_View_Default` — error message displayed, common causes listed, 'r'/'l'/'q' key hints
- `TestOnboardingError_View_WithPermissionsOverlay` — 'v' pressed during error, overlay visible

### Golden tests: `internal/app/toosmall_golden_test.go`

- `TestTooSmallView_Default` — 40×10 terminal, warning message centered
- `TestTooSmallView_BareMinimum` — terminal at minimum allowed dimensions

### Permissions overlay: `internal/ui/panes/onboarding_golden_test.go`

- `TestOnboardingPermissionsOverlay_View_Normal` — 80×24, all permission sections visible
- `TestOnboardingPermissionsOverlay_View_Narrow` — 40×24

## Files

### Create

- `internal/app/splash_golden_test.go`
- `internal/app/onboarding_golden_test.go`
- `internal/app/toosmall_golden_test.go`
- `internal/ui/panes/onboarding_golden_test.go` — permissions overlay snapshots
- `internal/app/testdata/TestSplashView_*.golden` (2 files)
- `internal/app/testdata/TestOnboarding*.golden` (7 files)
- `internal/app/testdata/TestTooSmallView_*.golden` (2 files)
- `internal/ui/panes/testdata/TestOnboardingPermissionsOverlay_View_*.golden` (2 files)

## Acceptance Criteria

- [ ] Splash: 2 golden snapshots (default, narrow)
- [ ] Registration: 3 golden snapshots (empty, with input, validation error)
- [ ] OAuth: 2 golden snapshots (spinner running, permissions overlay)
- [ ] Error: 2 golden snapshots (default, with permissions overlay)
- [ ] Too small: 2 golden snapshots (default, bare minimum)
- [ ] Permissions overlay: 2 golden snapshots (normal, narrow)
- [ ] All golden files contain recognizable UI elements (borders, FormField, URLBox, Spinner, key hints)
- [ ] `make ci` passes

## Tasks

- [ ] Create splash golden tests (2 snapshots)
      - test: `TestSplashView_Default`, `TestSplashView_Narrow`
- [ ] Create registration golden tests (3 snapshots) — requires building `app.App` with `currentView == viewOnboarding` and `onboardingStep == stepRegister`
      - test: `TestOnboardingRegister_View_EmptyField`, `TestOnboardingRegister_View_WithInput`, `TestOnboardingRegister_View_ValidationError`
- [ ] Create OAuth golden tests (2 snapshots) — step 2 with spinner and permissions overlay
      - test: `TestOnboardingOAuth_View_SpinnerRunning`, `TestOnboardingOAuth_View_PermissionsOverlay`
- [ ] Create error golden tests (2 snapshots) — step 2 error screen
      - test: `TestOnboardingError_View_Default`, `TestOnboardingError_View_WithPermissionsOverlay`
- [ ] Create too-small golden tests (2 snapshots)
      - test: `TestTooSmallView_Default`, `TestTooSmallView_BareMinimum`
- [ ] Create permissions overlay golden tests (2 snapshots)
      - test: `TestOnboardingPermissionsOverlay_View_Normal`, `TestOnboardingPermissionsOverlay_View_Narrow`
- [ ] Generate golden files and verify all tests pass
