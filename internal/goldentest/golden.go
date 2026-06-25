// Package goldentest provides golden-file snapshot testing helpers for
// Spotnik's Bubble Tea pane models. Uses github.com/charmbracelet/x/exp/teatest
// for in-process model testing.
//
// Golden files are stored in testdata/<testname>.golden relative to the calling
// test's working directory. Use `go test -update` to regenerate all golden files.
package goldentest

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// isUpdateMode returns true when the -update flag was passed to `go test`.
// The -update flag may be registered by either this package or by
// github.com/charmbracelet/x/exp/golden (imported transitively via teatest).
func isUpdateMode() bool {
	f := flag.Lookup("update")
	if f == nil {
		return false
	}
	getter, ok := f.Value.(flag.Getter)
	if !ok {
		return false
	}
	v, ok := getter.Get().(bool)
	return ok && v
}

// NewPaneTest creates a teatest.TestModel for a Bubble Tea pane model with
// the given terminal dimensions. This is a convenience wrapper that sets up
// the initial terminal size so the pane renders at the correct dimensions.
func NewPaneTest(t *testing.T, model tea.Model, width, height int) *teatest.TestModel {
	t.Helper()
	return teatest.NewTestModel(t, model,
		teatest.WithInitialTermSize(width, height),
	)
}

// AssertGolden compares got against a golden file stored in testdata/.
// The golden file is named testdata/<t.Name()>.golden.
//
// When -update is true, the golden file is written (or overwritten) with got.
// When -update is false and the golden file is missing, the test is failed.
// When -update is false and content differs, the diff is printed and the test
// is marked as failed.
func AssertGolden(t *testing.T, got string) {
	t.Helper()

	name := filepath.Join("testdata", t.Name()+".golden")

	if isUpdateMode() {
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatalf("goldentest: creating testdata dir: %v", err)
		}
		if err := os.WriteFile(name, []byte(got), 0644); err != nil {
			t.Fatalf("goldentest: writing golden file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("golden file missing: %s (run `go test -update` to generate)", name)
	}

	if got != string(want) {
		t.Errorf("golden mismatch (-want +got):\n%s", diffString(string(want), got))
	}
}

// ReadOutput reads all rendered output from a teatest.TestModel after the
// underlying program has finished. The caller must call tm.Quit() and then
// tm.WaitFinished() before calling ReadOutput to stop the program and flush
// all output.
func ReadOutput(tm *teatest.TestModel) string {
	out, _ := io.ReadAll(tm.Output())
	return string(out)
}

// WaitAndReadOutput quits the TestModel's program, waits for it to finish,
// and returns the complete rendered output. Use this for golden snapshots
// that need to capture the initial render.
func WaitAndReadOutput(t *testing.T, tm *teatest.TestModel) string {
	t.Helper()
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
	return ReadOutput(tm)
}

// diffString returns a simple unified diff between two strings.
// Returns empty string when the strings are identical.
func diffString(want, got string) string {
	if want == got {
		return ""
	}

	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")

	maxLen := len(wantLines)
	if len(gotLines) > maxLen {
		maxLen = len(gotLines)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("--- want (%d lines)\n", len(wantLines)))
	b.WriteString(fmt.Sprintf("+++ got  (%d lines)\n", len(gotLines)))

	diffCount := 0
	maxDiffs := 20

	for i := 0; i < maxLen && diffCount < maxDiffs; i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			diffCount++
			fmt.Fprintf(&b, "@@ line %d @@\n", i+1)
			if w != "" {
				fmt.Fprintf(&b, "-%s\n", w)
			}
			if g != "" {
				fmt.Fprintf(&b, "+%s\n", g)
			}
		}
	}
	if diffCount >= maxDiffs {
		b.WriteString("... (truncated, showing first 20 diffs)\n")
	}
	return b.String()
}
