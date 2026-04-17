---
title: "Notifications + Error Routing"
feature: 11-api-gateway
status: done
---

## Background
The notification system was a single `statusMsg` string field on the App struct rendered in error-red color for both success and error messages. There was no visual distinction between severity levels. The status bar conflated keybinding hints with transient feedback. After Feature 29 established data-carrying messages with Err fields, this story replaced the primitive status string with a BubbleUp-based toast notification system and routed all API errors through severity-typed toast overlays instead of inline pane error rendering.

Approved dependency: BubbleUp (`go.dalton.dog/bubbleup`) -- approved by owner on 2026-03-24. MIT-licensed, depends only on bubbletea + lipgloss.

Gap reference: G3, G5, G10 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

## Design

### BubbleUp Integration
Five alert types registered: success, error, warning, info, ratelimit. Each with appropriate theme colors and prefix icons.

Important: BubbleUp's `View()` method is intentionally empty. Use `Render(content)` not `View()`.

### statusMsg Replacement
All 16 `a.statusMsg = "..."` sites replaced with `a.alerts.NewAlertCmd()` calls. statusMsg field and statusDismissMsg type completely removed.

### Notification Mapping

| Current Usage | New Alert Key | Message |
|---|---|---|
| "Added to queue: ..." | success | "Added to queue: ..." |
| "<error>" | error | "<error>" |
| "Playback control not available..." | warning | "Playback control not available..." |
| "Rate limited -- pausing requests for Ns" | ratelimit | "Rate limited, retrying in Ns" |
| "Session expired. Run: spotnik auth" | error | "Session expired. Run: spotnik auth" |
| "Switching to <device>..." | info | "Switching to <device>..." |

### Error Routing
Remove inline error rendering from pane View() methods; emit toasts for all data-carrying Msgs with non-nil Err fields. Store error fields remain for retry logic only -- never read in View().

### Verification
```bash
grep -r 'statusMsg' internal/app/
# Expected: ZERO matches
grep -r 'statusDismissMsg' internal/app/
# Expected: ZERO matches
make ci
```

## Acceptance Criteria
- [ ] `statusMsg` field and `statusDismissMsg` type completely removed from codebase
- [ ] All 16 former `statusMsg` sites emit typed toast commands via `a.alerts.NewAlertCmd()`
- [ ] Five alert types registered: `success`, `error`, `warning`, `info`, `ratelimit`
- [ ] Toast overlays render via `alerts.Render(content)` as the final step of `View()`
- [ ] Status bar always shows keybinding hints (no error override)
- [ ] All pane `View()` methods remove inline error rendering
- [ ] `make ci` passes

## Tasks
- [ ] BubbleUp dependency + notification wrapper -- add go.dalton.dog/bubbleup, create themed wrapper in internal/ui/components/notifications.go
      - test: NewNotifications returns valid AlertModel with all 5 custom types registered
- [ ] Integrate BubbleUp into root App -- add alerts field to App, wire Init/Update/View lifecycle
      - test: App Init() includes alerts Init command; Update() forwards messages; View() calls alerts.Render()
- [ ] Replace statusMsg with toast commands -- replace all 16 statusMsg sites, remove field and dismiss type
      - test: each error/success case emits correct alert type; renderStatusBar() always returns hints; grep verification zero matches
- [ ] Route all API errors through toast -- remove inline error rendering from pane View() methods
      - test: each error Msg triggers a toast command; no Store error fields read in any pane View()
