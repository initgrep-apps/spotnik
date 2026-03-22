# Go Modules and Commands

Source: go.dev/doc/modules (layout, gomod-ref, managing-dependencies), go.dev/doc/code. Covers go.mod directives, dependency management, project layout, and essential commands.

## Table of Contents

- [go.mod Reference](#gomod-reference)
- [Dependency Management](#dependency-management)
- [Project Layout](#project-layout)
- [Build and Run](#build-and-run)
- [Go Commands Cheat Sheet](#go-commands-cheat-sheet)

## go.mod Reference

### Example

```
module github.com/someuser/myproject

go 1.22

toolchain go1.22.0

require (
    github.com/charmbracelet/bubbletea v0.27.0
    github.com/stretchr/testify v1.8.4
)

require (
    // indirect dependencies (auto-managed by go mod tidy)
    github.com/some/indirect v1.2.3 // indirect
)
```

### Directives

| Directive | Purpose |
|-----------|---------|
| `module` | Declares the module path (import prefix for all packages) |
| `go` | Minimum Go version required (mandatory since Go 1.21) |
| `toolchain` | Suggested Go toolchain version |
| `require` | Declares module dependencies with minimum versions |
| `replace` | Substitutes a module with another version or local path |
| `exclude` | Prevents a specific module version from being used |
| `retract` | Marks versions of your own module as not recommended |
| `godebug` | Sets GODEBUG defaults for main packages |
| `tool` | Declares tool dependencies (Go 1.24+) |

### `module` Directive

```
module github.com/someuser/modname      // v0 or v1
module github.com/someuser/modname/v2   // v2+ (must include major version)
```

The module path should be the repository URL where the module can be downloaded. It becomes the import prefix for all packages in the module.

### `go` Directive

```
go 1.22
```

- Since Go 1.21: mandatory minimum, toolchains refuse to use modules declaring newer versions.
- Affects language features available (e.g., `go 1.22` enables range-over-int).
- Input to toolchain selection.

### `require` Directive

```
require (
    github.com/charmbracelet/bubbletea v0.27.0
    github.com/stretchr/testify v1.8.4
)

// Indirect dependencies in separate block (Go 1.17+)
require (
    github.com/some/dep v1.0.0 // indirect
)
```

### `replace` Directive

```
// Replace with local directory (for development)
replace github.com/someuser/mylib => ../mylib

// Replace with different version
replace github.com/old/module v1.0.0 => github.com/new/module v2.0.0
```

Use `replace` for:
- Local development of a dependency
- Forking a dependency
- Fixing a bug in a dependency before the upstream publishes a fix

### `exclude` Directive

```
exclude github.com/broken/module v1.3.0
```

Prevents a specific version from being selected. Go selects the next allowed version.

### `retract` Directive

```
retract (
    v1.0.0  // Published with incorrect API
    [v1.1.0, v1.2.0]  // Range of versions
)
```

Used in your own module to mark versions as not recommended.

## Dependency Management

### Adding Dependencies

```bash
# Add a specific dependency
go get github.com/charmbracelet/bubbletea@v0.27.0

# Add latest version
go get github.com/charmbracelet/bubbletea@latest

# Add and update all dependencies
go get -u ./...

# Just sync go.mod with imports (preferred)
go mod tidy
```

### Updating Dependencies

```bash
# Update a specific dependency to latest
go get github.com/charmbracelet/bubbletea@latest

# Update all direct dependencies
go get -u ./...

# Update all (including indirect)
go get -u all

# Check for available updates
go list -m -u all
```

### Removing Dependencies

```bash
# Remove unused dependencies
go mod tidy
```

`go mod tidy` is the safest way to clean up — it adds missing and removes unused dependencies.

### Vendoring

```bash
# Copy dependencies to vendor/
go mod vendor

# Build using vendor directory
go build -mod=vendor ./...
```

At `go 1.14+`, if `vendor/modules.txt` exists and is consistent, vendoring is automatic.

### Module Graph

```bash
# Show all dependencies
go list -m all

# Show dependency graph
go mod graph

# Explain why a module is needed
go mod why github.com/some/dep

# Verify checksums
go mod verify
```

## Project Layout

### Single Command (CLI Tool)

```
project/
├── main.go              ← package main, entry point
├── cmd/
│   └── root.go          ← CLI commands (cobra)
├── internal/            ← internal packages (not importable externally)
│   ├── app/
│   ├── api/
│   ├── ui/
│   ├── state/
│   └── config/
├── testdata/
│   └── fixtures/        ← JSON test fixtures
├── go.mod
├── go.sum
└── Makefile
```

### Key Conventions

- **`internal/`** — cannot be imported by external modules. Use for all non-public packages.
- **`cmd/`** — conventional location for command entry points when you have multiple commands.
- **`testdata/`** — ignored by `go build`. Safe for test fixtures, golden files.
- **`main.go`** in root — entry point, should be minimal (delegate to `cmd/` or `internal/`).
- Package name = directory name (lowercase, no underscores).

### Package Organization Rules

1. **One package per directory** — all `.go` files in a directory must declare the same package.
2. **Package name matches directory** — `internal/api/` → `package api`.
3. **Test files in same directory** — `foo_test.go` alongside `foo.go`.
4. **External test packages** — `package foo_test` to test only the public API.
5. **Internal test packages** — `package foo` to test unexported functions.

### Import Path Convention

```go
// Internal package imports use the full module path
import "github.com/someuser/myproject/internal/api"
import "github.com/someuser/myproject/internal/state"
```

## Build and Run

```bash
# Build
go build -o bin/myapp ./...     # build all, output to bin/
go build -o bin/myapp .          # build current directory
go build -race -o bin/myapp .    # build with race detector

# Run
go run .                         # build and run
go run ./cmd/myapp               # run specific command

# Install (to $GOPATH/bin or $GOBIN)
go install ./...
go install github.com/someuser/myapp@latest  # remote install

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o bin/myapp-linux .
GOOS=darwin GOARCH=arm64 go build -o bin/myapp-darwin .
GOOS=windows GOARCH=amd64 go build -o bin/myapp.exe .
```

### Build Flags

```bash
# Embed version info
go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD)"

# Strip debug info (smaller binary)
go build -ldflags "-s -w"

# Trimpath (reproducible builds)
go build -trimpath
```

## Go Commands Cheat Sheet

### Testing

```bash
go test ./...                    # all tests
go test -v ./...                 # verbose
go test -run TestFoo ./pkg/      # specific test
go test -run TestFoo/subtest     # specific subtest
go test -race ./...              # race detection
go test -count=1 ./...           # no cache
go test -short ./...             # skip slow tests
go test -timeout 5m ./...        # timeout
go test -cover ./...             # coverage %
go test -coverprofile=c.out ./...  # coverage file
go test -tags integration ./...  # with build tags
go test -bench . ./...           # benchmarks
go test -benchmem -bench . ./... # benchmarks + alloc info
go test -fuzz FuzzFoo -fuzztime 30s  # fuzz testing
```

### Code Quality

```bash
go vet ./...                     # static analysis
go fmt ./...                     # format (or gofmt -s -w .)
golangci-lint run                # comprehensive linting
```

### Module Management

```bash
go mod init github.com/user/mod # initialize module
go mod tidy                      # sync deps with imports
go mod vendor                    # copy deps to vendor/
go mod verify                    # verify checksums
go mod graph                     # dependency graph
go mod why github.com/dep        # why is this dep needed?
go get github.com/dep@v1.2.3    # add/update dependency
go list -m -u all                # check for updates
```

### Debugging and Profiling

```bash
go tool pprof cpu.prof           # CPU profile
go tool pprof mem.prof           # memory profile
go tool cover -html=c.out        # coverage HTML
go tool trace trace.out          # execution trace
GODEBUG=gctrace=1 ./myapp       # GC tracing
```
