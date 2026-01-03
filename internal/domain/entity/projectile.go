package entity

import "math"

// Projectile represents a projectile entity (arrows, etc.)
type Projectile struct {
	X, Y     float64
	VX, VY   float64
	RemX     float64 // Sub-pixel remainder
	RemY     float64
	StartX   float64
	Active   bool
	IsPlayer bool // true if shot by player, false if by enemy

	// Physics config
	GravityAccel float64
	MaxFallSpeed float64
	MaxRange     float64
	Damage       int

	// Hitbox
	HitboxOffsetX int
	HitboxOffsetY int
	HitboxWidth   int
	HitboxHeight  int

	// Stuck state (when hitting wall)
	Stuck         bool
	StuckTimer    float64
	StuckDuration float64
	StuckRotation float64
}

// NewArrow creates a new arrow projectile
func NewArrow(x, y float64, facingRight bool, speed, launchAngleDeg, gravityAccel, maxFallSpeed, maxRange float64, damage int, isPlayer bool) *Projectile {
	angleRad := launchAngleDeg * math.Pi / 180
	dir := 1.0
	if !facingRight {
		dir = -1.0
	}

	return &Projectile{
		X:            x,
		Y:            y,
		VX:           dir * speed * math.Cos(angleRad),
		VY:           -speed * math.Sin(angleRad), // Negative because up is negative
		StartX:       x,
		Active:       true,
		IsPlayer:     isPlayer,
		GravityAccel: gravityAccel,
		MaxFallSpeed: maxFallSpeed,
		MaxRange:     maxRange,
		Damage:       damage,
		HitboxOffsetX: 2,
		HitboxOffsetY: 2,
		HitboxWidth:   12,
		HitboxHeight:  4,
	}
}

// NewArrowDirected creates a new arrow projectile toward a target direction
func NewArrowDirected(x, y, targetX, targetY, speed, gravityAccel, maxFallSpeed, maxRange float64, damage int, isPlayer bool) *Projectile {
	return NewArrowDirectedWithVelocity(x, y, targetX, targetY, speed, 0, 0, 0, gravityAccel, maxFallSpeed, maxRange, damage, isPlayer)
}

// NewArrowDirectedWithVelocity creates an arrow with player velocity influence
// playerVX, playerVY: player's current velocity (in pixels/sec, NOT 100x scaled)
// velocityInfluence: 0.0 = no influence, 1.0 = full influence
func NewArrowDirectedWithVelocity(x, y, targetX, targetY, speed, playerVX, playerVY, velocityInfluence, gravityAccel, maxFallSpeed, maxRange float64, damage int, isPlayer bool) *Projectile {
	dx := targetX - x
	dy := targetY - y
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1 {
		dist = 1
	}

	// Base arrow velocity (toward target)
	vx := (dx / dist) * speed
	vy := (dy / dist) * speed

	// Add player velocity with influence multiplier
	vx += playerVX * velocityInfluence
	vy += playerVY * velocityInfluence

	return &Projectile{
		X:             x,
		Y:             y,
		VX:            vx,
		VY:            vy,
		StartX:        x,
		Active:        true,
		IsPlayer:      isPlayer,
		GravityAccel:  gravityAccel,
		MaxFallSpeed:  maxFallSpeed,
		MaxRange:      maxRange,
		Damage:        damage,
		HitboxOffsetX: 2,
		HitboxOffsetY: 2,
		HitboxWidth:   12,
		HitboxHeight:  4,
	}
}

// Update updates the projectile physics (gravity only, movement handled separately)
func (p *Projectile) Update(dt float64) {
	if !p.Active {
		return
	}

	// Handle stuck state
	if p.Stuck {
		p.StuckTimer += dt
		if p.StuckTimer >= p.StuckDuration {
			p.Active = false
		}
		return
	}

	// Apply gravity acceleration
	p.VY += p.GravityAccel * dt

	// Clamp fall speed
	if p.VY > p.MaxFallSpeed {
		p.VY = p.MaxFallSpeed
	}
}

// ApplyVelocity calculates pixels to move and accumulates remainder
func (p *Projectile) ApplyVelocity(dt float64) (dx, dy int) {
	// Calculate movement with sub-pixel accumulation
	moveX := p.VX*dt + p.RemX
	moveY := p.VY*dt + p.RemY

	dx = int(moveX)
	dy = int(moveY)

	p.RemX = moveX - float64(dx)
	p.RemY = moveY - float64(dy)

	return dx, dy
}

// StickToWall makes the projectile stick to a wall
func (p *Projectile) StickToWall(duration float64) {
	p.StuckRotation = math.Atan2(p.VY, p.VX) // Save rotation before clearing velocity
	p.Stuck = true
	p.StuckTimer = 0
	p.StuckDuration = duration
	p.VX = 0
	p.VY = 0
}

// GetAlpha returns the alpha value (0-1) for rendering, fading in last second
func (p *Projectile) GetAlpha() float64 {
	if !p.Stuck {
		return 1.0
	}
	fadeStart := p.StuckDuration - 1.0
	if p.StuckTimer < fadeStart {
		return 1.0
	}
	return 1.0 - (p.StuckTimer-fadeStart)/1.0
}

// Rotation returns the rotation angle based on velocity vector
func (p *Projectile) Rotation() float64 {
	if p.Stuck {
		return p.StuckRotation
	}
	return math.Atan2(p.VY, p.VX)
}

// GetHitbox returns the hitbox in world coordinates
func (p *Projectile) GetHitbox() (x, y, w, h int) {
	return int(p.X) + p.HitboxOffsetX, int(p.Y) + p.HitboxOffsetY, p.HitboxWidth, p.HitboxHeight
}

// Deactivate marks the projectile as inactive
func (p *Projectile) Deactivate() {
	p.Active = false
}
