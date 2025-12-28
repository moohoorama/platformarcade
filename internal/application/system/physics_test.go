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
		player.X = 32
		player.VX = 100

		sys.moveX(player, 5)

		assert.Equal(t, 37, player.X)
		assert.False(t, player.OnWallRight)
	})

	t.Run("moves left without collision", func(t *testing.T) {
		player := createTestPlayer()
		player.X = 48
		player.VX = -100

		sys.moveX(player, -5)

		assert.Equal(t, 43, player.X)
		assert.False(t, player.OnWallLeft)
	})

	t.Run("stops at right wall", func(t *testing.T) {
		player := createTestPlayer()
		player.X = 44 // Close to right wall (starts at x=64)
		player.VX = 100

		sys.moveX(player, 20)

		assert.True(t, player.OnWallRight)
		assert.Equal(t, 0.0, player.VX)
	})

	t.Run("zero movement", func(t *testing.T) {
		player := createTestPlayer()
		player.X = 32

		sys.moveX(player, 0)

		assert.Equal(t, 32, player.X)
	})
}

func TestPhysicsSystem_MoveY(t *testing.T) {
	cfg := createTestPhysicsConfig()
	stage := createTestStage()
	sys := NewPhysicsSystem(cfg, stage)

	t.Run("falls without collision", func(t *testing.T) {
		player := createTestPlayer()
		player.Y = 32
		player.VY = 100

		sys.moveY(player, 5)

		assert.Equal(t, 37, player.Y)
		assert.False(t, player.OnGround)
	})

	t.Run("lands on ground", func(t *testing.T) {
		player := createTestPlayer()
		player.Y = 40 // Close to bottom (wall at y=64)
		player.VY = 100

		sys.moveY(player, 20)

		assert.True(t, player.OnGround)
		assert.Equal(t, 0.0, player.VY)
	})

	t.Run("hits ceiling", func(t *testing.T) {
		player := createTestPlayer()
		player.Y = 20
		player.VY = -100

		sys.moveY(player, -10)

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
		player.X = 32
		player.Y = 32

		result := sys.resolveOverlap(player)

		assert.True(t, result)
	})

	t.Run("overlapping wall - pushes out", func(t *testing.T) {
		player := createTestPlayer()
		player.X = 12 // Slightly inside left wall (body at x=14, needs 2px push to x=16)
		player.Y = 32

		initialX := player.X
		result := sys.resolveOverlap(player)

		assert.True(t, result)
		assert.NotEqual(t, initialX, player.X) // Should have moved
	})

	t.Run("completely stuck - resets to spawn", func(t *testing.T) {
		player := createTestPlayer()
		player.X = 0 // Completely inside corner
		player.Y = 0
		player.VX = 100
		player.VY = 100

		result := sys.resolveOverlap(player)

		assert.False(t, result)
		assert.Equal(t, stage.SpawnX, player.X)
		assert.Equal(t, stage.SpawnY, player.Y)
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
		player.VY = 500 // Already above max

		sys.applyGravity(player, 0.016)

		assert.Equal(t, cfg.Physics.MaxFallSpeed, player.VY)
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
		player.VY = 10 // Below apex threshold

		sys.applyGravity(player, 0.016)
		vyWithApex := player.VY

		player2 := createTestPlayer()
		player2.VY = 200 // Above apex threshold

		sys.applyGravity(player2, 0.016)
		vyWithoutApex := player2.VY - 200

		// Apex modifier should result in less gravity applied
		assert.Less(t, vyWithApex-10, vyWithoutApex)
	})
}

func TestPhysicsSystem_ApplyMovement(t *testing.T) {
	cfg := createTestPhysicsConfig()
	stage := createTestStage()
	sys := NewPhysicsSystem(cfg, stage)

	t.Run("moves player and resets flags", func(t *testing.T) {
		player := createTestPlayer()
		player.X = 32
		player.Y = 32
		player.OnGround = true
		player.OnCeiling = true
		player.OnWallLeft = true
		player.OnWallRight = true

		sys.applyMovement(player, 2, 3)

		// Position should change
		assert.Equal(t, 34, player.X)
		assert.Equal(t, 35, player.Y)
		// Flags should be updated based on movement result
		assert.False(t, player.OnCeiling)
		assert.False(t, player.OnWallLeft)
	})

	t.Run("handles zero movement", func(t *testing.T) {
		player := createTestPlayer()
		player.X = 32
		player.Y = 32

		sys.applyMovement(player, 0, 0)

		assert.Equal(t, 32, player.X)
		assert.Equal(t, 32, player.Y)
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
