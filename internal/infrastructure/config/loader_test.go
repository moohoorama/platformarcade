package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_LoadPhysics(t *testing.T) {
	loader := NewLoader("../../../cmd/game/configs")

	cfg, err := loader.LoadPhysics()
	require.NoError(t, err)

	assert.Equal(t, 320, cfg.Display.ScreenWidth)
	assert.Equal(t, 240, cfg.Display.ScreenHeight)
	assert.Equal(t, 60, cfg.Display.Framerate)
	assert.Equal(t, 800.0, cfg.Physics.Gravity)
	assert.Equal(t, 0.1, cfg.Jump.CoyoteTime)
	assert.True(t, cfg.Feedback.Hitstop.Enabled)
}

func TestLoader_LoadEntities(t *testing.T) {
	loader := NewLoader("../../../cmd/game/configs")

	cfg, err := loader.LoadEntities()
	require.NoError(t, err)

	assert.Equal(t, "player", cfg.Player.ID)
	assert.Equal(t, 100, cfg.Player.Stats.MaxHealth)
	assert.Equal(t, 8, cfg.Player.Hitbox.Head.Width)
	assert.Equal(t, 16, cfg.Player.Hitbox.Feet.Width)

	arrow, ok := cfg.Projectiles["playerArrow"]
	require.True(t, ok)
	assert.Equal(t, 20.0, arrow.Physics.LaunchAngleDeg)
	assert.Equal(t, 500.0, arrow.Physics.GravityAccel)

	slime, ok := cfg.Enemies["slime"]
	require.True(t, ok)
	assert.Equal(t, "patrol", slime.AI.Type)
}

func TestLoader_LoadStage(t *testing.T) {
	loader := NewLoader("../../../cmd/game/configs")

	cfg, err := loader.LoadStage("demo")
	require.NoError(t, err)

	assert.Equal(t, "demo", cfg.ID)
	assert.Equal(t, 640, cfg.Size.Width)
	assert.Equal(t, 480, cfg.Size.Height)
	assert.Equal(t, 16, cfg.Size.TileSize)
	assert.Equal(t, 48, cfg.PlayerSpawn.X)
	assert.Equal(t, 400, cfg.PlayerSpawn.Y)
	assert.Len(t, cfg.Layers.Collision, 30)

	wall, ok := cfg.TileMapping["#"]
	require.True(t, ok)
	assert.True(t, wall.Solid)
	assert.Equal(t, "wall", wall.Type)
}

func TestLoader_LoadAll(t *testing.T) {
	loader := NewLoader("../../../cmd/game/configs")

	cfg, err := loader.LoadAll()
	require.NoError(t, err)

	assert.NotNil(t, cfg.Physics)
	assert.NotNil(t, cfg.Entities)
}
