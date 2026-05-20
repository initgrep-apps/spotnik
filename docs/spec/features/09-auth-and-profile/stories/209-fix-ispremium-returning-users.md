---
title: "Fix: IsPremium false positive for returning users"
feature: 09-auth-and-profile
status: open
---

## Background

Returning users (saved token, no OAuth needed) see "Spotify Premium required"
toast when selecting a device, adding to queue, or adjusting volume — even
with a Premium subscription.

**Root cause:** `buildFetchCurrentUserCmd()` is dispatched only in
`SpinnerDoneMsg` (post-OAuth animation). Returning users skip OAuth entirely:
`App.Init()` → `splashDismissMsg` default case → `viewGrid`. No profile fetch
fires. `store.userProfile.Product` stays `""` all session → `IsPremium()`
returns `false` → all three premium gates in `handlers.go` incorrectly block.

Workaround: `spotnik auth forget` + re-register forces the OAuth flow which
calls `buildFetchCurrentUserCmd()` via `SpinnerDoneMsg`. Session works after.

## Design

### `internal/app/app.go` — add profile fetch to authenticated Init path

The authenticated `Init()` path builds `initCmds` (lines ~961–968). Append
`a.buildFetchCurrentUserCmd()`:

```go
initCmds := append(paneCmds,
    fetchPlaybackStateCmd(a.player, api.Background),
    tea.Tick(time.Second, func(_ time.Time) tea.Msg {
        return panes.TickMsg{}
    }),
    splashTimer,
    alertsInitCmd,
    a.buildFetchCurrentUserCmd(), // fetch user tier for premium gates
)
```

No other callers change. The existing `userProfileLoadedMsg` handler in
`routing.go` already handles all error cases (`errNilClient`, `ForbiddenError`,
network failures) — this is a pure dispatch addition.

If the OAuth flow also fires a second fetch via `SpinnerDoneMsg`, the gateway
dedup waiter coalesces the two concurrent `/v1/me` calls harmlessly.

### `internal/app/user_profile_test.go` — new test

Add `TestApp_Init_AuthenticatedPath_FetchesUserProfile`:

1. Build an `App` with `needsAuth = false`, `needsRegister = false`, stub
   `userAPI` returning `domain.UserProfile{Product: "premium"}`.
2. Call `a.Init()`.
3. Collect all messages from the returned batch (use existing `collectInitMsgs`
   helper).
4. Assert at least one message is a `userProfileLoadedMsg` with non-nil profile.

Negative case: `needsAuth = true` → assert batch does NOT contain
`userProfileLoadedMsg` (deferred to OAuth flow).

## Acceptance Criteria

- [ ] `App.Init()` authenticated path appends `buildFetchCurrentUserCmd()`
- [ ] `TestApp_Init_AuthenticatedPath_FetchesUserProfile` passes (positive + negative cases)
- [ ] No regression: OAuth flow `SpinnerDoneMsg` still dispatches its own fetch
- [ ] `make ci` passes

## Tasks

- [ ] Add `a.buildFetchCurrentUserCmd()` to `initCmds` in `App.Init()`
      authenticated path — `internal/app/app.go`
      - test: `go test ./internal/app/ -run TestApp_Init -v` → PASS

- [ ] Add `TestApp_Init_AuthenticatedPath_FetchesUserProfile` to
      `internal/app/user_profile_test.go`
      - test: `go test ./internal/app/ -run TestApp_Init_Authenticated -v` → PASS

- [ ] `make ci` passes
