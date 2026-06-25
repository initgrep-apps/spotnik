package goldentest

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// TestIsUpdateMode_DefaultsFalse verifies isUpdateMode returns false when
// the -update flag is not set.
func TestIsUpdateMode_DefaultsFalse(t *testing.T) {
	// flag.Lookup("update") should find the flag registered by
	// charmbracelet/x/exp/golden, but it should default to false.
	if isUpdateMode() {
		t.Error("isUpdateMode should return false when -update is not set")
	}
}

// TestIsUpdateMode_AfterSet returns true when flag is set.
func TestIsUpdateMode_AfterSet(t *testing.T) {
	// Set the -update flag to true
	f := flag.Lookup("update")
	if f == nil {
		t.Skip("update flag not registered (requires teatest/golden dependency)")
	}
	if err := f.Value.Set("true"); err != nil {
		t.Fatalf("failed to set update flag: %v", err)
	}
	if !isUpdateMode() {
		t.Error("isUpdateMode should return true after setting -update")
	}
	// Clean up: reset to false
	f.Value.Set("false")
}

// TestAssertGolden_Match verifies that AssertGolden passes when output matches
// the golden file exactly.
func TestAssertGolden_Match(t *testing.T) {
	dir := t.TempDir()

	// Create the testdata/golden file structure
	goldenDir := filepath.Join(dir, "testdata")
	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatal(err)
	}
	goldenPath := filepath.Join(goldenDir, "TestAssertGolden_Match.golden")
	if err := os.WriteFile(goldenPath, []byte("hello world\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to temp dir so AssertGolden finds testdata/ relative to cwd
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	// Register our own -update flag since we're in an isolated test
	oldFlag := flag.Lookup("update")
	if oldFlag == nil {
		flag.Bool("update", false, "")
		defer func() {
			// Can't unregister — just leave it
		}()
	}

	// This should NOT fail — output matches golden file
	AssertGolden(t, "hello world\n")
}

// TestAssertGolden_UpdateWritesGoldenFile verifies that when -update is set,
// the golden file is written with the provided content.
func TestAssertGolden_UpdateWritesGoldenFile(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	// Ensure our -update flag is registered and set to true
	oldFlag := flag.Lookup("update")
	if oldFlag == nil {
		flag.Bool("update", true, "")
	} else {
		oldFlag.Value.Set("true")
		defer oldFlag.Value.Set("false")
	}

	AssertGolden(t, "updated content\n")

	goldenPath := filepath.Join("testdata", "TestAssertGolden_UpdateWritesGoldenFile.golden")
	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden file should have been written: %v", err)
	}
	if string(data) != "updated content\n" {
		t.Errorf("golden file content mismatch: got %q, want %q", string(data), "updated content\n")
	}
}

// TestDiffString verifies the diff helper produces output for different strings.
func TestDiffString(t *testing.T) {
	diff := diffString("hello\nworld\n", "hello\nthere\n")
	if diff == "" {
		t.Error("diffString should produce non-empty diff for different strings")
	}
}

// TestDiffString_Same returns empty for identical strings.
func TestDiffString_Same(t *testing.T) {
	diff := diffString("hello\n", "hello\n")
	if diff != "" {
		t.Errorf("diffString should return empty for identical strings, got %q", diff)
	}
}
