// Package testhelpers provides shared test utilities for the Spotnik test suite.
package testhelpers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// fixturesDir is resolved once at package init time relative to this source file,
// making it independent of the caller's working directory or depth from the project root.
var fixturesDir = func() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("testhelpers: cannot determine file location via runtime.Caller")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(root, "testdata", "fixtures")
}()

// LoadFixture reads a JSON fixture from testdata/fixtures/ by name.
// Fails the test immediately if the file cannot be read.
func LoadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(fixturesDir, name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "testhelpers.LoadFixture: failed to read %s", path)
	return data
}
