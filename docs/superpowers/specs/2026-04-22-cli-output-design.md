# CLI Output Design — `internal/cliout`

**Date:** 2026-04-22
**Feature context:** `docs/spec/features/09-auth-and-profile/` — latest stories (141, 145) established
informal CLI guidelines while implementing styled auth subcommand output. This design
formalises those guidelines and moves the implementation out of `cmd/root.go` into a
reusable package.

**Supersedes:** the "CLI Output Design" inline section of story
`docs/spec/features/09-auth-and-profile/stories/145-cli-auth-ux-polish-2.md`. Once this
design lands, `docs/CLI-OUTPUT.md` is the canonical reference.

**Scope:** all current and future `spotnik` CLI subcommands. TUI output is out of scope
— governed by `docs/DESIGN.md`.

**Migration:** delivered as four sequential stories inside feature `NN-cli-output`.
`cmd/root.go` keeps shipping throughout — no big-bang rewrite.

---

## 1 — Problem

Current CLI output in `cmd/root.go` works but has two systemic issues:

1. **No written guidelines.** Rules (one accent colour, no emoji, no box borders, specific
   glyph set, `Padding(0, 2)` everywhere) live informally inside story 145. Future CLI
   work has no reference to consult. Readers must spelunk git history.

2. **Implementation scattered.** Styling is composed ad hoc at ~10 call sites:
   `cliAccentS.Render("→") + " Run " + cliAccentS.Render("spotnik auth register") + " to try again"`.
   Helpers `cliOut/cliLine/cliKV` exist but each call site still hand-builds the
   glyph/colour/word composition. Easy to drift. No typed message surface.

## 2 — Goals

1. Formal CLI output taxonomy — typed message catalogue with field-level emphasis rules.
2. Reusable Go package (`internal/cliout`) consumed by every CLI subcommand. No direct
   `fmt.Fprintln` for user-facing output after migration.
3. Canonical reference doc (`docs/CLI-OUTPUT.md`) — treated as a living spec like
   `keybinding.md` / `DESIGN.md`.
4. Consistent handling of colour, TTY/pipe detection, loading states, and interactive
   input.
5. Testability at structural level — assert on `[]Message`, not on styled strings.

## 3 — Non-goals

- TUI panes / renderers — governed by `DESIGN.md`, not changing here.
- Cross-binary CLI framework — this is spotnik-specific.
- Rich shell-native prompt features (arrow keys, history, paste highlight) — deliberate
  minimalism in favour of zero dependencies.

## 4 — Decisions (answers to brainstorming questions)

| # | Question | Decision |
|---|---|---|
| Q1 | Scope | New package `internal/cliout`, formal design doc, supersedes story 145 guidelines |
| Q2 | API shape | Hybrid — typed `Message` structs + single `Write(w, msgs...)`; thin fluent `Builder` wrapper over the same `[]Message` |
| Q3 | Loading states | Event-driven step lines + TTY-guarded hand-rolled spinner for unbounded waits |
| Q4 | Colour palette | Theme-driven with `HasDarkBackground()` auto-gate; `cli.palette = "fixed" \| "theme" \| "auto"` config (default `"auto"`) |
| Q5 | Prompt | Minimal — styled label + validated `bufio.Scanner` loop. No bubbletea |

## 5 — Message catalogue

Nine typed messages. Three rendering modes: **static** (lipgloss → `Fprintln`),
**dynamic** (spinner — goroutine + `\r`), **input** (prompt — scanner loop).

| # | Type | Purpose | Sample | Mode |
|---|---|---|---|---|
| 1 | `Header` | Primary status line — subject + state | `◉ Spotnik  authenticated` | static |
| 2 | `Step` | Inline progress event (≤1s) | `✓ Authorization received` | static |
| 3 | `KV` | Aligned key/value facts | `Client ID  present` | static |
| 4 | `Steps` | Numbered instruction list | `1  Go to developer.spotify.com/dashboard` | static |
| 5 | `Hint` | Action suggestion with command | `→ Run spotnik auth login to reconnect` | static |
| 6 | `URL` | Bare URL on its own line (optional label) | `https://accounts.spotify.com/authorize?…` | static |
| 7 | `Paragraph` | Free prose line (dim or default) | `Tokens and client ID removed` | static |
| 8 | `Spinner` | Long unbounded wait (>1s) | `⣾ Waiting for authorization` | dynamic |
| 9 | `Prompt` | Interactive input with validation | `Client ID: _` | input |

### 5.1 Statuses

Used by `Header` and `Step`. Pairs a glyph with a colour role.

| Name | Glyph | Colour role |
|---|---|---|
| `Active` | `◉` | Accent |
| `Inactive` | `◎` | Muted |
| `Success` | `✓` | Accent (green) |
| `Failure` | `✗` | Error (red) |
| `Warning` | `⚠` | Warning (yellow) |
| `Pending` | `◌` | Muted |

Glyph set is **frozen**. No emoji, no ASCII art, no box borders. Adding a new glyph
requires updating `docs/CLI-OUTPUT.md` in the same commit.

## 6 — Emphasis roles

Four levels. Baked into message types at field level — call sites never pass a colour.

| Level | Visual | Purpose |
|---|---|---|
| **Accent** | green bold | User must interact with this thing (URL, command, redirect URI, action arrow) |
| **Strong** | default fg, bold | Headline subject the eye lands on ("Spotnik", "Signed in") |
| **Plain** | default fg | Regular body (KV values, hint tails, step text) |
| **Muted** | dim | Secondary context (KV labels, captions, reasons, prompt labels, inactive state) |

Role colours (Success/Warning/Error) appear only on the **glyph** and at most one
tagged status word. Body text alongside stays Plain or Muted — keeps the palette quiet.

### 6.1 Field-level tagging

| Type.Field | Role |
|---|---|
| `Header.Glyph` | Status colour |
| `Header.Subject` | Strong |
| `Header.State` | Plain |
| `Step.Glyph` | Status colour |
| `Step.Text` | Plain |
| `KV.Pairs[].Label` | Muted |
| `KV.Pairs[].Value` | Plain |
| `KV.Pairs[].Caption` | Muted |
| `Steps.Items[].Number` | Muted |
| `Steps.Items[].Text` | Plain |
| `Hint.Arrow` | Accent |
| `Hint.Verb` | Plain |
| `Hint.Cmd` | Accent |
| `Hint.Tail` | Plain |
| `URL.Label` | Muted |
| `URL.Href` | Accent |
| `Paragraph.Text` | Plain or Muted (explicit field) |
| `Spinner.Text` (spinning) | Muted |
| `Prompt.Label` | Muted |
| `Prompt.Value` (typed) | Plain |
| Validation-failure line (rendered, not a struct field) | Error glyph + Plain text |

### 6.2 Hard rules

1. One Accent span per logical call to action — URL OR command, not both wrapped.
2. Never accent a value that's only informational (e.g., client ID string in a KV row
   is Plain, not Accent — it's a fact, not an action).
3. Role colours (yellow/red) appear only on glyph + at most one status word.
4. Muted is the default for labels and captions — when in doubt, dim it.
5. Strong is bold, not bright — contrast via weight, not colour.

## 7 — Colour policy

### 7.1 Resolution order

At every CLI invocation, `cliout.palette()` resolves in order:

1. **Env** — `NO_COLOR` set (any value) → all colour disabled, plain text everywhere.
2. **Config** — `cli.palette` in `config.toml`:
   - `"fixed"` — always the fixed palette
   - `"theme"` — always the active TUI theme's tokens (no contrast check)
   - `"auto"` (default) — decide per step 3
3. **Auto-detect** (only reached when config is `"auto"`):
   - `isatty(stderr)` **AND** `termenv.HasDarkBackground() == true` **AND**
     `termenv.EnvNoColor() == false` → **theme palette**
   - Otherwise → **fixed palette**

### 7.2 Fixed palette (safe default)

```go
var Fixed = Palette{
    Accent:  lipgloss.AdaptiveColor{Dark: "#1DB954", Light: "#1A8C41"},
    Success: lipgloss.AdaptiveColor{Dark: "#1DB954", Light: "#1A8C41"},
    Error:   lipgloss.AdaptiveColor{Dark: "#FF5555", Light: "#CC0000"},
    Warning: lipgloss.AdaptiveColor{Dark: "#F1C40F", Light: "#B8860B"},
    Muted:   lipgloss.AdaptiveColor{Dark: "#6C7083", Light: "#888888"},
    Plain:   lipgloss.AdaptiveColor{Dark: "", Light: ""},
}
```

Same hex values story 145 shipped. No regression.

### 7.3 Theme palette

```go
func themePalette(t theme.Theme) Palette {
    return Palette{
        Accent:  t.Accent(),
        Success: t.Success(),
        Error:   t.Error(),
        Warning: t.Warning(),
        Muted:   t.TextMuted(),
        Plain:   t.TextPrimary(),
    }
}
```

### 7.4 Non-TTY stripping

ANSI stripping is applied **on the first `Write` / `StartSpinner` / `Ask` call** (lazy),
not in `init()` — `init()` runs before we know whether output is a TTY. First-call
resolution checks `isatty(w)` and `NO_COLOR`; if either indicates stripping is needed,
`lipgloss.SetColorProfile(termenv.Ascii)` is called once (idempotent via a
`sync.Once`). Guarantees:

- `spotnik auth status > log.txt` → plain text file, no ANSI
- `spotnik auth status | less -R` → default keeps colour (stderr TTY detection; `less -R`
  handles ANSI)
- CI logs → plain text (CI sets `NO_COLOR=1` or output is not a TTY)

### 7.5 Config addition

`internal/config/config.go`:

```toml
[cli]
# Palette: "auto" (default) | "fixed" | "theme"
# - auto:  theme colours on dark-bg terminals, fixed elsewhere
# - fixed: always the built-in Spotnik palette
# - theme: inherit the TUI theme (may be unreadable on light terminals)
palette = "auto"
```

Backwards compatible — missing field resolves to `"auto"`.

## 8 — Package layout

```
internal/cliout/
├── doc.go         — package doc: references docs/CLI-OUTPUT.md
├── palette.go     — Palette struct, Fixed, themePalette, resolution
├── message.go     — Message interface + 9 typed structs + sealed marker
├── render.go      — Write(w, msgs...), renderStatic switch, padding
├── builder.go     — Builder (fluent façade over []Message)
├── spinner.go     — StartSpinner, SpinnerHandle (goroutine + \r)
├── prompt.go      — Ask, Prompt type, validator loop
├── tty.go         — isTTY helpers, NO_COLOR check, profile pinning
├── testing.go     — Capture, Recorder, SetTestMode
└── *_test.go
```

Flat layout — no sub-packages. One responsibility: turn `Message` values into bytes.

### 8.1 Core types

```go
// Message is implemented by every CLI message type.
// render returns the string; it never calls Fprintln itself.
type Message interface {
    render(p Palette) string
    isMessage() // sealed marker — external packages can't add types
}

type Status int
const (
    Active Status = iota
    Inactive
    Success
    Failure
    Warning
    Pending
)

type Header    struct { Status Status; Subject, State string }
type Step      struct { Status Status; Text string }
type KV        struct { Pairs []KVPair }
type KVPair    struct { Label, Value, Caption string }
type Steps     struct { Items []string } // numbered 1..N automatically
type Hint      struct { Verb, Cmd, Tail string }
type URL       struct { Label, Href string } // Label optional precursor
type Paragraph struct { Text string; Dim bool }
type Spinner   struct { Text string }
type Prompt    struct {
    Label, Placeholder string
    Validate           func(string) error
}
```

### 8.2 Entry points

```go
// Write renders each message and prints with the standard 2-char left
// padding and one leading blank line for section separation.
func Write(w io.Writer, msgs ...Message)

// WriteInline renders with no leading blank — for compact step-by-step progress.
func WriteInline(w io.Writer, msgs ...Message)

// New returns a Builder over the same []Message.
func New() *Builder

// StartSpinner returns a handle to a running spinner (or no-op on non-TTY).
func StartSpinner(w io.Writer, text string) *SpinnerHandle

// Ask renders a Prompt and returns the validated value.
// Returns ErrAborted on Ctrl+C, EOF, or 3 validation failures.
func Ask(r io.Reader, w io.Writer, p Prompt) (string, error)
```

### 8.3 Builder

```go
type Builder struct { msgs []Message }

func (b *Builder) Header(s Status, subject, state string) *Builder
func (b *Builder) KV(pairs ...KVPair) *Builder
func (b *Builder) Hint(verb, cmd, tail string) *Builder
func (b *Builder) URL(label, href string) *Builder
func (b *Builder) Step(s Status, text string) *Builder
func (b *Builder) Paragraph(text string) *Builder
func (b *Builder) Dim(text string) *Builder
func (b *Builder) Steps(items ...string) *Builder

func (b *Builder) Messages() []Message              // for tests
func (b *Builder) WriteTo(w io.Writer) (int64, error)

// Pair is a shorthand constructor for KVPair to keep call sites compact.
func Pair(label, value string) KVPair            { return KVPair{Label: label, Value: value} }
func PairWithCaption(label, value, cap string) KVPair { return KVPair{Label: label, Value: value, Caption: cap} }
```

Both forms are valid and compile to the same slice:

```go
// Data form
cliout.Write(w,
    cliout.Header{Status: cliout.Active, Subject: "Spotnik", State: "authenticated"},
    cliout.KV{Pairs: []cliout.KVPair{
        {Label: "Client ID", Value: "present"},
        {Label: "Expires",   Value: "Wed, 23 Apr 2026 10:00 UTC"},
    }},
)

// Fluent form
cliout.New().
    Header(cliout.Active, "Spotnik", "authenticated").
    KV(cliout.Pair("Client ID", "present"), cliout.Pair("Expires", "Wed, 23 Apr 2026 10:00 UTC")).
    WriteTo(w)
```

## 9 — Spinner

### 9.1 Behaviour

- **TTY:** goroutine redraws `⣾ Waiting for authorization` on the same line every
  100 ms using `\r`. Frames `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`. Text is Muted; frame char is Accent.
- **Non-TTY / pipe / `NO_COLOR`:** writes `◌ Waiting for authorization\n` once, returns
  a no-op handle.
- **Resolution:**
  - `Done(text)` — clears line (`\r\x1b[K` on TTY), writes `✓ text` as `Step{Success}`.
  - `Fail(text)` — writes `✗ text` as `Step{Failure}`.
  - `Stop()` — silent cancel (no line written). Idempotent.
- **SIGINT:** a package-level signal handler is installed lazily on the first
  `StartSpinner` call (via `sync.Once`) and remains installed for the process
  lifetime. It cancels all active spinner handles, restores the cursor
  (`\x1b[?25h`), and exits 130. Only installed when the process actually spins;
  CLI commands that never spin (e.g. `auth status`, `auth logout`) do not alter
  the default SIGINT behaviour.
- **Cursor hide:** `\x1b[?25l` on start, `\x1b[?25h` on resolve. TTY only.

### 9.2 API

```go
type SpinnerHandle struct { /* unexported */ }

func StartSpinner(w io.Writer, text string) *SpinnerHandle
func (h *SpinnerHandle) Done(text string)
func (h *SpinnerHandle) Fail(text string)
func (h *SpinnerHandle) Stop()
```

### 9.3 Test mode

```go
// SetTestMode disables animation and signal handlers.
// Spinner becomes line-per-event — Start writes one line, Done/Fail writes one line.
func SetTestMode(enabled bool)
```

`TestMain` in each CLI-testing package calls `cliout.SetTestMode(true)`.

## 10 — Prompt

### 10.1 Behaviour

- Styled `Label: ` in Muted, then cursor. `bufio.Scanner` read.
- Trimmed input passed to `Validate`. `nil` return → accept. Non-nil → print
  `✗ <err>` as `Step{Failure}` beneath the prompt, re-prompt.
- **Retries:** up to 3 validation failures. After third, print
  `✗ Giving up after 3 attempts` and return `ErrAborted`.
- **Ctrl+C / EOF:** returns `ErrAborted`. Caller treats as `errAlreadyPrinted` (no
  styled block from `Execute`).
- **Empty input:** handled by caller's validator. No special casing.

### 10.2 API

```go
type Prompt struct {
    Label       string
    Placeholder string              // Muted; shown after label before cursor
    Validate    func(string) error  // nil accepts anything
}

var ErrAborted = errors.New("prompt aborted")

func Ask(r io.Reader, w io.Writer, p Prompt) (string, error)
```

### 10.3 Example

```go
validateClientID := func(s string) error {
    s = strings.TrimSpace(s)
    if len(s) != 32 {
        return fmt.Errorf("client ID must be 32 characters (got %d)", len(s))
    }
    if _, err := hex.DecodeString(s); err != nil {
        return fmt.Errorf("client ID must be hexadecimal")
    }
    return nil
}

clientID, err := cliout.Ask(os.Stdin, w, cliout.Prompt{
    Label:       "Client ID",
    Placeholder: "32 hex characters from developer.spotify.com/dashboard",
    Validate:    validateClientID,
})
if err != nil {
    return errAlreadyPrinted
}
```

### 10.4 Output walkthroughs

Happy path:
```
  Client ID: a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6
```

Validation fail, retry succeeds:
```
  Client ID: abc
  ✗ client ID must be 32 characters (got 3)

  Client ID: a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6
```

Three strikes:
```
  Client ID: abc
  ✗ client ID must be 32 characters (got 3)

  Client ID: xyz
  ✗ client ID must be hexadecimal

  Client ID:
  ✗ client ID must be 32 characters (got 0)

  ✗ Giving up after 3 attempts
```

Caller returns `errAlreadyPrinted`; `Execute()` exits 1 without re-printing.

## 11 — Canonical reference doc

`docs/CLI-OUTPUT.md` — sibling of `DESIGN.md`, `keybinding.md`. Living reference;
future CLI work consults it instead of this design record.

### 11.1 Outline

```
# CLI Output Design

## Purpose
  - What CLI output is for
  - Relationship to TUI theme (not coupled; see Palette)

## Hard rules
  - No emoji / box borders / ASCII art
  - One accent colour per CTA span
  - All blocks wrapped with Padding(0, 2)
  - Sentence case, imperative hints

## Message catalogue
  - Table + one subsection per type with field spec + rendered example

## Emphasis roles
  - Accent / Strong / Plain / Muted + field-level tag matrix

## Glyph set
  - Frozen table; never introduce a new glyph without updating this doc

## Palette resolution
  - NO_COLOR > config > auto-detect
  - Fixed palette hex values
  - Theme-driven conditions
  - cli.palette escape hatch

## Spinner
  - When to use (unbounded waits > 1s)
  - TTY vs non-TTY
  - SIGINT contract

## Prompt
  - Validator contract
  - Retry count (3)
  - Abort behaviour

## Writing a new CLI command — checklist
  1. Import internal/cliout
  2. Never use fmt.Fprintln directly for user-facing output
  3. Pick a message type; propose a new type in a PR if none fits
  4. Add call-site test with cliout.Capture
  5. Update this doc if the taxonomy changes

## Relationship to other docs
  - Supersedes story 145 inline guidelines
  - TUI output is governed by DESIGN.md
```

Target: ~300 lines. Register matches `keybinding.md` — reference, not prose.

### 11.2 Maintenance rule

Added to `CLAUDE.md` § "What Agents Must NEVER Do":

> Add a new message type or glyph to `internal/cliout` without updating
> `docs/CLI-OUTPUT.md` in the same commit.

Mirrors the existing three-location keybinding rule.

## 12 — Testing strategy

### 12.1 Three levels

1. **Unit — per message type.** `message_test.go`: one test per struct, golden-string
   assertion per Status / field combination against the `Fixed` palette. No io.Writer.

2. **Integration — `Capture` helper.** `cliout.Capture(fn func(w io.Writer)) []Message`
   runs the callback against a Recorder that appends each written message without
   rendering. Lets `cmd/` tests assert on structure, not styled strings:

   ```go
   got := cliout.Capture(func(w io.Writer) { PrintForgetSuccess(w) })
   assert.Equal(t, []cliout.Message{
       cliout.Step{Status: cliout.Success, Text: "Session ended"},
       cliout.Paragraph{Text: "Tokens and client ID removed", Dim: true},
       cliout.Hint{Verb: "Run", Cmd: "spotnik auth register", Tail: "to set up again"},
   }, got)
   ```

   Resilient to palette changes.

3. **Golden file — one per command.** `cmd/root_test.go` runs each auth command
   against the Fixed palette on a non-TTY writer, compares output to
   `testdata/golden/auth_status_authenticated.txt` etc. `-update` flag refreshes.
   Catches layout regressions (padding, blank lines, order).

### 12.2 Test mode

`cliout.SetTestMode(true)` in each CLI-testing package's `TestMain` disables spinner
animation, SIGINT handlers, and pins `termenv.ColorProfile` to `Ascii`. Deterministic
regardless of host terminal.

### 12.3 Coverage target

`internal/cliout` — **90% minimum**, higher than the project 80% floor. Small focused
package; every branch matters; called from every CLI subcommand.

## 13 — Migration plan

This design ships the package + reference doc only. `cmd/root.go` keeps working
unchanged. Migration is broken into four stories inside a new feature
`NN-cli-output`.

| Story | Deliverable | Behaviour change |
|---|---|---|
| A | Create `internal/cliout` package (static types only), `docs/CLI-OUTPUT.md`, unit tests | None — no call sites yet |
| B | Migrate `cmd/root.go` call sites to `cliout.*`; remove local `cliGreen/.../cliOut/cliKV` | Equivalent output |
| C | Add `[cli] palette` to config schema; wire resolution (Section 7) | `cli.palette = "theme"` users see theme colours |
| D | Implement `Spinner` + `Prompt` dynamic types; integrate into `RunAuthFlow` + `runRegister` | Animated spinner on TTY; validated prompt |

Each story merges green independently against `make ci`.

## 14 — Open decisions (none)

All advisor gaps (contrast fallback mechanism, doc location, living ref doc, migration
scope, non-TTY stripping, prompt validation UX, story 145 supersede, SIGINT handling)
are addressed in sections 6, 7, 9, 10, 11, 13. Design is ready for implementation
planning.

## 15 — References

- `docs/spec/features/09-auth-and-profile/stories/141-cli-auth-ux-polish.md` — first
  pass at styled auth output
- `docs/spec/features/09-auth-and-profile/stories/145-cli-auth-ux-polish-2.md` —
  informal guidelines superseded by this design
- `docs/DESIGN.md` — TUI design tokens (theme interface)
- `docs/ARCHITECTURE.md` — project layout
- `github.com/muesli/termenv` — `HasDarkBackground`, `EnvNoColor`, `ColorProfile`
- `github.com/charmbracelet/lipgloss` — `AdaptiveColor`, `SetColorProfile`
