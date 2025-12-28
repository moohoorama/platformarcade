package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

func createTestPlayerForInput() *entity.Player {
	hitbox := entity.TrapezoidHitbox{
		Head: entity.HitboxRect{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: entity.HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: entity.HitboxRect{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	return entity.NewPlayer(32, 32, hitbox, 100)
}

func createTestInputConfig() *config.PhysicsConfig {
	return &config.PhysicsConfig{
		Physics: config.PhysicsSettings{
			Gravity:      800,
			MaxFallSpeed: 400,
		},
		Movement: config.MovementConfig{
			MaxSpeed:     150,
			Acceleration: 800,
			Deceleration: 600,
			AirControl:   0.7,
		},
		Jump: config.JumpConfig{
			Force:                  280,
			CoyoteTime:             0.1,
			JumpBuffer:             0.1,
			VariableJumpMultiplier: 0.5,
			FallMultiplier:         1.5,
		},
		Dash: config.DashConfig{
			Speed:           300,
			Duration:        0.15,
			Cooldown:        0.5,
			IframesDuration: 0.15,
		},
	}
}

func TestNewInputSystem(t *testing.T) {
	cfg := createTestInputConfig()

	sys := NewInputSystem(cfg)

	require.NotNil(t, sys)
	assert.Equal(t, cfg, sys.config)
}

func TestInputSystem_UpdateTimers(t *testing.T) {
	cfg := createTestInputConfig()
	sys := NewInputSystem(cfg)

	t.Run("decrements coyote timer when in air", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.CoyoteTimer = 0.1
		player.OnGround = false

		sys.updateTimers(player, 0.05)

		assert.InDelta(t, 0.05, player.CoyoteTimer, 0.001)
	})

	t.Run("resets coyote timer when on ground", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.CoyoteTimer = 0.0
		player.OnGround = true

		sys.updateTimers(player, 0.016)

		assert.Equal(t, cfg.Jump.CoyoteTime, player.CoyoteTimer)
	})

	t.Run("decrements dash timer", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.DashTimer = 0.1
		player.Dashing = true

		sys.updateTimers(player, 0.05)

		assert.InDelta(t, 0.05, player.DashTimer, 0.001)
		assert.True(t, player.Dashing)
	})

	t.Run("ends dash when timer expires", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.DashTimer = 0.01
		player.Dashing = true

		sys.updateTimers(player, 0.02)

		assert.False(t, player.Dashing)
	})

	t.Run("restores dash on ground", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.CanDash = false
		player.OnGround = true

		sys.updateTimers(player, 0.016)

		assert.True(t, player.CanDash)
	})

	t.Run("decrements iframe timer", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.IframeTimer = 0.5

		sys.updateTimers(player, 0.1)

		assert.InDelta(t, 0.4, player.IframeTimer, 0.001)
	})

	t.Run("decrements stun timer", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.StunTimer = 0.5

		sys.updateTimers(player, 0.1)

		assert.InDelta(t, 0.4, player.StunTimer, 0.001)
	})
}

func TestInputSystem_HandleMovement(t *testing.T) {
	cfg := createTestInputConfig()
	sys := NewInputSystem(cfg)

	t.Run("accelerates right", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.VX = 0
		player.OnGround = true

		input := InputState{Right: true}
		sys.handleMovement(player, input)

		assert.Greater(t, player.VX, 0.0)
	})

	t.Run("accelerates left", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.VX = 0
		player.OnGround = true

		input := InputState{Left: true}
		sys.handleMovement(player, input)

		assert.Less(t, player.VX, 0.0)
	})

	t.Run("decelerates when no input", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.VX = 100
		player.OnGround = true

		input := InputState{}
		sys.handleMovement(player, input)

		assert.Less(t, player.VX, 100.0)
	})

	t.Run("no movement when dashing", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.VX = 300
		player.Dashing = true

		input := InputState{Left: true}
		sys.handleMovement(player, input)

		assert.Equal(t, 300.0, player.VX)
	})
}

func TestInputSystem_HandleDash(t *testing.T) {
	cfg := createTestInputConfig()
	sys := NewInputSystem(cfg)

	t.Run("dashes when can dash", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.CanDash = true
		player.FacingRight = true

		input := InputState{Dash: true}
		sys.handleDash(player, input)

		assert.True(t, player.Dashing)
		assert.False(t, player.CanDash)
		assert.Equal(t, cfg.Dash.Speed, player.VX)
		assert.Equal(t, 0.0, player.VY)
	})

	t.Run("dashes left when facing left", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.CanDash = true
		player.FacingRight = false

		input := InputState{Dash: true}
		sys.handleDash(player, input)

		assert.Equal(t, -cfg.Dash.Speed, player.VX)
	})

	t.Run("cannot dash when already dashing", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.Dashing = true
		player.CanDash = true

		input := InputState{Dash: true}
		sys.handleDash(player, input)

		assert.True(t, player.Dashing)
	})

	t.Run("cannot dash when can dash is false", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.CanDash = false

		input := InputState{Dash: true}
		sys.handleDash(player, input)

		assert.False(t, player.Dashing)
	})

	t.Run("cannot dash during cooldown", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.CanDash = true
		player.DashCooldown = 0.5

		input := InputState{Dash: true}
		sys.handleDash(player, input)

		assert.False(t, player.Dashing)
	})
}

func TestInputSystem_HandleJump(t *testing.T) {
	cfg := createTestInputConfig()
	sys := NewInputSystem(cfg)

	t.Run("jumps when on ground", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.OnGround = true
		player.JumpBufferTimer = 0.05

		input := InputState{}
		sys.handleJump(player, input)

		assert.Equal(t, -cfg.Jump.Force, player.VY)
		assert.False(t, player.OnGround)
		assert.Equal(t, 0.0, player.JumpBufferTimer)
	})

	t.Run("jumps with coyote time", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.OnGround = false
		player.CoyoteTimer = 0.05
		player.JumpBufferTimer = 0.05

		input := InputState{}
		sys.handleJump(player, input)

		assert.Equal(t, -cfg.Jump.Force, player.VY)
		assert.Equal(t, 0.0, player.CoyoteTimer)
	})

	t.Run("buffers jump when pressed", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.JumpBufferTimer = 0

		input := InputState{JumpPressed: true}
		sys.handleJump(player, input)

		assert.Equal(t, cfg.Jump.JumpBuffer, player.JumpBufferTimer)
	})

	t.Run("variable jump reduces velocity on release", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.VY = -200

		input := InputState{JumpReleased: true}
		sys.handleJump(player, input)

		assert.Equal(t, -200*cfg.Jump.VariableJumpMultiplier, player.VY)
	})

	t.Run("variable jump only affects upward velocity", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.VY = 100 // Falling

		input := InputState{JumpReleased: true}
		sys.handleJump(player, input)

		assert.Equal(t, 100.0, player.VY) // Unchanged
	})

	t.Run("cannot jump without buffer or ground", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.OnGround = false
		player.CoyoteTimer = 0
		player.JumpBufferTimer = 0
		player.VY = 0

		input := InputState{}
		sys.handleJump(player, input)

		assert.Equal(t, 0.0, player.VY)
	})
}

func TestInputSystem_HandleMovementAirControl(t *testing.T) {
	cfg := createTestInputConfig()
	sys := NewInputSystem(cfg)

	t.Run("air control affects acceleration", func(t *testing.T) {
		playerAir := createTestPlayerForInput()
		playerAir.OnGround = false
		playerAir.VX = 0

		input := InputState{Right: true}
		sys.handleMovement(playerAir, input)

		// Air control should still allow movement
		assert.Greater(t, playerAir.VX, 0.0)
	})
}

func TestInputSystem_DecelerationToZero(t *testing.T) {
	cfg := createTestInputConfig()
	sys := NewInputSystem(cfg)

	t.Run("decelerates to zero", func(t *testing.T) {
		player := createTestPlayerForInput()
		player.OnGround = true
		player.VX = 50

		input := InputState{}
		// Decelerate multiple times
		for i := 0; i < 100; i++ {
			sys.handleMovement(player, input)
		}

		// Should reach zero or very close
		assert.InDelta(t, 0.0, player.VX, 1.0)
	})
}
