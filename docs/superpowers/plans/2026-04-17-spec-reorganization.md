# Spec Reorganization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Collapse 27 fragmented spec feature directories into 11 high-level features that reflect the actual shape of the Spotnik product.

**Architecture:** Create 11 new feature directories (01–11) with consolidated feature.md files, move all story files from old feature dirs into the appropriate new ones, delete the 27 old dirs, and update `docs/spec/00-overview.md` to reflect the new structure. No code changes — this is purely a documentation reorganization.

**Tech Stack:** Git mv for tracked file moves, Markdown for feature.md content.

---

## Collision Warning

The old `docs/spec/features/03-playback/` has the **same directory name** as the new `03-playback` feature. Task 4 handles this with a temporary rename before creating the new dir. All other new feature names are unique relative to existing old feature dir names.

## Story 79 Note

`79-toast-theme-integration.md` lives in `16-vivid-themes/stories/` — it goes to `08-theming`.  
`79-preference-store-engine.md` lives in `17-bootstrap/stories/` — it goes to `09-auth-and-profile`.  
Both keep their filenames (different dirs, no collision).

---

## Task 1: Create branch

**Files:**
- No file changes

- [ ] **Step 1: Create and switch to feature branch**

```bash
git checkout main && git pull origin main
git checkout -b refactor/spec-reorganization
```

Expected: new branch `refactor/spec-reorganization` checked out

---

## Task 2: Create 01-ui-foundation

**Files:**
- Create: `docs/spec/features/01-ui-foundation/feature.md`
- Create: `docs/spec/features/01-ui-foundation/stories/` (directory)
- Move from `12-layout/stories/`: 15, 26, 41–44, 49, 50, 52–54
- Move from `21-help-overlay/stories/`: 108

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p docs/spec/features/01-ui-foundation/stories
```

- [ ] **Step 2: Write feature.md**

Create `docs/spec/features/01-ui-foundation/feature.md`:

```markdown
---
title: "UI Layout & Components"
status: done
---

## Description

Grid layout manager, reusable pane borders and components, responsive design presets, and the in-app help overlay. Provides the geometric foundation every other feature builds on — LayoutManager assigns coordinates to each pane, btop-style rounded borders frame content, and the preset system switches between full dashboard, listening, library, and discovery layouts. Reusable table and filter components live here. The help overlay pane renders the full keybinding reference grouped by category.

## Acceptance Criteria

- [ ] LayoutManager assigns pane coordinates without overlap at all terminal sizes
- [ ] Btop-style rounded borders render correctly and resize cleanly
- [ ] Four preset layouts switch without pane rendering artifacts
- [ ] Reusable table component renders dense sortable content across all panes
- [ ] Help overlay opens on `?` and displays all keybindings grouped by category
- [ ] All layout calculations and components covered by unit tests
```

- [ ] **Step 3: Move story files from 12-layout**

```bash
git mv docs/spec/features/12-layout/stories/15-fix-ux-polish.md                docs/spec/features/01-ui-foundation/stories/
git mv docs/spec/features/12-layout/stories/26-view-height-enforcement.md       docs/spec/features/01-ui-foundation/stories/
git mv docs/spec/features/12-layout/stories/41-grid-layout-manager.md           docs/spec/features/01-ui-foundation/stories/
git mv docs/spec/features/12-layout/stories/42-btop-borders.md                  docs/spec/features/01-ui-foundation/stories/
git mv docs/spec/features/12-layout/stories/43-responsive-design.md             docs/spec/features/01-ui-foundation/stories/
git mv docs/spec/features/12-layout/stories/44-preset-system.md                 docs/spec/features/01-ui-foundation/stories/
git mv docs/spec/features/12-layout/stories/49-reusable-table.md                docs/spec/features/01-ui-foundation/stories/
git mv docs/spec/features/12-layout/stories/50-reusable-filter.md               docs/spec/features/01-ui-foundation/stories/
git mv docs/spec/features/12-layout/stories/52-component-integration.md         docs/spec/features/01-ui-foundation/stories/
git mv docs/spec/features/12-layout/stories/53-cleanup.md                       docs/spec/features/01-ui-foundation/stories/
git mv docs/spec/features/12-layout/stories/54-fix-table-alignment.md           docs/spec/features/01-ui-foundation/stories/
```

- [ ] **Step 4: Move story files from 21-help-overlay**

```bash
git mv docs/spec/features/21-help-overlay/stories/108-help-overlay-implementation.md docs/spec/features/01-ui-foundation/stories/
```

- [ ] **Step 5: Stage and commit**

```bash
git add docs/spec/features/01-ui-foundation/
git commit -m "refactor(spec): create 01-ui-foundation (absorbs 12-layout, 21-help-overlay)"
```

---

## Task 3: Create 02-api-infrastructure

**Files:**
- Create: `docs/spec/features/02-api-infrastructure/feature.md`
- Move from `00-architecture/stories/`: 21, 22, 34, 35
- Move from `10-error-resilience/stories/`: 18, 19, 24, 27
- Move from `11-api-gateway/stories/`: 20, 23, 25, 28–33, 37–39, 65
- Move from `27-gateway-rate-protection/stories/`: 126, 127

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p docs/spec/features/02-api-infrastructure/stories
```

- [ ] **Step 2: Write feature.md**

Create `docs/spec/features/02-api-infrastructure/feature.md`:

```markdown
---
title: "API Gateway & Reliability"
status: in-progress
---

## Description

Centralized HTTP gateway that all Spotify API calls route through. Implements token-bucket rate limiting (10 req/s, burst 10), in-flight request deduplication for GET requests, priority classification (Interactive vs Background), adaptive idle polling, TTL-based response staleness tracking, and a typed error system (RateLimitError, AuthError, ValidationError). The error resilience stories establish token refresh on 401, rate-limit backoff on 429, and typed errors throughout. Architecture health stories enforce import boundaries, eliminate dead code, and align domain types. Gateway rate protection rejects Interactive requests during active backoff and applies the token bucket to user-triggered commands to prevent hold-key 429s.

## Acceptance Criteria

- [ ] All requests route through Gateway — no direct http.Client.Do calls in API methods
- [ ] Token bucket enforces 10 req/s with burst 10; Interactive requests rejected during backoff
- [ ] In-flight dedup prevents duplicate concurrent GET requests for the same endpoint
- [ ] 429 triggers backoff for Retry-After seconds with ratelimit toast; 401 triggers token refresh + retry
- [ ] Typed errors propagate to toast notifications; no inline error boxes in View()
- [ ] Import boundaries enforced: ui/ never imports api/, api/ never imports ui/
- [ ] Open: stories 21, 22, 34, 35 (architecture health gaps)
```

- [ ] **Step 3: Move story files from 00-architecture**

```bash
git mv docs/spec/features/00-architecture/stories/21-import-boundary-fixes.md  docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/00-architecture/stories/22-app-decomposition.md       docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/00-architecture/stories/34-docs-dead-code-init.md     docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/00-architecture/stories/35-type-design-alignment.md   docs/spec/features/02-api-infrastructure/stories/
```

- [ ] **Step 4: Move story files from 10-error-resilience**

```bash
git mv docs/spec/features/10-error-resilience/stories/18-fix-error-architecture.md     docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/10-error-resilience/stories/19-p0-correctness-fixes.md        docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/10-error-resilience/stories/24-typed-errors-token-provider.md docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/10-error-resilience/stories/27-error-resilience.md            docs/spec/features/02-api-infrastructure/stories/
```

- [ ] **Step 5: Move story files from 11-api-gateway**

```bash
git mv docs/spec/features/11-api-gateway/stories/20-elm-architecture-purity.md           docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/11-api-gateway/stories/23-api-interfaces-mocks.md              docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/11-api-gateway/stories/25-api-dry-refactoring.md               docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/11-api-gateway/stories/28-api-cleanup-followup.md              docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/11-api-gateway/stories/29-elm-purity-data-carrying.md          docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/11-api-gateway/stories/30-centralized-gateway.md               docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/11-api-gateway/stories/31-toast-notifications.md               docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/11-api-gateway/stories/32-ttl-staleness.md                     docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/11-api-gateway/stories/33-adaptive-idle-polling.md             docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/11-api-gateway/stories/37-gateway-hardening.md                 docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/11-api-gateway/stories/38-notification-staleness-hardening.md  docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/11-api-gateway/stories/39-idle-polish-test-gaps.md             docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/11-api-gateway/stories/65-gateway-internal-watermarks.md       docs/spec/features/02-api-infrastructure/stories/
```

- [ ] **Step 6: Move story files from 27-gateway-rate-protection**

```bash
git mv docs/spec/features/27-gateway-rate-protection/stories/126-reject-interactive-during-backoff.md docs/spec/features/02-api-infrastructure/stories/
git mv docs/spec/features/27-gateway-rate-protection/stories/127-apply-token-bucket-to-interactive.md docs/spec/features/02-api-infrastructure/stories/
```

- [ ] **Step 7: Stage and commit**

```bash
git add docs/spec/features/02-api-infrastructure/
git commit -m "refactor(spec): create 02-api-infrastructure (absorbs 00-arch, 10-error-resilience, 11-api-gateway, 27-rate-protection)"
```

---

## Task 4: Create 03-playback (collision handling required)

**Files:**
- Rename old `03-playback` → `_old-03-playback` first (name collision)
- Create: `docs/spec/features/03-playback/feature.md`
- Move from `_old-03-playback/stories/`: 03, 11, 36
- Move from `13-nowplaying/stories/`: 45, 58, 58b, 59, 60
- Move from `20-playback-context/stories/`: 105, 106, 107
- Move from `24-controls-cleanup/stories/`: 118, 119, 120, 121
- Move from `25-nowplaying-controls-polish/stories/`: 122, 123
- Move from `26-playback-correctness/stories/`: 124, 125

- [ ] **Step 1: Rename old 03-playback to avoid collision**

```bash
git mv docs/spec/features/03-playback docs/spec/features/_old-03-playback
```

- [ ] **Step 2: Create new 03-playback directory**

```bash
mkdir -p docs/spec/features/03-playback/stories
```

- [ ] **Step 3: Write feature.md**

Create `docs/spec/features/03-playback/feature.md`:

```markdown
---
title: "Playback & NowPlaying"
status: in-progress
---

## Description

Everything the user sees and does to control playback. The NowPlaying pane renders the current track with artist/album metadata, a braille/block animated visualizer, gradient seek bar, and transport control labels. A tea.Tick loop fires every 1000ms to keep Spotnik in sync with Spotify; local seek interpolation smooths the progress bar between polls. Context-aware playback fills the Spotify queue from whatever song list the user triggered play from — album, playlist, liked songs, or top tracks. Controls include play/pause (Space), skip (n), seek (←/→), volume in 1% steps (+/-), shuffle (s), and repeat (r) with superscript icon for repeat-one (↻¹). Correctness fixes remove gateway debounce and add request-aware deduplication for Interactive GETs.

## Acceptance Criteria

- [ ] Currently playing track visible within 1s of app launch
- [ ] All transport controls respond under 200ms (optimistic update shown immediately)
- [ ] Seek bar updates every 1s via local interpolation; gradient renders correctly
- [ ] NowPlaying visualizer animates in sync with playback using braille or block chars
- [ ] Context-aware playback queues the full source collection when playing any song
- [ ] Repeat-one state shows ↻¹ superscript icon; volume shows partial-block bar
- [ ] Open: stories 11 (playback UX), 36 (command safety errors)
```

- [ ] **Step 4: Move stories from _old-03-playback**

```bash
git mv docs/spec/features/_old-03-playback/stories/03-playback-controls.md  docs/spec/features/03-playback/stories/
git mv docs/spec/features/_old-03-playback/stories/11-fix-playback-ux.md     docs/spec/features/03-playback/stories/
git mv docs/spec/features/_old-03-playback/stories/36-command-safety-errors.md docs/spec/features/03-playback/stories/
```

- [ ] **Step 5: Move stories from 13-nowplaying**

```bash
git mv docs/spec/features/13-nowplaying/stories/45-nowplaying-pane.md         docs/spec/features/03-playback/stories/
git mv docs/spec/features/13-nowplaying/stories/58-split-layout.md            docs/spec/features/03-playback/stories/
git mv docs/spec/features/13-nowplaying/stories/58b-design-docs-update.md     docs/spec/features/03-playback/stories/
git mv docs/spec/features/13-nowplaying/stories/59-visualizer-engine.md       docs/spec/features/03-playback/stories/
git mv docs/spec/features/13-nowplaying/stories/60-nowplaying-redesign.md     docs/spec/features/03-playback/stories/
```

- [ ] **Step 6: Move stories from 20-playback-context**

```bash
git mv docs/spec/features/20-playback-context/stories/105-context-aware-playback-song-list-panes.md docs/spec/features/03-playback/stories/
git mv docs/spec/features/20-playback-context/stories/106-playlist-full-functionality.md           docs/spec/features/03-playback/stories/
git mv docs/spec/features/20-playback-context/stories/107-album-drill-down-track-play.md           docs/spec/features/03-playback/stories/
```

- [ ] **Step 7: Move stories from 24-controls-cleanup**

```bash
git mv docs/spec/features/24-controls-cleanup/stories/118-playback-key-bugs.md          docs/spec/features/03-playback/stories/
git mv docs/spec/features/24-controls-cleanup/stories/119-time-range-rebind-artist-play.md docs/spec/features/03-playback/stories/
git mv docs/spec/features/24-controls-cleanup/stories/120-dead-pane-actions.md          docs/spec/features/03-playback/stories/
git mv docs/spec/features/24-controls-cleanup/stories/121-help-overlay-polish.md        docs/spec/features/03-playback/stories/
```

- [ ] **Step 8: Move stories from 25-nowplaying-controls-polish**

```bash
git mv docs/spec/features/25-nowplaying-controls-polish/stories/122-repeat-superscript-icon.md    docs/spec/features/03-playback/stories/
git mv docs/spec/features/25-nowplaying-controls-polish/stories/123-volume-1pct-partial-blocks.md docs/spec/features/03-playback/stories/
```

- [ ] **Step 9: Move stories from 26-playback-correctness**

```bash
git mv docs/spec/features/26-playback-correctness/stories/124-request-aware-dedup.md        docs/spec/features/03-playback/stories/
git mv docs/spec/features/26-playback-correctness/stories/125-remove-gateway-debounce.md    docs/spec/features/03-playback/stories/
```

- [ ] **Step 10: Stage and commit**

```bash
git add docs/spec/features/03-playback/ docs/spec/features/_old-03-playback
git commit -m "refactor(spec): create 03-playback (absorbs old 03, 13-nowplaying, 20-context, 24-cleanup, 25-polish, 26-correctness)"
```

---

## Task 5: Create 04-queue-and-devices

**Files:**
- Create: `docs/spec/features/04-queue-and-devices/feature.md`
- Move from `06-queue/stories/`: 06, 12, 46
- Move from `07-devices/stories/`: 07, 13

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p docs/spec/features/04-queue-and-devices/stories
```

- [ ] **Step 2: Write feature.md**

Create `docs/spec/features/04-queue-and-devices/feature.md`:

```markdown
---
title: "Queue & Device Switching"
status: in-progress
---

## Description

Two related overlays that extend playback control. The Queue pane renders the upcoming track queue in a dense bubble-table with filter support (f key) and live tick-loop refresh. The Devices overlay lists all Spotify Connect devices the user has available and lets them transfer playback to any device with Enter. Both are accessed from Page A via keyboard shortcuts (q for queue, d for devices).

## Acceptance Criteria

- [ ] Queue pane shows upcoming tracks in dense table; refreshes on each playback poll tick
- [ ] Queue filter (f) narrows tracks by name without API calls
- [ ] Device overlay lists all available Spotify Connect devices
- [ ] Selecting a device (Enter) transfers playback within 200ms with optimistic feedback
- [ ] Both panes handle empty state (no queue, no devices) gracefully
- [ ] Open: stories 12 (queue overflow), 13 (device errors)
```

- [ ] **Step 3: Move story files from 06-queue**

```bash
git mv docs/spec/features/06-queue/stories/06-queue-management.md      docs/spec/features/04-queue-and-devices/stories/
git mv docs/spec/features/06-queue/stories/12-fix-queue-overflow.md     docs/spec/features/04-queue-and-devices/stories/
git mv docs/spec/features/06-queue/stories/46-queue-pane-migration.md   docs/spec/features/04-queue-and-devices/stories/
```

- [ ] **Step 4: Move story files from 07-devices**

```bash
git mv docs/spec/features/07-devices/stories/07-device-switcher.md      docs/spec/features/04-queue-and-devices/stories/
git mv docs/spec/features/07-devices/stories/13-fix-devices-errors.md   docs/spec/features/04-queue-and-devices/stories/
```

- [ ] **Step 5: Stage and commit**

```bash
git add docs/spec/features/04-queue-and-devices/
git commit -m "refactor(spec): create 04-queue-and-devices (absorbs 06-queue, 07-devices)"
```

---

## Task 6: Create 05-library

**Files:**
- Create: `docs/spec/features/05-library/feature.md`
- Move from `04-library/stories/`: 04, 10, 47
- Move from `09-playlists/stories/`: 09

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p docs/spec/features/05-library/stories
```

- [ ] **Step 2: Write feature.md**

Create `docs/spec/features/05-library/feature.md`:

```markdown
---
title: "Library Browser & Playlists"
status: in-progress
---

## Description

Three dedicated panes for browsing the user's music library: Playlists, Albums, and LikedSongs. Each is an independent grid pane with dense bubble-table rows and filter support. Selecting a playlist or album drills into its track list. The Playlist Manager extends the playlists pane with full CRUD — create (n), rename (r), delete, and track reordering (Shift+↑/↓). Lazy loading fetches paginated results from Spotify as the user scrolls.

## Acceptance Criteria

- [ ] Playlists, Albums, and LikedSongs each render as independent grid panes
- [ ] Entering a playlist or album shows its track list with album art metadata
- [ ] Playlist create/rename/delete operations reflect immediately (optimistic update)
- [ ] Track reordering with Shift+↑/↓ persists to Spotify API
- [ ] Filter (f key) narrows rows without additional API calls
- [ ] Open: story 10 (library display fixes)
```

- [ ] **Step 3: Move story files from 04-library**

```bash
git mv docs/spec/features/04-library/stories/04-library-browser.md    docs/spec/features/05-library/stories/
git mv docs/spec/features/04-library/stories/10-fix-library-display.md docs/spec/features/05-library/stories/
git mv docs/spec/features/04-library/stories/47-library-split.md       docs/spec/features/05-library/stories/
```

- [ ] **Step 4: Move story files from 09-playlists**

```bash
git mv docs/spec/features/09-playlists/stories/09-playlist-manager.md docs/spec/features/05-library/stories/
```

- [ ] **Step 5: Stage and commit**

```bash
git add docs/spec/features/05-library/
git commit -m "refactor(spec): create 05-library (absorbs 04-library, 09-playlists)"
```

---

## Task 7: Create 06-search

**Files:**
- Create: `docs/spec/features/06-search/feature.md`
- Move from `05-search/stories/`: 05, 16
- Move from `19-search-redesign/stories/`: 81–104

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p docs/spec/features/06-search/stories
```

- [ ] **Step 2: Write feature.md**

Create `docs/spec/features/06-search/feature.md`:

```markdown
---
title: "Search"
status: in-progress
---

## Description

Full-screen search overlay opened with `/`. Queries Spotify across four tabs (All / Songs / Artists / Albums / Playlists) with a 300ms debounce universal to all input events. Results render with rich metadata via custom list delegates — album art year, artist genre, playlist track count. Prefix autocomplete (`:songs`, `:artists`, `:albums`, `:playlists`) narrows search to a specific type and is promoted to a prompt tag. Pagination controls (Ctrl+←/→) page through results. Per-page context cancellation prevents stale results from earlier keystrokes overwriting newer ones. Store cleanup and message type refactors keep the data flow Elm-pure.

## Acceptance Criteria

- [ ] `/` opens search overlay; Escape closes it and resets state
- [ ] Debounce fires 300ms after last keypress — no per-keystroke API calls
- [ ] Results display across four tabs with rich metadata per result type
- [ ] Prefix autocomplete narrows to a single type and shows prompt tag
- [ ] Ctrl+←/→ pages through results for the current tab
- [ ] Stale in-flight requests cancelled when a new search supersedes them
- [ ] Enter plays selected item; Ctrl+a adds to queue
- [ ] Open: story 16 (search result fixes)
```

- [ ] **Step 3: Move story files from 05-search**

```bash
git mv docs/spec/features/05-search/stories/05-search-overlay.md      docs/spec/features/06-search/stories/
git mv docs/spec/features/05-search/stories/16-fix-search-results.md   docs/spec/features/06-search/stories/
```

- [ ] **Step 4: Move story files from 19-search-redesign**

```bash
git mv docs/spec/features/19-search-redesign/stories/81-search-store-pagination.md           docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/82-search-overlay-layout.md             docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/83-search-prefetch-engine.md            docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/84-search-list-delegate.md              docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/85-search-prefix-autocomplete.md        docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/86-search-cleanup-integration.md        docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/87-search-rich-results.md               docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/88-search-ux-polish.md                  docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/89-search-prefix-ux.md                  docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/90-search-panel-layout.md               docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/91-search-post-impl-fixes.md            docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/92-search-list-focus-styling.md         docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/93-search-fix-stale-list-height.md      docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/94-search-fix-spinner-feedback.md       docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/95-search-fix-help-bar-colors.md        docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/96-search-fix-stale-overlay-state-on-reopen.md docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/97-search-store-cleanup.md              docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/98-search-message-types-refactor.md     docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/99-search-universal-debounce.md         docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/100-search-app-context-cancellation.md  docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/101-search-commands-per-page-fetch.md   docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/102-search-pagination-controls-view.md  docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/103-search-gateway-interactive-debounce.md docs/spec/features/06-search/stories/
git mv docs/spec/features/19-search-redesign/stories/104-search-integration-tests-coverage.md docs/spec/features/06-search/stories/
```

- [ ] **Step 5: Stage and commit**

```bash
git add docs/spec/features/06-search/
git commit -m "refactor(spec): create 06-search (absorbs 05-search, 19-search-redesign)"
```

---

## Task 8: Create 07-stats

**Files:**
- Create: `docs/spec/features/07-stats/feature.md`
- Move from `08-stats/stories/`: 08, 14, 48, 55

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p docs/spec/features/07-stats/stories
```

- [ ] **Step 2: Write feature.md**

Create `docs/spec/features/07-stats/feature.md`:

```markdown
---
title: "Stats & Listening History"
status: in-progress
---

## Description

Three panes displaying the authenticated user's Spotify listening history: TopTracks, TopArtists, and RecentlyPlayed. Each is an independent grid pane backed by the Spotify `/me/top/` and `/me/player/recently-played` endpoints. Time range cycles between past 4 weeks, 6 months, and all time via the `g` key. TopArtists supports Enter to play the artist's top tracks. RecentlyPlayed shows relative timestamps (FormatRelativeTime).

## Acceptance Criteria

- [ ] TopTracks, TopArtists, and RecentlyPlayed render as independent grid panes
- [ ] Time range (g key) cycles short/medium/long correctly for both top tracks and top artists
- [ ] TopArtists Enter plays the selected artist
- [ ] RecentlyPlayed shows human-readable relative timestamps
- [ ] Empty state handled cleanly when no history is available
- [ ] Open: story 55 (recently played empty state fix)
```

- [ ] **Step 3: Move story files from 08-stats**

```bash
git mv docs/spec/features/08-stats/stories/08-stats-dashboard.md         docs/spec/features/07-stats/stories/
git mv docs/spec/features/08-stats/stories/14-fix-views-rendering.md      docs/spec/features/07-stats/stories/
git mv docs/spec/features/08-stats/stories/48-stats-split.md              docs/spec/features/07-stats/stories/
git mv docs/spec/features/08-stats/stories/55-fix-recently-played-empty.md docs/spec/features/07-stats/stories/
```

- [ ] **Step 4: Stage and commit**

```bash
git add docs/spec/features/07-stats/
git commit -m "refactor(spec): create 07-stats (absorbs 08-stats)"
```

---

## Task 9: Create 08-theming

**Files:**
- Create: `docs/spec/features/08-theming/feature.md`
- Move from `01-theme/stories/`: 01, 40
- Move from `16-vivid-themes/stories/`: 70–75, 77–79

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p docs/spec/features/08-theming/stories
```

- [ ] **Step 2: Write feature.md**

Create `docs/spec/features/08-theming/feature.md`:

```markdown
---
title: "Theming & Appearance"
status: done
---

## Description

Token-based color theming with a Theme interface implemented by 11 built-in themes. Original themes (black, dracula-inspired, etc.) established the Theme interface and 16 semantic color tokens. Vivid themes added TOML config-driven loading, always-colorful pane borders, per-column accent colors, and 6 additional vibrant themes (Dracula, Gruvbox, Tokyo Night, Rose Pine, Solarized, Synthwave). The runtime theme switcher overlay (`t` key) lets users preview and select themes; the selection persists on exit via the preference store.

## Acceptance Criteria

- [ ] Theme interface implemented by all 11 built-in themes — no missing methods
- [ ] TOML theme files load correctly at startup; invalid TOML shows error toast
- [ ] Pane borders always display the active theme's border color
- [ ] Runtime switcher (`t`) previews and applies themes without restart
- [ ] Selected theme persists across app restarts via preference store
- [ ] No hardcoded hex values in component code — all colors via Theme tokens
```

- [ ] **Step 3: Move story files from 01-theme**

```bash
git mv docs/spec/features/01-theme/stories/01-theme-infrastructure.md docs/spec/features/08-theming/stories/
git mv docs/spec/features/01-theme/stories/40-extended-tokens.md       docs/spec/features/08-theming/stories/
```

- [ ] **Step 4: Move story files from 16-vivid-themes**

```bash
git mv docs/spec/features/16-vivid-themes/stories/70-config-driven-themes.md                docs/spec/features/08-theming/stories/
git mv docs/spec/features/16-vivid-themes/stories/71-colorful-borders-and-columns.md        docs/spec/features/08-theming/stories/
git mv docs/spec/features/16-vivid-themes/stories/72-new-theme-files.md                     docs/spec/features/08-theming/stories/
git mv docs/spec/features/16-vivid-themes/stories/73-theme-switcher-overlay.md              docs/spec/features/08-theming/stories/
git mv docs/spec/features/16-vivid-themes/stories/74-quick-theme-fixes.md                   docs/spec/features/08-theming/stories/
git mv docs/spec/features/16-vivid-themes/stories/75-ui-polish-borders-statusbar-overlays.md docs/spec/features/08-theming/stories/
git mv docs/spec/features/16-vivid-themes/stories/77-overlay-border-statusbar-fixes.md      docs/spec/features/08-theming/stories/
git mv docs/spec/features/16-vivid-themes/stories/78-border-statusbar-fix-v2.md             docs/spec/features/08-theming/stories/
git mv docs/spec/features/16-vivid-themes/stories/79-toast-theme-integration.md             docs/spec/features/08-theming/stories/
```

- [ ] **Step 5: Stage and commit**

```bash
git add docs/spec/features/08-theming/
git commit -m "refactor(spec): create 08-theming (absorbs 01-theme, 16-vivid-themes)"
```

---

## Task 10: Create 09-auth-and-profile

**Files:**
- Create: `docs/spec/features/09-auth-and-profile/feature.md`
- Move from `02-auth/stories/`: 02, 17
- Move from `17-bootstrap/stories/`: 76, 79-preference-store-engine.md, 80
- Move from `23-user-profile-subscription/stories/`: 114–117

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p docs/spec/features/09-auth-and-profile/stories
```

- [ ] **Step 2: Write feature.md**

Create `docs/spec/features/09-auth-and-profile/feature.md`:

```markdown
---
title: "Auth, Bootstrap & User Profile"
status: in-progress
---

## Description

Authentication via PKCE OAuth flow, token refresh on 401, and keychain-backed token storage. First-launch bootstrap generates a config file with embedded default client ID and prompts the user through setup. The preference store persists user choices (theme, layout preset, visualizer type) with debounced flush on change. The user profile overlay (`u` key) displays name, subscription tier, and country. Premium gating blocks playback controls and device transfer for Free users with a splash toast notice.

## Acceptance Criteria

- [ ] PKCE OAuth flow completes and stores tokens in system keychain
- [ ] Token refresh fires automatically on 401; original request retried once
- [ ] First launch creates config file with embedded client ID; no manual setup required
- [ ] Preference store persists theme/preset/visualizer selection across restarts
- [ ] Profile overlay (`u`) displays name, subscription tier, and country
- [ ] Playback keys and device transfer blocked for Free tier with Premium-required toast
- [ ] Open: story 17 (auth UX improvements)
```

- [ ] **Step 3: Move story files from 02-auth**

```bash
git mv docs/spec/features/02-auth/stories/02-oauth-authentication.md docs/spec/features/09-auth-and-profile/stories/
git mv docs/spec/features/02-auth/stories/17-fix-auth-ux.md           docs/spec/features/09-auth-and-profile/stories/
```

- [ ] **Step 4: Move story files from 17-bootstrap**

```bash
git mv docs/spec/features/17-bootstrap/stories/76-bootstrap-config-embedded-clientid.md docs/spec/features/09-auth-and-profile/stories/
git mv docs/spec/features/17-bootstrap/stories/79-preference-store-engine.md             docs/spec/features/09-auth-and-profile/stories/
git mv docs/spec/features/17-bootstrap/stories/80-wire-preset-visualizer-persistence.md  docs/spec/features/09-auth-and-profile/stories/
```

- [ ] **Step 5: Move story files from 23-user-profile-subscription**

```bash
git mv docs/spec/features/23-user-profile-subscription/stories/114-data-layer.md              docs/spec/features/09-auth-and-profile/stories/
git mv docs/spec/features/23-user-profile-subscription/stories/115-profile-ui.md              docs/spec/features/09-auth-and-profile/stories/
git mv docs/spec/features/23-user-profile-subscription/stories/116-subscription-gating.md     docs/spec/features/09-auth-and-profile/stories/
git mv docs/spec/features/23-user-profile-subscription/stories/117-profile-overlay-ux-fixes.md docs/spec/features/09-auth-and-profile/stories/
```

- [ ] **Step 6: Stage and commit**

```bash
git add docs/spec/features/09-auth-and-profile/
git commit -m "refactor(spec): create 09-auth-and-profile (absorbs 02-auth, 17-bootstrap, 23-user-profile)"
```

---

## Task 11: Create 10-developer-tools

**Files:**
- Create: `docs/spec/features/10-developer-tools/feature.md`
- Move from `14-nerd-status/stories/`: 51, 56, 61–69
- Move from `22-developer-foundations/stories/`: 109–113

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p docs/spec/features/10-developer-tools/stories
```

- [ ] **Step 2: Write feature.md**

Create `docs/spec/features/10-developer-tools/feature.md`:

```markdown
---
title: "Developer Visibility (Page B)"
status: done
---

## Description

Page B (`0` key) surfaces the internals of Spotnik's API layer for developers. The RequestFlow pane visualizes each gateway decision in real time — showing request type, priority, dedup result, rate-limit status, and backoff state — with a replay engine for stepping through past events. The NetworkLog pane is a scrollable table of every API request with timestamp, method, endpoint, HTTP status, priority classification, and gateway decision. Developer foundations stories add onboarding docs, test infrastructure, StateReader interface, BasePane pattern, and RebuildTableTheme helper.

## Acceptance Criteria

- [ ] Page B (`0` key) toggles between Page A (music) and Page B (developer view)
- [ ] RequestFlow pane renders each gateway decision with correct decision reason
- [ ] Replay engine steps through past request flow events correctly
- [ ] NetworkLog scrolls through all requests; filter (f) narrows by endpoint or status
- [ ] StateReader interface decouples panes from concrete Store type for testing
- [ ] All developer panes covered by unit tests using StateReader mocks
```

- [ ] **Step 3: Move story files from 14-nerd-status**

```bash
git mv docs/spec/features/14-nerd-status/stories/51-request-flow-pane.md     docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/14-nerd-status/stories/56-fix-request-flow-data.md  docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/14-nerd-status/stories/61-network-log-pane.md       docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/14-nerd-status/stories/62-gateway-events-pane.md    docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/14-nerd-status/stories/63-requestflow-boxed-guards.md docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/14-nerd-status/stories/64-page-b-integration.md     docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/14-nerd-status/stories/66-request-flow-replay.md    docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/14-nerd-status/stories/67-request-flow-boxed.md     docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/14-nerd-status/stories/68-gateway-event-journal.md  docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/14-nerd-status/stories/69-network-log-polish.md     docs/spec/features/10-developer-tools/stories/
```

- [ ] **Step 4: Move story files from 22-developer-foundations**

```bash
git mv docs/spec/features/22-developer-foundations/stories/109-onboarding-docs-test-infra.md       docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/22-developer-foundations/stories/110-statereader-file-splits-table-tests.md docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/22-developer-foundations/stories/111-basepane-tablehelper-auth-fix.md     docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/22-developer-foundations/stories/112-test-coverage-gaps.md               docs/spec/features/10-developer-tools/stories/
git mv docs/spec/features/22-developer-foundations/stories/113-statereader-cleanup-nil-guard.md     docs/spec/features/10-developer-tools/stories/
```

- [ ] **Step 5: Stage and commit**

```bash
git add docs/spec/features/10-developer-tools/
git commit -m "refactor(spec): create 10-developer-tools (absorbs 14-nerd-status, 22-developer-foundations)"
```

---

## Task 12: Create 11-cicd

**Files:**
- Create: `docs/spec/features/11-cicd/feature.md`
- Move from `15-cicd/stories/`: 57

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p docs/spec/features/11-cicd/stories
```

- [ ] **Step 2: Write feature.md**

Create `docs/spec/features/11-cicd/feature.md`:

```markdown
---
title: "CI/CD & Release"
status: open
---

## Description

GitHub Actions pipeline and GoReleaser configuration for automated multi-platform distribution. The pipeline runs lint, tests, and coverage gate on every PR, then on merge to main builds release binaries for macOS (amd64/arm64) and Linux (amd64) via GoReleaser. Artifacts are attached to GitHub Releases.

## Acceptance Criteria

- [ ] `make ci` (lint + test + 80% coverage) runs cleanly on GitHub Actions for every PR
- [ ] Merge to main triggers GoReleaser build producing macOS and Linux binaries
- [ ] GitHub Release created automatically with binaries attached
- [ ] Pipeline fails fast on lint or coverage threshold violations
```

- [ ] **Step 3: Move story file from 15-cicd**

```bash
git mv docs/spec/features/15-cicd/stories/57-cicd-release-pipeline.md docs/spec/features/11-cicd/stories/
```

- [ ] **Step 4: Stage and commit**

```bash
git add docs/spec/features/11-cicd/
git commit -m "refactor(spec): create 11-cicd (absorbs 15-cicd)"
```

---

## Task 13: Delete all old feature directories

- [ ] **Step 1: Remove old feature directories**

```bash
git rm -r docs/spec/features/00-architecture
git rm -r docs/spec/features/01-theme
git rm -r docs/spec/features/02-auth
git rm -r docs/spec/features/_old-03-playback
git rm -r docs/spec/features/04-library
git rm -r docs/spec/features/05-search
git rm -r docs/spec/features/06-queue
git rm -r docs/spec/features/07-devices
git rm -r docs/spec/features/08-stats
git rm -r docs/spec/features/09-playlists
git rm -r docs/spec/features/10-error-resilience
git rm -r docs/spec/features/11-api-gateway
git rm -r docs/spec/features/12-layout
git rm -r docs/spec/features/13-nowplaying
git rm -r docs/spec/features/14-nerd-status
git rm -r docs/spec/features/15-cicd
git rm -r docs/spec/features/16-vivid-themes
git rm -r docs/spec/features/17-bootstrap
git rm -r docs/spec/features/19-search-redesign
git rm -r docs/spec/features/20-playback-context
git rm -r docs/spec/features/21-help-overlay
git rm -r docs/spec/features/22-developer-foundations
git rm -r docs/spec/features/23-user-profile-subscription
git rm -r docs/spec/features/24-controls-cleanup
git rm -r docs/spec/features/25-nowplaying-controls-polish
git rm -r docs/spec/features/26-playback-correctness
git rm -r docs/spec/features/27-gateway-rate-protection
```

- [ ] **Step 2: Commit deletion**

```bash
git commit -m "refactor(spec): delete all old feature directories"
```

---

## Task 14: Update docs/spec/00-overview.md

**Files:**
- Modify: `docs/spec/00-overview.md` — replace feature table

- [ ] **Step 1: Replace the Features table in 00-overview.md**

Replace the entire `## Features` table with:

```markdown
## Features

| # | Feature | Path | Status | Stories | Description |
|---|---------|------|--------|---------|-------------|
| 01 | UI Layout & Components | `features/01-ui-foundation/` | done | 15, 26, 41–44, 49, 50, 52–54, 108 | Grid layout manager, btop borders, reusable table/filter components, help overlay |
| 02 | API Gateway & Reliability | `features/02-api-infrastructure/` | in-progress | 18–35, 37–39, 65, 126–127 | Centralized gateway, rate limiting, dedup, error types, architecture health |
| 03 | Playback & NowPlaying | `features/03-playback/` | in-progress | 03, 11, 36, 45, 58–60, 105–107, 118–125 | Transport controls, NowPlaying display, visualizer, context-aware playback, polish |
| 04 | Queue & Device Switching | `features/04-queue-and-devices/` | in-progress | 06, 07, 12, 13, 46 | Queue viewer pane, Spotify Connect device selection |
| 05 | Library Browser & Playlists | `features/05-library/` | in-progress | 04, 09, 10, 47 | Browse playlists/albums/liked songs, full playlist management |
| 06 | Search | `features/06-search/` | in-progress | 05, 16, 81–104 | Full-screen overlay, multi-tab results, prefix autocomplete, pagination |
| 07 | Stats & Listening History | `features/07-stats/` | in-progress | 08, 14, 48, 55 | Top tracks, top artists, recently played with time-range cycling |
| 08 | Theming & Appearance | `features/08-theming/` | done | 01, 40, 70–75, 77–79 | Token-based themes, TOML config, runtime switcher, 11 built-in themes |
| 09 | Auth, Bootstrap & User Profile | `features/09-auth-and-profile/` | in-progress | 02, 17, 76, 79*, 80, 114–117 | PKCE OAuth, token refresh, first-launch bootstrap, profile overlay, Premium gating |
| 10 | Developer Visibility (Page B) | `features/10-developer-tools/` | done | 51, 56, 61–69, 109–113 | Request flow pane, network log, developer foundations |
| 11 | CI/CD & Release | `features/11-cicd/` | open | 57 | GitHub Actions, GoReleaser, multi-platform distribution |
```

- [ ] **Step 2: Update the last-updated date in 00-overview.md**

Change the footer line to:

```markdown
*Last updated: 2026-04-17* (spec reorganized: 27 features → 11)
```

- [ ] **Step 3: Remove the old Unresolved Issues section reference if stale**

Verify `issues.md` still exists at `docs/spec/issues.md`. If it does, leave the reference as-is. If it doesn't exist, remove the reference block.

- [ ] **Step 4: Commit**

```bash
git add docs/spec/00-overview.md
git commit -m "refactor(spec): update 00-overview.md to reflect 11-feature reorganization"
```

---

## Task 15: Open PR

- [ ] **Step 1: Push branch**

```bash
git push origin refactor/spec-reorganization
```

- [ ] **Step 2: Open PR**

```bash
gh pr create \
  --title "refactor(spec): consolidate 27 feature dirs into 11 high-level features" \
  --body "$(cat <<'EOF'
## Summary

- Collapses 27 incrementally-grown spec feature directories into 11 high-level features reflecting the actual product shape
- System/foundation features numbered first (01–02), user features next (03–10), CI/CD last (11)
- All story files moved with `git mv` — original story numbers preserved
- Old feature directories deleted; `00-overview.md` updated with new feature table
- Design doc: `docs/superpowers/specs/2026-04-17-spec-reorganization-design.md`

## New structure

| # | Feature |
|---|---------|
| 01 | UI Layout & Components |
| 02 | API Gateway & Reliability |
| 03 | Playback & NowPlaying |
| 04 | Queue & Device Switching |
| 05 | Library Browser & Playlists |
| 06 | Search |
| 07 | Stats & Listening History |
| 08 | Theming & Appearance |
| 09 | Auth, Bootstrap & User Profile |
| 10 | Developer Visibility (Page B) |
| 11 | CI/CD & Release |

## Test plan

- [ ] Verify all story files are present in new dirs: `find docs/spec/features -name "*.md" | wc -l` should match original count
- [ ] Verify no old feature directories remain: `ls docs/spec/features/` shows only 01–11 dirs
- [ ] Verify `00-overview.md` renders correctly (no broken table syntax)
- [ ] Spot-check: open 3 random story files in new locations — content intact

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Self-Review Notes

- **Spec coverage:** All 11 new features have tasks. All 27 old features have explicit `git mv` commands. Story 79 collision (two files with same number, different names, different old dirs) resolved correctly — each goes to a different new dir with no renaming needed.
- **Placeholder scan:** No TBDs. All `git mv` commands use exact filenames from the filesystem listing.
- **Type consistency:** No code — N/A.
- **Open stories table:** Matches the design doc.
- **Old dirs not missed:** 00, 01, 02, `_old-03`, 04, 05, 06, 07, 08, 09, 10, 11, 12, 13, 14, 15, 16, 17, 19, 20, 21, 22, 23, 24, 25, 26, 27 — all 27 removed in Task 13. (No `18-*` dir exists, correctly omitted.)
