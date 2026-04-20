# Effective Go Patterns

Source: go.dev/doc/effective_go — condensed to non-obvious patterns and idioms.

## Table of Contents

- [Naming](#naming)
- [Control Flow](#control-flow)
- [Functions](#functions)
- [Data Structures](#data-structures)
- [Methods](#methods)
- [Interfaces](#interfaces)
- [Embedding](#embedding)
- [Concurrency](#concurrency)
- [Errors and Panic](#errors-and-panic)

## Naming

### Package Names

- Short, concise, lowercase, single-word. No underscores or mixedCaps.
- Package name is the base name of its source directory.
- Don't stutter: `bufio.Reader`, not `bufio.BufReader`. `ring.New()`, not `ring.NewRing()`.
- The importer sees `pkg.Name`, so exported names use the package as context.

### Getters and Setters

```go
// Getter: no "Get" prefix
func (u *User) Name() string { return u.name }

// Setter: "Set" prefix
func (u *User) SetName(n string) { u.name = n }
```

### Interface Names

- One-method interfaces: method name + `-er` suffix: `Reader`, `Writer`, `Formatter`, `CloseNotifier`.
- Honor canonical names: if your type has `Read`, `Write`, `Close`, `Flush`, `String` — give them the same signature and meaning as the stdlib.

## Control Flow

### Guard Clause Pattern (No Else After Return)

```go
// GOOD: guard clause, happy path runs down
f, err := os.Open(name)
if err != nil {
    return err
}
d, err := f.Stat()
if err != nil {
    f.Close()
    return err
}
codeUsing(f, d)
```

The error cases end in `return`, so no `else` is needed. This is strongly idiomatic.

### Redeclaration with `:=`

```go
f, err := os.Open(name)   // declares f and err
d, err := f.Stat()        // declares d, REASSIGNS err (same scope)
```

`:=` can reuse a variable if: (1) same scope, (2) at least one new variable is declared.

### Switch Without Expression

```go
func unhex(c byte) byte {
    switch {
    case '0' <= c && c <= '9':
        return c - '0'
    case 'a' <= c && c <= 'f':
        return c - 'a' + 10
    case 'A' <= c && c <= 'F':
        return c - 'A' + 10
    }
    return 0
}
```

### Type Switch

```go
switch v := value.(type) {
case string:
    return v
case Stringer:
    return v.String()
default:
    return fmt.Sprintf("%v", value)
}
```

## Functions

### Multiple Return Values

```go
func nextInt(b []byte, i int) (int, int) {
    // returns value AND new position
}
```

Always return `(result, error)` from functions that can fail.

### Named Result Parameters

```go
func ReadFull(r Reader, buf []byte) (n int, err error) {
    for len(buf) > 0 && err == nil {
        var nr int
        nr, err = r.Read(buf)
        n += nr
        buf = buf[nr:]
    }
    return
}
```

Use named returns sparingly — mainly when they document the meaning of results or simplify defers.

### Defer

- Executes when the enclosing function returns (not when the block exits).
- Multiple defers execute in LIFO order.
- Deferred function arguments are evaluated when the defer executes, not when the deferred function runs.

```go
// Trace function entry/exit
func trace(s string) string {
    log.Println("entering:", s)
    return s
}
func un(s string) {
    log.Println("leaving:", s)
}
func a() {
    defer un(trace("a"))  // trace("a") runs NOW, un() runs on return
    log.Println("in a")
}
```

## Data Structures

### `new` vs `make`

```go
// new(T) — allocates zeroed T, returns *T
p := new(SyncedBuffer)  // type *SyncedBuffer, zeroed and ready

// make(T, args) — only for slices, maps, channels
// Returns initialized (non-zero) value of type T (not *T)
s := make([]int, 10, 100)  // slice: len=10, cap=100
m := make(map[string]int)  // map: ready to use
c := make(chan int, 10)     // buffered channel: cap=10
```

### Composite Literals

```go
// Can take address of composite literal
return &File{fd: fd, name: name}

// Fields can be in any order (labeled)
return &File{name: name, fd: fd}

// Unlabeled: must match declaration order exactly
return &File{fd, name, nil, 0}
```

### Slice Tricks

```go
// Append — may reallocate
x = append(x, elem)
x = append(x, slice...)  // append another slice

// 2D slice — allocate backing array once
data := make([]int, rows*cols)
picture := make([][]int, rows)
for i := range picture {
    picture[i], data = data[:cols], data[cols:]
}
```

### Map Idioms

```go
// Comma-ok pattern
val, ok := m[key]
if !ok {
    // key not present
}

// Delete
delete(m, key)

// Zero value is useful — maps return zero for missing keys
attended := map[string]bool{}
if attended[person] {  // false if not in map
    // ...
}
```

## Methods

### Pointer vs Value Receivers

```go
// Value methods: called on pointer OR value
func (s Sequence) Len() int { return len(s) }

// Pointer methods: called on pointer ONLY
func (p *ByteSlice) Write(data []byte) (n int, err error) {
    *p = append(*p, data...)
    return len(data), nil
}
```

Rule: value methods can be invoked on pointers and values. Pointer methods can only be invoked on pointers. Exception: if the value is addressable, the compiler auto-inserts `&`.

## Interfaces

### Implicit Satisfaction

Types implement interfaces implicitly — no `implements` keyword. A type can implement multiple interfaces.

### Accept Interfaces, Return Structs

```go
// GOOD: function accepts interface
func Process(r io.Reader) error { ... }

// GOOD: function returns concrete type
func NewClient(baseURL string) *Client { ... }
```

### Generality Pattern

If a type exists only to implement an interface and has no exported methods beyond that interface, don't export the type itself:

```go
// Export interface, not the implementation
type StoreReader interface {
    Read(key string) ([]byte, error)
}

// unexported implementation
type fileStore struct { dir string }
func (fs *fileStore) Read(key string) ([]byte, error) { ... }

// Constructor returns the interface
func NewStoreReader(dir string) StoreReader {
    return &fileStore{dir: dir}
}
```

### Compile-Time Interface Check

```go
var _ json.Marshaler = (*RawMessage)(nil)
```

Use this only when there are no static conversions in the code that would catch the mismatch.

## Embedding

Go doesn't have subclassing. Use embedding to "borrow" implementations.

### Interface Embedding

```go
type ReadWriter interface {
    Reader
    Writer
}
```

### Struct Embedding

```go
type ReadWriter struct {
    *Reader
    *Writer
}
```

Embedded methods are promoted — `rw.Read()` calls `rw.Reader.Read()`. The outer type can override them.

Key difference from subclassing: when an embedded method runs, `this`/`self` refers to the embedded field, not the outer struct.

```go
type Job struct {
    Command string
    *log.Logger
}

job := &Job{Command: "cmd", Logger: log.New(os.Stderr, "Job: ", log.Ldate)}
job.Println("starting now...")  // calls job.Logger.Println
```

## Concurrency

### Share by Communicating

> Do not communicate by sharing memory; instead, share memory by communicating.

One goroutine has access to a value at a time. Pass values on channels. Data races cannot occur by design.

### Goroutine Closure Pitfall (Pre-1.22)

```go
// Go 1.22+: loop variable is per-iteration (safe)
for _, v := range values {
    go func() {
        process(v)  // each goroutine gets its own v
    }()
}

// Pre-1.22: loop variable was shared — needed explicit copy
for _, v := range values {
    v := v  // shadow v with per-iteration copy
    go func() {
        process(v)
    }()
}
```

### Channel as Semaphore

```go
var sem = make(chan struct{}, MaxOutstanding)

func handle(r *Request) {
    sem <- struct{}{}  // acquire
    process(r)
    <-sem              // release
}
```

### Fixed Worker Pool (Preferred Pattern)

```go
func Serve(queue chan *Request, quit chan bool) {
    for i := 0; i < MaxWorkers; i++ {
        go func() {
            for r := range queue {
                process(r)
            }
        }()
    }
    <-quit
}
```

This limits goroutine count, unlike the semaphore approach which still creates unbounded goroutines.

### Channels of Channels (Request/Response)

```go
type Request struct {
    args       []int
    resultChan chan int
}

// Client sends request with its own response channel
req := &Request{args: []int{3, 4, 5}, resultChan: make(chan int)}
clientRequests <- req
result := <-req.resultChan
```

### Leaky Buffer (Object Pool via Channel)

```go
var freeList = make(chan *Buffer, 100)

func getBuffer() *Buffer {
    select {
    case b := <-freeList:
        return b  // reuse
    default:
        return new(Buffer)  // allocate
    }
}

func putBuffer(b *Buffer) {
    select {
    case freeList <- b:
        // returned to pool
    default:
        // pool full, GC will reclaim
    }
}
```

## Errors and Panic

### Error Is a Value

```go
type error interface {
    Error() string
}

// Custom error with context
type PathError struct {
    Op   string
    Path string
    Err  error
}

func (e *PathError) Error() string {
    return e.Op + " " + e.Path + ": " + e.Err.Error()
}
func (e *PathError) Unwrap() error { return e.Err }
```

### Panic and Recover

- `panic` for truly unrecoverable errors (programmer bugs, impossible state).
- Never `panic` for normal errors — return `error` instead.
- `recover` only works inside a deferred function.

```go
func safelyDo(work func()) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("recovered: %v", r)
        }
    }()
    work()
    return nil
}
```

Library convention: even if a package uses panic internally, its public API should still present explicit error return values.
