---
title: "Search Input: Command Prefix Autocomplete"
feature: 19-search-redesign
status: open
---

## Background

The search bar supports command prefixes (`:songs`, `:artists`, `:albums`, `:playlists`) that filter results to a single category. This story implements the prefix parsing, inline autocomplete hints, and Tab-completion.

## Bubble Tea Components

This story extends the `textinput.Model` (already in use) with custom prefix parsing and inline hint rendering. No new Bubble Tea components are introduced — the autocomplete hints are rendered as styled lipgloss text below the input.

| Component | Import | Role in This Story |
|---|---|---|
| **textinput** | `github.com/charmbracelet/bubbles/textinput` | Search bar input — `SetValue()` for Tab completion, `Value()` for prefix parsing, `CursorEnd()` after completion |

**Reference**: See `/bubbletea` skill `references/components.md` for textinput API. Key methods used:
- `textinput.SetValue(s)` — programmatically set input (used by Tab completion to insert completed prefix)
- `textinput.CursorEnd()` — move cursor to end after completion
- `textinput.Value()` — read current input for prefix parsing on every keystroke
- `textinput.Update(msg)` — forward key events for normal typing/backspace handling

## Design

### Prefix Definitions

```go
var searchPrefixes = []string{":songs", ":artists", ":albums", ":playlists"}

var prefixToTab = map[string]searchTab{
    ":songs":     tabSongs,
    ":artists":   tabArtists,
    ":albums":    tabAlbums,
    ":playlists": tabPlaylists,
}
```

### Prefix State Machine

The input text is parsed on every keystroke:

1. **No prefix**: Input doesn't start with `:` → normal search, activeTab unchanged
2. **Partial prefix**: Input starts with `:` but no space yet → show matching hints, don't search yet
3. **Complete prefix**: Input matches a known prefix followed by a space → lock prefix, set activeTab, strip prefix from query for API

```go
type prefixState int

const (
    prefixNone     prefixState = iota // no colon prefix
    prefixTyping                       // typing a prefix (e.g., ":so")
    prefixLocked                       // prefix complete, typing query (e.g., ":songs the dealer")
)
```

Add to `SearchOverlay`:

```go
lockedPrefix string      // the confirmed prefix (e.g., ":songs")
prefixState  prefixState
```

### Parsing Logic

On every input change (keystroke, backspace):

```go
func (o *SearchOverlay) parsePrefix() {
    value := o.input.Value()

    if !strings.HasPrefix(value, ":") {
        o.prefixState = prefixNone
        o.lockedPrefix = ""
        return
    }

    // Find the first space
    spaceIdx := strings.Index(value, " ")
    if spaceIdx == -1 {
        // Still typing the prefix
        o.prefixState = prefixTyping
        o.lockedPrefix = ""
        return
    }

    // Check if the part before space is a valid prefix
    candidate := value[:spaceIdx]
    if _, ok := prefixToTab[candidate]; ok {
        o.prefixState = prefixLocked
        o.lockedPrefix = candidate
    } else {
        // Invalid prefix — treat as normal search
        o.prefixState = prefixNone
        o.lockedPrefix = ""
    }
}
```

### Extracting the Clean Query

```go
func (o *SearchOverlay) cleanQuery() string {
    if o.prefixState == prefixLocked {
        value := o.input.Value()
        return strings.TrimSpace(value[len(o.lockedPrefix):])
    }
    return o.input.Value()
}

func (o *SearchOverlay) activeAPITypes() []string {
    if o.prefixState == prefixLocked {
        if tab, ok := prefixToTab[o.lockedPrefix]; ok {
            return tabToAPITypes[tab]
        }
    }
    return tabToAPITypes[o.activeTab]
}
```

### When Prefix Locks → Sync Tab

When a prefix becomes locked (user types `:songs ` with trailing space):
- Set `o.activeTab` to the matching tab
- The debounce fires with `cleanQuery()` and `activeAPITypes()`

### Inline Autocomplete Hints

When `prefixState == prefixTyping`, render a hint line below the input showing matching prefixes:

```go
func (o *SearchOverlay) renderPrefixHints(width int) string {
    if o.prefixState != prefixTyping {
        return ""
    }

    partial := o.input.Value() // e.g., ":so"
    var matches []string
    for _, p := range searchPrefixes {
        if strings.HasPrefix(p, partial) {
            matches = append(matches, p)
        }
    }

    if len(matches) == 0 {
        return ""
    }

    hintStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())
    return hintStyle.Render("  " + strings.Join(matches, "  "))
}
```

This line appears between the input and the separator in `View()`. When not in `prefixTyping` state, it's omitted (no extra line consumed).

### Tab Completion

When `Tab` is pressed and `prefixState == prefixTyping`:

```go
func (o *SearchOverlay) tabCompletePrefix() (tea.Model, tea.Cmd) {
    partial := o.input.Value()
    var matches []string
    for _, p := range searchPrefixes {
        if strings.HasPrefix(p, partial) {
            matches = append(matches, p)
        }
    }

    if len(matches) == 1 {
        // Unique match — complete with trailing space
        o.input.SetValue(matches[0] + " ")
        o.input.CursorEnd()
        o.parsePrefix()
        return o, nil
    }
    // Multiple matches or no match — don't complete, let user keep typing
    return o, nil
}
```

### Tab Key Routing Update

The `Tab` key handler now checks prefix state first:

```go
case tea.KeyTab:
    if o.prefixState == prefixTyping {
        return o.tabCompletePrefix()
    }
    return o.moveSectionForward() // cycles tabs
```

### Debounce Integration

The debounce logic uses `cleanQuery()` instead of raw input:

```go
// In handleKey default (regular typing):
o.parsePrefix()
if o.prefixState == prefixTyping {
    // Don't debounce yet — user is still typing the prefix
    return o, cmd
}
q := o.cleanQuery()
debounceCmd := debounceSearch(q)
return o, tea.Batch(cmd, debounceCmd)
```

The `searchDebounceMsg` and `SearchRequestMsg` carry the clean query. The app handler uses `activeAPITypes()` to determine which types to search.

Update `SearchRequestMsg`:

```go
type SearchRequestMsg struct {
    Query string
    Types []string // API type values
}
```

## Acceptance Criteria

- [ ] Typing `:` shows inline hints of matching prefixes below input
- [ ] Hints filter as user types (`:s` → `:songs`, `:a` → `:artists :albums`)
- [ ] `Tab` completes unique prefix match with trailing space
- [ ] `Tab` with multiple matches does nothing (user keeps typing)
- [ ] Completed prefix + space locks the prefix and syncs the active tab
- [ ] Locked prefix is stripped from the API query (`:songs kk` → `q=kk&type=track`)
- [ ] Backspacing into the prefix unlocks it and returns to hint state
- [ ] No search fires while user is still typing the prefix (no trailing space yet)
- [ ] Debounce uses clean query, not raw input with prefix
- [ ] `SearchRequestMsg` carries both Query and Types
- [ ] make ci passes

## Tasks

- [ ] Define `searchPrefixes`, `prefixToTab` mapping, and `prefixState` enum
      - test: all 4 prefixes map to correct tabs; prefixToTab covers all entries
- [ ] Add prefix state fields to `SearchOverlay` and implement `parsePrefix()`
      - test: no colon → prefixNone; `:so` → prefixTyping; `:songs ` → prefixLocked; `:invalid ` → prefixNone
- [ ] Implement `cleanQuery()` and `activeAPITypes()`
      - test: locked `:songs kk` → cleanQuery="kk", types=["track"]; no prefix "kk" → cleanQuery="kk", types from activeTab
- [ ] Implement `renderPrefixHints()` for inline autocomplete display
      - test: `:s` shows `:songs`; `:a` shows `:artists :albums`; `:songs` shows `:songs`; `hello` shows nothing
- [ ] Implement `tabCompletePrefix()` for Tab completion
      - test: `:s` + Tab → `:songs ` (unique match); `:a` + Tab → no change (2 matches); `:artists` + Tab → `:artists `
- [ ] Update Tab key routing to check prefix state first
      - test: Tab during prefixTyping completes prefix; Tab during prefixNone cycles tabs
- [ ] Update debounce to use cleanQuery and skip during prefixTyping
      - test: typing `:so` does not fire debounce; typing `:songs kk` fires debounce with query="kk"
- [ ] Update `SearchRequestMsg` to carry Types and wire in app.go
      - test: app handler uses msg.Types for buildSearchBatchCmd; default types used when empty
