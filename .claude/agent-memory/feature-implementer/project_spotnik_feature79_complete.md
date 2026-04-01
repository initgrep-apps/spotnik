---
name: project_spotnik_feature79_complete
description: Story 79 (PreferenceStore Engine): new internal/prefs package, debounced flush in app.go, PersistTheme deletion
type: project
---

## Story 79 — PreferenceStore Engine with Debounced Flush

**What was built:**
- New `internal/prefs/` package: `PreferenceStore` struct, `New()`, `Set()`, `HasPending()`, `FlushCmd()`, `FlushedMsg`
- `FlushCmd()` snapshots pending, clears, writes TOML preserving `[spotify]` section, re-queues on failure
- Wired into `app.go`: `prefs` and `prefsDirtyGen` fields on App struct, `prefsFlushTickMsg` type, `schedulePrefsFlush()` helper, `handlePrefsMsg()` router
- Replaced `persistThemeChoice()` / `config.PersistTheme()` with `prefs.Set("theme", id)` + `schedulePrefsFlush()` in ThemeSwitchMsg handler
- Deleted `PersistTheme`, `PersistThemeTo`, `persistThemeToPath` from `internal/config/config.go` and all their tests

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/prefs/prefs.go` — full package
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/prefs/prefs_test.go` — 10 tests
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/prefs_test.go` — 5 app-level tests

**Patterns established:**
- `prefs` package imports `config.PreferencesConfig` for TOML layout — this is an allowed cross-package dependency (not an import cycle)
- Exported test accessors on App: `Prefs()`, `PrefsDirtyGen()`, `SchedulePrefsFlush()` — consistent with existing patterns
- `handlePrefsMsg(msg) (tea.Model, tea.Cmd, bool)` pattern: early-return handler called before the main switch in `handleMsg`, returns `(model, cmd, handled)` bool
- Generation counter debounce: `prefsDirtyGen++` in `schedulePrefsFlush()`, tick message carries `generation int`, only flush if `m.generation == a.prefsDirtyGen`

**Gotchas:**
- `prefsDirtyGen` is incremented in `schedulePrefsFlush()` NOT in `prefs.Set()` — they are separate calls. Comment must reflect this.
- `writeToDisk` uses type assertions (`val.(string)`, `val.(int)`) — safe for current callsites but would panic if wrong type passed to `prefs.Set()` with a known key. Future story 80 will add preset/visualizer int callsites.
- `prefs.Set("theme", m.ThemeID)` is called inside `Update()` directly (not in a Cmd) — this is correct because it only stages in-memory state; disk I/O happens asynchronously in `FlushCmd()`.
- The old `persistThemeChoice` ran disk I/O inside a `func() tea.Msg` closure — violating the spirit of Elm architecture. New pattern is cleaner.
- `FlushCmd()` is called at execution time (not returned immediately) by the prefs handler — the tick fires 500ms after `schedulePrefsFlush()`, then `handlePrefsMsg` calls `a.prefs.FlushCmd()` which runs in the Tea runtime.

**Testing notes:**
- `TestApp_PrefsFlushTick_StaleGen_Ignored` uses `HasPending()` as observable side-effect (not cmd inspection) — because `Update()` batches with alerts cmd making cmd inspection unreliable for the nil case
- `TestFlushCmd_RequeuesOnFailure` triggers failure by pointing path to a directory (not a file) — `os.OpenFile` on a directory gives EISDIR
- Coverage: prefs 91.1%, app 82.1%, config 88.6%
