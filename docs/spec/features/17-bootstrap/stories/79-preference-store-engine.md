---
title: "PreferenceStore Engine with Debounced Flush"
feature: 17-bootstrap
status: open
---

## Background

Currently `config.PersistTheme()` is a one-off function that reads the entire TOML,
updates one field, and writes it back. Each new persisted preference would require
duplicating this pattern, creating redundant file I/O and potential race conditions
between concurrent writes.

This story introduces a `PreferenceStore` — a coalescing preference writer that
batches in-memory changes and flushes them to disk in a single debounced write.
It plugs into the Bubble Tea event loop via the standard Cmd/Msg pattern. It replaces
`PersistTheme()` entirely.

## Design

### Package Location

```
internal/prefs/
├── prefs.go       ← PreferenceStore type, Set, FlushCmd, Load
└── prefs_test.go
```

### PreferenceStore Type

```go
package prefs

// PreferenceStore manages in-memory preference state and coalesces writes to disk.
// It is the single point of truth for runtime preference changes. Thread-safe.
type PreferenceStore struct {
    mu      sync.Mutex
    path    string         // config file path
    pending map[string]any // dirty preferences not yet flushed
}

// New creates a PreferenceStore targeting the given config file path.
func New(path string) *PreferenceStore {
    return &PreferenceStore{
        path:    path,
        pending: make(map[string]any),
    }
}

// Set marks a preference as dirty. The value is held in memory until FlushCmd
// writes it to disk. Thread-safe — can be called from Update() safely.
func (s *PreferenceStore) Set(key string, value any) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.pending[key] = value
}

// HasPending returns true if there are unsaved preference changes.
func (s *PreferenceStore) HasPending() bool {
    s.mu.Lock()
    defer s.mu.Unlock()
    return len(s.pending) > 0
}
```

### Flush as a Bubble Tea Command

```go
// FlushedMsg is emitted after a flush attempt completes.
// Err is nil on success.
type FlushedMsg struct {
    Err error
}

// FlushCmd returns a tea.Cmd that writes all pending preferences to disk.
// It reads the existing config file, applies pending changes to the
// [preferences] section, and writes it back atomically. Pending map is
// cleared after a successful write.
func (s *PreferenceStore) FlushCmd() tea.Cmd {
    return func() tea.Msg {
        s.mu.Lock()
        if len(s.pending) == 0 {
            s.mu.Unlock()
            return FlushedMsg{}
        }
        // Snapshot pending and clear.
        snapshot := make(map[string]any, len(s.pending))
        for k, v := range s.pending {
            snapshot[k] = v
        }
        s.pending = make(map[string]any)
        s.mu.Unlock()

        err := s.writeToDisk(snapshot)
        if err != nil {
            // Re-queue failed changes so they retry on next flush.
            s.mu.Lock()
            for k, v := range snapshot {
                if _, exists := s.pending[k]; !exists {
                    s.pending[k] = v
                }
            }
            s.mu.Unlock()
        }
        return FlushedMsg{Err: err}
    }
}
```

### Disk Write Implementation

```go
// writeToDisk reads the existing config TOML, applies the snapshot of
// preference changes, and writes the result back. Creates the file and
// parent directory if needed.
func (s *PreferenceStore) writeToDisk(snapshot map[string]any) error {
    // Use the same raw TOML struct pattern as the existing config package
    // to preserve the [spotify] section and any unknown fields.
    raw := struct {
        Spotify struct {
            ClientID string `toml:"client_id,omitempty"`
        } `toml:"spotify"`
        Preferences config.PreferencesConfig `toml:"preferences"`
    }{
        Preferences: config.PreferencesConfig{
            Theme:      "black",
            VolumeStep: 5,
        },
    }

    // Read existing file (ignore missing).
    if _, err := toml.DecodeFile(s.path, &raw); err != nil && !errors.Is(err, os.ErrNotExist) {
        return fmt.Errorf("reading config for preference update: %w", err)
    }

    // Apply snapshot fields to the preferences section.
    for key, val := range snapshot {
        switch key {
        case "theme":
            raw.Preferences.Theme = val.(string)
        case "preset":
            raw.Preferences.Preset = val.(int)
        case "visualizer":
            raw.Preferences.Visualizer = val.(int)
        }
    }

    // Write back.
    if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
        return fmt.Errorf("creating config directory: %w", err)
    }

    f, err := os.OpenFile(s.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
    if err != nil {
        return fmt.Errorf("opening config for write: %w", err)
    }

    enc := toml.NewEncoder(f)
    encErr := enc.Encode(raw)
    if closeErr := f.Close(); closeErr != nil && encErr == nil {
        return fmt.Errorf("closing config: %w", closeErr)
    }
    if encErr != nil {
        return fmt.Errorf("writing config: %w", encErr)
    }
    return nil
}
```

### Debounced Flush via Generation Counter

The generation counter ensures only the latest change triggers a disk write.
When multiple preference changes happen within 500ms, stale timers are ignored
and only the final timer flushes.

In `app.go`:

```go
// New fields on App struct:
prefs         *prefs.PreferenceStore
prefsDirtyGen int  // incremented on every prefs.Set call

// Message type for the debounce timer:
type prefsFlushTickMsg struct{ generation int }
```

Helper method on App:

```go
// schedulePrefsFlush marks preferences dirty and starts a 500ms debounce timer.
// Returns the tea.Cmd for the timer. Call this after every prefs.Set().
func (a *App) schedulePrefsFlush() tea.Cmd {
    a.prefsDirtyGen++
    gen := a.prefsDirtyGen
    return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
        return prefsFlushTickMsg{generation: gen}
    })
}
```

In Update():

```go
case prefsFlushTickMsg:
    // Only flush if no newer preference change happened (debounce).
    if msg.generation == a.prefsDirtyGen {
        return a, a.prefs.FlushCmd()
    }
    return a, nil

case prefs.FlushedMsg:
    if msg.Err != nil {
        // Non-critical — log to stderr, no user toast.
        fmt.Fprintf(os.Stderr, "spotnik: prefs flush failed: %v\n", msg.Err)
    }
    return a, nil
```

### Replacing PersistTheme

The existing `config.PersistTheme()` and `config.PersistThemeTo()` functions are
deleted. The `persistThemeChoice` method on App is replaced:

Before (current):
```go
case panes.ThemeSwitchMsg:
    // ... theme loading and propagation ...
    return a, tea.Batch(
        a.alerts.NewAlertCmd("success", "Theme: "+newTheme.Name()),
        func() tea.Msg {
            a.persistThemeChoice(m.ThemeID)
            return nil
        },
    )
```

After:
```go
case panes.ThemeSwitchMsg:
    // ... theme loading and propagation ...
    a.prefs.Set("theme", m.ThemeID)
    return a, tea.Batch(
        a.alerts.NewAlertCmd("success", "Theme: "+newTheme.Name()),
        a.schedulePrefsFlush(),
    )
```

### Initialization

In `app.New()`:

```go
a.prefs = prefs.New(config.DefaultConfigPath())
```

### Files Changed

| Action | File | Purpose |
|---|---|---|
| Create | `internal/prefs/prefs.go` | PreferenceStore type, New, Set, HasPending, FlushCmd, writeToDisk, FlushedMsg |
| Create | `internal/prefs/prefs_test.go` | Full test coverage for PreferenceStore |
| Modify | `internal/app/app.go` | Add `prefs` and `prefsDirtyGen` fields; add `schedulePrefsFlush()` helper; add `prefsFlushTickMsg` type; handle `prefsFlushTickMsg` and `FlushedMsg` in Update |
| Modify | `internal/app/app.go` | Replace `persistThemeChoice` with `prefs.Set("theme", id)` + `schedulePrefsFlush()` in ThemeSwitchMsg handler |
| Modify | `internal/config/config.go` | Delete `PersistTheme()`, `PersistThemeTo()`, `persistThemeToPath()` |
| Modify | `internal/config/config_test.go` | Delete PersistTheme tests (replaced by prefs package tests) |

## Acceptance Criteria

- [ ] `prefs.New(path)` creates a PreferenceStore targeting the given path
- [ ] `prefs.Set(key, value)` stores the value in memory (does not touch disk)
- [ ] `prefs.FlushCmd()` writes all pending changes to disk in one TOML write
- [ ] `prefs.FlushCmd()` preserves existing `[spotify]` section and unknown fields
- [ ] `prefs.FlushCmd()` clears pending map on success
- [ ] `prefs.FlushCmd()` re-queues changes on write failure (retry on next flush)
- [ ] `prefs.FlushCmd()` is a no-op when nothing is pending
- [ ] Generation counter debounce: 4 rapid Set() calls produce 1 disk write
- [ ] Theme switch now goes through PreferenceStore (PersistTheme deleted)
- [ ] Failed flush logs to stderr, no user toast
- [ ] `make ci` passes

## Tasks

- [ ] Create `internal/prefs/prefs.go` with `PreferenceStore` struct, `New()` constructor, `Set()`, `HasPending()`, and `FlushedMsg` type
      - test: `TestNew_CreatesStore`, `TestSet_StoresInMemory`, `TestSet_MultiplKeys`, `TestHasPending_TrueAfterSet`
- [ ] Add `FlushCmd()` method and `writeToDisk()` implementation. FlushCmd snapshots pending, clears it, writes to disk. Re-queues on failure.
      - test: `TestFlushCmd_WritesToDisk`, `TestFlushCmd_PreservesSpotifySection`, `TestFlushCmd_ClearsPendingOnSuccess`, `TestFlushCmd_RequeuesOnFailure`, `TestFlushCmd_NoopWhenEmpty`
- [ ] Add `FlushCmd()` coalescing test: Set theme + preset + visualizer, single FlushCmd writes all three in one file write
      - test: `TestFlushCmd_CoalescesMultipleChanges`
- [ ] Add `prefsFlushTickMsg` type and `schedulePrefsFlush()` helper to `internal/app/app.go`. Add `prefs` and `prefsDirtyGen` fields to App struct. Initialize `prefs` in `app.New()`.
      - test: `TestApp_SchedulePrefsFlush_IncrementsGeneration`
- [ ] Handle `prefsFlushTickMsg` in `Update()`: only flush if generation matches. Handle `prefs.FlushedMsg`: log error to stderr on failure.
      - test: `TestApp_PrefsFlushTick_MatchingGen_Flushes`, `TestApp_PrefsFlushTick_StaleGen_Ignored`
- [ ] Replace `persistThemeChoice()` in ThemeSwitchMsg handler with `a.prefs.Set("theme", m.ThemeID)` + `a.schedulePrefsFlush()`. Delete `persistThemeChoice` method.
      - test: `TestApp_ThemeSwitch_UsesPreferenceStore` (verify Set called, FlushCmd scheduled)
- [ ] Delete `PersistTheme()`, `PersistThemeTo()`, and `persistThemeToPath()` from `internal/config/config.go`. Delete their tests from `config_test.go`.
      - test: `make ci` passes — no compilation errors from removed functions
