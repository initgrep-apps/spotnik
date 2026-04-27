---
name: project_spotnik_feature40_complete
description: Feature 40 (Theme Enhancement): 16 new tokens added to Theme interface, implemented across all 5 themes, test patterns used
type: project
---

## Feature 40 — Theme Enhancement

**Built:**
- Extended `Theme` interface 26→42 methods (+16 tokens)
- Implemented 16 tokens across 5 theme structs (black, monokai, catppuccin, nord, light)
- Tests: 80 exact hex assertions + compile-time interface checks

**Key files:**
- `internal/ui/theme/theme.go` — interface def, new methods grouped under "Gradient bars", "Visualizer", "Tables", "Status", "Per-pane borders"
- `internal/ui/theme/black.go` / `monokai.go` / `catppuccin.go` / `nord.go` / `light.go` — implementations
- `internal/ui/theme/theme_test.go` — tests

**Patterns:**
- New token groups: section comments match existing style (e.g. `// Gradient bars`)
- Interface comments include `(Feature NN)` forward-pointers to future consumers — OK, no rot
- `var _ Theme = &XTheme{}` compile-time checks in test file (not prod) for all 5 structs
- `allMethodsReturnNonEmpty` helper in test file = place to add coverage when interface grows

**Gotchas:**
- `gofmt` reformats `{` brace spacing in method signatures — run `gofmt -w` before `make ci` after stub batches, not just `go fmt ./...`
- Theme structs exported (`BlackTheme`, not `blackTheme`) so external `_test` package uses them for compile-time checks
- Per-pane border methods have long names (`PaneBorderRecentlyPlayed`) — gofmt aligns differently, let formatter handle

**Testing:**
- Table-driven `TestNewTokens_ExactValues` with struct holding 16 expected values per theme = right pattern
- 100% coverage on theme package; 84.2% overall
- `allMethodsReturnNonEmpty` covers all 42 methods — extend whenever interface grows