package ecs

import (
	"image/color"
	"math"
)

// PositionScale is the internal position scale factor.
// 1 pixel = 256 internal units for sub-pixel precision.
// Using 256 (2^8) allows bit-shift optimization for pixel conversion.
const PositionScale = 256

// PositionShift is the bit shift amount for pixel conversion (log2(256) = 8)
const PositionShift = 8

// Position represents an entity's position (256x scaled)
type Position struct {
	X, Y int
}

// PixelX returns the pixel X coordinate
func (p Position) PixelX() int { return p.X >> PositionShift }

// PixelY returns the pixel Y coordinate
func (p Position) PixelY() int { return p.Y >> PositionShift }

// Velocity represents movement speed in internal units per substep.
// All values are integers for deterministic simulation.
type Velocity struct {
	X, Y int // IU per substep
}

// Movement represents movement/collision state
type Movement struct {
	OnGround    bool
	OnCeiling   bool
	OnWallLeft  bool
	OnWallRight bool
	WasOnGround bool // for coyote time

	Stunned bool // Cannot control
	HitStun int  // Hit stagger frames
}

// Health represents entity health with iframe
type Health struct {
	Current int
	Max     int
	Iframe  int // Invincibility frames (0 = can be hit)
}

// TakeDamage applies damage if not invincible, returns true if dead
func (h *Health) TakeDamage(amount int) bool {
	if h.Iframe > 0 {
		return false
	}
	h.Current -= amount
	return h.Current <= 0
}

// IsAlive returns true if health > 0
func (h *Health) IsAlive() bool {
	return h.Current > 0
}

// Heal restores health up to max
func (h *Health) Heal(amount int) {
	h.Current += amount
	if h.Current > h.Max {
		h.Current = h.Max
	}
}

// Hitbox represents a collision area
type Hitbox struct {
	OffsetX, OffsetY int
	Width, Height    int
}

// GetWorldRect returns the hitbox in world coordinates
func (h Hitbox) GetWorldRect(pixelX, pixelY int, facingRight bool, spriteWidth int) (x, y, w, hh int) {
	offsetX := h.OffsetX
	if !facingRight {
		offsetX = spriteWidth - h.OffsetX - h.Width
	}
	return pixelX + offsetX, pixelY + h.OffsetY, h.Width, h.Height
}

// HitboxTrapezoid is for player (head/body/feet)
type HitboxTrapezoid struct {
	Head Hitbox
	Body Hitbox
	Feet Hitbox
}

// Facing represents which direction entity faces
type Facing struct {
	Right bool
}

// AIType defines enemy AI behavior type
type AIType int

const (
	AIPatrol AIType = iota
	AIAggressive
	AIRanged
	AIChase
)

// AI represents enemy behavior
type AI struct {
	Type           AIType
	DetectRange    int // pixels
	AttackRange    int // pixels
	PatrolDistance int // pixels
	JumpForce      int // IU per substep
	MoveSpeed      int // IU per substep
	ContactDamage  int
	Flying         bool

	// State
	PatrolStartX int
	PatrolDir    int
	AttackTimer  int // frames (cooldown)
	HitTimer     int // frames (hit stun)
	HitTimerMax  int // initial HitTimer value (for decay calculation)

	// Knockback (initial values for smooth deceleration)
	KnockbackVelX int // initial knockback X velocity (IU/substep)
	KnockbackVelY int // initial knockback Y velocity (IU/substep)

	// Gold drop
	GoldDropMin int
	GoldDropMax int
}

// Dash represents dash ability state
type Dash struct {
	Active   bool
	Timer    int  // remaining dash frames
	Cooldown int  // cooldown frames
	CanDash  bool // reset when grounded
}

// Projectile represents projectile-specific data
type Projectile struct {
	StartX        int // pixel X at spawn
	GravityAccel  int // IU per substep²
	MaxFallSpeed  int // IU per substep
	MaxRange      int // pixels
	Damage        int
	IsPlayerOwned bool

	// Stuck state
	Stuck         bool
	StuckTimer    int     // frames
	StuckDuration int     // frames
	StuckRotation float64 // radians (rendering only)
}

// Rotation returns the rotation angle based on velocity (for rendering)
func (p *Projectile) Rotation(vx, vy int) float64 {
	if p.Stuck {
		return p.StuckRotation
	}
	return math.Atan2(float64(vy), float64(vx))
}

// GetAlpha returns alpha for rendering (fading when stuck)
func (p *Projectile) GetAlpha() float64 {
	if !p.Stuck {
		return 1.0
	}
	fadeStart := p.StuckDuration - 60 // fade in last second
	if p.StuckTimer < fadeStart {
		return 1.0
	}
	return 1.0 - float64(p.StuckTimer-fadeStart)/60.0
}

// Gold represents gold pickup data
type Gold struct {
	Amount        int
	Grounded      bool
	CollectDelay  int // frames until collectible
	Gravity       int // IU per substep²
	BouncePercent int // 0-100 (70 = 70% velocity retained on bounce)
	CollectRadius int // pixels
	HitboxWidth   int // pixels
	HitboxHeight  int // pixels
}

// Player represents player-specific data
type Player struct {
	Gold           int
	EquippedArrows [4]ArrowType
	CurrentArrow   ArrowType

	// Timers (frames)
	CoyoteTimer     int
	JumpBufferTimer int
	IframeTimer     int
	StunTimer       int
}

// IsInvincible returns true if player has active i-frames or is dashing
func (p *Player) IsInvincible(dashing bool) bool {
	return p.IframeTimer > 0 || dashing
}

// IsStunned returns true if player is stunned
func (p *Player) IsStunned() bool {
	return p.StunTimer > 0
}

// ArrowType represents the type of arrow
type ArrowType int

const (
	ArrowGray   ArrowType = iota
	ArrowRed
	ArrowBlue
	ArrowPurple
)

// ArrowColors maps arrow types to their colors
var ArrowColors = map[ArrowType]color.RGBA{
	ArrowGray:   {128, 128, 128, 255},
	ArrowRed:    {255, 80, 80, 255},
	ArrowBlue:   {80, 80, 255, 255},
	ArrowPurple: {180, 80, 255, 255},
}
