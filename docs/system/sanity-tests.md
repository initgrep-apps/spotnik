# Sanity Test Cases — Spotnik

> Behavioral test cases in Given/When/Then format. Designed for manual testing and future component-test automation.
> **Rule:** Any change that modifies user-facing behavior must add/update relevant cases here.

---

## Priority Tiers

| Tier | Meaning |
|------|---------|
| **P0** | Must pass before any merge. Core app broken if fails. |
| **P1** | Should pass. Significant user impact if fails. |
| **P2** | Nice to pass. Cosmetic or edge-case. |

---

## 01. Auth & Onboarding

### First Launch — Registration Flow
**P0**

```
GIVEN spotnik is launched for the first time
  AND no config.toml exists
WHEN the splash screen finishes
THEN the registration screen (Step 1) is shown
  AND the redirect URI is displayed (with configured callback port)
  AND the UI shows "enter your Client ID"
```

```
GIVEN the registration screen is showing
  AND the input field is empty
WHEN the user presses 'c'
THEN the redirect URI is copied to the clipboard
```

```
GIVEN the registration screen is showing
WHEN the user types a valid 32-character hex Client ID and presses Enter
THEN the Client ID is saved to ~/.config/spotnik/config.toml
  AND Step 2 (OAuth) begins
  AND a browser window opens with the Spotify authorization URL
```

```
GIVEN the registration screen is showing
WHEN the user types an invalid Client ID and presses Enter
THEN a validation error is shown
  AND the user can retry up to 3 times
```

### First Launch — OAuth Flow (Step 2)
**P0**

```
GIVEN Step 2 (OAuth) is showing
WHEN the user presses 'c'
THEN the full authorization URL is copied to the clipboard
```

```
GIVEN Step 2 (OAuth) is showing
WHEN the user presses 'v'
THEN the permissions overview overlay opens
```

```
GIVEN Step 2 (OAuth) is showing
  AND the user authorizes in the browser
WHEN the callback server receives the authorization code
THEN tokens are exchanged and stored in the system keychain
  AND the main TUI launches
```

### OAuth Error
**P1**

```
GIVEN Step 2 (OAuth) encounters an error
WHEN the error screen is shown
THEN pressing 'r' retries the registration flow
  AND pressing 'l' re-launches OAuth without resetting Client ID
  AND pressing 'q' quits the application
```

### Returning User — Token Restore
**P0**

```
GIVEN tokens are stored in the system keychain
WHEN spotnik is launched
THEN the splash screen shows briefly
  AND the main TUI launches without showing auth screens
  AND the NowPlaying pane shows the current track within 1s
```

### Returning User — Token Expired / Missing
**P1**

```
GIVEN tokens are missing from the keychain
  AND a Client ID exists in config.toml
WHEN spotnik is launched
THEN the auth screen (Step 2) is shown directly (no registration step)
```

### Token Refresh on 401
**P0**

```
GIVEN an API call returns a 401 Unauthorized
WHEN the error is received
THEN spotnik automatically refreshes the access token
  AND retries the original request once
  AND if refresh succeeds, the request completes normally
  AND if refresh fails, a toast notification is shown
```

### Auth CLI Commands
**P1**

```
GIVEN spotnik is invoked as `spotnik auth register`
WHEN the user follows the prompts
THEN Client ID is collected, OAuth flow runs, tokens are stored
  AND the command exits 0
```

```
GIVEN tokens are stored and Client ID is in config
WHEN `spotnik auth login` is invoked
THEN existing tokens are cleared
  AND a new OAuth flow starts
  AND exits 0
```

```
GIVEN tokens are stored
WHEN `spotnik auth logout` is invoked
THEN tokens are cleared from keychain
  AND Client ID remains in config
  AND exits 0
```

```
GIVEN tokens and Client ID are stored
WHEN `spotnik auth forget` is invoked
THEN tokens are cleared from keychain
  AND Client ID is removed from config.toml
  AND exits 0
```

```
GIVEN tokens and Client ID are stored
WHEN `spotnik auth status` is invoked
THEN prints Client ID presence (yes/no)
  AND prints token state (valid/expired/missing)
```

---

## 02. Playback Controls

### Play / Pause
**P0**

```
GIVEN the main TUI is running
  AND a track is currently playing
WHEN the user presses Space
THEN playback pauses
  AND the NowPlaying title shows the pause glyph (⏸)
  AND the visualizer stops animating
```

```
GIVEN playback is paused
WHEN the user presses Space
THEN playback resumes
  AND the NowPlaying title shows the play glyph (▶)
  AND the visualizer resumes animating
```

### Seek Forward / Backward
**P1**

```
GIVEN a track is playing at position 30s
WHEN the user presses → (right arrow)
THEN the seek bar advances ~5s
  AND a debounced seek request is sent to Spotify
```

```
GIVEN a track is playing at position 30s
WHEN the user presses ← (left arrow)
THEN the seek bar retreats ~5s
```

```
GIVEN rapid left/right arrow presses
WHEN multiple seeks are triggered
THEN only the final seek position is sent to Spotify (debounced)
```

### Skip Track
**P1**

```
GIVEN a track is playing
WHEN the user presses Shift+→
THEN the next track starts playing
  AND the NowPlaying pane updates to show the new track
```

```
GIVEN a track is playing
WHEN the user presses Shift+←
THEN the previous track starts playing
```

### Volume
**P1**

```
GIVEN current volume is 50%
WHEN the user presses +
THEN volume increases by ~5%
  AND a debounced volume change request is sent to Spotify
  AND the volume bar updates optimistically
```

```
GIVEN current volume is 50%
WHEN the user presses -
THEN volume decreases by ~5%
```

### Shuffle
**P1**

```
GIVEN shuffle is off
WHEN the user presses 's'
THEN shuffle turns on
  AND a shuffle toggle request is sent to Spotify
```

```
GIVEN shuffle is on
WHEN the user presses 's'
THEN shuffle turns off
```

### Repeat Mode
**P1**

```
GIVEN repeat is off
WHEN the user presses 'r'
THEN repeat cycles to "context" (repeat all)
  AND is displayed in the playback controls
```

```
GIVEN repeat is "off"
WHEN the user presses 'r' three times
THEN repeat cycles: off → context → track → off
  AND repeat-one shows the ↻¹ superscript icon
```

### Visualizer
**P2**

```
GIVEN the NowPlaying pane is visible
WHEN the user presses 'v'
THEN the visualizer pattern cycles to the next animation
  AND the visualizer continues animating with the new pattern
```

### Premium Gating
**P1**

```
GIVEN the user has a Spotify Free account
WHEN any playback key is pressed (Space, s, r, ←, →, Shift+←, Shift+→, +, -)
THEN playback is NOT sent to Spotify
  AND a toast is shown: "Spotify Premium required"
```

---

## 03. NowPlaying Display

### Track Mode
**P0**

```
GIVEN a track is currently playing
WHEN the NowPlaying pane renders
THEN the track name and artist are displayed in the InfoBox
  AND the album name is shown
  AND a visualizer animation is visible
  AND a seek bar shows current progress
  AND playback controls (play/pause/shuffle/repeat) are visible
```

### Episode Mode
**P1**

```
GIVEN a podcast episode is currently playing
WHEN the NowPlaying pane renders
THEN the episode name and show name are displayed in the InfoBox
  AND the seek bar shows episode duration
  AND the InfoBox border shows a podcast notch indicator
  AND the title shows "[progress] episode_name" format in compact mode
```

### Adaptive Layout — Narrow Terminal
**P1**

```
GIVEN the terminal width is very narrow (< ~60 cols)
WHEN the NowPlaying pane renders
THEN the InfoBox is dropped (does not render)
  AND the visualizer fills the full content area
  AND the seek bar remains visible
```

### Adaptive Layout — Normal Width
**P1**

```
GIVEN the terminal width is normal (>= ~80 cols)
WHEN the NowPlaying pane renders
THEN the InfoBox overlays the left ~25% of the visualizer area
  AND the seek bar is positioned on the right side
  AND equal padding surrounds the content
```

### Compact Preset
**P2**

```
GIVEN the Dashboard or Library/Discovery preset is active
  AND the NowPlaying pane height is less than 8 rows
WHEN the NowPlaying pane renders
THEN compact track info is embedded in the pane title bar
  AND controls are still visible
  AND no excess padding is shown
```

### Seek Bar Interpolation
**P1**

```
GIVEN a track is playing
WHEN 5 seconds pass between poll ticks
THEN the seek bar advances smoothly (1s local interpolation)
  AND at the next poll, the position snaps to the actual Spotify position
```

---

## 04. Queue

### Queue Display
**P0**

```
GIVEN the main TUI is running
  AND songs are queued
WHEN the user views the Queue pane
THEN upcoming tracks are listed in a table
  AND each row shows: track name, artist, duration
  AND the # column shows track numbers
```

### Queue Filter
**P1**

```
GIVEN the Queue pane is focused
WHEN the user presses 'f'
THEN the filter input activates
  AND typing a track name filters the queue table in real-time
  AND no API calls are made during filtering
```

```
GIVEN the Queue filter is active with a query
WHEN the user presses Esc
THEN the filter is cleared
  AND all tracks are shown again
```

### Queue — Play from Queue
**P1**

```
GIVEN the Queue pane is focused
  AND a track is selected
WHEN the user presses Enter
THEN that track starts playing immediately
```

### Mixed Content Queue (Tracks + Episodes)
**P1**

```
GIVEN the queue contains both tracks and podcast episodes
WHEN the Queue pane renders
THEN tracks show the ♪ symbol in the type column
  AND episodes show the ◆ symbol in the type column
```

```
GIVEN the Queue pane is focused
  AND an episode row is selected
WHEN the user presses Enter
THEN the episode starts playing
```

### Empty Queue
**P2**

```
GIVEN no tracks are queued
WHEN the Queue pane renders
THEN an empty state message is displayed
  AND the pane does not show an error
```

---

## 05. Devices

### Device List
**P0**

```
GIVEN the user presses 'd'
WHEN the device overlay opens
THEN all available Spotify Connect devices are listed
  AND the currently active device is marked with a ✓ glyph
```

### Transfer Playback
**P1**

```
GIVEN the device overlay is open
  AND a different device is selected
WHEN the user presses Enter
THEN playback transfers to the selected device
  AND the overlay closes
  AND optimistic feedback is shown immediately
```

### Empty Devices
**P2**

```
GIVEN no Spotify Connect devices are available
WHEN the device overlay opens
THEN an empty state message is displayed
```

---

## 06. Search

### Open / Close
**P0**

```
GIVEN the main TUI is running
WHEN the user presses '/'
THEN the search overlay opens
  AND the input field is focused
  AND the placeholder text cycles through search types
```

```
GIVEN the search overlay is open
WHEN the user presses Esc
THEN the overlay closes
  AND search state is fully reset
```

### Debounced Search
**P0**

```
GIVEN the search overlay is open
WHEN the user types a query
THEN no API call is made on each keystroke
  AND 300ms after the last keystroke, the search request fires
```

### Tab Cycling
**P1**

```
GIVEN the search overlay is open with results
WHEN the user presses Tab
THEN the result tab cycles: All → Songs → Artists → Albums → Playlists → All
  AND results re-render for the selected tab
```

```
GIVEN the search overlay is open
WHEN the user presses Shift+Tab
THEN the result tab cycles backward
```

### Prefix Autocomplete
**P1**

```
GIVEN the search overlay is open
WHEN the user types `:songs` followed by a space
THEN the prefix locks to "Songs"
  AND the prompt tag changes to "Search Songs"
  AND subsequent typing filters within songs only
```

```
GIVEN the prefix is locked to "Songs"
WHEN the user presses Backspace on an empty query
THEN the prefix unlocks
  AND the tab returns to "All"
```

### Pagination
**P1**

```
GIVEN the search overlay has results with multiple pages
WHEN the user presses PgDn
THEN the next page of results loads
  AND the pagination bar updates the page number
```

```
GIVEN the search overlay is on page 2+
WHEN the user presses PgUp
THEN the previous page of results loads
```

```
GIVEN the search overlay is on the first page
WHEN the pagination bar renders
THEN the prev arrow (PgUp) is dimmed
```

```
GIVEN the search overlay is on the last page
WHEN the pagination bar renders
THEN the next arrow (PgDn) is dimmed
```

### Play Result
**P1**

```
GIVEN the search overlay has results
  AND a track result is selected
WHEN the user presses Enter
THEN the track starts playing
  AND the overlay remains open (does not close)
```

### Add to Queue from Search
**P2**

```
GIVEN the search overlay has results
  AND a track result is selected
WHEN the user presses Ctrl+A
THEN the track is added to the playback queue
  AND a confirmation toast appears
```

### Clear Input
**P2**

```
GIVEN the search overlay has a query typed
WHEN the user presses Ctrl+U
THEN the input clears to empty
  AND results reset
```

### Stale Request Cancellation
**P1**

```
GIVEN a search request is in-flight
WHEN the user types a new query before the first response arrives
THEN the first in-flight request is cancelled
  AND only the second request's results are displayed
```

### Search Overlay Structure
**P2**

```
GIVEN the search overlay is open
WHEN it renders
THEN two panels are shown: Search (left ~30%) + Results (right ~70%)
  AND a tab bar is present with 5 tabs
  AND no bottom keybar is rendered
  AND the Results panel border shows action notches (ctrl+a, tab, pgdn, pgup)
```

---

## 07. Library Browser

### Playlists Pane
**P1**

```
GIVEN the Playlists pane is visible
WHEN playlists are loaded
THEN each playlist row shows: name, track count
  AND Spotify-owned playlists show a locked glyph
```

```
GIVEN the Playlists pane is focused
  AND a playlist is selected
WHEN the user presses Enter
THEN the track sub-view opens
  AND shows the playlist's tracks in a table
  AND the title updates to show the playlist name + track count
```

```
GIVEN the playlist track sub-view is open
WHEN the user presses Esc
THEN the sub-view closes
  AND returns to the playlist list view
  AND scroll position resets
```

```
GIVEN the Playlists list view is focused
WHEN the user presses 'f'
THEN the filter activates and filters playlists by name
```

### Albums Pane
**P1**

```
GIVEN the Albums pane is visible
WHEN albums are loaded
THEN each album row shows: album name, artist, release year
```

```
GIVEN the Albums pane is focused
  AND an album is selected
WHEN the user presses Enter
THEN the track sub-view opens showing the album's tracks
```

### LikedSongs Pane
**P1**

```
GIVEN the LikedSongs pane is visible
WHEN songs are loaded
THEN each row shows: track name, artist, duration
  AND the # column shows index numbers
```

```
GIVEN the LikedSongs pane is focused
  AND the data is empty
WHEN the pane renders
THEN an empty state message is displayed
```

### Playlist Management (CRUD)
**P2**

```
GIVEN the Playlists pane is focused
WHEN the user creates, renames, or deletes a playlist
THEN the operation reflects immediately (optimistic update)
  AND the change persists on Spotify
```

---

## 08. Stats & Listening History

### TopTracks Pane
**P1**

```
GIVEN the TopTracks pane is visible on the Player page
WHEN top tracks data is loaded
THEN tracks are listed with rank, name, artist, duration
```

```
GIVEN the TopTracks pane is focused
WHEN the user presses 'g'
THEN the time range cycles: past 4 weeks → 6 months → all time
  AND the table refreshes with new data for the selected range
```

```
GIVEN the TopTracks pane is focused
  AND a track is selected
WHEN the user presses Enter
THEN that track starts playing
  AND the full list is queued for playback
```

### TopArtists Pane
**P1**

```
GIVEN the TopArtists pane is visible
WHEN top artists data is loaded
THEN artists are listed with rank, name, followers, popularity
```

```
GIVEN the TopArtists pane is focused
WHEN the user presses 'g'
THEN the time range cycles independently from TopTracks
```

```
GIVEN the TopArtists pane is focused
  AND an artist is selected
WHEN the user presses Enter
THEN the artist context starts playing on Spotify
```

### RecentlyPlayed Pane
**P1**

```
GIVEN the RecentlyPlayed pane is visible
WHEN recently played data is loaded
THEN tracks are listed with human-readable relative timestamps
  AND timestamps show "2h ago", "yesterday", etc.
```

### Empty States (Stats)
**P2**

```
GIVEN a stats pane has no data
WHEN it renders
THEN an empty state message is shown instead of blank rows
```

---

## 09. Theming

### Theme Switcher
**P0**

```
GIVEN the main TUI is running
WHEN the user presses 't'
THEN the theme switcher overlay opens
  AND all 13 available themes are listed
  AND the currently active theme is marked with ✓
```

```
GIVEN the theme switcher overlay is open
WHEN the user selects a different theme and presses Enter
THEN the theme is applied immediately to the entire UI
  AND the overlay closes
```

### Theme Persistence
**P1**

```
GIVEN a non-default theme is selected
WHEN spotnik is restarted
THEN the previously selected theme is still active
```

### All Themes Load
**P1**

```
GIVEN spotnik starts with any of the 13 built-in themes
WHEN the theme is loaded
THEN all color tokens are populated (no missing methods)
  AND pane borders display the theme's border color
  AND no hardcoded hex values appear outside theme files
```

### TOML Config Theme
**P1**

```
GIVEN a valid TOML theme file exists in the user theme directory
WHEN spotnik loads themes
THEN the user theme is available in the theme list
  AND overrides any built-in theme with the same ID
```

```
GIVEN an invalid TOML theme file exists
WHEN spotnik loads themes
THEN an error toast is shown
  AND the app continues with the default theme
```

### Page Labels
**P2**

```
GIVEN the main TUI is running
WHEN the status bar renders
THEN the page label shows "Music" (not "A") for the Player page
  AND shows "Stats" (not "B") for the Stats page
```

---

## 10. Layout & Page Control

### Page Toggle
**P0**

```
GIVEN the Player page is active
WHEN the user presses '0'
THEN the Stats page activates
  AND the layout changes to the Stats preset
  AND pressing '0' again returns to the Player page
```

### Preset Cycling
**P1**

```
GIVEN the Player page is active
WHEN the user presses 'p'
THEN the preset cycles to the next one:
     Dashboard → Listening → Podcast → Library → Discovery → Podcast Dashboard → Dashboard
```

```
GIVEN the Stats page is active
WHEN the user presses 'p'
THEN only one preset (Stats) exists — no cycling occurs
```

### Pane Toggle
**P2**

```
GIVEN the Player page Dashboard preset is active
WHEN the user presses '1', '2', '3', ... '8'
THEN the corresponding pane toggles visibility
  AND remaining panes adjust to fill available space
```

### Focus Rotation
**P1**

```
GIVEN multiple panes are visible
WHEN the user presses Tab
THEN focus moves to the next visible pane
  AND the newly focused pane's border changes to the active color
```

```
GIVEN a pane is focused
WHEN the user presses Shift+Tab
THEN focus moves to the previous visible pane
```

### Layout Integrity
**P2**

```
GIVEN any preset is active
WHEN the terminal is resized
THEN all panes resize proportionally
  AND no pane overlaps another
  AND rounded borders remain intact
```

---

## 11. Help Overlay
**P2**

```
GIVEN the main TUI is running
WHEN the user presses '?'
THEN the help overlay opens
  AND all keybindings are displayed grouped by category
  AND pressing Esc closes the overlay
```

---

## 12. User Profile

### Profile Display
**P2**

```
GIVEN the main TUI is running
WHEN the user presses 'u'
THEN the profile overlay opens
  AND shows: display name, subscription tier (Premium/Free), country
```

### Logout (Double-Key)
**P2**

```
GIVEN the profile overlay is open
WHEN the user presses 'l' once
THEN a confirmation prompt appears
  AND a toast says "press l again to confirm"
```

```
GIVEN the logout confirmation is armed
WHEN the user presses 'l' a second time
THEN tokens are cleared from the keychain
  AND spotnik quits
  AND the Client ID remains in config.toml
```

```
GIVEN the logout confirmation is armed
WHEN the user presses a different key
THEN confirmation is cancelled
  AND the new key's action is processed
```

### Forget (Double-Key)
**P2**

```
GIVEN the profile overlay is open
WHEN the user presses 'f' twice
THEN tokens are cleared from the keychain
  AND the Client ID is removed from config.toml
  AND spotnik quits
```

---

## 13. Error Handling & Resilience

### Rate Limiting (429)
**P0**

```
GIVEN Spotify returns a 429 Too Many Requests
WHEN the error is received
THEN a rate limit toast is shown with countdown
  AND the gateway pauses requests for the Retry-After duration
  AND after backoff expires, requests resume
```

### Network Error Recovery
**P1**

```
GIVEN the network is unavailable at startup
WHEN the app launches
THEN no network errors are shown in the UI at launch
  AND panes start polling with exponential backoff
  AND the first failure emits a toast
  AND auto-recovery works when network returns
```

### Playback Poll Error Throttling
**P1**

```
GIVEN playback polling fails
WHEN exactly 3 consecutive errors occur
THEN a toast notification is shown
  AND subsequent errors before recovery do not spam additional toasts
```

### Context Cancellation
**P2**

```
GIVEN an API request is in-flight
WHEN the user triggers an action that cancels that request
THEN the error is handled gracefully (no crash, no panic)
  AND an appropriate user-facing error message is shown if needed
```

---

## 14. Podcast Features

### Content-Aware NowPlaying
**P0**

```
GIVEN a podcast episode is playing
WHEN `currently_playing_type == "episode"`
THEN the NowPlaying pane renders episode info (not track info)
  AND shows show name / publisher
```

```
GIVEN a track is playing
WHEN `currently_playing_type == "track"`
THEN the NowPlaying pane renders track info
```

### Episode Details Overlay
**P1**

```
GIVEN an episode is playing
WHEN the user presses 'i'
THEN the Episode Details overlay opens
  AND shows: episode description, show name, release date, duration
```

```
GIVEN a track is playing (not an episode)
WHEN the user presses 'i'
THEN nothing happens (silent no-op)
```

### FollowedShows Drill-Down
**P1**

```
GIVEN the FollowedShows pane is focused
  AND a show is selected
WHEN the user presses Enter
THEN the episode sub-view opens showing that show's episodes
```

```
GIVEN the episode sub-view is open
WHEN the user presses Esc
THEN returns to the show list view
```

### SavedEpisodes Pane
**P2**

```
GIVEN the SavedEpisodes pane is visible
WHEN saved episodes are loaded
THEN each row shows: episode name, show name, duration
```

### Auto-Switch Preset
**P2**

```
GIVEN the user plays content from search
WHEN the content type is a track/album/artist
THEN the preset auto-switches to a Player-appropriate preset
WHEN the content type is a show/episode
THEN the preset auto-switches to the Podcast preset
```

---

## 15. Developer Tools (Stats Page)

### Page Toggle via '0'
**P2**

```
GIVEN the Player page is active
WHEN the user presses '0'
THEN the Stats page shows with: NowPlaying (compact), GatewayHealth, PollingTraffic, GatewayLive, NetworkLog
```

### GatewayHealth Pane
**P2**

```
GIVEN the Stats page is active
WHEN the GatewayHealth pane renders
THEN 4 health rows are displayed with appropriate colors
```

### GatewayLive Pane
**P2**

```
GIVEN the Stats page is active
WHEN the GatewayLive pane renders
THEN recent API requests are displayed in reverse-chronological order
  AND the buffer maintains up to 500 entries
```

---

## 16. Glyph & Accessibility

### ASCII Mode
**P2**

```
GIVEN `LANG=C` or `ui.glyphs = "ascii"` in config
WHEN spotnik renders
THEN all borders use ASCII characters (not Unicode rounded corners)
  AND toast glyphs use ASCII equivalents (x/+/!/> instead of ✗/✓/◬/→)
  AND the spinner uses ASCII frames
  AND the visualizer uses ASCII bars
```

### Unicode Mode
**P2**

```
GIVEN `LANG=en_US.UTF-8` and `ui.glyphs = "unicode"` (or "auto")
WHEN spotnik renders
THEN all borders use Unicode rounded corners (╭╮╰╯)
  AND toast glyphs use Unicode symbols (✗/✓/◬/→)
  AND the spinner uses Unicode braille frames
```

---

## 17. Scroll & Navigation

### Pane Scroll
**P1**

```
GIVEN a table pane has more rows than visible height
WHEN the user presses 'j' or ↓
THEN the cursor moves down one row
  AND the viewport scrolls when the cursor reaches the bottom
```

```
GIVEN a table pane is scrolled down
WHEN the user presses 'k' or ↑
THEN the cursor moves up one row
  AND the viewport scrolls when the cursor reaches the top
```

### Universal Esc Behavior
**P1**

```
GIVEN a table pane is focused
  AND the filter is active
WHEN the user presses Esc
THEN the filter is cleared first
```

```
GIVEN a table pane is focused
  AND no filter is active
  AND the scroll position is not at the top
WHEN the user presses Esc
THEN the scroll position resets to page 1 (top)
```

```
GIVEN an overlay is open
WHEN the user presses Esc
THEN the overlay closes (highest priority)
```

---

## 18. Golden Test Coverage

> Each behavioral test case below has a corresponding golden file snapshot in
> `internal/ui/panes/testdata/`. Golden tests capture the exact `View()` output
> at fixed terminal dimensions (80×24 and 40×24). Run `go test ./... -update` to
> regenerate all golden files after intentional rendering changes.

### 01. Auth & Onboarding
- Profile overlay: `TestProfileOverlay_View_Premium`, `TestProfileOverlay_View_Free`, `TestProfileOverlay_View_Loading`, `TestProfileOverlay_View_Error`, `TestProfileOverlay_View_LogoutConfirmation`, `TestProfileOverlay_View_ForgetConfirmation`

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

### 08. Stats & Listening History
- TopTracks: `TestTopTracksPane_View_Tracks`, `TestTopTracksPane_View_EmptyState`, `TestTopTracksPane_View_Narrow`, `TestTopTracksPane_View_FilterActive`, `TestTopTracksPane_View_FilterActive_NoMatches`, `TestTopTracksPane_View_MediumTerm`
- TopArtists: `TestTopArtistsPane_View_Artists`, `TestTopArtistsPane_View_EmptyState`, `TestTopArtistsPane_View_Narrow`, `TestTopArtistsPane_View_FilterActive`, `TestTopArtistsPane_View_FilterActive_NoMatches`, `TestTopArtistsPane_View_LongTerm`
- RecentlyPlayed: `TestRecentlyPlayedPane_View_Tracks`, `TestRecentlyPlayedPane_View_EmptyState`, `TestRecentlyPlayedPane_View_Narrow`, `TestRecentlyPlayedPane_View_FilterActive`, `TestRecentlyPlayedPane_View_FilterActive_NoMatches`

### 09. Theming
- Theme overlay: `TestThemeOverlay_View_ThemeList`, `TestThemeOverlay_View_Narrow`

### 11. Help Overlay
- Keybindings: `TestHelpOverlay_View_Keybindings`, `TestHelpOverlay_View_Narrow`

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

*Last updated: 2026-06-26*
*Total: 17 categories, 100+ test cases*
