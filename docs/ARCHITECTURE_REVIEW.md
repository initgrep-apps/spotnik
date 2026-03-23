# Architecture Review: Spotnik

> **Date:** 2026-03-23
> **Scope:** Full codebase analysis against `docs/ARCHITECTURE.md`, Elm Architecture principles, Go idioms, and Bubble Tea best practices.
> **Method:** Parallel static analysis of all packages — `app/`, `api/`, `state/`, `config/`, `keychain/`, `ui/panes/`, `ui/components/`, `ui/theme/`.

---

## Executive Summary

Spotnik is a well-architected Bubble Tea application that follows the Elm Architecture faithfully across 9 completed features. The unidirectional data flow (Msg -> Update -> Cmd -> Msg) is cleanly maintained, the Store is the genuine single source of truth, and the theme system is properly abstracted. However, the codebase has accumulated technical debt in several areas: `app.go` has grown to 1,722 lines, the API client layer lacks interfaces for testability, two UI panes violate the `ui/ -> api/` import boundary, and several panes don't enforce View() height constraints. None of these are show-stoppers, but addressing them will significantly improve maintainability as the app grows.

**Overall Grade: B+** — Strong architecture, clean Elm flow, needs targeted refactoring.

---

## Code Flow Diagrams

### A. Application Lifecycle

```
╭─────────────────────────────────────────────────────────────────────╮
│                         main.go                                      │
│                     tea.NewProgram(app)                               │
╰────────────────────────────┬────────────────────────────────────────╯
                             │
                             ▼
╭─────────────────────────────────────────────────────────────────────╮
│                       cmd/root.go                                    │
│  1. config.Load()          ← ~/.config/spotnik/config.toml           │
│  2. keychain.Get(token)    ← OS keychain lookup                      │
│  3. token valid?                                                     │
│     ├─ YES → build API clients with LoggingTransport                 │
│     │        app.SetPlayer(), SetLibrary(), SetSearch(), etc.        │
│     └─ NO  → app.SetNeedsAuth(true, clientID, tokenStore)           │
│  4. tea.NewProgram(app, tea.WithAltScreen()).Run()                   │
╰────────────────────────────┬────────────────────────────────────────╯
                             │
                             ▼
╭─────────────────────────────────────────────────────────────────────╮
│                      App.Init()                                      │
│                                                                      │
│  needsAuth?                                                          │
│  ├─ YES → return splashTimer (5s)                                    │
│  │        (defers everything until auth succeeds)                    │
│  │                                                                   │
│  └─ NO  → return tea.Batch(                                          │
│               fetchPlaybackStateCmd,    ← immediate first poll       │
│               libraryPane.Init(),       ← load playlists/albums      │
│               tea.Tick(1s → TickMsg),   ← start polling loop         │
│               splashTimer(5s),          ← show splash                │
│           )                                                          │
╰─────────────────────────────────────────────────────────────────────╯
```

### B. Auth Flow (When needsAuth = true)

```
splashDismissMsg (after 5s)
        │
        ▼
currentView = viewAuth
        │
        ├──► prepareAuthCmd(clientID)
        │       │
        │       ▼
        │    Generate PKCE verifier + challenge
        │    Start local HTTP server on random port
        │    Open browser → Spotify auth URL
        │       │
        │       ▼
        │    authPreparedMsg { authURL, verifier, codeCh, serverClose }
        │       │
        │       ▼
        │    waitForCallbackCmd(...)
        │       │ (blocks on codeCh until browser redirects)
        │       ▼
        │    Exchange code for tokens
        │    Store in keychain
        │       │
        │       ▼
        │    authSuccessMsg { accessToken }
        │       │
        │       ▼
        │    Build all 6 API clients with LoggingTransport
        │    currentView = viewMain
        │    Start tea.Batch(fetchPlayback, libraryInit, tickLoop)
        │
        └──► authErrorMsg { err }
                │
                ▼
             Show error in auth panel, press q to quit
```

### C. Update() Message Routing (The Brain)

```
╭──────────────────────────────────────────────────────────────────────╮
│                        App.Update(msg)                                │
│                                                                       │
│  msg.(type) switch:                                                   │
│                                                                       │
│  ┌─ SYSTEM MESSAGES ─────────────────────────────────────────────┐   │
│  │  splashDismissMsg    → transition to viewAuth or viewMain     │   │
│  │  authPreparedMsg     → store URL, start waitForCallback       │   │
│  │  authSuccessMsg      → build clients, start tick loop         │   │
│  │  authErrorMsg        → show error in auth panel               │   │
│  │  tea.WindowSizeMsg   → propagate SetSize() to all panes      │   │
│  │  statusDismissMsg    → clear status bar                       │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  ┌─ POLLING LOOP ────────────────────────────────────────────────┐   │
│  │  TickMsg                                                       │   │
│  │    ├── forward to playerPane (progress bar animation)          │   │
│  │    ├── re-arm: tea.Tick(1s → TickMsg)                          │   │
│  │    ├── if backoffTicks > 0: decrement, skip fetches            │   │
│  │    ├── if tickCount % 3 == 0: fetchPlaybackStateCmd            │   │
│  │    └── if tickCount % 9 == 0: fetchQueueCmd                    │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  ┌─ KEYBOARD ROUTING ───────────────────────────────────────────┐    │
│  │  tea.KeyMsg                                                    │   │
│  │    │                                                           │   │
│  │    ├── deviceOverlayOpen? ──YES──► devicePane.Update(key)      │   │
│  │    │                                                           │   │
│  │    ├── searchOpen? ────────YES──► searchPane.Update(key)       │   │
│  │    │                                                           │   │
│  │    ├── viewAuth? ─────────YES──► only q/Ctrl+C/Esc → Quit     │   │
│  │    │                                                           │   │
│  │    ├── "q"  ───────────────────► tea.Quit                      │   │
│  │    ├── "2"  ───────────────────► openStats()                   │   │
│  │    ├── "3"  ───────────────────► openPlaylists()               │   │
│  │    ├── "1"  ───────────────────► closeStats/closePlaylists()   │   │
│  │    │                                                           │   │
│  │    ├── viewStats? ────────YES──► statsPane.Update(key)         │   │
│  │    ├── viewPlaylists? ────YES──► playlistPane.Update(key)      │   │
│  │    │                                                           │   │
│  │    ├── "/" ────────────────────► openSearch()                   │   │
│  │    ├── "d" ────────────────────► openDeviceOverlay()           │   │
│  │    ├── Tab ────────────────────► rotateFocus(+1)               │   │
│  │    ├── Shift+Tab ─────────────► rotateFocus(-1)                │   │
│  │    │                                                           │   │
│  │    ├── isPlaybackKey? ────YES──► playerPane.Update(key)        │   │
│  │    │   (space/n/p/+/-/s/r/←/→)  (temporarily set focused)     │   │
│  │    │                                                           │   │
│  │    └── else: route to focused pane                             │   │
│  │        ├── focusLibrary → libraryPane.Update(key)              │   │
│  │        ├── focusQueue   → queuePane.Update(key)                │   │
│  │        └── focusPlayer  → playerPane.Update(key)               │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  ┌─ DATA MESSAGES (pane requests → API calls → result msgs) ────┐   │
│  │  PlaybackRequestMsg       → buildPlaybackAPICmd(action)        │   │
│  │  PlaybackCmdSentMsg       → check err, status bar, re-fetch    │   │
│  │  PlaybackStateFetchedMsg  → playerPane.Update(msg)             │   │
│  │  QueueLoadedMsg           → (store already updated, no-op)     │   │
│  │  RateLimitedMsg           → activate backoff, status msg       │   │
│  │                                                                │   │
│  │  SearchRequestMsg         → buildSearchCmd(query)              │   │
│  │  SearchResultsMsg         → searchPane.Update(msg)             │   │
│  │  SearchClosedMsg          → closeSearch()                      │   │
│  │  PlayContextMsg           → buildPlayContextCmd, close search  │   │
│  │  PlayTrackMsg             → buildPlayTrackCmd, close search    │   │
│  │  AddToQueueMsg            → buildAddToQueueCmd                 │   │
│  │  AddToQueueResultMsg      → status bar feedback                │   │
│  │                                                                │   │
│  │  FetchPlaylistsRequestMsg → buildFetchPlaylistsCmd(offset)     │   │
│  │  FetchAlbumsRequestMsg    → buildFetchAlbumsCmd(offset)        │   │
│  │  FetchLikedTracksRequestMsg → buildFetchLikedTracksCmd(offset) │   │
│  │  FetchRecentlyPlayedRequestMsg → buildFetchRecentlyPlayedCmd   │   │
│  │  LikeTrackRequestMsg     → buildToggleLikeCmd                  │   │
│  │  LikeToggleResultMsg     → check err, status bar               │   │
│  │                                                                │   │
│  │  FetchDevicesRequestMsg   → buildFetchDevicesCmd               │   │
│  │  DeviceOverlayClosedMsg   → closeDeviceOverlay()               │   │
│  │  TransferPlaybackMsg      → buildTransferPlaybackCmd           │   │
│  │  DeviceTransferredMsg     → check err, re-fetch playback       │   │
│  │                                                                │   │
│  │  FetchStatsMsg            → buildFetchStatsCmd(timeRange)      │   │
│  │  StatsLoadedMsg           → statsPane.Update(msg)              │   │
│  │                                                                │   │
│  │  PlaylistCreateRequestMsg → buildCreatePlaylistCmd             │   │
│  │  PlaylistCreatedMsg       → re-fetch playlists                 │   │
│  │  PlaylistRenameRequestMsg → buildRenamePlaylistCmd             │   │
│  │  PlaylistRenamedMsg       → re-fetch playlists                 │   │
│  │  PlaylistRemoveRequestMsg → buildRemovePlaylistTrackCmd        │   │
│  │  PlaylistReorderRequestMsg→ buildReorderPlaylistTracksCmd      │   │
│  │  FetchPlaylistTracksRequestMsg → buildFetchPlaylistTracksCmd   │   │
│  │  PlaylistTracksLoadedMsg  → playlistPane.Update(msg)           │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  ┌─ CATCH-ALL ───────────────────────────────────────────────────┐   │
│  │  Overlay open?                                                 │   │
│  │    ├── deviceOverlayOpen → devicePane.Update(msg)              │   │
│  │    └── searchOpen        → searchPane.Update(msg)              │   │
│  │  else: libraryPane.Update(msg)  ← catches library loaded msgs │   │
│  └───────────────────────────────────────────────────────────────┘   │
╰──────────────────────────────────────────────────────────────────────╯
```

### D. Tick Polling Loop (Heartbeat)

```
App.Init()
    │
    ▼
tea.Tick(1s) ──► TickMsg
                    │
                    ▼
              ╭─────────────────────────────────────╮
              │         App.Update(TickMsg)          │
              │                                     │
              │  1. Forward to playerPane            │
              │     (localProgressMs += 1000)        │
              │                                     │
              │  2. Re-arm: tea.Tick(1s → TickMsg)   │
              │                                     │
              │  3. backoffTicks > 0?                │
              │     ├─ YES: decrement, skip fetches  │
              │     └─ NO:  continue                 │
              │                                     │
              │  4. tickCount % 3 == 0?              │
              │     └─ YES: fetchPlaybackStateCmd ──────────╮
              │                                     │      │
              │  5. tickCount % 9 == 0?              │      │
              │     └─ YES: fetchQueueCmd ─────────────────╮│
              │                                     │     ││
              │  6. tickCount++                       │     ││
              ╰─────────────────────────────────────╯     ││
                    ▲                                      ││
                    │                                      ││
                    └──────────── tea.Tick(1s) ◄────────────┘│
                                                             │
              ╭──────────────────────────────────────╮       │
              │    fetchPlaybackStateCmd (tea.Cmd)    │◄──────╯
              │                                      │
              │  player.GetPlaybackState(ctx)         │
              │    │                                  │
              │    ├─ OK: store.SetPlaybackState(ps)  │
              │    │       store.SetCurrentTrack(t)   │
              │    │       → PlaybackStateFetchedMsg  │
              │    │                                  │
              │    ├─ 429: parse Retry-After           │
              │    │       → RateLimitedMsg{secs}     │
              │    │                                  │
              │    └─ err: → PlaybackStateFetchedMsg  │
              ╰──────────────────────────────────────╯
```

### E. Pane Request → Command → Result Flow

```
Example: User presses space to pause playback

╭─────────────╮     tea.KeyMsg(" ")      ╭────────────────────╮
│   Bubble Tea │ ──────────────────────► │  App.Update(key)   │
│   Runtime    │                         │                    │
╰──────────────╯                         │  isPlaybackKey? Y  │
                                         │  → playerPane      │
                                         │    .Update(key)     │
                                         ╰────────┬───────────╯
                                                   │
                                    ╭──────────────▼──────────────╮
                                    │     PlayerPane.Update()      │
                                    │                              │
                                    │  space → toggle play/pause   │
                                    │  return (model, cmd) where   │
                                    │  cmd emits:                  │
                                    │  PlaybackRequestMsg{         │
                                    │    Action: ActionToggle      │
                                    │  }                           │
                                    ╰──────────────────────────────╯
                                                   │
                                    ╭──────────────▼──────────────╮
                                    │      App.Update(             │
                                    │        PlaybackRequestMsg)   │
                                    │                              │
                                    │  → buildPlaybackAPICmd(      │
                                    │      ActionToggle)           │
                                    │    returns tea.Cmd           │
                                    ╰──────────────┬──────────────╯
                                                   │
                              ╭────────────────────▼────────────────────╮
                              │     tea.Cmd executes async               │
                              │                                          │
                              │  if store.IsPlaying():                   │
                              │      player.Pause(ctx)                   │
                              │  else:                                   │
                              │      player.Play(ctx, PlayOptions{})     │
                              │                                          │
                              │  → PlaybackCmdSentMsg{ Err: nil/err }    │
                              ╰────────────────────┬────────────────────╯
                                                   │
                              ╭────────────────────▼────────────────────╮
                              │  App.Update(PlaybackCmdSentMsg)          │
                              │                                          │
                              │  Err == nil?                             │
                              │  ├─ YES: fetchPlaybackStateCmd           │
                              │  │       (re-poll immediately)           │
                              │  └─ NO:  statusMsg = "✗ ..."             │
                              │          + re-poll + 4s dismiss timer    │
                              ╰────────────────────────────────────────╯
```

### F. Search Overlay Flow

```
User presses "/"
        │
        ▼
openSearch()
  ├── prevFocus = current focus
  ├── searchOpen = true
  └── searchPane.Init() → focus textinput, start cursor blink
        │
        ▼
╭───────────────────────────────────────────────────────╮
│              SEARCH OVERLAY ACTIVE                      │
│  (all KeyMsg routed to searchPane.Update)              │
│                                                        │
│  User types "lo"                                       │
│    ├── textinput captures each keystroke                │
│    └── on each key: tea.Tick(300ms → debounceTickMsg)  │
│                                                        │
│  300ms passes with no new key                          │
│    └── debounceTickMsg fires                           │
│        query matches current input?                    │
│          └── YES: emit SearchRequestMsg{Query: "lo"}   │
╰───────────────────────┬───────────────────────────────╯
                        │
         ╭──────────────▼──────────────╮
         │   App.Update(               │
         │     SearchRequestMsg)       │
         │                             │
         │   buildSearchCmd("lo")      │
         │     store.SetSearchQuery    │  ← NOTE: mutation
         │     store.SetSearchLoading  │    in cmd builder
         │     return tea.Cmd:         │
         │       search.Search(ctx,    │
         │         "lo", types)        │
         │       store.SetResults(r)   │
         │       → SearchResultsMsg   │
         ╰──────────────┬─────────────╯
                        │
         ╭──────────────▼──────────────╮
         │  App.Update(                │
         │    SearchResultsMsg)        │
         │                             │
         │  store.SetSearchLoading(f)  │
         │  searchPane.Update(msg)     │
         │  → overlay re-renders       │
         │    with results             │
         ╰────────────────────────────╯
                        │
         User navigates results, presses Enter
                        │
         ╭──────────────▼──────────────╮
         │  PlayTrackMsg or            │
         │  PlayContextMsg or          │
         │  AddToQueueMsg              │
         │                             │
         │  → searchOpen = false       │
         │  → dispatch play/queue cmd  │
         ╰────────────────────────────╯
                   OR
         User presses Esc
                        │
         ╭──────────────▼──────────────╮
         │  SearchClosedMsg            │
         │  → closeSearch()            │
         │  → searchOpen = false       │
         ╰────────────────────────────╯
```

### G. View() Rendering Pipeline

```
╭──────────────────────────────────────────────────────────────────╮
│                        App.View()                                 │
│                                                                   │
│  1. Terminal too small? → renderTooSmall()                         │
│                                                                   │
│  2. viewSplash? → renderSplash()                                  │
│                                                                   │
│  3. viewAuth? → renderAuthPanel(theme, w, h, url, status)         │
│                                                                   │
│  4. viewStats? → renderStatsHeader + statsPane.View()             │
│                  + renderStatsStatusBar                            │
│                                                                   │
│  5. viewPlaylists? → renderPlaylistsHeader + playlistPane.View()  │
│                      + renderPlaylistsStatusBar                   │
│                                                                   │
│  6. viewMain:                                                     │
│     ╭─────────────────────────────────────────────────────╮      │
│     │                  renderHeader()                      │      │
│     │  ╭─ now playing ─╮╭─── device ──╮╭─── time ───╮    │      │
│     │  │ ♫ Song - Artist││ 🔊 MacBook  ││  12:34 PM  │    │      │
│     │  ╰────────────────╯╰─────────────╯╰────────────╯    │      │
│     ╰─────────────────────────────────────────────────────╯      │
│     ╭────────────╮╭──────────────────╮╭───────────────╮          │
│     │  Library   ││     Player       ││    Queue      │          │
│     │  (22%)     ││     (50%)        ││    (28%)      │          │
│     │            ││                  ││               │          │
│     │ ▸ Playlsts ││  Song Title      ││ 1. Track A   │          │
│     │   ▸ My Mix ││  Artist Name     ││ 2. Track B   │          │
│     │ ▸ Albums   ││                  ││ 3. Track C   │          │
│     │ ▸ Liked    ││  ▶━━━━━━━━━○──── ││ 4. Track D   │          │
│     │ ▸ Recent   ││  2:15 / 3:45    ││ ► Track E    │          │
│     │            ││                  ││               │          │
│     │            ││  ◄◄  ▶  ►►       ││               │          │
│     │            ││  🔀 off  🔁 off  ││               │          │
│     │            ││  Vol: ████░░ 60% ││               │          │
│     ╰────────────╯╰──────────────────╯╰───────────────╯          │
│     ╭─────────────────────────────────────────────────────╮      │
│     │              renderStatusBar()                       │      │
│     │  Tab:switch  /:search  d:device  q:quit   ✓ Added   │      │
│     ╰─────────────────────────────────────────────────────╯      │
│                                                                   │
│  7. Overlay? Compose on top:                                      │
│     ├── deviceOverlayOpen → dim background + overlay top-right    │
│     └── searchOpen → dim background + overlay centered            │
╰──────────────────────────────────────────────────────────────────╯
```

### H. Focus Routing State Machine

```
                    Tab (+1)
       ╭──────────────────────────────╮
       │                              │
       ▼                              │
╭──────────╮  Tab   ╭──────────╮  Tab   ╭──────────╮
│  Player  │ ─────► │ Library  │ ─────► │  Queue   │
│ (default)│        │          │        │          │
╰──────────╯ ◄───── ╰──────────╯ ◄───── ╰──────────╯
       ▲    Shift+Tab          Shift+Tab   │
       │                                   │
       ╰───────────────────────────────────╯
                  Shift+Tab (-1)

  OVERLAY INTERCEPT:
  ╭────────────────────────────────────────────────╮
  │  When searchOpen or deviceOverlayOpen:         │
  │  ALL KeyMsg → overlay pane                     │
  │  Focus state is paused (not rotated)           │
  │  prevFocus saved on open                       │
  │  ⚠ prevFocus NOT restored on close (see gap)   │
  ╰────────────────────────────────────────────────╯

  PLAYBACK KEY BYPASS:
  ╭────────────────────────────────────────────────╮
  │  Keys: space n p + - s r ← →                  │
  │  ALWAYS routed to playerPane regardless of     │
  │  focus. PlayerPane is temporarily SetFocused   │
  │  for the duration of the Update() call.        │
  ╰────────────────────────────────────────────────╯

  VIEW-LEVEL ROUTING:
  ╭────────────────────────────────────────────────╮
  │  viewStats     → all keys to statsPane         │
  │  viewPlaylists → all keys to playlistPane      │
  │  (after global keys: q, 1, 2, 3)              │
  ╰────────────────────────────────────────────────╯
```

### I. Data Flow: Store as Single Source of Truth

```
╭────────────────╮  request msg   ╭───────────────╮  tea.Cmd    ╭────────────╮
│     Panes      │ ─────────────► │   App.Update  │ ──────────► │  API Client│
│  (read-only)   │                │   (router)    │             │  (HTTP)    │
╰───────┬────────╯                ╰───────────────╯             ╰─────┬──────╯
        │                                                             │
        │ reads                                                 writes│
        │                                                             │
        │         ╭───────────────────────────────────╮               │
        │         │             Store                  │               │
        │         │        (sync.RWMutex)              │               │
        │         │                                    │               │
        ├── RLock │  PlaybackState  *api.PlaybackState │ Lock ◄────────┤
        ├── RLock │  CurrentTrack   *api.Track         │ Lock ◄────────┤
        ├── RLock │  Queue          []api.Track        │ Lock ◄────────┤
        ├── RLock │  Playlists      []api.SimplePlaylist│ Lock ◄────────┤
        ├── RLock │  SearchResults  *api.SearchResult   │ Lock ◄────────┤
        ├── RLock │  TopTracks      map[string][]Track  │ Lock ◄────────┤
        ├── RLock │  Devices        []api.Device        │ Lock ◄────────┤
        ├── RLock │  ErrorMessage   string              │ Lock ◄────────┤
        │         │  ...50+ accessor methods            │               │
        │         ╰───────────────────────────────────╯               │
        │                                                             │
        ▼                                                             │
  View() reads ──────► string output ──────► terminal         result msg
                                                               │
                                                               ▼
                                                        App.Update
                                                        (result handling)
```

### J. build*Cmd Pattern (Command Factory)

```
╭─────────────────────────────────────────────────────────────╮
│  buildXxxCmd(params) tea.Cmd                                 │
│                                                              │
│  // Capture references from App struct                       │
│  client := a.xxxClient                                       │
│  store  := a.store                                           │
│                                                              │
│  return func() tea.Msg {           ← returned as tea.Cmd     │
│      // Nil-guard (no mock interface available)              │
│      if client == nil {                                      │
│          return XxxResultMsg{}     ← empty success           │
│      }                                                       │
│                                                              │
│      // Execute API call (runs on Bubble Tea's goroutine)    │
│      result, err := client.DoThing(context.Background())     │
│                                                              │
│      if err != nil {                                         │
│          store.SetXxxError(err)    ← write error to store    │
│          return XxxResultMsg{}                               │
│      }                                                       │
│                                                              │
│      store.ClearXxxError()         ← clear previous error    │
│      store.SetXxx(result)          ← write data to store     │
│      return XxxResultMsg{}         ← signal completion       │
│  }                                                           │
│                                                              │
│  ⚠ 18 of these exist in app.go (~350 lines)                 │
│  ⚠ Should be extracted to commands.go                        │
╰─────────────────────────────────────────────────────────────╯
```

### K. Dependency Graph (Import Boundaries)

```
                    ╭──────────────╮
                    │   main.go    │
                    ╰──────┬───────╯
                           │
                    ╭──────▼───────╮
                    │  cmd/root.go │
                    ╰──────┬───────╯
                           │
                    ╭──────▼───────╮
                    │  internal/app │
                    ╰──┬──┬──┬──┬──╯
                       │  │  │  │
          ╭────────────╯  │  │  ╰────────────────╮
          │               │  │                    │
   ╭──────▼──────╮ ╭─────▼──▼──────╮  ╭──────────▼──────────╮
   │ internal/api │ │ internal/state │  │    internal/ui/      │
   │              │ │                │  │  ╭──────────────╮   │
   │ Player       │ │ Store          │  │  │  panes/      │   │
   │ LibraryClient│ │ NetLog         │  │  │  components/ │   │
   │ SearchClient │ │                │  │  │  theme/      │   │
   │ DevicesClient│ ╰───────┬────────╯  │  ╰──────────────╯   │
   │ UserClient   │         │           ╰──────────────────────╯
   │ PlaylistsClnt│         │
   ╰──────────────╯         │
          ▲                 │
          │   state/ imports api/ (for types)
          ╰─────────────────╯

   FORBIDDEN (but violated):
   ╭──────────────────────────────────────────╮
   │  ui/panes/devices.go ──imports──► api/   │  ✗
   │  ui/panes/search.go  ──imports──► api/   │  ✗
   ╰──────────────────────────────────────────╯

   FORBIDDEN (clean):
   ╭──────────────────────────────────────────╮
   │  api/  ──✗──► ui/     (zero violations)  │  ✓
   │  state/──✗──► ui/     (zero violations)  │  ✓
   ╰──────────────────────────────────────────╯

   SEPARATE CONCERNS:
   ╭──────────────╮    ╭──────────────╮
   │internal/config│    │int./keychain │
   │  (TOML load)  │    │ (TokenStore) │
   ╰──────────────╯    ╰──────────────╯
```

### L. Optimistic Update Pattern (Playlist Manager)

```
User presses Shift+↓ to reorder track

╭─────────────────────────────────────────────────────────╮
│  PlaylistManager.Update(Shift+Down)                      │
│                                                          │
│  1. Save: pm.prevTracks = copy(pm.tracks)                │
│  2. Swap: pm.tracks[cursor] ↔ pm.tracks[cursor+1]       │
│  3. UI immediately shows reordered list ← OPTIMISTIC     │
│  4. Emit: PlaylistReorderRequestMsg{                     │
│       PlaylistID, RangeStart, InsertBefore, RangeLength  │
│     }                                                    │
╰──────────────────────────┬──────────────────────────────╯
                           │
            ╭──────────────▼──────────────╮
            │  App dispatches              │
            │  buildReorderPlaylistTracksCmd│
            │                              │
            │  playlistsAPI.Reorder(ctx,..)│
            ╰──────────────┬──────────────╯
                           │
              ╭────────────▼────────────╮
              │ PlaylistReorderResultMsg │
              │                         │
              │  Err == nil?            │
              │  ├─ YES: keep optimistic│
              │  │       state as-is    │
              │  │                      │
              │  └─ NO:  ROLLBACK       │
              │       pm.tracks =       │
              │         pm.prevTracks   │
              │       status: "✗ ..."   │
              ╰─────────────────────────╯
```

---

## 1. Elm Architecture Compliance

The Elm Architecture (Model-Update-View with Commands for side effects) is the foundation of Bubble Tea. Spotnik's adherence is evaluated below.

### Positives

| Principle | Status | Evidence |
|---|---|---|
| State is immutable between Updates | COMPLIANT | `App` struct fields mutated only within `Update()` and private helpers called from it |
| View() is pure (no side effects) | COMPLIANT | `app.go:1283-1337` — reads store + pane views, returns string, no I/O |
| Side effects only via tea.Cmd | COMPLIANT | All 18 `build*Cmd` functions return closures; no direct API calls in Update |
| Messages are typed structs | COMPLIANT | All 34+ message types in `panes/messages.go` are typed structs, zero strings |
| Message naming is consistent | COMPLIANT | `NounVerbMsg` / `NounVerbRequestMsg` / `NounVerbResultMsg` pattern throughout |
| tea.Tick for polling (no time.Sleep) | COMPLIANT | Zero `time.Sleep` calls in entire codebase; tick loop at 1s via `tea.Tick` |
| tea.Batch for concurrent commands | COMPLIANT | Used correctly in Init(), Update(), and tick handler |
| Children don't talk to each other | COMPLIANT | All inter-pane communication goes through messages via root model |

### Gaps

| Issue | Location | Impact | Fix |
|---|---|---|---|
| Store mutation in cmd builder body | `app.go:1116-1117` | `buildSearchCmd` calls `store.SetSearchQuery()` and `store.SetSearchLoading(true)` synchronously before returning the closure. Breaks the invariant that store writes happen only inside commands. | Move these two writes inside the returned closure, or handle them in `Update()` before building the command |
| View() mutates state | `library.go:401` | `LibraryPane.View()` calls `p.tree.UpdateFromStore(p.store)` — a write operation inside what should be a pure render. Idempotent but fragile. | Remove from `View()`. The identical call in `Update()` at line 356 is sufficient |
| SearchOverlay writes to store | `search.go:206` | `o.store.SetSearchResults(nil)` on Ctrl+U. Only pane that directly writes to the store. | Emit a `SearchClearedMsg` and let the root model handle the store write |

---

## 2. Go Idioms & Patterns

Evaluated against Effective Go, standard library patterns, and the go-dev skill reference.

### Positives

| Pattern | Status | Evidence |
|---|---|---|
| Project layout (`internal/`, `cmd/`, `testdata/`) | COMPLIANT | Textbook Go project structure |
| `context.Context` as first parameter | COMPLIANT | Every API method signature; passed to `http.NewRequestWithContext` |
| Error wrapping with `fmt.Errorf("...: %w")` | COMPLIANT | Consistent across all API clients |
| `sync.RWMutex` for concurrent access | COMPLIANT | Store and NetLog both use RLock/Lock correctly with defer |
| Table-driven tests | COMPLIANT | Used consistently across `api/`, `state/`, `config/`, `ui/panes/` |
| `httptest.NewServer` for API mocks | COMPLIANT | Every API client test; no external mock libraries |
| Exported types have doc comments | COMPLIANT | All 30+ exported types and functions documented |
| No `panic()` in production paths | COMPLIANT | Zero production panics found |
| `testify` assert/require discipline | COMPLIANT | `require` for fatal conditions, `assert` for non-fatal |

### Gaps

| Issue | Location | Impact | Fix |
|---|---|---|---|
| `Get` prefix on all getters | All `api/*.go` | `GetPlaybackState`, `GetPlaylists`, etc. violate Go convention (Effective Go: omit Get prefix) | Rename to `PlaybackState()`, `Playlists()`, etc. — breaking change, do in one pass |
| No `SpotifyClient` interface | `api/` package | Architecture spec calls for it; without it, app-level tests can't inject fakes. The nil-guard pattern (`if client == nil`) is a workaround, not a solution | Define per-domain interfaces where they are *used* (in `app/` or as a shared `internal/api/` interface), not per-client. See remediation plan below |
| No `TokenProvider` interface | All API clients | Token is baked in at construction as a string. No refresh propagation to live clients. Token expiry between polls silently causes 401s | Define `TokenProvider interface { Token() (string, error) }` and call it per-request in `newRequest()` |
| Duplicated HTTP helpers | All `api/*.go` | `newRequest`/`doJSON`/`doNoContent` copy-pasted across 6 client files (~150 LOC duplication) | Extract a shared `baseClient` struct or package-level generic helpers |
| Error type matching via string parsing | `app.go:1264-1280` | `parse429RetryAfter` uses `strings.Contains(msg, "429")`. Fragile; any error message format change breaks rate-limit backoff | Define `type RateLimitError struct { RetryAfter int }` in `api/` and use `errors.As` |
| 403 check via string matching | `app.go:658` | `strings.Contains(errMsg, "403")` | Same fix: use typed error + `errors.As` |
| No `fetchAll` pagination helper | `api/` package | Architecture spec defines a generic `fetchAll[T]` but none exists. Each paginated call uses fixed limit/offset | Implement the generic helper per the spec |

---

## 3. Import Boundary Compliance

The architecture specifies strict one-way dependencies. The forbidden imports are:

```
api/ -> ui/    FORBIDDEN
ui/ -> api/    FORBIDDEN
state/ -> ui/  FORBIDDEN
state/ -> api/ ALLOWED (necessary — store holds api types)
```

### Results

| Boundary | Status | Violation |
|---|---|---|
| `api/` -> `ui/` | COMPLIANT | Zero violations |
| `state/` -> `ui/` | COMPLIANT | Zero violations |
| `state/` -> `api/` | COMPLIANT | One-way dependency, architecturally required |
| `ui/` -> `api/` | **GAP** | **2 production files violate this rule** |

**Violations:**

1. **`internal/ui/panes/devices.go:11`** — imports `api` for `api.Device` type in overlay struct and `NewDevicesLoadedMsg` constructor. The file's own header comment says "It never imports api/ directly" — the comment contradicts the code.

2. **`internal/ui/panes/search.go:13`** — imports `api` for `api.SearchResult`, `api.Track`, `api.SearchArtist`, `api.SearchAlbum`, `api.SearchPlaylist` in render helpers (lines 498-569).

**Remediation:** Define UI-facing DTOs or use the message types to carry pre-formatted data. The `messages.go` pattern (where messages carry primitive data, not api types) should be extended to these two panes.

---

## 4. Bubble Tea Best Practices

### Positives

| Practice | Status | Evidence |
|---|---|---|
| Root model owns focus state | COMPLIANT | `app.go:880-914` — `rotateFocus()` centralizes Tab cycling |
| Overlay captures all input when active | COMPLIANT | `app.go:476-491` — overlay check before pane dispatch |
| Tick polling via `tea.Tick` | COMPLIANT | 1s playback poll + queue poll on tick |
| Debounce via `tea.Tick` | COMPLIANT | 300ms search debounce in `SearchOverlay` |
| `tea.WithAltScreen()` | COMPLIANT | Full-screen mode enabled |
| `lipgloss.JoinHorizontal` for layout | COMPLIANT | Three-pane composition in `View()` |
| Bubbles components used appropriately | COMPLIANT | `textinput` for search/playlist naming, `spinner` for loading |
| Pane sizing via `SetSize()` | COMPLIANT | `WindowSizeMsg` propagated with correct ratios (22%/50%/28%) |

### Gaps

| Issue | Location | Impact | Fix |
|---|---|---|---|
| `prevFocus` stored but never restored | `app.go:297-300, 311-314` | `closeSearch()` and `closeDeviceOverlay()` don't reassign `a.focus = a.prevFocus`. Accidentally works because focus isn't changed while overlays are open, but fragile | Explicitly restore `a.focus = a.prevFocus` in both close functions |
| Catch-all message routing | `app.go:858` | All unhandled messages fall through to `libraryPane.Update(msg)`. Silently swallows messages the library pane doesn't handle | Add an explicit default case or route only known message types |
| View() height overflow | `library.go`, `stats.go`, `playlists.go:704` | LibraryPane, StatsView, and PlaylistManager render unbounded item lists without height capping | Implement viewport scrolling modeled after `QueuePane.visibleTrackCount()` pattern |
| No `bubbles/viewport` for scrolling | `netlog.go`, `stats.go` | Hand-rolled scroll logic duplicates what `bubbles/viewport` provides | Consider adopting viewport for new scroll-heavy panes |

---

## 5. Testing Architecture

### Positives

| Area | Status | Evidence |
|---|---|---|
| Table-driven tests throughout | COMPLIANT | Consistent in all `*_test.go` files |
| httptest.NewServer for API mocking | COMPLIANT | No external mock libraries |
| JSON fixtures in `testdata/fixtures/` | COMPLIANT | 16 descriptive fixture files |
| TokenStore interface with InMemoryTokenStore | COMPLIANT | Full test-friendly implementation with compile-time check |
| Store accessor coverage | COMPLIANT | Every Get/Set/Clear method tested including all 9 error state triples |

### Gaps

| Issue | Location | Impact | Fix |
|---|---|---|---|
| No MockClient for app-level tests | `app_test.go` | Uses nil-guard pattern instead of injecting fakes. Cannot test error handling (401/429/403) at the app level | Create MockClient(s) implementing the SpotifyClient interface(s) once defined |
| No `//go:build integration` tags | `keychain_test.go` | OS keychain tests use runtime `t.Skipf` instead of build tags. CI can't exclude them cleanly | Add `//go:build integration` to keychain tests |
| No 401 retry-once test coverage | `app.go` | 401 handling (refresh token + retry) is not implemented at all, let alone tested | Implement the 401 flow per architecture spec, then test |

---

## 6. UI & Design Compliance

### Positives

| Rule | Status | Evidence |
|---|---|---|
| Three-pane layout frozen | COMPLIANT | Layout assembled only in `app.go:View()` |
| Theme interface (24 tokens) | COMPLIANT | 5 complete implementations, safe `Load()` with fallback |
| Default theme is `black` | COMPLIANT | Config defaults to "black", Load() falls back to "black" |
| Rounded corners only (╭╮╰╯) | COMPLIANT | `RoundedBorder()` used consistently |
| Status bar always visible | COMPLIANT | Rendered in every branch of `View()` |
| Components use theme tokens | COMPLIANT | 4 stateless components, all theme-driven |

### Gaps

| Issue | Location | Impact | Fix |
|---|---|---|---|
| Hardcoded hex `#000000` | `app.go:1354, 1375` | Overlay whitespace color bypasses theme; looks wrong on non-black themes | Replace with `a.theme.Base()` |
| Dead code: `nextRepeatMode()` | `player.go:231` | Declared but never called | Remove the function |
| stats.go compile break (WIP) | `stats.go:525` | References `sv.netLogView` but the field was removed from the struct. Code does not compile | Either re-add `netLogView *NetLogView` to the struct or remove the reference |

---

## 7. app.go Decomposition

At 1,722 lines, `app.go` is the largest file and the most pressing maintainability concern. Breakdown:

| Section | ~Lines | Extractable? |
|---|---|---|
| Struct + constructor + setters | 175 | No — root model definition |
| `Init()` + helpers | 100 | No — small and cohesive |
| `Update()` dispatch | 500 | Partially — playlist messages (lines 779-855) are logically separate |
| `build*Cmd` (18 functions) | 350 | **Yes** — move to `internal/app/commands.go` |
| Rendering helpers | 460 | **Yes** — move to `internal/app/render.go` |
| Key helpers, focus rotation | 55 | No — tightly coupled to Update |

**Recommended split:**

```
internal/app/
├── app.go          (~600 lines) — struct, Init, Update, focus routing
├── commands.go     (~400 lines) — all build*Cmd functions + fetchPlaybackStateCmd
├── render.go       (~500 lines) — View() + all render* helpers
├── auth.go         (139 lines)  — unchanged
├── splash.go       (small)      — unchanged
```

Additionally, the three duplicate header renderers (`renderHeader`, `renderStatsHeader`, `renderPlaylistsHeader`) share ~85% code and should be collapsed into `renderHeader(label string)`. Same for the three status bar renderers.

---

## 8. Performance Considerations

| Area | Status | Notes |
|---|---|---|
| Polling rate | COMPLIANT | 1s tick with backoff on 429 |
| No goroutine leaks | COMPLIANT | Only one goroutine (PKCE auth server), properly closed |
| No `time.Sleep` | COMPLIANT | All timing via `tea.Tick` |
| View() allocation | ACCEPTABLE | Styles constructed per render call; no caching, but lipgloss is fast for this scale |
| Store locking | COMPLIANT | RWMutex with proper read/write separation |
| Search debounce | COMPLIANT | 300ms delay prevents excessive API calls |
| Pagination | PARTIAL | Fixed page sizes; no pre-fetching or lazy loading beyond initial page. Large libraries may feel slow |

---

## 9. Prioritized Remediation Plan

### P0 — Compile-Breaking / Correctness

| # | Issue | Files | Effort |
|---|---|---|---|
| 1 | Fix stats.go `netLogView` reference (WIP) | `panes/stats.go` | 15 min |
| 2 | Fix `prevFocus` restoration on overlay close | `app.go:297-300, 311-314` | 10 min |
| 3 | Remove `View()` mutation in LibraryPane | `panes/library.go:401` | 5 min |

### P1 — Architecture Violations

| # | Issue | Files | Effort |
|---|---|---|---|
| 4 | Fix `ui/ -> api/` import in `devices.go` | `panes/devices.go`, `panes/messages.go` | 1-2 hrs |
| 5 | Fix `ui/ -> api/` import in `search.go` | `panes/search.go`, `panes/messages.go` | 1-2 hrs |
| 6 | Replace hardcoded `#000000` with `theme.Base()` | `app.go:1354, 1375` | 5 min |
| 7 | Move store mutations out of `buildSearchCmd` body | `app.go:1116-1117` | 10 min |
| 8 | Emit `SearchClearedMsg` instead of direct store write | `panes/search.go:206` | 30 min |

### P2 — Testability & Go Idioms

| # | Issue | Files | Effort |
|---|---|---|---|
| 9 | Define per-domain interfaces for API clients | `api/`, `app/` | 3-4 hrs |
| 10 | Create MockClient(s) for app-level testing | `api/mock_*.go`, `app_test.go` | 2-3 hrs |
| 11 | Implement `TokenProvider` interface | All `api/*.go` | 2 hrs |
| 12 | Replace string-based error matching with typed errors | `api/player.go`, `app.go` | 1-2 hrs |
| 13 | Implement `fetchAll[T]` pagination helper | `api/pagination.go` | 1 hr |
| 14 | Add `//go:build integration` to keychain tests | `keychain_test.go` | 10 min |
| 15 | Remove `Get` prefix from API getters | All `api/*.go` + all callers | 2 hrs (mechanical) |

### P3 — Maintainability

| # | Issue | Files | Effort |
|---|---|---|---|
| 16 | Split `app.go` into `commands.go` + `render.go` | `app/` | 1-2 hrs |
| 17 | Collapse 3 duplicate header renderers into 1 | `app.go` | 30 min |
| 18 | Collapse 3 duplicate status bar renderers into 1 | `app.go` | 30 min |
| 19 | Extract shared `baseClient` for HTTP helpers | All `api/*.go` | 2 hrs |
| 20 | Add height-capped rendering to LibraryPane, StatsView, PlaylistManager | `panes/library.go`, `panes/stats.go`, `panes/playlists.go` | 3-4 hrs |
| 21 | Remove dead `nextRepeatMode()` | `panes/player.go:231` | 5 min |

### P4 — Missing Spec Features

| # | Issue | Files | Effort |
|---|---|---|---|
| 22 | Implement 401 token refresh + retry-once | `app.go`, `api/auth.go` | 3-4 hrs |
| 23 | Extend 429 backoff to library/search errors | `app.go` | 1-2 hrs |
| 24 | Extend 403 handling beyond playback | `app.go` | 30 min |

---

## 10. Summary Scorecard

| Dimension | Score | Notes |
|---|---|---|
| **Elm Architecture** | 9/10 | Textbook unidirectional flow; 3 minor violations noted |
| **Go Idioms** | 7/10 | Strong error wrapping + testing, but no interfaces, Get prefix, string error matching |
| **Bubble Tea Patterns** | 8/10 | Clean focus routing, overlays, tick polling; height overflow in 3 panes |
| **Import Boundaries** | 8/10 | 2 production violations out of dozens of correct boundaries |
| **Testing** | 8/10 | Comprehensive unit tests; no app-level integration testing with mocks |
| **Theme/Design** | 9/10 | 2 hardcoded hex values; everything else clean |
| **Maintainability** | 6/10 | `app.go` at 1,722 lines with duplicated renderers is the main drag |
| **Performance** | 9/10 | No bottlenecks; proper polling, debounce, and locking |
| **Overall** | **8/10** | Strong foundation, needs targeted P1/P2 refactoring |

---

*Generated by parallel static analysis of all packages. No code was modified during this review.*
