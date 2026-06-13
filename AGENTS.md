# AGENTS.md — Spotnik

> **Primary guidance file for OpenCode agents working on this codebase.**

Full project rules, architecture constraints, testing requirements, and development workflow
are in `CLAUDE.md` — read it in full before writing any code or making any decision.

## What We Are Building

**Spotnik** — a terminal Spotify client for developers. Keyboard-driven, single binary,
beautiful in a terminal. Not a Spotify clone — a developer-first music environment.
Target user: developer with Spotify Premium who lives in the terminal all day.

## Quick Reference

| What | Command |
|---|---|
| Build | `rtk make build` |
| Run | `make run` |
| Test | `rtk go test ./...` |
| Lint | `rtk make lint` |
| Coverage | `make test-coverage` (min 80%) |
| CI gate | `make ci` |

## Key Architecture Constraints

- **Go 1.26+** single binary, Bubble Tea TUI framework
- **All API data lives in the Store** — never in pane structs
- **Side effects only via tea.Cmd** — never call API inside `Update()` directly
- **View() must be pure** — no external calls, just read state → render
- **ui/ never imports api/** — data flows through messages and store only
- **All API errors route through toast notifications** — no inline error boxes in View()
- **Never hardcode hex colours** — use Theme interface tokens
- **Never use time.Sleep** — use tea.Tick for polling

## Branch Discipline

- Never work directly on `main`
- Feature branches: `feat/NN-feature-name`
- Fix branches: `fix/NNN-story-name`
- One PR per feature, never merge your own PR unless orchestrating

## Spec Location

Feature specs: `docs/spec/features/NN-name/`
System docs: `docs/system/{architecture,design,cli,tui,api-guide}.md`