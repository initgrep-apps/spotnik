package testhelpers_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadFixture_ReturnsNonEmptyContent verifies that LoadFixture successfully
// reads a known fixture from testdata/fixtures/ and returns non-empty content.
//
// Note: the error path in LoadFixture (os.ReadFile failure) calls t.Fatal via
// require.NoError, which terminates the test immediately. That branch cannot
// be exercised without a fake *testing.T; it is documented here rather than
// tested to avoid testing private/internal framework internals.
func TestLoadFixture_ReturnsNonEmptyContent(t *testing.T) {
	// playback_state.json is a stable fixture guaranteed to exist in testdata/fixtures/.
	data := testhelpers.LoadFixture(t, "playback_state.json")
	require.NotEmpty(t, data, "fixture file should not be empty")
	assert.Contains(t, string(data), "{", "fixture content should be valid JSON")
}
