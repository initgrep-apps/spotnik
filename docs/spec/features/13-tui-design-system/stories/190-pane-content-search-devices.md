---
title: "Pane content cleanup: devices content / search spinner / search_delegate"
feature: 13-tui-design-system
status: open
---

## Background

Stories 185–186 fixed the **chrome** of the devices and search overlays. The audit
(§3.2) identified additional inline-glyph leaks in the **content** of those overlays
that bypass the catalogue. This story handles three related fixes inside the
search/devices domain:

1. **`internal/ui/panes/devices.go:209,212,219,222`** — raw `◉` / `○` for
   active/available device indicators; `:258–264` — custom device-type icons
   `⊡⊞⊟⊠`; `:145–147` — custom "No devices found" empty message.
2. **`internal/ui/panes/search.go:214`** — uses `bubbles/spinner.Model` for first-page
   loading. The bubbles spinner has no ASCII fallback path. Replace with
   `uikit.Spinner` so the loading indicator honours `ActiveMode()`.
3. **`internal/ui/panes/search_delegate.go:62–77`** — `categorySymbol` hardcodes
   `♪`, `★`, `◎`, `▤`, `·` (with `▤` previously having no `GlyphRole`). Plus
   `:95, 335` raw `│` / `·` separators.

All three changes share the search/devices content domain and migrate to existing
uikit primitives or to the catalogue rows added in story 183 (`GlyphPlaylist`,
`GlyphSeparator`, `GlyphDeviceComputer/Phone/Speaker/TV`).

**Depends on:** story 183 (catalogue audit — `GlyphPlaylist`, `GlyphSeparator`,
device-type icons).

**Plan tasks:** 5.1, 5.2, 5.3 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

**Files:** `internal/ui/panes/devices.go`, `internal/ui/panes/devices_test.go`,
`internal/ui/panes/search.go`, `internal/ui/panes/search_delegate.go`,
`internal/ui/panes/search_delegate_test.go`.

## Design

### `devices.go`

Replace the four `◉` / `○` literals at `:209,212,219,222` with
`uikit.GlyphFor(uikit.GlyphActive, m)` / `uikit.GlyphFor(uikit.GlyphAvailable, m)`.
(Implementer chooses inline `GlyphFor` calls vs. `uikit.StatusGlyph` helper based on
which keeps the call site shortest; either is acceptable.)

Replace the device-type icon function at `:258–264`:

```go
func deviceTypeIcon(deviceType string) string {
    m := uikit.ActiveMode()
    switch deviceType {
    case "Computer":
        return uikit.GlyphFor(uikit.GlyphDeviceComputer, m)
    case "Smartphone":
        return uikit.GlyphFor(uikit.GlyphDevicePhone, m)
    case "Speaker":
        return uikit.GlyphFor(uikit.GlyphDeviceSpeaker, m)
    case "TV":
        return uikit.GlyphFor(uikit.GlyphDeviceTV, m)
    default:
        return uikit.GlyphFor(uikit.GlyphInactive, m)
    }
}
```

Replace the custom "No devices found" rendering at `:145–147`:

```go
if len(o.devices) == 0 {
    return uikit.EmptyState{
        Text:   "No devices found",
        Hint:   "Open Spotify on a device to see it here",
        Width:  o.width,
        Height: o.height - 4,
        Theme:  o.theme,
    }.Render()
}
```

### `search.go` — bubbles/spinner → uikit.Spinner

Remove the `"github.com/charmbracelet/bubbles/spinner"` import. Replace the field:

```go
// before
sp spinner.Model
// after
sp *uikit.Spinner
```

Constructor:

```go
s.sp = uikit.NewSpinner("Loading...", theme)
```

Update the `Update` and `View` paths to call `s.sp.Update(...)` / `s.sp.View()`.
`uikit.Spinner` mirrors the bubbles API where possible; the diff is mechanical.

### `search_delegate.go`

`categorySymbol` (`:62–77`):

```go
func categorySymbol(category string) string {
    m := uikit.ActiveMode()
    switch category {
    case "track":
        return uikit.GlyphFor(uikit.GlyphMusicNote, m)
    case "artist":
        return uikit.GlyphFor(uikit.GlyphPinned, m)
    case "album":
        return uikit.GlyphFor(uikit.GlyphInactive, m)
    case "playlist":
        return uikit.GlyphFor(uikit.GlyphPlaylist, m)
    default:
        return uikit.GlyphFor(uikit.GlyphSeparator, m)
    }
}
```

Line 95 (left selection border): `uikit.GlyphFor(uikit.GlyphVRule, uikit.ActiveMode())`.
Line 335 (row separator): `uikit.GlyphFor(uikit.GlyphSeparator, uikit.ActiveMode())`.

## Acceptance Criteria

- [ ] `panes/devices.go` no longer contains literal `◉`, `○`, `⊡`, `⊞`, `⊟`, `⊠`;
      every device indicator and device-type icon resolves via `uikit.GlyphFor`
- [ ] The "No devices found" state in `devices.go:145–147` renders through
      `uikit.EmptyState` (no custom string concatenation)
- [ ] `panes/search.go:214` uses `*uikit.Spinner`; the
      `"github.com/charmbracelet/bubbles/spinner"` import is removed from the file
- [ ] `panes/search_delegate.go:62–77` `categorySymbol` returns values exclusively
      via `uikit.GlyphFor`
- [ ] `panes/search_delegate.go:95` and `:335` resolve their separators via
      `GlyphVRule` and `GlyphSeparator`
- [ ] New test `TestDevicesOverlay_AsciiContent` confirms ASCII output of the
      devices overlay contains `(*)`, `[p]` (or other matching ASCII forms) and
      contains none of `◉`, `○`, `⊡`, `⊞`, `⊟`, `⊠`
- [ ] New test `TestSearchDelegate_AsciiCategorySymbols` covers each category
      (track / artist / album / playlist) in ASCII mode
- [ ] All existing search / devices tests pass
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Tasks 5.1, 5.2, 5.3 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

- [ ] Branch: `fix/13-pane-content-search-devices`
- [ ] Replace `◉`/`○` literals at `devices.go:209,212,219,222` via `uikit.GlyphFor`
- [ ] Rewrite `deviceTypeIcon` at `devices.go:258–264` to switch on type and return
      `uikit.GlyphFor(GlyphDeviceComputer/Phone/Speaker/TV, m)`
- [ ] Migrate the empty message at `devices.go:145–147` to `uikit.EmptyState`
- [ ] Add `TestDevicesOverlay_AsciiContent`
- [ ] Run `go test ./internal/ui/panes/ -run TestDevicesOverlay -v` → PASS
- [ ] Commit: `fix(devices): route status glyphs, device icons, and empty state through uikit`
- [ ] Replace `bubbles/spinner.Model` with `*uikit.Spinner` in `panes/search.go:214`;
      drop the bubbles spinner import; update `Update` / `View` paths
- [ ] Run `go test ./internal/ui/panes/ -run TestSearch -v` → PASS
- [ ] Commit: `fix(search): swap bubbles/spinner for uikit.Spinner with ascii fallback`
- [ ] Rewrite `categorySymbol` in `search_delegate.go:62–77`; replace separators at
      `:95, 335`
- [ ] Add `TestSearchDelegate_AsciiCategorySymbols` covering all four categories
- [ ] Run `go test ./internal/ui/panes/ -run TestSearchDelegate -v` → PASS
- [ ] Commit: `fix(search-delegate): route categorySymbol and separators through GlyphFor`
- [ ] `make ci` → PASS
- [ ] Push branch + open PR
