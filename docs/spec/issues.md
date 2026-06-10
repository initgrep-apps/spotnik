# Spotnik — Issues / Follow-ups

> Placeholder for unresolved items captured during PR reviews and triage.
> Triage into feature stories when ready to fix.

---

## Open

### Wrong song plays from Music page panes (intermittent, unconfirmed)

**Observed:** When pressing Enter on a track in Playlists, Albums, Queue,
RecentlyPlayed, TopTracks, or TopArtists, a different song than the highlighted
one plays. Reproducible multiple times in one session but not currently
reproducible.

**Investigation status:** No code-level root cause found. All panes use
`filteredTracks()[Table().SelectedIndex()]` — index math looks correct. No
obvious race in command closures. Bubble-table `GetHighlightedRowIndex()` returns
global cursor index (not page-local), so pagination is not the cause.

**Hypothesis:** Possible race between a `TickMsg`-triggered `refreshRows()` call
and a keypress arriving at the same event-loop iteration, causing the cursor to
reference a stale pre-refresh index. Needs a reproduction recipe to confirm.

**Next step:** If reproduced again, note which pane, whether a filter was active,
and whether the app had just polled. Triage into a fix story with that context.

*Added 2026-05-20 — triage session.*

---

### PollingTrafficPane Stats row: icon collision + time-range scope

**Found:** 2026-05-20 | **Source:** PR #292 Review
**Feature:** 14-page-b-redesign

Minor items from story 211 PR review:

1. **Icon collision**: Stats row uses `GlyphMusicNote` (spec-prescribed), which is the same glyph as the Playback row. Creates visual ambiguity in the traffic pane. Consider a semantically distinct glyph in a future cleanup pass.

2. **Time-range hardcode**: `StatsFetchedAt("short_term")` is correct for the polling loop (which always polls `short_term`), but if the user loads a different range first, the row shows "never fetched" even though data is present. Consider taking `max(short_term, medium_term, long_term)` timestamps for a more accurate staleness summary.

---

### Search: minor pre-existing latent issues

**Found:** 2026-05-21 | **Source:** PR #297 Review
**Feature:** 06-search

Minor items from stories 212–213 PR review:

1. **`handleEnter` silent no-op on empty URI slice**: When the user presses Enter on a track result and the filtered `uris` slice is empty (e.g. `IsTrack=true` item with blank URI), `buildPlayTrackListCmd` returns `nil` and nothing happens — no toast, no error. Nearly unreachable in practice (existing `si.URI == ""` guard already covers the selected item), but the edge has no user feedback. Add a guard with an info toast if `len(uris) == 0`. Location: `internal/ui/panes/search.go` `handleEnter`.

2. **`syncInputToTab` silent no-op for unknown tab value**: If a future `SearchTab` constant is added to the iota without updating `tabToPrefixMap`, `syncInputToTab` exits silently leaving stale Prompt state. Not reachable today (only five tabs, all in the map), but worth an `// NOTE:` comment at the `else` branch documenting the invariant. Location: `internal/ui/panes/search_prefix.go` `syncInputToTab`.

3. **Vestigial private `buildPromptTag` wrapper**: The `(o *SearchOverlay) buildPromptTag(prefix string)` method only delegates to the exported `BuildPromptTag(o.theme, prefix)`. All call sites could call `BuildPromptTag` directly. Safe to remove in a cleanup pass. Location: `internal/ui/panes/search_prefix.go` ~line 194.

4. **Exported no-op accessors `RenderPrefixHints` / `ShowHintLine`**: Both always return `""` / `false` after story 212. Their test coverage documents the always-false contract, but the exported surface is dead code. Safe to remove in a cleanup pass. Location: `internal/ui/panes/search_prefix.go`.

---

### Album Art: minor review items

**Found:** 2026-05-22 | **Source:** PR #299 Review
**Feature:** 17-album-art

Minor items from story 214 PR review:

1. **BestImage interior pointer escape**: `BestImage` returns `*AlbumImage` from a value receiver, allowing callers to mutate the original `Album` data via the slice backing array. Consider `(AlbumImage, bool)` return pattern in a future refactor.

2. **BestImage dual-dimension fallback gap**: No test case covers the scenario where some images exceed `minSize` in one dimension but not both, and fallback must still select largest width. Add to `TestAlbum_BestImage`.

3. **BestImage boundary test gap**: No test for `Width == minSize && Height == minSize` exact boundary. An off-by-one refactor would not be caught.

4. **BestImage doc precision**: Comment says "smallest image" but tie-breaking is by width only. Height is not considered. Clarify in doc comment.

---

### LayoutManager MinHeight: minor review items

**Found:** 2026-05-22 | **Source:** PR #300 Review
**Feature:** 17-album-art

Minor items from story 215 PR review:

1. **MinHeight + TogglePane interaction untested**: When all cells in a MinHeight row are hidden via `TogglePane`, the row collapses and its MinHeight should be freed for redistribution. No test covers this interaction.

2. **No `reserved == contentH` boundary test**: When MinHeight sum exactly equals content height, remaining is 0 and every row should get exactly its MinHeight. This boundary is untested.

3. **No `totalHWeight == 0` branch test**: If all visible rows have HeightWeight 0, the switch falls through to `h = row.minHeight`. No preset uses this, but the branch is reachable.

4. **Type-design: anemic Row/Cell structs**: No constructors or validation. Negative weights and MinHeight are representable and silently clamped by recompute rather than rejected at construction time. Consider a `Preset.Validate()` method.

---

### Album Art: HTTP timeout and renderer race condition

**Found:** 2026-05-24 | **Source:** PR #305 Review
**Feature:** 17-album-art

Minor items from story 218 PR review (pre-existing from stories 216/217):

1. **`FetchAlbumArtCmd` uses `http.Get` with no timeout**: Deviates from project HTTP client pattern (`api/base.go` uses `Timeout: 30s`). A slow CDN image URL will cause the Bubble Tea command goroutine to hang indefinitely. Fix: use `&http.Client{Timeout: 30 * time.Second}`.

2. **`AlbumArtRenderer.SetResult` race on same-track/different-dimension fetches**: `Init()` dispatches a conservative-size fetch and `WindowSizeMsg` dispatches a resize-size fetch. Both target the same track ID, so the first to complete wins regardless of whether its dimensions match the current pane. Fix: add a monotonic sequence number to `AlbumArtRenderer` and `AlbumArtFetchedMsg`; reject stale results.

---

### HeaderBar: package doc example string stale

**Found:** 2026-05-24 | **Source:** PR #306 Review
**Feature:** 17-album-art

The `internal/uikit/header_bar.go` package-level doc comment still contains the literal example string `"spotnik ─ Music ─ preset N"`. Since the actual output is now `"spotnik ─ Music ─ Dashboard"` (or whatever the active preset name is), updating that example string would keep documentation fully accurate. Trivial docstring fix. Location: `internal/uikit/header_bar.go` package comment.

---

## NowPlaying overlay: spec size table math error
**Found:** 2026-06-10 | **Source:** PR #313 Review
**Feature:** 17-album-art (story 222)

The spec's "Size examples" table claims InfoBox drops at SetSize(60, 16).
At cw=56, vizRows=14, the formula gives infoWidth=cw/4=14, vizWidth=41 > npMinViz=10,
so the InfoBox is NOT dropped. The implementer shifted the test to SetSize(16, 16)
which does trigger the fallback.

Items to log:
1. Update story 222 spec to either:
   a) Replace the (60, 16) row with SetSize(16, 16) — matching the test, OR
   b) Strengthen `npMaxInfoPct` to 3 so the 25% cap drops InfoBox earlier, OR
   c) Document the actual break-point (~width ≤ 35 at height 16)
2. Pick one and update both the spec table and the acceptance criteria.

---

## Story 223 Review Follow-ups (2026-06-10)

### Test coverage gaps (PR #316)

1. **`buildInfoLines` `innerH < 1` branch uncovered** — At `SetSize(10, 2)`, `innerH` clamps to 1 but the truncation to `lines[:1]` is not exercised. Add stress test for height < 3.

2. **Zero/negative dimension stress tests missing** — `SetSize(0, 0)` and `SetSize(-1, -1)` should be verified as no-panic with reasonable output.

3. **`renderSideBySide` `keepViz < 0` branch uncovered** — When `targetH = 0`, the seek-bar-preservation guard is not hit. Add `SetSize(10, 0)` or `SetSize(10, 1)` test.

4. **`TestNowPlayingPane_Adaptive_InfoBoxNoOverlayBackground` too broad** — Scans entire `View()` output for `ESC[48`; will break if visualizer patterns add background colors. Scope assertion to InfoBox interior only (lines between `╭` and `╰`).

5. **Album drop not directly verified in `LibraryPreset` test** — `TestNowPlayingPane_Adaptive_LibraryPreset` asserts controls/volume visible but does not assert `After Hours` is absent from InfoBox interior.

6. **`npInfoMin` capping behavior not directly tested** — At `SetSize(50, 14)`, `cw=50`, `infoWidth=50/3=16 < npInfoMin=28`, so cap to 28. No test measures this.

7. **`VisualizerPattern()` uncovered** — Exported getter has 0% coverage; trivial but should have one-line test per project rules.
