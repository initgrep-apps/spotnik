---
title: "Header: show preset name instead of raw index"
feature: 17-album-art
status: open
---

## Background

The Music page header currently renders `Music | 0` — a raw preset index.
`Manager.ActivePresetName()` already exists in `internal/ui/layout/layout.go`
and returns the correct string. The bug is that `renderHeader()` in
`internal/app/render.go` calls `ActivePresetIndex()` instead and passes an
`int` to `HeaderBar.Preset`.

The fix is two-part: rename `HeaderBar.Preset int` → `PresetName string`, and
thread `ActivePresetName()` through the render pipeline. Stats page keeps its
existing behaviour (no preset name in header) by passing an empty string.

## Design

### `internal/uikit/header_bar.go` — rename field, drop fmt import

```go
// Before
type HeaderBar struct {
    Width      int
    AppName    string
    Page       string
    Preset     int      // -1 hides the segment
    RightChips []string
    Theme      theme.Theme
}

// render:
if h.Preset >= 0 {
    left += sep + muted.Render(fmt.Sprintf("preset %d", h.Preset))
}

// After
type HeaderBar struct {
    Width      int
    AppName    string
    Page       string
    PresetName string   // empty string hides the preset segment
    RightChips []string
    Theme      theme.Theme
}

// render:
if h.PresetName != "" {
    left += sep + muted.Render(h.PresetName)
}
```

Remove `"fmt"` from the import block (no longer used).

### `internal/app/render.go` — wire ActivePresetName()

```go
// Before
preset := a.layout.ActivePresetIndex()
if a.layout.ActivePage() == layout.PageStats {
    preset = -1
}
return uikit.HeaderBar{
    ...
    Preset: preset,
    ...
}.Render()

// After
presetName := a.layout.ActivePresetName()
if a.layout.ActivePage() == layout.PageStats {
    presetName = ""
}
return uikit.HeaderBar{
    ...
    PresetName: presetName,
    ...
}.Render()
```

`ActivePresetName()` is already implemented at `internal/ui/layout/layout.go:226`.
No layout package changes needed.

## Acceptance Criteria

- [ ] `HeaderBar` struct has `PresetName string` (not `Preset int`)
- [ ] `HeaderBar.Render()` renders the preset name segment when `PresetName != ""`
- [ ] `HeaderBar.Render()` hides the preset segment when `PresetName == ""`
- [ ] Music page header shows `Music | Dashboard` when on preset 0
- [ ] Music page header shows `Music | Listening` when on preset 1
- [ ] Stats page header shows no preset segment
- [ ] `"fmt"` import removed from `header_bar.go` (if no other usage remains)
- [ ] `make ci` passes

## Tasks

- [ ] Update `internal/uikit/header_bar_test.go` to use `PresetName: "Name"` instead
      of `Preset: N` and update string assertions (tests must fail first):
      - `Preset: 0` → `PresetName: "Dashboard"`
      - `Preset: 1` → `PresetName: "Listening"`
      - `Preset: 3` → `PresetName: "Discovery"`
      - `Preset: -1` → `PresetName: ""`
      - `assert.Contains(t, plain, "preset 0"` → `assert.Contains(t, plain, "Dashboard"`

- [ ] Update `internal/app/render_test.go`: rename `TestRenderHeader_ContainsPresetIndex`
      → `TestRenderHeader_ContainsPresetName`; assert output contains `"Dashboard"` and
      does not contain `"preset 0"`

- [ ] Run tests to confirm they fail:
      `rtk go test ./internal/uikit/... ./internal/app/... -run "TestHeaderBar|TestRenderHeader" -v`

- [ ] Update `internal/uikit/header_bar.go`:
      - rename `Preset int` → `PresetName string`
      - replace `fmt.Sprintf("preset %d", h.Preset)` render block with `h.PresetName` check
      - remove `"fmt"` import if unused

- [ ] Update `internal/app/render.go` `renderHeader()`:
      - replace `ActivePresetIndex()` call with `ActivePresetName()`
      - replace `preset = -1` branch with `presetName = ""`
      - update `HeaderBar{}` literal to use `PresetName: presetName`

- [ ] Run full test suite to catch any other callers of the renamed field:
      `rtk go test ./... 2>&1 | grep -E "FAIL|ok"`
      Fix any compilation errors from remaining `Preset:` references.

- [ ] `make ci` passes
