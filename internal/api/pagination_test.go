package api

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchAll_ThreePages(t *testing.T) {
	// Simulate a 3-page response: page sizes 3, 3, 2 (total 8 items).
	calls := 0
	fetchPage := func(_ context.Context, offset int) ([]string, int, error) {
		calls++
		all := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
		total := len(all)
		pageSize := 3
		end := offset + pageSize
		if end > total {
			end = total
		}
		return all[offset:end], total, nil
	}

	items, err := fetchAll[string](context.Background(), 100, fetchPage)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c", "d", "e", "f", "g", "h"}, items)
	assert.Equal(t, 3, calls, "expected 3 page fetches")
}

func TestFetchAll_StopsAtMaxItems(t *testing.T) {
	// Total is 10, but maxItems is 5. Pages are size 3.
	// After page 2 (offset=6 >= maxItems=5), fetching stops — items 7-10 are not fetched.
	calls := 0
	fetchPage := func(_ context.Context, offset int) ([]int, int, error) {
		calls++
		all := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		total := len(all)
		pageSize := 3
		end := offset + pageSize
		if end > total {
			end = total
		}
		return all[offset:end], total, nil
	}

	items, err := fetchAll[int](context.Background(), 5, fetchPage)
	require.NoError(t, err)
	// maxItems=5 cap stops after page 2 (offset becomes 6 >= 5), so items 7-10 are not fetched.
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, items)
	assert.Equal(t, 2, calls, "expected only 2 page fetches before cap stops iteration")
}

func TestFetchAll_EmptyFirstPage(t *testing.T) {
	fetchPage := func(_ context.Context, offset int) ([]string, int, error) {
		return []string{}, 0, nil
	}

	items, err := fetchAll[string](context.Background(), 100, fetchPage)
	require.NoError(t, err)
	assert.Empty(t, items, "expected empty result for empty first page")
}

func TestFetchAll_PropagatesError(t *testing.T) {
	expectedErr := errors.New("fetch failed")
	fetchPage := func(_ context.Context, offset int) ([]string, int, error) {
		if offset == 0 {
			return []string{"a", "b"}, 6, nil
		}
		return nil, 0, expectedErr
	}

	_, err := fetchAll[string](context.Background(), 100, fetchPage)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestFetchAll_SinglePage(t *testing.T) {
	fetchPage := func(_ context.Context, offset int) ([]string, int, error) {
		return []string{"x", "y"}, 2, nil
	}

	items, err := fetchAll[string](context.Background(), 100, fetchPage)
	require.NoError(t, err)
	assert.Equal(t, []string{"x", "y"}, items)
}
