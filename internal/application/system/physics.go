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

// Update applies physics to the player with sub-step support.
// subSteps controls the number of physics iterations per frame:
// - 10 = normal speed (full frame processed)
// - 1 = slow motion (1/10 speed)
func (s *PhysicsSystem) Update(player *entity.Player, dt float64, subSteps int) {
	// Store previous ground state for coyote time
	player.WasOnGround = player.OnGround

	// Each sub-step processes 1/10 of a frame
	dtPerStep := dt / 10.0

	for i := 0; i < subSteps; i++ {
		s.updateStep(player, dtPerStep)
	}

	// Update facing direction based on velocity
	if player.VX > 0 {
		player.FacingRight = true
	} else if player.VX < 0 {
		player.FacingRight = false
	}
}

// updateStep performs a single physics sub-step
func (s *PhysicsSystem) updateStep(player *entity.Player, dt float64) {
	// Apply gravity
	s.applyGravity(player, dt)

	// Get units to move this sub-step (100x scaled)
	dx, dy := player.ApplyVelocity(dt)

	// Apply movement with collision detection
	s.applyMovement(player, dx, dy)
}

// applyGravity applies gravity acceleration to the player
// Note: VY is in 100x scaled units, config values are in pixels
func (s *PhysicsSystem) applyGravity(player *entity.Player, dt float64) {
	if player.Dashing {
		return // No gravity during dash
	}

	// Don't apply gravity when on ground and not moving up
	// (ground's normal force balances gravity)
	if player.OnGround && player.VY >= 0 {
		return
	}

	// Config gravity is in pixels/sec², convert to 100x units
	gravity := s.config.Physics.Gravity * entity.PositionScale

	// Apply apex modifier (reduced gravity at jump peak)
	// Threshold is in pixels/sec, convert to 100x units
	if s.config.Jump.ApexModifier.Enabled {
		thresholdScaled := s.config.Jump.ApexModifier.Threshold * entity.PositionScale
		if absFloat(player.VY) < thresholdScaled {
			gravity *= s.config.Jump.ApexModifier.GravityMultiplier
		}
	}

	// Apply fall multiplier (faster falling)
	if player.VY > 0 {
		gravity *= s.config.Jump.FallMultiplier
	}

	player.VY += gravity * dt

	// Clamp to max fall speed (convert to 100x units)
	maxFallSpeedScaled := s.config.Physics.MaxFallSpeed * entity.PositionScale
	if player.VY > maxFallSpeedScaled {
		player.VY = maxFallSpeedScaled
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

	// If no vertical movement, check if we're standing on ground
	// This prevents OnGround from becoming false when player is idle
	if dy == 0 {
		s.checkGroundContact(player)
	}

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

// checkGroundContact checks if the player's feet are touching ground
// Called when there's no vertical movement to maintain OnGround state
func (s *PhysicsSystem) checkGroundContact(player *entity.Player) {
	// Check if moving down by 1 unit would cause collision (feet touching ground)
	if s.checkCollisionY(player, entity.PositionScale) {
		player.OnGround = true
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
// dx is in 100x scaled units
func (s *PhysicsSystem) checkCollisionX(player *entity.Player, dx int) bool {
	// Convert to pixel coordinates for hitbox calculation
	pixelX := (player.X + dx) / entity.PositionScale
	pixelY := player.Y / entity.PositionScale

	// Use body hitbox for horizontal collision
	hb := player.Hitbox.Body
	x, y, w, h := hb.GetWorldRect(pixelX, pixelY, player.FacingRight, 16)

	// Check collision in pixel space
	return s.isSolidRect(x, y, w, h)
}

// checkCollisionY checks for vertical collision
// dy is in 100x scaled units
func (s *PhysicsSystem) checkCollisionY(player *entity.Player, dy int) bool {
	// Convert to pixel coordinates for hitbox calculation
	pixelX := player.X / entity.PositionScale
	pixelY := (player.Y + dy) / entity.PositionScale

	if dy > 0 {
		// Moving down - use feet hitbox (wider, more forgiving)
		hb := player.Hitbox.Feet
		x, y, w, h := hb.GetWorldRect(pixelX, pixelY, player.FacingRight, 16)
		return s.isSolidRect(x, y, w, h)
	}

	// Moving up - use head hitbox (narrower, more forgiving)
	hb := player.Hitbox.Head
	x, y, w, h := hb.GetWorldRect(pixelX, pixelY, player.FacingRight, 16)
	return s.isSolidRect(x, y, w, h)
}

// tryCornerCorrection attempts to nudge player around corners
// Works in 100x scaled units
func (s *PhysicsSystem) tryCornerCorrection(player *entity.Player) {
	if !s.config.Collision.CornerCorrection.Enabled {
		return
	}

	// Config margin is in pixels, convert to 100x units
	margin := s.config.Collision.CornerCorrection.Margin * entity.PositionScale

	// Try nudging left
	for i := entity.PositionScale; i <= margin; i += entity.PositionScale {
		if !s.checkCollisionY(player, -entity.PositionScale) {
			return // Already clear
		}
		testX := player.X - i
		if !s.checkCollisionYAt(player, testX, -entity.PositionScale) {
			player.X = testX
			return
		}
	}

	// Try nudging right
	for i := entity.PositionScale; i <= margin; i += entity.PositionScale {
		testX := player.X + i
		if !s.checkCollisionYAt(player, testX, -entity.PositionScale) {
			player.X = testX
			return
		}
	}
}

// checkCollisionYAt checks vertical collision at a specific X position
// atX is in 100x scaled units, dy is in 100x scaled units
func (s *PhysicsSystem) checkCollisionYAt(player *entity.Player, atX int, dy int) bool {
	// Convert to pixel coordinates
	pixelX := atX / entity.PositionScale
	pixelY := (player.Y + dy) / entity.PositionScale

	hb := player.Hitbox.Head
	x, y, w, h := hb.GetWorldRect(pixelX, pixelY, player.FacingRight, 16)
	return s.isSolidRect(x, y, w, h)
}

// resolveOverlap pushes player out of any solid tiles they're currently overlapping
// Returns true if overlap was resolved, false if player is stuck
// Works in 100x scaled units
func (s *PhysicsSystem) resolveOverlap(player *entity.Player) bool {
	const maxPushOutPixels = 8 // Maximum pixels to push out per axis
	maxPushOut := maxPushOutPixels * entity.PositionScale

	// Check if currently overlapping using body hitbox (convert to pixels for check)
	hb := player.Hitbox.Body
	pixelX := player.X / entity.PositionScale
	pixelY := player.Y / entity.PositionScale
	x, y, w, h := hb.GetWorldRect(pixelX, pixelY, player.FacingRight, 16)

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
	step := entity.PositionScale // Check 1 pixel at a time

	// Try pushing left
	for i := step; i <= maxPushOut; i += step {
		testPixelX := (player.X - i) / entity.PositionScale
		tx, ty, tw, th := hb.GetWorldRect(testPixelX, pixelY, player.FacingRight, 16)
		if !s.isSolidRect(tx, ty, tw, th) {
			options = append(options, pushOption{-i, 0, i})
			break
		}
	}

	// Try pushing right
	for i := step; i <= maxPushOut; i += step {
		testPixelX := (player.X + i) / entity.PositionScale
		tx, ty, tw, th := hb.GetWorldRect(testPixelX, pixelY, player.FacingRight, 16)
		if !s.isSolidRect(tx, ty, tw, th) {
			options = append(options, pushOption{i, 0, i})
			break
		}
	}

	// Try pushing up
	for i := step; i <= maxPushOut; i += step {
		testPixelY := (player.Y - i) / entity.PositionScale
		tx, ty, tw, th := hb.GetWorldRect(pixelX, testPixelY, player.FacingRight, 16)
		if !s.isSolidRect(tx, ty, tw, th) {
			options = append(options, pushOption{0, -i, i})
			break
		}
	}

	// Try pushing down
	for i := step; i <= maxPushOut; i += step {
		testPixelY := (player.Y + i) / entity.PositionScale
		tx, ty, tw, th := hb.GetWorldRect(pixelX, testPixelY, player.FacingRight, 16)
		if !s.isSolidRect(tx, ty, tw, th) {
			options = append(options, pushOption{0, i, i})
			break
		}
	}

	// Pick the smallest push-out
	if len(options) == 0 {
		// Can't resolve - reset player to spawn position (convert spawn to 100x)
		player.X = s.stage.SpawnX * entity.PositionScale
		player.Y = s.stage.SpawnY * entity.PositionScale
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
// x, y, w, h are in PIXEL coordinates
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
