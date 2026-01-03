package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

func createTestPhysicsConfig() *config.PhysicsConfig {
	return &config.PhysicsConfig{
		Physics: config.PhysicsSettings{
			Substeps:     1,
			Gravity:      800,
			MaxFallSpeed: 400,
		},
		Jump: config.JumpConfig{
			Force:                  280,
			CoyoteTime:             0.1,
			JumpBuffer:             0.1,
			VariableJumpMultiplier: 0.5,
			FallMultiplier:         1.5,
			ApexModifier: config.ApexModifierConfig{
				Enabled:           true,
				Threshold:         50,
				GravityMultiplier: 0.5,
			},
		},
		Collision: config.CollisionConfig{
			CornerCorrection: config.MarginConfig{
				Enabled: true,
				Margin:  4,
			},
		},
	}
}

func createTestStage() *entity.Stage {
	// Create a simple stage with walls around empty center
	// 5x5 tile map with 16px tiles = 80x80 pixels
	tiles := make([][]entity.Tile, 5)
	for y := 0; y < 5; y++ {
		tiles[y] = make([]entity.Tile, 5)
		for x := 0; x < 5; x++ {
			// Walls on edges, empty in center
			if x == 0 || x == 4 || y == 0 || y == 4 {
				tiles[y][x] = entity.Tile{Type: entity.TileWall, Solid: true}
			} else {
				tiles[y][x] = entity.Tile{Type: entity.TileEmpty, Solid: false}
			}
		}
	}

	return &entity.Stage{
		Width:    5,
		Height:   5,
		TileSize: 16,
		Tiles:    tiles,
		SpawnX:   32,
		SpawnY:   32,
	}
}

func createTestPlayer() *entity.Player {
	hitbox := entity.TrapezoidHitbox{
		Head: entity.HitboxRect{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: entity.HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: entity.HitboxRect{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	return entity.NewPlayer(32, 32, hitbox, 100)
}

func TestNewPhysicsSystem(t *testing.T) {
	cfg := createTestPhysicsConfig()
	stage := createTestStage()

	sys := NewPhysicsSystem(cfg, stage)

	require.NotNil(t, sys)
	assert.Equal(t, cfg, sys.config)
	assert.Equal(t, stage, sys.stage)
}

func TestPhysicsSystem_IsSolidRect(t *testing.T) {
	cfg := createTestPhysicsConfig()
	stage := createTestStage()
	sys := NewPhysicsSystem(cfg, stage)

	tests := []struct {
		name       string
		x, y, w, h int
		want       bool
	}{
		{
			name: "empty center",
			x:    24, y: 24, w: 8, h: 8,
			want: false,
		},
		{
			name: "touching left wall",
			x:    0, y: 24, w: 8, h: 8,
			want: true,
		},
		{
			name: "touching top wall",
			x:    24, y: 0, w: 8, h: 8,
			want: true,
		},
		{
			name: "large hitbox spanning multiple empty tiles",
			x:    20, y: 20, w: 30, h: 30,
			want: false,
		},
		{
			name: "large hitbox touching wall",
			x:    16, y: 16, w: 49, h: 49, // Extends to pixel 64 which is the wall tile
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sys.isSolidRect(tt.x, tt.y, tt.w, tt.h)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPhysicsSystem_MoveX(t *testing.T) {
	cfg := createTestPhysicsConfig()
	stage := createTestStage()
	sys := NewPhysicsSystem(cfg, stage)

	t.Run("moves right without collision", func(t *testing.T) {
		player := createTestPlayer()
		// Player starts at pixel (32, 32) = scaled (3200, 3200)
		player.VX = 10000 // 100 pixels/sec in 100x scale

		// Move 5 pixels (500 units) right
		sys.moveX(player, 500)

		assert.Equal(t, 37, player.PixelX())
		assert.False(t, player.OnWallRight)
	})

	t.Run("moves left without collision", func(t *testing.T) {
		player := createTestPlayer()
		player.SetPixelPos(48, 32) // Set to pixel (48, 32)
		player.VX = -10000

		// Move 5 pixels (500 units) left
		sys.moveX(player, -500)

		assert.Equal(t, 43, player.PixelX())
		assert.False(t, player.OnWallLeft)
	})

	t.Run("stops at right wall", func(t *testing.T) {
		player := createTestPlayer()
		player.SetPixelPos(44, 32) // Close to right wall (starts at x=64)
		player.VX = 10000

		// Try to move 20 pixels right (2000 units)
		sys.moveX(player, 2000)

		assert.True(t, player.OnWallRight)
		assert.Equal(t, 0.0, player.VX)
	})

	t.Run("zero movement", func(t *testing.T) {
		player := createTestPlayer()
		initialPixelX := player.PixelX()

		sys.moveX(player, 0)

		assert.Equal(t, initialPixelX, player.PixelX())
	})
}

func TestPhysicsSystem_MoveY(t *testing.T) {
	cfg := createTestPhysicsConfig()
	stage := createTestStage()
	sys := NewPhysicsSystem(cfg, stage)

	t.Run("falls without collision", func(t *testing.T) {
		player := createTestPlayer()
		// Player starts at pixel (32, 32)
		player.VY = 10000 // 100 pixels/sec in 100x scale

		// Move 5 pixels (500 units) down
		sys.moveY(player, 500)

		assert.Equal(t, 37, player.PixelY())
		assert.False(t, player.OnGround)
	})

	t.Run("lands on ground", func(t *testing.T) {
		player := createTestPlayer()
		player.SetPixelPos(32, 40) // Close to bottom (wall at y=64)
		player.VY = 10000

		// Try to move 20 pixels (2000 units) down
		sys.moveY(player, 2000)

		assert.True(t, player.OnGround)
		assert.Equal(t, 0.0, player.VY)
	})

	t.Run("hits ceiling", func(t *testing.T) {
		player := createTestPlayer()
		player.SetPixelPos(32, 20)
		player.VY = -10000

		// Try to move 10 pixels (1000 units) up
		sys.moveY(player, -1000)

		assert.True(t, player.OnCeiling)
		assert.Equal(t, 0.0, player.VY)
	})
}

func TestPhysicsSystem_ResolveOverlap(t *testing.T) {
	cfg := createTestPhysicsConfig()
	stage := createTestStage()
	sys := NewPhysicsSystem(cfg, stage)

	t.Run("no overlap - returns true", func(t *testing.T) {
		player := createTestPlayer()
		// Player starts at pixel (32, 32) which is in empty center

		result := sys.resolveOverlap(player)

		assert.True(t, result)
	})

	t.Run("overlapping wall - pushes out", func(t *testing.T) {
		player := createTestPlayer()
		player.SetPixelPos(12, 32) // Slightly inside left wall

		initialPixelX := player.PixelX()
		result := sys.resolveOverlap(player)

		assert.True(t, result)
		assert.NotEqual(t, initialPixelX, player.PixelX()) // Should have moved
	})

	t.Run("completely stuck - resets to spawn", func(t *testing.T) {
		player := createTestPlayer()
		player.SetPixelPos(0, 0) // Completely inside corner
		player.VX = 10000
		player.VY = 10000

		result := sys.resolveOverlap(player)

		assert.False(t, result)
		// spawn position is 32, 32 in pixels, which becomes 3200, 3200 in 100x
		assert.Equal(t, stage.SpawnX, player.PixelX())
		assert.Equal(t, stage.SpawnY, player.PixelY())
		assert.Equal(t, 0.0, player.VX)
		assert.Equal(t, 0.0, player.VY)
	})
}

func TestPhysicsSystem_ApplyGravity(t *testing.T) {
	cfg := createTestPhysicsConfig()
	stage := createTestStage()
	sys := NewPhysicsSystem(cfg, stage)

	t.Run("applies gravity", func(t *testing.T) {
		player := createTestPlayer()
		player.VY = 0

		sys.applyGravity(player, 0.016)

		assert.Greater(t, player.VY, 0.0)
	})

	t.Run("clamps to max fall speed", func(t *testing.T) {
		player := createTestPlayer()
		// VY is now in 100x scale, so 50000 = 500 pixels/sec (above max 400)
		player.VY = 50000

		sys.applyGravity(player, 0.016)

		// Max fall speed is 400 pixels/sec = 40000 in 100x scale
		maxFallSpeedScaled := cfg.Physics.MaxFallSpeed * entity.PositionScale
		assert.Equal(t, maxFallSpeedScaled, player.VY)
	})

	t.Run("no gravity during dash", func(t *testing.T) {
		player := createTestPlayer()
		player.VY = 0
		player.Dashing = true

		sys.applyGravity(player, 0.016)

		assert.Equal(t, 0.0, player.VY)
	})

	t.Run("apex modifier reduces gravity", func(t *testing.T) {
		player := createTestPlayer()
		// VY in 100x scale: 1000 = 10 pixels/sec (below apex threshold of 50)
		player.VY = 1000

		sys.applyGravity(player, 0.016)
		vyWithApex := player.VY

		player2 := createTestPlayer()
		// VY = 20000 = 200 pixels/sec (above apex threshold of 50)
		player2.VY = 20000

		sys.applyGravity(player2, 0.016)
		vyWithoutApex := player2.VY - 20000

		// Apex modifier should result in less gravity applied
		assert.Less(t, vyWithApex-1000, vyWithoutApex)
	})
}

func TestPhysicsSystem_ApplyMovement(t *testing.T) {
	cfg := createTestPhysicsConfig()
	stage := createTestStage()
	sys := NewPhysicsSystem(cfg, stage)

	t.Run("moves player and resets flags", func(t *testing.T) {
		player := createTestPlayer()
		// Player starts at pixel (32, 32)
		player.OnGround = true
		player.OnCeiling = true
		player.OnWallLeft = true
		player.OnWallRight = true

		// Move 2 pixels (200 units) right, 3 pixels (300 units) down
		sys.applyMovement(player, 200, 300)

		// Position should change
		assert.Equal(t, 34, player.PixelX())
		assert.Equal(t, 35, player.PixelY())
		// Flags should be updated based on movement result
		assert.False(t, player.OnCeiling)
		assert.False(t, player.OnWallLeft)
	})

	t.Run("handles zero movement", func(t *testing.T) {
		player := createTestPlayer()
		initialPixelX := player.PixelX()
		initialPixelY := player.PixelY()

		sys.applyMovement(player, 0, 0)

		assert.Equal(t, initialPixelX, player.PixelX())
		assert.Equal(t, initialPixelY, player.PixelY())
	})
}

func TestPhysicsSystem_CornerCorrection(t *testing.T) {
	cfg := createTestPhysicsConfig()
	cfg.Collision.CornerCorrection.Enabled = true
	cfg.Collision.CornerCorrection.Margin = 4
	stage := createTestStage()
	sys := NewPhysicsSystem(cfg, stage)

	t.Run("attempts corner correction on ceiling hit", func(t *testing.T) {
		player := createTestPlayer()
		player.X = 32
		player.Y = 20
		player.VY = -100

		// Move up to hit ceiling
		sys.moveY(player, -10)

		// Should have hit ceiling and attempted corner correction
		assert.True(t, player.OnCeiling)
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("sign", func(t *testing.T) {
		assert.Equal(t, 1, sign(5))
		assert.Equal(t, -1, sign(-5))
		assert.Equal(t, 0, sign(0))
	})

	t.Run("abs", func(t *testing.T) {
		assert.Equal(t, 5, abs(5))
		assert.Equal(t, 5, abs(-5))
		assert.Equal(t, 0, abs(0))
	})

	t.Run("absFloat", func(t *testing.T) {
		assert.Equal(t, 5.5, absFloat(5.5))
		assert.Equal(t, 5.5, absFloat(-5.5))
		assert.Equal(t, 0.0, absFloat(0.0))
	})
}

// ============================================================
// 100x Scale + Sub-step Tests (Phase 2)
// ============================================================

func TestPhysicsSystem_SubSteps(t *testing.T) {
	cfg := createTestPhysicsConfig()
	stage := createTestStage()
	sys := NewPhysicsSystem(cfg, stage)

	t.Run("10 sub-steps produces normal speed movement", func(t *testing.T) {
		player := createTestPlayer()
		// Player starts at pixel (32,32) = scaled (3200, 3200)
		initialPixelX := player.PixelX()
		initialPixelY := player.PixelY()

		// Set velocity: 120 pixels/sec = 12000 units/sec
		player.VX = 12000
		player.VY = 0

		dt := 1.0 / 60.0 // One frame at 60fps
		subSteps := 10   // Normal speed

		sys.Update(player, dt, subSteps)

		// After 1 frame at 120 pixels/sec:
		// movement = 120 * (1/60) = 2 pixels
		expectedPixelX := initialPixelX + 2
		assert.Equal(t, expectedPixelX, player.PixelX(), "Should move 2 pixels right")
		assert.Equal(t, initialPixelY, player.PixelY(), "Y should not change")
	})

	t.Run("1 sub-step produces 1/10 speed (slow motion)", func(t *testing.T) {
		player := createTestPlayer()
		initialPixelX := player.PixelX()

		// Set velocity: 120 pixels/sec = 12000 units/sec
		player.VX = 12000
		player.VY = 0

		dt := 1.0 / 60.0
		subSteps := 1 // Slow motion

		sys.Update(player, dt, subSteps)

		// In slow motion, only 1/10 of movement happens
		// Per sub-step: 12000 * (1/600) = 20 units = 0.2 pixels
		// So after 1 sub-step, movement is about 0 pixels (rounded down)
		// But after 10 frames of slow motion: 10 * 20 = 200 units = 2 pixels
		assert.Equal(t, initialPixelX, player.PixelX(), "Should barely move in 1 sub-step")
	})

	t.Run("multiple slow motion frames accumulate correctly", func(t *testing.T) {
		player := createTestPlayer()
		initialX := player.X // Track internal units

		player.VX = 12000 // 120 pixels/sec in 100x scale
		player.VY = 0

		dt := 1.0 / 60.0
		subSteps := 1 // Slow motion

		// Run 10 frames of slow motion
		for i := 0; i < 10; i++ {
			sys.Update(player, dt, subSteps)
		}

		// After 10 slow-motion frames, should have moved ~2 pixels (200 units)
		movedUnits := player.X - initialX
		assert.InDelta(t, 200, movedUnits, 20, "Should move about 200 units (2 pixels) after 10 slow-mo frames")
	})
}

func TestPhysicsSystem_CollisionScaled(t *testing.T) {
	cfg := createTestPhysicsConfig()
	stage := createTestStage()
	sys := NewPhysicsSystem(cfg, stage)

	t.Run("collision detection works with 100x scale", func(t *testing.T) {
		player := createTestPlayer()
		// Player at pixel (32, 32) = scaled (3200, 3200)
		// This is in the empty center of the stage

		// Verify no collision at center
		assert.False(t, player.OnWallRight)
		assert.False(t, player.OnGround)

		// Move toward right wall (wall starts at pixel 64)
		player.VX = 50000 // Fast movement
		player.VY = 0

		sys.Update(player, 1.0/60.0, 10)

		// Should have moved but not past the wall
		assert.True(t, player.PixelX() <= 64-16, "Should not pass through wall")
	})

	t.Run("isSolidRect works with pixel coordinates", func(t *testing.T) {
		// isSolidRect takes pixel coordinates (not 100x scaled)
		// Center of stage (pixel 24-40) should be empty
		centerX := 24
		centerY := 24
		width := 8
		height := 8

		result := sys.isSolidRect(centerX, centerY, width, height)
		assert.False(t, result, "Center should be empty")

		// Left wall (pixel 0-16) should be solid
		wallX := 0
		wallY := 24

		result = sys.isSolidRect(wallX, wallY, width, height)
		assert.True(t, result, "Wall should be solid")
	})
}
