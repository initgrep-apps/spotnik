package panes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Task 1: replayDisplayState zero value and animationPhase ordering ---

func TestReplayDisplayState_ZeroValue(t *testing.T) {
	var ds replayDisplayState
	assert.Nil(t, ds.requests, "zero-value replayDisplayState should have nil requests map")
	assert.Empty(t, ds.decisions, "zero-value replayDisplayState should have empty decisions")
}

func TestAnimationPhase_ConstantOrdering(t *testing.T) {
	// Phase constants must be in ascending order so phase comparisons work.
	assert.True(t, phaseEntered < phaseAtGateway, "phaseEntered must be < phaseAtGateway")
	assert.True(t, phaseAtGateway < phaseInFlight, "phaseAtGateway must be < phaseInFlight")
	assert.True(t, phaseInFlight < phaseCompleted, "phaseInFlight must be < phaseCompleted")
	assert.True(t, phaseCompleted < phaseDone, "phaseCompleted must be < phaseDone")
}
