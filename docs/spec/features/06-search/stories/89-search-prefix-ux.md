---
title: "Search Prefix UX: Animated Placeholder, Ghost Completion, Tag Highlight, Tab Sync"
feature: 19-search-redesign
status: done
---

## Background

The search overlay has command prefixes (`:songs`, `:artists`, `:albums`, `:playlists`) that filter results by category. But nobody discovers them — the placeholder is generic, the completion is custom-built and hidden, and there's no visual feedback when a prefix is valid. Meanwhile, tab switching and prefix input are out of sync.

This story replaces the entire prefix UX with a cohesive, interactive system built on `textinput`'s native `SetSuggestions` feature plus three new behaviors:

1. **Animated cycling placeholder** — when the input is empty, the placeholder cycles through `:songs`, `:artists`, etc. in a theme color (not muted), like a typing demo
2. **Native inline ghost completion** — `textinput.SetSuggestions` shows the rest of a partial prefix as ghost text (`:so|ngs` where `ngs` is muted), Tab accepts
3. **Prompt-based prefix tag** — when a prefix locks, it moves into the textinput's `Prompt` field with a colored background, visually separating it from the query as a "chip/tag"
4. **Styled hint pills row** — the hint row below the input shows all prefixes as styled pills with category badge colors, dimming non-matching ones
5. **Bidirectional tab ↔ prefix sync** — tab switching updates the Prompt tag (not the input Value), keeping everything in sync

## Design

### Part 1: Animated Cycling Placeholder

**Files: `search.go`, `search_prefix.go`**

When the input is empty and unfocused (no text entered yet), the placeholder cycles through themed messages on a 2-second `tea.Tick`:

```
> :songs search for tracks...          ← 2s
> :artists find your favorite artists...  ← 2s
> :albums browse albums...             ← 2s
> :playlists discover playlists...     ← 2s
```

**Implementation:**

Add to `SearchOverlay`:

```go
placeholderIdx int  // cycles 0..3
```

Define the placeholder texts:

```go
var searchPlaceholders = []string{
    ":songs search for tracks...",
    ":artists find your favorite artists...",
    ":albums browse albums...",
    ":playlists discover playlists...",
}
```

On tick (every 2s), increment `placeholderIdx` and update:

```go
type searchPlaceholderTickMsg struct{}

func searchPlaceholderTick() tea.Cmd {
    return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
        return searchPlaceholderTickMsg{}
    })
}
```

In `Update()`:

```go
case searchPlaceholderTickMsg:
    if o.input.Value() == "" {
        o.placeholderIdx = (o.placeholderIdx + 1) % len(searchPlaceholders)
        o.input.Placeholder = searchPlaceholders[o.placeholderIdx]
        return o, searchPlaceholderTick()
    }
    // Stop cycling once user starts typing; restart when cleared
    return o, nil
```

**Placeholder style**: Set `PlaceholderStyle` to `Info()` or `ColumnSecondary()` — a visible theme color, NOT muted. This makes the cycling placeholders look like actionable suggestions rather than passive hints.

```go
ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(t.Info())
```

**Restart on clear**: When the input is cleared (Ctrl+U or backspace to empty), restart the placeholder tick.

### Part 2: Native Inline Ghost Completion via SetSuggestions

**Files: `search.go`, `search_prefix.go`**

Replace the custom `tabCompletePrefix()` and `renderPrefixHints()` inline logic with the textinput's built-in suggestion system.

**Setup in `NewSearchOverlay`:**

```go
ti.ShowSuggestions = true
ti.SetSuggestions([]string{":songs ", ":artists ", ":albums ", ":playlists "})
// Note: trailing space included so accepting completes to locked state
```

Set the suggestion/completion style to `TextMuted()` so ghost text appears dim:

```go
ti.CompletionStyle = lipgloss.NewStyle().Foreground(t.TextMuted())
```

**How it works natively:**
- User types `:so` → textinput renders `:so`<span style="color:gray">`ngs `</span> (ghost text in CompletionStyle)
- User types `:a` → ghost shows `rtists ` (first alphabetical match) or `lbums ` depending on implementation
- User presses Tab → ghost text is accepted, input becomes `:songs ` (with trailing space)
- The textinput's `KeyMap.AcceptSuggestion` handles Tab acceptance

**After Tab acceptance**: The input value is now `:songs ` — our existing `parsePrefix()` fires on the next Update cycle, detects the valid prefix + space, and transitions to `PrefixLocked`. This triggers the Prompt tag (Part 3).

**Remove custom code:**
- Delete `tabCompletePrefix()` from `search_prefix.go` — `SetSuggestions` replaces it
- The `Tab` key routing in `handleKey` no longer needs the `PrefixTyping` check for tab completion — `textinput.Update()` handles it via `AcceptSuggestion`
- Tab key still cycles tabs when NOT in prefix-typing state (this is unchanged)

**Tab key routing update:**

```go
case tea.KeyTab:
    // When the textinput has an active suggestion, let it handle Tab
    // (textinput.Update will accept the suggestion internally).
    // When no suggestion is active, cycle tabs.
    if o.prefixState == PrefixTyping {
        // Forward to textinput — it will accept the suggestion
        var cmd tea.Cmd
        o.input, cmd = o.input.Update(msg)
        o.parsePrefix() // re-check after acceptance
        if o.prefixState == PrefixLocked {
            o.promoteToPromptTag() // Part 3
        }
        return o, cmd
    }
    return o.cycleTabForward()
```

### Part 3: Prompt-Based Prefix Tag

**Files: `search.go`, `search_prefix.go`**

When a prefix locks (either via typing `:songs ` or Tab-accepting a suggestion), the prefix visually moves from the input text into the `Prompt` field as a styled "tag":

```
Before lock:   > :songs pirates of the caribbean
After lock:    :songs > pirates of the caribbean
               ^^^^^^
               Prompt with SelectedBg()/SelectedFg() background — looks like a chip
```

**Visual:**

```
╭─ Search ─────────────────────────────────────────────────╮
│  :songs > pirates of the car_                             │
│  ^^^^^^                                                   │
│  styled tag                                               │
╰──────────────────────────────────────────────────────────╯
```

When no prefix is locked, the prompt is the default `> `:

```
╭─ Search ─────────────────────────────────────────────────╮
│  > pirates of the car_                                    │
╰──────────────────────────────────────────────────────────╯
```

**Implementation — `promoteToPromptTag()`:**

When `parsePrefix()` transitions to `PrefixLocked`:

```go
func (o *SearchOverlay) promoteToPromptTag() {
    // Extract clean query before modifying the input
    query := o.cleanQuery()

    // Style the prefix as a tag in the Prompt
    tagStyle := lipgloss.NewStyle().
        Background(o.theme.SelectedBg()).
        Foreground(o.theme.SelectedFg()).
        Bold(true).
        PaddingLeft(1).
        PaddingRight(1)
    o.input.Prompt = tagStyle.Render(o.lockedPrefix) + " "

    // Value now holds only the clean query
    o.input.SetValue(query)
    o.input.CursorEnd()
}
```

**Implementation — `demoteFromPromptTag()`:**

When the user backspaces at position 0 while a prefix is locked (wants to edit the prefix), move the prefix back into the value:

```go
func (o *SearchOverlay) demoteFromPromptTag() {
    // Reconstruct the full input with prefix
    query := o.input.Value()
    o.input.Prompt = "> "  // reset to default prompt
    o.input.SetValue(o.lockedPrefix + " " + query)
    o.input.CursorEnd()

    // Reset prefix state — user is now editing freely
    o.lockedPrefix = ""
    o.prefixState = PrefixNone
    o.parsePrefix() // re-evaluate (may re-lock if still valid)
}
```

**Backspace at position 0 detection:**

In `handleKey`, before forwarding to textinput:

```go
case tea.KeyBackspace:
    if o.prefixState == PrefixLocked && o.input.Position() == 0 {
        o.demoteFromPromptTag()
        return o, nil
    }
    // Otherwise forward to textinput normally
```

**Impact on `cleanQuery()` and `activeAPITypes()`:**

When the prefix is in the Prompt (locked state), `input.Value()` already IS the clean query:

```go
func (o *SearchOverlay) cleanQuery() string {
    if o.prefixState == PrefixLocked {
        // Value is already clean — prefix is in the Prompt
        return strings.TrimSpace(o.input.Value())
    }
    return o.input.Value()
}
```

`activeAPITypes()` is unchanged — it reads from `lockedPrefix` which is still set.

**Impact on `parsePrefix()`:**

When prefix is already in the Prompt, `parsePrefix()` shouldn't try to re-parse from `Value()` (which no longer has the prefix). Add a guard:

```go
func (o *SearchOverlay) parsePrefix() {
    // If prefix is already promoted to Prompt tag, skip re-parsing.
    // The Prompt holds the prefix; Value holds only the query.
    if o.prefixState == PrefixLocked && o.lockedPrefix != "" {
        return
    }

    value := o.input.Value()
    // ... existing parsing logic ...
}
```

### Part 4: Styled Hint Pills Row

**File: `search_prefix.go` — `renderPrefixHints()`**

Redesign the hint row below the input. Instead of plain text, render each prefix as a styled "pill" with its category badge color. Non-matching pills are dimmed.

**When input is empty** (placeholder cycling):

```
  :songs   :artists   :albums   :playlists
  ^^^^^^   ^^^^^^^^   ^^^^^^^   ^^^^^^^^^^
  all in their badge colors (Success, KeyHint, SeekBar, SectionHeader)
```

**When typing `:so`** (partial match):

```
  :songs   :artists   :albums   :playlists
  ^^^^^^
  highlighted        ← dimmed/muted ──────→
```

**When prefix is locked** (`:songs` active):

```
  (hint row hidden — the Prompt tag makes it redundant)
```

**Implementation:**

```go
func (o *SearchOverlay) renderPrefixHints(width int) string {
    // Hide when prefix is locked — the Prompt tag is visible enough
    if o.prefixState == PrefixLocked {
        return ""
    }

    partial := o.input.Value()
    var pills []string

    for _, prefix := range SearchPrefixes {
        category := prefixToCategory(prefix) // ":songs" → "track"
        color := NewSearchItemDelegate(o.theme).badgeColor(category)

        style := lipgloss.NewStyle()
        if partial == "" || strings.HasPrefix(prefix, partial) {
            // Matching or empty — show in category color
            style = style.Foreground(color).Bold(true)
        } else {
            // Non-matching — dim
            style = style.Foreground(o.theme.TextMuted())
        }
        pills = append(pills, style.Render(prefix))
    }

    return "  " + strings.Join(pills, "   ")
}

func prefixToCategory(prefix string) string {
    switch prefix {
    case ":songs":     return "track"
    case ":artists":   return "artist"
    case ":albums":    return "album"
    case ":playlists": return "playlist"
    default:           return ""
    }
}
```

**Panel 1 height**: The hint row is visible when input is empty OR when in PrefixTyping state. Hidden when PrefixLocked (Prompt tag is enough) or when typing a normal query:

```go
searchBarH := 3
if o.input.Value() == "" || o.prefixState == PrefixTyping {
    searchBarH = 4  // extra line for hint pills
}
```

### Part 5: Bidirectional Tab ↔ Prefix Sync

**Files: `search.go`, `search_prefix.go`**

When the user cycles tabs via Tab/Shift+Tab, the Prompt tag and prefix state update to match.

**`syncInputToTab()`** — works with the Prompt-based system:

```go
var tabToPrefixMap = map[SearchTab]string{
    TabSongs:     ":songs",
    TabArtists:   ":artists",
    TabAlbums:    ":albums",
    TabPlaylists: ":playlists",
}

func (o *SearchOverlay) syncInputToTab() {
    // Get the clean query (prefix-free)
    query := o.cleanQuery()

    if o.activeTab == TabAll {
        // Strip prefix — reset to default prompt
        o.input.Prompt = "> "
        o.input.SetValue(query)
        o.lockedPrefix = ""
        o.prefixState = PrefixNone
    } else if prefix, ok := tabToPrefixMap[o.activeTab]; ok {
        // Set the prefix tag in Prompt
        o.lockedPrefix = prefix
        o.prefixState = PrefixLocked

        tagStyle := lipgloss.NewStyle().
            Background(o.theme.SelectedBg()).
            Foreground(o.theme.SelectedFg()).
            Bold(true).
            PaddingLeft(1).
            PaddingRight(1)
        o.input.Prompt = tagStyle.Render(prefix) + " "
        o.input.SetValue(query)
    }
    o.input.CursorEnd()
}
```

**Wire into tab cycling:**

```go
func (o *SearchOverlay) cycleTabForward() (tea.Model, tea.Cmd) {
    o.activeTab = SearchTab((int(o.activeTab) + 1) % NumTabs)
    o.syncInputToTab()
    o.rebuildListItems()
    query := o.cleanQuery()
    types := TabToAPITypes(o.activeTab)
    return o, func() tea.Msg {
        return SearchTabChangedMsg{Types: types, Query: query}
    }
}

func (o *SearchOverlay) cycleTabBackward() (tea.Model, tea.Cmd) {
    o.activeTab = SearchTab((int(o.activeTab) + NumTabs - 1) % NumTabs)
    o.syncInputToTab()
    o.rebuildListItems()
    query := o.cleanQuery()
    types := TabToAPITypes(o.activeTab)
    return o, func() tea.Msg {
        return SearchTabChangedMsg{Types: types, Query: query}
    }
}
```

**Edge cases:**
- Tab to Songs when input is empty → Prompt = styled `:songs`, Value = `""`
- Tab to All when query was "kk" → Prompt = `> `, Value = `"kk"`
- Tab to Artists when was `:songs kk` → Prompt = styled `:artists`, Value = `"kk"`
- Tab to All when input is empty → Prompt = `> `, Value = `""`, placeholder cycling resumes

### Part 6: SetTheme Propagation

When the theme changes at runtime, update:
- `PlaceholderStyle` with new `Info()` color
- `CompletionStyle` with new `TextMuted()` color
- If a prefix tag is active, re-render the Prompt with new `SelectedBg()`/`SelectedFg()`

```go
func (o *SearchOverlay) SetTheme(th theme.Theme) {
    o.theme = th
    o.input.PlaceholderStyle = lipgloss.NewStyle().Foreground(th.Info())
    o.input.CompletionStyle = lipgloss.NewStyle().Foreground(th.TextMuted())
    if o.prefixState == PrefixLocked {
        o.promoteToPromptTag() // re-renders with new theme colors
    }
    // ... existing delegate/help theme updates ...
}
```

## Acceptance Criteria

- [ ] Empty input shows animated cycling placeholder in `Info()` color, cycling every 2s
- [ ] Placeholder cycles through all 4 prefix messages, wrapping
- [ ] Placeholder cycling stops when user types, restarts when input is cleared
- [ ] `textinput.SetSuggestions` enabled with all 4 prefixes (trailing space included)
- [ ] Typing `:so` shows ghost `ngs ` in `TextMuted()` inline
- [ ] Tab accepts the ghost suggestion, completing the prefix
- [ ] Custom `tabCompletePrefix()` removed — `SetSuggestions` replaces it
- [ ] Locked prefix appears as styled tag in Prompt with `SelectedBg()`/`SelectedFg()` background
- [ ] Input Value contains only the clean query when prefix is locked
- [ ] Backspace at position 0 while locked demotes the tag back to editable input
- [ ] `cleanQuery()` returns correct value in both locked (Prompt) and unlocked (Value) states
- [ ] Hint pills row shows all 4 prefixes in their category badge colors
- [ ] Non-matching pills are dimmed during partial typing
- [ ] Hint row hidden when prefix is locked
- [ ] Tab/Shift+Tab tab switching updates the Prompt tag and prefix state
- [ ] Switching to All tab removes the Prompt tag, restores default `> ` prompt
- [ ] Tab switching preserves the clean query
- [ ] Theme switching updates placeholder style, completion style, and active Prompt tag
- [ ] `make ci` passes

## Tasks

- [ ] Add `placeholderIdx` field and `searchPlaceholders` list to `SearchOverlay`
      - test: `searchPlaceholders` has 4 entries; each starts with a valid prefix command
- [ ] Implement `searchPlaceholderTick` and wire into `Update()`
      - test: tick increments idx and wraps; placeholder text updates; typing stops tick; clearing restarts tick
- [ ] Set `PlaceholderStyle` to `Info()` color in `NewSearchOverlay`
      - test: placeholder style foreground matches theme `Info()`
- [ ] Enable `SetSuggestions` with prefix list in `NewSearchOverlay`
      - test: `ti.ShowSuggestions` is true; suggestions contain all 4 prefixes with trailing space
- [ ] Set `CompletionStyle` to `TextMuted()` color
      - test: completion style foreground matches theme `TextMuted()`
- [ ] Remove custom `tabCompletePrefix()` from `search_prefix.go`
      - test: function no longer exists; build succeeds; Tab in PrefixTyping state forwards to textinput
- [ ] Update Tab key routing to forward to textinput when PrefixTyping
      - test: Tab during PrefixTyping lets textinput accept suggestion; Tab during PrefixNone cycles tabs
- [ ] Implement `promoteToPromptTag()` — move locked prefix to styled Prompt
      - test: after promotion, Prompt contains prefix text with background style; Value contains only clean query; cursor at end
- [ ] Implement `demoteFromPromptTag()` — move Prompt tag back to input Value
      - test: after demotion, Prompt is `> `; Value contains prefix + space + query; prefixState is PrefixNone
- [ ] Wire Backspace at position 0 to call `demoteFromPromptTag()`
      - test: Backspace at pos 0 while locked → demotes; Backspace at pos 3 while locked → normal backspace in query
- [ ] Update `cleanQuery()` for Prompt-based locked state
      - test: locked with Value="kk" → returns "kk"; unlocked with Value=":songs kk" → returns "kk"; unlocked with Value="kk" → returns "kk"
- [ ] Guard `parsePrefix()` to skip re-parsing when prefix is in Prompt
      - test: calling parsePrefix() when already locked + promoted does not change state
- [ ] Redesign `renderPrefixHints()` as styled pills with category colors
      - test: empty input → all 4 pills in category colors; `:so` → `:songs` highlighted, others muted; locked → empty string
- [ ] Update search bar height for pill row visibility
      - test: empty input → height 4; PrefixTyping → height 4; PrefixLocked → height 3; normal query → height 3
- [ ] Implement `syncInputToTab()` using Prompt-based system
      - test: TabSongs + query "kk" → Prompt has styled `:songs`, Value="kk"; TabAll → Prompt="> ", Value="kk"; TabArtists + empty → Prompt has styled `:artists`, Value=""
- [ ] Wire `syncInputToTab()` into `cycleTabForward()` and `cycleTabBackward()`
      - test: cycle All→Songs with "kk" → Prompt changes, Value stays "kk", prefixState=Locked; cycle Songs→All strips tag
- [ ] Update `SetTheme` to propagate PlaceholderStyle, CompletionStyle, and re-render Prompt tag
      - test: SetTheme with new theme → placeholder/completion styles updated; active Prompt tag re-styled
