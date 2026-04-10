# Controls Cleanup & Fix — Design Spec

**Date:** 2026-04-10
**Status:** Draft — open questions noted inline

---

## Overview

A set of UI/control observations were raised covering broken keys, misleading actions,
help overlay polish, and missing features. This spec documents every observation
faithfully, the confirmed root causes from code analysis, open questions, and the
agreed fix approach for each group.

---

## Group 1 — Playback Key Bugs

### 1A. Space (play/pause) not working

**Observation:** Pressing Space does nothing — play/pause has no effect.

**Root cause (confirmed):** Both `routing.go:35` (`isPlaybackKey`) and
`nowplaying.go:355` (`handleKey`) check:
```go
msg.Type == tea.KeyRunes && string(msg.Runes) == " "
```
Bubbletea v0.27 delivers Space as `tea.KeySpace` (a named key type with empty
`Runes`), not as a rune. So Space never matches, never routes to `NowPlayingPane`,
and falls through to the focused pane which ignores it.

**Fix:**
- `routing.go` `isPlaybackKey`: add `|| m.Type == tea.KeySpace`
- `nowplaying.go` `handleKey`: add `|| msg.Type == tea.KeySpace` to the space case
- Also add `tea.KeySpace` to `isPremiumOnlyPlaybackKey` (Space triggers Play/Pause which is premium-only)

---

### 1B. `n` duplicates `→` for next track

**Observation:** Both `n` and `→` skip to next track. Duplicate binding. `←` (previous)
works fine and has no duplicate. Remove `n`.

**Root cause (confirmed):** `nowplaying.go:363-365`:
```go
case msg.Type == tea.KeyRunes && string(msg.Runes) == "n",
    msg.Type == tea.KeyRight:
    return p, emitPlaybackRequest(ActionNext)
```

**Side-effect of `n` being a playback key:** `routing.go:36` includes `"n"` in
`isPlaybackKey`, so `n` is intercepted globally and never reaches panes. This is why
`n` (new playlist) in PlaylisitsPane never fires (see Group 3).

**Fix:**
- Remove `"n"` from `isPlaybackKey` and `isPremiumOnlyPlaybackKey` in `routing.go`
- Remove the `"n"` case from `nowplaying.go:handleKey`
- Update `docs/keybinding.md`, `docs/DESIGN.md §17`, and `help_overlay.go` (remove `n` from Playback section)

---

### 1C + 1D. Volume on phone — 403, misleading toast, gateway dedup burst

**Observations (from user + screenshot analysis):**

1. Volume `+`/`-` works when MacBook is the active Spotify device.
2. Volume `+`/`-` does NOT work when phone is the active Spotify device — a
   "Spotify Premium required" toast appears.
3. The network log (screenshot) confirms: `PUT /v1/me/player/volume` requests are
   **allowed by the gateway** (decision = allowed, Interactive priority) and
   **reach Spotify**, which returns **403** with latency ~55–380ms.
4. After each 403, `handlers.go:508` triggers `fetchPlaybackStateCmd` (a GET on
   `/v1/me/player`). When multiple volume presses each trigger a refetch, those
   GET requests arrive in bursts and get **inflight-deduped** with the background
   poll. These dedup events are visible in the gateway panel as
   `→ GET /v1/me/player entered [n]`.
5. User reports "a lot of requests get deduped even though I didn't press much."

**Confirmed from code:**

- `routing.go:204-208` has a client-side `IsPremium()` gate — BUT this is NOT the
  cause here. The screenshot proves requests reach Spotify (non-zero latency, HTTP
  response received). The profile IS loaded before volume is pressed. The gate is
  NOT firing.

- The 403 comes from Spotify's API. Spotify restricts `PUT /v1/me/player/volume`
  on certain device types. Mobile devices do not support programmatic volume control
  via the Web API — this is a Spotify platform restriction, not a Premium issue.

- `handlers.go:505-510` hardcodes the toast text to "Spotify Premium required" for
  **all** `ForbiddenError` responses from playback commands. This is wrong for the
  phone-volume case where the actual restriction is device type, not subscription tier.

**Open question:** What exact message does Spotify include in the 403 response body
for this case? The `ForbiddenError.Message` field carries the Spotify-side message
(e.g. "Player command failed: Premium required" vs "Player command failed: Not
available for current device"). The correct fix depends on this. Need to log or
inspect `forbiddenErr.Message` in the volume-403 case.

**Agreed on:**
- The gateway dedup burst (observation 4/5) is **correct behavior** — it is a
  consequence of each 403 triggering a playback state refetch. Multiple rapid volume
  presses each trigger a refetch, causing a GET burst that deduplicates correctly.
  No gateway change needed for dedup.
- The misleading "Spotify Premium required" toast for volume-on-phone must be fixed.
  Two approaches:
  - **Option A (preferred):** Use `forbiddenErr.Message` verbatim in the toast for
    all playback command 403s, instead of hardcoding "Spotify Premium required".
    This surfaces Spotify's actual reason.
  - **Option B:** Detect volume-specific 403s and show "Volume control not supported
    on this device". Requires knowing the exact Spotify message first (open question above).
- The client-side `IsPremium()` gate (`routing.go:204-208`) is NOT the root cause
  here but remains a latent problem (false-positive if profile loads late). Keep it
  for now — do not remove until the toast message fix is confirmed working.

**Deferred:** The `RequestKey` uses `(Method, Path)` without query params. Volume is
`PUT` (not deduped), so this is not currently a problem. Noted for future reference.

---

## Group 2 — `t` Key Conflict: Theme vs Time Range

**Observation:** In TopTracks and TopArtists panes, pressing `t` is supposed to cycle
the time range (4wk → 6mo → all). It does nothing because `t` is intercepted globally
at `routing.go:145-151` to open the theme switcher before the pane ever sees the key.

**Decision:** Option B — change the time-range cycling key in TopTracks and
TopArtists panes to a key not claimed globally. Global keys must stay independent
of pane context.

**Proposed replacement key:** `g` (mnemonic: "go to next range"). Not used anywhere.

**Fix:**
- `toptracks_pane.go`: change `"t"` → `"g"` in the `Update` key case and `Actions()` label
- `topartists_pane.go`: same
- Update `docs/keybinding.md` (add `g: Cycle time range | TopTracks / TopArtists`)
- Update `docs/DESIGN.md §17`
- Update `help_overlay.go` `helpContent` (Pane Actions section)

---

## Group 3 — Dead / Misleading Pane Actions Cleanup

### 3A. Playlists pane: `n` (new) and `r` (rename)

**Observations:**
- `n` in Playlists: pressing it changes the current song instead of creating a
  playlist. Root cause: `"n"` is in `isPlaybackKey` (routing.go:36) so it is
  intercepted globally as "next track" before reaching the pane. The pane's `n`
  handler (`playlists_pane.go:310-314`) is never reached.
- `r` in Playlists: pressing it cycles repeat mode instead of renaming. Root cause:
  `"r"` is in `isPlaybackKey` (routing.go:37). Same interception issue.
- Both are stubbed with `// TODO(feature-53)` — create/rename operations are not
  implemented and not planned for the current scope.

**Fix:**
- `playlists_pane.go` `Actions()`: remove `{Key: "n", Label: "new"}` and
  `{Key: "r", Label: "rename"}`
- `playlists_pane.go` `handleListViewKey`: remove the `"n"` and `"r"` key cases
- Update `docs/keybinding.md`, `docs/DESIGN.md §17`, `help_overlay.go`

---

### 3B. Queue pane: `A` (add)

**Observation:** Queue border shows `A: add`. Pressing `A` does nothing. No handler
exists for `A` in `queue.go Update()`.

**Fix:**
- `queue.go` `Actions()`: remove `{Key: "A", Label: "add"}`
- Remove `A: add to queue` from help overlay Pane Actions (context note: it remains
  valid in the Search overlay — update help to clarify "Search overlay" context)

---

### 3C. Albums pane: `i` (like) shows forbidden error

**Observation:** When browsing album tracks and pressing `i`, a forbidden error toast
appears. The tracks shown are already liked (they are in the user's saved albums).
The like/unlike operation should be removed from this context.

**Root cause (from code):** The albums pane has NO `i` handler. However, `i` is
listed in the help overlay's Pane Actions section without context annotation, and
LikedSongs pane does handle `i` (always as unlike). The forbidden error the user sees
is likely from the LikedSongs pane receiving the key when focus was there, not Albums.

**Fix:**
- Update help overlay: add context annotation to `i` entry — "(Liked Songs)" or remove it entirely from Pane Actions. Since we're also removing `i` from LikedSongs (3D below), remove it from help entirely.
- No code change to albums pane needed.

---

### 3D. LikedSongs pane: `i` (unlike) getting forbidden error

**Observation:** Pressing `i` in the Liked Songs pane shows a forbidden error toast.
The user wants this operation removed.

**Root cause:** `likedsongs_pane.go:151-161` always emits `LikeTrackRequestMsg{Unlike: true}`.
The `user-library-modify` scope IS included in `auth.go:28`, so it is not a scope issue.
The 403 may come from tokens granted before the scope was added (requiring re-auth), or
a Spotify restriction on specific tracks.

**Fix:**
- `likedsongs_pane.go` `Actions()`: remove `{Key: "i", Label: "like"}`
- `likedsongs_pane.go` `handleKey`: remove the `"i"` key case
- Update `docs/keybinding.md`, `docs/DESIGN.md §17`, `help_overlay.go` (remove `i: like / unlike` from Pane Actions)

---

### 3E. Help overlay: `x` (remove track) not implemented

**Observation:** Help overlay shows `x: remove track`. The pane code has:
```go
case key.Type == tea.KeyRunes && string(key.Runes) == "x":
    // NOTE: 'x' (remove track) is out of scope for story 106 — remains non-functional.
    return p, nil
```
It is explicitly a no-op. Showing it in the help overlay is misleading.

**Fix:** Remove `{Key: "x", Label: "remove track"}` from `help_overlay.go` helpContent,
`docs/keybinding.md`, and `docs/DESIGN.md §17`. Keep the code no-op as a placeholder.

---

### 3F. Playlists track sub-view: `Shift+↑/↓` (reorder) not working + not needed

**Observation:** Shift+Up/Down is listed in the help overlay for reorder. It does not
work. User says it is not currently needed.

**Root cause:** The pane checks `key.Type == tea.KeyShiftUp` / `tea.KeyShiftDown`.
Most terminals do not send the xterm shift-arrow escape sequences that bubbletea maps
to these key types. The keys are never recognized.

**Fix:**
- `playlists_pane.go` `handleTrackViewKey`: remove the `tea.KeyShiftUp` and
  `tea.KeyShiftDown` cases
- `playlists_pane.go` `Actions()` when `inTrackView`: remove `Shift+↑/↓` action hint
- Update `docs/keybinding.md`, `docs/DESIGN.md §17`, `help_overlay.go` (remove
  `Shift+↑/↓: reorder (playlists)` from Pane Actions)

---

## Group 4 — TopArtists: Enter to Play Artist

**Observation:** Pressing Enter on an artist in the TopArtists pane does nothing.
User expects it to play all songs from the artist.

**Current state:** `topartists_pane.go:165-167` explicitly comments out Enter:
```go
// NOTE: Enter has no action for artists — artists aren't directly playable
// (PlayContextMsg requires explicit artist play support).
```

**Spotify API capability (confirmed):** `PUT /v1/me/player/play` with
`context_uri: spotify:artist:{id}` IS supported. Spotify plays the artist's top
tracks as a context (similar to album/playlist context play).

**Domain model (confirmed):** `domain.FullArtist.URI` exists (`types.go:268-269`):
```go
// URI is the Spotify URI of the artist (e.g. "spotify:artist:...").
URI string `json:"uri"`
```

**No quirks known.** Spotify's artist context play works the same as album/playlist.
The existing `buildPlayContextCmd` in commands.go handles this without changes.

**Fix:**
- `topartists_pane.go` `Update`: add `case tea.KeyEnter` that emits
  `PlayContextMsg{ContextURI: artist.URI}` for the selected artist
- `topartists_pane.go` `Actions()`: add `{Key: "Enter", Label: "play artist"}`
- Update `docs/keybinding.md` and `docs/DESIGN.md §17` (Enter: play artist | TopArtists)
- Update `help_overlay.go` Pane Actions: `Enter` entry already says "select / play" — confirm it covers this

---

## Group 5 — Help Overlay Polish

### 5A. Capitalized labels

**Observation:** Labels are lowercase ("shuffle", "help", "filter"). User wants title-case.

**Fix:** In `help_overlay.go` `helpContent`, capitalize all label strings.
Examples: `"search"` → `"Search"`, `"quit"` → `"Quit"`, `"filter"` → `"Filter"`,
`"play / pause"` → `"Play / Pause"`, etc.

### 5B. Key hints in bold

**Observation:** Key column text is not bold. User wants keys bold for visual hierarchy.

**Fix:** In `help_overlay.go` `renderColumn`, add `.Bold(true)` to `keyStyle`:
```go
keyStyle := lipgloss.NewStyle().Foreground(o.theme.KeyHint()).Bold(true)
```

### 5C. Remove `j / k: scroll` from Navigation

**Observation:** `j / k: scroll` is listed in the Navigation section. User says j/k
is implicit (works everywhere) and should not take up space in the help overlay.
Up/Down arrows also work when a pane is focused.

**Fix:** Remove `{Key: "j / k", Label: "scroll"}` from the Navigation bindings in
`helpContent`. Keep `Esc: close overlay`.

### 5D. Remove j/k labels from Stats and Network Log pane borders

**Observation:** Stats and NetworkLog pane `Actions()` show j/k scroll hints
explicitly. User says implicit — remove.

**Fix:** Check and remove any `{Key: "j/k", ...}` or scroll-hint entries from
`Actions()` in stats and network log panes.

---

## NowPlaying Actions display

**Observation:** User reports only `s` (shuffle) and `r` (repeat) are visible in the
NowPlaying pane border. Other playback controls (Space, +/-, v) are defined in
`Actions()` but not visible.

**Needs investigation:** Either the border is truncating actions (too narrow), or the
user is observing a different pane. `nowplaying.go:104-112` does define all five:
`s`, `r`, `space`, `+/-`, `v`. Check `layout.RenderPaneBorder` for action truncation
behavior. Not blocked — can be investigated during implementation.

---

## Files to Change (Summary)

| File | Change |
|---|---|
| `internal/app/routing.go` | Remove `"n"` from `isPlaybackKey`/`isPremiumOnlyPlaybackKey`; add `tea.KeySpace` to both |
| `internal/ui/panes/nowplaying.go` | Add `tea.KeySpace` to space case; remove `"n"` case |
| `internal/app/handlers.go` | Fix hardcoded "Spotify Premium required" in PlaybackCmdSentMsg ForbiddenError handler (volume on phone) |
| `internal/ui/panes/toptracks_pane.go` | Change `"t"` → `"g"` for time range key |
| `internal/ui/panes/topartists_pane.go` | Change `"t"` → `"g"`; add Enter → PlayContextMsg |
| `internal/ui/panes/playlists_pane.go` | Remove `n`, `r` from Actions + handlers; remove Shift+Up/Down |
| `internal/ui/panes/queue.go` | Remove `A` from Actions |
| `internal/ui/panes/likedsongs_pane.go` | Remove `i` from Actions + handleKey |
| `internal/ui/panes/help_overlay.go` | Remove `n`, `j/k`, `x`, `Shift+↑/↓`, `i`, `A` entries; capitalize labels; bold keys; add `g` for time range |
| `docs/keybinding.md` | Sync all changes |
| `docs/DESIGN.md §17` | Sync all changes |

---

## Open Questions Before Implementation

1. **Volume 403 message on phone:** What exact string is in `forbiddenErr.Message`
   when Spotify rejects volume for a phone device? This determines whether Option A
   (use raw message) or Option B (custom string) is better for the toast fix.

2. **NowPlaying actions truncation:** Why does the user only see `s` and `r` in the
   border? Is `layout.RenderPaneBorder` truncating actions when the pane is narrow?

3. **`i` unlike scope:** The 403 may require user to re-authenticate to pick up the
   `user-library-modify` scope. Should we add a note in the toast? Moot if we remove
   the feature entirely.
