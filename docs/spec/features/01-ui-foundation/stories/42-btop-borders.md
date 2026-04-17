---
title: "Custom Border Renderer"
feature: 12-layout
status: done
---

## Background
The current border rendering uses lipgloss.RoundedBorder() with a single accent color. It has no title, no toggle key indicator, and no action shortcuts. The new DESIGN.md specifies btop-style borders where the top border line contains the pane title, toggle key superscript, and action shortcuts. Each pane has a distinct accent color. Focused panes show full accent color; unfocused panes use dimmed/faint borders. Filter mode replaces action shortcuts with the filter query.

Design reference: docs/DESIGN.md sections 5, 10.

## Design

### Top Border Anatomy
`╭─ ` + superscript toggle key (¹²³⁴⁵⁶⁷⁸) in KeyHint() color + title in accent color (bold when focused) + dash fill + actions right-aligned (ᐅ + key in KeyHint() + label in TextMuted(), separated by ` ─── `) + ` ╮`

### Side/Bottom Borders
- Side: `│` + content line padded to Width-2 + `│`
- Bottom: `╰` + `─` x (Width-2) + `╯`

### Filter Mode
`filtering: "query" ─── ᐅEsc close`

### Narrow Handling
First drop actions (show only title), if still too narrow truncate title with `...`, minimum width 10.

### Content Handling
Each content line truncated to Width-2 using lipgloss.Width(). Lines shorter than Width-2 padded with spaces. Fewer lines than Height-2 padded with empty lines.

## Acceptance Criteria
- [ ] `RenderPaneBorder()` produces btop-style borders matching DESIGN.md exactly
- [ ] Top border contains: corner, toggle key superscript, title, dashes, action shortcuts, corner
- [ ] Superscript digits render correctly for keys 1-8
- [ ] Focused borders use full accent color, unfocused use Faint(true)
- [ ] Filter mode replaces actions with filtering text
- [ ] Output dimensions exactly match requested Width x Height
- [ ] Unicode-safe width measurement using lipgloss.Width()
- [ ] No hardcoded hex colors
- [ ] `make ci` passes

## Tasks
- [ ] Implement RenderPaneBorder function in internal/ui/layout/border.go
      - test: basic border with title; toggle key superscript; 2 actions right-aligned; exact width/height; content padded/truncated; filter mode; empty actions; focused/unfocused styling
- [ ] Handle edge cases and content truncation
      - test: very narrow border; minimum width=10; content overflow truncated with ...; Unicode content; ᐅ width measurement
- [ ] Border integration tests
      - test: NowPlaying/Playlists/Queue borders; active filter; Page B borders; exact dimensions; multiple borders side-by-side; all 5 themes
