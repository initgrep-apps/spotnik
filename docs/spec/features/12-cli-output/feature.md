---
title: "CLI Output Renderer"
status: open
---

## Description

Formalise the informal CLI output guidelines established in feature 09 (stories 141, 145)
into a reusable package `internal/cliout` plus a living reference doc
`docs/CLI-OUTPUT.md`. All current and future `spotnik` CLI subcommands consume the
package; no direct `fmt.Fprintln` for user-facing output.

**Design record:** `docs/superpowers/specs/2026-04-22-cli-output-design.md` — contains
the full rationale (brainstorming decisions, advisor gap resolutions, testing strategy).
This feature implements that spec.

**What changes:**

- New package `internal/cliout` with 9 typed message structs (`Header/Step/KV/Steps/
  Hint/URL/Paragraph/Spinner/Prompt`), a sealed `Message` interface, `Write(w, msgs...)`
  entry point, fluent `Builder`, and TTY-aware spinner + validated prompt primitives.
- New reference doc `docs/CLI-OUTPUT.md` — canonical guideline for CLI output work.
- `cmd/root.go` migrated to `cliout.*`; local `cliGreen/cliRed/cliYellow/cliDim`,
  `cliAccentS/cliDimS/cliErrS/cliWarnS/cliWrap`, `cliOut/cliLine/cliKV` removed.
- New config field `[cli] palette = "auto" | "fixed" | "theme"` (default `"auto"`)
  with auto-detect via `termenv.HasDarkBackground()`.
- OAuth flow (`RunAuthFlow`, `runRegister`) uses the new `Spinner` and `Prompt` types.

**Supersedes:** the "CLI Output Design" inline section of story
`09-auth-and-profile/stories/145-cli-auth-ux-polish-2.md`. Once merged,
`docs/CLI-OUTPUT.md` is the canonical reference. This feature's story 147 adds a
CLAUDE.md "NEVER Do" entry pinning that rule.

**Depends on:** nothing — `cmd/root.go` already works as-is. Each story below merges
green against `make ci` independently.

## Acceptance Criteria

- [ ] `internal/cliout` package exists with 9 typed message structs, sealed `Message`
      interface, `Write`, `WriteInline`, `Builder`, `Pair`, `PairWithCaption`
- [ ] `internal/cliout` has ≥90% test coverage (higher than project 80% floor)
- [ ] `internal/cliout.Capture` helper captures `[]Message` without rendering for
      structural tests
- [ ] `internal/cliout.SetTestMode(true)` disables spinner animation and SIGINT
      handlers for deterministic tests
- [ ] `docs/CLI-OUTPUT.md` exists with the taxonomy, emphasis roles, glyph set,
      palette resolution, spinner rules, prompt rules, and new-command checklist
- [ ] `CLAUDE.md` Reading Order references `docs/CLI-OUTPUT.md`
- [ ] `CLAUDE.md` "What Agents Must NEVER Do" has an entry for message-type/glyph
      additions without updating `docs/CLI-OUTPUT.md`
- [ ] `cmd/root.go` call sites use `cliout.*`; no local `cliGreen`/`cliOut`/`cliKV`
- [ ] All five auth subcommands produce output equivalent to what ships today, with
      one intentional layout change: `spotnik auth register` now renders the
      redirect URI on its own line (accent-coloured) below the numbered instruction
      "Add this redirect URI:". Golden files confirm stability post-migration.
- [ ] `config.Config` has a `CLI` section with `Palette string` field
- [ ] `config.Bootstrap` writes `[cli] palette = "auto"` into new config files
- [ ] `NO_COLOR` env var strips ANSI from all CLI output
- [ ] `cli.palette = "fixed"` forces the built-in palette
- [ ] `cli.palette = "theme"` uses TUI theme tokens
- [ ] `cli.palette = "auto"` uses theme tokens on TTY + dark bg, fixed otherwise
- [ ] `RunAuthFlow` uses `cliout.StartSpinner` for the callback wait; spinner animates
      on TTY and prints a single static line on pipe
- [ ] `runRegister` uses `cliout.Ask` with a validator that rejects non-32-char or
      non-hex input; re-prompts up to 3 times
- [ ] Spinner survives Ctrl+C — cursor restored, exit 130, no dangling terminal state
- [ ] `make ci` passes after every story

## Story Order

1. `146-cliout-package-and-static-types.md` — create the package with static message
   types, palette resolution, tests, reference doc. No call-site changes.
2. `147-cliout-migrate-cmd-root.md` — migrate `cmd/root.go` auth subcommand output
   to `cliout.*`; remove local helpers.
3. `148-cliout-palette-config.md` — add `[cli] palette` to the config schema;
   wire resolution.
4. `149-cliout-spinner-and-prompt.md` — implement the dynamic `Spinner` and `Prompt`
   message types; integrate into `RunAuthFlow` and `runRegister`.
