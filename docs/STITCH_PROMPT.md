# Google Stitch Design Prompt — spotnik

> Copy the prompt for each screen individually into Google Stitch.
> Each prompt is self-contained. Feed the Global Context block at the top of every screen prompt.
> Feed screens one at a time for best results.

---

## Global Design Context (Paste this at the TOP of every single screen prompt)

```
Design a CLI (command-line interface) screen for a developer tool called "Spotnik" — a Spotify
music player that runs entirely inside a terminal as a text-based UI (TUI).

CRITICAL — WHAT THIS IS:
- This is a pure text interface that runs inside a terminal emulator (like iTerm2, Alacritty,
  or the macOS Terminal app)
- It is NOT a web app, NOT a mobile app, NOT a desktop GUI app
- There are NO window controls (no close button, no minimize button, no maximize button)
- There are NO scroll bars
- There is NO mouse cursor or hover states
- There are NO clickable buttons, dropdowns, or form fields in the GUI sense
- Everything is keyboard-driven — there is no mouse interaction at all
- The entire UI is made of plain text characters, spaces, and Unicode box-drawing symbols
- Think: what you see when you run "htop" or "vim" or "lazygit" in a terminal

WHAT THE OUTPUT SHOULD LOOK LIKE:
- A flat, rectangular screen filled entirely with monospace text — Monokai dark theme
- No window chrome of any kind — just the raw terminal content
- The canvas background is #272822 (Monokai dark brown-grey) from edge to edge
- If you show it inside a terminal emulator window for context, show ONLY a thin, very dark
  borderless container with no title bar, no traffic-light buttons, no OS window decorations
- The best reference: imagine a screenshot of someone's terminal running "lazygit" or "btop" —
  flat, text-only, Monokai-coloured, no GUI elements whatsoever

TYPOGRAPHY:
- Monospace font ONLY throughout — JetBrains Mono, Fira Code, or similar
- Every character occupies the same fixed-width cell — like a grid
- No proportional fonts anywhere
- Text sizes: all body text the same size. Only bold weight is used for emphasis.
- No font scaling, no large hero text — everything is the same monospace character size

LAYOUT BUILDING BLOCKS:
- Panes are created using Unicode box-drawing characters: ╭ ╮ ╰ ╯ │ ─ ├ ┤ ┬ ┴ ┼
- Rounded corners ONLY: ╭╮╰╯ — never use sharp corners ┌┐└┘
- Pane borders are just these characters — NOT graphical boxes or card components
- Section headers inside panes are plain bold text followed by a ─ line divider
- Everything sits on the same flat Monokai surface — there is no elevation, shadow, or depth

COLOR PALETTE — Monokai theme (use these exact hex values, no substitutions):
  Background (entire canvas):   #272822  Monokai dark — warm dark brown-grey
  Panel surface (inside panes): #3e3d32  slightly lighter surface
  Overlay backgrounds:          #49483e  for search/device modals
  Inactive pane border:         #3e3d32  same as panel — subtle separation
  Active/focused pane border:   #66d9ef  Monokai cyan — the ONLY strong border color
  Primary text:                 #f8f8f2  Monokai near-white — all main content
  Secondary text:               #cfcfc2  slightly dimmer — artist names, subtitles
  Muted text:                   #75715e  Monokai comment brown — timestamps, counts, hints
  Selected item background:     #49483e  warm dark highlight
  Selected item text:           #f8f8f2  near-white on selected
  Section headers (bold):       #66d9ef  Monokai cyan — same as active border
  Playing indicator ▶:          #a6e22e  Monokai green — currently playing
  Seek bar fill:                #fd971f  Monokai orange
  Volume bar fill:              #fd971f  Monokai orange
  Error text:                   #f92672  Monokai pink-red
  Active device indicator ◉:    #66d9ef  Monokai cyan
  Status bar background:        #1e1f1c  deepest dark — slightly below canvas
  Status bar text:              #75715e  Monokai comment brown
  Keybinding key labels:        #66d9ef  Monokai cyan (e.g. "Space", "Tab", "j/k")
  Keybinding descriptions:      #75715e  Monokai comment brown

SYMBOLS USED (Unicode/ASCII only — no icon libraries):
  ▶   currently playing indicator (green)
  ⏮ ⏸ ▶ ⏭   transport controls
  🔀 🔁 🔂   shuffle and repeat (use text fallbacks ">?" ">>" if emoji unreliable)
  ██████   seek/volume bar fill (U+2588 FULL BLOCK)
  ░░░░░░   seek/volume bar empty (U+2591 LIGHT SHADE)
  ◉   active device (teal)
  ○   inactive device / no device (muted)
  ▸   collapsed section arrow
  ▾   expanded section arrow
  ●   search result section marker
  ✗   error symbol (red)
  █   text input cursor (blinking block)

WHAT MAKES A GOOD RESULT:
- Pure black background, text floating on nothing
- Dense, information-rich layout — like a real developer tool, not a spacious web app
- Clear visual hierarchy through color alone (bold + color, not size)
- The only colors that stand out are: ice blue (focus/active), green (playing), red (error)
- Everything else is greyscale — text on black
- It should look like a screenshot from a developer's terminal, not a design mockup
```

---

## Screen 01 — Main View (Music Playing, Player Pane Focused)

```
[PASTE GLOBAL CONTEXT ABOVE FIRST]

Draw the main screen of spotnik with music actively playing.
Show this as a flat terminal output — Monokai dark background, text only, no window chrome.

THE FULL TERMINAL CANVAS is divided into this structure from top to bottom:

━━━ ROW 1: HEADER BAR (full width, single line) ━━━
Monokai dark background (#272822). Text only:
Left side: "Spotnik" in primary text (#f8f8f2), bold
Right side: "◉ MacBook Pro Speakers" — ◉ in teal (#66d9ef), text in secondary grey (#cfcfc2)

━━━ ROWS 2 to N-1: THREE PANES SIDE BY SIDE ━━━

Separated by a single horizontal line ─ from the header, then three vertical panes:

LEFT PANE (22% of terminal width):
Border: inactive (#3e3d32 rounded corners ╭╯)
Content:
  LIBRARY          ← section header: bold, ice blue #66d9ef
  ─────────────────
  ▸ Playlists  (12)   ← all text in primary #f8f8f2, count in muted #75715e
  ▸ Albums      (8)
  ▸ Liked    (287)
  ▸ Podcasts    (3)

  ─────────────────
  RECENTLY PLAYED  ← section header: bold, ice blue
  ─────────────────
  ▶ Blinding Lights  ← ▶ in green #a6e22e
    Save Your Tears  ← 2-space indent, primary text
    Starboy
    Levitating
    Peaches

CENTER PANE (50% of terminal width) — THIS PANE IS FOCUSED:
Border: BRIGHT ICE BLUE #66d9ef rounded corners ╭╮╰╯ — clearly different from side panes
Content:
  NOW PLAYING        ← section header: bold, ice blue
  ─────────────────────────────────────
                           (empty line)
  Blinding Lights    ← track name: bold, primary text #f8f8f2
  The Weeknd         ← artist: secondary grey #cfcfc2
  After Hours        ← album: muted #75715e
                           (empty line)
  ██████████████░░░░░░░░░░░░░░   ← seek bar: ice blue fill, #3e3d32 empty
  2:34 ───────────────────── 4:12   ← time labels: muted text, ─ fill between
                           (empty line)
  ⏮    ⏸    ⏭       🔀    🔁   ← transport: ⏮⏸⏭ in primary, 🔀🔁 in muted (inactive)
  ─────────────────────────────────────
  VOL  ██████████░░░░░░  65%   ← volume: ice blue fill, muted empty, % in muted

RIGHT PANE (28% of terminal width):
Border: inactive (#3e3d32)
Content:
  QUEUE              ← section header: bold, ice blue
  ─────────────────
  ▶ NOW              ← green #a6e22e
    Blinding Lights  ← primary text
    The Weeknd       ← secondary grey
                     (empty line)
  NEXT UP            ← bold, ice blue
  ─────────────────
  1  Save Your Tears ← SELECTED: dark blue bg #49483e, primary text fg
     The Weeknd
  2  Starboy
     The Weeknd
  3  Can't Feel...
     The Weeknd
  4  In Your Eyes
     The Weeknd
  5  Repeat After Me
     Post Malone
                     (empty line)
  5 tracks remaining ← muted #75715e

━━━ LAST ROW: STATUS BAR (full width, single line) ━━━
Pure black background. Keyboard hint text only:
"  /search   j/k move   Space play   Tab pane   d devices   ? help   q quit"
Key labels like "Space", "Tab", "j/k" in ice blue #66d9ef
Descriptions like "move", "play", "pane" in muted grey #75715e

IMPORTANT RENDERING NOTES:
- No shadows, no gradients, no elevation between panes — all flat on Monokai dark (#272822)
- The ONLY visual difference between active and inactive pane is border color
- All three panes are the same background color (#3e3d32) — barely visible from canvas
- The whole thing looks like text floating on Monokai warm dark
```

---

## Screen 02 — Main View (Nothing Playing / Empty State)

```
[PASTE GLOBAL CONTEXT ABOVE FIRST]

Draw the main spotnik screen when nothing is currently playing.
Flat terminal canvas, no window chrome, pure black.

Same three-pane layout as Screen 01. Only the CENTER PANE content changes:

CENTER PANE content (player pane focused, blue border):
  NOW PLAYING
  ─────────────────────────────────────
  (large empty space — vertically centered in pane)

      ○  Nothing playing

      Open Spotify on any device
      and start playing music.

      Press d to see your devices.

  (all text above in muted #75715e)
  (○ symbol in muted grey)

LEFT PANE: unchanged from Screen 01 (library browser)
RIGHT PANE:
  QUEUE
  ─────────────────
  (centered vertically)
      Queue is empty
  (muted text)

HEADER BAR: right side shows "○ No active device" — ○ in muted, text in secondary grey
STATUS BAR: same as Screen 01
```

---

## Screen 03 — Search Overlay (Results Visible)

```
[PASTE GLOBAL CONTEXT ABOVE FIRST]

Draw the main spotnik screen with a search overlay floating in the center.
Flat terminal canvas, Monokai dark background (#272822), no window chrome.

BACKGROUND PANES: all three panes are visible but DIMMED —
render all background text in very dark muted grey (#2a2a2a) so it visually recedes.
Pane borders at #3e3d32 (almost invisible). Do not erase the background — just dim it.

SEARCH OVERLAY — floating text box in the center of the terminal:
This is NOT a GUI modal. It is a text box made of box-drawing characters.
  ╭─────────────────────────────────────────────╮
  │  🔍 Search                                  │
  │  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄  │
  │  > blinding lig█                            │
  │  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄  │
  │                                             │
  │  ● TRACKS                                   │
  │  ▶ Blinding Lights  ·  The Weeknd           │  ← SELECTED: dark blue bg #49483e
  │    Blinding Lights  ·  Sunday Service C...  │
  │    Blinding Light   ·  Maroon 5             │
  │                                             │
  │  ● ARTISTS                                  │
  │    The Weeknd                               │
  │    Sunday Service Choir                     │
  │                                             │
  │  ● PLAYLISTS                                │
  │    Blinding Pop Hits                        │
  │    Late Night Drives                        │
  │                                             │
  ╰─────────────────────────────────────────────╯

Overlay styling:
- Box border: ice blue #66d9ef (rounded ╭╮╰╯)
- Box interior background: #49483e (just barely lighter than canvas)
- "🔍 Search" header: ice blue bold
- ┄ dashed divider lines: muted #75715e
- "> blinding lig█": primary text #f8f8f2, █ is the text cursor
- "● TRACKS" / "● ARTISTS" / "● PLAYLISTS": ● in ice blue, label in ice blue bold
- Track names: primary text, artist names: secondary grey #cfcfc2
- SELECTED row: dark blue bg #49483e, ▶ in green #a6e22e before the track name
- Non-selected rows: no background, just text on #49483e

STATUS BAR (changed for search context):
"  Esc close   j/k move   Enter play   Tab section   a add to queue"
Keys in ice blue, descriptions in muted grey.
```

---

## Screen 04 — Search Overlay (Loading State)

```
[PASTE GLOBAL CONTEXT ABOVE FIRST]

Draw the spotnik search overlay immediately after the user typed — results are loading.
Flat terminal canvas. Background panes dimmed as in Screen 03.

SEARCH OVERLAY (same box dimensions as Screen 03):
  ╭─────────────────────────────────────────────╮
  │  🔍 Search                                  │
  │  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄  │
  │  > blinding█                                │
  │  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄  │
  │                                             │
  │     ⣾  Searching...                         │
  │                                             │
  │                                             │
  │                                             │
  ╰─────────────────────────────────────────────╯

- ⣾ spinner character in ice blue #66d9ef
- "Searching..." in muted grey #75715e
- No results sections shown yet — just the spinner and empty space
- Everything else same as Screen 03
```

---

## Screen 05 — Device Switcher Overlay

```
[PASTE GLOBAL CONTEXT ABOVE FIRST]

Draw the main spotnik screen with the device switcher text box open.
Flat terminal canvas. Background panes slightly dimmed.

DEVICE SWITCHER — a small text box anchored to the top-right of the terminal,
just below the header bar on the right side. NOT centered. NOT floating in the middle.

  ╭──────────────────────────────╮
  │  DEVICES                     │
  │  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄  │
  │  ◉  MacBook Pro   [active]   │  ← ◉ in teal #66d9ef, [active] in green, no bg
  │  ○  iPhone 14                │  ← SELECTED: dark blue bg #49483e
  │  ○  Kitchen Speaker          │
  │  ○  Living Room TV           │
  ╰──────────────────────────────╯

- Box border: ice blue #66d9ef, rounded corners
- Box background: #49483e
- "DEVICES" header: ice blue bold
- ┄ dashed divider: muted
- Active device line: ◉ in teal, "[active]" in green #a6e22e, no highlight background
- Selected (non-active) device: dark blue background #49483e
- Other devices: ○ in muted grey, text in primary

The rest of the screen:
- Header bar still visible: "Spotnik" left, "◉ MacBook Pro Speakers" right
- Three panes dimmed in background
- STATUS BAR: "  j/k move   Enter switch device   Esc close"
```

---

## Screen 06 — Stats View

```
[PASTE GLOBAL CONTEXT ABOVE FIRST]

Draw the Stats view of spotnik. This COMPLETELY REPLACES the three-pane layout.
Flat terminal canvas, Monokai dark background, no window chrome.

HEADER BAR (top, full width, single line):
"Spotnik  [STATS]" — "Spotnik" in primary, "[STATS]" in ice blue bold
Right side: "◉ MacBook Pro Speakers" in teal + secondary grey

MAIN BODY — split into two sections vertically:

━━━ TOP HALF: Two columns side by side ━━━

LEFT COLUMN (50% width):
  TOP TRACKS                 ← ice blue bold header
  ─────────────────────────
  Time range:  [4wk]  6mo  all
  (  [4wk] has dark blue bg #49483e, bold, ice blue text — active
     6mo and all are muted grey — inactive  )
                              (empty line)
   1  Blinding Lights   The Weeknd
   2  Levitating        Dua Lipa
   3  Save Your Tears   The Weeknd   ← SELECTED: dark blue bg
   4  Peaches           J. Bieber
   5  Mood              24kGoldn
   6  Stay              The Kid LAROI
   7  good 4 u          Olivia Rodrigo
   8  Montero           Lil Nas X
   9  drivers license   Olivia Rodrigo
  10  Kiss Me More      Doja Cat
  ─────────────────────────
  25 total  ·  j/k to scroll    ← muted text

RIGHT COLUMN (50% width, separated by │ from left):
  TOP ARTISTS                ← ice blue bold header
  ─────────────────────────
  Time range:  [4wk]  6mo  all
                              (same toggle style)
   1  The Weeknd
   2  Dua Lipa
   3  Post Malone
   4  Justin Bieber
   5  Taylor Swift
   6  Olivia Rodrigo
   7  24kGoldn
   8  The Kid LAROI
   9  Doja Cat
  10  Post Malone

━━━ BOTTOM QUARTER: Full width ━━━

  RECENTLY PLAYED                          ← ice blue bold, full width
  ──────────────────────────────────────────────────────────────────────────────
  Blinding Lights  ·  The Weeknd  ·  After Hours                    3 min ago
  Levitating       ·  Dua Lipa    ·  Future Nostalgia               18 min ago
  Starboy          ·  The Weeknd  ·  Starboy                        34 min ago
  good 4 u         ·  Olivia Rodrigo  ·  SOUR                       1 hr ago
  Stay             ·  The Kid LAROI   ·  F*CK LOVE 3                2 hr ago

  (track in primary, · separators in muted, artist/album in secondary, time right-aligned muted)

STATUS BAR: "  1 library   Tab section   j/k move   f time range   Enter play"
```

---

## Screen 07 — Playlist Manager

```
[PASTE GLOBAL CONTEXT ABOVE FIRST]

Draw the Playlist Manager view of spotnik.
Flat terminal canvas, Monokai dark background, no window chrome.

HEADER BAR: "Spotnik  [PLAYLISTS]" — [PLAYLISTS] in ice blue. Right: active device.

MAIN BODY — two vertical panes separated by │:

LEFT PANE (30% width) — inactive border (#3e3d32):
  MY PLAYLISTS           ← ice blue bold
  ───────────────────────
  ▶ Chill Vibes     (24) ← ▶ green #a6e22e (currently playing context)
    Workout Mix      (48) ← SELECTED: dark blue bg #49483e
    Late Night Cd   (112)
    Road Trip        (67)
    Coding Focus     (33)
    Deep Focus       (18)
  ───────────────────────
    + New Playlist        ← ice blue text, no ▶

RIGHT PANE (70% width) — ACTIVE border (ice blue #66d9ef):
  Chill Vibes                          ✎ Rename   + Add Track
  ─────────────────────────────────────────────────────────────
   1  Blinding Lights   ·  The Weeknd                   4:20
   2  Levitating        ·  Dua Lipa                     3:23   ← SELECTED dark blue bg
   3  Save Your Tears   ·  The Weeknd                   3:35
   4  Peaches           ·  Justin Bieber                3:18
   5  Mood              ·  24kGoldn                     2:21
   6  good 4 u          ·  Olivia Rodrigo               2:58
   7  Stay              ·  The Kid LAROI                2:38
   8  Kiss Me More      ·  Doja Cat                     3:29
  ...
  ─────────────────────────────────────────────────────────────
  24 tracks  ·  ~1hr 34min                         ← muted footer

  (track name: primary, · separator + artist: secondary grey, duration: muted right-aligned)
  (✎ Rename and + Add Track: ice blue, right-aligned in the sub-header row)

STATUS BAR: "  Enter play   r rename   n new playlist   x remove   ↑↓ reorder   1 library"
```

---

## Screen 08 — Playlist Manager (Inline New Playlist Input)

```
[PASTE GLOBAL CONTEXT ABOVE FIRST]

Draw the Playlist Manager with an active text input for creating a new playlist.
Flat terminal canvas, Monokai dark background, no window chrome.

Same layout as Screen 07. In the LEFT PANE only:
The "+ New Playlist" line is now an active text input row:

  MY PLAYLISTS
  ───────────────────────
  ▶ Chill Vibes     (24)
    Workout Mix      (48)
    Late Night Cd   (112)
    Road Trip        (67)
    Coding Focus     (33)
    Deep Focus       (18)
  ───────────────────────
  > Late Night Jazz█      ← text input row: > prompt, typed text, █ cursor
                           (ice blue border underline on this row, or just ice blue > prompt)
  Enter to save  Esc cancel  ← muted text, one line below

This input is just a text row inside the pane — NOT a GUI dialog box or popup.
The > prompt character is in ice blue. The typed text and █ cursor in primary white.
The hint text is in muted grey.

Right pane: unchanged, showing current playlist tracks.
STATUS BAR: "  Enter create playlist   Esc cancel"
```

---

## Screen 09 — Help Overlay

```
[PASTE GLOBAL CONTEXT ABOVE FIRST]

Draw the full keyboard shortcut reference screen.
Flat terminal canvas, Monokai dark background, no window chrome.
Background three-pane view is visible but fully dimmed (#2a2a2a text, #3e3d32 borders).

HELP TEXT BOX — large, centered, covering most of the terminal:
  ╭──────────────────────────────────────────────────────────────────────────╮
  │  KEYBOARD SHORTCUTS                                          ? to close  │
  │  ────────────────────────────────────────────────────────────────────── │
  │                                                                          │
  │  PLAYBACK                         NAVIGATION                            │
  │  Space    Play / Pause            j / ↓    Move down                    │
  │  n / →    Next track              k / ↑    Move up                      │
  │  p / ←    Previous track          Tab      Next pane                    │
  │  + / -    Volume up / down        Shift+Tab  Previous pane              │
  │  s        Toggle shuffle          Enter    Select / Play                 │
  │  r        Cycle repeat            Esc      Close / Cancel               │
  │  l        Like / unlike           g        Jump to top                  │
  │                                   G        Jump to bottom               │
  │                                                                          │
  │  VIEWS                            LIBRARY & SEARCH                      │
  │  1        Library                 /        Search                       │
  │  2        Stats                   a        Add to queue                 │
  │  3        Playlist manager        x        Remove item                  │
  │  d        Device switcher         PgUp/Dn  Scroll page                  │
  │  ?        This help               Ctrl+U   Clear search                 │
  │  q        Quit                                                           │
  │                                                                          │
  ╰──────────────────────────────────────────────────────────────────────────╯

Styling:
- Box border: ice blue #66d9ef, rounded corners ╭╮╰╯
- Box background: #49483e
- "KEYBOARD SHORTCUTS": ice blue bold
- "? to close": muted grey, right-aligned on header row
- ─ divider: muted
- Section headers ("PLAYBACK", "NAVIGATION", "VIEWS", "LIBRARY & SEARCH"): ice blue bold
- Key names (Space, Enter, j, k, Tab, etc.): ice blue #66d9ef
- Descriptions (Play / Pause, Move down, etc.): primary text #f8f8f2

STATUS BAR: "  ? close"
```

---

## Screen 10 — First Run / Auth Screen

```
[PASTE GLOBAL CONTEXT ABOVE FIRST]

Draw the first-run screen shown when Spotnik needs to authenticate with Spotify.
Flat terminal canvas, pure black, absolutely no window chrome.

The terminal is just: black void, one centered text box, nothing else.

  ╭──────────────────────────────────────────────────╮
  │                                                  │
  │  Spotnik                                           │
  │  ────────────────────────────────────────────    │
  │  Terminal Spotify Player                         │
  │                                                  │
  │  Connecting to Spotify...                        │
  │                                                  │
  │  Opening your browser to complete sign-in.       │
  │  If it doesn't open, visit this URL:             │
  │                                                  │
  │  https://accounts.spotify.com/authorize?...      │
  │                                                  │
  │  ⣾  Waiting for authorization                    │
  │                                                  │
  │  ────────────────────────────────────────────    │
  │  Ctrl+C to cancel                                │
  │                                                  │
  ╰──────────────────────────────────────────────────╯

- "Spotnik": bold, primary text #f8f8f2 — same size as all other monospace text, just bold
- "Terminal Spotify Player": muted #75715e
- "Connecting to Spotify...": primary text
- URL line: ice blue #66d9ef (truncated with ...)
- "⣾": spinner character in ice blue (shows the app is waiting, not frozen)
- "Waiting for authorization": primary text
- "Ctrl+C to cancel": muted #75715e
- Box border: ice blue #66d9ef
- Box background: #3e3d32
- Everything outside the box: Monokai dark #272822, empty
- No other UI elements anywhere on screen
```

---

## Screen 11 — Error State in Status Bar

```
[PASTE GLOBAL CONTEXT ABOVE FIRST]

Draw the main spotnik screen (identical to Screen 01 — music playing, three panes)
with one difference: the status bar at the bottom shows an error instead of key hints.

All three panes are completely normal and unchanged.

STATUS BAR (bottom row, full width) — CHANGED:
Instead of key hints, show:
"  ✗  No connection to Spotify. Retrying in 5s..."
- ✗ symbol: red #f92672
- All error text: red #f92672
- Status bar background: same Monokai dark #272822

This demonstrates that errors appear only in the status bar — they do NOT
interrupt the panes or show a popup. The rest of the UI stays completely intact.
The status bar is the only visual that changes.
```

---

## Screen 12 — Terminal Too Small

```
[PASTE GLOBAL CONTEXT ABOVE FIRST]

Draw the screen shown when the user's terminal is too small to run Spotnik.
Flat terminal canvas, Monokai dark background, no window chrome.

The terminal itself is drawn small and narrow to convey it's undersized.
One centered text box, nothing else on screen:

  ╭─────────────────────────────────────────╮
  │                                         │
  │  Spotnik needs more space                 │
  │                                         │
  │  Current size:   78 × 20               │
  │  Minimum needed: 100 × 24              │
  │                                         │
  │  Resize your terminal window and        │
  │  Spotnik will resume automatically.       │
  │                                         │
  ╰─────────────────────────────────────────╯

- "Spotnik needs more space": bold, primary text
- "Current size:" label: muted grey. "78 × 20" value: red #f92672 (problem)
- "Minimum needed:" label: muted grey. "100 × 24" value: green #a6e22e (target)
- Body text: muted grey
- Box border: muted #75715e (not ice blue — nothing is in focus)
- Box background: #3e3d32
- Outside the box: Monokai dark
```

---

## Tips for Getting the Best Results from Stitch

**If Stitch generates a web or mobile UI instead of a terminal:**
Add this line right after the global context:
> "Do not generate a web app, mobile app, or desktop GUI. Generate a terminal text UI screenshot — flat black background, monospace text only, no GUI elements, no buttons, no scroll bars, identical to what you'd see running htop or lazygit in a terminal."

**If the layout looks too spacious or web-like:**
> "Make the layout much denser. Reduce all line spacing. Every row should be a single monospace text line with no extra padding. The text should feel tightly packed like a real terminal."

**If pane borders look like GUI card components:**
> "Pane borders must be drawn using only these characters: ╭ ╮ ╰ ╯ │ ─. They are single-character-wide text lines, not graphical borders or shadows."

**If colors are wrong:**
> "Use these exact hex values: background #272822, active border #66d9ef, primary text #f8f8f2, secondary text #cfcfc2, green for playing indicator #a6e22e."

**If it adds a window title bar or traffic light buttons:**
> "There is no title bar, no close button, no minimize button, no window chrome of any kind. The terminal content fills the entire canvas edge to edge."

---

*Last updated: 2026-02-21*
