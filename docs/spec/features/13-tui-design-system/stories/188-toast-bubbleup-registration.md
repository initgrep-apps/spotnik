---
title: "Toast bubbleup alert registration moves into uikit (mode-aware prefixes)"
feature: 13-tui-design-system
status: done
---

## Background

`internal/ui/components/notifications.go:30,35,40,45,50` registers five
`bubbleup.AlertDefinition` entries at construction time, each with a hardcoded glyph
prefix (`✓`, `✗`, `◬`, `→`, `⧖`). The definitions are passed to `bubbleup.New(...)`
once during app startup and never re-resolved. So even after `uikit.Use(cfg.UI.Glyphs)`
runs, ASCII mode still renders unicode toast prefixes — `bubbleup` already cached the
literal strings.

Audit §3.3 recommends moving the alert-definition construction into a new
`uikit.RegisterBubbleupAlerts(theme)` helper that resolves prefixes through
`GlyphFor(role, ActiveMode())` at registration time. Because `uikit.Use` runs **before**
notifications are constructed (per `cmd/root.go:runApp` ordering), the helper sees the
right `ActiveMode()` and emits the correct prefix.

After the move, `notifications.go` becomes a thin wrapper that calls
`uikit.RegisterBubbleupAlerts(theme)` and feeds the result to `bubbleup.New(defs...)`.

**Depends on:** story 183 (catalogue audit). Uses existing intent roles
(`GlyphSuccess`, `GlyphError`, `GlyphWarning`, `GlyphInfo`, `GlyphRateLimit`) — all
already in `glyph.go`.

**Plan tasks:** 4.3 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

**Files modified:** `internal/uikit/toast.go`, `internal/uikit/toast_test.go`,
`internal/ui/components/notifications.go`,
`internal/ui/components/notifications_test.go`.

## Design

### `uikit.RegisterBubbleupAlerts(theme)` — new helper

Add to `internal/uikit/toast.go`:

```go
// RegisterBubbleupAlerts builds the bubbleup alert definitions for the five
// toast intents. Glyph prefixes are resolved via GlyphFor at call time so the
// result honours ActiveMode().
func RegisterBubbleupAlerts(theme theme.Theme) []bubbleup.AlertDefinition {
    m := ActiveMode()
    return []bubbleup.AlertDefinition{
        {
            Type:   "success",
            Prefix: GlyphFor(GlyphSuccess, m),
            Style:  lipgloss.NewStyle().Foreground(theme.Success()),
        },
        {
            Type:   "error",
            Prefix: GlyphFor(GlyphError, m),
            Style:  lipgloss.NewStyle().Foreground(theme.Error()),
        },
        {
            Type:   "warning",
            Prefix: GlyphFor(GlyphWarning, m),
            Style:  lipgloss.NewStyle().Foreground(theme.Warning()),
        },
        {
            Type:   "info",
            Prefix: GlyphFor(GlyphInfo, m),
            Style:  lipgloss.NewStyle().Foreground(theme.Info()),
        },
        {
            Type:   "ratelimit",
            Prefix: GlyphFor(GlyphRateLimit, m),
            Style:  lipgloss.NewStyle().Foreground(theme.Warning()),
        },
    }
}
```

(Implementer: confirm `bubbleup.AlertDefinition` field names match the actual library
shape — read `notifications.go` first to copy exactly.)

### `notifications.go` becomes a thin wrapper

```go
package components

import (
    "github.com/koki-develop/bubbleup"

    "github.com/initgrep-apps/spotnik/internal/uikit"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

func NewNotifications(t theme.Theme) *bubbleup.Model {
    defs := uikit.RegisterBubbleupAlerts(t)
    return bubbleup.New(defs...)
}
```

### Test

`internal/uikit/toast_test.go`:

```go
func TestRegisterBubbleupAlerts_AsciiPrefixes(t *testing.T) {
    SetModeForTest(GlyphASCII)
    defer SetModeForTest(GlyphUnicode)

    defs := RegisterBubbleupAlerts(theme.Load("black"))
    wantPrefixes := map[string]string{
        "success":   "+",
        "error":     "x",
        "warning":   "!",
        "info":      ">",
        "ratelimit": "~",
    }
    for _, d := range defs {
        want, ok := wantPrefixes[d.Type]
        if !ok {
            continue
        }
        if d.Prefix != want {
            t.Errorf("alert %q ascii prefix = %q, want %q", d.Type, d.Prefix, want)
        }
    }
}
```

A unicode-mode sibling test confirms `+ → ✓`, `x → ✗`, `! → ◬`, `> → →`, `~ → ⧖`.

## Acceptance Criteria

- [ ] `internal/uikit/toast.go` exports `RegisterBubbleupAlerts(theme theme.Theme)
      []bubbleup.AlertDefinition`
- [ ] The helper resolves all five prefixes via `GlyphFor(role, ActiveMode())` at
      call time — no captured `m` outside the function body, no init-time caching
- [ ] `internal/ui/components/notifications.go` `NewNotifications` calls
      `uikit.RegisterBubbleupAlerts(t)` and passes the result to `bubbleup.New(...)`;
      no hardcoded prefix literals (`✓ ✗ ◬ → ⧖`) remain in this file
- [ ] New test `TestRegisterBubbleupAlerts_AsciiPrefixes` confirms `+ x ! > ~` in
      ASCII mode for the five intents
- [ ] Sibling test confirms `✓ ✗ ◬ → ⧖` in unicode mode
- [ ] All existing notifications tests pass
- [ ] Toast notifications fired in-app under `ui.glyphs = "ascii"` produce ASCII
      prefixes
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 4.3 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

- [ ] Branch: `fix/13-toast-bubbleup-registration`
- [ ] Read `internal/ui/components/notifications.go` to record the exact
      `bubbleup.AlertDefinition` field shape
- [ ] Write failing `TestRegisterBubbleupAlerts_AsciiPrefixes` (and unicode sibling)
      → FAIL
- [ ] Add `RegisterBubbleupAlerts(theme)` to `internal/uikit/toast.go` resolving all
      five prefixes via `GlyphFor`
- [ ] Replace `internal/ui/components/notifications.go` body with the thin wrapper
      calling the helper
- [ ] Run tests → PASS
- [ ] Commit: `fix(toast): resolve bubbleup alert prefixes via GlyphFor at registration`
- [ ] `make ci` → PASS
- [ ] Push branch + open PR
