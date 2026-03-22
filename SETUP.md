# SETUP.md — First-Time Setup Guide

> This file is for the project owner. Read it once, follow it top to bottom, then you're done.
> Estimated time: 30–45 minutes.

---

## 1. Spotify Developer App

You need a Spotify app to get a `client_id`. This is required before Feature 02 (Authentication) can be tested.

1. Go to [developer.spotify.com/dashboard](https://developer.spotify.com/dashboard)
2. Log in with your Spotify account
3. Click **Create app**
4. Fill in:
   - App name: `spotnik`
   - App description: `Terminal Spotify client`
   - Redirect URI: `http://localhost:8888/callback`
   - Check: **Web API**
5. Click **Save**
6. On the app page, click **Settings** → copy your **Client ID**

> Keep this page open — you'll need the Client ID in step 6.

---

## 2. Install Go

```bash
# macOS
brew install go

# Linux
# Download from https://go.dev/dl and follow instructions

# Verify
go version   # should show go1.22 or higher
```

---

## 3. Install Dev Tools

```bash
# Linter (required — CI fails without it)
brew install golangci-lint

# Verify
golangci-lint --version
```

---

## 4. Create the GitHub Repository

1. Go to [github.com/new](https://github.com/new)
2. Name it `spotnik`
3. Set to **Private** for now (make public when ready to launch)
4. Do **not** add README, .gitignore, or license — we'll add them
5. Clone it locally:

```bash
git clone https://github.com/[yourusername]/spotnik.git
cd spotnik
```

---

## 5. Add Project Documentation

Copy all the docs from the outputs folder into your repo root:

```
spotnik/
├── CLAUDE.md
├── PRODUCT.md
├── SETUP.md          ← this file
├── Makefile
└── docs/
    ├── DESIGN.md
    ├── ARCHITECTURE.md
    └── features/
        ├── 00-overview.md
        ├── 01-auth.md
        ├── 02-playback.md
        ├── 03-playback.md
        ├── 04-search.md
        ├── 05-queue.md
        ├── 06-devices.md
        ├── 07-stats.md
        └── 08-playlists.md
```

---

## 6. Initialize the Go Module

```bash
cd spotnik

# Initialize module — replace with your actual GitHub username
go mod init github.com/initgrep-apps/spotnik
```

Now update the placeholder in two files:

- `CLAUDE.md` — find `github.com/[owner]/spotnik`, replace with your module path
- `Makefile` — find `github.com/[owner]/spotnik`, replace with your module path

---

## 7. Install Dependencies

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles
go get github.com/zalando/go-keyring
go get github.com/BurntSushi/toml
go get github.com/stretchr/testify
go get github.com/spf13/cobra

# Tidy the module
go mod tidy
```

---

## 8. Create Your Config File

```bash
mkdir -p ~/.config/spotnik
```

Create `~/.config/spotnik/config.toml`:

```toml
[spotify]
client_id = "your_client_id_from_step_1"
```

---

## 9. Add a .gitignore

Create `.gitignore` in the repo root:

```
# Binary
bin/

# Coverage
coverage.out
coverage.html

# Config (never commit credentials)
.env
*.env

# OS
.DS_Store
Thumbs.db

# IDE
.vscode/
.idea/
*.swp
```

---

## 10. First Commit

```bash
git add .
git commit -m "chore: initial project structure and documentation"
git push origin main
```

---

## 11. Verify Everything Works

```bash
# Should run (no Go files yet, but confirms tooling is set up)
go mod verify

# Should show available targets
make help
```

---

## You're Ready

Setup is complete. Your next step is to open Claude Code in the repo root and start Feature 01 (Theme System):

```bash
cd spotnik
claude
```

Give Claude Code this prompt to begin:

> "Read CLAUDE.md, PRODUCT.md, docs/ARCHITECTURE.md, and docs/features/01-theme-system.md in that order.
> Then implement Feature 01 — Theme System — working through each task in the spec in order.
> Write tests alongside each task. Do not move to the next task until the current one has passing tests."

Once Feature 01 is complete and committed, use this prompt to start Feature 02:

> "Read CLAUDE.md, PRODUCT.md, docs/ARCHITECTURE.md, and docs/features/02-auth.md in that order.
> Then implement Feature 02 — Authentication — working through each task in the spec in order.
> Write tests alongside each task. Do not move to the next task until the current one has passing tests."

---

## Quick Reference — Daily Commands

```bash
make run        # Build and run Spotnik
make test       # Run all tests
make lint       # Run linter
make coverage   # Check test coverage (must be ≥ 80%)
make ci         # Full check — run this before every commit
```

---

*Last updated: 2026-02-21*
