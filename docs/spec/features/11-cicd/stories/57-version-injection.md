---
title: "Version Injection"
feature: 11-cicd
status: done
---

## Background
`internal/app/splash.go` has `const appVersion = "v1.1.0"` hardcoded. The Makefile injects
`main.version` and `main.buildTime` via LDFLAGS but the variables are never declared in
`main.go`. This story wires dynamic version injection through the call chain so every build
gets the version from LDFLAGS, and `spotnik --version` works correctly.

## Design

### Version flow: main.go → cmd → app → splash

1. **`main.go`**: Add package-level vars:
   ```go
   var version = "dev"
   var buildTime = ""
   func main() { cmd.Execute(version, buildTime) }
   ```
2. **`cmd/root.go`**: Change `Execute()` to `Execute(version, buildTime string)`.
   Set `rootCmd.Version = version`. Pass into `AppOptions`.
3. **`internal/app/app.go`**: Add `Version string` and `BuildTime string` to `AppOptions`
   and the `App` struct.
4. **`internal/app/splash.go`**: Remove `const appVersion`. Read version from `a.version`.
5. **`internal/app/splash_test.go`**: Update assertions — pass `"dev"` or `"v0.1.0"` as
   injected version; no longer references `appVersion` const.

### Build injection
- GoReleaser: `-X main.version={{.Version}} -X main.buildTime={{.Date}}`
- Makefile: `-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)` (already present)
- Local dev: falls back to `"dev"`

## Acceptance Criteria
- [ ] `spotnik --version` prints the injected version string
- [ ] `const appVersion` removed from `splash.go`
- [ ] Local dev build shows `"dev"` as version
- [ ] `make ci` passes

## Tasks
- [ ] Add `var version`, `var buildTime` to `main.go`; pass to `cmd.Execute()`
      - test: `make build && ./bin/spotnik --version` outputs `"dev"`
- [ ] Update `cmd/root.go` Execute signature; set `rootCmd.Version`; forward to AppOptions
      - test: cobra --version flag works
- [ ] Add `Version`, `BuildTime` fields to `AppOptions` and `App` in `internal/app/app.go`
      - test: fields present on struct
- [ ] Remove `const appVersion`; use `a.version` in `internal/app/splash.go`
      - test: `go test ./internal/app/... -run TestRenderSplash` with injected version
- [ ] Update `internal/app/splash_test.go` assertions to use injected value
      - test: all splash tests pass
