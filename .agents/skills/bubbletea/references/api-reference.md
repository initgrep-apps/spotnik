# Bubble Tea API Reference (v1)

## Table of Contents
- [Core Types](#core-types)
- [Message Types](#message-types)
- [Command Functions](#command-functions)
- [Program Options](#program-options)
- [Key Constants](#key-constants)
- [Logging](#logging)

## Core Types

### tea.Model (interface)

```go
type Model interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string
}
```

### tea.Msg (interface)

```go
type Msg interface{}
// Any type can be a Msg. Use typed structs for custom messages.
```

### tea.Cmd (func type)

```go
type Cmd func() Msg
// A function that performs I/O and returns a Msg.
// nil Cmd = no command.
```

### tea.Program

```go
p := tea.NewProgram(model, opts ...ProgramOption) *Program

p.Run() (Model, error)       // blocking, returns final model
p.Send(msg Msg)              // send a message from outside
p.Quit()                     // quit the program
p.Kill()                     // force kill
p.Wait()                     // wait for program to finish
p.ReleaseTerminal() error    // release terminal control
p.RestoreTerminal() error    // restore terminal control
p.Printf(template, args...)  // print outside the TUI
p.Println(args...)           // println outside the TUI
```

Deprecated (still works in v1): `p.Start()`, `p.StartReturningModel()`.

## Message Types

### tea.KeyMsg

```go
type KeyMsg struct {
    Type  KeyType
    Runes []rune
    Alt   bool
}

func (k KeyMsg) String() string  // e.g. "ctrl+c", "enter", "a"
```

### tea.WindowSizeMsg

```go
type WindowSizeMsg struct {
    Width  int
    Height int
}
```

### tea.MouseMsg / tea.MouseEvent

```go
type MouseMsg struct {
    X      int
    Y      int
    Type   MouseEventType
    Alt    bool
    Ctrl   bool
    Action MouseAction
    Button MouseButton
}
```

### Other Built-in Messages

| Type | When sent |
|---|---|
| `tea.FocusMsg` | Terminal gained focus |
| `tea.BlurMsg` | Terminal lost focus |
| `tea.QuitMsg` | Program quitting |
| `tea.InterruptMsg` | Program received interrupt (Ctrl+C without handler) |
| `tea.ResumeMsg` | Program resumed after suspend |
| `tea.SuspendMsg` | Program about to suspend |

## Command Functions

### tea.Batch

```go
func Batch(cmds ...Cmd) Cmd
```
Run multiple commands concurrently. All messages arrive as they complete (unordered).

### tea.Sequence

```go
func Sequence(cmds ...Cmd) Cmd
```
Run commands in order. Each command's resulting message is delivered to `Update()` before
the next command runs. Use when order matters.

### tea.Tick

```go
func Tick(d time.Duration, fn func(time.Time) Msg) Cmd
```
One-shot timer. Returns the message from `fn` after duration `d`.
Must re-arm in `Update()` for periodic polling.

### tea.Every

```go
func Every(duration time.Duration, fn func(time.Time) Msg) Cmd
```
Like Tick but aligned to wall clock intervals. Useful for clock displays.

### tea.Quit

```go
func Quit() Msg
```
Returns a `QuitMsg`. Use as: `return m, tea.Quit`

### tea.WindowSize

```go
func WindowSize() Cmd
```
Requests the current window size. Delivers a `WindowSizeMsg`.

### tea.SetWindowTitle

```go
func SetWindowTitle(title string) Cmd
```

### tea.ClearScreen

```go
func ClearScreen() Msg
```

### Screen Mode Commands

```go
func EnterAltScreen() Msg    // switch to alternate screen buffer
func ExitAltScreen() Msg     // back to normal screen
func EnableMouseCellMotion() Msg
func EnableMouseAllMotion() Msg
func DisableMouse() Msg
func EnableBracketedPaste() Msg
func DisableBracketedPaste() Msg
func EnableReportFocus() Msg
func DisableReportFocus() Msg
func HideCursor() Msg
func ShowCursor() Msg
```

### tea.Exec

```go
func Exec(c ExecCommand, fn ExecCallback) Cmd
func ExecProcess(c *exec.Cmd, fn ExecCallback) Cmd
```
Run an external process (e.g., open $EDITOR). Releases terminal control during execution.

## Program Options

```go
tea.WithAltScreen()           // start in alternate screen buffer
tea.WithMouseCellMotion()     // enable mouse (motion within cells)
tea.WithMouseAllMotion()      // enable mouse (all motion)
tea.WithContext(ctx)           // cancel via context
tea.WithFPS(fps int)           // set renderer FPS (default 60)
tea.WithFilter(fn)             // filter messages before Update
tea.WithInput(r io.Reader)     // custom input source
tea.WithInputTTY()             // open /dev/tty for input
tea.WithOutput(w io.Writer)    // custom output destination
tea.WithReportFocus()          // receive FocusMsg/BlurMsg
tea.WithoutBracketedPaste()    // disable bracketed paste
tea.WithoutCatchPanics()       // let panics propagate
tea.WithoutRenderer()          // no rendering (testing)
tea.WithoutSignalHandler()     // no SIGINT handling
tea.WithoutSignals()           // ignore all signals
tea.WithEnvironment(env)       // custom environment variables
```

## Key Constants

### KeyType values (v1)

```go
tea.KeyRunes          // regular character input
tea.KeyEnter
tea.KeyTab
tea.KeyShiftTab
tea.KeyBackspace
tea.KeyEsc
tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight
tea.KeyHome, tea.KeyEnd
tea.KeyPgUp, tea.KeyPgDown
tea.KeyDelete, tea.KeyInsert
tea.KeySpace
tea.KeyCtrlA ... tea.KeyCtrlZ
tea.KeyF1 ... tea.KeyF20
```

### String matching (preferred approach)

```go
switch msg.String() {
case "enter":     // Enter key
case "tab":       // Tab
case "shift+tab": // Shift+Tab
case "up":        // Arrow up
case "ctrl+c":    // Ctrl+C
case " ":         // Space (v1), becomes "space" in v2
case "esc":       // Escape
case "backspace": // Backspace
case "f1":        // F1
}
```

## Logging

```go
// Log to file (stdout is occupied by the TUI)
f, err := tea.LogToFile("debug.log", "prefix")
defer f.Close()

// With custom logger
f, err := tea.LogToFileWith("debug.log", "prefix", myLogger)
```

Use `log.Print()` / `log.Printf()` after calling `LogToFile` — output goes to the file.
Monitor with `tail -f debug.log`.

## Variables

```go
var ErrInterrupted = errors.New("program was interrupted")
```
