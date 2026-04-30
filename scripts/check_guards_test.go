// Package scripts holds end-to-end tests that verify the CI guard scripts
// actually detect violations and exit non-zero. Each test creates a temporary
// directory tree with known-bad or known-good fixture files, runs the guard
// script from that directory, and asserts the expected exit code.
//
// Path resolution: go test sets the working directory to the package directory
// (scripts/). We resolve the repo root via filepath.Dir so script paths are
// always absolute.
package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot returns the absolute path to the repository root by walking up from
// this test file's location.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller should succeed")
	// thisFile == .../spotnik/scripts/check_guards_test.go
	// scripts/ is one level below repo root.
	return filepath.Dir(filepath.Dir(thisFile))
}

// scriptPath returns the absolute path to a guard script by name.
func scriptPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "scripts", name)
}

// runScript executes a bash script from dir, returning exit code and combined output.
func runScript(t *testing.T, dir, script string) (exitCode int, output string) {
	t.Helper()
	cmd := exec.Command("bash", script)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), string(out)
		}
		t.Fatalf("unexpected error running script: %v", err)
	}
	return 0, string(out)
}

// makeTree creates an internal/ and cmd/ directory tree under dir so that
// grep in the scripts does not fail with "no such file" when -r is used.
// Returns the internal/ subdir path.
func makeTree(t *testing.T, dir string) string {
	t.Helper()
	internalDir := filepath.Join(dir, "internal", "pkg")
	require.NoError(t, os.MkdirAll(internalDir, 0o755))
	cmdDir := filepath.Join(dir, "cmd")
	require.NoError(t, os.MkdirAll(cmdDir, 0o755))
	return filepath.Join(dir, "internal", "pkg")
}

// writeFile creates a file at path (parent dirs created as needed).
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// ---------------------------------------------------------------------------
// check-banned-glyphs.sh tests
// ---------------------------------------------------------------------------

// TestBannedGlyphs_Clean verifies that a clean tree (no banned glyphs) exits 0.
func TestBannedGlyphs_Clean(t *testing.T) {
	dir := t.TempDir()
	makeTree(t, dir)
	writeFile(t, filepath.Join(dir, "internal", "pkg", "clean.go"),
		"package pkg\n\nfunc Hello() string { return \"world\" }\n")

	code, out := runScript(t, dir, scriptPath(t, "check-banned-glyphs.sh"))
	assert.Equal(t, 0, code, "expected exit 0 for clean tree; output: %s", out)
	assert.Contains(t, out, "OK")
}

// TestBannedGlyphs_Violation verifies that a production .go file containing
// a banned glyph (⚠) causes a non-zero exit.
func TestBannedGlyphs_Violation(t *testing.T) {
	dir := t.TempDir()
	makeTree(t, dir)
	writeFile(t, filepath.Join(dir, "internal", "pkg", "bad.go"),
		"package pkg\n\nconst warn = \"⚠ warning\"\n")

	code, out := runScript(t, dir, scriptPath(t, "check-banned-glyphs.sh"))
	assert.NotEqual(t, 0, code, "expected non-zero exit for banned glyph; output: %s", out)
	assert.Contains(t, strings.ToUpper(out), "ERROR")
}

// TestBannedGlyphs_TestFileExempt verifies that _test.go files containing
// a banned glyph are exempt and the script exits 0.
func TestBannedGlyphs_TestFileExempt(t *testing.T) {
	dir := t.TempDir()
	makeTree(t, dir)
	writeFile(t, filepath.Join(dir, "internal", "pkg", "example_test.go"),
		"package pkg_test\n\nconst x = \"⚠\"\n")

	code, out := runScript(t, dir, scriptPath(t, "check-banned-glyphs.sh"))
	assert.Equal(t, 0, code,
		"_test.go files are exempt from banned-glyph check; output: %s", out)
}

// ---------------------------------------------------------------------------
// check-catalogue-leaks.sh tests
// ---------------------------------------------------------------------------

// TestCatalogueLeaks_Clean verifies that a clean tree exits 0.
func TestCatalogueLeaks_Clean(t *testing.T) {
	dir := t.TempDir()
	makeTree(t, dir)
	writeFile(t, filepath.Join(dir, "internal", "pkg", "clean.go"),
		"package pkg\n\nfunc Hello() string { return \"world\" }\n")

	code, out := runScript(t, dir, scriptPath(t, "check-catalogue-leaks.sh"))
	assert.Equal(t, 0, code, "expected exit 0 for clean tree; output: %s", out)
	assert.Contains(t, out, "OK")
}

// TestCatalogueLeaks_Violation verifies that a production .go file (not in the
// exempt list) containing a raw catalogue character (╭) causes a non-zero exit.
func TestCatalogueLeaks_Violation(t *testing.T) {
	dir := t.TempDir()
	makeTree(t, dir)
	// "╭" is a catalogue character but this file is NOT in the exempt list.
	writeFile(t, filepath.Join(dir, "internal", "pkg", "bad.go"),
		"package pkg\n\nconst border = \"╭\"\n")

	code, out := runScript(t, dir, scriptPath(t, "check-catalogue-leaks.sh"))
	assert.NotEqual(t, 0, code, "expected non-zero exit for catalogue leak; output: %s", out)
	assert.Contains(t, strings.ToUpper(out), "ERROR")
}

// TestCatalogueLeaks_CommentOnly verifies that a file where the catalogue
// character appears only in a comment (after //) is not flagged as a leak.
func TestCatalogueLeaks_CommentOnly(t *testing.T) {
	dir := t.TempDir()
	makeTree(t, dir)
	writeFile(t, filepath.Join(dir, "internal", "pkg", "comment.go"),
		"package pkg\n\n// uses ╭ in comments only\nfunc Foo() {}\n")

	code, out := runScript(t, dir, scriptPath(t, "check-catalogue-leaks.sh"))
	assert.Equal(t, 0, code,
		"comment-only catalogue glyph should not be flagged; output: %s", out)
}

// TestCatalogueLeaks_TestFileExempt verifies that _test.go files are exempt
// from the catalogue-leak check.
func TestCatalogueLeaks_TestFileExempt(t *testing.T) {
	dir := t.TempDir()
	makeTree(t, dir)
	writeFile(t, filepath.Join(dir, "internal", "pkg", "border_test.go"),
		"package pkg_test\n\nconst x = \"╭\"\n")

	code, out := runScript(t, dir, scriptPath(t, "check-catalogue-leaks.sh"))
	assert.Equal(t, 0, code,
		"_test.go files are exempt from catalogue-leak check; output: %s", out)
}

// TestCatalogueLeaks_SpinnerBrailleViolation verifies that a production .go
// file containing a raw braille spinner rune (⠋) causes a non-zero exit.
// Spinner braille frames are dispatched via SpinnerFrames(mode) — raw literals
// elsewhere indicate a leak outside the sanctioned dispatch path.
func TestCatalogueLeaks_SpinnerBrailleViolation(t *testing.T) {
	dir := t.TempDir()
	makeTree(t, dir)
	// "⠋" is a braille spinner frame added to the catalogue guard in story 193.
	writeFile(t, filepath.Join(dir, "internal", "pkg", "spinner.go"),
		"package pkg\n\nconst frame = \"⠋\"\n")

	code, out := runScript(t, dir, scriptPath(t, "check-catalogue-leaks.sh"))
	assert.NotEqual(t, 0, code,
		"raw braille spinner rune in production code should be flagged; output: %s", out)
	assert.Contains(t, strings.ToUpper(out), "ERROR")
}

// ---------------------------------------------------------------------------
// check-render-pane-border.sh tests
// ---------------------------------------------------------------------------

// TestRenderPaneBorder_Clean verifies that a tree with no direct
// layout.RenderPaneBorder calls exits 0.
func TestRenderPaneBorder_Clean(t *testing.T) {
	dir := t.TempDir()
	makeTree(t, dir)
	writeFile(t, filepath.Join(dir, "internal", "pkg", "clean.go"),
		"package pkg\n\nfunc Hello() string { return \"world\" }\n")

	code, out := runScript(t, dir, scriptPath(t, "check-render-pane-border.sh"))
	assert.Equal(t, 0, code, "expected exit 0 for clean tree; output: %s", out)
	assert.Contains(t, out, "OK")
}

// TestRenderPaneBorder_Violation verifies that a file outside the exempt paths
// that calls layout.RenderPaneBorder( directly triggers a non-zero exit.
func TestRenderPaneBorder_Violation(t *testing.T) {
	dir := t.TempDir()
	makeTree(t, dir)
	// Simulate a pane calling RenderPaneBorder directly (violates architecture).
	writeFile(t, filepath.Join(dir, "internal", "pkg", "bad_pane.go"),
		"package pkg\n\nimport \"layout\"\n\nfunc render() string {\n\treturn layout.RenderPaneBorder(nil)\n}\n")

	code, out := runScript(t, dir, scriptPath(t, "check-render-pane-border.sh"))
	assert.NotEqual(t, 0, code,
		"expected non-zero exit when RenderPaneBorder called outside uikit/layout; output: %s", out)
	assert.Contains(t, strings.ToUpper(out), "ERROR")
}

// TestRenderPaneBorder_CommentExempt verifies that a file referencing
// layout.RenderPaneBorder only in a doc comment is not flagged.
func TestRenderPaneBorder_CommentExempt(t *testing.T) {
	dir := t.TempDir()
	makeTree(t, dir)
	writeFile(t, filepath.Join(dir, "internal", "pkg", "doc.go"),
		"// Package pkg — wraps layout.RenderPaneBorder for convenience.\npackage pkg\n")

	code, out := runScript(t, dir, scriptPath(t, "check-render-pane-border.sh"))
	assert.Equal(t, 0, code,
		"comment-only RenderPaneBorder reference should not be flagged; output: %s", out)
}

// TestRenderPaneBorder_TestFileExempt verifies that _test.go files are exempt.
func TestRenderPaneBorder_TestFileExempt(t *testing.T) {
	dir := t.TempDir()
	makeTree(t, dir)
	writeFile(t, filepath.Join(dir, "internal", "pkg", "border_test.go"),
		"package pkg_test\n\nimport \"layout\"\n\nfunc TestFoo(t *testing.T) {\n\t_ = layout.RenderPaneBorder(nil)\n}\n")

	code, out := runScript(t, dir, scriptPath(t, "check-render-pane-border.sh"))
	assert.Equal(t, 0, code,
		"_test.go files are exempt from RenderPaneBorder check; output: %s", out)
}
