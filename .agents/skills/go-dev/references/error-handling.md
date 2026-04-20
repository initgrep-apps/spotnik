# Go Error Handling

Source: go.dev/blog/go1.13-errors, go.dev/doc/effective_go (Errors section). Covers error wrapping, errors.Is/As, custom types, sentinel errors, and decision guides.

## Table of Contents

- [Core Principles](#core-principles)
- [Error Wrapping with %w](#error-wrapping-with-w)
- [errors.Is and errors.As](#errorsis-and-errorsas)
- [Sentinel Errors](#sentinel-errors)
- [Custom Error Types](#custom-error-types)
- [When to Wrap vs Not Wrap](#when-to-wrap-vs-not-wrap)
- [Error Handling in APIs](#error-handling-in-apis)
- [Patterns and Anti-Patterns](#patterns-and-anti-patterns)

## Core Principles

1. Errors are values — not exceptions, not strings, not codes.
2. Always check errors: `if err != nil { ... }`
3. Add context when propagating: `fmt.Errorf("doing X: %w", err)`
4. Use `errors.Is` and `errors.As` (not `==` comparison)
5. The `error` interface has one method: `Error() string`

```go
type error interface {
    Error() string
}
```

## Error Wrapping with %w

Go 1.13+ introduced `%w` in `fmt.Errorf` to wrap errors while preserving the chain.

```go
// Wrap with context — callers can inspect the underlying error
if err != nil {
    return fmt.Errorf("reading config file %s: %w", path, err)
}

// DON'T wrap — underlying error is an implementation detail
if err != nil {
    return fmt.Errorf("reading config: %v", err)  // %v, not %w
}
```

`%w` makes the wrapped error available to `errors.Is` and `errors.As`. `%v` creates a new error with the text only — the original error is not inspectable.

## errors.Is and errors.As

### errors.Is — Check Error Identity

Traverses the entire error chain looking for a match.

```go
var ErrNotFound = errors.New("not found")

// Replaces: if err == ErrNotFound
if errors.Is(err, ErrNotFound) {
    // err, or any error it wraps, matches ErrNotFound
}

// Works through wrapping layers
err := fmt.Errorf("user lookup: %w", ErrNotFound)
errors.Is(err, ErrNotFound)  // true!
```

### errors.As — Extract Error Type

Traverses the chain looking for an error of a specific type.

```go
var pathErr *os.PathError

// Replaces: if e, ok := err.(*os.PathError); ok
if errors.As(err, &pathErr) {
    // pathErr is set to the matched error
    fmt.Println("failed path:", pathErr.Path)
}
```

Note: takes a **pointer to a pointer** (or pointer to an interface). This is correct because it needs to set the value.

### Custom Is Method

A type can customize how `errors.Is` matches it:

```go
type Error struct {
    Path string
    User string
}

func (e *Error) Is(target error) bool {
    t, ok := target.(*Error)
    if !ok {
        return false
    }
    // Match if non-zero fields match
    return (e.Path == t.Path || t.Path == "") &&
           (e.User == t.User || t.User == "")
}

// Usage:
if errors.Is(err, &Error{User: "alice"}) {
    // err has User field set to "alice"
}
```

## Sentinel Errors

Package-level error values for specific conditions callers need to handle.

```go
// Define
var (
    ErrNotFound     = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
    ErrRateLimit    = errors.New("rate limit exceeded")
)

// Return — always wrap, never return directly
func FetchItem(name string) (*Item, error) {
    if itemNotFound(name) {
        return nil, fmt.Errorf("%q: %w", name, ErrNotFound)
    }
    // ...
}

// Check
if errors.Is(err, ErrNotFound) {
    // handle not found
}
```

**Why wrap instead of returning directly?** If you return `ErrNotFound` directly, callers might write `if err == pkg.ErrNotFound`, which breaks when you later add context. Wrapping forces callers to use `errors.Is`, which is future-proof.

## Custom Error Types

When callers need structured data from errors, not just identity.

```go
// Define
type APIError struct {
    StatusCode int
    Message    string
    RetryAfter int  // seconds, for 429 responses
    Err        error
}

func (e *APIError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("API error %d: %s: %v", e.StatusCode, e.Message, e.Err)
    }
    return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error { return e.Err }

// Return
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("GET %s: %w", path, err)
    }
    if resp.StatusCode >= 400 {
        return nil, &APIError{
            StatusCode: resp.StatusCode,
            Message:    http.StatusText(resp.StatusCode),
            RetryAfter: parseRetryAfter(resp.Header),
        }
    }
    // ...
}

// Check
var apiErr *APIError
if errors.As(err, &apiErr) {
    if apiErr.StatusCode == 429 {
        time.Sleep(time.Duration(apiErr.RetryAfter) * time.Second)
        // retry
    }
}
```

## When to Wrap vs Not Wrap

### Wrap (`%w`) when:

- The caller **provided** the resource that caused the error (e.g., caller passed an `io.Reader`)
- The error is part of your **public API contract** (documented in godoc)
- The underlying error type is **stable** and won't change

### Don't wrap (`%v`) when:

- The underlying error is an **implementation detail** (e.g., which database driver you use)
- Wrapping would **commit your API** to always returning that specific error type
- You want to **prevent callers** from depending on internal error structure

```go
// WRAP: caller provided the io.Reader, they should see IO errors
func Parse(r io.Reader) (*Data, error) {
    _, err := io.ReadAll(r)
    if err != nil {
        return nil, fmt.Errorf("parsing data: %w", err)  // %w
    }
}

// DON'T WRAP: database is an implementation detail
func LookupUser(name string) (*User, error) {
    row := db.QueryRow("SELECT ...")
    if err := row.Scan(&u); err != nil {
        return nil, fmt.Errorf("looking up user %q: %v", name, err)  // %v
    }
}
```

## Error Handling in APIs

### HTTP Client Error Pattern

```go
func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, error) {
    // ... build request ...

    resp, err := c.httpClient.Do(req.WithContext(ctx))
    if err != nil {
        return nil, fmt.Errorf("%s %s: %w", method, path, err)
    }
    defer resp.Body.Close()

    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("%s %s: reading body: %w", method, path, err)
    }

    if resp.StatusCode >= 400 {
        return nil, &APIError{
            StatusCode: resp.StatusCode,
            Message:    string(data),
        }
    }

    return data, nil
}
```

### Layer Error Context

Each layer adds its own context:

```go
// API layer
func (c *Client) GetTrack(ctx context.Context, id string) (*Track, error) {
    data, err := c.do(ctx, "GET", "/tracks/"+id, nil)
    if err != nil {
        return nil, fmt.Errorf("getting track %s: %w", id, err)
    }
    // ...
}

// Result chain: "getting track abc123: GET /tracks/abc123: connection refused"
```

## Patterns and Anti-Patterns

### DO

```go
// Always add context
return fmt.Errorf("loading config: %w", err)

// Use errors.Is for sentinel comparison
if errors.Is(err, io.EOF) { ... }

// Use errors.As for type extraction
var pathErr *fs.PathError
if errors.As(err, &pathErr) { ... }

// Return early on error (guard clause)
if err != nil {
    return err
}
```

### DON'T

```go
// Don't compare with ==
if err == io.EOF { ... }  // BREAKS with wrapped errors

// Don't use type assertion directly
if e, ok := err.(*os.PathError); ok { ... }  // BREAKS with wrapping

// Don't ignore errors silently
result, _ := SomeFunc()  // only ok for explicitly documented cases

// Don't panic for expected errors
if err != nil {
    panic(err)  // use return instead
}

// Don't double-wrap with the same context
return fmt.Errorf("failed: %w", fmt.Errorf("failed: %w", err))
```
