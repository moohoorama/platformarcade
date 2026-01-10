package replay

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrameInput_JSONMarshal(t *testing.T) {
	input := FrameInput{
		F:  10,
		L:  true,
		R:  false,
		J:  true,
		JP: true,
		MX: 100,
		MY: 200,
	}

	data, err := json.Marshal(input)
	require.NoError(t, err)

	var decoded FrameInput
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, input.F, decoded.F)
	assert.Equal(t, input.L, decoded.L)
	assert.Equal(t, input.J, decoded.J)
	assert.Equal(t, input.MX, decoded.MX)
	assert.Equal(t, input.MY, decoded.MY)
}

func TestReplayData_JSONMarshal(t *testing.T) {
	data := ReplayData{
		Version:   "1.0",
		Seed:      12345,
		Stage:     "demo",
		StartTime: "2024-01-01T00:00:00Z",
		Frames: []FrameInput{
			{F: 0, MX: 100, MY: 100},
			{F: 1, R: true, MX: 110, MY: 100},
		},
	}

	jsonData, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded ReplayData
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.Version, decoded.Version)
	assert.Equal(t, data.Seed, decoded.Seed)
	assert.Equal(t, data.Stage, decoded.Stage)
	assert.Equal(t, len(data.Frames), len(decoded.Frames))
}

func TestReplayer_GetInput(t *testing.T) {
	data := ReplayData{
		Version: "1.0",
		Seed:    42,
		Stage:   "test",
		Frames: []FrameInput{
			{F: 0, L: true, MX: 100, MY: 100},
			{F: 1, R: true, J: true, JP: true, MX: 110, MY: 95},
			{F: 2, MX: 120, MY: 90},
		},
	}

	replayer := NewReplayer(data)

	// Frame 0
	input, ok := replayer.GetInput()
	require.True(t, ok)
	assert.True(t, input.Left)
	assert.False(t, input.Right)
	assert.Equal(t, 100, input.MouseX)

	// Frame 1
	input, ok = replayer.GetInput()
	require.True(t, ok)
	assert.False(t, input.Left)
	assert.True(t, input.Right)
	assert.True(t, input.Jump)
	assert.True(t, input.JumpPressed)

	// Frame 2
	input, ok = replayer.GetInput()
	require.True(t, ok)
	assert.False(t, input.Left)
	assert.False(t, input.Right)

	// End of frames
	_, ok = replayer.GetInput()
	assert.False(t, ok)
}

func TestReplayer_CurrentFrame(t *testing.T) {
	data := CreateTestReplayData(5, 100, 100)
	replayer := NewReplayer(data)

	assert.Equal(t, 0, replayer.CurrentFrame())

	replayer.GetInput()
	assert.Equal(t, 1, replayer.CurrentFrame())

	replayer.GetInput()
	replayer.GetInput()
	assert.Equal(t, 3, replayer.CurrentFrame())
}

func TestReplayer_TotalFrames(t *testing.T) {
	data := CreateTestReplayData(10, 100, 100)
	replayer := NewReplayer(data)

	assert.Equal(t, 10, replayer.TotalFrames())
}

func TestReplayer_Seed(t *testing.T) {
	data := ReplayData{
		Seed:   99999,
		Frames: []FrameInput{},
	}
	replayer := NewReplayer(data)

	assert.Equal(t, int64(99999), replayer.Seed())
}

func TestReplayer_Reset(t *testing.T) {
	data := CreateTestReplayData(3, 100, 100)
	replayer := NewReplayer(data)

	// Advance to end
	replayer.GetInput()
	replayer.GetInput()
	replayer.GetInput()
	_, ok := replayer.GetInput()
	assert.False(t, ok)

	// Reset
	replayer.Reset()
	assert.Equal(t, 0, replayer.CurrentFrame())

	// Should be able to read again
	input, ok := replayer.GetInput()
	assert.True(t, ok)
	assert.Equal(t, 100, input.MouseX)
}

func TestCreateTestReplayData(t *testing.T) {
	data := CreateTestReplayData(60, 200, 150)

	assert.Equal(t, "1.0", data.Version)
	assert.Equal(t, int64(12345), data.Seed)
	assert.Equal(t, "test", data.Stage)
	assert.Equal(t, 60, len(data.Frames))

	// Check all frames have correct mouse position
	for i, frame := range data.Frames {
		assert.Equal(t, i, frame.F, "Frame number mismatch at index %d", i)
		assert.Equal(t, 200, frame.MX)
		assert.Equal(t, 150, frame.MY)
	}
}

func TestReplayer_ReturnsCorrectInputState(t *testing.T) {
	// Test that all fields are correctly mapped
	data := ReplayData{
		Frames: []FrameInput{
			{
				F:   0,
				L:   true,
				R:   true,
				U:   true,
				D:   true,
				J:   true,
				JP:  true,
				JR:  true,
				Dsh: true,
				MX:  123,
				MY:  456,
				MC:  true,
				RCP: true,
				RCR: true,
			},
		},
	}

	replayer := NewReplayer(data)
	input, ok := replayer.GetInput()

	require.True(t, ok)
	assert.True(t, input.Left)
	assert.True(t, input.Right)
	assert.True(t, input.Up)
	assert.True(t, input.Down)
	assert.True(t, input.Jump)
	assert.True(t, input.JumpPressed)
	assert.True(t, input.JumpReleased)
	assert.True(t, input.Dash)
	assert.Equal(t, 123, input.MouseX)
	assert.Equal(t, 456, input.MouseY)
	assert.True(t, input.MouseClick)
	assert.True(t, input.RightClickPressed)
	assert.True(t, input.RightClickReleased)
}
