# CI/CD, Release Pipeline, and README Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Set up automated CI/CD, tag-based semver releases via GoReleaser + release-please, multi-platform distribution (Homebrew, Scoop, DEB, RPM), a proper README, and version injection — so every tag push ships binaries everywhere automatically.

**Architecture:** Three GitHub Actions workflows: CI (lint+test on every push/PR), release-please (manages Release PRs + changelog on push to main), and release (GoReleaser on tag push). Version flows from git tags → LDFLAGS → `main.go` → `cmd` → `app` → splash screen. GoReleaser cross-compiles for 5 platforms and pushes to Homebrew tap + Scoop bucket repos.

**Tech Stack:** GitHub Actions, GoReleaser, release-please, Go LDFLAGS injection

**Spec:** `docs/superpowers/specs/2026-03-26-cicd-release-readme-design.md`

---

### Task 1: Version Injection — Wire version from main.go through to splash

**Files:**
- Modify: `main.go`
- Modify: `cmd/root.go:36-41` (Execute function)
- Modify: `internal/app/app.go:164-171` (AppOptions struct)
- Modify: `internal/app/splash.go:9-10,14,28` (remove const, add param)
- Modify: `internal/app/splash_test.go` (update assertions)
- Modify: `internal/app/render.go:180` (pass version to renderSplashView)

- [ ] **Step 1: Add version vars to main.go**

Replace the contents of `main.go` with:

```go
// Spotnik — a terminal Spotify client for developers.
// This file is the entry point only — no logic lives here.
package main

import "github.com/initgrep-apps/spotnik/cmd"

// version and buildTime are injected at build time via LDFLAGS:
//   -X main.version=<tag> -X main.buildTime=<timestamp>
var (
	version   = "dev"
	buildTime = ""
)

func main() {
	cmd.Execute(version, buildTime)
}
```

- [ ] **Step 2: Update cmd.Execute() to accept and forward version**

In `cmd/root.go`, change the `Execute` function:

```go
// Execute is the entry point called from main.go.
func Execute(version, buildTime string) {
	rootCmd.Version = version
	appVersion = version
	appBuildTime = buildTime
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

Add package-level vars at the top of `cmd/root.go` (after the `rootCmd` var):

```go
// appVersion and appBuildTime are set by Execute() from main.go LDFLAGS.
var (
	appVersion   = "dev"
	appBuildTime = ""
)
```

In `runApp`, update the `opts` construction to include version:

```go
opts := app.AppOptions{
    NeedsAuth:  needsAuth,
    ClientID:   cfg.ClientID,
    TokenStore: store,
    Version:    appVersion,
    BuildTime:  appBuildTime,
}
```

- [ ] **Step 3: Add Version and BuildTime to AppOptions**

In `internal/app/app.go`, add fields to `AppOptions`:

```go
type AppOptions struct {
	NeedsAuth  bool
	ClientID   string
	TokenStore keychain.TokenStore
	// TokenBaseURL overrides the Spotify token endpoint for tests.
	// Leave empty for production (uses the real Spotify endpoint).
	TokenBaseURL string
	// Version is the build version string (e.g. "v0.1.0" or "dev").
	Version string
	// BuildTime is the UTC build timestamp.
	BuildTime string
}
```

Add a `version` field to the `App` struct (after `tokenBaseURL string` on line 123):

```go
// version is the build version displayed on the splash screen.
version string
```

In the `New()` function, compute version before the return statement (after `volStep` logic, before `return &App{`):

```go
version := opts.Version
if version == "" {
    version = "dev"
}
```

Then add `version: version,` to the `return &App{...}` struct literal, after `tokenBaseURL`:

```go
return &App{
    // ... existing fields ...
    tokenBaseURL:    opts.TokenBaseURL,
    version:         version,
    lastInteraction: time.Now(),
    idleThreshold:   idleThresholdSecs * time.Second,
}
```

- [ ] **Step 4: Update splash.go — remove const, add version parameter**

Replace `internal/app/splash.go` entirely:

```go
package app

import (
	"github.com/charmbracelet/lipgloss"
	figure "github.com/common-nighthawk/go-figure"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// renderSplashView builds the splash screen using go-figure ASCII art.
// It is a standalone function so it can be tested without an App instance.
func renderSplashView(t theme.Theme, version string, width, height int) string {
	fig := figure.NewFigure("SPOTNIK", "doom", false)
	banner := fig.String()

	bannerStyle := lipgloss.NewStyle().
		Foreground(t.ActiveBorder()).
		Bold(true)

	tagline := lipgloss.NewStyle().
		Foreground(t.TextMuted()).
		Render("A terminal Spotify client for developers")

	versionText := lipgloss.NewStyle().
		Foreground(t.TextMuted()).
		Render(version)

	content := lipgloss.JoinVertical(lipgloss.Center,
		bannerStyle.Render(banner),
		"",
		tagline,
		versionText,
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
```

- [ ] **Step 5: Update render.go to pass version**

In `internal/app/render.go:180`, change:

```go
return renderSplashView(a.theme, a.width, a.height)
```

to:

```go
return renderSplashView(a.theme, a.version, a.width, a.height)
```

- [ ] **Step 6: Update splash_test.go**

Replace `internal/app/splash_test.go`:

```go
package app

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestRenderSplash_ContainsBranding(t *testing.T) {
	th := theme.Load("black")
	view := renderSplashView(th, "v0.1.0", 120, 40)

	// go-figure "doom" font renders letters as ASCII art, so we check for
	// recognizable fragments from the rendered output.
	assert.Contains(t, view, "___", "splash should contain go-figure ASCII art")
	assert.Contains(t, view, "v0.1.0", "splash should contain the version")
	assert.Contains(t, view, "terminal Spotify client", "splash should contain the tagline")
}

func TestRenderSplash_SmallTerminal(t *testing.T) {
	th := theme.Load("black")
	// Even with a small terminal, renderSplashView should not panic.
	view := renderSplashView(th, "dev", 40, 10)
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "dev")
}
```

- [ ] **Step 7: Run tests to verify**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./internal/app/... -run TestRenderSplash -v`

Expected: Both tests PASS.

- [ ] **Step 8: Run full test suite**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && go test ./... -count=1`

Expected: All tests PASS (no other code references `appVersion` const).

- [ ] **Step 9: Verify build with LDFLAGS**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && make build`

Expected: Binary builds successfully. The Makefile already injects `-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)`.

- [ ] **Step 10: Commit**

```bash
git add main.go cmd/root.go internal/app/app.go internal/app/splash.go internal/app/splash_test.go internal/app/render.go
git commit -m "feat(version): inject version from LDFLAGS through to splash screen

Remove hardcoded const appVersion. Version now flows:
main.go (LDFLAGS) → cmd.Execute() → AppOptions → App.version → splash.

Enables 'spotnik --version' via rootCmd.Version."
```

---

### Task 2: GoReleaser Configuration

**Files:**
- Create: `.goreleaser.yml`

- [ ] **Step 1: Create .goreleaser.yml**

Create `.goreleaser.yml` in project root:

```yaml
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - main: .
    binary: spotnik
    env:
      - CGO_ENABLED=0
    flags:
      - -trimpath
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.buildTime={{.Date}}
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64

archives:
  - format: tar.gz
    name_template: >-
      spotnik_{{.Version}}_{{.Os}}_{{.Arch}}
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: checksums.txt
  algorithm: sha256

changelog:
  skip: true

brews:
  - repository:
      owner: initgrep-apps
      name: homebrew-tap
      token: "{{ .Env.RELEASE_PAT }}"
    name: spotnik
    homepage: https://github.com/initgrep-apps/spotnik
    description: Terminal Spotify client for developers
    license: MIT
    install: |
      bin.install "spotnik"
    test: |
      system "#{bin}/spotnik", "--version"

scoops:
  - repository:
      owner: initgrep-apps
      name: scoop-bucket
      token: "{{ .Env.RELEASE_PAT }}"
    name: spotnik
    homepage: https://github.com/initgrep-apps/spotnik
    description: Terminal Spotify client for developers
    license: MIT

nfpms:
  - package_name: spotnik
    vendor: initgrep-apps
    homepage: https://github.com/initgrep-apps/spotnik
    maintainer: irshad.mike@gmail.com
    description: Terminal Spotify client for developers
    license: MIT
    formats:
      - deb
      - rpm
    bindir: /usr/bin
```

- [ ] **Step 2: Validate config syntax**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && cat .goreleaser.yml | head -5`

Expected: File exists and starts with the schema comment.

- [ ] **Step 3: Commit**

```bash
git add .goreleaser.yml
git commit -m "chore(release): add GoReleaser configuration

Cross-compiles for linux/darwin (amd64+arm64) and windows (amd64).
Publishes to Homebrew tap, Scoop bucket, and DEB/RPM packages."
```

---

### Task 3: GitHub Actions — CI Workflow

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create the CI workflow**

```bash
mkdir -p .github/workflows
```

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: ["**"]
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  ci:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Install golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
          args: --timeout=5m

      - name: Run CI checks
        run: make fmt-check tidy-check test-coverage build
```

Note: `golangci-lint-action` both installs and runs the linter, so we skip `make lint` and run the remaining `ci` targets directly.

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add GitHub Actions CI workflow

Runs fmt-check, tidy-check, lint, test-coverage, and build on every
push and PR. Uses golangci-lint-action for linter installation."
```

---

### Task 4: GitHub Actions — Release Workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Create the release workflow**

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          RELEASE_PAT: ${{ secrets.RELEASE_PAT }}
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add GitHub Actions release workflow

Triggered on v* tag push. Runs GoReleaser to cross-compile, create
GitHub Release, push Homebrew formula, Scoop manifest, and DEB/RPM."
```

---

### Task 5: GitHub Actions — release-please Workflow + Config

**Files:**
- Create: `.github/workflows/release-please.yml`
- Create: `.release-please-manifest.json`
- Create: `release-please-config.json`

- [ ] **Step 1: Create the release-please workflow**

Create `.github/workflows/release-please.yml`:

```yaml
name: Release Please

on:
  push:
    branches: [main]

permissions:
  contents: write
  pull-requests: write

jobs:
  release-please:
    runs-on: ubuntu-latest
    steps:
      - uses: googleapis/release-please-action@v4
        with:
          config-file: release-please-config.json
          manifest-file: .release-please-manifest.json
```

- [ ] **Step 2: Create .release-please-manifest.json**

Create `.release-please-manifest.json`:

```json
{
  ".": "0.0.0"
}
```

- [ ] **Step 3: Create release-please-config.json**

Create `release-please-config.json`:

```json
{
  "packages": {
    ".": {
      "release-type": "go",
      "bump-minor-pre-major": true,
      "bump-patch-for-minor-pre-major": true,
      "changelog-sections": [
        { "type": "feat", "section": "Features" },
        { "type": "fix", "section": "Bug Fixes" },
        { "type": "refactor", "section": "Refactoring" },
        { "type": "test", "section": "Tests" },
        { "type": "chore", "section": "Miscellaneous" }
      ]
    }
  }
}
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/release-please.yml .release-please-manifest.json release-please-config.json
git commit -m "ci: add release-please workflow and configuration

Analyzes Conventional Commits on push to main. Creates/updates a
Release PR with version bump and CHANGELOG. Merging the PR creates
a v* tag which triggers the release workflow."
```

---

### Task 6: LICENSE File

**Files:**
- Create: `LICENSE`

- [ ] **Step 1: Create MIT LICENSE**

Create `LICENSE` with the standard MIT license text:

```
MIT License

Copyright (c) 2026 initgrep-apps

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 2: Commit**

```bash
git add LICENSE
git commit -m "chore: add MIT license"
```

---

### Task 7: README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Write the full README**

Replace `README.md` with:

```markdown
# Spotnik

A keyboard-driven terminal Spotify client for developers who live in the terminal.

[![CI](https://github.com/initgrep-apps/spotnik/actions/workflows/ci.yml/badge.svg)](https://github.com/initgrep-apps/spotnik/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/initgrep-apps/spotnik)](https://github.com/initgrep-apps/spotnik/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/initgrep-apps/spotnik)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

<!-- TODO: Add terminal screenshot or GIF demo here -->

## Installation

### Homebrew (macOS / Linux)

```bash
brew install initgrep-apps/tap/spotnik
```

### Scoop (Windows)

```powershell
scoop bucket add spotnik https://github.com/initgrep-apps/scoop-bucket
scoop install spotnik
```

### DEB (Ubuntu / Debian)

Download the `.deb` from the [latest release](https://github.com/initgrep-apps/spotnik/releases/latest), then:

```bash
sudo dpkg -i spotnik_*_linux_amd64.deb
```

### RPM (Fedora / RHEL)

Download the `.rpm` from the [latest release](https://github.com/initgrep-apps/spotnik/releases/latest), then:

```bash
sudo rpm -i spotnik_*_linux_amd64.rpm
```

### Go install

```bash
go install github.com/initgrep-apps/spotnik@latest
```

### Binary download

Pre-built binaries for Linux, macOS, and Windows are available on the
[Releases page](https://github.com/initgrep-apps/spotnik/releases/latest).

## Prerequisites

- **Spotify Premium** account (required for playback control)
- A **Spotify Developer App** — create one at [developer.spotify.com/dashboard](https://developer.spotify.com/dashboard)

### Setup

1. Create a Spotify app at the developer dashboard
2. Note your **Client ID**
3. Create the config file:

```bash
mkdir -p ~/.config/spotnik
cat > ~/.config/spotnik/config.toml << 'EOF'
[spotify]
client_id = "your-client-id-here"
EOF
```

## Quick Start

```bash
spotnik
```

On first run, Spotnik opens your browser for Spotify authorization (PKCE flow — no client secret needed). After auth completes, you're in.

## Keybindings

| Key | Action |
|-----|--------|
| `Space` | Play / Pause |
| `>` / `.` | Next track |
| `<` / `,` | Previous track |
| `+` / `-` | Volume up / down |
| `Tab` / `Shift+Tab` | Navigate panes |
| `j` / `k` | Scroll down / up |
| `Enter` | Select / play item |
| `/` | Open search |
| `A` (Shift+a) | Add to queue |
| `d` | Switch device |
| `0` | Toggle Page A / Page B |
| `p` | Cycle layout preset |
| `f` | Filter in focused pane |
| `?` | Help |
| `q` | Quit |

See [`docs/DESIGN.md`](docs/DESIGN.md) for the full keybinding table.

## Configuration

Config file: `~/.config/spotnik/config.toml`

```toml
[spotify]
client_id = "your-client-id"

[ui]
theme = "black"   # Options: black, dark, light, solarized, nord
```

## Building from Source

Requires Go as specified in [`go.mod`](go.mod) (currently 1.26+).

```bash
git clone https://github.com/initgrep-apps/spotnik.git
cd spotnik
make build
./bin/spotnik
```

### Available Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build for current platform |
| `make run` | Build and run |
| `make test` | Run all tests |
| `make lint` | Run linter |
| `make ci` | Full CI check (lint + test + build) |

## Contributing

1. Fork the repo and create a feature branch: `git checkout -b feat/my-feature`
2. Use [Conventional Commits](https://www.conventionalcommits.org/): `feat(scope): description`
3. Run `make ci` before pushing — it must pass
4. Open a PR against `main`

## License

[MIT](LICENSE)
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add comprehensive README with install instructions

Covers all 6 installation methods (Homebrew, Scoop, DEB, RPM, go install,
binary), prerequisites, quick start, keybindings, config, and contributing."
```

---

### Task 8: Cleanup — Remove Versioning Table from Overview

**Files:**
- Modify: `docs/features/00-overview.md:69-88`

- [ ] **Step 1: Remove the versioning section**

Delete the entire `## Versioning` section from `docs/features/00-overview.md` (lines 69-88), including the trailing `---` separator and the `*Last updated*` line. Replace with just:

```markdown
---

*Last updated: 2026-03-26*
```

The features table above stays unchanged.

- [ ] **Step 2: Commit**

```bash
git add docs/features/00-overview.md
git commit -m "chore: remove static versioning table from feature overview

Versioning is now tag-based semver managed by release-please.
The static version-to-feature mapping is no longer needed."
```

---

### Task 9: Full CI Validation

- [ ] **Step 1: Run make ci**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && make ci`

Expected: All checks pass — fmt-check, tidy-check, lint, test-coverage (≥80%), build.

- [ ] **Step 2: Verify spotnik --version works**

Run: `cd /Users/irshadsheikh/dev/github/apps/spotnik && ./bin/spotnik --version`

Expected: Prints the version string (likely `spotnik version dev` or the git-describe output).

- [ ] **Step 3: Fix any issues found**

If any step fails, fix and amend the relevant commit.

---

### Task 10: Manual GitHub Setup (Owner Action)

These steps must be done by the repo owner in the browser. They are NOT automatable.

- [ ] **Step 1: Create `initgrep-apps/homebrew-tap` repo**

Go to GitHub → New repository under `initgrep-apps` org.
Name: `homebrew-tap`. Public. Initialize with README.

- [ ] **Step 2: Create `initgrep-apps/scoop-bucket` repo**

Same as above. Name: `scoop-bucket`. Public. Initialize with README.

- [ ] **Step 3: Create a fine-grained PAT**

GitHub Settings → Developer settings → Personal access tokens → Fine-grained tokens.
- Scoped to `initgrep-apps` org
- Repository access: `homebrew-tap` and `scoop-bucket`
- Permissions: Contents → Read and write

- [ ] **Step 4: Add RELEASE_PAT secret to spotnik repo**

Go to `initgrep-apps/spotnik` → Settings → Secrets and variables → Actions.
New repository secret: `RELEASE_PAT` = the PAT from step 3.

- [ ] **Step 5: Verify by pushing to main**

Once all code is merged to main, release-please should create the first Release PR.
Merging that PR creates tag `v0.1.0` → triggers GoReleaser → binaries ship.
