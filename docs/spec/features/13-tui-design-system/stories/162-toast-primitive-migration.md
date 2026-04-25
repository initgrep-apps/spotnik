---
title: "Toast — typed API wrapping bubbleup; migrate all a.alerts.NewAlertCmd sites"
feature: 13-tui-design-system
status: done
---

## Background

`Toast` is the canonical notification primitive — a typed struct with `Intent`,
`Title`, `Body`, and `TTL` — wrapping the existing `bubbleup` alert model. This
replaces every raw `a.alerts.NewAlertCmd(kind, msg)` call site in
`internal/app/handlers.go` with `a.toasts.Cmd(uikit.Toast{...})`.

Default TTLs per intent: Success/Info 4s, Warning 5s, Error 6s, RateLimit uses
Retry-After seconds (default 30s if unspecified).

Content rules (from design §7.4): Title ≤ 48 runes hard-truncated; Body ≤ 160
runes hard-truncated; sentence case; no emoji; past-participle verb for
completions ("Copied", "Saved"), noun + state for async events ("Device
disconnected", "Rate-limited").

**Depends on:** S1. Design record §7.4 (Toast contract), §10 rule 10 (all API
errors route through Toast). Full step-by-step: Task 13 (S13) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Types

```go
type ToastIntent int
const (
    ToastSuccess ToastIntent = iota
    ToastError
    ToastWarning
    ToastInfo
    ToastRateLimit
)

type Toast struct {
    Intent ToastIntent
    Title  string
    Body   string
    TTL    time.Duration
}

func DefaultTTL(i ToastIntent) time.Duration
func ToastGlyph(i ToastIntent, m GlyphMode) string
func (t Toast) Normalize() Toast // truncates Title/Body, defaults TTL
```

### ToastManager

```go
type ToastManager struct { model *bubbleup.AlertModel }

func NewToastManager(model *bubbleup.AlertModel) *ToastManager
func (tm *ToastManager) Cmd(t Toast) tea.Cmd
```

`Cmd` normalises the toast, maps intent → bubbleup alert key
(`success` / `error` / `warning` / `info` / `ratelimit`), and returns a
`tea.Cmd` via `tm.model.NewAlertCmd`.

### Call-site migration

In `internal/app/app.go`:

```go
a.toasts = uikit.NewToastManager(a.alerts)
```

In `internal/app/handlers.go`, every:

```go
return a, a.alerts.NewAlertCmd("error", err.Error())
```

becomes:

```go
return a, a.toasts.Cmd(uikit.Toast{
    Intent: uikit.ToastError,
    Title:  "Spotify unreachable",
    Body:   err.Error(),
})
```

Mapping: `"success"` → `ToastSuccess`, `"error"` → `ToastError`, etc.

### Glyphs by intent

Success `✓/+` · Error `✗/x` · Warning `◬/!` · Info `→/>` · RateLimit `⧖/~`.

### Roles

| Field | Role |
|---|---|
| Toast.Glyph | intent role |
| Toast.Title | Strong |
| Toast.Body | Plain |
| Toast.Border | intent role |

## Acceptance Criteria

- [ ] `internal/uikit/toast.go` defines `ToastIntent`, `Toast`, `DefaultTTL`,
      `ToastGlyph`, `Toast.Normalize`, `ToastManager`, `NewToastManager`, `Cmd`
- [ ] `toast_test.go` covers:
      - `TestToast_DefaultTTL_ByIntent` — 4s / 4s / 5s / 6s for Success /
        Info / Warning / Error
      - `TestToast_TruncatesTitle48Runes`
      - `TestToast_TruncatesBody160Runes`
      - `TestToast_GlyphByIntent` — `✓` / `✗` / `◬` / `→` / `⧖` in unicode
- [ ] `internal/app/app.go` constructs `a.toasts = uikit.NewToastManager(a.alerts)`
- [ ] Every `a.alerts.NewAlertCmd` call site in `internal/app/handlers.go` is
      migrated — `grep -n 'a.alerts.NewAlertCmd' internal/` returns no matches
      (except within `notifications.go` which constructs the model)
- [ ] Existing notification tests still PASS
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 13 (S13) in plan.

- [ ] Branch: `feat/13-uikit-toast`
- [ ] Write failing `toast_test.go` (Step 13.1)
- [ ] Implement `toast.go` with types, `Normalize`, `DefaultTTL`, `ToastGlyph`,
      `ToastManager`, `Cmd` (Step 13.2)
- [ ] Wire `a.toasts = uikit.NewToastManager(a.alerts)` in `app.go` (Step 13.3)
- [ ] Migrate every `a.alerts.NewAlertCmd` call in `handlers.go` using the
      intent mapping table; match titles to §7.4 rules (Step 13.3)
- [ ] `make ci` → PASS (Step 13.4)
- [ ] Commit + push + open PR
