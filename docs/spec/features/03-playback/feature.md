---
title: "Playback & NowPlaying"
status: done
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
- [ ] Open: stories 11 (playback UX), 36 (command safety errors), 197 (volume debounce + queue icon cleanup)
