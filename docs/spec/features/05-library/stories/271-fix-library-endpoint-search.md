---
title: "Fix: Switch to /me/library Endpoint, Fix Search Overlay l Key"
feature: 05-library
status: open
---

## Background

Two bugs remain in the like/unlike feature after 4 iterations:

1. **403 Forbidden on like/unlike** — The API client uses the deprecated `PUT /v1/me/tracks` and `DELETE /v1/me/tracks` endpoints with JSON body `{"ids": ["<trackID>"]}`. Spotify has deprecated these entity-specific endpoints in favor of the unified `/v1/me/library` endpoint. The correct format is:

   ```
   PUT    /v1/me/library?uris=spotify:track:<id>
   DELETE /v1/me/library?uris=spotify:track:<id>
   ```

   No JSON body. Track URI goes in the `uris` query parameter. The old endpoint returns 403 because Spotify no longer supports it.

2. **Search overlay `l` key doesn't trigger** — `handleToggleLike()` at `search.go:731` checks `o.store == nil` and returns nil silently if the store isn't wired. Need to verify `SetStore` is called before the overlay is opened, and ensure the `l` keybinding works when a track result is selected.

## Design

### Bug 1: Switch to /me/library endpoint

**`internal/api/library.go`** — Rewrite `LikeTrack` and `UnlikeTrack`:

```go
// LikeTrack adds the given track to the user's library via PUT /me/library.
func (l *LibraryClient) LikeTrack(ctx context.Context, trackID string) error {
    uri := "spotify:track:" + trackID
    path := "/v1/me/library?uris=" + url.QueryEscape(uri)
    req, err := l.newRequest(ctx, http.MethodPut, path, nil)
    if err != nil {
        return fmt.Errorf("creating like track request: %w", err)
    }
    return l.doNoContent(req)
}

// UnlikeTrack removes the given track from the user's library via DELETE /me/library.
func (l *LibraryClient) UnlikeTrack(ctx context.Context, trackID string) error {
    uri := "spotify:track:" + trackID
    path := "/v1/me/library?uris=" + url.QueryEscape(uri)
    req, err := l.newRequest(ctx, http.MethodDelete, path, nil)
    if err != nil {
        return fmt.Errorf("creating unlike track request: %w", err)
    }
    return l.doNoContent(req)
}
```

Key changes:
- Endpoint: `/v1/me/tracks` → `/v1/me/library`
- Method: PUT/DELETE with JSON body → PUT/DELETE with query param `uris`
- No `Content-Type: application/json` header needed (no body)
- Track URI format: `spotify:track:<id>` (URL-encoded)
- Import `net/url` for `url.QueryEscape`

**`internal/api/library_test.go`** — Update tests to verify new endpoint format:
- Assert request URL contains `/v1/me/library?uris=spotify:track:<id>`
- Assert no JSON body in request
- Assert no `Content-Type: application/json` header

### Bug 2: Fix search overlay `l` key

**`internal/ui/panes/search.go`** — Verify `handleToggleLike` works:

The `handleToggleLike` method at line 730 already has correct logic:
1. Check `o.store == nil` → return nil
2. Check `selected == nil` → return nil
3. Check `!si.IsTrack || si.URI == ""` → return nil
4. Build `domain.Track` from `SearchListItem` fields
5. Emit `ToggleLikeRequestMsg`

The `l` key is intercepted in `handleKey` at line 635 before text input. The `resultActions()` at line 126 already shows `l like` conditionally when a track is selected.

**Root cause investigation:** `SetStore` is called at `app.go:369` during app initialization. If the store is nil when the overlay opens, `l` silently fails. Verify:
- `SetStore` is called before the overlay is first opened
- The store reference is not cleared during theme switches or overlay close/reopen

**Fix:** If `SetStore` is not being called before the overlay opens, ensure it is. If the store is correctly wired, add a debug log or test to verify the flow.

**`internal/ui/panes/search.go`** — Also ensure `resultActions()` shows `l like` only when:
- A search result item is selected
- The selected item is a track (`IsTrack == true`)
- The overlay has search results (not empty)

This is already implemented at line 126-139. Verify it works correctly.

## Files

### Modify

- `internal/api/library.go` — rewrite `LikeTrack` and `UnlikeTrack` to use `/v1/me/library?uris=...`
- `internal/api/library_test.go` — update tests for new endpoint format
- `internal/ui/panes/search.go` — verify/fix `handleToggleLike` store wiring

## Acceptance Criteria

- [ ] `LikeTrack` sends `PUT /v1/me/library?uris=spotify:track:<id>` (no JSON body)
- [ ] `UnlikeTrack` sends `DELETE /v1/me/library?uris=spotify:track:<id>` (no JSON body)
- [ ] Pressing `l` on a track no longer shows 403 "Like track failed" error
- [ ] Pressing `l` on a track successfully likes/unlikes (200 OK from Spotify)
- [ ] Pressing `l` in search overlay on a track result triggers like/unlike
- [ ] Search overlay shows `l like` border hint only when a track result is selected
- [ ] `make ci` passes

## Tasks

- [ ] Rewrite `LikeTrack` in `internal/api/library.go` — use `PUT /v1/me/library?uris=spotify:track:<id>` with `url.QueryEscape`, no JSON body
      - test: `TestLikeTrack_UsesLibraryEndpoint`, `TestLikeTrack_NoJsonBody`
- [ ] Rewrite `UnlikeTrack` in `internal/api/library.go` — use `DELETE /v1/me/library?uris=spotify:track:<id>` with `url.QueryEscape`, no JSON body
      - test: `TestUnlikeTrack_UsesLibraryEndpoint`, `TestUnlikeTrack_NoJsonBody`
- [ ] Update existing library tests for new endpoint format — verify URL contains `/me/library?uris=`, no body, no Content-Type header
      - test: update `TestLikeTrack_SendsPUT`, `TestUnlikeTrack_SendsDELETE` and related tests
- [ ] Verify search overlay `SetStore` is called before overlay opens — check `app.go:369` and ensure store is wired
      - test: `TestSearchOverlay_L_OnTrack_EmitsToggleLikeRequest` (existing, verify passes)
- [ ] Verify search overlay `resultActions()` shows `l like` conditionally — already implemented, verify with test
      - test: `TestSearchOverlay_Actions_ShowsLikeWhenTrackSelected` (existing, verify passes)
