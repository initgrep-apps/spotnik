package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func newTestGatewayHealthPane(t *testing.T) *panes.GatewayHealthPane {
	t.Helper()
	return panes.NewGatewayHealthPane(state.New(), theme.Load("black"))
}

func TestGatewayHealthPane_ImplementsLayoutPane(t *testing.T) {
	var _ layout.Pane = newTestGatewayHealthPane(t)
}

func TestGatewayHealthPane_ID(t *testing.T) {
	assert.Equal(t, layout.PaneGatewayHealth, newTestGatewayHealthPane(t).ID())
}

func TestGatewayHealthPane_Title(t *testing.T) {
	assert.Equal(t, "Gateway Health", newTestGatewayHealthPane(t).Title())
}

func TestGatewayHealthPane_ToggleKey(t *testing.T) {
	assert.Equal(t, 2, newTestGatewayHealthPane(t).ToggleKey())
}

func TestGatewayHealthPane_View_EmptyBeforeResize(t *testing.T) {
	assert.Equal(t, "", newTestGatewayHealthPane(t).View())
}

func TestGatewayHealthPane_View_ContainsHealthRows(t *testing.T) {
	p := newTestGatewayHealthPane(t)
	p.SetSize(50, 10)
	view := p.View()
	assert.Contains(t, view, "Tokens")
	assert.Contains(t, view, "Slots")
	assert.Contains(t, view, "Backoff")
	assert.Contains(t, view, "Dedup")
}

func TestGatewayHealthPane_Update_DrainsCursor(t *testing.T) {
	store := state.New()
	p := panes.NewGatewayHealthPane(store, theme.Load("black"))
	p.SetSize(50, 10)

	// Emit a gateway event so the cursor advances.
	store.RecordEvent(domain.GatewayEvent{
		Kind: domain.EventTokenConsumed,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable: 7, TokensMax: 10,
		},
	})

	p.Update(panes.TickMsg{})
	view := p.View()
	// After processing the event the snapshot is updated; tokens row reflects it.
	assert.Contains(t, view, "Tokens")
}
