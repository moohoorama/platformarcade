package main

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/younwookim/mg/internal/application/replay"
	"github.com/younwookim/mg/internal/application/scene/playing"
	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/ecs"
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

// createTestWorld creates an ECS world with a player on ground
func createTestWorld(stage *entity.Stage) *ecs.World {
	hitbox := ecs.HitboxTrapezoid{
		Head: ecs.Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: ecs.Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: ecs.Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	world := ecs.NewWorld()
	world.CreatePlayer(stage.SpawnX, stage.SpawnY, hitbox, 100)

	// Set player on ground
	mov := world.Movement[world.PlayerID]
	mov.OnGround = true
	world.Movement[world.PlayerID] = mov

	return world
}

// toECSPhysicsConfig converts config.PhysicsConfig to ecs.PhysicsConfig
// All velocity values are converted to IU/substep at this point
func toECSPhysicsConfig(cfg *config.PhysicsConfig) ecs.PhysicsConfig {
	return ecs.PhysicsConfig{
		Gravity:                 ecs.ToIUPerSubstep(cfg.Physics.Gravity),
		MaxFallSpeed:            ecs.ToIUPerSubstep(cfg.Physics.MaxFallSpeed),
		MaxSpeed:                ecs.ToIUPerSubstep(cfg.Movement.MaxSpeed),
		Acceleration:            ecs.ToIUPerSubstep(cfg.Movement.Acceleration),
		Deceleration:            ecs.ToIUPerSubstep(cfg.Movement.Deceleration),
		AirControlPct:           ecs.PctToInt(cfg.Movement.AirControl),
		TurnaroundPct:           ecs.PctToInt(cfg.Movement.TurnaroundBoost),
		JumpForce:               ecs.ToIUPerSubstep(cfg.Jump.Force),
		VarJumpPct:              ecs.PctToInt(cfg.Jump.VariableJumpMultiplier),
		CoyoteFrames:            int(cfg.Jump.CoyoteTime * 60),
		JumpBufferFrames:        int(cfg.Jump.JumpBuffer * 60),
		DashSpeed:               ecs.ToIUPerSubstep(cfg.Dash.Speed),
		DashFrames:              int(cfg.Dash.Duration * 60),
		DashCooldownFrames:      int(cfg.Dash.Cooldown * 60),
		DashIframes:             int(cfg.Dash.Duration * 60),
		ApexModEnabled:          cfg.Jump.ApexModifier.Enabled,
		ApexThreshold:           ecs.ToIUPerSubstep(cfg.Jump.ApexModifier.Threshold),
		ApexGravityPct:          ecs.PctToInt(cfg.Jump.ApexModifier.GravityMultiplier),
		FallMultiplierPct:       ecs.PctToInt(cfg.Jump.FallMultiplier),
		CornerCorrectionMargin:  4,
		CornerCorrectionEnabled: true,
	}
}

// SimulationResult contains the results of a replay simulation
type SimulationResult struct {
	VYValues      []int
	VXValues      []int
	Positions     []struct{ X, Y int }
	FinalFrame    int
	VYMin         int
	VYMax         int
	VYFluctuation bool
}

// simulateWithReplay runs a game simulation using replayed inputs
func simulateWithReplay(replayer *replay.Replayer, cfg *config.PhysicsConfig, stage *entity.Stage, world *ecs.World) SimulationResult {
	result := SimulationResult{
		VYValues:  make([]int, 0, replayer.TotalFrames()),
		VXValues:  make([]int, 0, replayer.TotalFrames()),
		Positions: make([]struct{ X, Y int }, 0, replayer.TotalFrames()),
	}

	ecsCfg := toECSPhysicsConfig(cfg)

	for {
		input, ok := replayer.GetInput()
		if !ok {
			break
		}

		// Update game state using ECS systems
		ecs.UpdateTimers(world)
		ecs.UpdatePlayerInput(world, ecs.InputState{
			Left:         input.Left,
			Right:        input.Right,
			Up:           input.Up,
			Down:         input.Down,
			JumpPressed:  input.JumpPressed,
			JumpReleased: input.JumpReleased,
			Dash:         input.Dash,
		}, ecsCfg)
		// Normal speed: 10 sub-steps per frame
		for i := 0; i < 10; i++ {
			ecs.UpdatePlayerPhysics(world, stage, ecsCfg)
		}

		// Record state from ECS components
		vel := world.Velocity[world.PlayerID]
		pos := world.Position[world.PlayerID]
		result.VYValues = append(result.VYValues, vel.Y)
		result.VXValues = append(result.VXValues, vel.X)
		result.Positions = append(result.Positions, struct{ X, Y int }{pos.PixelX(), pos.PixelY()})
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
	world := createTestWorld(stage)

	// Run simulation
	result := simulateWithReplay(replayer, cfg, stage, world)

	t.Logf("Simulated %d frames", result.FinalFrame)
	t.Logf("VY range: min=%d, max=%d", result.VYMin, result.VYMax)
	t.Logf("VY fluctuation detected: %v", result.VYFluctuation)

	// After physics fix: VY should be stable at 0 when on ground
	// Before fix: VY would oscillate between 0 and ~346
	assert.False(t, result.VYFluctuation, "VY should not fluctuate when player is idle on ground")
	assert.Equal(t, 0, result.VYValues[len(result.VYValues)-1], "Final VY should be 0")
}

func TestReplayIdlePlayer_TrajectoryStability(t *testing.T) {
	// This test simulates the trajectory calculation that was wobbling
	// When player is idle, trajectory should be consistent

	replayData := replay.CreateTestReplayData(60, 200, 100) // Mouse at (200, 100)
	replayer := replay.NewReplayer(replayData)

	cfg := createTestConfig()
	ecsCfg := toECSPhysicsConfig(cfg)
	stage := createTestStageWithGround()
	world := createTestWorld(stage)

	// Simulate and calculate trajectory direction each frame
	type TrajectorySnapshot struct {
		PlayerVY   int
		AdjustedVY int // VY after OnGround check
		OnGround   bool
	}
	snapshots := make([]TrajectorySnapshot, 0, 60)

	for {
		input, ok := replayer.GetInput()
		if !ok {
			break
		}

		ecs.UpdateTimers(world)
		ecs.UpdatePlayerInput(world, ecs.InputState{
			Left:         input.Left,
			Right:        input.Right,
			Up:           input.Up,
			Down:         input.Down,
			JumpPressed:  input.JumpPressed,
			JumpReleased: input.JumpReleased,
			Dash:         input.Dash,
		}, ecsCfg)
		for i := 0; i < 10; i++ {
			ecs.UpdatePlayerPhysics(world, stage, ecsCfg)
		}

		// Calculate trajectory velocity (same as rendering code)
		vel := world.Velocity[world.PlayerID]
		mov := world.Movement[world.PlayerID]
		playerVY := vel.Y / ecs.PositionScale
		adjustedVY := playerVY
		if mov.OnGround {
			adjustedVY = 0
		}

		snapshots = append(snapshots, TrajectorySnapshot{
			PlayerVY:   playerVY,
			AdjustedVY: adjustedVY,
			OnGround:   mov.OnGround,
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
				t.Logf("Frame %d: PlayerVY=%d, AdjustedVY=%d, OnGround=%v",
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
	world1 := createTestWorld(stage)
	replayer1 := replay.NewReplayer(replayData)
	result1 := simulateWithReplay(replayer1, cfg, stage, world1)

	world2 := createTestWorld(stage)
	replayer2 := replay.NewReplayer(replayData)
	result2 := simulateWithReplay(replayer2, cfg, stage, world2)

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
	world := createTestWorld(stage)

	result := simulateWithReplay(replayer, cfg, stage, world)

	t.Logf("Simulated %d frames with movement", result.FinalFrame)
	t.Logf("Final position: (%d, %d)", result.Positions[len(result.Positions)-1].X, result.Positions[len(result.Positions)-1].Y)

	// Player should have moved right
	assert.Greater(t, result.Positions[59].X, result.Positions[0].X, "Player should move right during frames 30-60")

	// Player should have jumped (Y decreased then increased)
	// Compare minimum Y during jump phase to the Y before jump
	// Use the position at frame 59 (just before jump) as baseline
	baselineY := result.Positions[59].Y
	minY := baselineY
	for i := 60; i < 90; i++ {
		if result.Positions[i].Y < minY {
			minY = result.Positions[i].Y
		}
	}
	t.Logf("Baseline Y (frame 59): %d, Min Y during jump: %d", baselineY, minY)
	assert.Less(t, minY, baselineY, "Player should jump (Y should decrease from baseline)")
}

func TestRecorderAndReplayer(t *testing.T) {
	// Test that recorder and replayer work together
	seed := int64(12345)
	stage := "demo"

	// Record some inputs
	recorder := playing.NewRecorder(seed, stage)
	inputs := []playing.RecordableInput{
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
