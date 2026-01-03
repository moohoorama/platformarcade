package playing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/younwookim/mg/internal/application/replay"
	"github.com/younwookim/mg/internal/application/scene"
	"github.com/younwookim/mg/internal/application/system"
	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

// createTestConfig creates a minimal config for testing
func createTestConfig() *config.GameConfig {
	return &config.GameConfig{
		Physics: &config.PhysicsConfig{
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
			Display: config.DisplayConfig{
				ScreenWidth:  320,
				ScreenHeight: 240,
				Scale:        2,
				Framerate:    60,
			},
			Feedback: config.FeedbackConfig{
				ScreenShake: config.ScreenShakeConfig{
					Intensity: 5.0,
					Decay:     0.9,
				},
			},
			Combat: config.CombatConfig{
				Iframes: 1.0,
			},
			ArrowSelect: config.ArrowSelectConfig{
				Radius:      40,
				MinDistance: 10,
				MaxFrame:    10,
			},
		},
		Entities: &config.EntitiesConfig{
			Player: config.PlayerConfig{
				Stats: config.PlayerStats{
					MaxHealth: 100,
				},
				Hitbox: config.HitboxConfig{
					Head: config.Rect{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
					Body: config.Rect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
					Feet: config.Rect{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
				},
				Sprite: config.SpriteConfig{
					FrameWidth:  16,
					FrameHeight: 24,
				},
			},
		},
	}
}

// createTestStageConfig creates a minimal stage config for testing
func createTestStageConfig() *config.StageConfig {
	return &config.StageConfig{
		Name: "test",
		Size: config.StageSizeConfig{
			Width:    10,
			Height:   5,
			TileSize: 16,
		},
		PlayerSpawn: config.PositionConfig{X: 80, Y: 46},
	}
}

// createTestStage creates a test stage with ground at y=4
func createTestStage() *entity.Stage {
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

func TestPlaying_ImplementsScene(t *testing.T) {
	// Compile-time check that Playing implements scene.Scene
	var _ scene.Scene = (*Playing)(nil)
}

func TestNewPlaying(t *testing.T) {
	cfg := createTestConfig()
	stageCfg := createTestStageConfig()
	stage := createTestStage()

	p := New(cfg, stageCfg, stage, "")

	assert.NotNil(t, p)
	assert.NotNil(t, p.player)
	assert.Equal(t, 100, p.player.MaxHealth)
}

func TestPlaying_Update_ReturnsNilWhenPlaying(t *testing.T) {
	cfg := createTestConfig()
	stageCfg := createTestStageConfig()
	stage := createTestStage()

	p := New(cfg, stageCfg, stage, "")

	// Normal update should return nil (stay on same scene)
	next, err := p.Update(1.0 / 60.0)

	assert.NoError(t, err)
	assert.Nil(t, next, "Should return nil when continuing to play")
}

func TestPlaying_OnEnter(t *testing.T) {
	cfg := createTestConfig()
	stageCfg := createTestStageConfig()
	stage := createTestStage()

	p := New(cfg, stageCfg, stage, "")

	// OnEnter should not panic
	assert.NotPanics(t, func() {
		p.OnEnter()
	})
}

func TestPlaying_OnExit(t *testing.T) {
	cfg := createTestConfig()
	stageCfg := createTestStageConfig()
	stage := createTestStage()

	p := New(cfg, stageCfg, stage, "")

	// OnExit should not panic
	assert.NotPanics(t, func() {
		p.OnExit()
	})
}

func TestPlaying_WithRecorder(t *testing.T) {
	cfg := createTestConfig()
	stageCfg := createTestStageConfig()
	stage := createTestStage()

	// Create with recording enabled
	p := New(cfg, stageCfg, stage, "test_replay.json")

	assert.NotNil(t, p.recorder)

	// Update should record frames
	_, err := p.Update(1.0 / 60.0)
	require.NoError(t, err)

	assert.Equal(t, 1, p.recorder.FrameCount())
}

func TestPlaying_SimulateWithReplay(t *testing.T) {
	cfg := createTestConfig()
	stageCfg := createTestStageConfig()
	stage := createTestStage()

	p := New(cfg, stageCfg, stage, "")

	// Player starts on ground (spawn position is on ground level)
	p.player.OnGround = true
	p.player.VY = 0

	// Create test replay data
	replayData := replay.CreateTestReplayData(60, 160, 120)
	replayer := replay.NewReplayer(replayData)

	// Simulate with replay
	inputSystem := system.NewInputSystem(cfg.Physics)
	physicsSystem := system.NewPhysicsSystem(cfg.Physics, stage)
	dt := 1.0 / 60.0

	for {
		input, ok := replayer.GetInput()
		if !ok {
			break
		}

		inputSystem.UpdatePlayer(p.player, input, dt)
		physicsSystem.Update(p.player, dt, 10)
	}

	// Player should still be on ground after idle simulation
	assert.True(t, p.player.OnGround)
	assert.Equal(t, 0.0, p.player.VY)
}

func TestRecorder_StopAndIsRecording(t *testing.T) {
	r := NewRecorder("test", "test.json")

	assert.True(t, r.IsRecording())

	r.Stop()

	assert.False(t, r.IsRecording())
}

func TestRecorder_DoesNotRecordWhenStopped(t *testing.T) {
	r := NewRecorder("test", "test.json")
	r.Stop()

	// Should not record when stopped
	input := system.InputState{Left: true}
	r.RecordFrame(input)

	assert.Equal(t, 0, r.FrameCount())
}

func TestPlaying_Draw(t *testing.T) {
	cfg := createTestConfig()
	stageCfg := createTestStageConfig()
	stage := createTestStage()

	p := New(cfg, stageCfg, stage, "")

	// Draw should not panic even with nil screen (stub implementation)
	assert.NotPanics(t, func() {
		p.Draw(nil)
	})
}

func TestPlaying_OnExitWithRecorder(t *testing.T) {
	cfg := createTestConfig()
	stageCfg := createTestStageConfig()
	stage := createTestStage()

	// Use temp file for recorder
	tmpFile := "/tmp/test_playing_onexit.json"

	p := New(cfg, stageCfg, stage, tmpFile)

	// Record some frames
	_, _ = p.Update(1.0 / 60.0)
	_, _ = p.Update(1.0 / 60.0)

	// OnExit should save without panic
	assert.NotPanics(t, func() {
		p.OnExit()
	})
}
