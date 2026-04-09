---
title: "Onboarding Docs, Test Infrastructure & Doc Corrections"
feature: 22-developer-foundations
status: done
---

## Background

The 2026-04-08 code audit identified four categories of gap in this story:

1. **Missing onboarding docs** — `README.md` is 3 lines, no `CONTRIBUTING.md`,
   `DEV-SETUP.md`, `TESTING.md`, or `PANE-TEMPLATE.md` exist. New contributors (and
   autonomous agents) have no entry point.
2. **Duplicated fixture loading** — `os.ReadFile("../../testdata/fixtures/<name>.json")`
   appears 15+ times across 7 `internal/api/*_test.go` files. Path is fragile and the
   pattern is not centralised.
3. **Undiscoverable integration tests** — 5 integration test files exist with
   `//go:build integration` but there is no `make test-integration` target.
4. **Inaccurate reference docs** — `docs/ARCHITECTURE.md` is missing 6 sections;
   `docs/DESIGN.md` has 3 outdated entries.

All changes in this story are purely additive or corrective. No behaviour changes.

**Source:** `docs/code-audit/code-audit-design.md` §4–§5,
`docs/code-audit/phase1-docs-testinfra.md`

**Depends on:** Nothing. Can be implemented on a clean branch from `main`.

---

## Design

### Task 1 — Centralised fixture loader

**File to create:** `internal/testhelpers/fixtures.go`

```go
package testhelpers

import (
    "os"
    "path/filepath"
    "runtime"
    "testing"

    "github.com/stretchr/testify/require"
)

var fixturesDir = func() string {
    _, file, _, ok := runtime.Caller(0)
    if !ok {
        panic("testhelpers: cannot determine file location via runtime.Caller")
    }
    root := filepath.Join(filepath.Dir(file), "..", "..")
    return filepath.Join(root, "testdata", "fixtures")
}()

// LoadFixture reads a JSON fixture from testdata/fixtures/ by name.
// Fails the test immediately if the file cannot be read.
func LoadFixture(t *testing.T, name string) []byte {
    t.Helper()
    path := filepath.Join(fixturesDir, name)
    data, err := os.ReadFile(path)
    require.NoError(t, err, "testhelpers.LoadFixture: failed to read %s", path)
    return data
}
```

`runtime.Caller(0)` resolves the path relative to the helper file itself, so callers
in any package do not need to know their depth from the project root.

**Files to update:** every `internal/api/*_test.go` that contains
`os.ReadFile(…testdata/fixtures…)`:
- `models_test.go`, `player_test.go`, `library_test.go`, `search_test.go`,
  `devices_test.go`, `playlists_test.go`, `user_test.go`

Replace each occurrence of:
```go
fixture, err := os.ReadFile("../../testdata/fixtures/<name>.json")
require.NoError(t, err)
```
with:
```go
fixture := testhelpers.LoadFixture(t, "<name>.json")
```

Remove the now-unused `os` import from any file where it is no longer referenced.

### Task 2 — `make test-integration` target

**File to update:** `Makefile`

Add after the `test:` block:
```makefile
## Run integration tests (requires build tag)
test-integration:
	@echo "→ Running integration tests..."
	$(GO) test -tags integration ./... -race -count=1
	@echo "✓ Integration tests passed"
```

Add `test-integration` to the `.PHONY` declaration.

### Task 3 — Onboarding documentation

Create the following files. Content must match the established project conventions
(Go 1.22, cobra CLI, Bubble Tea, PKCE auth, testify, `make ci` gate):

| File | Purpose |
|------|---------|
| `README.md` | Replace the 3-line stub: project description, features list, install instructions from source, key bindings table, link to CONTRIBUTING |
| `CONTRIBUTING.md` | Branch naming, commit conventions (Conventional Commits), `make ci` gate, test requirements (`LoadFixture`, table-driven, 80% threshold), architecture rules summary, PR process |
| `docs/DEV-SETUP.md` | Prerequisites (Go 1.22+, golangci-lint), Spotify app setup (dashboard, redirect URI), `.env` file for `SPOTIFY_CLIENT_ID`, build/run commands, test commands, linting, debugging tips |
| `docs/TESTING.md` | When to run each make target, table-driven test pattern, httptest pattern for API clients, pane test pattern, `LoadFixture` usage, integration test file conventions, Elm purity test locations |
| `docs/PANE-TEMPLATE.md` | Step-by-step: add `PaneID` constant, create pane file with full struct/interface scaffold, add message types to `messages.go`, wire into `app.go` handler, add to preset, write tests, verify coverage |

### Task 4 — Fix `docs/ARCHITECTURE.md`

Add six missing sections (insert at the indicated locations — read the current
file first to find the right position):

1. **PreferenceStore** — after the "State Management" section. Document
   `internal/prefs/prefs.go`, generation-counter debounce pattern, three supported
   prefs (theme, preset, visualizer), disk path.
2. **Page / Preset / Toggle system** — after the "Render Pipeline" section. Document
   `TogglePage()`, `CyclePreset()`, `TogglePane(id)`, key bindings (`0`, `p`, `1–8`),
   Page A's 4 presets, Page B's 1 preset.
3. **View lifecycle** — after the "Architectural Overview" section. Document three
   `currentView` modes (`viewSplash → viewAuth → viewGrid`) and their transitions.
4. **`SetTheme` in Pane interface** — in the existing Pane interface description,
   add `SetTheme(th theme.Theme)` with a note that table-based panes must rebuild
   their tables with new column colors.
5. **Overlay routing precedence** — in the "Message Flow" section, document the
   guard order: theme overlay → device overlay → search overlay → auth view →
   active filter → global shortcuts → playback keys → focused pane.
6. **`http.DefaultClient` known issue** — in the "API Client Design" section, add
   a note that `postTokenRequest` in `auth.go` uses `http.DefaultClient` until
   Story 111 fixes it.

### Task 5 — Fix `docs/DESIGN.md`

Three targeted corrections (read the file first, locate the exact sections):

1. **§2 Pane interface** — add `SetTheme(th theme.Theme)` to the method table with
   description: "Updates the pane's theme for runtime switching; table panes must
   rebuild their tables with new column colors."
2. **§17 Keybindings** — find the `?` row and suffix the label with
   `*(PLANNED — not yet implemented)*`. Do not remove the row.
3. **§18 Themes** — update the theme count from "5 existing themes" to "11 themes".
   For each of the 6 undocumented themes (dracula, gruvbox, rosepine, solarized,
   synthwave, tokyonight), add a subsection following the same structure as the
   existing documented themes (theme ID, display name, token → hex value table).
   Read the corresponding `.toml` files in `internal/ui/theme/themes/` to get the
   values. Consolidate repeated token definitions into a single "Token Reference"
   table at the top of §18 and use compact hex-only rows per theme.

---

## Acceptance Criteria

- [ ] `internal/testhelpers/fixtures.go` exists and compiles (`go build ./internal/testhelpers/...`)
- [ ] All 7 `internal/api/*_test.go` files use `testhelpers.LoadFixture`; no `os.ReadFile`
      calls for fixtures remain
- [ ] `go test ./internal/api/... -race -count=1` passes
- [ ] `make test-integration` target exists and runs the 5 integration test files
- [ ] `README.md` has features list, install instructions, and keybinding table
- [ ] `CONTRIBUTING.md`, `docs/DEV-SETUP.md`, `docs/TESTING.md`, `docs/PANE-TEMPLATE.md` exist
- [ ] `docs/ARCHITECTURE.md` contains sections: PreferenceStore, Page/Preset/Toggle,
      View Lifecycle, overlay routing precedence; Pane interface includes `SetTheme`
- [ ] `docs/DESIGN.md`: `SetTheme` in §2, `?` marked as planned in §17, all 11 themes in §18
- [ ] `make ci` passes

## Tasks

- [ ] Create `internal/testhelpers/fixtures.go` with `LoadFixture` helper
      - test: `go build ./internal/testhelpers/...` compiles with no errors
- [ ] Replace inline fixture loads in all 7 `internal/api/*_test.go` files
      - test: `go test ./internal/api/... -race -count=1` passes; no `os.ReadFile` fixture calls remain
- [ ] Add `make test-integration` target to `Makefile`
      - test: `make test-integration` runs and all 5 integration tests pass
- [ ] Write `README.md` (replace 3-line stub)
      - test: content review — features, install, keybindings present
- [ ] Write `CONTRIBUTING.md`
      - test: content review — branch naming, commit format, `make ci`, test requirements, PR process
- [ ] Write `docs/DEV-SETUP.md`
      - test: content review — prerequisites, Spotify app setup, `.env` usage, make targets
- [ ] Write `docs/TESTING.md`
      - test: content review — table-driven pattern, httptest pattern, `LoadFixture`, integration tag
- [ ] Write `docs/PANE-TEMPLATE.md`
      - test: content review — all 6 steps present, code scaffold correct
- [ ] Add 6 missing sections to `docs/ARCHITECTURE.md`
      - test: `grep` for each section heading confirms presence; `make ci` passes
- [ ] Fix `docs/DESIGN.md` §2, §17, §18
      - test: `grep 'SetTheme' docs/DESIGN.md` → hit; `grep 'PLANNED' docs/DESIGN.md` → hit;
        count of theme subsections in §18 = 11
