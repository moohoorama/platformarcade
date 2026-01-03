package system

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

// InputSystem handles player input
type InputSystem struct {
	config *config.PhysicsConfig
}

// NewInputSystem creates a new input system
func NewInputSystem(cfg *config.PhysicsConfig) *InputSystem {
	return &InputSystem{config: cfg}
}

// InputState holds the current input state
type InputState struct {
	Left         bool
	Right        bool
	Up           bool
	Down         bool
	Jump         bool
	JumpPressed  bool
	JumpReleased bool
	Attack       bool
	Dash         bool
	MouseX       int
	MouseY       int
	MouseClick   bool
	// Right click for arrow selection
	RightClickPressed  bool
	RightClickReleased bool
}

// GetInput reads the current input state
func (s *InputSystem) GetInput() InputState {
	mx, my := ebiten.CursorPosition()
	return InputState{
		Left:               ebiten.IsKeyPressed(ebiten.KeyA),
		Right:              ebiten.IsKeyPressed(ebiten.KeyD),
		Up:                 ebiten.IsKeyPressed(ebiten.KeyW),
		Down:               ebiten.IsKeyPressed(ebiten.KeyS),
		Jump:               ebiten.IsKeyPressed(ebiten.KeyW),
		JumpPressed:        inpututil.IsKeyJustPressed(ebiten.KeyW),
		JumpReleased:       inpututil.IsKeyJustReleased(ebiten.KeyW),
		Attack:             false,
		Dash:               inpututil.IsKeyJustPressed(ebiten.KeySpace),
		MouseX:             mx,
		MouseY:             my,
		MouseClick:         inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		RightClickPressed:  inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
		RightClickReleased: inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight),
	}
}

// UpdatePlayer updates the player based on input
func (s *InputSystem) UpdatePlayer(player *entity.Player, input InputState, dt float64) {
	// Update timers
	s.updateTimers(player, dt)

	// Skip input if stunned
	if player.IsStunned() {
		// Apply friction while stunned
		player.VX *= 0.9
		return
	}

	// Skip movement input if dashing
	if player.Dashing {
		return
	}

	// Horizontal movement
	s.handleMovement(player, input)

	// Jump
	s.handleJump(player, input)

	// Dash
	s.handleDash(player, input)
}

// updateTimers updates various player timers
func (s *InputSystem) updateTimers(player *entity.Player, dt float64) {
	// Coyote time
	if player.OnGround {
		player.CoyoteTimer = s.config.Jump.CoyoteTime
	} else if player.CoyoteTimer > 0 {
		player.CoyoteTimer -= dt
	}

	// Jump buffer
	if player.JumpBufferTimer > 0 {
		player.JumpBufferTimer -= dt
	}

	// Dash
	if player.DashTimer > 0 {
		player.DashTimer -= dt
		if player.DashTimer <= 0 {
			player.Dashing = false
		}
	}
	if player.DashCooldown > 0 {
		player.DashCooldown -= dt
	}

	// Iframes
	if player.IframeTimer > 0 {
		player.IframeTimer -= dt
	}

	// Stun
	if player.StunTimer > 0 {
		player.StunTimer -= dt
	}

	// Reset dash on ground
	if player.OnGround {
		player.CanDash = true
	}
}

// handleMovement handles horizontal movement
func (s *InputSystem) handleMovement(player *entity.Player, input InputState) {
	targetVX := 0.0
	maxSpeed := s.config.Movement.MaxSpeed

	if input.Left {
		targetVX = -maxSpeed
		player.FacingRight = false
	}
	if input.Right {
		targetVX = maxSpeed
		player.FacingRight = true
	}

	// Air control
	if !player.OnGround {
		targetVX *= s.config.Movement.AirControl
	}

	// Acceleration/Deceleration
	if targetVX != 0 {
		accel := s.config.Movement.Acceleration

		// Turnaround boost
		if (player.VX > 0 && targetVX < 0) || (player.VX < 0 && targetVX > 0) {
			accel *= s.config.Movement.TurnaroundBoost
		}

		// Approach target velocity
		if player.VX < targetVX {
			player.VX += accel * (1.0 / 60.0) // Normalized to 60fps
			if player.VX > targetVX {
				player.VX = targetVX
			}
		} else if player.VX > targetVX {
			player.VX -= accel * (1.0 / 60.0)
			if player.VX < targetVX {
				player.VX = targetVX
			}
		}
	} else {
		// Deceleration
		decel := s.config.Movement.Deceleration * (1.0 / 60.0)
		if player.VX > 0 {
			player.VX -= decel
			if player.VX < 0 {
				player.VX = 0
			}
		} else if player.VX < 0 {
			player.VX += decel
			if player.VX > 0 {
				player.VX = 0
			}
		}
	}
}

// handleJump handles jumping
func (s *InputSystem) handleJump(player *entity.Player, input InputState) {
	// Buffer jump input
	if input.JumpPressed {
		player.JumpBufferTimer = s.config.Jump.JumpBuffer
	}

	// Can jump if on ground or has coyote time
	canJump := player.OnGround || player.CoyoteTimer > 0
	wantsJump := player.JumpBufferTimer > 0

	if canJump && wantsJump {
		player.VY = -s.config.Jump.Force
		player.OnGround = false
		player.CoyoteTimer = 0
		player.JumpBufferTimer = 0
	}

	// Variable jump height (release to reduce upward velocity)
	if input.JumpReleased && player.VY < 0 {
		player.VY *= s.config.Jump.VariableJumpMultiplier
	}
}

// handleDash handles dashing
func (s *InputSystem) handleDash(player *entity.Player, input InputState) {
	if !input.Dash || !player.CanDash || player.DashCooldown > 0 {
		return
	}

	// Start dash
	player.Dashing = true
	player.DashTimer = s.config.Dash.Duration
	player.DashCooldown = s.config.Dash.Cooldown
	player.CanDash = false
	player.IframeTimer = s.config.Dash.IframesDuration

	// Set dash velocity
	dir := 1.0
	if !player.FacingRight {
		dir = -1.0
	}
	player.VX = dir * s.config.Dash.Speed
	player.VY = 0
}
