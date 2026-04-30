---
title: "CI guards: banned-glyph + catalogue-leak + chrome-caller, LANG=C matrix, smoke test"
feature: 13-tui-design-system
status: done
---

## Background

Stories 183–191 close every glyph-fallback gap identified by the audit. This story is
the wrap-up: it adds the CI machinery that prevents regressions and runs the manual
smoke test that gates the feature complete.

Three guards land:

1. **`scripts/check-banned-glyphs.sh`** — fails CI on any of the 13 banned glyphs
   (`⚠`, `ᐅ`, `┌┐└┘`, `╔╗╚╝`, `✅`, `❌`, `❗`).
2. **`scripts/check-catalogue-leaks.sh`** — fails CI when catalogue characters appear
   outside `internal/uikit/glyph.go` and the canonical doc files.
3. **`scripts/check-render-pane-border.sh`** — fails CI when any file outside
   `internal/uikit/` (and `internal/ui/layout/` itself, which owns the function) calls
   `layout.RenderPaneBorder` directly. Plan task 3.7 placed this guard in story 186,
   but moving it here avoids a merge-order trap where 186 lands before 185 and the
   guard self-fails.

Plus a CI matrix that runs the full test suite under both `LANG=en_US.UTF-8` and
`LANG=C`, and the manual smoke test that walks every visible surface under
`ui.glyphs = "ascii"`.

After this story, the audit's §7 acceptance criteria can be ticked.

**Depends on:** stories 184, 185, 186, 187, 188, 189, 190, 191. The catalogue-leak
and chrome-caller guards both fail until those stories land. Cliout ASCII test sweep
(plan task 6.1) consolidates here as well.

**Plan tasks:** 3.7 (chrome guard moved here from story 186), 6.1, 6.2, 6.3.

**Files created:** `scripts/check-banned-glyphs.sh`,
`scripts/check-catalogue-leaks.sh`, `scripts/check-render-pane-border.sh`.
**Modified:** `Makefile`, `.github/workflows/ci.yml`,
`internal/cliout/*_test.go` (parameterise any remaining single-mode assertions).

## Design

### `scripts/check-banned-glyphs.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail
BANNED=( "⚠" "ᐅ" "┌" "┐" "└" "┘" "╔" "╗" "╚" "╝" "✅" "❌" "❗" )
for g in "${BANNED[@]}"; do
    if grep -rn --include="*.go" "$g" internal/ cmd/ 2>/dev/null; then
        echo "ERROR: banned glyph '$g' present in source"
        exit 1
    fi
done
echo "OK: no banned glyphs"
```

### `scripts/check-catalogue-leaks.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail
CHARS=( "╭" "╮" "╰" "╯" "✓" "✗" "◬" "→" "⧖" "◉" "◎" "○" "●" "◌" "▶" "▷" "⏸" "≡" "↻" "⇄" "♪" "▤" "█" "▒" "•" "…" )

LEAKS=""
for c in "${CHARS[@]}"; do
    found=$(grep -rn --include="*.go" "$c" internal/ cmd/ 2>/dev/null \
        | grep -v "internal/uikit/glyph.go" \
        | grep -v "_test.go" || true)
    if [ -n "$found" ]; then
        LEAKS="$LEAKS\n$found"
    fi
done

if [ -n "$LEAKS" ]; then
    echo "ERROR: catalogue characters leaked outside internal/uikit/glyph.go:"
    printf "%b\n" "$LEAKS"
    exit 1
fi
echo "OK: no catalogue leaks"
```

### `scripts/check-render-pane-border.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail

OFFENDERS=$(grep -rn --include="*.go" "layout\.RenderPaneBorder\|RenderPaneBorder(" internal/ \
    | grep -v "internal/uikit/" \
    | grep -v "internal/ui/layout/" \
    | grep -v "_test.go" || true)

if [ -n "$OFFENDERS" ]; then
    echo "ERROR: layout.RenderPaneBorder called outside internal/uikit/ — use uikit.PaneChrome / OverlayChrome instead."
    echo "$OFFENDERS"
    exit 1
fi
echo "OK: no direct RenderPaneBorder callers outside uikit"
```

### `Makefile` target

```makefile
.PHONY: check-glyphs
check-glyphs:
	@scripts/check-banned-glyphs.sh
	@scripts/check-catalogue-leaks.sh
	@scripts/check-render-pane-border.sh
```

Wire `check-glyphs` into `make ci` so it runs alongside lint + test.

### CI workflow

`.github/workflows/ci.yml` — add the guard step:

```yaml
- name: Glyph fallback guards
  run: make check-glyphs
```

Add the locale matrix (adapt to existing workflow shape):

```yaml
strategy:
  matrix:
    locale: [en_US.UTF-8, C]
steps:
  - run: LANG=${{ matrix.locale }} make test
```

### cliout ASCII test sweep

After story 184 the cliout suite parameterises most assertions. Run a final pass to
confirm every glyph-bearing message type has at least one ASCII assertion. Promote
single-mode tests to a both-modes loop:

```go
for _, mode := range []uikit.GlyphMode{uikit.GlyphUnicode, uikit.GlyphASCII} {
    t.Run(modeName(mode), func(t *testing.T) {
        uikit.SetModeForTest(mode)
        defer uikit.SetModeForTest(uikit.GlyphUnicode)
        // assertions resolve via uikit.GlyphFor
    })
}
```

### Manual smoke test

Build the binary, set `ui.glyphs = "ascii"` in `~/.config/spotnik/config.toml`, run
under `LANG=C`, and walk:

- All 10 grid panes on Page A
- All Page B panes (gateway, polling, networklog, etc.)
- Each overlay: devices, help, search, profile, themes
- Splash screen
- Onboarding flow (start with no token)
- Toast notifications: trigger one of each intent (success / error / warning / info /
  ratelimit) and confirm prefix
- Visualizer: confirm `# = .` columns appear in nowplaying

Document each surface as a checklist item in the PR description; attach screenshots
if helpful.

Confirm no regressions under `LANG=en_US.UTF-8` and `glyphs = "auto"`.

## Acceptance Criteria

- [ ] `scripts/check-banned-glyphs.sh` exists, is executable, and passes locally on
      `main` after this story merges
- [ ] `scripts/check-catalogue-leaks.sh` exists, is executable, and passes locally on
      `main` after this story merges
- [ ] `scripts/check-render-pane-border.sh` exists, is executable, and passes
      locally on `main` after this story merges; flags any new direct caller of
      `layout.RenderPaneBorder` from outside `internal/uikit/`
- [ ] `make check-glyphs` runs all three scripts and exits 0 on a clean tree
- [ ] `make ci` invokes `make check-glyphs`
- [ ] The CI workflow (`.github/workflows/ci.yml`) has a `Glyph fallback guards`
      step that runs `make check-glyphs`
- [ ] CI runs the test suite under a `LANG` matrix (`en_US.UTF-8` and `C`); both
      legs pass
- [ ] Every cliout glyph-bearing message-type test runs in both unicode and ASCII
      modes (the both-modes loop pattern, or two pinned tests)
- [ ] Manual smoke test under `LANG=C` and `ui.glyphs = "ascii"` walks every grid
      pane, overlay, splash, onboarding, every toast intent, and the visualizer;
      results documented in the PR description as a checklist
- [ ] Sanity pass under `LANG=en_US.UTF-8` and `ui.glyphs = "auto"`: no regressions
- [ ] Audit §7 acceptance criteria all tick:
      catalogue characters appear only in `uikit/glyph.go` + canonical docs;
      every primitive has an ASCII snapshot test;
      `cliout` resolves every glyph and spinner frame via `uikit`;
      ASCII config produces a fully readable TUI;
      visualizer renders `# = .` columns in ASCII mode;
      `LANG=C` and `LANG=en_US.UTF-8` both pass full test suite;
      banned-glyph grep returns zero hits.
- [ ] `feature.md` Glyph-fallback acceptance criteria all tick; `feature.md` status
      updates to `done`
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Tasks 3.7, 6.1, 6.2, 6.3 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

- [ ] Branch: `chore/13-glyph-fallback-ci-guards`
- [ ] Create `scripts/check-banned-glyphs.sh`; `chmod +x`
- [ ] Create `scripts/check-catalogue-leaks.sh`; `chmod +x`
- [ ] Create `scripts/check-render-pane-border.sh`; `chmod +x`
- [ ] Append `check-glyphs` target to `Makefile`; chain into `ci` target
- [ ] Run `make check-glyphs` locally → PASS
- [ ] If any guard fails, audit the offender and either move it into a primitive or
      schedule a follow-up story before merging this one
- [ ] Commit: `chore(ci): guard against direct layout.RenderPaneBorder callers`
      (the chrome guard is one logical commit)
- [ ] Commit: `chore(ci): add banned-glyph and catalogue-leak guards plus LANG=C matrix`
      (combined with workflow edit)
- [ ] Run cliout ASCII test sweep — promote any single-mode assertions to both-modes
      loops; commit: `test(cliout): close ASCII test-coverage gaps for every glyph-bearing message type`
- [ ] Edit `.github/workflows/ci.yml` to add `Glyph fallback guards` step + `LANG`
      matrix; commit alongside the guards commit above
- [ ] `make ci` → PASS
- [ ] Build the binary; set `ui.glyphs = "ascii"` in config; run under `LANG=C`
- [ ] Walk every visible surface (Page A grid panes, Page B panes, all overlays,
      splash, onboarding, every toast intent, visualizer); document in the PR
      description
- [ ] Walk the same surfaces under `LANG=en_US.UTF-8` + `glyphs = "auto"`; confirm
      no regressions
- [ ] Update `feature.md` status: `in-progress` → `done`; tick the
      Glyph-fallback acceptance-criteria checkboxes
- [ ] Update `docs/spec/00-overview.md` Stories column for feature 13 to extend
      through 192; refresh the description and "Last updated" line
- [ ] Push branch + open PR
