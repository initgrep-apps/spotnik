# CLI Output Reference

This document is the canonical reference for all CLI output in spotnik. Read it before
writing or modifying any CLI subcommand output.

**See also:** `design.md` (TUI theme tokens) · `../../README.md#keybindings` (keybindings)

---

## 1 — Purpose

`internal/cliout` renders styled CLI output for all spotnik subcommands. It provides a
typed message taxonomy, palette-aware rendering, TTY detection, and a test helper for
structural assertions. Every user-facing CLI string must go through this package — no
direct `fmt.Fprintln` for user-facing output.

TUI panes and pane borders are governed by `design.md` — not this document.
CLI output and TUI output share theme tokens when `cli.palette = "theme"` is set, but
they are rendered by separate code paths.

---

## 2 — Hard Rules

1. **No emoji, no box borders, no ASCII art.** Use only glyphs from the frozen glyph set (§5).
2. **One Accent span per call to action.** URL or command, never both wrapped in the same block.
3. **All blocks use `Padding(0, 2)`.** The 2-char left indent is applied by `Write` / `WriteInline` automatically. Never add extra padding at the call site.
4. **Sentence case, imperative hints.** "Run `spotnik auth login` to reconnect" — not "You should run…".
5. **Never accent an informational value.** A client ID string in a KV row is Plain (it is a fact, not an action).
6. **Role colours on glyph + at most one status word.** Body text alongside stays Plain or Muted.
7. **Strong is bold, not bright.** Contrast via weight, not additional colour.

---

## 3 — Message Catalogue

Nine typed message structs. Import `github.com/initgrep-apps/spotnik/internal/cliout`.

| # | Type | Purpose | Sample |
|---|---|---|---|
| 1 | `Header` | Primary status line | `◉ Spotnik  authenticated` |
| 2 | `Step` | Inline progress event | `✓ Authorization received` |
| 3 | `KV` | Aligned key/value facts | `Client ID  present` |
| 4 | `Steps` | Numbered instruction list | `1  Go to developer.spotify.com/dashboard` |
| 5 | `Hint` | Action suggestion | `→ Run spotnik auth login to reconnect` |
| 6 | `URL` | Bare URL on its own line | `https://accounts.spotify.com/authorize?…` |
| 7 | `Paragraph` | Free prose line | `Tokens and client ID removed` |
| 8 | `Spinner` | Long unbounded wait (>1 s) | `⣾ Waiting for authorization` |
| 9 | `Prompt` | Interactive input with validation | `Client ID: _` |

### 3.1 Header

```go
type Header struct {
    Status  Status // glyph + colour role
    Subject string // bold headline
    State   string // plain descriptor
}
```

Rendered as `<glyph>  <Subject>  <State>`. Always the first message in a block.

### 3.2 Step

```go
type Step struct {
    Status Status
    Text   string
}
```

Rendered as `<glyph> <Text>`. Use for each completed or failed sub-step.

### 3.3 KV

```go
type KV struct { Pairs []KVPair }
type KVPair struct {
    Label   string // muted, right-padded
    Value   string // plain
    Caption string // optional muted trailing context
}
```

Labels are padded to the width of the longest label so values column-align. Use
`cliout.Pair(label, value)` and `cliout.PairWithCaption(label, value, caption)` as
shorthand constructors.

### 3.4 Steps

```go
type Steps struct { Items []string }
```

Numbered `1..N` automatically. Numbers are Muted; text is Plain. Use for ordered
instructions (e.g. OAuth setup steps shown during `auth register`).

### 3.5 Hint

```go
type Hint struct {
    Verb string // plain action word, e.g. "Run"
    Cmd  string // accent-coloured command
    Tail string // plain trailing context
}
```

Rendered as `→ <Verb> <Cmd> <Tail>`. Arrow and Cmd are Accent. Empty fields are omitted.

### 3.6 URL

```go
type URL struct {
    Label string // optional muted precursor line
    Href  string // accent-coloured URL
}
```

When `Label` is empty, only the Href line is rendered. When set, rendered as two lines:
muted label above, accent URL below.

### 3.7 Paragraph

```go
type Paragraph struct {
    Text string
    Dim  bool // Muted when true; Plain when false
}
```

Use `Dim: true` for secondary context (reasons, "already done" notes). Use the `Builder`
helper `.Dim(text)` as shorthand.

### 3.8 Spinner

```go
type Spinner struct { Text string }
```

Animated on TTY: goroutine redraws `⣾ Text` on the same line every 100 ms using `\r`.
On non-TTY / pipe / `NO_COLOR`: writes `◌ Text\n` once. Use only for unbounded waits
longer than ~1 s. Implemented in Story 149; `render()` panics if called before then.

### 3.9 Prompt

```go
type Prompt struct {
    Label       string
    Placeholder string
    Validate    func(string) error
}
```

Renders a styled `Label: ` prompt and reads from stdin. Passes trimmed input to
`Validate`. Retries up to 3 times on validation failure. Returns `ErrAborted` on
Ctrl+C, EOF, or 3 consecutive failures. Implemented in Story 149.

---

## 4 — Emphasis Roles

Four emphasis levels. Baked into message types at field level — call sites never pass a
colour directly.

| Level | Visual | Purpose |
|---|---|---|
| **Accent** | green bold | URL, command, redirect URI, action arrow |
| **Strong** | default fg, bold | Headline subject (`Header.Subject`) |
| **Plain** | default fg | Body text, KV values, hint tails |
| **Muted** | dim | Labels, captions, prompts, inactive state |

### 4.1 Field-level tag matrix

| Type.Field | Role |
|---|---|
| `Header.Status glyph` | Status colour (Accent / Error / Warning / Muted) |
| `Header.Subject` | Strong |
| `Header.State` | Plain |
| `Step.Status glyph` | Status colour |
| `Step.Text` | Plain |
| `KV.Pairs[].Label` | Muted |
| `KV.Pairs[].Value` | Plain |
| `KV.Pairs[].Caption` | Muted |
| `Steps.Items[].Number` | Muted |
| `Steps.Items[].Text` | Plain |
| `Hint.Arrow (→)` | Accent |
| `Hint.Verb` | Plain |
| `Hint.Cmd` | Accent |
| `Hint.Tail` | Plain |
| `URL.Label` | Muted |
| `URL.Href` | Accent |
| `Paragraph.Text` (Dim=false) | Plain |
| `Paragraph.Text` (Dim=true) | Muted |
| `Spinner.Text` | Muted (animated frame is Accent) |
| `Prompt.Label` | Muted |
| `Prompt.Value` (typed) | Plain |
| Validation-failure line | Error glyph + Plain text |

---

## 5 — Glyph Set

**Frozen.** Never introduce a new glyph without updating this table in the same commit.

| Glyph | Meaning | Used by |
|---|---|---|
| `◉` | Active | `Header{Active}`, `Step{Active}` |
| `◎` | Inactive | `Header{Inactive}`, `Step{Inactive}` |
| `✓` | Success | `Header{StatusSuccess}`, `Step{StatusSuccess}` |
| `✗` | Failure | `Header{StatusFailure}`, `Step{StatusFailure}` |
| `◬` | Warning | `Header{StatusWarning}`, `Step{StatusWarning}` |
| `◌` | Pending | `Header{Pending}`, `Step{Pending}`, Spinner non-TTY |
| `→` | Action | `Hint.Arrow` |
| `⣾⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` | Spinning | `Spinner` TTY animation frames |
| `♥` | Liked track | Liked track heart prefix (NowPlaying, LikedSongs, Queue, TopTracks, RecentlyPlayed, Playlists track view, Albums track view, Search) |

---

## 6 — Palette Resolution

At every CLI invocation, palette resolution proceeds in this order:

1. **`NO_COLOR` env var** — if set to any non-empty value, all colour is disabled.
   Plain text only, no ANSI escapes.
2. **`cli.palette` config** (set in `~/.config/spotnik/config.toml`):
   - `"fixed"` — always the Fixed palette (§6.1)
   - `"theme"` — always the active TUI theme's colour tokens (§6.2)
   - `"auto"` (default) — auto-detect per step 3
3. **Auto-detect** (only when `cli.palette = "auto"`):
   - Output is a TTY **AND** `termenv.HasDarkBackground() == true` → **theme palette**
   - Otherwise → **fixed palette**

ANSI colour stripping is applied on the first `Write` / `StartSpinner` / `Ask` call
(lazy), not in `init()`. First-call resolution checks `isTTY(w)` and `NO_COLOR`;
if stripping is needed, `lipgloss.SetColorProfile(termenv.Ascii)` is called once
(idempotent via `sync.Once`).

### 6.1 Fixed palette

```go
var Fixed = Palette{
    Accent:  lipgloss.AdaptiveColor{Dark: "#1DB954", Light: "#1A8C41"},
    Success: lipgloss.AdaptiveColor{Dark: "#1DB954", Light: "#1A8C41"},
    Error:   lipgloss.AdaptiveColor{Dark: "#FF5555", Light: "#CC0000"},
    Warning: lipgloss.AdaptiveColor{Dark: "#F1C40F", Light: "#B8860B"},
    Muted:   lipgloss.AdaptiveColor{Dark: "#6C7083", Light: "#888888"},
    Plain:   lipgloss.AdaptiveColor{Dark: "",        Light: ""},
}
```

Matches story 145 hex values. Safe on any terminal.

### 6.2 Theme palette

When `cli.palette = "theme"` or auto-detect resolves to theme:

```go
Palette{
    Accent:  theme.Accent(),      // optional TOML field; falls back to SeekBar()
    Success: theme.Success(),
    Error:   theme.Error(),
    Warning: theme.Warning(),
    Muted:   theme.TextMuted(),
    Plain:   theme.TextPrimary(),
}
```

### 6.3 Setting the palette

Call `cliout.Use(palette)` once at CLI startup after config is loaded. The default is
`Fixed` — callers that never call `Use` get a safe default.

---

## 7 — Spinner Contract

`Spinner` is implemented in Story 149. Contracts documented here for reference.

- **When to use:** unbounded waits longer than ~1 s (e.g. OAuth callback wait).
  Short steps (<1 s) use `Step{Active, "…"}` then replace with `Step{Success, "…"}`.
- **TTY:** goroutine redraws `⣾ Text` every 100 ms on the same line via `\r`.
  Frame sequence: `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`. Text is Muted; frame char is Accent.
- **Non-TTY / pipe:** writes `◌ Text\n` once and returns a no-op handle.
- **Resolve:** `Done(text)` → clears line, writes `✓ text`. `Fail(text)` → `✗ text`. `Stop()` → silent cancel.
- **SIGINT:** a package-level signal handler is installed lazily on first `StartSpinner`
  call. It cancels all active spinners, restores the cursor, and exits 130.
- **Cursor:** hidden on start (`\x1b[?25l`), restored on resolve (`\x1b[?25h`). TTY only.
- **Test mode:** `cliout.SetTestMode(true)` disables animation; each event writes one line.

---

## 8 — Prompt Contract

`Prompt` is implemented in Story 149. Contracts documented here for reference.

- Renders `Label: ` in Muted, then cursor. Reads from stdin via `bufio.Scanner`.
- Trimmed input → `Validate`. `nil` return → accepted. Non-nil → print `✗ <err>`, re-prompt.
- **Retries:** up to 3 validation failures. After the third, prints `✗ Giving up after 3 attempts`
  and returns `ErrAborted`.
- **Ctrl+C / EOF:** returns `ErrAborted`. Caller should treat as `errAlreadyPrinted`.
- **Example validator** (32-char hex client ID):

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
    Label:    "Client ID",
    Validate: validateClientID,
})
```

---

## 9 — Writing a New CLI Command

Follow this checklist when adding a new `spotnik` subcommand or output block:

1. **Import `internal/cliout`** — never use `fmt.Fprintln` for user-facing output.
2. **Pick a message type from §3** — if no existing type fits, propose a new type in a PR and update this doc in the same commit.
3. **Use the Builder or direct structs:**
   ```go
   cliout.New().
       Header(cliout.Active, "Spotnik", "authenticated").
       KV(cliout.Pair("Client ID", "present")).
       WriteTo(w)
   ```
4. **Add a `cliout.Capture` test** for structural assertions:
   ```go
   got := cliout.Capture(func(w io.Writer) { myCommand.printStatus(w) })
   require.Len(t, got, 2)
   assert.Equal(t, cliout.Header{Status: cliout.Active, Subject: "Spotnik"}, got[0])
   ```
5. **Update this doc** if you add a message type, a glyph, or change a rendering rule.

---

## 10 — Relationship to Other Docs

- **Supersedes:** inline CLI output guidelines in
  `docs/spec/features/09-auth-and-profile/stories/145-cli-auth-ux-polish-2.md`. Those
  guidelines are now stale; this document is the canonical reference.
- **TUI output:** governed exclusively by `design.md`. Theme tokens, pane borders,
  and visualiser colours are out of scope for this document.
- **Keybindings:** governed by `../../README.md#keybindings`. The two documents do not overlap.
- **Design record:** `docs/superpowers/specs/2026-04-22-cli-output-design.md` contains
  the full rationale (brainstorming decisions, advisor gap resolutions). Consult it for
  the "why"; consult this document for the "what".
