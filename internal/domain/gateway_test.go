package domain_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestGatewayDecision_Values(t *testing.T) {
	tests := []struct {
		name string
		val  domain.GatewayDecision
		want int
	}{
		{"DecisionAllowed is zero value", domain.DecisionAllowed, 0},
		{"DecisionWaited is 1", domain.DecisionWaited, 1},
		{"DecisionDeduped is 2", domain.DecisionDeduped, 2},
		{"DecisionBlocked is 3", domain.DecisionBlocked, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, domain.GatewayDecision(tt.want), tt.val)
		})
	}
}

func TestGatewayDecision_Distinct(t *testing.T) {
	decisions := []domain.GatewayDecision{
		domain.DecisionAllowed,
		domain.DecisionWaited,
		domain.DecisionDeduped,
		domain.DecisionBlocked,
	}
	seen := make(map[domain.GatewayDecision]bool)
	for _, d := range decisions {
		assert.False(t, seen[d], "GatewayDecision values must be distinct")
		seen[d] = true
	}
}
