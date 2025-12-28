package entity

// Body represents the physical body of an entity
// Position is stored as integers for deterministic physics
// Velocity is stored as float for smooth movement, with remainder accumulation
type Body struct {
	X, Y       int     // Pixel position (integer)
	VX, VY     float64 // Velocity (pixels per second)
	RemX, RemY float64 // Subpixel remainder for smooth movement

	OnGround     bool
	OnCeiling    bool
	OnWallLeft   bool
	OnWallRight  bool
	FacingRight  bool
	WasOnGround  bool // For coyote time
}

// TrapezoidHitbox represents a hitbox approximated by three rects
// Head is narrow (corner correction), Feet is wide (coyote time)
type TrapezoidHitbox struct {
	Head HitboxRect // Narrow - forgiving ceiling collision
	Body HitboxRect // Standard hitbox
	Feet HitboxRect // Wide - forgiving ground collision
}

// HitboxRect represents a single hitbox rectangle
type HitboxRect struct {
	OffsetX int
	OffsetY int
	Width   int
	Height  int
}

// GetWorldRect returns the hitbox rect in world coordinates
func (hr HitboxRect) GetWorldRect(bodyX, bodyY int, facingRight bool, spriteWidth int) (x, y, w, h int) {
	offsetX := hr.OffsetX
	if !facingRight {
		// Mirror the offset for left-facing
		offsetX = spriteWidth - hr.OffsetX - hr.Width
	}
	return bodyX + offsetX, bodyY + hr.OffsetY, hr.Width, hr.Height
}

// ApplyVelocity applies velocity to position, returning integer pixels to move
// Remainder is accumulated for next frame
func (b *Body) ApplyVelocity(dt float64) (dx, dy int) {
	moveX := b.VX*dt + b.RemX
	moveY := b.VY*dt + b.RemY

	dx = int(moveX)
	dy = int(moveY)

	b.RemX = moveX - float64(dx)
	b.RemY = moveY - float64(dy)

	return dx, dy
}

// Player represents the player entity
type Player struct {
	Body
	Hitbox TrapezoidHitbox

	Health    int
	MaxHealth int
	Gold      int

	// Timers
	CoyoteTimer      float64
	JumpBufferTimer  float64
	DashTimer        float64
	DashCooldown     float64
	IframeTimer      float64
	StunTimer        float64

	// State
	Dashing    bool
	Attacking  bool
	CanDash    bool
}

// NewPlayer creates a new player with default values
func NewPlayer(x, y int, hitbox TrapezoidHitbox, maxHealth int) *Player {
	return &Player{
		Body: Body{
			X:           x,
			Y:           y,
			FacingRight: true,
		},
		Hitbox:    hitbox,
		Health:    maxHealth,
		MaxHealth: maxHealth,
		CanDash:   true,
	}
}

// IsInvincible returns true if player is currently invincible
func (p *Player) IsInvincible() bool {
	return p.IframeTimer > 0 || p.Dashing
}

// IsStunned returns true if player is currently stunned
func (p *Player) IsStunned() bool {
	return p.StunTimer > 0
}
