package entity

// PositionScale is the internal position scale factor.
// 1 pixel = 100 internal units. This provides 0.01 pixel precision.
const PositionScale = 100

// Body represents the physical body of an entity
// Position is stored at 100x scale for sub-pixel precision without floats.
// Velocity is stored as float in 100x scale units per second.
type Body struct {
	X, Y   int     // 100x scaled position (divide by PositionScale for pixels)
	VX, VY float64 // 100x scaled velocity (units per second)

	OnGround     bool
	OnCeiling    bool
	OnWallLeft   bool
	OnWallRight  bool
	FacingRight  bool
	WasOnGround  bool // For coyote time
}

// PixelX returns the pixel X position (internal X / PositionScale)
func (b *Body) PixelX() int {
	return b.X / PositionScale
}

// PixelY returns the pixel Y position (internal Y / PositionScale)
func (b *Body) PixelY() int {
	return b.Y / PositionScale
}

// SetPixelPos sets the position from pixel coordinates (converts to 100x scale)
func (b *Body) SetPixelPos(x, y int) {
	b.X = x * PositionScale
	b.Y = y * PositionScale
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

// ApplyVelocity applies velocity to position, returning integer units to move.
// With 100x scale, no remainder accumulation is needed as precision is built-in.
func (b *Body) ApplyVelocity(dt float64) (dx, dy int) {
	dx = int(b.VX * dt)
	dy = int(b.VY * dt)
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

	// Arrow system
	EquippedArrows [4]ArrowType // 장착된 화살 (방향별: 0=오른쪽, 1=위, 2=왼쪽, 3=아래)
	CurrentArrow   ArrowType    // 현재 사용 중인 화살
}

// NewPlayer creates a new player with default values.
// x, y are pixel coordinates which are internally stored at 100x scale.
func NewPlayer(x, y int, hitbox TrapezoidHitbox, maxHealth int) *Player {
	return &Player{
		Body: Body{
			X:           x * PositionScale,
			Y:           y * PositionScale,
			FacingRight: true,
		},
		Hitbox:    hitbox,
		Health:    maxHealth,
		MaxHealth: maxHealth,
		CanDash:   true,
		// Default: all 4 arrow types equipped
		EquippedArrows: [4]ArrowType{ArrowGray, ArrowRed, ArrowBlue, ArrowPurple},
		CurrentArrow:   ArrowGray,
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
