package api

import "context"

// fetchAll fetches all pages of a paginated endpoint.
//
// fetchPage is called for each page and returns (items, total, error) for a
// given offset. total is the total number of items available on the server.
//
// maxItems is a safety cap to prevent runaway loops — fetchAll will stop
// collecting items once len(accumulated) >= maxItems.
//
// Returns the accumulated items from all pages, or the first error encountered.
func fetchAll[T any](ctx context.Context, maxItems int, fetchPage func(ctx context.Context, offset int) ([]T, int, error)) ([]T, error) {
	var all []T
	offset := 0

	for {
		items, total, err := fetchPage(ctx, offset)
		if err != nil {
			return nil, err
		}

		all = append(all, items...)
		offset += len(items)

		// Stop when: no items returned, all items collected, or safety cap reached.
		if len(items) == 0 || offset >= total || offset >= maxItems {
			break
		}
	}

	return all, nil
}
