package system

import (
	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

// PhysicsSystem handles physics simulation with Intent & Apply model
type PhysicsSystem struct {
	config *config.PhysicsConfig
	stage  *entity.Stage
}

// NewPhysicsSystem creates a new physics system
func NewPhysicsSystem(cfg *config.PhysicsConfig, stage *entity.Stage) *PhysicsSystem {
	return &PhysicsSystem{
		config: cfg,
		stage:  stage,
	}
}

// Update applies physics to the player
func (s *PhysicsSystem) Update(player *entity.Player, dt float64) {
	// Store previous ground state for coyote time
	player.WasOnGround = player.OnGround

	// Apply gravity
	s.applyGravity(player, dt)

	// Get pixels to move this frame
	dx, dy := player.ApplyVelocity(dt)

	// Apply movement with substeps
	s.applyMovement(player, dx, dy)

	// Update facing direction based on velocity
	if player.VX > 0 {
		player.FacingRight = true
	} else if player.VX < 0 {
		player.FacingRight = false
	}
}

// applyGravity applies gravity acceleration to the player
func (s *PhysicsSystem) applyGravity(player *entity.Player, dt float64) {
	if player.Dashing {
		return // No gravity during dash
	}

	gravity := s.config.Physics.Gravity

	// Apply apex modifier (reduced gravity at jump peak)
	if s.config.Jump.ApexModifier.Enabled {
		if absFloat(player.VY) < s.config.Jump.ApexModifier.Threshold {
			gravity *= s.config.Jump.ApexModifier.GravityMultiplier
		}
	}

	// Apply fall multiplier (faster falling)
	if player.VY > 0 {
		gravity *= s.config.Jump.FallMultiplier
	}

	player.VY += gravity * dt

	// Clamp to max fall speed
	if player.VY > s.config.Physics.MaxFallSpeed {
		player.VY = s.config.Physics.MaxFallSpeed
	}
}

// applyMovement moves the player with substep collision detection
func (s *PhysicsSystem) applyMovement(player *entity.Player, dx, dy int) {
	// Reset collision flags
	player.OnGround = false
	player.OnCeiling = false
	player.OnWallLeft = false
	player.OnWallRight = false

	// First, resolve any existing overlaps (push-out)
	s.resolveOverlap(player)

	// Move X axis (1 pixel substeps)
	s.moveX(player, dx)

	// Move Y axis (1 pixel substeps)
	s.moveY(player, dy)

	// Final overlap resolution after movement
	s.resolveOverlap(player)
}

// moveX moves player horizontally with collision
func (s *PhysicsSystem) moveX(player *entity.Player, dx int) {
	if dx == 0 {
		return
	}

	step := sign(dx)
	for i := 0; i < abs(dx); i++ {
		if s.checkCollisionX(player, step) {
			// Hit wall
			player.VX = 0
			player.RemX = 0
			if step > 0 {
				player.OnWallRight = true
			} else {
				player.OnWallLeft = true
			}
			return
		}
		player.X += step
	}
}

// moveY moves player vertically with collision
func (s *PhysicsSystem) moveY(player *entity.Player, dy int) {
	if dy == 0 {
		return
	}

	step := sign(dy)
	for i := 0; i < abs(dy); i++ {
		if s.checkCollisionY(player, step) {
			player.VY = 0
			player.RemY = 0
			if step > 0 {
				// Hit ground
				player.OnGround = true
			} else {
				// Hit ceiling
				player.OnCeiling = true
				// Try corner correction
				s.tryCornerCorrection(player)
			}
			return
		}
		player.Y += step
	}
}

// checkCollisionX checks for horizontal collision using body hitbox
func (s *PhysicsSystem) checkCollisionX(player *entity.Player, dx int) bool {
	// Use body hitbox for horizontal collision
	hb := player.Hitbox.Body
	x, y, w, h := hb.GetWorldRect(player.X+dx, player.Y, player.FacingRight, 16)

	// Check all corners
	return s.isSolidRect(x, y, w, h)
}

// checkCollisionY checks for vertical collision
func (s *PhysicsSystem) checkCollisionY(player *entity.Player, dy int) bool {
	if dy > 0 {
		// Moving down - use feet hitbox (wider, more forgiving)
		hb := player.Hitbox.Feet
		x, y, w, h := hb.GetWorldRect(player.X, player.Y+dy, player.FacingRight, 16)
		return s.isSolidRect(x, y, w, h)
	}

	// Moving up - use head hitbox (narrower, more forgiving)
	hb := player.Hitbox.Head
	x, y, w, h := hb.GetWorldRect(player.X, player.Y+dy, player.FacingRight, 16)
	return s.isSolidRect(x, y, w, h)
}

// tryCornerCorrection attempts to nudge player around corners
func (s *PhysicsSystem) tryCornerCorrection(player *entity.Player) {
	if !s.config.Collision.CornerCorrection.Enabled {
		return
	}

	margin := s.config.Collision.CornerCorrection.Margin

	// Try nudging left
	for i := 1; i <= margin; i++ {
		if !s.checkCollisionY(player, -1) {
			return // Already clear
		}
		testX := player.X - i
		if !s.checkCollisionYAt(player, testX, -1) {
			player.X = testX
			return
		}
	}

	// Try nudging right
	for i := 1; i <= margin; i++ {
		testX := player.X + i
		if !s.checkCollisionYAt(player, testX, -1) {
			player.X = testX
			return
		}
	}
}

// checkCollisionYAt checks vertical collision at a specific X position
func (s *PhysicsSystem) checkCollisionYAt(player *entity.Player, atX int, dy int) bool {
	hb := player.Hitbox.Head
	x, y, w, h := hb.GetWorldRect(atX, player.Y+dy, player.FacingRight, 16)
	return s.isSolidRect(x, y, w, h)
}

// resolveOverlap pushes player out of any solid tiles they're currently overlapping
// Returns true if overlap was resolved, false if player is stuck
func (s *PhysicsSystem) resolveOverlap(player *entity.Player) bool {
	const maxPushOut = 8 // Maximum pixels to push out per axis

	// Check if currently overlapping using body hitbox
	hb := player.Hitbox.Body
	x, y, w, h := hb.GetWorldRect(player.X, player.Y, player.FacingRight, 16)

	if !s.isSolidRect(x, y, w, h) {
		return true // No overlap
	}

	// Try pushing in each direction to find the smallest push-out
	// Check all 4 directions and pick the smallest displacement

	type pushOption struct {
		dx, dy   int
		distance int
	}
	var options []pushOption

	// Try pushing left
	for i := 1; i <= maxPushOut; i++ {
		tx, ty, tw, th := hb.GetWorldRect(player.X-i, player.Y, player.FacingRight, 16)
		if !s.isSolidRect(tx, ty, tw, th) {
			options = append(options, pushOption{-i, 0, i})
			break
		}
	}

	// Try pushing right
	for i := 1; i <= maxPushOut; i++ {
		tx, ty, tw, th := hb.GetWorldRect(player.X+i, player.Y, player.FacingRight, 16)
		if !s.isSolidRect(tx, ty, tw, th) {
			options = append(options, pushOption{i, 0, i})
			break
		}
	}

	// Try pushing up
	for i := 1; i <= maxPushOut; i++ {
		tx, ty, tw, th := hb.GetWorldRect(player.X, player.Y-i, player.FacingRight, 16)
		if !s.isSolidRect(tx, ty, tw, th) {
			options = append(options, pushOption{0, -i, i})
			break
		}
	}

	// Try pushing down
	for i := 1; i <= maxPushOut; i++ {
		tx, ty, tw, th := hb.GetWorldRect(player.X, player.Y+i, player.FacingRight, 16)
		if !s.isSolidRect(tx, ty, tw, th) {
			options = append(options, pushOption{0, i, i})
			break
		}
	}

	// Pick the smallest push-out
	if len(options) == 0 {
		// Can't resolve - reset player to spawn position
		player.X = s.stage.SpawnX
		player.Y = s.stage.SpawnY
		player.VX = 0
		player.VY = 0
		return false
	}

	best := options[0]
	for _, opt := range options[1:] {
		if opt.distance < best.distance {
			best = opt
		}
	}

	player.X += best.dx
	player.Y += best.dy

	// Set collision flags based on push direction
	if best.dx > 0 {
		player.OnWallLeft = true
		player.VX = 0
	} else if best.dx < 0 {
		player.OnWallRight = true
		player.VX = 0
	}
	if best.dy > 0 {
		player.OnCeiling = true
		player.VY = 0
	} else if best.dy < 0 {
		player.OnGround = true
		player.VY = 0
	}
	return true
}

// isSolidRect checks if any tile in the rect is solid
// Iterates all tiles the rectangle overlaps to handle any hitbox size
func (s *PhysicsSystem) isSolidRect(x, y, w, h int) bool {
	tileSize := s.stage.TileSize
	if tileSize <= 0 {
		tileSize = 16 // fallback
	}

	// Calculate tile range that the rect overlaps
	startTX := x / tileSize
	endTX := (x + w - 1) / tileSize
	startTY := y / tileSize
	endTY := (y + h - 1) / tileSize

	// Check all tiles in the range
	for ty := startTY; ty <= endTY; ty++ {
		for tx := startTX; tx <= endTX; tx++ {
			if s.stage.GetTile(tx, ty).Solid {
				return true
			}
		}
	}

	return false
}

// Helper functions
func sign(x int) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
