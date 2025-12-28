package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGameState_String(t *testing.T) {
	tests := []struct {
		state    GameState
		expected string
	}{
		{StateMenu, "Menu"},
		{StateLoading, "Loading"},
		{StatePlaying, "Playing"},
		{StatePaused, "Paused"},
		{StateGameOver, "GameOver"},
		{StateStageClear, "StageClear"},
		{GameState(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestGameStateConstants(t *testing.T) {
	// Verify the iota ordering
	assert.Equal(t, GameState(0), StateMenu)
	assert.Equal(t, GameState(1), StateLoading)
	assert.Equal(t, GameState(2), StatePlaying)
	assert.Equal(t, GameState(3), StatePaused)
	assert.Equal(t, GameState(4), StateGameOver)
	assert.Equal(t, GameState(5), StateStageClear)
}
