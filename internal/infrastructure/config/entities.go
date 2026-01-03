package config

// EntitiesConfig is the root config for entities.json
type EntitiesConfig struct {
	Player      PlayerConfig               `json:"player"`
	Projectiles map[string]ProjectileConfig `json:"projectiles"`
	Enemies     map[string]EnemyConfig      `json:"enemies"`
	Pickups     map[string]PickupConfig     `json:"pickups"`
	Effects     map[string]EffectConfig     `json:"effects"`
}

type PlayerConfig struct {
	ID      string       `json:"id"`
	Sprite  SpriteConfig `json:"sprite"`
	Hitbox  HitboxConfig `json:"hitbox"`
	Hurtbox Rect         `json:"hurtbox"`
	Stats   PlayerStats  `json:"stats"`
}

type SpriteConfig struct {
	Sheet      string                     `json:"sheet"`
	FrameWidth  int                       `json:"frameWidth"`
	FrameHeight int                       `json:"frameHeight"`
	Animations  map[string]AnimationConfig `json:"animations"`
}

type AnimationConfig struct {
	Row    int `json:"row"`
	Frames int `json:"frames"`
	FPS    int `json:"fps"`
}

type HitboxConfig struct {
	Head Rect `json:"head"`
	Body Rect `json:"body"`
	Feet Rect `json:"feet"`
}

type Rect struct {
	OffsetX int `json:"offsetX"`
	OffsetY int `json:"offsetY"`
	Width   int `json:"width"`
	Height  int `json:"height"`
}

type PlayerStats struct {
	MaxHealth    int `json:"maxHealth"`
	AttackDamage int `json:"attackDamage"`
}

type ProjectileConfig struct {
	ID      string                 `json:"id"`
	Sprite  SpriteConfig           `json:"sprite"`
	Hitbox  Rect                   `json:"hitbox"`
	Physics ProjectilePhysicsConfig `json:"physics"`
	Damage  int                    `json:"damage"`
}

type ProjectilePhysicsConfig struct {
	Speed            float64 `json:"speed"`
	LaunchAngleDeg   float64 `json:"launchAngleDeg"`
	GravityAccel     float64 `json:"gravityAccel"`
	MaxFallSpeed     float64 `json:"maxFallSpeed"`
	MaxRange         float64 `json:"maxRange"`
	RotateToVelocity bool    `json:"rotateToVelocity"`
	Piercing         bool    `json:"piercing"`
}

type EnemyConfig struct {
	ID      string           `json:"id"`
	Sprite  SpriteConfig     `json:"sprite"`
	Hitbox  EnemyHitboxConfig `json:"hitbox"`
	Hurtbox Rect             `json:"hurtbox"`
	Stats   EnemyStats       `json:"stats"`
	AI      AIConfig         `json:"ai"`
}

type EnemyHitboxConfig struct {
	Body Rect `json:"body"`
}

type EnemyStats struct {
	MaxHealth     int      `json:"maxHealth"`
	ContactDamage int      `json:"contactDamage"`
	MoveSpeed     float64  `json:"moveSpeed,omitempty"`
	GoldDrop      GoldDrop `json:"goldDrop"`
}

type GoldDrop struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type AIConfig struct {
	Type           string  `json:"type"`
	DetectRange    float64 `json:"detectRange,omitempty"`
	PatrolDistance float64 `json:"patrolDistance,omitempty"`
	PauseDuration  float64 `json:"pauseDuration,omitempty"`
	AttackRange    float64 `json:"attackRange,omitempty"`
	AttackCooldown float64 `json:"attackCooldown,omitempty"`
	Projectile     string  `json:"projectile,omitempty"`
	ChaseSpeed     float64 `json:"chaseSpeed,omitempty"`
	Flying         bool    `json:"flying,omitempty"`
	JumpForce      float64 `json:"jumpForce,omitempty"` // For aggressive AI
}

type PickupConfig struct {
	ID         string             `json:"id"`
	Sprite     SpriteConfig       `json:"sprite"`
	Hitbox     Rect               `json:"hitbox"`
	Physics    PickupPhysicsConfig `json:"physics,omitempty"`
	HealAmount int                `json:"healAmount,omitempty"`
}

type PickupPhysicsConfig struct {
	Gravity       float64 `json:"gravity"`
	BounceDecay   float64 `json:"bounceDecay"`
	CollectDelay  float64 `json:"collectDelay"`
	CollectRadius float64 `json:"collectRadius"`
}

type EffectConfig struct {
	ID       string       `json:"id"`
	Sprite   SpriteConfig `json:"sprite"`
	Duration float64      `json:"duration"`
}
