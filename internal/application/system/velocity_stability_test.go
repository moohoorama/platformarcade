package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

// TestVelocityStabilityWhenIdle tests that player velocity remains stable when standing still
func TestVelocityStabilityWhenIdle(t *testing.T) {
	// Setup config
	cfg := &config.PhysicsConfig{
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
			ApexModifier: config.ApexModifierConfig{
				Enabled: false,
			},
			FallMultiplier: 1.6,
		},
	}

	// Create a simple stage with ground
	stage := &entity.Stage{
		Width:    10,
		Height:   5,
		TileSize: 16,
		Tiles:    make([][]entity.Tile, 5),
	}
	for y := 0; y < 5; y++ {
		stage.Tiles[y] = make([]entity.Tile, 10)
		for x := 0; x < 10; x++ {
			// Ground at y=4
			if y == 4 {
				stage.Tiles[y][x] = entity.Tile{Solid: true}
			}
		}
	}

	// Create player standing on ground
	hitbox := entity.TrapezoidHitbox{
		Head: entity.HitboxRect{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: entity.HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: entity.HitboxRect{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	player := entity.NewPlayer(80, 40, hitbox, 100)
	player.OnGround = true
	player.VX = 0
	player.VY = 0

	physicsSystem := NewPhysicsSystem(cfg, stage)
	inputSystem := NewInputSystem(cfg)

	// No input (standing still)
	noInput := InputState{
		Left:  false,
		Right: false,
	}

	t.Run("VX should remain 0 when idle", func(t *testing.T) {
		// Reset
		player.VX = 0
		player.VY = 0
		player.OnGround = true

		// Record velocities over multiple frames
		vxValues := make([]float64, 0, 60)
		for i := 0; i < 60; i++ {
			inputSystem.UpdatePlayer(player, noInput, 1.0/60.0)
			physicsSystem.Update(player, 1.0/60.0, 10)
			vxValues = append(vxValues, player.VX)
		}

		// Check all VX values are 0
		for i, vx := range vxValues {
			assert.Equal(t, 0.0, vx, "Frame %d: VX should be 0, got %f", i, vx)
		}
	})

	t.Run("VY should remain stable when on ground", func(t *testing.T) {
		// Position player directly on ground
		// Ground at tile y=4 (pixel 64), feet offset=18, height=6
		// Player y=46 means feet at 64-70, which overlaps ground
		// Player y=45 means feet at 63-69, checks tiles 3-4, tile 4 is solid
		player.SetPixelPos(80, 46)
		player.VX = 0
		player.VY = 0
		player.OnGround = true // Already on ground

		t.Logf("Initial: Y=%d (internal=%d), OnGround=%v, VY=%f",
			player.PixelY(), player.Y, player.OnGround, player.VY)

		// Run physics and track VY
		vyValues := make([]float64, 0, 60)
		for i := 0; i < 60; i++ {
			beforeVY := player.VY
			beforeOnGround := player.OnGround

			inputSystem.UpdatePlayer(player, noInput, 1.0/60.0)
			physicsSystem.Update(player, 1.0/60.0, 10)

			if i < 5 {
				t.Logf("Frame %d: VY %.2f->%.2f, OnGround %v->%v, Y=%d",
					i, beforeVY, player.VY, beforeOnGround, player.OnGround, player.PixelY())
			}
			vyValues = append(vyValues, player.VY)
		}

		// Check VY stability
		t.Logf("VY values: min=%f, max=%f", minFloat(vyValues), maxFloat(vyValues))
		t.Logf("Final: OnGround=%v, Y=%d", player.OnGround, player.PixelY())

		// Count fluctuations
		fluctuationCount := 0
		for i := 1; i < len(vyValues); i++ {
			if vyValues[i] != vyValues[0] {
				fluctuationCount++
			}
		}
		t.Logf("Fluctuation count: %d out of 60 frames", fluctuationCount)

		// The main assertion: VY should be consistently 0 when on ground
		assert.Equal(t, 0.0, vyValues[len(vyValues)-1], "Final VY should be 0 when on ground")
	})

	t.Run("Velocity before and after physics should be consistent", func(t *testing.T) {
		// Reset
		player.SetPixelPos(80, 40)
		player.VX = 0
		player.VY = 0
		player.OnGround = true

		// Simulate and record velocity at different points
		type velocitySnapshot struct {
			beforePhysicsVX, beforePhysicsVY float64
			afterPhysicsVX, afterPhysicsVY   float64
		}
		snapshots := make([]velocitySnapshot, 0, 10)

		for i := 0; i < 10; i++ {
			inputSystem.UpdatePlayer(player, noInput, 1.0/60.0)

			snap := velocitySnapshot{
				beforePhysicsVX: player.VX,
				beforePhysicsVY: player.VY,
			}

			physicsSystem.Update(player, 1.0/60.0, 10)

			snap.afterPhysicsVX = player.VX
			snap.afterPhysicsVY = player.VY
			snapshots = append(snapshots, snap)
		}

		// Log the snapshots for analysis
		for i, snap := range snapshots {
			t.Logf("Frame %d: before(VX=%f, VY=%f) -> after(VX=%f, VY=%f)",
				i, snap.beforePhysicsVX, snap.beforePhysicsVY,
				snap.afterPhysicsVX, snap.afterPhysicsVY)
		}
	})
}

func minFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
