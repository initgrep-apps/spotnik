# Feature 31 — Notifications + Error Routing

> **Feature:** Replace the single `statusMsg` string with a BubbleUp-based toast
> notification system. Route all API errors through toast notifications instead of
> inline pane error rendering.

## Context

The current notification system is a single `statusMsg` string field on the App struct
(app.go line 93-94) rendered in error-red color (`theme.Error()`) for both success and
error messages (render.go lines 194-210). There's no visual distinction between severity
levels. The status bar conflates keybinding hints with transient feedback.

After Feature 29 (data-carrying messages), all Msgs carry `Err` fields. This feature
routes those errors through a proper notification system.

**Gap reference:** G3, G5, G10 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

**Depends on:** Feature 29 (needs data-carrying Msgs with Err fields)

**Approved dependency:** BubbleUp (`go.dalton.dog/bubbleup`) — approved by owner on 2026-03-24.
MIT-licensed, depends only on bubbletea + lipgloss (already in deps).

---

## Task 1: BubbleUp dependency + notification wrapper

**Problem:** No toast notification system exists.

**Fix:**

1. `go get go.dalton.dog/bubbleup`

2. Inspect the BubbleUp API after install:
   ```bash
   go doc go.dalton.dog/bubbleup AlertDefinition
   go doc go.dalton.dog/bubbleup AlertModel.Render
   ```
   Confirm: `ForeColor` is `string`, `Render(content string) string` overlays alert on content.
   **Important:** BubbleUp's `View()` method is intentionally empty. Use `Render(content)` not `View()`.

3. Create `internal/ui/components/notifications.go`:
   ```go
   // NewNotifications creates a BubbleUp AlertModel configured with Spotnik
   // theme colors and custom alert types.
   func NewNotifications(t theme.Theme) bubbleup.AlertModel

   // Alert type keys: "success", "error", "warning", "info", "ratelimit"
   ```

4. Register custom alert types matching the theme:
   ```go
   successAlert := bubbleup.AlertDefinition{
       Key:       "success",
       ForeColor: string(theme.Success()),    // explicit conversion from lipgloss.Color
       Prefix:    "✓",
   }
   errorAlert := bubbleup.AlertDefinition{
       Key:       "error",
       ForeColor: string(theme.Error()),
       Prefix:    "✗",
   }
   warningAlert := bubbleup.AlertDefinition{
       Key:       "warning",
       ForeColor: string(theme.Warning()),
       Prefix:    "!",
   }
   infoAlert := bubbleup.AlertDefinition{
       Key:       "info",
       ForeColor: string(theme.KeyHint()),
       Prefix:    "→",
   }
   rateLimitAlert := bubbleup.AlertDefinition{
       Key:       "ratelimit",
       ForeColor: string(theme.Warning()),
       Prefix:    "⧖",
   }
   ```

**Files:**
- Modify: `go.mod` — add `go.dalton.dog/bubbleup`
- Create: `internal/ui/components/notifications.go`
- Create: `internal/ui/components/notifications_test.go`

**Tests:**
- Unit: `NewNotifications` returns a valid AlertModel with all 5 custom types registered
- Unit: verify `ForeColor` conversion compiles and produces expected hex strings

**Commit:** `feat(ui): add BubbleUp notification wrapper`

---

## Task 2: Integrate BubbleUp into root App

**Problem:** No notification model exists in the App.

**Fix:**

1. Add `alerts bubbleup.AlertModel` field to App struct (app.go)

2. In `New()`: initialize with `components.NewNotifications(t)`

3. In `Init()`: batch `a.alerts.Init()` with existing commands

4. In `Update()`: for every message, also pass to `a.alerts.Update(msg)` and batch
   any returned command. This is necessary for BubbleUp's internal timer management
   (auto-dismiss).

5. In `View()` (render.go): as the **final step**, call `a.alerts.Render(existingView)`
   instead of returning `existingView` directly. This overlays any active alert on top
   of the full rendered view.
   **Critical:** Do NOT call `a.alerts.View()` — it returns empty string by design.

**Files:**
- Modify: `internal/app/app.go` — add alerts field, Init/Update wiring
- Modify: `internal/app/render.go` — call Render() as final overlay step

**Tests:**
- Unit: App Init() includes alerts Init command
- Unit: App Update() forwards messages to alerts model
- Unit: App View() calls alerts.Render() on the final output

**Commit:** `feat(ui): integrate BubbleUp into root App model`

---

## Task 3: Replace statusMsg with toast commands

**Problem:** 16 `a.statusMsg = "..."` assignment sites across app.go and routing.go,
plus `statusDismissMsg` type and its handler.

**Current statusMsg sites** (all in `internal/app/`):

| File | Line | Current Message | New Alert Type |
|---|---|---|---|
| app.go | 530 | `"Rate limited — pausing requests for %ds"` | `ratelimit` |
| app.go | 541 | `"Session expired. Run: spotnik auth"` | `error` |
| app.go | 566 | `"Playback control not available on this device"` | `warning` |
| app.go | 568 | `"✗ %s"` (playback error) | `error` |
| app.go | 601 | `"✗ %s"` (forbidden) | `error` |
| app.go | 603 | `"✗ %s"` (add-to-queue error) | `error` |
| app.go | 608 | `"✓ Added to queue: %s"` | `success` |
| app.go | 610 | `"✓ Added to queue"` | `success` |
| app.go | 632 | `"✗ %s"` (like toggle error) | `error` |
| app.go | 647 | `"Switching to %s..."` | `info` |
| app.go | 656 | `"✗ %s"` (device transfer error) | `error` |
| app.go | 666 | `""` (statusDismissMsg clears) | N/A — removed |
| routing.go | 218 | `"✗ %s"` (playlist created error) | `error` |
| routing.go | 229 | `"✗ %s"` (playlist renamed error) | `error` |
| routing.go | 251 | `"✗ %s"` (playlist remove error) | `error` |
| routing.go | 268 | `"✗ %s"` (playlist reorder error) | `error` |

**Fix:**

1. Replace each `a.statusMsg = "message"` with returning a `a.alerts.NewAlertCmd("type", "message")`
   command (batch with any existing commands at that site).

2. Remove the `statusMsg string` field from App struct.

3. Remove the `statusDismissMsg` type and its handler in Update() (app.go line 127-128, 666).
   BubbleUp handles auto-dismiss internally.

4. Update `renderStatusBar()` (render.go lines 194-210):
   - Remove the `if a.statusMsg != ""` branch
   - Status bar now ALWAYS shows keybinding hints
   - Toast notifications appear as overlays via `alerts.Render()`, not in the status bar

**Files:**
- Modify: `internal/app/app.go` — replace all statusMsg sites, remove field + dismiss type
- Modify: `internal/app/routing.go` — replace all statusMsg sites
- Modify: `internal/app/render.go` — simplify renderStatusBar()
- Modify: `internal/app/render_test.go` — update tests for new status bar behavior

**Tests:**
- Unit: verify each error/success case emits the correct alert type
- Unit: `renderStatusBar()` always returns hints (no error override)
- Grep verification: `grep -r 'statusMsg' internal/app/` → ZERO matches

**Commit:** `refactor(ui): replace statusMsg with BubbleUp toasts`

---

## Task 4: Route all API errors through toast

**Problem:** Pane View() methods read Store error fields to show inline error messages.
After Feature 29, all data-carrying Msgs have `Err` fields. Errors should route through
toast, not inline pane rendering.

**Fix:**

1. In `app.go` `Update()` handlers, for every data-carrying Msg with non-nil `Err`:
   - Emit a toast via `a.alerts.NewAlertCmd("error", errorMessage + retryHint)`
   - Example: `PlaybackStateFetchedMsg{Err: err}` → toast "Playback update failed"
   - Example: `LibraryLoadedMsg{Err: err}` → toast "Failed to load playlists. Press Tab to retry"

2. Remove Store error field reads from pane `View()` methods:
   - `internal/ui/panes/search.go` — remove error display
   - `internal/ui/panes/playlists.go` — remove error display
   - `internal/ui/panes/stats.go` — remove error display
   - `internal/ui/panes/devices.go` — remove error display

   Store error fields remain for retry logic only — never read in `View()`.

3. Update docs:
   - **`docs/ARCHITECTURE.md`**: New section "Notification System" — BubbleUp integration,
     alert types, severity mapping, error routing through toasts
   - **`docs/ARCHITECTURE.md`** → "Error Handling Conventions": Update to document that all
     errors route through notification system, not inline pane rendering
   - **`docs/DESIGN.md`** → "Status Bar" section: Remove "Error mode" (replaced by toast overlay);
     status bar becomes hints-only
   - **`docs/DESIGN.md`**: New section "Toast Notifications" — position, severity colors,
     dismiss behavior, error routing
   - **`CLAUDE.md`** → "Architecture Rules": Add "All API errors route through the notification
     system (toast), not inline pane rendering"

**Files:**
- Modify: `internal/app/app.go` — add toast commands for all error Msgs
- Modify: `internal/app/routing.go` — add toast commands for playlist error Msgs
- Modify: `internal/ui/panes/search.go` — remove inline error rendering
- Modify: `internal/ui/panes/playlists.go` — remove inline error rendering
- Modify: `internal/ui/panes/stats.go` — remove inline error rendering
- Modify: `internal/ui/panes/devices.go` — remove inline error rendering
- Modify: `docs/ARCHITECTURE.md` — add Notification System section
- Modify: `docs/DESIGN.md` — update Status Bar, add Toast section
- Modify: `CLAUDE.md` — add error routing rule

**Tests:**
- Unit: each error Msg triggers a toast command
- Unit: no Store error fields are read in any pane View() method
- Grep verification: `grep -r 'Error()' internal/ui/panes/*` should show no Store error reads

**Commit 1:** `feat(ui): route all API errors through toast notifications`
**Commit 2:** `docs: add notification system and error routing documentation`

---

## Verification

```bash
# statusMsg completely removed
grep -r 'statusMsg' internal/app/
# Expected: ZERO matches

# All errors trigger toast
grep -r 'statusDismissMsg' internal/app/
# Expected: ZERO matches

make ci
# Expected: Full pass
```

---

## Notification Mapping Reference

| Current `statusMsg` Usage | New BubbleUp Alert Key | Message |
|---|---|---|
| `"✓ Added to queue: ..."` | `success` | "Added to queue: ..." |
| `"✗ <error>"` | `error` | "<error>" |
| `"Playback control not available..."` | `warning` | "Playback control not available..." |
| `"Rate limited — pausing requests for Ns"` | `ratelimit` | "Rate limited, retrying in Ns" |
| `"Session expired. Run: spotnik auth"` | `error` | "Session expired. Run: spotnik auth" |
| `"Switching to <device>..."` | `info` | "Switching to <device>..." |

---

*Depends on: Feature 29*
*Blocked by: Feature 29*
*Blocks: Nothing*
