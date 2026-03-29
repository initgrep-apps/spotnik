# Feature 73: Playlist Gateway Fix

## Problem

After commit `4f904f0` wired the API gateway into the pre-auth startup path,
two playlist regressions appeared:

1. **Track counts always 0** — all playlists display `0` in the Tracks column
2. **Playlist tracks fail to load** — pressing Enter shows toast:
   "Failed to load playlist tracks. Press Enter to retry"

## Root Causes

### 1. RequestKey ignores query parameters (dedup collision)

The gateway dedup key uses only `Method + Path`, ignoring query parameters:

```go
key := RequestKey{Method: req.Method, Path: req.URL.Path}
```

Two requests to the same path with different query strings (e.g. different
`offset` values) share one dedup key. If they overlap, the second caller
receives the first caller's cached response — wrong data.

### 2. User-triggered fetches use Background priority

`buildFetchPlaylistTracksCmd` and `buildFetchPlaylistsCmd` use
`context.Background()` which defaults to `Background` priority. During
gateway backoff (from any 429), Background requests are immediately
rejected. User-triggered actions should use `Interactive` priority.

## Tasks

### T1 — Gateway integration tests for playlists

Add tests in `internal/api/library_test.go`:

- `TestGetPlaylists_WithGateway_PreservesTrackCount`: exercise `Playlists()`
  through gateway, assert `TrackCount == 42`
- `TestGetPlaylistTracks_WithGateway_ReturnsTracks`: exercise
  `PlaylistTracks()` through gateway, assert tracks returned

### T2 — Include query parameters in dedup key

- Add `RawQuery` field to `RequestKey` in `gateway.go`
- Update key construction in `doJSON`, `doJSONOptional`, `doNoContent`
  (3 sites in `base.go`)
- Add `TestGateway_DedupKeyIncludesQueryParams` test

### T3 — Use Interactive priority for user-triggered fetches

In `commands.go`:

- `buildFetchPlaylistTracksCmd`: wrap context with `api.Interactive`
- `buildFetchPlaylistsCmd`: wrap context with `api.Interactive`

## Files

| File | Change |
|------|--------|
| `internal/api/gateway.go` | Add `RawQuery` to `RequestKey` |
| `internal/api/base.go` | Include `RawQuery` in key construction |
| `internal/api/library_test.go` | Gateway integration tests |
| `internal/api/gateway_test.go` | Dedup query param test |
| `internal/app/commands.go` | Interactive priority for user fetches |
