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
