package app_test

// scroll_flow_test.go — Story 264: Cross-cutting Esc scroll-to-top integration test.
//
// Verifies that pressing Esc on a table pane with no active filter and no overlay
// open resets the table scroll position back to page 1 (top).

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEscScrollToTop_ResetsScrollOnTablePanes verifies that Esc resets the scroll
// position of a focused table pane (Queue) to the top when no filter is active and
// no overlay is open. The full flow is driven through the root App so the key is
// routed via the standard focused-pane path.
func TestEscScrollToTop_ResetsScrollOnTablePanes(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)
	a.Update(app.SplashDismissMsgForTest{})

	// Seed the queue with enough items to scroll past page 1. Deliver via
	// QueueLoadedMsg so the handler writes the store AND calls RefreshRows on the
	// QueuePane (direct store seeding does not populate the table rows).
	items := make([]domain.QueueItem, 20)
	for i := range items {
		items[i] = domain.QueueItem{
			Type:  domain.QueueItemTypeTrack,
			Track: &domain.Track{ID: "q" + itoa(i), Name: "Track " + itoa(i+1), URI: "spotify:track:q" + itoa(i)},
		}
	}
	a.Update(panes.QueueLoadedMsg{Items: items})

	// Tab to the Queue pane.
	for !a.QueueFocused() {
		mm, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
		a = mm.(*app.App)
	}
	require.True(t, a.QueueFocused(), "Queue should be focused")
	qp := a.QueuePane()
	require.NotNil(t, qp)

	// Scroll down with 'j' until we advance past page 1. Re-fetch the pane each
	// iteration because Update may return a new QueuePane instance.
	for {
		qp = a.QueuePane()
		if qp.Table().CurrentPage() > 1 {
			break
		}
		mm, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		a = mm.(*app.App)
	}
	require.Greater(t, a.QueuePane().Table().CurrentPage(), 1, "should have scrolled past page 1")

	// Press Esc — with no filter active and no overlay open, the table resets to page 1.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	a = m.(*app.App)
	assert.Equal(t, 1, a.QueuePane().Table().CurrentPage(),
		"Esc with no filter active should reset the Queue table to page 1")
}

// itoa converts an int to its decimal string representation. A tiny self-contained
// helper avoids importing strconv solely for one call site.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
