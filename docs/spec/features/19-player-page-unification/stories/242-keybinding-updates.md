---
title: "Keybinding updates (3 locations)"
feature: 19-player-page-unification
status: open
---

## Background

The keybinding system needs updates for the 2-page model and the new `i` key.
Per AGENTS.md rule #15, keybindings must be updated in all 3 locations in the
same commit.

## Design

### Changes

1. **`0` key**: Cycles Player → Stats → Player (was Music → Podcasts → Stats → Music)
2. **`p` key**: Cycles 6 presets on Player page (was 4 on Music, 2 on Podcasts)
3. **`i` key**: Opens Episode Details overlay when episode is playing (new)
4. **Contextual toggle keys**: Music presets 1-8, podcast presets 1-4

### 3 locations (must be in same commit)

1. **`README.md`** Keybindings section
   - Update `0` description: "Cycle Player / Stats"
   - Update `p` description: "Cycle preset on Player page"
   - Add `i`: "Show episode details (when episode playing)"
   - Remove Podcasts page toggle keys section
   - Update toggle key descriptions for contextual behavior

2. **`docs/system/design.md`** §17 Keybinding table
   - Update `0` row: "Cycle Player / Stats" (2-page model)
   - Add `i` row: "Show episode details overlay | When episode is playing"
   - Remove Podcasts page toggle keys
   - Update toggle key table for contextual pane switching

3. **`internal/ui/panes/help_overlay.go`** `helpContent`
   - Add `i` entry to Playback section: `{Key: "i", Label: "episode details"}`
   - Conditionally show only when `CurrentlyPlayingType == "episode"`
   - Update `0` description
   - Remove Podcasts page entries

## Files

### Modify

- `README.md`
- `docs/system/design.md`
- `internal/ui/panes/help_overlay.go`

## Acceptance Criteria

- [ ] All 3 locations updated in the same commit
- [ ] `0` key described as "Cycle Player / Stats" in all 3 locations
- [ ] `i` key present in all 3 locations with description "episode details"
- [ ] Podcasts page toggle keys removed from all 3 locations
- [ ] Contextual toggle key behavior documented
- [ ] `make ci` passes