# Go Testing Patterns

Source: go.dev/doc (testing tutorial, Effective Go, Go blog). Covers table-driven tests, httptest, subtests, fuzz testing, benchmarks, and conventions.

## Table of Contents

- [Testing Basics](#testing-basics)
- [Table-Driven Tests](#table-driven-tests)
- [Subtests](#subtests)
- [HTTP Mock Server](#http-mock-server)
- [Test Fixtures](#test-fixtures)
- [Integration Tests](#integration-tests)
- [Fuzz Testing](#fuzz-testing)
- [Benchmarks](#benchmarks)
- [Test Helpers](#test-helpers)
- [Coverage](#coverage)
- [Common Commands](#common-commands)

## Testing Basics

- Test files: `*_test.go` — same package as code being tested.
- Test functions: `func TestXxx(t *testing.T)` — name starts with uppercase after `Test`.
- `t.Errorf()` marks failure but continues. `t.Fatalf()` marks failure and stops.
- `t.Helper()` marks a function as a test helper — failure messages report the caller's line.
- `t.Parallel()` marks a test to run in parallel with other parallel tests.

```go
package greetings

import "testing"

func TestHelloName(t *testing.T) {
    name := "Gladys"
    msg, err := Hello(name)
    if err != nil {
        t.Fatalf("Hello(%q) returned error: %v", name, err)
    }
    if msg == "" {
        t.Errorf("Hello(%q) returned empty string", name)
    }
}
```

## Table-Driven Tests

The idiomatic Go testing pattern. Define test cases as a slice of structs, loop with `t.Run`.

### Basic Template

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
                if err == nil {
                    t.Fatal("expected error, got nil")
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if got != tt.want {
                t.Errorf("Foo(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}
```

### With testify (assert/require)

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFoo(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {name: "valid", input: "hello", want: "HELLO"},
        {name: "error", input: "bad", wantErr: true},
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

`require` = fail immediately (like `t.Fatal`). `assert` = continue after failure (like `t.Error`). Use `require` for preconditions, `assert` for actual checks.

## Subtests

`t.Run` creates subtests that:
- Run independently (can be filtered with `-run TestFoo/valid`)
- Have their own `t` (failures don't stop other subtests)
- Support `t.Parallel()` within each subtest

```go
func TestGroup(t *testing.T) {
    t.Run("create", func(t *testing.T) {
        t.Parallel()
        // ...
    })
    t.Run("delete", func(t *testing.T) {
        t.Parallel()
        // ...
    })
}
```

Run specific subtest: `go test -run TestGroup/create`

## HTTP Mock Server

Use `net/http/httptest` — no external mock libraries needed.

### Basic Mock Server

```go
func TestGetUser(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        if r.URL.Path != "/api/users/123" {
            t.Errorf("unexpected path: %s", r.URL.Path)
        }
        if r.Header.Get("Authorization") != "Bearer test-token" {
            t.Errorf("missing auth header")
        }

        // Return response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        fmt.Fprint(w, `{"id":"123","name":"Alice"}`)
    }))
    defer srv.Close()

    client := NewAPIClient(srv.URL, "test-token")
    user, err := client.GetUser(context.Background(), "123")
    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}
```

### Multi-Endpoint Mock (Router Pattern)

```go
func TestClient(t *testing.T) {
    mux := http.NewServeMux()

    mux.HandleFunc("GET /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")  // Go 1.22+ enhanced routing
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"id":"%s","name":"Alice"}`, id)
    })

    mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusCreated)
        fmt.Fprint(w, `{"id":"new-id"}`)
    })

    srv := httptest.NewServer(mux)
    defer srv.Close()

    client := NewAPIClient(srv.URL)
    // ... test various endpoints
}
```

### Testing Error Responses

```go
func TestGetUser_ServerError(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusInternalServerError)
        fmt.Fprint(w, `{"error":"internal server error"}`)
    }))
    defer srv.Close()

    client := NewAPIClient(srv.URL)
    _, err := client.GetUser(context.Background(), "123")
    require.Error(t, err)
    assert.Contains(t, err.Error(), "500")
}

func TestGetUser_RateLimit(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Retry-After", "30")
        w.WriteHeader(http.StatusTooManyRequests)
    }))
    defer srv.Close()

    client := NewAPIClient(srv.URL)
    _, err := client.GetUser(context.Background(), "123")
    require.Error(t, err)
    // Verify rate limit error type if using custom errors
}
```

## Test Fixtures

Store test data in `testdata/` directories. The `go` tool ignores `testdata/` during builds.

```
project/
├── internal/api/
│   ├── client.go
│   ├── client_test.go
│   └── testdata/fixtures/
│       ├── user.json
│       ├── playlist.json
│       └── error_response.json
```

```go
func loadFixture(t *testing.T, name string) []byte {
    t.Helper()
    data, err := os.ReadFile(filepath.Join("testdata", "fixtures", name))
    if err != nil {
        t.Fatalf("failed to load fixture %s: %v", name, err)
    }
    return data
}

func TestParseUser(t *testing.T) {
    data := loadFixture(t, "user.json")
    var user User
    err := json.Unmarshal(data, &user)
    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}
```

## Integration Tests

Use build tags to separate integration tests from unit tests.

### Convention

```go
//go:build integration

package mypackage_test  // external test package

import "testing"

func TestIntegration_FullFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    // Test multi-component flow
}
```

File naming: `*_integration_test.go`

### Running

```bash
# Unit tests only (default)
go test ./...

# Integration tests only
go test -tags integration ./...

# Both
go test -tags integration ./...

# Skip integration even with tag
go test -tags integration -short ./...
```

### Makefile targets

```makefile
test:
    go test ./... -race -count=1

test-integration:
    go test ./... -race -count=1 -tags integration

ci: lint test test-integration
```

## Fuzz Testing

Go 1.18+ includes native fuzz testing. Finds edge cases automatically.

```go
func FuzzParseQuery(f *testing.F) {
    // Seed corpus — initial inputs
    f.Add("key=value")
    f.Add("")
    f.Add("a=1&b=2")

    f.Fuzz(func(t *testing.T, input string) {
        result, err := ParseQuery(input)
        if err != nil {
            return  // invalid input is ok, just don't crash
        }
        // Property: encoding the result should be parseable
        encoded := result.Encode()
        _, err = ParseQuery(encoded)
        if err != nil {
            t.Errorf("round-trip failed: ParseQuery(%q) then Encode() gave %q which failed to parse: %v",
                input, encoded, err)
        }
    })
}
```

Run: `go test -fuzz FuzzParseQuery -fuzztime 30s`

## Benchmarks

```go
func BenchmarkFoo(b *testing.B) {
    for b.Loop() {   // Go 1.24+ preferred (replaces for i := 0; i < b.N; i++)
        Foo("input")
    }
}

// With setup
func BenchmarkFooLargeInput(b *testing.B) {
    input := strings.Repeat("x", 10000)
    b.ResetTimer()
    for b.Loop() {
        Foo(input)
    }
}
```

Run: `go test -bench BenchmarkFoo -benchmem`

The `-benchmem` flag reports allocations per operation.

## Test Helpers

### Helper Function Pattern

```go
func newTestClient(t *testing.T, handler http.Handler) *Client {
    t.Helper()  // ensures failure reports caller's line, not this line
    srv := httptest.NewServer(handler)
    t.Cleanup(srv.Close)  // auto-cleanup when test finishes
    return NewClient(srv.URL)
}
```

### t.Cleanup

```go
func TestWithTempDir(t *testing.T) {
    dir := t.TempDir()  // auto-cleaned up
    // ... use dir
}

func TestWithResource(t *testing.T) {
    db := openTestDB(t)
    t.Cleanup(func() { db.Close() })
    // ...
}
```

## Coverage

```bash
# Quick coverage percentage
go test -cover ./...

# Detailed coverage profile
go test -coverprofile=coverage.out ./...

# HTML report
go tool cover -html=coverage.out

# Function-level coverage
go tool cover -func=coverage.out

# Per-package coverage
go test -cover ./internal/api/...
```

## Common Commands

```bash
# Run all tests
go test ./...

# Verbose output
go test -v ./...

# Run specific test
go test -run TestFoo ./internal/api/

# Run specific subtest
go test -run TestFoo/valid_input ./internal/api/

# Race detector (always use in CI)
go test -race ./...

# Disable cache
go test -count=1 ./...

# Timeout
go test -timeout 30s ./...

# Short mode (skip slow tests)
go test -short ./...

# With build tags
go test -tags integration ./...
```
