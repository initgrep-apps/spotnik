# Sanity Test Cases — Spotnik

> Behavioral test cases in Given/When/Then format. Manual testing + component-test automation.
> **Rule:** Any change modifying user-facing behavior must add/update cases here.

---

## Priority Tiers

| Tier | Meaning |
|---|---|
| **P0** | Must pass before merge. Core app broken if fails. |
| **P1** | Should pass. Significant user impact. |
| **P2** | Nice to pass. Cosmetic or edge-case. |

---

## 01. Auth & Onboarding

### First Launch — Registration (Step 1)
**P0**

```
GIVEN spotnik launched first time
  AND no config.toml exists
WHEN splash screen finishes
THEN registration screen (Step 1) shown
  AND redirect URI displayed (with configured callback port)
  AND UI shows "enter your Client ID"
```

```
GIVEN registration screen showing
  AND input field empty
WHEN user presses 'c'
THEN redirect URI copied to clipboard
```

```
GIVEN registration screen showing
WHEN user types valid 32-char hex Client ID + Enter
THEN Client ID saved to ~/.config/spotnik/config.toml
  AND Step 2 (OAuth) begins
  AND browser opens with Spotify authorization URL
```

```
GIVEN registration screen showing
WHEN user types invalid Client ID + Enter
THEN validation error shown
  AND user can retry up to 3 times
```

### First Launch — OAuth (Step 2)
**P0**

```
GIVEN Step 2 (OAuth) showing
WHEN user presses 'c'
THEN full authorization URL copied to clipboard
```

```
GIVEN Step 2 (OAuth) showing
WHEN user presses 'v'
THEN permissions overview overlay opens
```

```
GIVEN Step 2 (OAuth) showing
  AND user authorizes in browser
WHEN callback server receives authorization code
THEN tokens exchanged + stored in system keychain
  AND main TUI launches
```

### OAuth Error
**P1**

```
GIVEN Step 2 (OAuth) encounters error
WHEN error screen shown
THEN pressing 'r' retries registration flow
  AND pressing 'l' re-launches OAuth without resetting Client ID
  AND pressing 'q' quits
```

### Returning User — Token Restore
**P0**

```
GIVEN tokens stored in system keychain
WHEN spotnik launched
THEN splash shows briefly
  AND main TUI launches without auth screens
  AND NowPlaying pane shows current track within 1s
```

### Returning User — Token Expired / Missing
**P1**

```
GIVEN tokens missing from keychain
  AND Client ID exists in config.toml
WHEN spotnik launched
THEN auth screen (Step 2) shown directly (no registration)
```

### Token Refresh on 401
**P0**

```
GIVEN API call returns 401 Unauthorized
WHEN error received
THEN spotnik auto-refreshes access token
  AND retries original request once
  AND if refresh succeeds, request completes normally
  AND if refresh fails, toast shown
```

### Proactive Token Refresh (#396)
**P1**

```
GIVEN access token expires within 5 minutes
WHEN next AccessToken() call made
THEN RefreshableTokenProvider refreshes proactively (no 401 needed)
  AND single-flight mutex prevents concurrent refresh
  AND original request proceeds with new token
```

### Auth CLI Commands
**P1**

```
GIVEN spotnik invoked as `spotnik auth register`
WHEN user follows prompts
THEN Client ID collected, OAuth flow runs, tokens stored
  AND command exits 0
```

```
GIVEN tokens stored + Client ID in config
WHEN `spotnik auth login` invoked
THEN existing tokens cleared
  AND new OAuth flow starts
  AND exits 0
```

```
GIVEN tokens stored
WHEN `spotnik auth logout` invoked
THEN tokens cleared from keychain
  AND Client ID remains in config
  AND exits 0
```

```
GIVEN tokens + Client ID stored
WHEN `spotnik auth forget` invoked
THEN tokens cleared from keychain
  AND Client ID removed from config.toml
  AND exits 0
```

```
GIVEN tokens + Client ID stored
WHEN `spotnik auth status` invoked
THEN prints Client ID presence (yes/no)
  AND prints token state (valid/expired/missing)
```

---

## 02. Playback Controls

### Play / Pause
**P0**

```
GIVEN main TUI running
  AND track currently playing
WHEN user presses Space
THEN playback pauses
  AND NowPlaying title shows pause glyph (⏸)
  AND visualizer stops animating
```

```
GIVEN playback paused
WHEN user presses Space
THEN playback resumes
  AND NowPlaying title shows play glyph (▶)
  AND visualizer resumes animating
```

### Seek Forward / Backward
**P1**

```
GIVEN track playing at position 30s
WHEN user presses → (right arrow)
THEN seek bar advances ~5s
  AND debounced seek request sent to Spotify
```

```
GIVEN track playing at position 30s
WHEN user presses ← (left arrow)
THEN seek bar retreats ~5s
```

```
GIVEN rapid left/right arrow presses
WHEN multiple seeks triggered
THEN only final seek position sent to Spotify (debounced)
  AND stale ticks discarded via seq field mismatch (#318)
  AND HasPending() preserves local progress during debounce
```

### Skip Track
**P1**

```
GIVEN track playing
WHEN user presses Shift+→
THEN next track starts playing
  AND NowPlaying pane updates to show new track
```

```
GIVEN track playing
WHEN user presses Shift+←
THEN previous track starts playing
```

### Volume
**P1**

```
GIVEN current volume 50%
WHEN user presses +
THEN volume increases ~5%
  AND debounced volume change request sent to Spotify
  AND volume bar updates optimistically (#269)
```

```
GIVEN current volume 50%
WHEN user presses -
THEN volume decreases ~5%
```

### Shuffle
**P1**

```
GIVEN shuffle off
WHEN user presses 's'
THEN shuffle turns on
  AND shuffle toggle request sent to Spotify
```

```
GIVEN shuffle on
WHEN user presses 's'
THEN shuffle turns off
```

### Repeat Mode
**P1**

```
GIVEN repeat off
WHEN user presses 'r'
THEN repeat cycles to "context" (repeat all)
  AND displayed in playback controls
```

```
GIVEN repeat "off"
WHEN user presses 'r' three times
THEN repeat cycles: off → context → track → off
  AND repeat-one shows ↻¹ superscript icon
```

### Visualizer
**P2**

```
GIVEN NowPlaying pane visible
WHEN user presses 'v'
THEN visualizer pattern cycles to next animation
  AND visualizer continues animating with new pattern
```

### Premium Gating
**P1**

```
GIVEN user has Spotify Free account
WHEN any playback key pressed (Space, s, r, ←, →, Shift+←, Shift+→, +, -)
THEN playback NOT sent to Spotify
  AND toast shown: "Spotify Premium required"
```

---

## 03. NowPlaying Display

### Track Mode
**P0**

```
GIVEN track currently playing
WHEN NowPlaying pane renders
THEN track name + artist displayed in InfoBox
  AND album name shown
  AND visualizer animation visible
  AND seek bar shows current progress
  AND playback controls (play/pause/shuffle/repeat) visible
```

### Episode Mode
**P1**

```
GIVEN podcast episode currently playing
WHEN NowPlaying pane renders
THEN episode name + show name displayed in InfoBox
  AND seek bar shows episode duration
  AND InfoBox border shows podcast notch indicator
  AND title shows "[progress] episode_name" format in compact mode
  AND no episode info embedded in pane border (#397)
```

### Adaptive Layout — Narrow Terminal
**P1**

```
GIVEN terminal width very narrow (< ~60 cols)
WHEN NowPlaying pane renders
THEN InfoBox dropped (does not render)
  AND visualizer fills full content area
  AND seek bar remains visible
```

### Adaptive Layout — Normal Width
**P1**

```
GIVEN terminal width normal (>= ~80 cols)
WHEN NowPlaying pane renders
THEN InfoBox overlays left ~25% of visualizer area
  AND seek bar positioned on right side
  AND equal padding surrounds content
```

### Compact Preset
**P2**

```
GIVEN Dashboard or Library/Discovery preset active
  AND NowPlaying pane height < 8 rows
WHEN NowPlaying pane renders
THEN compact track info embedded in pane title bar
  AND controls still visible
  AND no excess padding
```

### Seek Bar Interpolation
**P1**

```
GIVEN track playing
WHEN 5 seconds pass between poll ticks
THEN seek bar advances smoothly (1s local interpolation)
  AND at next poll, position snaps to actual Spotify position
```

---

## 04. Queue

### Queue Display
**P0**

```
GIVEN main TUI running
  AND songs queued
WHEN user views Queue pane
THEN upcoming tracks listed in table
  AND each row shows: #, type (♪/◆), track name, artist, duration
```

### Queue Filter
**P1**

```
GIVEN Queue pane focused
WHEN user presses 'f'
THEN filter input activates
  AND typing track name filters queue table in real-time
  AND no API calls made during filtering
```

```
GIVEN Queue filter active with query
WHEN user presses Esc
THEN filter cleared
  AND all tracks shown again
```

### Queue — Play from Queue
**P1**

```
GIVEN Queue pane focused
  AND track selected
WHEN user presses Enter
THEN that track starts playing immediately
```

### Mixed Content Queue (Tracks + Episodes) (#336)
**P1**

```
GIVEN queue contains both tracks + podcast episodes
WHEN Queue pane renders
THEN tracks show ♪ in type column
  AND episodes show ◆ in type column
  AND # column shows index
```

```
GIVEN Queue pane focused
  AND episode row selected
WHEN user presses Enter
THEN episode starts playing
```

### Empty Queue
**P2**

```
GIVEN no tracks queued
WHEN Queue pane renders
THEN empty state message displayed
  AND pane does not show error
```

---

## 05. Devices

### Device List
**P0**

```
GIVEN user presses 'd'
WHEN device overlay opens
THEN all available Spotify Connect devices listed
  AND currently active device marked with ✓ glyph
```

### Transfer Playback
**P1**

```
GIVEN device overlay open
  AND different device selected
WHEN user presses Enter
THEN playback transfers to selected device
  AND overlay closes
  AND optimistic feedback shown immediately
```

### Empty Devices
**P2**

```
GIVEN no Spotify Connect devices available
WHEN device overlay opens
THEN empty state message displayed
```

### 404 Device Error Clear (#396)
**P1**

```
GIVEN device transfer returns 404
WHEN error received
THEN device error state cleared
  AND device list returns to clean state
  AND toast shown for failed transfer
```

---

## 06. Search

### Open / Close
**P0**

```
GIVEN main TUI running
WHEN user presses '/'
THEN search overlay opens
  AND input field focused
  AND placeholder text cycles through search types
  AND overlay border uses theme Active color (#411)
  AND textinput prompt tag renders themed prefix pill
```

```
GIVEN search overlay open
WHEN user presses Esc
THEN overlay closes
  AND search state fully reset
```

### Debounced Search
**P0**

```
GIVEN search overlay open
WHEN user types query
THEN no API call made on each keystroke
  AND 300ms after last keystroke, search request fires
  AND request dispatched with api.Interactive priority
```

### Tab Cycling (7 tabs)
**P1**

```
GIVEN search overlay open with results
WHEN user presses Tab
THEN result tab cycles: All → Songs → Artists → Albums → Playlists → Shows → Episodes → All
  AND tab switch fires new SearchRequestMsg with new TabToAPITypes
  AND page resets to 1
  AND cursor resets (#404)
  AND results re-render for selected tab
```

```
GIVEN search overlay open
WHEN user presses Shift+Tab
THEN result tab cycles backward
```

### Prefix Autocomplete
**P1**

```
GIVEN search overlay open
WHEN user types `:songs` + space
THEN prefix locks to "Songs"
  AND prompt tag changes to "Search Songs"
  AND subsequent typing filters within songs only
```

```
GIVEN prefix locked to "Songs"
WHEN user presses Backspace on empty query
THEN prefix unlocks
  AND tab returns to "All"
```

### Pagination
**P1**

```
GIVEN search overlay has results with multiple pages
WHEN user presses PgDn
THEN next page of results loads
  AND pagination bar updates page number
```

```
GIVEN search overlay on page 2+
WHEN user presses PgUp
THEN previous page of results loads
```

```
GIVEN search overlay on first page
WHEN pagination bar renders
THEN prev arrow (PgUp) dimmed
```

```
GIVEN search overlay on last page
WHEN pagination bar renders
THEN next arrow (PgDn) dimmed
```

### Play Result
**P1**

```
GIVEN search overlay has results
  AND track result selected
WHEN user presses Enter
THEN track starts playing
  AND overlay remains open (does not close)
```

### Add to Queue from Search
**P2**

```
GIVEN search overlay has results
  AND track result selected
WHEN user presses Ctrl+A (or 'A')
THEN track added to playback queue
  AND confirmation toast appears
```

### Stale Request Cancellation
**P1**

```
GIVEN search request in-flight
WHEN user types new query before first response arrives
THEN first in-flight request cancelled (gen counter guard)
  AND only second request's results displayed
```

### Search Overlay Structure
**P2**

```
GIVEN search overlay open
WHEN it renders
THEN two panels shown: Search (left ~30%) + Results (right ~70%)
  AND tab bar present with 7 tabs (All/Songs/Artists/Albums/Playlists/Shows/Episodes)
  AND no bottom keybar rendered
  AND Results panel border shows action notches (ctrl+a, tab, pgdn, pgup)
```

---

## 07. Library Browser

### Playlists Pane
**P1**

```
GIVEN Playlists pane visible
WHEN playlists loaded
THEN each playlist row shows: name, track count
  AND Spotify-owned playlists show locked glyph
```

```
GIVEN Playlists pane focused
  AND playlist selected
WHEN user presses Enter
THEN track sub-view opens
  AND shows playlist's tracks in table
  AND title updates to playlist name + track count
```

```
GIVEN playlist track sub-view open
WHEN user presses Esc
THEN sub-view closes
  AND returns to playlist list view
  AND scroll position resets
```

```
GIVEN Playlists list view focused
WHEN user presses 'f'
THEN filter activates + filters playlists by name
```

### Albums Pane
**P1**

```
GIVEN Albums pane visible
WHEN albums loaded
THEN each album row shows: album name, artist, release year
```

```
GIVEN Albums pane focused
  AND album selected
WHEN user presses Enter
THEN track sub-view opens showing album's tracks
  AND border title shows album name truncated to 30 chars (#397)
```

### LikedSongs Pane
**P1**

```
GIVEN LikedSongs pane visible
WHEN songs loaded
THEN each row shows: track name, artist, duration
  AND # column shows index numbers
```

```
GIVEN LikedSongs pane focused
  AND data empty
WHEN pane renders
THEN empty state message displayed
```

### Like / Unlike Tracks (cross-pane, #384/#385)
**P1**

```
GIVEN any track-displaying pane focused (LikedSongs, Queue, TopTracks,
     RecentlyPlayed, Playlists track sub-view, Albums track sub-view,
     Search results)
  AND track selected
WHEN user presses 'l'
THEN ToggleLikeRequestMsg emitted
  AND root app applies premium gate
  AND store optimistically updated (AddLikedTrack or RemoveLikedTrack)
  AND toast shown on success ("♥ Liked" or "Unliked")
  AND ToggleLikeResultMsg carries rollback on failure
```

```
GIVEN track liked in store
WHEN any pane renders that track
THEN track name prefixed with ♥ heart glyph
```

```
GIVEN track NOT liked in store
WHEN any pane renders that track
THEN track name has NO heart prefix
```

```
GIVEN NowPlaying pane focused
  AND track selected
WHEN user presses 'l'
THEN nothing happens (NowPlaying is playback control pane, does not emit)
```

```
GIVEN Queue pane focused
  AND episode selected
WHEN user presses 'l'
THEN nothing happens (episodes not likable via /me/tracks)
```

```
GIVEN Search overlay open
  AND non-track result (album/artist/playlist/show/episode) selected
WHEN user presses 'l'
THEN nothing happens (only track results likable)
```

### Playlist Management (CRUD)
**P2**

```
GIVEN Playlists pane focused
WHEN user creates, renames, or deletes playlist
THEN operation reflects immediately (optimistic update)
  AND change persists on Spotify
```

---

## 08. Stats & Listening History

### TopTracks Pane
**P1**

```
GIVEN TopTracks pane visible on Player page
WHEN top tracks data loaded
THEN tracks listed with rank, name, artist, duration
```

```
GIVEN TopTracks pane focused
WHEN user presses 'g'
THEN time range cycles: past 4 weeks → 6 months → all time
  AND table refreshes with new data for selected range
```

```
GIVEN TopTracks pane focused
  AND track selected
WHEN user presses Enter
THEN track starts playing
  AND full list queued for playback
```

### TopArtists Pane
**P1**

```
GIVEN TopArtists pane visible
WHEN top artists data loaded
THEN artists listed with rank, name, followers, popularity
```

```
GIVEN TopArtists pane focused
WHEN user presses 'g'
THEN time range cycles independently from TopTracks
```

```
GIVEN TopArtists pane focused
  AND artist selected
WHEN user presses Enter
THEN artist context starts playing on Spotify
```

### RecentlyPlayed Pane
**P1**

```
GIVEN RecentlyPlayed pane visible
WHEN recently played data loaded
THEN tracks listed with human-readable relative timestamps
  AND timestamps show "2h ago", "yesterday", etc.
```

### Context-Aware Empty States (#406)
**P1**

```
GIVEN any table pane (TopTracks, TopArtists, RecentlyPlayed, Playlists,
     Albums, LikedSongs, FollowedShows, SavedEpisodes) has NeverFetched state
WHEN pane renders
THEN empty state shows "Loading <category>..." with Fetching status
```

```
GIVEN any table pane fetch returned Error
WHEN pane renders
THEN empty state shows "Unable to load <category>" with Error status
```

```
GIVEN any table pane throttled by gateway (IsThrottled=true)
WHEN pane renders
THEN empty state shows RateLimited status with retry-after seconds hint
```

```
GIVEN any table pane fetch succeeded with empty data
WHEN pane renders
THEN empty state shows None status with action hint
```

---

## 09. Theming

### Theme Switcher
**P0**

```
GIVEN main TUI running
WHEN user presses 't'
THEN theme switcher overlay opens
  AND all 13 available themes listed
  AND currently active theme marked with ✓
```

```
GIVEN theme switcher overlay open
WHEN user selects different theme + Enter
THEN theme applied immediately to entire UI
  AND overlay closes
```

### Theme Persistence
**P1**

```
GIVEN non-default theme selected
WHEN spotnik restarted
THEN previously selected theme still active
```

### All Themes Load
**P1**

```
GIVEN spotnik starts with any of 13 built-in themes
  (black, monokai, catppuccin, nord, light, dracula, gruvbox, rosepine,
   solarized, synthwave, tokyonight, mono-dark, mono-light)
WHEN theme loaded
THEN all color tokens populated (no missing methods)
  AND pane borders display theme's border color
  AND no hardcoded hex values appear outside theme files
```

### Mono Themes (#288)
**P1**

```
GIVEN mono-dark theme loaded
WHEN UI renders
THEN all colors grayscale (no color tokens leak)
  AND borders + bars use gray ramp only
```

```
GIVEN mono-light theme loaded
WHEN UI renders
THEN all colors grayscale with inverted background
  AND borders + bars use gray ramp only
```

### TOML Config Theme
**P1**

```
GIVEN valid TOML theme file exists in user theme directory
WHEN spotnik loads themes
THEN user theme available in theme list
  AND overrides any built-in theme with same ID
```

```
GIVEN invalid TOML theme file exists
WHEN spotnik loads themes
THEN error toast shown
  AND app continues with default theme
```

### Page Labels
**P2**

```
GIVEN main TUI running
WHEN status bar renders
THEN page label shows "Music" (not "A") for Player page
  AND shows "Stats" (not "B") for Stats page
```

---

## 10. Layout & Page Control

### Page Toggle
**P0**

```
GIVEN Player page active
WHEN user presses '0'
THEN Stats page activates
  AND layout changes to Stats preset
  AND pressing '0' again returns to Player page (2-cycle)
  AND hidden-map resets on page switch
```

### Preset Cycling
**P1**

```
GIVEN Player page active
WHEN user presses 'p'
THEN preset cycles: Dashboard → Listening → Podcast → Library → Discovery → Podcast Dashboard → Dashboard
```

```
GIVEN Stats page active
WHEN user presses 'p'
THEN only one preset (Stats) exists — no cycling occurs
```

### Pane Toggle
**P2**

```
GIVEN Player page Dashboard preset active
WHEN user presses '1', '2', '3', ... '8'
THEN corresponding pane toggles visibility
  AND remaining panes adjust to fill available space
```

```
GIVEN Stats page active
WHEN user presses '2', '3', '4', '5'
THEN corresponding Stats pane toggles visibility
  AND key '1' unused on Stats page
```

### Focus Rotation
**P1**

```
GIVEN multiple panes visible
WHEN user presses Tab
THEN focus moves to next visible pane
  AND newly focused pane's border changes to active color
```

```
GIVEN pane focused
WHEN user presses Shift+Tab
THEN focus moves to previous visible pane
```

### Layout Integrity
**P2**

```
GIVEN any preset active
WHEN terminal resized
THEN all panes resize proportionally
  AND no pane overlaps another
  AND rounded borders remain intact
```

---

## 11. Help Overlay
**P2**

```
GIVEN main TUI running
WHEN user presses '?'
THEN help overlay opens
  AND all keybindings displayed grouped by category
  AND pressing Esc closes overlay
```

---

## 12. User Profile

### Profile Display
**P2**

```
GIVEN main TUI running
WHEN user presses 'u'
THEN profile overlay opens
  AND shows: display name, subscription tier (Premium/Free), country
```

### Logout (Double-Key)
**P2**

```
GIVEN profile overlay open
WHEN user presses 'l' once
THEN confirmation prompt appears
  AND toast says "press l again to confirm"
```

```
GIVEN logout confirmation armed
WHEN user presses 'l' second time
THEN tokens cleared from keychain
  AND spotnik quits
  AND Client ID remains in config.toml
```

```
GIVEN logout confirmation armed
WHEN user presses different key
THEN confirmation cancelled
  AND new key's action processed
```

### Forget (Double-Key)
**P2**

```
GIVEN profile overlay open
WHEN user presses 'f' twice
THEN tokens cleared from keychain
  AND Client ID removed from config.toml
  AND spotnik quits
```

---

## 13. Error Handling & Resilience

### Rate Limiting (429) — Gateway Reality (#409)
**P0**

```
GIVEN Spotify returns 429 Too Many Requests
WHEN error received
THEN rate limit toast shown with countdown
  AND gateway sets backoffUntil = now + Retry-After seconds
  AND Background requests rejected immediately with *RateLimitError
  AND Interactive requests wait (blocking) for backoff expiry, then proceed
  AND token bucket applies to all priorities
  AND adaptive rate reduction on repeated 429s
  AND recovery after recoveryInterval
  AND store.SetThrottle() updated for UI observability
```

### In-Flight Dedup
**P1**

```
GIVEN Background GET request in-flight for endpoint X
WHEN second Background GET for same X arrives
THEN second caller joins as waiter
  AND receives copy of buffered response
  AND no second HTTP call made
```

```
GIVEN Background GET in-flight for endpoint X
WHEN Interactive GET for same X arrives
THEN Interactive always fires fresh HTTP call (never joins)
  AND prevents stale pre-command poll join
```

```
GIVEN PUT/POST/DELETE request in-flight
WHEN another same-method request arrives
THEN never deduplicated (non-idempotent)
```

### Priority Routing
**P1**

```
GIVEN caller tags context with api.WithPriority(ctx, api.Interactive)
WHEN request processed by gateway
THEN Interactive bypasses token bucket
  AND Interactive bypasses in-flight dedup
  AND Interactive blocked immediately during backoff (not queued)
```

```
GIVEN caller uses default Background priority
WHEN request processed by gateway
THEN Background throttled through token bucket
  AND Background joins in-flight dedup
  AND Background rejected immediately during backoff
```

### Network Error Recovery
**P1**

```
GIVEN network unavailable at startup
WHEN app launches
THEN no network errors shown in UI at launch
  AND panes start polling with exponential backoff
  AND first failure emits toast
  AND auto-recovery works when network returns
```

### Playback Poll Error Throttling
**P1**

```
GIVEN playback polling fails
WHEN exactly 3 consecutive errors occur
THEN toast notification shown
  AND subsequent errors before recovery do not spam additional toasts
```

### Context Cancellation
**P2**

```
GIVEN API request in-flight
WHEN user triggers action that cancels that request
THEN error handled gracefully (no crash, no panic)
  AND appropriate user-facing error message shown if needed
```

---

## 14. Podcast Features

### Content-Aware NowPlaying
**P0**

```
GIVEN podcast episode playing
WHEN `currently_playing_type == "episode"`
THEN NowPlaying pane renders episode info (not track info)
  AND shows show name / publisher
```

```
GIVEN track playing
WHEN `currently_playing_type == "track"`
THEN NowPlaying pane renders track info
```

### Episode Details Overlay (#334)
**P1**

```
GIVEN episode playing
WHEN user presses 'i'
THEN Episode Details overlay opens
  AND shows: episode description, show name, release date, duration
```

```
GIVEN track playing (not episode)
WHEN user presses 'i'
THEN nothing happens (silent no-op)
```

### FollowedShows Drill-Down
**P1**

```
GIVEN FollowedShows pane focused
  AND show selected
WHEN user presses Enter
THEN episode sub-view opens showing that show's episodes
  AND border title shows show name truncated to 20 chars (#397)
```

```
GIVEN episode sub-view open
WHEN user presses Esc
THEN returns to show list view
```

### SavedEpisodes Pane
**P2**

```
GIVEN SavedEpisodes pane visible
WHEN saved episodes loaded
THEN each row shows: episode name, show name, duration
```

### Podcast Dashboard Preset
**P1**

```
GIVEN user plays podcast content from search
WHEN preset auto-switches
THEN PodcastDashboard preset activates
  AND layout shows 4 panes: NowPlaying, FollowedShows, SavedEpisodes, Queue
```

### Auto-Switch Preset
**P2**

```
GIVEN user plays content from search
WHEN content type is track/album/artist
THEN preset auto-switches to Player-appropriate preset
WHEN content type is show/episode
THEN preset auto-switches to PodcastDashboard preset
```

---

## 15. Developer Tools (Stats Page)

### Page Toggle via '0'
**P2**

```
GIVEN Player page active
WHEN user presses '0'
THEN Stats page shows with: NowPlaying (compact), GatewayHealth, PollingTraffic, GatewayLive, NetworkLog
  AND Row 2 weights 1:1:3 (GatewayLive dominant, #409)
```

### GatewayHealth Pane
**P2**

```
GIVEN Stats page active
WHEN GatewayHealth pane renders
THEN 4 health rows displayed with appropriate colors:
  Tokens (dot bar), Slots (square bar), Backoff (countdown), Dedup (waiter count)
```

### GatewayLive Pane
**P2**

```
GIVEN Stats page active
WHEN GatewayLive pane renders
THEN recent API requests displayed in reverse-chronological order
  AND buffer maintains up to 500 entries
  AND boxed layout (3 sub-boxes: APP, GATEWAY, SPOTIFY) when width ≥ 60
  AND flat fallback when width < 60
```

### NetworkLog Pane
**P2**

```
GIVEN Stats page active
WHEN NetworkLog pane renders
THEN completed + blocked API requests displayed
  AND columns: TIME, METHOD, ENDPOINT, STATUS, LATENCY, PRI, DECISION, NOTES
  AND blocked requests show status 0 + "✗ blocked"
```

---

## 16. Glyph & Accessibility

### ASCII Mode
**P2**

```
GIVEN LANG=C or LC_ALL=C or LC_CTYPE=C or `ui.glyphs = "ascii"` in config
WHEN spotnik renders
THEN all borders use ASCII characters (not Unicode rounded corners)
  AND toast glyphs use ASCII equivalents (x/+/!/> instead of ✗/✓/◬/→)
  AND spinner uses ASCII frames
  AND visualizer uses ASCII bars
```

### Unicode Mode
**P2**

```
GIVEN LANG=en_US.UTF-8 (or LC_ALL/LC_CTYPE) AND `ui.glyphs = "unicode"` (or "auto")
WHEN spotnik renders
THEN all borders use Unicode rounded corners (╭╮╰╯)
  AND toast glyphs use Unicode symbols (✗/✓/◬/→)
  AND spinner uses Unicode braille frames
```

---

## 17. Scroll & Navigation

### Pane Scroll
**P1**

```
GIVEN table pane has more rows than visible height
WHEN user presses 'j' or ↓
THEN cursor moves down one row
  AND viewport scrolls when cursor reaches bottom
```

```
GIVEN table pane scrolled down
WHEN user presses 'k' or ↑
THEN cursor moves up one row
  AND viewport scrolls when cursor reaches top
```

### Universal Esc Behavior
**P1**

```
GIVEN table pane focused
  AND filter active
WHEN user presses Esc
THEN filter cleared first
```

```
GIVEN table pane focused
  AND no filter active
  AND scroll position not at top
WHEN user presses Esc
THEN scroll position resets to page 1 (top)
```

```
GIVEN overlay open
WHEN user presses Esc
THEN overlay closes (highest priority)
```

---

## 18. Golden Test Coverage

> Each behavioral test case below has corresponding golden file snapshot in
> `internal/ui/panes/testdata/`. Golden tests capture exact `View()` output
> at fixed terminal dimensions (80×24 and 40×24).
>
> **Regeneration protocol:**
> ```
> go test ./... -update              # regenerate all golden files
> make test-golden-ascii             # ASCII glyph mode (GOLDEN_MODE=ascii)
> ```
> CI runs golden tests in `make ci` — mismatch fails build. Never change
> rendering output without regenerating golden files.
>
> Golden helpers (`internal/goldentest/golden.go`):
> - `NewPaneTest(t, model, width, height)` — `golden.go:42`
> - `AssertGolden(t, got)` — `golden.go:56`
> - `WaitAndReadOutput(t, tm)` — `golden.go:96`
> - Golden path: `testdata/<TestName>.golden`

### 01. Auth & Onboarding
- Onboarding screens: `TestOnboarding_*` (register/oauth/error steps)
- Onboarding permissions overlay: `TestOnboardingPermissionsOverlay_*`

### 02. Playback Controls
- NowPlaying track: `TestNowPlayingPane_View_TrackPlaying`, `TestNowPlayingPane_View_TrackPaused`, `TestNowPlayingPane_View_TrackNoData`
- NowPlaying episode: `TestNowPlayingPane_View_EpisodePlaying`, `TestNowPlayingPane_View_EpisodePaused`
- Seek bar: `TestNowPlayingPane_View_SeekBar_AtPosition`
- Volume: `TestNowPlayingPane_View_VolumeBar`
- Compact strip: `TestNowPlayingPane_View_CompactStrip`
- Edge cases: `TestNowPlayingPane_View_AdType_EmptyState`, `TestNowPlayingPane_View_UnknownType_EmptyState`

### 03. NowPlaying Display
- Wide layout: `TestNowPlayingPane_View_Wide`
- Narrow fallback: `TestNowPlayingPane_View_NarrowFallback`

### 04. Queue
- Normal: `TestQueuePane_View_WithTracks_Normal`
- Mixed content: `TestQueuePane_View_MixedContent`
- Episodes narrow: `TestQueuePane_View_WithEpisodes_Narrow`
- Empty: `TestQueuePane_View_Empty`
- Filter: `TestQueuePane_View_FilterActive`, `TestQueuePane_View_FilterActive_NoMatches`

### 05. Devices
- Device list: `TestDevicesPane_View_Devices`
- Empty: `TestDevicesPane_View_Empty`
- Narrow: `TestDevicesPane_View_Narrow`

### 06. Search
- Idle: `TestSearchOverlay_Golden_Idle`
- With query: `TestSearchOverlay_Golden_WithQuery`
- With results: `TestSearchOverlay_Golden_WithResults`
- No results: `TestSearchOverlay_Golden_NoResults`
- Page 2: `TestSearchOverlay_Golden_Page2`
- Prefix locked: `TestSearchOverlay_Golden_PrefixLocked`
- Narrow: `TestSearchOverlay_Golden_Narrow`

### 07. Library Browser
- Playlists: `TestPlaylistsPane_View_ListView`, `TestPlaylistsPane_View_EmptyState`, `TestPlaylistsPane_View_Narrow`, `TestPlaylistsPane_View_FilterActive`, `TestPlaylistsPane_View_SpotifyOwnedLocked`, `TestPlaylistsPane_View_TrackSubView`, `TestPlaylistsPane_View_TrackSubView_FilterActive`
- Albums: `TestAlbumsPane_View_AlbumList`, `TestAlbumsPane_View_EmptyState`, `TestAlbumsPane_View_Narrow`, `TestAlbumsPane_View_FilterActive`, `TestAlbumsPane_View_TrackSubView`, `TestAlbumsPane_View_TrackSubView_FilterActive`
- LikedSongs: `TestLikedSongsPane_View_Tracks`, `TestLikedSongsPane_View_EmptyState`, `TestLikedSongsPane_View_Narrow`, `TestLikedSongsPane_View_FilterActive`
- Empty states (#406): `TestAlbumsPane_View_EmptyState_NeverFetched`, `TestAlbumsPane_View_EmptyState_Fetching`, `TestAlbumsPane_View_EmptyState_Error`, `TestAlbumsPane_View_EmptyState_RateLimited` (similar for TopArtists, FollowedShows, etc.)

### 08. Stats & Listening History
- TopTracks: `TestTopTracksPane_View_Tracks`, `TestTopTracksPane_View_EmptyState`, `TestTopTracksPane_View_Narrow`, `TestTopTracksPane_View_FilterActive`, `TestTopTracksPane_View_FilterActive_NoMatches`, `TestTopTracksPane_View_MediumTerm`
- TopArtists: `TestTopArtistsPane_View_Artists`, `TestTopArtistsPane_View_EmptyState`, `TestTopArtistsPane_View_Narrow`, `TestTopArtistsPane_View_FilterActive`, `TestTopArtistsPane_View_FilterActive_NoMatches`, `TestTopArtistsPane_View_LongTerm`
- RecentlyPlayed: `TestRecentlyPlayedPane_View_Tracks`, `TestRecentlyPlayedPane_View_EmptyState`, `TestRecentlyPlayedPane_View_Narrow`, `TestRecentlyPlayedPane_View_FilterActive`, `TestRecentlyPlayedPane_View_FilterActive_NoMatches`

### 09. Theming
- Theme overlay: `TestThemeOverlay_View_ThemeList`, `TestThemeOverlay_View_Narrow`
- Mono themes: `TestThemeOverlay_View_MonoDark`, `TestThemeOverlay_View_MonoLight` (#288)

### 11. Help Overlay
- Keybindings: `TestHelpOverlay_View_Keybindings`, `TestHelpOverlay_View_Narrow`

### 12. User Profile
- Profile overlay: `TestProfileOverlay_View_Premium`, `TestProfileOverlay_View_Free`, `TestProfileOverlay_View_Loading`, `TestProfileOverlay_View_Error`, `TestProfileOverlay_View_LogoutConfirmation`, `TestProfileOverlay_View_ForgetConfirmation`

### 14. Podcast Features
- Episode details: `TestEpisodeDetailsOverlay_View_EpisodeInfo`, `TestEpisodeDetailsOverlay_View_Narrow`
- FollowedShows: `TestFollowedShowsPane_View_Shows`, `TestFollowedShowsPane_View_EmptyState`, `TestFollowedShowsPane_View_Narrow`, `TestFollowedShowsPane_View_FilterActive`, `TestFollowedShowsPane_View_EpisodeSubView`
- SavedEpisodes: `TestSavedEpisodesPane_View_Episodes`, `TestSavedEpisodesPane_View_EmptyState`, `TestSavedEpisodesPane_View_Narrow`, `TestSavedEpisodesPane_View_FilterActive`

### 15. Developer Tools (Stats Page)
- GatewayHealth: `TestGatewayHealthPane_View_AllHealthy`, `TestGatewayHealthPane_View_MixedHealth`
- GatewayLive: `TestGatewayLivePane_View_WithEntries`, `TestGatewayLivePane_View_Empty`
- PollingTraffic: `TestPollingTrafficPane_View_Fresh`, `TestPollingTrafficPane_View_Stale`
- NetworkLog: `TestNetworkLogPane_View_WithEntries`, `TestNetworkLogPane_View_Empty`

---

## 19. CI Workflow (#418 least-privilege)

### ci.yml — Permissions
**P0**

```
GIVEN ci.yml workflow runs on push or PR
WHEN workflow executes
THEN permissions are `contents: read` only (least-privilege)
  AND no write permissions granted
  AND job runs `make ci` (fmt-check → tidy-check → lint → coverage 80% → check-glyphs → build)
  AND locale matrix runs `LANG=en_US.UTF-8` AND `LANG=C`
```

### release.yml — Sigstore Attestation
**P1**

```
GIVEN release.yml triggered by tag push `v*`
WHEN workflow executes
THEN permissions are `contents: write`, `id-token: write`, `attestations: write`
  AND goreleaser builds + publishes release
  AND Sigstore SLSA provenance attestation generated
```

### release-please.yml — Automation
**P1**

```
GIVEN release-please.yml triggered by push to main
WHEN workflow executes
THEN permissions are `contents: write`, `pull-requests: write`
  AND release-please manages version bumps + changelog
  AND release-type: go
  AND config from release-please-config.json
```

### make ci Gate Order
**P0**

```
GIVEN `make ci` invoked
WHEN sequence runs
THEN order is: fmt-check → tidy-check → lint → test-coverage → check-glyphs → build
  AND coverage threshold 80% enforced (CI fails below)
  AND golden tests run as part of suite (mismatch fails build)
```

---

*Last updated: 2026-07-18*
*Total: 19 categories, 130+ test cases*
