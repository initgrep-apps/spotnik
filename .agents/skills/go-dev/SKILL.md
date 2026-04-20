---
name: go-dev
description: |
  Idiomatic Go development patterns, testing, error handling, concurrency, and project
  structure. Use when writing Go code, designing Go packages, setting up tests, handling
  errors, managing modules, or structuring Go projects. Triggers on: Go files, go.mod,
  table-driven tests, httptest, goroutines, channels, context, error wrapping, Go CLI
  commands (go build, go test, go vet). Provides non-obvious patterns and decision
  guides that go beyond basic Go syntax knowledge.
---

# Go Development Patterns

Source: go.dev/doc (Effective Go, modules, testing, error handling, concurrency patterns).

## Quick Decision Guides

### Pointer vs Value Receiver

```
Use pointer receiver (*T) when:
  - Method modifies the receiver
  - T is large (struct with many fields)
  - Consistency: if any method on T uses pointer, all should

Use value receiver (T) when:
  - Method does not modify receiver
  - T is small (int, small struct)
  - T is a map, func, or chan (already reference types)

Rule: value methods can be called on pointers AND values,
      but pointer methods can ONLY be called on pointers.
```

### When to Use Interfaces

```
DO:
  - Accept interfaces, return concrete types
  - Keep interfaces small (1-3 methods)
  - Define interfaces where they are USED, not where implemented
  - Name single-method interfaces: method + "-er" (Reader, Writer, Stringer)

DON'T:
  - Create interfaces before you have 2+ implementations
  - Put interfaces in the implementing package
  - Create "god interfaces" with many methods
```

### Error Handling Strategy

```
Return context:     fmt.Errorf("getting user %s: %w", id, err)
Wrap with %w:       when caller should inspect the underlying error
Use %v (not %w):    when underlying error is an implementation detail
Sentinel errors:    var ErrNotFound = errors.New("not found")
Custom error type:  when caller needs structured data from the error
Check errors:       errors.Is(err, target) and errors.As(err, &target)
Never:              compare with == (breaks when errors are wrapped)
```

**Deep dive:** See [references/error-handling.md](references/error-handling.md)

### Concurrency Decision

```
Use goroutine + channel when:
  - Fan-out/fan-in pattern (distribute work, collect results)
  - Pipeline stages (data flows through processing steps)
  - Signaling completion or cancellation
  - Producer/consumer queues

Use sync.Mutex when:
  - Simple shared counter or map
  - Critical section is short and predictable
  - No need for communication between goroutines

Always:
  - Pass context.Context as first parameter
  - Use select with context.Done() for cancellation
  - Use sync.WaitGroup to wait for goroutine completion
  - Goroutines are NOT garbage collected — ensure they can exit
```

**Deep dive:** See [references/effective-patterns.md](references/effective-patterns.md) — Concurrency section

## Naming Conventions

```go
// Package names: short, lowercase, no underscores, no mixedCaps
package httputil  // good
package http_util // bad

// Avoid stutter — package name is already the prefix
http.Server       // good  (not http.HTTPServer)
bytes.Buffer      // good  (not bytes.ByteBuffer)

// Getters: no "Get" prefix
func (u *User) Name() string      // good
func (u *User) GetName() string   // bad

// Setters: "Set" prefix is fine
func (u *User) SetName(n string)  // good

// Interfaces: method + "-er"
type Reader interface { Read(p []byte) (n int, err error) }
type Stringer interface { String() string }

// Unexported = lowercase first letter (package-private)
// Exported = uppercase first letter (public API)

// MixedCaps always, never underscores
var maxRetryCount int  // good
var max_retry_count    // bad

// Acronyms: all caps or all lowercase
var httpClient  // unexported
var HTTPClient  // exported
var urlParser   // unexported
var URLParser   // exported
```

## Interface Compliance Check

Verify at compile time that a type implements an interface:

```go
// Place in the file that defines the type
var _ io.Reader = (*MyType)(nil)
```

Only use when there are no static conversions already present in the code (rare).

## Project Layout

For a CLI tool with internal packages (like this project):

```
project/
├── main.go                  ← package main, entry point only
├── cmd/
│   └── root.go              ← CLI setup (cobra commands)
├── internal/                ← cannot be imported by external modules
│   ├── app/                 ← core application logic
│   ├── api/                 ← HTTP client, external API calls
│   ├── ui/                  ← TUI components
│   ├── state/               ← shared state/store
│   └── config/              ← configuration loading
├── testdata/fixtures/       ← JSON fixtures for tests
├── go.mod
└── go.sum
```

Key rules:
- `internal/` prevents external imports — use it for all non-public code
- Tests live next to the code they test (`foo_test.go` alongside `foo.go`)
- `testdata/` is ignored by `go build` — safe for fixtures
- Entry point (`main.go`) should be minimal — delegate to `cmd/` or `internal/`

**Deep dive:** See [references/modules-and-commands.md](references/modules-and-commands.md)

## Testing Patterns

### Table-Driven Test Template

```go
func TestFoo(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {name: "valid input", input: "hello", want: "HELLO"},
        {name: "empty input", input: "", want: ""},
        {name: "error case", input: "bad", wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Foo(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### HTTP Mock Server Template

```go
func TestAPICall(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/api/v1/resource", r.URL.Path)
        assert.Equal(t, "GET", r.Method)

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"key": "value"})
    }))
    defer srv.Close()

    client := NewClient(srv.URL)
    result, err := client.GetResource(context.Background())
    require.NoError(t, err)
    assert.Equal(t, "value", result.Key)
}
```

### Integration Test Convention

```go
//go:build integration

package mypackage_test

import "testing"

func TestIntegration_FullFlow(t *testing.T) {
    // ...
}
```

Run: `go test -tags integration ./...`

**Deep dive:** See [references/testing-patterns.md](references/testing-patterns.md)

## Essential Go Commands

```bash
go build ./...              # compile all packages
go test ./...               # run all tests
go test -v ./...            # verbose test output
go test -run TestFoo ./...  # run specific test
go test -race ./...         # detect race conditions
go test -count=1 ./...      # disable test caching
go test -cover ./...        # show coverage percentage
go test -coverprofile=c.out ./... && go tool cover -html=c.out  # coverage report

go vet ./...                # static analysis
go fmt ./...                # format all files (gofmt)
go mod tidy                 # sync go.mod/go.sum with imports
go mod vendor               # copy dependencies to vendor/

golangci-lint run           # comprehensive linting (if installed)
```

## References

Read these only when you need deeper detail on a specific topic:

- **[Effective Go Patterns](references/effective-patterns.md)** — naming, interfaces, embedding, concurrency idioms, error patterns from Effective Go
- **[Testing Patterns](references/testing-patterns.md)** — table-driven tests, httptest, subtests, fuzz testing, benchmarks, test fixtures
- **[Error Handling](references/error-handling.md)** — wrapping with %w, errors.Is/As, custom error types, sentinel errors, when to wrap
- **[Modules and Commands](references/modules-and-commands.md)** — go.mod directives, dependency management, project layout, go commands
