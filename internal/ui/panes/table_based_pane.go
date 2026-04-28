// Package panes — TableBasedPane is the embedded base for every table-backed
// pane that supports in-pane text filtering. It owns the Filter and Table
// references and provides the shared filter-routing block (HandleFilterKey)
// that every filterable pane uses identically.
//
// Concrete panes:
//   - embed *TableBasedPane (pointer embedding so the pane can swap table/filter
//     references during SetTheme rebuild)
//   - construct it via NewTableBasedPane(store, theme, focused, table, filter)
//   - call tbp.HandleFilterKey(keyMsg, refreshRows, resizeTable) at the top of
//     their Update; if consumed=true, return cmd
//   - implement pane-specific keys after HandleFilterKey returns false
package panes

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Compile-time checks: TableBasedPane satisfies the two filter interfaces.
var _ layout.FilterablePane = &TableBasedPane{}
var _ layout.FilterQueryPane = &TableBasedPane{}

// TableBasedPane embeds BasePane and adds Table+Filter for table-backed panes.
//
// It satisfies layout.FilterablePane via HasActiveFilter (overrides BasePane's
// false default) and layout.FilterQueryPane via ActiveFilterQuery, so concrete
// panes that embed it inherit both interface implementations and do not need
// to declare them.
type TableBasedPane struct {
	BasePane
	table  *components.Table
	filter *components.Filter
}

// NewTableBasedPane constructs a TableBasedPane with the given dependencies.
// The caller is responsible for constructing the Table and Filter with the
// pane's specific column layout — the base does not impose a column shape.
func NewTableBasedPane(
	store state.StateReader,
	th theme.Theme,
	focused bool,
	table *components.Table,
	filter *components.Filter,
) *TableBasedPane {
	return &TableBasedPane{
		BasePane: BasePane{store: store, theme: th, focused: focused},
		table:    table,
		filter:   filter,
	}
}

// Table returns the embedded table reference. Used by panes for row updates,
// SetFocused/SetSize forwarding, and SetTheme rebuilds.
func (b *TableBasedPane) Table() *components.Table { return b.table }

// Filter returns the embedded filter reference. Used by panes for query
// inspection (filteredXxx helpers) and SetTheme rebuilds.
func (b *TableBasedPane) Filter() *components.Filter { return b.filter }

// SwapTableAndFilter replaces both references in one atomic call. Used by
// SetTheme implementations that rebuild Table and Filter together.
func (b *TableBasedPane) SwapTableAndFilter(t *components.Table, f *components.Filter) {
	b.table = t
	b.filter = f
}

// HasActiveFilter reports whether the filter input is currently capturing keys.
// Overrides BasePane's default-false implementation to satisfy
// layout.FilterablePane.
func (b *TableBasedPane) HasActiveFilter() bool { return b.filter.IsActive() }

// ActiveFilterQuery returns the current filter query for border display.
// Satisfies layout.FilterQueryPane. Returns the LIVE query (updates on every
// keystroke) — this is the intentional UX contract: the user sees their query
// in the border as they type it.
func (b *TableBasedPane) ActiveFilterQuery() string { return b.filter.Query() }

// HandleFilterKey processes the three filter-related key paths shared by every
// filterable table pane:
//
//  1. Filter is active → forward the key to filter.Update; if filter just
//     closed (Enter/Esc consumed it), refocus the table and call resizeTable;
//     always call refreshRows so live filtering takes effect.
//  2. Filter is inactive and the key is 'f' → toggle filter on, table loses
//     focus, call resizeTable.
//  3. Filter is inactive and the key is Esc → if a committed query exists,
//     ClearQuery + refreshRows; otherwise table.GotoTop().
//
// Returns (consumed=true) if the key was handled — the pane should return
// cmd without further processing. Returns (false, nil) if the key should
// fall through to pane-specific handling.
//
// Hooks (called after Filter state is updated, so callers reading
// p.filter.IsActive() / p.filter.Query() inside the hook see the new state):
//   - refreshRows: re-read the store with current query and update rows
//   - resizeTable: adjust table height for filter-bar visibility
//
// A nil hook is silently substituted with a no-op so a programmer error in
// pane wiring degrades to a missing UI refresh rather than a TUI crash.
// Tests in table_based_pane_test.go pin the no-op behaviour so future
// refactors that drop a hook are caught at unit-test time. (Project rule:
// no panic() in production code paths.)
func (b *TableBasedPane) HandleFilterKey(
	msg tea.KeyMsg,
	refreshRows func(),
	resizeTable func(),
) (consumed bool, cmd tea.Cmd) {
	if refreshRows == nil {
		refreshRows = func() {}
	}
	if resizeTable == nil {
		resizeTable = func() {}
	}

	// Path 1 — filter active: forward and refresh.
	if b.filter.IsActive() {
		cmd = b.filter.Update(msg)
		if !b.filter.IsActive() {
			// Filter closed via Enter or Esc — restore table focus + height.
			b.table.SetFocused(true)
			resizeTable()
		}
		// Always refresh: query may have changed (typing) or been committed/cancelled.
		refreshRows()
		return true, cmd
	}

	// Path 2 — 'f' opens the filter. Match exactly one rune to avoid swallowing
	// multi-rune key events (paste-bursts, IME, function keys with rune payloads).
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == 'f' {
		b.filter.Toggle()
		b.table.SetFocused(false)
		resizeTable()
		return true, nil
	}

	// Path 3 — Esc when filter is closed.
	if msg.Type == tea.KeyEscape {
		if b.filter.Query() != "" {
			b.filter.ClearQuery()
			refreshRows()
			return true, nil
		}
		b.table.GotoTop()
		return true, nil
	}

	return false, nil
}

// Actions returns the default action shortcut for filterable table panes:
// [{Key: "f", Label: "filter"}]. The filter-mode label (rendered by border.go
// when FilterQuery != "") takes over the right segment automatically; the
// hint is harmless in the brief active+empty-query window before typing.
//
// Panes that need additional actions (e.g. Albums list-view, Playlists
// reorder hints) override this method and call back into the base via
// BaseFilterAction() to compose their own slice.
func (b *TableBasedPane) Actions() []layout.Action {
	return []layout.Action{{Key: "f", Label: "filter"}}
}

// BaseFilterAction returns the single {f, filter} action so subclasses can
// compose their own Actions() without duplicating the literal.
func (b *TableBasedPane) BaseFilterAction() layout.Action {
	return layout.Action{Key: "f", Label: "filter"}
}
