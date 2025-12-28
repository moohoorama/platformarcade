package entity

import "math"

// Projectile represents a projectile entity (arrows, etc.)
type Projectile struct {
	X, Y     float64
	VX, VY   float64
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

// Update updates the projectile physics
func (p *Projectile) Update(dt float64) {
	if !p.Active {
		return
	}

	// Apply gravity acceleration
	p.VY += p.GravityAccel * dt

	// Clamp fall speed
	if p.VY > p.MaxFallSpeed {
		p.VY = p.MaxFallSpeed
	}

	// Update position
	p.X += p.VX * dt
	p.Y += p.VY * dt

	// Check range
	if math.Abs(p.X-p.StartX) > p.MaxRange {
		p.Active = false
	}
}

// Rotation returns the rotation angle based on velocity vector
func (p *Projectile) Rotation() float64 {
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
