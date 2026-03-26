---
name: project_spotnik_feature40_complete
description: Feature 40 (Theme Enhancement): 16 new tokens added to Theme interface, implemented across all 5 themes, test patterns used
type: project
---

## Feature 40 — Theme Enhancement

**What was built:**
- Extended `Theme` interface from 26 to 42 methods (added 16 new tokens)
- Implemented all 16 tokens in all 5 theme structs (black, monokai, catppuccin, nord, light)
- Added comprehensive tests: 80 exact hex value assertions + compile-time interface checks

**Key files:**
- `internal/ui/theme/theme.go` — interface definition, new methods grouped under "Gradient bars", "Visualizer", "Tables", "Status", "Per-pane borders" sections
- `internal/ui/theme/black.go` / `monokai.go` / `catppuccin.go` / `nord.go` / `light.go` — implementations
- `internal/ui/theme/theme_test.go` — tests

**Patterns established:**
- New token groups use section comments matching existing style (e.g. `// Gradient bars`)
- Interface comments include `(Feature NN)` forward-pointers to future features that will consume the token — acceptable and doesn't cause comment rot
- `var _ Theme = &XTheme{}` compile-time checks added in test file (not production) for all 5 structs
- `allMethodsReturnNonEmpty` helper in test file is the place to add coverage when interface grows

**Gotchas:**
- `gofmt` reformats function alignment (the `{` brace spacing in method signatures) — always run `gofmt -w` before `make ci` after adding multiple method stubs, not just `go fmt ./...`
- Theme struct types ARE exported (`BlackTheme`, not `blackTheme`) so the external `_test` package can use them for compile-time checks
- Per-pane border methods have very long names (`PaneBorderRecentlyPlayed`) which gofmt aligns differently — let the formatter handle it

**Testing notes:**
- Table-driven `TestNewTokens_ExactValues` with a struct holding all 16 expected values per theme is the right pattern for this type of data
- 100% coverage achieved on theme package; 84.2% overall
- The `allMethodsReturnNonEmpty` helper covers ALL 42 methods — extend it whenever the interface grows
