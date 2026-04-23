// Package cliout renders styled CLI output for the spotnik command-line interface.
//
// The package defines a small taxonomy of message types (Header, Step, KV, Steps,
// Hint, URL, Paragraph, Spinner, Prompt). Callers build []Message values or use
// the fluent Builder and call Write / WriteInline. Rendering is palette-aware and
// honours NO_COLOR and TTY detection automatically.
//
// Before adding a new message type or glyph, read docs/CLI-OUTPUT.md — it is the
// canonical reference for the project's CLI output conventions.
package cliout
