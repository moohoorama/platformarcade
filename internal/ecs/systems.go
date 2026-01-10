package ecs

import (
	"math"
)

// Stage interface for collision detection
type Stage interface {
	IsSolidAt(px, py int) bool
	GetTileType(px, py int) int
	GetTileDamage(px, py int) int
	GetWidth() int
	GetHeight() int
	GetTileSize() int
	GetSpawnX() int
	GetSpawnY() int
}

const (
	TileEmpty = 0
	TileWall  = 1
	TileSpike = 2
)

// ToIUPerSubstep converts pixels/sec to IU/substep.
// Formula: pixels_per_sec * PositionScale / 60 / 10
// = pixels_per_sec * 256 / 600
func ToIUPerSubstep(pixelsPerSec float64) int {
	return int(pixelsPerSec * float64(PositionScale) / 600.0)
}

// ToIUAccelPerFrame converts pixels/sec² to IU velocity change per frame.
// Acceleration: velocity changes by (accel / 60) pixels/sec per frame.
// Convert to IU/substep: * 256 / 600
// Combined: pixels_per_sec_sq * 256 / 36000
func ToIUAccelPerFrame(pixelsPerSecSq float64) int {
	return int(pixelsPerSecSq * float64(PositionScale) / 36000.0)
}

// PctToInt converts a 0.0-1.0+ float to 0-100+ percentage int.
func PctToInt(f float64) int {
	return int(f * 100)
}

// PhysicsConfig holds physics configuration.
// All velocity/acceleration values are in IU (internal units) per substep.
// Conversion: pixels_per_sec * PositionScale / 600
type PhysicsConfig struct {
	// Physics (IU per substep)
	Gravity      int // IU/substep²
	MaxFallSpeed int // IU/substep

	// Movement (IU per substep)
	MaxSpeed        int // IU/substep
	Acceleration    int // IU/substep²
	Deceleration    int // IU/substep²
	AirControlPct   int // 0-100 (percentage)
	TurnaroundPct   int // 0-100 (percentage, 100 = no boost)

	// Jump
	JumpForce         int // IU/substep (initial upward velocity)
	VarJumpPct        int // 0-100 (percentage of jump force when released early)
	CoyoteFrames      int
	JumpBufferFrames  int
	ApexModEnabled    bool
	ApexThreshold     int // IU/substep (velocity threshold for apex modifier)
	ApexGravityPct    int // 0-100 (percentage of gravity at apex)
	FallMultiplierPct int // 100 = normal, 160 = 1.6x faster fall

	// Dash
	DashSpeed          int // IU/substep
	DashFrames         int
	DashCooldownFrames int
	DashIframes        int

	// Collision
	CornerCorrectionMargin  int
	CornerCorrectionEnabled bool

	// Knockback
	KnockbackDecay int // IU/frame linear deceleration during stun
}

// UpdateTimers decrements all frame-based timers
func UpdateTimers(w *World) {
	// Player timers
	for id := range w.IsPlayer {
		player := w.PlayerData[id]
		if player.CoyoteTimer > 0 {
			player.CoyoteTimer--
		}
		if player.JumpBufferTimer > 0 {
			player.JumpBufferTimer--
		}
		if player.IframeTimer > 0 {
			player.IframeTimer--
		}
		if player.StunTimer > 0 {
			player.StunTimer--
		}
		w.PlayerData[id] = player

		dash := w.Dash[id]
		if dash.Timer > 0 {
			dash.Timer--
			if dash.Timer == 0 {
				dash.Active = false
			}
		}
		if dash.Cooldown > 0 {
			dash.Cooldown--
		}
		w.Dash[id] = dash

		// Reset dash on ground
		mov := w.Movement[id]
		if mov.OnGround {
			dash.CanDash = true
			w.Dash[id] = dash
		}
	}

	// Enemy AI timers and knockback deceleration
	for id := range w.IsEnemy {
		ai := w.AI[id]
		if ai.HitTimer > 0 {
			ai.HitTimer--

			// Calculate velocity proportional to remaining HitTimer
			// This ensures velocity reaches 0 exactly when HitTimer reaches 0
			vel := w.Velocity[id]
			if ai.HitTimerMax > 0 {
				// vel = initialVel * (remainingTimer / maxTimer)
				vel.X = ai.KnockbackVelX * ai.HitTimer / ai.HitTimerMax
				vel.Y = ai.KnockbackVelY * ai.HitTimer / ai.HitTimerMax
			} else {
				vel.X = 0
				vel.Y = 0
			}
			w.Velocity[id] = vel
		}
		if ai.AttackTimer > 0 {
			ai.AttackTimer--
		}
		w.AI[id] = ai
	}

	// Projectile stuck timers
	toDestroy := make([]EntityID, 0)
	for id := range w.IsProjectile {
		proj := w.ProjectileData[id]
		if proj.Stuck {
			proj.StuckTimer++
			if proj.StuckTimer >= proj.StuckDuration {
				toDestroy = append(toDestroy, id)
				continue
			}
			w.ProjectileData[id] = proj
		}
	}
	for _, id := range toDestroy {
		w.DestroyEntity(id)
	}

	// Gold collect delay
	for id := range w.IsGold {
		gold := w.GoldData[id]
		if gold.CollectDelay > 0 {
			gold.CollectDelay--
			w.GoldData[id] = gold
		}
	}
}

// InputState holds input for the current frame
type InputState struct {
	Left, Right, Up, Down bool
	JumpPressed           bool
	JumpReleased          bool
	Dash                  bool
}

// UpdatePlayerInput processes player input
// All values are integers in IU/substep units
func UpdatePlayerInput(w *World, input InputState, cfg PhysicsConfig) {
	id := w.PlayerID
	if id == 0 {
		return
	}

	player := w.PlayerData[id]
	dash := w.Dash[id]
	mov := w.Movement[id]
	vel := w.Velocity[id]
	facing := w.Facing[id]

	// Skip if stunned (linear deceleration toward zero)
	if player.IsStunned() {
		decay := cfg.KnockbackDecay
		if decay == 0 {
			decay = 10 // default fallback
		}
		if vel.X > 0 {
			vel.X -= decay
			if vel.X < 0 {
				vel.X = 0
			}
		} else if vel.X < 0 {
			vel.X += decay
			if vel.X > 0 {
				vel.X = 0
			}
		}
		w.Velocity[id] = vel
		return
	}

	// Skip movement if dashing
	if dash.Active {
		return
	}

	// Coyote time
	if mov.OnGround {
		player.CoyoteTimer = cfg.CoyoteFrames
	}

	// Movement - MaxSpeed is already in IU/substep
	targetVX := 0
	maxSpeed := cfg.MaxSpeed

	if input.Left {
		targetVX = -maxSpeed
		facing.Right = false
	}
	if input.Right {
		targetVX = maxSpeed
		facing.Right = true
	}

	// Air control (percentage)
	if !mov.OnGround {
		targetVX = targetVX * cfg.AirControlPct / 100
	}

	// Acceleration/Deceleration
	if targetVX != 0 {
		accel := cfg.Acceleration
		// Turnaround boost (percentage)
		if (vel.X > 0 && targetVX < 0) || (vel.X < 0 && targetVX > 0) {
			accel = accel * cfg.TurnaroundPct / 100
		}
		// Approach target
		if vel.X < targetVX {
			vel.X += accel
			if vel.X > targetVX {
				vel.X = targetVX
			}
		} else if vel.X > targetVX {
			vel.X -= accel
			if vel.X < targetVX {
				vel.X = targetVX
			}
		}
	} else {
		// Deceleration
		decel := cfg.Deceleration
		if vel.X > 0 {
			vel.X -= decel
			if vel.X < 0 {
				vel.X = 0
			}
		} else if vel.X < 0 {
			vel.X += decel
			if vel.X > 0 {
				vel.X = 0
			}
		}
	}

	// Jump buffer
	if input.JumpPressed {
		player.JumpBufferTimer = cfg.JumpBufferFrames
	}

	// Jump - JumpForce is in IU/substep, negate for upward
	canJump := mov.OnGround || player.CoyoteTimer > 0
	wantsJump := player.JumpBufferTimer > 0
	if canJump && wantsJump {
		vel.Y = -cfg.JumpForce
		mov.OnGround = false
		player.CoyoteTimer = 0
		player.JumpBufferTimer = 0
	}

	// Variable jump height (percentage)
	if input.JumpReleased && vel.Y < 0 {
		vel.Y = vel.Y * cfg.VarJumpPct / 100
	}

	// Dash
	if input.Dash && dash.CanDash && dash.Cooldown <= 0 {
		dash.Active = true
		dash.Timer = cfg.DashFrames
		dash.Cooldown = cfg.DashCooldownFrames
		dash.CanDash = false
		player.IframeTimer = cfg.DashIframes

		dir := 1
		if !facing.Right {
			dir = -1
		}
		vel.X = dir * cfg.DashSpeed
		vel.Y = 0
	}

	w.PlayerData[id] = player
	w.Dash[id] = dash
	w.Movement[id] = mov
	w.Velocity[id] = vel
	w.Facing[id] = facing
}

// ApplyPlayerGravity applies gravity to player velocity (call once per frame)
// Gravity is in IU velocity change per frame.
func ApplyPlayerGravity(w *World, cfg PhysicsConfig) {
	id := w.PlayerID
	if id == 0 {
		return
	}

	vel := w.Velocity[id]
	mov := w.Movement[id]
	dash := w.Dash[id]

	if dash.Active || (mov.OnGround && vel.Y >= 0) {
		return
	}

	gravity := cfg.Gravity

	// Apex modifier (percentage)
	if cfg.ApexModEnabled {
		if abs(vel.Y) < cfg.ApexThreshold {
			gravity = gravity * cfg.ApexGravityPct / 100
		}
	}

	// Fall multiplier (percentage, 100 = normal)
	if vel.Y > 0 {
		gravity = gravity * cfg.FallMultiplierPct / 100
	}

	vel.Y += gravity
	w.Velocity[id] = vel
}

// UpdatePlayerPhysics updates player physics for 1 substep
// All values are integers. Velocity is in IU/substep.
// Call this function multiple times per frame for normal speed,
// or fewer times for slow motion.
func UpdatePlayerPhysics(w *World, stage Stage, cfg PhysicsConfig) {
	id := w.PlayerID
	if id == 0 {
		return
	}

	pos := w.Position[id]
	vel := w.Velocity[id]
	mov := w.Movement[id]
	hitbox := w.HitboxTrapezoid[id]
	facing := w.Facing[id]

	mov.WasOnGround = mov.OnGround

	{
		// NOTE: Gravity is applied separately via ApplyPlayerGravity (once per frame)

		// Clamp fall speed
		if vel.Y > cfg.MaxFallSpeed {
			vel.Y = cfg.MaxFallSpeed
		}

		// Position change = velocity (IU/substep)
		dx := vel.X
		dy := vel.Y

		// Reset collision flags
		mov.OnGround = false
		mov.OnCeiling = false
		mov.OnWallLeft = false
		mov.OnWallRight = false

		// Resolve overlaps first
		resolvePlayerOverlap(w, id, stage, &pos, &vel, &mov, hitbox, facing.Right)

		// Move X
		movePlayerX(stage, &pos, &vel, &mov, hitbox, facing.Right, dx)

		// Move Y
		movePlayerY(stage, &pos, &vel, &mov, hitbox, facing.Right, dy, cfg)

		// Check ground contact when not moving vertically
		if dy == 0 {
			if checkPlayerCollisionY(stage, pos, hitbox, facing.Right, 1) {
				mov.OnGround = true
			}
		}

		// Final overlap resolution
		resolvePlayerOverlap(w, id, stage, &pos, &vel, &mov, hitbox, facing.Right)
	}

	// Update facing based on velocity
	if vel.X > 0 {
		facing.Right = true
	} else if vel.X < 0 {
		facing.Right = false
	}

	w.Position[id] = pos
	w.Velocity[id] = vel
	w.Movement[id] = mov
	w.Facing[id] = facing
}

func movePlayerX(stage Stage, pos *Position, vel *Velocity, mov *Movement, hitbox HitboxTrapezoid, facingRight bool, dx int) {
	if dx == 0 {
		return
	}

	step := sign(dx)
	for i := 0; i < abs(dx); i++ {
		if checkPlayerCollisionX(stage, *pos, hitbox, facingRight, step) {
			vel.X = 0
			if step > 0 {
				mov.OnWallRight = true
			} else {
				mov.OnWallLeft = true
			}
			return
		}
		pos.X += step
	}
}

func movePlayerY(stage Stage, pos *Position, vel *Velocity, mov *Movement, hitbox HitboxTrapezoid, facingRight bool, dy int, cfg PhysicsConfig) {
	if dy == 0 {
		return
	}

	step := sign(dy)
	for i := 0; i < abs(dy); i++ {
		if checkPlayerCollisionY(stage, *pos, hitbox, facingRight, step) {
			vel.Y = 0
			if step > 0 {
				mov.OnGround = true
			} else {
				mov.OnCeiling = true
				// Corner correction
				if cfg.CornerCorrectionEnabled {
					tryCornerCorrection(stage, pos, hitbox, facingRight, cfg.CornerCorrectionMargin)
				}
			}
			return
		}
		pos.Y += step
	}
}

func checkPlayerCollisionX(stage Stage, pos Position, hitbox HitboxTrapezoid, facingRight bool, dx int) bool {
	pixelX := (pos.X + dx) / PositionScale
	pixelY := pos.Y / PositionScale
	hb := hitbox.Body
	x, y, w, h := hb.GetWorldRect(pixelX, pixelY, facingRight, 16)
	return isSolidRect(stage, x, y, w, h)
}

func checkPlayerCollisionY(stage Stage, pos Position, hitbox HitboxTrapezoid, facingRight bool, dy int) bool {
	pixelX := pos.X / PositionScale
	pixelY := (pos.Y + dy) / PositionScale

	var hb Hitbox
	if dy > 0 {
		hb = hitbox.Feet
	} else {
		hb = hitbox.Head
	}
	x, y, w, h := hb.GetWorldRect(pixelX, pixelY, facingRight, 16)
	return isSolidRect(stage, x, y, w, h)
}

func tryCornerCorrection(stage Stage, pos *Position, hitbox HitboxTrapezoid, facingRight bool, margin int) {
	marginScaled := margin * PositionScale
	pixelY := (pos.Y - PositionScale) / PositionScale
	hb := hitbox.Head

	// Try nudging left
	for i := PositionScale; i <= marginScaled; i += PositionScale {
		testPixelX := (pos.X - i) / PositionScale
		x, y, w, h := hb.GetWorldRect(testPixelX, pixelY, facingRight, 16)
		if !isSolidRect(stage, x, y, w, h) {
			pos.X -= i
			return
		}
	}

	// Try nudging right
	for i := PositionScale; i <= marginScaled; i += PositionScale {
		testPixelX := (pos.X + i) / PositionScale
		x, y, w, h := hb.GetWorldRect(testPixelX, pixelY, facingRight, 16)
		if !isSolidRect(stage, x, y, w, h) {
			pos.X += i
			return
		}
	}
}

func resolvePlayerOverlap(w *World, id EntityID, stage Stage, pos *Position, vel *Velocity, mov *Movement, hitbox HitboxTrapezoid, facingRight bool) {
	maxPushOut := 8 * PositionScale
	pixelX := pos.X / PositionScale
	pixelY := pos.Y / PositionScale
	hb := hitbox.Body
	x, y, ww, h := hb.GetWorldRect(pixelX, pixelY, facingRight, 16)

	if !isSolidRect(stage, x, y, ww, h) {
		return
	}

	type pushOption struct {
		dx, dy, dist int
	}
	var options []pushOption
	step := PositionScale

	// Try each direction
	for i := step; i <= maxPushOut; i += step {
		// Left
		testPX := (pos.X - i) / PositionScale
		tx, ty, tw, th := hb.GetWorldRect(testPX, pixelY, facingRight, 16)
		if !isSolidRect(stage, tx, ty, tw, th) {
			options = append(options, pushOption{-i, 0, i})
			break
		}
	}
	for i := step; i <= maxPushOut; i += step {
		// Right
		testPX := (pos.X + i) / PositionScale
		tx, ty, tw, th := hb.GetWorldRect(testPX, pixelY, facingRight, 16)
		if !isSolidRect(stage, tx, ty, tw, th) {
			options = append(options, pushOption{i, 0, i})
			break
		}
	}
	for i := step; i <= maxPushOut; i += step {
		// Up
		testPY := (pos.Y - i) / PositionScale
		tx, ty, tw, th := hb.GetWorldRect(pixelX, testPY, facingRight, 16)
		if !isSolidRect(stage, tx, ty, tw, th) {
			options = append(options, pushOption{0, -i, i})
			break
		}
	}
	for i := step; i <= maxPushOut; i += step {
		// Down
		testPY := (pos.Y + i) / PositionScale
		tx, ty, tw, th := hb.GetWorldRect(pixelX, testPY, facingRight, 16)
		if !isSolidRect(stage, tx, ty, tw, th) {
			options = append(options, pushOption{0, i, i})
			break
		}
	}

	if len(options) == 0 {
		return
	}

	best := options[0]
	for _, opt := range options[1:] {
		if opt.dist < best.dist {
			best = opt
		}
	}

	pos.X += best.dx
	pos.Y += best.dy

	if best.dx > 0 {
		mov.OnWallLeft = true
		vel.X = 0
	} else if best.dx < 0 {
		mov.OnWallRight = true
		vel.X = 0
	}
	if best.dy > 0 {
		mov.OnCeiling = true
		vel.Y = 0
	} else if best.dy < 0 {
		mov.OnGround = true
		vel.Y = 0
	}
}

func isSolidRect(stage Stage, x, y, w, h int) bool {
	tileSize := 16 // TODO: get from stage
	startTX := x / tileSize
	endTX := (x + w - 1) / tileSize
	startTY := y / tileSize
	endTY := (y + h - 1) / tileSize

	for ty := startTY; ty <= endTY; ty++ {
		for tx := startTX; tx <= endTX; tx++ {
			if stage.IsSolidAt(tx*tileSize, ty*tileSize) {
				return true
			}
		}
	}
	return false
}

// UpdateEnemyAI updates enemy AI behavior for one substep
// Gravity is applied separately via ApplyEnemyGravity (once per frame)
func UpdateEnemyAI(w *World, stage Stage, arrowCfg ProjectileConfig, cfg PhysicsConfig) {
	playerPos := w.GetPlayerPosition()
	playerPX, playerPY := playerPos.PixelX(), playerPos.PixelY()

	for id := range w.IsEnemy {
		pos := w.Position[id]
		vel := w.Velocity[id]
		ai := w.AI[id]
		facing := w.Facing[id]
		mov := w.Movement[id]

		// If hit stunned, apply knockback movement (no AI control)
		// Note: deceleration is applied in UpdateTimers (once per frame)
		if ai.HitTimer > 0 {
			// Apply knockback movement (both X and Y)
			moveEnemyKnockbackX(stage, &pos, &vel, vel.X)
			if !ai.Flying {
				moveEnemyY(stage, &pos, &vel, &mov, vel.Y)
			}
			w.Position[id] = pos
			w.Velocity[id] = vel
			w.Movement[id] = mov
			continue
		}

		px, py := pos.PixelX(), pos.PixelY()
		dx := playerPX - px
		dy := playerPY - py
		// Approximate distance using taxicab metric for int
		dist := abs(dx) + abs(dy)

		switch ai.Type {
		case AIPatrol:
			updatePatrolAI(stage, &pos, &vel, &ai, &facing, &mov)
		case AIAggressive:
			updateAggressiveAI(w, stage, &pos, &vel, &ai, &facing, &mov, dx, dy, dist, arrowCfg)
		case AIRanged:
			updateRangedAI(w, stage, &pos, &vel, &ai, &facing, &mov, dx, dist, arrowCfg)
		case AIChase:
			updateChaseAI(stage, &pos, &vel, &ai, &facing, &mov, dx, dy, dist)
		}

		w.Position[id] = pos
		w.Velocity[id] = vel
		w.AI[id] = ai
		w.Facing[id] = facing
		w.Movement[id] = mov
	}
}

func updatePatrolAI(stage Stage, pos *Position, vel *Velocity, ai *AI, facing *Facing, mov *Movement) {
	// Move using AI's MoveSpeed (already in IU/substep)
	moveX := ai.PatrolDir * ai.MoveSpeed
	moveEnemyX(stage, pos, vel, ai, facing, mov, moveX)

	// Turn at patrol bounds
	px := pos.PixelX()
	if ai.PatrolDir > 0 && px >= ai.PatrolStartX+ai.PatrolDistance {
		ai.PatrolDir = -1
		facing.Right = false
	} else if ai.PatrolDir < 0 && px <= ai.PatrolStartX-ai.PatrolDistance {
		ai.PatrolDir = 1
		facing.Right = true
	}

	// Apply Y movement from velocity (gravity is applied separately per frame)
	if !ai.Flying {
		moveEnemyY(stage, pos, vel, mov, vel.Y)
	}
}

func updateAggressiveAI(w *World, stage Stage, pos *Position, vel *Velocity, ai *AI, facing *Facing, mov *Movement, dx, dy, dist int, arrowCfg ProjectileConfig) {
	// Apply Y movement from velocity (gravity is applied separately per frame)
	moveEnemyY(stage, pos, vel, mov, vel.Y)

	// Face player
	facing.Right = dx > 0

	// Charge toward player using MoveSpeed (IU/substep)
	if dx > 0 {
		moveEnemyX(stage, pos, vel, ai, facing, mov, ai.MoveSpeed)
	} else if dx < 0 {
		moveEnemyX(stage, pos, vel, ai, facing, mov, -ai.MoveSpeed)
	}

	// Jump if player above
	playerAbove := dy < -20
	if playerAbove && mov.OnGround && ai.JumpForce > 0 {
		vel.Y = -ai.JumpForce
		mov.OnGround = false
	}

	// Shoot
	if dist < ai.AttackRange && ai.AttackTimer <= 0 {
		spawnEnemyArrow(w, pos, facing.Right, arrowCfg)
		ai.AttackTimer = 90 // 1.5 seconds at 60fps
	}
}

func updateRangedAI(w *World, stage Stage, pos *Position, vel *Velocity, ai *AI, facing *Facing, mov *Movement, dx, dist int, arrowCfg ProjectileConfig) {
	facing.Right = dx > 0

	// Apply Y movement from velocity (gravity is applied separately per frame)
	if !ai.Flying {
		moveEnemyY(stage, pos, vel, mov, vel.Y)
	}

	if dist < ai.AttackRange && ai.AttackTimer <= 0 {
		spawnEnemyArrow(w, pos, facing.Right, arrowCfg)
		ai.AttackTimer = 90
	}
}

func updateChaseAI(stage Stage, pos *Position, vel *Velocity, ai *AI, facing *Facing, mov *Movement, dx, dy, dist int) {
	// Apply Y movement from velocity (gravity is applied separately per frame)
	if !ai.Flying {
		moveEnemyY(stage, pos, vel, mov, vel.Y)
	}

	if dist > ai.DetectRange {
		return
	}

	if dx > 0 {
		moveEnemyX(stage, pos, vel, ai, facing, mov, ai.MoveSpeed)
		facing.Right = true
	} else if dx < 0 {
		moveEnemyX(stage, pos, vel, ai, facing, mov, -ai.MoveSpeed)
		facing.Right = false
	}

	if ai.Flying {
		if dy > 0 {
			moveEnemyY(stage, pos, vel, mov, ai.MoveSpeed)
		} else if dy < 0 {
			moveEnemyY(stage, pos, vel, mov, -ai.MoveSpeed)
		}
	}
}

func moveEnemyX(stage Stage, pos *Position, vel *Velocity, ai *AI, facing *Facing, mov *Movement, moveX int) {
	if moveX == 0 {
		return
	}

	step := sign(moveX)
	steps := abs(moveX)

	hitbox := Hitbox{OffsetX: 2, OffsetY: 4, Width: 12, Height: 20} // Default enemy hitbox

	// moveX is in IU, step 1 IU at a time
	for i := 0; i < steps; i++ {
		// Check collision at next pixel boundary
		nextPixelX := (pos.X + step) / PositionScale
		if nextPixelX != pos.PixelX() {
			// About to cross pixel boundary, check collision
			x := nextPixelX + hitbox.OffsetX
			y := pos.PixelY() + hitbox.OffsetY
			w := hitbox.Width
			h := hitbox.Height

			checkX := x
			if step > 0 {
				checkX = x + w - 1
			}

			if stage.IsSolidAt(checkX, y) || stage.IsSolidAt(checkX, y+h-1) || stage.IsSolidAt(checkX, y+h/2) {
				ai.PatrolDir *= -1
				facing.Right = ai.PatrolDir > 0
				return
			}
		}
		pos.X += step // 1 IU per step
	}
}

// moveEnemyKnockbackX moves enemy horizontally during knockback (no AI logic)
func moveEnemyKnockbackX(stage Stage, pos *Position, vel *Velocity, moveX int) {
	if moveX == 0 {
		return
	}

	step := sign(moveX)
	steps := abs(moveX)

	hitbox := Hitbox{OffsetX: 2, OffsetY: 4, Width: 12, Height: 20}

	for i := 0; i < steps; i++ {
		nextPixelX := (pos.X + step) / PositionScale
		if nextPixelX != pos.PixelX() {
			x := nextPixelX + hitbox.OffsetX
			y := pos.PixelY() + hitbox.OffsetY
			w := hitbox.Width
			h := hitbox.Height

			checkX := x
			if step > 0 {
				checkX = x + w - 1
			}

			if stage.IsSolidAt(checkX, y) || stage.IsSolidAt(checkX, y+h-1) || stage.IsSolidAt(checkX, y+h/2) {
				vel.X = 0
				return
			}
		}
		pos.X += step
	}
}

func moveEnemyY(stage Stage, pos *Position, vel *Velocity, mov *Movement, moveY int) {
	if moveY == 0 {
		return
	}

	step := sign(moveY)
	steps := abs(moveY)

	hitbox := Hitbox{OffsetX: 2, OffsetY: 4, Width: 12, Height: 20}

	// moveY is in IU, step 1 IU at a time
	for i := 0; i < steps; i++ {
		// Check collision at next pixel boundary
		nextPixelY := (pos.Y + step) / PositionScale
		if nextPixelY != pos.PixelY() {
			// About to cross pixel boundary, check collision
			x := pos.PixelX() + hitbox.OffsetX
			y := nextPixelY + hitbox.OffsetY
			w := hitbox.Width
			h := hitbox.Height

			checkY := y
			if step > 0 {
				checkY = y + h - 1
			}

			if stage.IsSolidAt(x, checkY) || stage.IsSolidAt(x+w-1, checkY) || stage.IsSolidAt(x+w/2, checkY) {
				if step > 0 {
					mov.OnGround = true
				}
				vel.Y = 0
				return
			}
		}
		pos.Y += step // 1 IU per step
		mov.OnGround = false
	}
}

// ApplyEnemyGravity applies gravity to all enemies (call once per frame)
// gravity: IU velocity change per frame
// maxFall: max fall speed in IU/substep
func ApplyEnemyGravity(w *World, stage Stage, gravity, maxFall int) {
	for id := range w.IsEnemy {
		ai := w.AI[id]
		if ai.Flying {
			continue
		}

		mov := w.Movement[id]
		vel := w.Velocity[id]

		// If on ground, verify ground still exists below
		if mov.OnGround && vel.Y >= 0 {
			pos := w.Position[id]
			hitbox := w.Hitbox[id]
			// Check 1 pixel below feet
			checkY := pos.PixelY() + hitbox.OffsetY + hitbox.Height
			groundExists := stage.IsSolidAt(pos.PixelX()+hitbox.OffsetX, checkY) ||
				stage.IsSolidAt(pos.PixelX()+hitbox.OffsetX+hitbox.Width-1, checkY) ||
				stage.IsSolidAt(pos.PixelX()+hitbox.OffsetX+hitbox.Width/2, checkY)
			if !groundExists {
				mov.OnGround = false
				w.Movement[id] = mov
			}
		}

		if mov.OnGround {
			continue
		}

		vel.Y += gravity
		if vel.Y > maxFall {
			vel.Y = maxFall
		}
		w.Velocity[id] = vel
	}
}

// ApplyProjectileGravity applies gravity to all projectiles (call once per frame)
func ApplyProjectileGravity(w *World) {
	for id := range w.IsProjectile {
		proj := w.ProjectileData[id]
		if proj.Stuck {
			continue
		}

		vel := w.Velocity[id]
		vel.Y += proj.GravityAccel
		if vel.Y > proj.MaxFallSpeed {
			vel.Y = proj.MaxFallSpeed
		}
		w.Velocity[id] = vel
	}
}

// ApplyGoldGravity applies gravity to all gold pickups (call once per frame)
func ApplyGoldGravity(w *World) {
	for id := range w.IsGold {
		gold := w.GoldData[id]
		if gold.Grounded {
			continue
		}

		vel := w.Velocity[id]
		vel.Y += gold.Gravity
		w.Velocity[id] = vel
	}
}

func spawnEnemyArrow(w *World, pos *Position, facingRight bool, cfg ProjectileConfig) {
	px := pos.PixelX() + 8
	py := pos.PixelY() + 8

	dir := 1
	if !facingRight {
		dir = -1
	}

	// Simple horizontal arrow: 220 pixels/sec ≈ 94 IU/substep
	vx := dir * 94
	vy := 0

	w.CreateProjectile(px, py, vx, vy, cfg, false)
}

// UpdateProjectiles updates all projectile physics and movement for one substep
// Gravity is applied separately via ApplyProjectileGravity (once per frame)
func UpdateProjectiles(w *World, stage Stage) {
	toDestroy := make([]EntityID, 0)

	for id := range w.IsProjectile {
		pos := w.Position[id]
		vel := w.Velocity[id]
		proj := w.ProjectileData[id]

		if proj.Stuck {
			continue
		}

		// Movement is velocity (IU/substep)
		dx := vel.X
		dy := vel.Y

		// Substep movement for collision detection
		totalSteps := abs(dx)
		if abs(dy) > totalSteps {
			totalSteps = abs(dy)
		}
		if totalSteps == 0 {
			w.Position[id] = pos
			w.Velocity[id] = vel
			continue
		}

		// Integer-based diagonal stepping
		stepX := dx / totalSteps
		stepY := dy / totalSteps
		remX := dx % totalSteps
		remY := dy % totalSteps
		accumX, accumY := 0, 0

		for i := 0; i < totalSteps; i++ {
			moveX := stepX
			moveY := stepY

			// Distribute remainder evenly
			accumX += abs(remX)
			if accumX >= totalSteps {
				accumX -= totalSteps
				moveX += sign(remX)
			}
			accumY += abs(remY)
			if accumY >= totalSteps {
				accumY -= totalSteps
				moveY += sign(remY)
			}

			pos.X += moveX
			pos.Y += moveY

			px, py := pos.PixelX(), pos.PixelY()
			if stage.IsSolidAt(px, py) {
				proj.StuckRotation = math.Atan2(float64(vel.Y), float64(vel.X))
				proj.Stuck = true
				proj.StuckTimer = 0
				vel.X = 0
				vel.Y = 0
				break
			}
		}

		// Check max range (pixels)
		traveled := abs(pos.PixelX() - proj.StartX)
		if traveled > proj.MaxRange {
			toDestroy = append(toDestroy, id)
			continue
		}

		w.Position[id] = pos
		w.Velocity[id] = vel
		w.ProjectileData[id] = proj
	}

	for _, id := range toDestroy {
		w.DestroyEntity(id)
	}
}

// UpdateGoldPhysics updates gold pickup physics for one substep
// Gravity is applied separately via ApplyGoldGravity (once per frame)
func UpdateGoldPhysics(w *World, stage Stage) {
	for id := range w.IsGold {
		pos := w.Position[id]
		vel := w.Velocity[id]
		gold := w.GoldData[id]

		if gold.Grounded {
			continue
		}

		// Move X (vel.X is in IU, step 1 IU at a time)
		dx := vel.X
		for i := 0; i < abs(dx); i++ {
			step := sign(dx)
			nextPixelX := (pos.X + step) / PositionScale
			if nextPixelX != pos.PixelX() {
				// About to cross pixel boundary, check collision
				if stage.IsSolidAt(nextPixelX, pos.PixelY()) ||
					stage.IsSolidAt(nextPixelX, pos.PixelY()+gold.HitboxHeight-1) {
					// Bounce: reverse and decay (percentage)
					vel.X = -vel.X * gold.BouncePercent / 100
					break
				}
			}
			pos.X += step // 1 IU per step
		}

		// Move Y (vel.Y is in IU, step 1 IU at a time)
		dy := vel.Y
		for i := 0; i < abs(dy); i++ {
			step := sign(dy)
			nextPixelY := (pos.Y + step) / PositionScale
			if nextPixelY != pos.PixelY() {
				// About to cross pixel boundary, check collision
				if stage.IsSolidAt(pos.PixelX(), nextPixelY+gold.HitboxHeight-1) ||
					stage.IsSolidAt(pos.PixelX()+gold.HitboxWidth-1, nextPixelY+gold.HitboxHeight-1) {
					if step > 0 {
						gold.Grounded = true
						vel.Y = 0
						vel.X = 0
					} else {
						vel.Y = -vel.Y * gold.BouncePercent / 100
					}
					break
				}
			}
			pos.Y += step // 1 IU per step
		}

		w.Position[id] = pos
		w.Velocity[id] = vel
		w.GoldData[id] = gold
	}
}

// CollectGold checks for gold collection by player
// Uses squared distance comparison for integer math
func CollectGold(w *World) {
	playerID := w.PlayerID
	if playerID == 0 {
		return
	}

	playerPos := w.Position[playerID]
	playerHitbox := w.HitboxTrapezoid[playerID]
	playerData := w.PlayerData[playerID]

	px := playerPos.PixelX() + playerHitbox.Body.OffsetX + playerHitbox.Body.Width/2
	py := playerPos.PixelY() + playerHitbox.Body.OffsetY + playerHitbox.Body.Height/2

	toDestroy := make([]EntityID, 0)

	for id := range w.IsGold {
		gold := w.GoldData[id]
		if gold.CollectDelay > 0 {
			continue
		}

		pos := w.Position[id]
		gx := pos.PixelX() + gold.HitboxWidth/2
		gy := pos.PixelY() + gold.HitboxHeight/2

		// Squared distance comparison (avoid sqrt)
		dx := px - gx
		dy := py - gy
		distSq := dx*dx + dy*dy
		radiusSq := gold.CollectRadius * gold.CollectRadius
		if distSq < radiusSq {
			playerData.Gold += gold.Amount
			toDestroy = append(toDestroy, id)
		}
	}

	w.PlayerData[playerID] = playerData

	for _, id := range toDestroy {
		w.DestroyEntity(id)
	}
}

// DamageResult holds information about damage events
type DamageResult struct {
	HitstopFrames   int
	ScreenShake     float64 // Rendering only
	PlayerDamaged   bool
	PlayerKnockback struct {
		VX, VY int // IU/substep
	}
}

// UpdateDamage checks collisions and applies damage
// knockbackForce, knockbackUp: IU/substep
func UpdateDamage(w *World, knockbackForce, knockbackUp int, iframeFrames int) DamageResult {
	result := DamageResult{}

	// Player projectiles vs enemies
	enemiesToDestroy := make([]EntityID, 0)
	projToDestroy := make([]EntityID, 0)

	for projID := range w.IsProjectile {
		proj := w.ProjectileData[projID]
		if !proj.IsPlayerOwned || proj.Stuck {
			continue
		}

		projPos := w.Position[projID]
		projHit := w.Hitbox[projID]
		projPX, projPY := projPos.PixelX(), projPos.PixelY()

		for enemyID := range w.IsEnemy {
			enemyPos := w.Position[enemyID]
			enemyHit := w.Hitbox[enemyID]
			enemyPX, enemyPY := enemyPos.PixelX(), enemyPos.PixelY()

			if rectsOverlap(
				projPX+projHit.OffsetX, projPY+projHit.OffsetY, projHit.Width, projHit.Height,
				enemyPX+enemyHit.OffsetX, enemyPY+enemyHit.OffsetY, enemyHit.Width, enemyHit.Height,
			) {
				health := w.Health[enemyID]
				ai := w.AI[enemyID]
				health.Current -= proj.Damage

				// Calculate knockback based on projectile velocity direction
				projVel := w.Velocity[projID]
				kbVelX, kbVelY := calcKnockbackFromVelocity(projVel.X, projVel.Y, knockbackForce)

				// Set hit stun and store initial knockback values
				hitFrames := 12
				ai.HitTimer = hitFrames
				ai.HitTimerMax = hitFrames
				ai.KnockbackVelX = kbVelX
				ai.KnockbackVelY = kbVelY

				// Apply initial knockback velocity
				vel := w.Velocity[enemyID]
				vel.X = kbVelX
				vel.Y = kbVelY
				w.Velocity[enemyID] = vel

				result.HitstopFrames = 3
				result.ScreenShake = 4.0

				if health.Current <= 0 {
					enemiesToDestroy = append(enemiesToDestroy, enemyID)
				} else {
					w.Health[enemyID] = health
					w.AI[enemyID] = ai
				}

				projToDestroy = append(projToDestroy, projID)
				break
			}
		}
	}

	// Spawn gold for killed enemies
	for _, id := range enemiesToDestroy {
		pos := w.Position[id]
		ai := w.AI[id]
		amount := ai.GoldDropMin
		if ai.GoldDropMax > ai.GoldDropMin {
			amount += (ai.GoldDropMax - ai.GoldDropMin) / 2 // simple average
		}
		w.CreateGold(pos.PixelX()+8, pos.PixelY(), amount, GoldConfig{
			Gravity:       ToIUAccelPerFrame(400), // 400 pixels/sec² → IU velocity change per frame
			BouncePercent: 50,                     // 50% velocity retained on bounce
			CollectDelay:  18,                     // 0.3 seconds
			HitboxWidth:   8,
			HitboxHeight:  8,
			CollectRadius: 16,
		})
		w.DestroyEntity(id)
	}

	for _, id := range projToDestroy {
		w.DestroyEntity(id)
	}

	// Enemy projectiles vs player
	playerID := w.PlayerID
	if playerID != 0 {
		playerData := w.PlayerData[playerID]
		dash := w.Dash[playerID]

		if !playerData.IsInvincible(dash.Active) {
			playerPos := w.Position[playerID]
			playerHitbox := w.HitboxTrapezoid[playerID]
			playerFacing := w.Facing[playerID]
			playerPX, playerPY := playerPos.PixelX(), playerPos.PixelY()
			px, py, pw, ph := playerHitbox.Body.GetWorldRect(playerPX, playerPY, playerFacing.Right, 16)

			for projID := range w.IsProjectile {
				proj := w.ProjectileData[projID]
				if proj.IsPlayerOwned || proj.Stuck {
					continue
				}

				projPos := w.Position[projID]
				projHit := w.Hitbox[projID]
				projPX, projPY := projPos.PixelX(), projPos.PixelY()

				if rectsOverlap(
					projPX+projHit.OffsetX, projPY+projHit.OffsetY, projHit.Width, projHit.Height,
					px, py, pw, ph,
				) {
					health := w.Health[playerID]
					health.Current -= proj.Damage
					playerData.IframeTimer = iframeFrames
					w.Health[playerID] = health
					w.PlayerData[playerID] = playerData

					result.PlayerDamaged = true
					result.ScreenShake = 6.0

					// Knockback (values already in IU/substep)
					dir := 1
					if projPos.PixelX() > playerPX {
						dir = -1
					}
					result.PlayerKnockback.VX = dir * knockbackForce
					result.PlayerKnockback.VY = -knockbackUp

					w.DestroyEntity(projID)
					break
				}
			}
		}

		// Enemy contact vs player
		if !playerData.IsInvincible(dash.Active) {
			playerPos := w.Position[playerID]
			playerHitbox := w.HitboxTrapezoid[playerID]
			playerFacing := w.Facing[playerID]
			playerPX, playerPY := playerPos.PixelX(), playerPos.PixelY()
			px, py, pw, ph := playerHitbox.Body.GetWorldRect(playerPX, playerPY, playerFacing.Right, 16)

			for enemyID := range w.IsEnemy {
				enemyPos := w.Position[enemyID]
				enemyHit := w.Hitbox[enemyID]
				ai := w.AI[enemyID]
				enemyPX, enemyPY := enemyPos.PixelX(), enemyPos.PixelY()

				if rectsOverlap(
					enemyPX+enemyHit.OffsetX, enemyPY+enemyHit.OffsetY, enemyHit.Width, enemyHit.Height,
					px, py, pw, ph,
				) {
					health := w.Health[playerID]
					health.Current -= ai.ContactDamage
					playerData.IframeTimer = iframeFrames
					playerData.StunTimer = 12 // stun frames
					w.Health[playerID] = health
					w.PlayerData[playerID] = playerData

					result.PlayerDamaged = true
					result.ScreenShake = 6.0

					// Knockback
					dir := 1
					if enemyPX > playerPX {
						dir = -1
					}
					result.PlayerKnockback.VX = dir * knockbackForce
					result.PlayerKnockback.VY = -knockbackUp
					break
				}
			}
		}

		// Apply knockback
		if result.PlayerDamaged {
			vel := w.Velocity[playerID]
			vel.X = result.PlayerKnockback.VX
			vel.Y = result.PlayerKnockback.VY
			w.Velocity[playerID] = vel
		}
	}

	return result
}

// ResolveEnemyCollisions pushes overlapping enemies apart
func ResolveEnemyCollisions(w *World) {
	enemies := make([]EntityID, 0, len(w.IsEnemy))
	for id := range w.IsEnemy {
		enemies = append(enemies, id)
	}

	for i := 0; i < len(enemies); i++ {
		e1 := enemies[i]
		pos1 := w.Position[e1]
		hit1 := w.Hitbox[e1]
		px1, py1 := pos1.PixelX(), pos1.PixelY()
		x1, y1, w1, h1 := px1+hit1.OffsetX, py1+hit1.OffsetY, hit1.Width, hit1.Height

		for j := i + 1; j < len(enemies); j++ {
			e2 := enemies[j]
			pos2 := w.Position[e2]
			hit2 := w.Hitbox[e2]
			px2, py2 := pos2.PixelX(), pos2.PixelY()
			x2, y2, w2, h2 := px2+hit2.OffsetX, py2+hit2.OffsetY, hit2.Width, hit2.Height

			if !rectsOverlap(x1, y1, w1, h1, x2, y2, w2, h2) {
				continue
			}

			cx1 := float64(x1) + float64(w1)/2
			cx2 := float64(x2) + float64(w2)/2

			pushAmount := 2 * PositionScale
			if cx1 < cx2 {
				pos1.X -= pushAmount
				pos2.X += pushAmount
			} else {
				pos1.X += pushAmount
				pos2.X -= pushAmount
			}

			w.Position[e1] = pos1
			w.Position[e2] = pos2
		}
	}
}

// Helper functions
func rectsOverlap(x1, y1, w1, h1, x2, y2, w2, h2 int) bool {
	return x1 < x2+w2 && x1+w1 > x2 && y1 < y2+h2 && y1+h1 > y2
}

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

// calcKnockbackFromVelocity calculates knockback velocity based on projectile direction.
// Returns knockback velocity in the same direction as the projectile was traveling.
func calcKnockbackFromVelocity(velX, velY, force int) (kbX, kbY int) {
	// Calculate magnitude using integer approximation
	// Use the larger component as base, add fraction of smaller
	absX := abs(velX)
	absY := abs(velY)

	if absX == 0 && absY == 0 {
		// Fallback: push right if no velocity
		return force, 0
	}

	// Approximate magnitude: max + 0.5*min (rough approximation of sqrt(x²+y²))
	var mag int
	if absX > absY {
		mag = absX + absY/2
	} else {
		mag = absY + absX/2
	}

	if mag == 0 {
		mag = 1
	}

	// Normalize and scale by force
	kbX = velX * force / mag
	kbY = velY * force / mag

	return kbX, kbY
}
