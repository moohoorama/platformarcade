package entity

// AIType defines the type of AI behavior
type AIType int

const (
	AIPatrol AIType = iota
	AIRanged
	AIChase
)

// Enemy represents an enemy entity
type Enemy struct {
	ID     EntityID
	X, Y   int
	VX, VY float64
	Active bool

	// Sub-pixel remainder for smooth movement
	RemX, RemY float64

	// Properties
	EnemyType     string
	MaxHealth     int
	Health        int
	ContactDamage int
	MoveSpeed     float64
	FacingRight   bool

	// Hitbox
	HitboxOffsetX int
	HitboxOffsetY int
	HitboxWidth   int
	HitboxHeight  int

	// AI
	AIType         AIType
	DetectRange    float64
	PatrolDistance float64
	AttackRange    float64
	AttackCooldown float64
	Flying         bool

	// State
	PatrolStartX int
	PatrolDir    int
	AttackTimer  float64
	HitTimer     float64

	// Gold drop
	GoldDropMin int
	GoldDropMax int
}

// NewEnemy creates a new enemy
func NewEnemy(id EntityID, x, y int, enemyType string) *Enemy {
	return &Enemy{
		ID:          id,
		X:           x,
		Y:           y,
		Active:      true,
		EnemyType:   enemyType,
		FacingRight: false,
		PatrolStartX: x,
		PatrolDir:   -1,
	}
}

// TakeDamage applies damage to the enemy
func (e *Enemy) TakeDamage(damage int) bool {
	e.Health -= damage
	e.HitTimer = 0.2 // Hit stun
	return e.Health <= 0
}

// IsAlive returns true if enemy is still alive
func (e *Enemy) IsAlive() bool {
	return e.Health > 0 && e.Active
}

// GetHitbox returns the hitbox in world coordinates
func (e *Enemy) GetHitbox() (x, y, w, h int) {
	return e.X + e.HitboxOffsetX, e.Y + e.HitboxOffsetY, e.HitboxWidth, e.HitboxHeight
}

// Gold represents a gold pickup
type Gold struct {
	X, Y     float64
	VX, VY   float64
	Amount   int
	Active   bool
	Grounded bool
	CollectDelay float64

	// Physics
	Gravity     float64
	BounceDecay float64

	// Hitbox (from config)
	HitboxWidth   int
	HitboxHeight  int
	CollectRadius float64
}

// NewGold creates a new gold pickup
func NewGold(x, y float64, amount int, gravity, bounceDecay, collectDelay float64, hitboxW, hitboxH int, collectRadius float64) *Gold {
	return &Gold{
		X:             x,
		Y:             y,
		VX:            float64((amount%10)-5) * 20, // Random spread
		VY:            -100,                         // Pop up
		Amount:        amount,
		Active:        true,
		Gravity:       gravity,
		BounceDecay:   bounceDecay,
		CollectDelay:  collectDelay,
		HitboxWidth:   hitboxW,
		HitboxHeight:  hitboxH,
		CollectRadius: collectRadius,
	}
}

// Update is now handled by CombatSystem with proper collision detection

// CanCollect returns true if gold can be collected
func (g *Gold) CanCollect() bool {
	return g.Active && g.CollectDelay <= 0
}
