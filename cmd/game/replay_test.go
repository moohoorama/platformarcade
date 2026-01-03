package main

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/younwookim/mg/internal/application/replay"
	"github.com/younwookim/mg/internal/application/scene/playing"
	"github.com/younwookim/mg/internal/application/system"
	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

// createTestConfig creates a minimal config for testing
func createTestConfig() *config.PhysicsConfig {
	return &config.PhysicsConfig{
		Physics: config.PhysicsSettings{
			Gravity:      800,
			MaxFallSpeed: 400,
		},
		Movement: config.MovementConfig{
			Acceleration: 2000,
			Deceleration: 2500,
			MaxSpeed:     120,
			AirControl:   0.8,
		},
		Jump: config.JumpConfig{
			Force:                  280,
			VariableJumpMultiplier: 0.4,
			CoyoteTime:             0.1,
			JumpBuffer:             0.1,
			ApexModifier: config.ApexModifierConfig{
				Enabled:           true,
				Threshold:         20,
				GravityMultiplier: 0.5,
			},
			FallMultiplier: 1.6,
		},
		Dash: config.DashConfig{
			Speed:    300,
			Duration: 0.15,
			Cooldown: 0.5,
		},
	}
}

// createTestStageWithGround creates a test stage with ground at y=4
func createTestStageWithGround() *entity.Stage {
	stage := &entity.Stage{
		Width:    10,
		Height:   5,
		TileSize: 16,
		SpawnX:   80,
		SpawnY:   46,
		Tiles:    make([][]entity.Tile, 5),
	}
	for y := 0; y < 5; y++ {
		stage.Tiles[y] = make([]entity.Tile, 10)
		for x := 0; x < 10; x++ {
			if y == 4 {
				stage.Tiles[y][x] = entity.Tile{Solid: true}
			}
		}
	}
	return stage
}

// createTestPlayer creates a player on ground at spawn position
func createTestPlayer(stage *entity.Stage) *entity.Player {
	hitbox := entity.TrapezoidHitbox{
		Head: entity.HitboxRect{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: entity.HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: entity.HitboxRect{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	player := entity.NewPlayer(stage.SpawnX, stage.SpawnY, hitbox, 100)
	player.OnGround = true
	return player
}

// SimulationResult contains the results of a replay simulation
type SimulationResult struct {
	VYValues      []float64
	VXValues      []float64
	Positions     []struct{ X, Y int }
	FinalFrame    int
	VYMin         float64
	VYMax         float64
	VYFluctuation bool
}

// simulateWithReplay runs a game simulation using replayed inputs
func simulateWithReplay(replayer *replay.Replayer, cfg *config.PhysicsConfig, stage *entity.Stage, player *entity.Player) SimulationResult {
	inputSystem := system.NewInputSystem(cfg)
	physicsSystem := system.NewPhysicsSystem(cfg, stage)
	dt := 1.0 / 60.0

	result := SimulationResult{
		VYValues:  make([]float64, 0, replayer.TotalFrames()),
		VXValues:  make([]float64, 0, replayer.TotalFrames()),
		Positions: make([]struct{ X, Y int }, 0, replayer.TotalFrames()),
	}

	for {
		input, ok := replayer.GetInput()
		if !ok {
			break
		}

		// Update game state
		inputSystem.UpdatePlayer(player, input, dt)
		physicsSystem.Update(player, dt, 10) // Normal speed: 10 sub-steps

		// Record state
		result.VYValues = append(result.VYValues, player.VY)
		result.VXValues = append(result.VXValues, player.VX)
		result.Positions = append(result.Positions, struct{ X, Y int }{player.PixelX(), player.PixelY()})
		result.FinalFrame = replayer.CurrentFrame()
	}

	// Calculate stats
	if len(result.VYValues) > 0 {
		result.VYMin = result.VYValues[0]
		result.VYMax = result.VYValues[0]
		for _, vy := range result.VYValues {
			if vy < result.VYMin {
				result.VYMin = vy
			}
			if vy > result.VYMax {
				result.VYMax = vy
			}
		}
		// Check for fluctuation (different values after stabilization)
		// Skip first 10 frames for settling
		if len(result.VYValues) > 10 {
			stableVY := result.VYValues[10]
			for _, vy := range result.VYValues[10:] {
				if vy != stableVY {
					result.VYFluctuation = true
					break
				}
			}
		}
	}

	return result
}

func TestReplayIdlePlayer_VelocityStability(t *testing.T) {
	// Create test replay data: player standing still for 120 frames (2 seconds)
	replayData := replay.CreateTestReplayData(120, 160, 120)
	replayer := replay.NewReplayer(replayData)

	cfg := createTestConfig()
	stage := createTestStageWithGround()
	player := createTestPlayer(stage)

	// Run simulation
	result := simulateWithReplay(replayer, cfg, stage, player)

	t.Logf("Simulated %d frames", result.FinalFrame)
	t.Logf("VY range: min=%f, max=%f", result.VYMin, result.VYMax)
	t.Logf("VY fluctuation detected: %v", result.VYFluctuation)

	// After physics fix: VY should be stable at 0 when on ground
	// Before fix: VY would oscillate between 0 and ~346
	assert.False(t, result.VYFluctuation, "VY should not fluctuate when player is idle on ground")
	assert.Equal(t, 0.0, result.VYValues[len(result.VYValues)-1], "Final VY should be 0")
}

func TestReplayIdlePlayer_TrajectoryStability(t *testing.T) {
	// This test simulates the trajectory calculation that was wobbling
	// When player is idle, trajectory should be consistent

	replayData := replay.CreateTestReplayData(60, 200, 100) // Mouse at (200, 100)
	replayer := replay.NewReplayer(replayData)

	cfg := createTestConfig()
	stage := createTestStageWithGround()
	player := createTestPlayer(stage)

	inputSystem := system.NewInputSystem(cfg)
	physicsSystem := system.NewPhysicsSystem(cfg, stage)
	dt := 1.0 / 60.0

	// Simulate and calculate trajectory direction each frame
	type TrajectorySnapshot struct {
		PlayerVY   float64
		AdjustedVY float64 // VY after OnGround check
		OnGround   bool
	}
	snapshots := make([]TrajectorySnapshot, 0, 60)

	for {
		input, ok := replayer.GetInput()
		if !ok {
			break
		}

		inputSystem.UpdatePlayer(player, input, dt)
		physicsSystem.Update(player, dt, 10)

		// Calculate trajectory velocity (same as main.go)
		playerVY := player.VY / entity.PositionScale
		adjustedVY := playerVY
		if player.OnGround {
			adjustedVY = 0
		}

		snapshots = append(snapshots, TrajectorySnapshot{
			PlayerVY:   playerVY,
			AdjustedVY: adjustedVY,
			OnGround:   player.OnGround,
		})
	}

	// Check that adjusted VY is stable (should all be 0 when on ground)
	t.Logf("Captured %d trajectory snapshots", len(snapshots))

	stableCount := 0
	for i, snap := range snapshots {
		if i >= 5 { // Skip first few frames for settling
			if snap.AdjustedVY == 0 && snap.OnGround {
				stableCount++
			} else {
				t.Logf("Frame %d: PlayerVY=%f, AdjustedVY=%f, OnGround=%v",
					i, snap.PlayerVY, snap.AdjustedVY, snap.OnGround)
			}
		}
	}

	t.Logf("Stable frames: %d/%d", stableCount, len(snapshots)-5)
	assert.Equal(t, len(snapshots)-5, stableCount, "All frames after settling should have stable trajectory")
}

func TestReplayDeterminism(t *testing.T) {
	// Test that replaying the same inputs produces identical results
	replayData := replay.CreateTestReplayData(60, 160, 120)

	cfg := createTestConfig()
	stage := createTestStageWithGround()

	// Run simulation twice
	player1 := createTestPlayer(stage)
	replayer1 := replay.NewReplayer(replayData)
	result1 := simulateWithReplay(replayer1, cfg, stage, player1)

	player2 := createTestPlayer(stage)
	replayer2 := replay.NewReplayer(replayData)
	result2 := simulateWithReplay(replayer2, cfg, stage, player2)

	// Results should be identical
	require.Equal(t, len(result1.VYValues), len(result2.VYValues), "Frame count should match")

	for i := range result1.VYValues {
		assert.Equal(t, result1.VYValues[i], result2.VYValues[i], "VY at frame %d should match", i)
		assert.Equal(t, result1.VXValues[i], result2.VXValues[i], "VX at frame %d should match", i)
		assert.Equal(t, result1.Positions[i], result2.Positions[i], "Position at frame %d should match", i)
	}

	t.Log("Determinism verified: two runs with same replay produce identical results")
}

func TestReplayWithMovement(t *testing.T) {
	// Create replay data with movement
	data := replay.ReplayData{
		Version: "1.0",
		Seed:    12345,
		Stage:   "test",
		Frames:  make([]replay.FrameInput, 120),
	}

	// First 30 frames: idle
	// Next 30 frames: move right
	// Next 30 frames: jump
	// Last 30 frames: idle
	for i := 0; i < 120; i++ {
		data.Frames[i] = replay.FrameInput{F: i, MX: 160, MY: 120}
		if i >= 30 && i < 60 {
			data.Frames[i].R = true // Move right
		}
		if i >= 60 && i < 90 {
			data.Frames[i].J = true // Hold jump
			if i == 60 {
				data.Frames[i].JP = true // Jump pressed
			}
		}
	}

	replayer := replay.NewReplayer(data)
	cfg := createTestConfig()
	stage := createTestStageWithGround()
	player := createTestPlayer(stage)

	result := simulateWithReplay(replayer, cfg, stage, player)

	t.Logf("Simulated %d frames with movement", result.FinalFrame)
	t.Logf("Final position: (%d, %d)", result.Positions[len(result.Positions)-1].X, result.Positions[len(result.Positions)-1].Y)

	// Player should have moved right
	assert.Greater(t, result.Positions[59].X, result.Positions[0].X, "Player should move right during frames 30-60")

	// Player should have jumped (Y decreased then increased)
	// Find minimum Y (highest point during jump)
	minY := result.Positions[60].Y
	for i := 60; i < 90; i++ {
		if result.Positions[i].Y < minY {
			minY = result.Positions[i].Y
		}
	}
	assert.Less(t, minY, result.Positions[60].Y, "Player should jump (Y should decrease)")
}

func TestRecorderAndReplayer(t *testing.T) {
	// Test that recorder and replayer work together
	seed := int64(12345)
	stage := "demo"

	// Record some inputs
	recorder := playing.NewRecorder(seed, stage)
	inputs := []system.InputState{
		{Left: false, Right: true, MouseX: 100, MouseY: 100},
		{Left: false, Right: true, Jump: true, JumpPressed: true, MouseX: 110, MouseY: 95},
		{Left: false, Right: true, Jump: true, MouseX: 120, MouseY: 90},
		{Left: false, Right: false, MouseX: 130, MouseY: 100},
	}

	for _, input := range inputs {
		recorder.RecordFrame(input)
	}

	assert.Equal(t, 4, recorder.FrameCount())

	// Create replayer from recorded data
	replayer := replay.NewReplayer(recorder.GetData())
	assert.Equal(t, seed, replayer.Seed())
	assert.Equal(t, 4, replayer.TotalFrames())

	// Replay and verify
	for i, expectedInput := range inputs {
		replayedInput, ok := replayer.GetInput()
		require.True(t, ok, "Should have input for frame %d", i)
		assert.Equal(t, expectedInput.Right, replayedInput.Right, "Right at frame %d", i)
		assert.Equal(t, expectedInput.Jump, replayedInput.Jump, "Jump at frame %d", i)
		assert.Equal(t, expectedInput.JumpPressed, replayedInput.JumpPressed, "JumpPressed at frame %d", i)
		assert.Equal(t, expectedInput.MouseX, replayedInput.MouseX, "MouseX at frame %d", i)
		assert.Equal(t, expectedInput.MouseY, replayedInput.MouseY, "MouseY at frame %d", i)
	}

	// Should be at end
	_, ok := replayer.GetInput()
	assert.False(t, ok, "Should be at end of replay")
}

func TestReplaySeedDeterminism(t *testing.T) {
	// Test that same seed produces same random sequence
	seed := int64(42)

	rng1 := rand.New(rand.NewSource(seed))
	rng2 := rand.New(rand.NewSource(seed))

	for i := 0; i < 100; i++ {
		v1 := rng1.Intn(100)
		v2 := rng2.Intn(100)
		assert.Equal(t, v1, v2, "Random value at step %d should match", i)
	}
}
