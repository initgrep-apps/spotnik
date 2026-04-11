# Optimistic Playback Updates Implementation Plan

**Goal:** Eliminate ~500ms–1s UI lag on playback controls (volume, play/pause, shuffle, repeat) by writing the predicted state to the store immediately when the key is pressed, before the API roundtrip completes.

**Architecture:** Add a single `applyOptimisticUpdate` method on `*App` that deep-copies the current playback state, applies the predicted mutation, and writes it back to the store synchronously inside `Update()`. The existing API cmd fires after this and the API response overwrites with authoritative state. No new types, messages, or store fields needed.

**Tech Stack:** Go 1.22, Bubble Tea v0.27, `internal/state.Store`, `internal/ui/panes.PlaybackAction`

**Spec:** `docs/superpowers/specs/2026-04-11-optimistic-playback-design.md`

---

## File Map

| File | Action | Purpose |
|------|--------|---------|
| `internal/app/optimistic.go` | **Create** | `applyOptimisticUpdate` method — all mutation logic lives here |
| `internal/app/optimistic_test.go` | **Create** | Table-driven tests via `Update(PlaybackRequestMsg{})` |
| `internal/app/handlers.go` | **Modify** (line ~519) | Add one-line call before `buildPlaybackAPICmd` |
| `docs/ARCHITECTURE.md` | **Modify** (lines 209, 245, ~296) | Remove stale `n` key; add Optimistic Update Pattern section |

---

## Task 1: Fix stale `n` key in ARCHITECTURE.md

**Files:**
- Modify: `docs/ARCHITECTURE.md:209`
- Modify: `docs/ARCHITECTURE.md:245`

`n` (next track) was removed in story 118 but still appears in two places in the routing docs.

- [ ] **Step 1: Edit line 209 — key event flow diagram**

In `docs/ARCHITECTURE.md`, find and replace:
```
     ├── Playback keys (Space, n, +, -, s, r, v, ←, →) → always NowPlayingPane
```
With:
```
     ├── Playback keys (Space, +, -, s, r, v, ←, →) → always NowPlayingPane
```

- [ ] **Step 2: Edit line 245 — overlay routing precedence table**

Find and replace:
```
| 8 | Playback keys | `Space`, `n`, `+`, `-`, `s`, `r`, `v`, `←`, `→` → always NowPlayingPane |
```
With:
```
| 8 | Playback keys | `Space`, `+`, `-`, `s`, `r`, `v`, `←`, `→` → always NowPlayingPane |
```

- [ ] **Step 3: Commit**

```bash
git add docs/ARCHITECTURE.md
git commit -m "docs(arch): remove stale n key from playback routing table"
```

---

## Task 3: Write failing tests

**Files:**
- Create: `internal/app/optimistic_test.go`

Tests exercise `applyOptimisticUpdate` indirectly via `a.Update(PlaybackRequestMsg{Action: ...})`.
This is the correct approach because:
- `applyOptimisticUpdate` is called synchronously inside `Update()` before the async cmd is returned
- `a.Store().PlaybackState()` reflects the optimistic value as soon as `Update()` returns
- The returned cmd (nil player → `PlaybackCmdSentMsg{Err: errNilClient}`) does not need to execute

`newTestApp()` is already defined in `internal/app/staleness_test.go` (same `package app_test`) — do not redefine it.

- [ ] **Step 1: Create the test file**

```go
// internal/app/optimistic_test.go
package app_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyOptimisticUpdate(t *testing.T) {
	tests := []struct {
		name    string
		initial domain.PlaybackState
		action  panes.PlaybackAction
		check   func(t *testing.T, got *domain.PlaybackState)
	}{
		{
			name: "volume up increments by 1",
			initial: domain.PlaybackState{
				Device:    &domain.Device{ID: "d1", VolumePercent: 65},
				IsPlaying: true,
			},
			action: panes.ActionVolumeUp,
			check: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 66, got.Device.VolumePercent)
			},
		},
		{
			name: "volume up clamps at 100",
			initial: domain.PlaybackState{
				Device: &domain.Device{ID: "d1", VolumePercent: 100},
			},
			action: panes.ActionVolumeUp,
			check: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 100, got.Device.VolumePercent)
			},
		},
		{
			name: "volume down decrements by 1",
			initial: domain.PlaybackState{
				Device: &domain.Device{ID: "d1", VolumePercent: 65},
			},
			action: panes.ActionVolumeDown,
			check: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 64, got.Device.VolumePercent)
			},
		},
		{
			name: "volume down clamps at 0",
			initial: domain.PlaybackState{
				Device: &domain.Device{ID: "d1", VolumePercent: 0},
			},
			action: panes.ActionVolumeDown,
			check: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 0, got.Device.VolumePercent)
			},
		},
		{
			name:   "pause sets IsPlaying false",
			initial: domain.PlaybackState{IsPlaying: true},
			action: panes.ActionPause,
			check: func(t *testing.T, got *domain.PlaybackState) {
				assert.False(t, got.IsPlaying)
			},
		},
		{
			name:   "play sets IsPlaying true",
			initial: domain.PlaybackState{IsPlaying: false},
			action: panes.ActionPlay,
			check: func(t *testing.T, got *domain.PlaybackState) {
				assert.True(t, got.IsPlaying)
			},
		},
		{
			name:   "shuffle toggle on",
			initial: domain.PlaybackState{ShuffleState: false},
			action: panes.ActionToggleShuffle,
			check: func(t *testing.T, got *domain.PlaybackState) {
				assert.True(t, got.ShuffleState)
			},
		},
		{
			name:   "shuffle toggle off",
			initial: domain.PlaybackState{ShuffleState: true},
			action: panes.ActionToggleShuffle,
			check: func(t *testing.T, got *domain.PlaybackState) {
				assert.False(t, got.ShuffleState)
			},
		},
		{
			name:   "repeat cycles off→context",
			initial: domain.PlaybackState{RepeatState: "off"},
			action: panes.ActionCycleRepeat,
			check: func(t *testing.T, got *domain.PlaybackState) {
				assert.Equal(t, "context", got.RepeatState)
			},
		},
		{
			name:   "volume up with nil device does not panic",
			initial: domain.PlaybackState{IsPlaying: true, Device: nil},
			action: panes.ActionVolumeUp,
			check: func(t *testing.T, got *domain.PlaybackState) {
				assert.Nil(t, got.Device)
				// IsPlaying unchanged
				assert.True(t, got.IsPlaying)
			},
		},
		{
			name:   "next is a no-op",
			initial: domain.PlaybackState{
				Device: &domain.Device{ID: "d1", VolumePercent: 65},
			},
			action: panes.ActionNext,
			check: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 65, got.Device.VolumePercent)
			},
		},
		{
			name:   "previous is a no-op",
			initial: domain.PlaybackState{
				Device: &domain.Device{ID: "d1", VolumePercent: 65},
			},
			action: panes.ActionPrevious,
			check: func(t *testing.T, got *domain.PlaybackState) {
				require.NotNil(t, got.Device)
				assert.Equal(t, 65, got.Device.VolumePercent)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestApp()

			// Deep-copy initial state to avoid sharing pointers between test runs.
			initial := tt.initial
			if tt.initial.Device != nil {
				dev := *tt.initial.Device
				initial.Device = &dev
			}
			a.Store().SetPlaybackState(&initial)

			// Update() calls applyOptimisticUpdate synchronously before returning the cmd.
			// We do not need to execute the cmd — the optimistic write is already done.
			a.Update(panes.PlaybackRequestMsg{Action: tt.action})

			got := a.Store().PlaybackState()
			require.NotNil(t, got, "store must have a playback state after Update")
			tt.check(t, got)
		})
	}
}

// TestApplyOptimisticUpdate_NilPlaybackState_DoesNotPanic verifies that
// applyOptimisticUpdate is a no-op when the store has no playback state.
func TestApplyOptimisticUpdate_NilPlaybackState_DoesNotPanic(t *testing.T) {
	a := newTestApp()
	// Store has nil playback state by default — do not inject any.

	// Must not panic.
	require.NotPanics(t, func() {
		a.Update(panes.PlaybackRequestMsg{Action: panes.ActionVolumeUp})
	})

	assert.Nil(t, a.Store().PlaybackState(), "store should remain nil when no state was set")
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /path/to/spotnik && go test ./internal/app/ -run TestApplyOptimisticUpdate -v
```

Expected: FAIL — `a.Update(panes.PlaybackRequestMsg{...})` does not yet call `applyOptimisticUpdate`, so the store value won't change. Tests that check `got.Device.VolumePercent == 66` will fail because it stays at 65.

---

## Task 4: Implement `applyOptimisticUpdate`

**Files:**
- Create: `internal/app/optimistic.go`

- [ ] **Step 1: Create the implementation file**

```go
// internal/app/optimistic.go
package app

import (
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
)

// applyOptimisticUpdate immediately writes a predicted state change to the store
// before the API call fires. This gives instant UI feedback for playback actions
// whose outcome is fully predictable from local state alone.
//
// Only predictable actions mutate the store: volume, play/pause, shuffle, repeat.
// ActionNext and ActionPrevious are no-ops — the next track is determined by Spotify,
// not by local state.
//
// The authoritative API response overwrites the optimistic value when it arrives via
// PlaybackStateFetchedMsg. On error, the existing fetchPlaybackStateCmd fired from
// the PlaybackCmdSentMsg error path corrects the store automatically.
func (a *App) applyOptimisticUpdate(action panes.PlaybackAction) {
	ps := a.store.PlaybackState()
	if ps == nil {
		return
	}

	// Deep-copy: copy the struct value and any pointer fields to avoid aliasing
	// between the optimistic state and whatever the store already holds.
	updated := *ps
	if ps.Device != nil {
		dev := *ps.Device
		updated.Device = &dev
	}

	switch action {
	case panes.ActionVolumeUp:
		if updated.Device != nil {
			v := updated.Device.VolumePercent + a.volumeStep
			if v > 100 {
				v = 100
			}
			updated.Device.VolumePercent = v
		}
	case panes.ActionVolumeDown:
		if updated.Device != nil {
			v := updated.Device.VolumePercent - a.volumeStep
			if v < 0 {
				v = 0
			}
			updated.Device.VolumePercent = v
		}
	case panes.ActionPause:
		updated.IsPlaying = false
	case panes.ActionPlay:
		updated.IsPlaying = true
	case panes.ActionToggleShuffle:
		updated.ShuffleState = !updated.ShuffleState
	case panes.ActionCycleRepeat:
		updated.RepeatState = nextRepeatMode(updated.RepeatState)
	case panes.ActionNext, panes.ActionPrevious:
		// no-op: next track is determined by Spotify, not local state
	}

	a.store.SetPlaybackState(&updated)
}
```

Note: `nextRepeatMode` is already defined in `internal/app/commands.go` (same package) — no import needed.

- [ ] **Step 2: Run tests to verify they pass**

```bash
go test ./internal/app/ -run TestApplyOptimisticUpdate -v
```

Expected: all 13 cases PASS. If any fail, re-read the failure — likely a deep-copy pointer aliasing issue.

---

## Task 5: Wire into handlers.go

**Files:**
- Modify: `internal/app/handlers.go` — the `PlaybackRequestMsg` case (~line 519)

- [ ] **Step 1: Find the exact case**

The case currently reads:
```go
case panes.PlaybackRequestMsg:
    return a, a.buildPlaybackAPICmd(m.Action)
```

- [ ] **Step 2: Add the optimistic update call**

Replace with:
```go
case panes.PlaybackRequestMsg:
    a.applyOptimisticUpdate(m.Action)
    return a, a.buildPlaybackAPICmd(m.Action)
```

- [ ] **Step 3: Run the full app test suite**

```bash
go test ./internal/app/ -v
```

Expected: all existing tests pass. No behaviour changes — `buildPlaybackAPICmd` is unchanged, error paths are unchanged.

- [ ] **Step 4: Commit**

```bash
git add internal/app/optimistic.go internal/app/optimistic_test.go internal/app/handlers.go
git commit -m "feat(playback): add optimistic store update on key press

Writes predicted state to store immediately when +/-/space/s/r pressed
so the UI reflects the change on the next render frame rather than after
the full API roundtrip (~500ms-1s). Authoritative state overwrites on
PlaybackStateFetchedMsg. ActionNext/Previous remain unchanged (unpredictable).
"
```

---

## Task 6: Add Optimistic Update Pattern to ARCHITECTURE.md

**Files:**
- Modify: `docs/ARCHITECTURE.md` — after line 296 (end of Data-Carrying Messages section)

- [ ] **Step 1: Add the new subsection**

After the paragraph "All message types in `internal/ui/panes/messages.go` carry their data payload and an `Err error` field. `Update()` is the sole writer to the Store." and before the `---` separator at line 298, insert:

```markdown

### Optimistic Updates

For user-triggered actions where the new state is **fully predictable from local state**, `Update()` may write an optimistic value to the store immediately — before the API cmd fires — to give instant UI feedback.

**When to use:** volume up/down, play/pause, shuffle toggle, repeat cycle. These actions have deterministic outcomes given the current store state.

**When NOT to use:** actions whose outcome depends on server data (Next, Previous, any fetch). The next track is determined by Spotify, not local state.

**Pattern:**
```go
// In app.Update(), BEFORE returning the API cmd:
case panes.PlaybackRequestMsg:
    a.applyOptimisticUpdate(m.Action) // sync: store written, UI renders next frame
    return a, a.buildPlaybackAPICmd(m.Action) // async: API call, result overwrites store
```

The optimistic write happens in `Update()` — consistent with the Elm contract. Commands still never write to the store. When the API response arrives via `PlaybackStateFetchedMsg`, `store.SetPlaybackState()` overwrites the optimistic value with the authoritative one. On API error, the `fetchPlaybackStateCmd` fired from the `PlaybackCmdSentMsg` error handler corrects the store automatically.
```

- [ ] **Step 2: Verify the doc renders correctly**

```bash
grep -n "Optimistic" docs/ARCHITECTURE.md
```

Expected: two hits — the new section heading and the `When NOT to use` line.

- [ ] **Step 3: Commit**

```bash
git add docs/ARCHITECTURE.md
git commit -m "docs(arch): document optimistic update pattern"
```

---

## Task 7: Full CI gate

- [ ] **Step 1: Run full CI**

```bash
make ci
```

Expected: lint passes, all tests pass, coverage ≥ 80%.

If lint fails with `exhaustive` or similar on the `switch action` in `optimistic.go`: all `PlaybackAction` values are now explicitly handled — `ActionNext` and `ActionPrevious` have an explicit no-op case. This should satisfy any exhaustiveness checker.

If coverage drops below 80%: run `make test-coverage` to identify the gap, add missing test cases to `optimistic_test.go`.

- [ ] **Step 2: Final commit if any lint fixes were needed**

```bash
git add -A
git commit -m "fix(lint): address golangci-lint warnings in optimistic.go"
```

(Only if lint changes were needed — skip otherwise.)

---

## Task 8: Push and open PR

- [ ] **Step 1: Push branch**

```bash
git push origin feat/optimistic-playback
```

- [ ] **Step 2: Open PR**

```bash
gh pr create \
  --title "feat(playback): optimistic store update on key press" \
  --body "$(cat <<'EOF'
## Summary

- Adds `applyOptimisticUpdate` on `*App` that immediately writes predicted playback state to the store when a control key is pressed
- Wires it into the `PlaybackRequestMsg` handler before `buildPlaybackAPICmd` fires
- Fixes stale `n` key in ARCHITECTURE.md routing table (removed in story 118)
- Documents the Optimistic Update Pattern in ARCHITECTURE.md

## Behaviour

Before: UI bar updates ~500ms–1s after key press (100ms debounce + two API roundtrips).
After: UI bar updates on the next render frame after key press. API response overwrites with authoritative state silently.

Actions covered: volume +/-, play/pause, shuffle toggle, repeat cycle.
Actions NOT covered (unpredictable): Next, Previous.

## Test plan

- [ ] `make ci` passes (lint + tests + coverage ≥ 80%)
- [ ] Manual: press `+` repeatedly — volume bar increments immediately on each press
- [ ] Manual: press `space` — play/pause icon flips immediately
- [ ] Manual: press `s` — shuffle indicator toggles immediately
- [ ] Manual: press `r` — repeat indicator cycles immediately
- [ ] Manual: disconnect network, press `+` — volume bar increments then snaps back with error toast

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
