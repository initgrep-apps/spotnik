---
title: "Fix: Search pagination page size, indicator, and count mismatches"
feature: 18-search-redesign
status: open
---

## Background

Three related pagination display issues visible in manual testing:

1. **Bubble-table shows misleading "1/1"** — bubble-table's native page indicator
   is based on the local buffer size. Initially only 10 items are fetched, and with
   a tall terminal the `pageSize` (view height - 6) can be 20+, so 10 items fit in
   one page → "1/1". This misleads users into thinking there's only one page.

2. **Prefetched items appear on the same page** — when scrolling triggers a prefetch,
   `refreshRows()` adds the new items to the table. Since `pageSize` (set by view
   height) is larger than `maxResultsPerSection` (10), all 20 items appear on the
   same page. The user sees items jump from 9 to 20 in a single screen instead of
   clean page transitions.

3. **Count mismatches** — tab bar shows "Albums 13", bottom indicator shows
   "11-13 of 13", but the screen displays 20 rows. The custom `pageIndicator()`
   computes ranges based on cursor position and `maxResultsPerSection`, while
   bubble-table pages by view height. They disagree on what a "page" is.

All three stem from the same root cause: `components.Table.SetSize` sets `pageSize`
based on the terminal height, but search should page by API page size (10) so
pagination feels consistent with the tab bar counts and the API boundary.

## Design

### Cap search table page size at `maxResultsPerSection`

The fix is to cap the page size passed to `components.Table` so it never exceeds
`maxResultsPerSection`. This makes the table paginate at 10-item boundaries
regardless of terminal height.

Add a `MaxPageSize` field to `components.TableConfig`:

```go
// components/table.go — add to TableConfig
MaxPageSize int // optional; caps WithPageSize to this value when > 0
```

In `rebuild()` and `SetSize()`, after computing `pageSize`:

```go
if t.config.MaxPageSize > 0 && pageSize > t.config.MaxPageSize {
    pageSize = t.config.MaxPageSize
}
```

In `SearchOverlay.buildAllTables`, set `MaxPageSize: maxResultsPerSection` for
all 4 tables.

### Hide bubble-table's native page indicator

Bubble-table renders "1/2" at the bottom of its view. With the capped page size,
this indicator becomes accurate (e.g., "2/3" for 20 items with pageSize=10). But
the search overlay already has a custom `pageIndicator()` in the help bar showing
"11-20 of 39". Showing both is redundant and confusing.

Check if `evertras/bubble-table` supports disabling the native page indicator. If
so, disable it for search tables. If not, the indicator is part of the table View()
output — accept it as supplementary information since it will now be accurate.

### Fix `pageIndicator()` range calculation

With capped page size, the custom indicator should align with what the table
actually shows. The current formula uses `maxResultsPerSection` as page size,
which now matches the table's page size. Verify the formula produces correct
ranges:

- Page 1 (cursor 0-9): "1-10 of 39"
- Page 2 (cursor 10-19): "11-20 of 39"
- Last page (cursor 30-38): "31-39 of 39"

The current formula at search.go:897-905 should work correctly once page sizes match.
The only edge case: `end = min(pageStart + pageSize, bufLen)` — when the buffer
hasn't loaded all pages yet, `bufLen` may be less than `total`. This is correct
because we can't show items we haven't fetched.

### Files Changed

| Action | File | Purpose |
|--------|------|---------|
| Modify | `internal/ui/components/table.go` | Add `MaxPageSize` to `TableConfig`, apply cap |
| Modify | `internal/ui/components/table_test.go` | Test `MaxPageSize` capping |
| Modify | `internal/ui/panes/search.go` | Set `MaxPageSize: maxResultsPerSection` on all tables |
| Modify | `internal/ui/panes/search_test.go` | Verify pagination at 10-item boundaries |

## Acceptance Criteria

- [ ] Search tables page at 10-item boundaries regardless of terminal height
- [ ] Scrolling past item 10 shows page 2 (items 11-20), not all 20 on one screen
- [ ] Bubble-table page indicator reads "1/2" (accurate) or is hidden
- [ ] Custom page indicator "1-10 of N" matches the visible rows
- [ ] Tab bar count, page indicator, and visible row count are consistent
- [ ] Other panes using `components.Table` are unaffected (`MaxPageSize` defaults to 0)
- [ ] `make ci` passes

## Tasks

- [ ] **Add `MaxPageSize` to `components.Table`** — add `MaxPageSize int` field to
      `TableConfig`. In `rebuild()` and `SetSize()`, after computing `pageSize`,
      cap it: `if config.MaxPageSize > 0 && pageSize > config.MaxPageSize { pageSize = config.MaxPageSize }`.
      In `internal/ui/components/table.go`.
      - test: `TestTable_MaxPageSize_Caps` — verify page size capped at MaxPageSize
      - test: `TestTable_MaxPageSize_Zero_NoCap` — verify default behavior unchanged

- [ ] **Set MaxPageSize on search tables** — in `buildAllTables`, add
      `MaxPageSize: maxResultsPerSection` to all 4 `TableConfig` structs.
      In `internal/ui/panes/search.go`.
      - test: `TestSearchOverlay_TablePageSize_CappedAt10` — verify tables page at 10

- [ ] **Verify page indicator accuracy** — with capped page size, confirm
      `pageIndicator()` produces correct ranges. Adjust if needed.
      In `internal/ui/panes/search.go`.
      - test: `TestPageIndicator_Page1_With20Items` — "1-10 of 20"
      - test: `TestPageIndicator_Page2_With20Items` — "11-20 of 20"
      - test: `TestPageIndicator_LastPage_Partial` — "11-13 of 13"
