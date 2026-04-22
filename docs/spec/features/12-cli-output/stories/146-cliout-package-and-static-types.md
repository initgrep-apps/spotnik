---
title: "internal/cliout — package foundation, static message types, reference doc"
feature: 12-cli-output
status: open
---

## Background

This story creates the `internal/cliout` package with all **static** message types
(`Header`, `Step`, `KV`, `Steps`, `Hint`, `URL`, `Paragraph`), the palette resolution
logic, the `Write` / `WriteInline` entry points, the fluent `Builder`, and the
`Capture` test helper. `Spinner` and `Prompt` are stubbed out in `message.go` but
unused — Story 149 brings them alive.

Zero changes to `cmd/root.go`: the existing `cliGreen/cliOut/cliKV` helpers keep
shipping. Call sites migrate in Story 147. This keeps the story reviewable and rollback-safe.

The canonical reference doc `docs/CLI-OUTPUT.md` is written in this story — it's
what the package's `doc.go` points agents at.

**Depends on:** nothing. Design record at
`docs/superpowers/specs/2026-04-22-cli-output-design.md`.

## Design

### Package layout

```
internal/cliout/
├── doc.go         — package doc; points at docs/CLI-OUTPUT.md
├── palette.go     — Palette struct, Fixed, themePalette, resolve()
├── message.go     — Message interface + all 9 typed structs (Spinner/Prompt stubs)
├── render.go      — Write, WriteInline, rendering switch, padding (lipgloss.Padding(0,2))
├── builder.go     — Builder, Pair, PairWithCaption
├── tty.go         — isTTY(w io.Writer), checkNoColor(), pinProfile()
├── testing.go     — Capture, Recorder, SetTestMode
├── palette_test.go
├── message_test.go
├── render_test.go
├── builder_test.go
└── testing_test.go
```

### Theme interface extension — `Accent()`

The `Theme` interface (`internal/ui/theme/theme.go`) does not currently expose a
semantic "accent" token. CLI output requires one — the brand/CTA colour used on
URLs, commands, and action arrows. Two candidates considered:

- Reuse `SeekBar()` as a proxy — semantically conflates UI and CLI roles.
- Add a dedicated `Accent()` method — explicit, per-theme tunable.

Chosen: **add `Accent() lipgloss.Color` to the `Theme` interface** with a TOML
field `accent` and a fallback to `seek_bar` when the field is empty (keeps
existing user-authored themes loading without modification).

#### Changes to `internal/ui/theme/theme.go`

Add to the `Theme` interface (next to `Success`/`Warning`/`Error`):

```go
// Accent returns the CTA/brand colour used for CLI output, URLs, and commands.
Accent() lipgloss.Color
```

#### Changes to `internal/ui/theme/config_theme.go`

Add to `themeColors`:

```go
Accent string `toml:"accent"` // optional — falls back to seek_bar when empty
```

Update `validateColorFields` to **skip** `Accent` (optional field). Simplest
approach: add an allow-list inside the loop:

```go
optional := map[string]bool{"accent": true}
// ...inside the field loop:
name := ct.Field(i).Tag.Get("toml")
if cv.Field(i).String() == "" && !optional[name] {
    return fmt.Errorf(...)
}
```

Add method:

```go
// Accent returns the CLI/CTA accent colour. Falls back to SeekBar when the
// optional [colors].accent TOML field is not set.
func (t *ConfigTheme) Accent() lipgloss.Color {
    if t.c.Accent != "" {
        return lipgloss.Color(t.c.Accent)
    }
    return lipgloss.Color(t.c.SeekBar)
}
```

#### Theme TOMLs (optional — may ship in follow-up)

Setting `accent = "#..."` explicitly per TOML is recommended but not required.
Without it, Accent falls back to `seek_bar`, so all 11 built-in themes continue
to work unchanged. Add explicit accent values in a later story if colour tuning
becomes a concern.

#### Test addition — `internal/ui/theme/theme_test.go`

```go
func TestConfigTheme_Accent_fallsBackToSeekBarWhenUnset(t *testing.T) {
    // Parse a minimal theme TOML without an `accent` field.
    // Expect Accent() == SeekBar().
    th := theme.Load("black")
    assert.Equal(t, th.SeekBar(), th.Accent())
}

func TestConfigTheme_Accent_usesExplicitValueWhenSet(t *testing.T) {
    data := []byte(`id = "x"
name = "X"
[colors]
base = "#000000"
# ...all required fields...
accent = "#00ff00"
seek_bar = "#ff0000"
# ...
`)
    // ParseTheme(data) should return a Theme where Accent() == "#00ff00",
    // not the seek_bar fallback.
}
```

### `palette.go`

```go
package cliout

import (
    "sync"

    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
    "github.com/muesli/termenv"
)

// Palette holds the six role colours used by all message types.
type Palette struct {
    Accent  lipgloss.TerminalColor
    Success lipgloss.TerminalColor
    Error   lipgloss.TerminalColor
    Warning lipgloss.TerminalColor
    Muted   lipgloss.TerminalColor
    Plain   lipgloss.TerminalColor
}

// Fixed is the built-in palette — safe on any terminal, matches story 145 hex values.
var Fixed = Palette{
    Accent:  lipgloss.AdaptiveColor{Dark: "#1DB954", Light: "#1A8C41"},
    Success: lipgloss.AdaptiveColor{Dark: "#1DB954", Light: "#1A8C41"},
    Error:   lipgloss.AdaptiveColor{Dark: "#FF5555", Light: "#CC0000"},
    Warning: lipgloss.AdaptiveColor{Dark: "#F1C40F", Light: "#B8860B"},
    Muted:   lipgloss.AdaptiveColor{Dark: "#6C7083", Light: "#888888"},
    Plain:   lipgloss.AdaptiveColor{Dark: "", Light: ""}, // terminal default fg
}

// PaletteMode controls resolution strategy. Mirrors config.toml [cli] palette.
type PaletteMode int

const (
    ModeAuto PaletteMode = iota
    ModeFixed
    ModeTheme
)

// themePalette maps a theme.Theme onto a Palette.
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

// resolve picks the palette to render with given a mode, TTY check, NO_COLOR
// env flag, and (optionally) the active TUI theme. nil theme means fixed palette.
func resolve(mode PaletteMode, isTTY bool, noColor bool, t theme.Theme) Palette {
    if noColor {
        return Fixed
    }
    switch mode {
    case ModeFixed:
        return Fixed
    case ModeTheme:
        if t == nil {
            return Fixed
        }
        return themePalette(t)
    case ModeAuto:
        if isTTY && termenv.HasDarkBackground() && t != nil {
            return themePalette(t)
        }
        return Fixed
    default:
        return Fixed
    }
}

// Global palette state — callers set once at CLI startup via Use().
var (
    activeMu      sync.RWMutex
    activePalette = Fixed
)

// Use replaces the active palette. Called by cmd/root.go once config is loaded.
// Safe to call more than once; safe to skip (default is Fixed).
func Use(p Palette) {
    activeMu.Lock()
    defer activeMu.Unlock()
    activePalette = p
}

// current returns the active palette under RLock.
func current() Palette {
    activeMu.RLock()
    defer activeMu.RUnlock()
    return activePalette
}
```

### `message.go`

```go
package cliout

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
)

// Message is implemented by every CLI message type. External packages cannot
// implement this interface — the isMessage marker is unexported.
type Message interface {
    render(p Palette) string
    isMessage()
}

// Status is used by Header and Step.
type Status int

const (
    Active Status = iota
    Inactive
    StatusSuccess
    StatusFailure
    StatusWarning
    Pending
)

// statusGlyph returns the glyph for a status value.
func statusGlyph(s Status) string {
    switch s {
    case Active:
        return "◉"
    case Inactive:
        return "◎"
    case StatusSuccess:
        return "✓"
    case StatusFailure:
        return "✗"
    case StatusWarning:
        return "⚠"
    case Pending:
        return "◌"
    default:
        return "?"
    }
}

// statusColor maps a status to a palette role.
func statusColor(s Status, p Palette) lipgloss.TerminalColor {
    switch s {
    case Active:
        return p.Accent
    case Inactive:
        return p.Muted
    case StatusSuccess:
        return p.Success
    case StatusFailure:
        return p.Error
    case StatusWarning:
        return p.Warning
    case Pending:
        return p.Muted
    default:
        return p.Plain
    }
}

// Header — primary status line: "◉ Spotnik  authenticated"
type Header struct {
    Status  Status
    Subject string
    State   string
}

func (Header) isMessage() {}

func (h Header) render(p Palette) string {
    glyph := lipgloss.NewStyle().Foreground(statusColor(h.Status, p)).Bold(true).Render(statusGlyph(h.Status))
    subject := lipgloss.NewStyle().Foreground(p.Plain).Bold(true).Render(h.Subject)
    state := lipgloss.NewStyle().Foreground(p.Plain).Render(h.State)
    return glyph + "  " + subject + "  " + state
}

// Step — inline progress event: "✓ Authorization received"
type Step struct {
    Status Status
    Text   string
}

func (Step) isMessage() {}

func (s Step) render(p Palette) string {
    glyph := lipgloss.NewStyle().Foreground(statusColor(s.Status, p)).Bold(true).Render(statusGlyph(s.Status))
    text := lipgloss.NewStyle().Foreground(p.Plain).Render(s.Text)
    return glyph + " " + text
}

// KVPair is one row of a KV block.
type KVPair struct {
    Label   string
    Value   string
    Caption string // optional trailing muted context
}

// KV — aligned key/value block. Labels are right-padded so values align.
type KV struct {
    Pairs []KVPair
}

func (KV) isMessage() {}

func (kv KV) render(p Palette) string {
    if len(kv.Pairs) == 0 {
        return ""
    }
    maxLabel := 0
    for _, pair := range kv.Pairs {
        if len(pair.Label) > maxLabel {
            maxLabel = len(pair.Label)
        }
    }
    labelStyle := lipgloss.NewStyle().Foreground(p.Muted)
    valueStyle := lipgloss.NewStyle().Foreground(p.Plain)
    captionStyle := lipgloss.NewStyle().Foreground(p.Muted)

    lines := make([]string, len(kv.Pairs))
    for i, pair := range kv.Pairs {
        pad := strings.Repeat(" ", maxLabel-len(pair.Label))
        line := labelStyle.Render(pair.Label+pad) + "  " + valueStyle.Render(pair.Value)
        if pair.Caption != "" {
            line += "  " + captionStyle.Render("·  "+pair.Caption)
        }
        lines[i] = line
    }
    return strings.Join(lines, "\n")
}

// Steps — numbered instruction list. Numbers are 1..N automatic.
type Steps struct {
    Items []string
}

func (Steps) isMessage() {}

func (s Steps) render(p Palette) string {
    if len(s.Items) == 0 {
        return ""
    }
    numStyle := lipgloss.NewStyle().Foreground(p.Muted)
    textStyle := lipgloss.NewStyle().Foreground(p.Plain)
    lines := make([]string, len(s.Items))
    for i, item := range s.Items {
        num := fmt.Sprintf("%d", i+1)
        lines[i] = numStyle.Render(num) + "  " + textStyle.Render(item)
    }
    return strings.Join(lines, "\n")
}

// Hint — action suggestion: "→ Run spotnik auth login to reconnect"
type Hint struct {
    Verb string
    Cmd  string
    Tail string
}

func (Hint) isMessage() {}

func (h Hint) render(p Palette) string {
    arrow := lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Render("→")
    verb := lipgloss.NewStyle().Foreground(p.Plain).Render(h.Verb)
    cmd := lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Render(h.Cmd)
    tail := lipgloss.NewStyle().Foreground(p.Plain).Render(h.Tail)
    parts := []string{arrow}
    if h.Verb != "" {
        parts = append(parts, verb)
    }
    if h.Cmd != "" {
        parts = append(parts, cmd)
    }
    if h.Tail != "" {
        parts = append(parts, tail)
    }
    return strings.Join(parts, " ")
}

// URL — bare URL with optional precursor label.
type URL struct {
    Label string // optional muted precursor, e.g. "Visit this URL to authorize:"
    Href  string
}

func (URL) isMessage() {}

func (u URL) render(p Palette) string {
    href := lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Render(u.Href)
    if u.Label == "" {
        return href
    }
    label := lipgloss.NewStyle().Foreground(p.Muted).Render(u.Label)
    return label + "\n" + href
}

// Paragraph — free prose line. Dim=true renders in Muted, otherwise Plain.
type Paragraph struct {
    Text string
    Dim  bool
}

func (Paragraph) isMessage() {}

func (pg Paragraph) render(p Palette) string {
    if pg.Dim {
        return lipgloss.NewStyle().Foreground(p.Muted).Render(pg.Text)
    }
    return lipgloss.NewStyle().Foreground(p.Plain).Render(pg.Text)
}

// Spinner and Prompt are implemented in Story 149. They satisfy Message so
// the taxonomy is stable, but render() panics if called in this story.
type Spinner struct {
    Text string
}

func (Spinner) isMessage() {}

func (Spinner) render(_ Palette) string {
    panic("cliout.Spinner.render: not yet implemented (Story 149)")
}

type Prompt struct {
    Label       string
    Placeholder string
    Validate    func(string) error
}

func (Prompt) isMessage() {}

func (Prompt) render(_ Palette) string {
    panic("cliout.Prompt.render: not yet implemented (Story 149)")
}
```

### `render.go`

```go
package cliout

import (
    "fmt"
    "io"
    "strings"

    "github.com/charmbracelet/lipgloss"
)

// wrap applies the standard 2-char left indent to every line of a block.
var wrap = lipgloss.NewStyle().Padding(0, 2)

// Write renders each message with a leading blank line and standard padding.
// Safe for any io.Writer. If a Recorder is active (see testing.go), writes
// go to the recorder instead of the writer.
func Write(w io.Writer, msgs ...Message) {
    if rec := activeRecorder(); rec != nil {
        rec.append(msgs...)
        return
    }
    if len(msgs) == 0 {
        return
    }
    block := renderAll(current(), msgs)
    _, _ = fmt.Fprintln(w, "\n"+wrap.Render(block))
}

// WriteInline renders with no leading blank line — for compact step-by-step progress.
func WriteInline(w io.Writer, msgs ...Message) {
    if rec := activeRecorder(); rec != nil {
        rec.append(msgs...)
        return
    }
    if len(msgs) == 0 {
        return
    }
    block := renderAll(current(), msgs)
    _, _ = fmt.Fprintln(w, wrap.Render(block))
}

func renderAll(p Palette, msgs []Message) string {
    parts := make([]string, 0, len(msgs))
    for _, m := range msgs {
        s := m.render(p)
        if s != "" {
            parts = append(parts, s)
        }
    }
    return strings.Join(parts, "\n")
}
```

### `builder.go`

```go
package cliout

import (
    "io"
)

// Builder provides a fluent façade over []Message. All chain methods append to
// the same slice; WriteTo renders and flushes.
type Builder struct {
    msgs []Message
}

// New returns an empty Builder.
func New() *Builder { return &Builder{} }

func (b *Builder) Header(s Status, subject, state string) *Builder {
    b.msgs = append(b.msgs, Header{Status: s, Subject: subject, State: state})
    return b
}

func (b *Builder) Step(s Status, text string) *Builder {
    b.msgs = append(b.msgs, Step{Status: s, Text: text})
    return b
}

func (b *Builder) KV(pairs ...KVPair) *Builder {
    b.msgs = append(b.msgs, KV{Pairs: pairs})
    return b
}

func (b *Builder) Steps(items ...string) *Builder {
    b.msgs = append(b.msgs, Steps{Items: items})
    return b
}

func (b *Builder) Hint(verb, cmd, tail string) *Builder {
    b.msgs = append(b.msgs, Hint{Verb: verb, Cmd: cmd, Tail: tail})
    return b
}

func (b *Builder) URL(label, href string) *Builder {
    b.msgs = append(b.msgs, URL{Label: label, Href: href})
    return b
}

func (b *Builder) Paragraph(text string) *Builder {
    b.msgs = append(b.msgs, Paragraph{Text: text})
    return b
}

// Dim is shorthand for Paragraph{Dim: true}.
func (b *Builder) Dim(text string) *Builder {
    b.msgs = append(b.msgs, Paragraph{Text: text, Dim: true})
    return b
}

// Messages returns the accumulated slice for test assertions.
func (b *Builder) Messages() []Message { return b.msgs }

// WriteTo renders and flushes to w.
func (b *Builder) WriteTo(w io.Writer) (int64, error) {
    Write(w, b.msgs...)
    return 0, nil
}

// Pair is a shorthand constructor for KVPair.
func Pair(label, value string) KVPair { return KVPair{Label: label, Value: value} }

// PairWithCaption adds a trailing muted caption to a KV row.
func PairWithCaption(label, value, caption string) KVPair {
    return KVPair{Label: label, Value: value, Caption: caption}
}
```

### `tty.go`

```go
package cliout

import (
    "io"
    "os"
    "sync"

    "github.com/charmbracelet/lipgloss"
    "github.com/muesli/termenv"
    "golang.org/x/term"
)

var (
    profileOnce sync.Once
    testMode    bool
    testModeMu  sync.RWMutex
)

// isTTY returns whether w is *os.File pointing at a terminal. Returns false
// for any non-file writer (pipes, buffers, io.Discard).
func isTTY(w io.Writer) bool {
    f, ok := w.(*os.File)
    if !ok {
        return false
    }
    return term.IsTerminal(int(f.Fd()))
}

// checkNoColor honours the NO_COLOR env var (any non-empty value disables colour).
func checkNoColor() bool {
    return os.Getenv("NO_COLOR") != ""
}

// pinASCII forces lipgloss to render without ANSI escapes. Called once when
// the first Write/StartSpinner/Ask resolves to a non-TTY / NO_COLOR path.
func pinASCII() {
    profileOnce.Do(func() {
        lipgloss.SetColorProfile(termenv.Ascii)
    })
}

// SetTestMode enables or disables test mode. In test mode, pinASCII is called
// immediately and spinner animation is disabled (Story 149 uses this flag).
// Tests call this in TestMain for deterministic output.
func SetTestMode(enabled bool) {
    testModeMu.Lock()
    defer testModeMu.Unlock()
    testMode = enabled
    if enabled {
        pinASCII()
    }
}

func inTestMode() bool {
    testModeMu.RLock()
    defer testModeMu.RUnlock()
    return testMode
}
```

### `testing.go`

```go
package cliout

import (
    "bytes"
    "io"
    "sync"
)

// Recorder captures messages written via Write / WriteInline without rendering.
// Used by Capture and by test assertions on structure rather than styled strings.
type Recorder struct {
    mu   sync.Mutex
    msgs []Message
}

// append records messages.
func (r *Recorder) append(msgs ...Message) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.msgs = append(r.msgs, msgs...)
}

// Messages returns a copy of the recorded slice.
func (r *Recorder) Messages() []Message {
    r.mu.Lock()
    defer r.mu.Unlock()
    out := make([]Message, len(r.msgs))
    copy(out, r.msgs)
    return out
}

var (
    recorderMu sync.RWMutex
    recorder   *Recorder
)

// activeRecorder returns the current package-level recorder or nil.
func activeRecorder() *Recorder {
    recorderMu.RLock()
    defer recorderMu.RUnlock()
    return recorder
}

// Capture runs fn with a package-level Recorder installed.
// All Write/WriteInline calls during fn are captured and returned.
// Spinner/Prompt dynamic types (Story 149) also append to the recorder
// when active, but their input/animation side effects are skipped —
// tests assert on structure, not on stdin consumption or TTY bytes.
// Not thread-safe — run tests sequentially.
func Capture(fn func(w io.Writer)) []Message {
    r := &Recorder{}
    recorderMu.Lock()
    prev := recorder
    recorder = r
    recorderMu.Unlock()

    defer func() {
        recorderMu.Lock()
        recorder = prev
        recorderMu.Unlock()
    }()

    fn(&bytes.Buffer{}) // writer is discarded; recorder captures everything
    return r.Messages()
}
```

### `doc.go`

```go
// Package cliout renders styled CLI output for the spotnik command-line interface.
//
// The package defines a small taxonomy of message types (Header, Step, KV, Steps,
// Hint, URL, Paragraph, Spinner, Prompt). Callers build []Message values or use
// the fluent Builder and call Write / WriteInline. Rendering is palette-aware and
// honours NO_COLOR and TTY detection automatically.
//
// Before adding a new message type or glyph, read docs/CLI-OUTPUT.md — it is the
// canonical reference for the project's CLI output conventions.
package cliout
```

### `docs/CLI-OUTPUT.md`

Target ~300 lines, same register as `keybinding.md`. Sections:

1. Purpose + relationship to TUI theme
2. Hard rules (no emoji / box borders / ASCII art; `Padding(0,2)`; one accent per CTA)
3. Message catalogue — one subsection per type with field spec + rendered example
4. Emphasis roles (Accent / Strong / Plain / Muted) + field-level tag matrix
5. Glyph set (frozen) — table: `◉ ◎ ✓ ✗ ⚠ ◌ →`
6. Palette resolution (`NO_COLOR` > config > auto-detect); `cli.palette` escape hatch
7. Spinner contract (when to use, TTY vs non-TTY, SIGINT)
8. Prompt contract (validator, 3 retries, abort)
9. Writing a new CLI command — 5-step checklist
10. Relationship to other docs (supersedes story 145 inline guidelines; TUI out of scope)

Full content: copy sections 5, 6, 7, 9, 10 from `docs/superpowers/specs/2026-04-22-cli-output-design.md`
verbatim (they're written as reference material). Section 1 ("Purpose") is new, one paragraph.

### `CLAUDE.md` updates

Add to **Reading Order** section (after the paragraph about DESIGN.md/ARCHITECTURE.md):

```markdown
When writing or modifying CLI output, consult `docs/CLI-OUTPUT.md` — the canonical
reference for message types, glyphs, palette, and interactive prompts.
```

Add to **What Agents Must NEVER Do** list (bottom of that section):

```markdown
16. Add a new message type or glyph to `internal/cliout` without updating
    `docs/CLI-OUTPUT.md` in the same commit.
```

Renumber downstream items if any new additions arrived since this story was
drafted — check current state before committing.

### Tests

#### `palette_test.go`

```go
func TestResolve_NoColor_returnsFixed(t *testing.T) {
    // Any mode + noColor=true must return Fixed.
    for _, m := range []PaletteMode{ModeAuto, ModeFixed, ModeTheme} {
        got := resolve(m, true, true, theme.NewBlack())
        assert.Equal(t, Fixed, got, "mode %v with NO_COLOR", m)
    }
}

func TestResolve_ModeFixed_alwaysFixed(t *testing.T) {
    got := resolve(ModeFixed, true, false, theme.NewBlack())
    assert.Equal(t, Fixed, got)
}

func TestResolve_ModeTheme_withTheme_returnsThemeTokens(t *testing.T) {
    th := theme.NewBlack()
    got := resolve(ModeTheme, true, false, th)
    assert.Equal(t, th.Accent(), got.Accent)
    assert.Equal(t, th.TextMuted(), got.Muted)
}

func TestResolve_ModeTheme_nilTheme_fallsBackToFixed(t *testing.T) {
    got := resolve(ModeTheme, true, false, nil)
    assert.Equal(t, Fixed, got)
}

func TestResolve_ModeAuto_nonTTY_returnsFixed(t *testing.T) {
    got := resolve(ModeAuto, false, false, theme.NewBlack())
    assert.Equal(t, Fixed, got)
}

// HasDarkBackground is not mockable in tests; TestResolve_ModeAuto_tty_returnsTheme
// is omitted. Auto-detect behaviour is covered by integration in Story 148.

func TestUse_replacesActive(t *testing.T) {
    prev := current()
    t.Cleanup(func() { Use(prev) })
    custom := Fixed
    custom.Accent = lipgloss.AdaptiveColor{Dark: "#ff00ff"}
    Use(custom)
    assert.Equal(t, custom, current())
}
```

#### `message_test.go`

```go
func TestStatusGlyph(t *testing.T) {
    cases := []struct {
        s    Status
        want string
    }{
        {Active, "◉"},
        {Inactive, "◎"},
        {StatusSuccess, "✓"},
        {StatusFailure, "✗"},
        {StatusWarning, "⚠"},
        {Pending, "◌"},
    }
    for _, c := range cases {
        assert.Equal(t, c.want, statusGlyph(c.s))
    }
}

func TestHeader_renderActive(t *testing.T) {
    h := Header{Status: Active, Subject: "Spotnik", State: "authenticated"}
    out := h.render(Fixed)
    assert.Contains(t, out, "◉")
    assert.Contains(t, out, "Spotnik")
    assert.Contains(t, out, "authenticated")
}

func TestKV_alignsLabels(t *testing.T) {
    kv := KV{Pairs: []KVPair{
        {Label: "Client ID", Value: "present"},
        {Label: "Expires", Value: "Wed, 23 Apr"},
    }}
    out := kv.render(Fixed)
    // Expect both labels padded to 9 chars (len("Client ID")).
    lines := strings.Split(stripAnsi(out), "\n")
    require.Len(t, lines, 2)
    assert.True(t, strings.HasPrefix(lines[0], "Client ID  "))
    assert.True(t, strings.HasPrefix(lines[1], "Expires    "))
}

func TestSteps_numbersFrom1(t *testing.T) {
    s := Steps{Items: []string{"first", "second", "third"}}
    out := stripAnsi(s.render(Fixed))
    lines := strings.Split(out, "\n")
    require.Len(t, lines, 3)
    assert.True(t, strings.HasPrefix(lines[0], "1"))
    assert.True(t, strings.HasPrefix(lines[1], "2"))
    assert.True(t, strings.HasPrefix(lines[2], "3"))
}

func TestHint_renderWithAllFields(t *testing.T) {
    h := Hint{Verb: "Run", Cmd: "spotnik auth login", Tail: "to reconnect"}
    out := stripAnsi(h.render(Fixed))
    assert.Equal(t, "→ Run spotnik auth login to reconnect", out)
}

func TestHint_omitsEmptyFields(t *testing.T) {
    h := Hint{Cmd: "spotnik auth register"}
    out := stripAnsi(h.render(Fixed))
    assert.Equal(t, "→ spotnik auth register", out)
}

func TestURL_noLabel_rendersHrefOnly(t *testing.T) {
    u := URL{Href: "https://example.com"}
    out := stripAnsi(u.render(Fixed))
    assert.Equal(t, "https://example.com", out)
}

func TestURL_withLabel_rendersTwoLines(t *testing.T) {
    u := URL{Label: "Visit:", Href: "https://example.com"}
    out := stripAnsi(u.render(Fixed))
    assert.Equal(t, "Visit:\nhttps://example.com", out)
}

func TestParagraph_dimAndPlain(t *testing.T) {
    p1 := Paragraph{Text: "plain line"}
    p2 := Paragraph{Text: "dim line", Dim: true}
    assert.Contains(t, stripAnsi(p1.render(Fixed)), "plain line")
    assert.Contains(t, stripAnsi(p2.render(Fixed)), "dim line")
}

func TestSpinner_renderPanicsUntilStory149(t *testing.T) {
    assert.Panics(t, func() { Spinner{Text: "x"}.render(Fixed) })
}

func TestPrompt_renderPanicsUntilStory149(t *testing.T) {
    assert.Panics(t, func() { Prompt{Label: "x"}.render(Fixed) })
}
```

Add a tiny helper to strip ANSI for assertion stability:

```go
// stripAnsi removes ANSI escape sequences so tests can assert on visible text.
func stripAnsi(s string) string {
    // Regex: ESC [ ... m  (CSI SGR sequences)
    re := regexp.MustCompile("\x1b\\[[0-9;]*m")
    return re.ReplaceAllString(s, "")
}
```

Put `stripAnsi` in `message_test.go` (unexported; test-only).

#### `render_test.go`

```go
func TestWrite_emptySlice_writesNothing(t *testing.T) {
    var buf bytes.Buffer
    Write(&buf)
    assert.Empty(t, buf.String())
}

func TestWrite_addsLeadingBlankAndIndent(t *testing.T) {
    var buf bytes.Buffer
    Write(&buf, Paragraph{Text: "hello"})
    out := buf.String()
    assert.True(t, strings.HasPrefix(out, "\n"), "expected leading blank line")
    assert.Contains(t, out, "  hello") // 2-char left indent
}

func TestWriteInline_noLeadingBlank(t *testing.T) {
    var buf bytes.Buffer
    WriteInline(&buf, Paragraph{Text: "hello"})
    out := buf.String()
    assert.False(t, strings.HasPrefix(out, "\n"))
    assert.Contains(t, out, "  hello")
}

func TestWrite_joinsMessagesWithNewline(t *testing.T) {
    var buf bytes.Buffer
    Write(&buf,
        Header{Status: Active, Subject: "Spotnik", State: "authenticated"},
        Paragraph{Text: "body"},
    )
    out := buf.String()
    assert.Contains(t, out, "Spotnik")
    assert.Contains(t, out, "body")
    assert.Contains(t, out, "\n") // header and body on separate lines
}
```

#### `builder_test.go`

```go
func TestBuilder_Messages_accumulatesInOrder(t *testing.T) {
    b := New().
        Header(Active, "Spotnik", "authenticated").
        KV(Pair("Client ID", "present")).
        Hint("Run", "spotnik auth login", "to reconnect")

    got := b.Messages()
    require.Len(t, got, 3)
    assert.IsType(t, Header{}, got[0])
    assert.IsType(t, KV{}, got[1])
    assert.IsType(t, Hint{}, got[2])

    hdr := got[0].(Header)
    assert.Equal(t, Active, hdr.Status)
    assert.Equal(t, "Spotnik", hdr.Subject)
}

func TestBuilder_WriteTo_rendersAndFlushes(t *testing.T) {
    var buf bytes.Buffer
    _, err := New().
        Header(Active, "Spotnik", "authenticated").
        WriteTo(&buf)
    require.NoError(t, err)
    assert.Contains(t, buf.String(), "Spotnik")
    assert.Contains(t, buf.String(), "authenticated")
}

func TestPair_helper(t *testing.T) {
    p := Pair("Label", "Value")
    assert.Equal(t, KVPair{Label: "Label", Value: "Value"}, p)
}

func TestPairWithCaption_helper(t *testing.T) {
    p := PairWithCaption("Label", "Value", "caption")
    assert.Equal(t, KVPair{Label: "Label", Value: "Value", Caption: "caption"}, p)
}
```

#### `testing_test.go`

```go
func TestCapture_recordsMessagesWithoutRendering(t *testing.T) {
    got := Capture(func(w io.Writer) {
        Write(w,
            Header{Status: Active, Subject: "Spotnik", State: "authenticated"},
            Paragraph{Text: "body"},
        )
    })

    require.Len(t, got, 2)
    assert.Equal(t,
        Header{Status: Active, Subject: "Spotnik", State: "authenticated"},
        got[0])
    assert.Equal(t, Paragraph{Text: "body"}, got[1])
}

func TestCapture_nested_restoresPreviousRecorder(t *testing.T) {
    outer := Capture(func(w io.Writer) {
        Write(w, Paragraph{Text: "outer before"})
        inner := Capture(func(w2 io.Writer) {
            Write(w2, Paragraph{Text: "inner"})
        })
        assert.Len(t, inner, 1)
        Write(w, Paragraph{Text: "outer after"})
    })

    // Outer recorder must only contain outer's writes; inner was captured separately.
    require.Len(t, outer, 2)
    assert.Equal(t, "outer before", outer[0].(Paragraph).Text)
    assert.Equal(t, "outer after", outer[1].(Paragraph).Text)
}

func TestSetTestMode_pinsASCII(t *testing.T) {
    SetTestMode(true)
    t.Cleanup(func() { SetTestMode(false) })
    // lipgloss profile is global; just verify SetTestMode(true) doesn't panic
    // and inTestMode reports true.
    assert.True(t, inTestMode())
}
```

### `go.mod` additions

```
golang.org/x/term v0.x.x   // new direct dep for term.IsTerminal
```

`lipgloss`, `termenv`, `internal/ui/theme` are already in go.mod.

## Acceptance Criteria

- [ ] `internal/cliout/` directory exists with 9 files (`doc.go`, `palette.go`,
      `message.go`, `render.go`, `builder.go`, `tty.go`, `testing.go`, plus 5 `_test.go`)
- [ ] All 9 `Message` types compile; `Spinner` and `Prompt` panic from `render()` with
      a "not yet implemented (Story 149)" message
- [ ] `cliout.Fixed` has exactly the six hex values listed in §Fixed of the design spec
- [ ] `cliout.Use(p)` replaces the active palette; `current()` returns it under lock
- [ ] `cliout.Write(w, msgs...)` writes leading blank + `Padding(0, 2)` indent; empty
      slice writes nothing
- [ ] `cliout.WriteInline(w, msgs...)` writes `Padding(0, 2)` indent with no leading blank
- [ ] `cliout.Capture(fn)` captures `[]Message` without rendering; nested captures
      correctly scope to each call
- [ ] `cliout.SetTestMode(true)` pins `termenv.ColorProfile` to `Ascii`
- [ ] `cliout.New().Header(...).KV(...).Hint(...).WriteTo(w)` renders equivalent output
      to `cliout.Write(w, Header{...}, KV{...}, Hint{...})`
- [ ] `docs/CLI-OUTPUT.md` exists under `docs/` with all 10 outlined sections
- [ ] `theme.Theme` interface has `Accent() lipgloss.Color`; `ConfigTheme`
      implements it with seek_bar fallback; existing 11 theme TOMLs still
      parse without modification
- [ ] `CLAUDE.md` Reading Order references `docs/CLI-OUTPUT.md`
- [ ] `CLAUDE.md` "What Agents Must NEVER Do" has the message-type/glyph update rule
- [ ] `internal/cliout` package coverage ≥ 90% (`go test -cover ./internal/cliout/`)
- [ ] `cmd/root.go` is unchanged — no call-site migration in this story
- [ ] `make ci` passes

## Tasks

- [ ] Extend `theme.Theme` with `Accent() lipgloss.Color`; add `Accent` field
      (TOML key `accent`) to `themeColors`; add `Accent()` method to
      `ConfigTheme` with seek_bar fallback; loosen `validateColorFields` to
      treat `accent` as optional; write the two new theme tests
      - test: `go test ./internal/ui/theme/... -run TestConfigTheme_Accent -v` → PASS;
        `go test ./internal/ui/theme/...` → all prior tests still PASS

- [ ] Create `internal/cliout/doc.go` with the package doc comment
      - test: `go build ./internal/cliout/...` → clean

- [ ] Create `internal/cliout/palette.go` with `Palette`, `PaletteMode`, `Fixed`,
      `themePalette`, `resolve`, `Use`, `current`
      - test: `go build ./internal/cliout/...` → clean

- [ ] Write `palette_test.go` with the six `resolve` tests + `Use` test
      - test: `go test ./internal/cliout/... -run TestResolve -v` → PASS;
        `go test ./internal/cliout/... -run TestUse -v` → PASS

- [ ] Create `internal/cliout/message.go` with `Message` interface, `Status` enum,
      `statusGlyph`, `statusColor`, and the 9 message structs (`Spinner`/`Prompt`
      stub their `render` with `panic("not yet implemented (Story 149)")`)
      - test: `go build ./internal/cliout/...` → clean

- [ ] Write `message_test.go` with glyph/color tests, per-type render tests, and
      the stripAnsi helper; include tests asserting `Spinner.render`/`Prompt.render`
      panic
      - test: `go test ./internal/cliout/... -run TestHeader -v` → PASS;
        all `TestKV`/`TestSteps`/`TestHint`/`TestURL`/`TestParagraph`/
        `TestStatusGlyph`/`Test*_renderPanicsUntilStory149` → PASS

- [ ] Create `internal/cliout/tty.go` with `isTTY`, `checkNoColor`, `pinASCII`,
      `SetTestMode`, `inTestMode`
      - test: `go build ./internal/cliout/...` → clean

- [ ] Create `internal/cliout/render.go` with `wrap`, `Write`, `WriteInline`,
      `renderAll`. `Write`/`WriteInline` delegate to `activeRecorder` (from
      `testing.go`) when set
      - test: `go build ./internal/cliout/...` → clean

- [ ] Create `internal/cliout/testing.go` with `Recorder`, `activeRecorder`,
      `Capture`
      - test: `go build ./internal/cliout/...` → clean

- [ ] Write `render_test.go` with the four `TestWrite*` tests
      - test: `go test ./internal/cliout/... -run TestWrite -v` → PASS

- [ ] Create `internal/cliout/builder.go` with `Builder`, `New`, chain methods,
      `Messages`, `WriteTo`, `Pair`, `PairWithCaption`
      - test: `go build ./internal/cliout/...` → clean

- [ ] Write `builder_test.go` with the four builder tests
      - test: `go test ./internal/cliout/... -run TestBuilder -v` → PASS;
        `go test ./internal/cliout/... -run TestPair -v` → PASS

- [ ] Write `testing_test.go` with the three capture / test-mode tests
      - test: `go test ./internal/cliout/... -run TestCapture -v` → PASS;
        `go test ./internal/cliout/... -run TestSetTestMode -v` → PASS

- [ ] Add `golang.org/x/term` to go.mod via `go get golang.org/x/term`; run
      `go mod tidy`
      - test: `go build ./...` → clean

- [ ] Create `docs/CLI-OUTPUT.md` — start by copying sections 5, 6, 7, 9, 10 of
      `docs/superpowers/specs/2026-04-22-cli-output-design.md`; add a one-paragraph
      "Purpose" section 1; reformat as a reference doc (imperative mood, no
      decision-record framing); cross-reference DESIGN.md and keybinding.md
      - test: `wc -l docs/CLI-OUTPUT.md` ~ 200–350 lines; all 10 outlined sections
        present (visual check)

- [ ] Update `CLAUDE.md`: add "When writing or modifying CLI output, consult
      `docs/CLI-OUTPUT.md`" under Reading Order; add numbered item to "What Agents
      Must NEVER Do" requiring doc updates alongside new types/glyphs
      - test: `grep -n 'CLI-OUTPUT.md' CLAUDE.md` → at least two matches

- [ ] Run full coverage and confirm ≥90% for `internal/cliout`
      - test: `go test -cover ./internal/cliout/...` → coverage ≥ 90.0%

- [ ] Run `make ci` → PASS
