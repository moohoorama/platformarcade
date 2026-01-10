package ecs

// EntityID is a unique identifier for an entity (never recycled)
type EntityID uint64

// World holds all component maps and the next entity ID
type World struct {
	nextID EntityID

	// Components
	Position        map[EntityID]Position
	Velocity        map[EntityID]Velocity
	Movement        map[EntityID]Movement
	Health          map[EntityID]Health
	Hitbox          map[EntityID]Hitbox
	HitboxTrapezoid map[EntityID]HitboxTrapezoid
	Facing          map[EntityID]Facing
	AI              map[EntityID]AI
	Dash            map[EntityID]Dash
	ProjectileData  map[EntityID]Projectile
	GoldData        map[EntityID]Gold
	PlayerData      map[EntityID]Player

	// Tags
	IsPlayer     map[EntityID]struct{}
	IsEnemy      map[EntityID]struct{}
	IsProjectile map[EntityID]struct{}
	IsGold       map[EntityID]struct{}

	// Singleton references
	PlayerID EntityID
}

// NewWorld creates a new empty world
func NewWorld() *World {
	return &World{
		nextID:          1, // 0 is "nil"
		Position:        make(map[EntityID]Position),
		Velocity:        make(map[EntityID]Velocity),
		Movement:        make(map[EntityID]Movement),
		Health:          make(map[EntityID]Health),
		Hitbox:          make(map[EntityID]Hitbox),
		HitboxTrapezoid: make(map[EntityID]HitboxTrapezoid),
		Facing:          make(map[EntityID]Facing),
		AI:              make(map[EntityID]AI),
		Dash:            make(map[EntityID]Dash),
		ProjectileData:  make(map[EntityID]Projectile),
		GoldData:        make(map[EntityID]Gold),
		PlayerData:      make(map[EntityID]Player),
		IsPlayer:        make(map[EntityID]struct{}),
		IsEnemy:         make(map[EntityID]struct{}),
		IsProjectile:    make(map[EntityID]struct{}),
		IsGold:          make(map[EntityID]struct{}),
	}
}

// NewEntity returns a new unique entity ID
func (w *World) NewEntity() EntityID {
	id := w.nextID
	w.nextID++
	return id
}

// DestroyEntity removes all components for an entity
func (w *World) DestroyEntity(id EntityID) {
	delete(w.Position, id)
	delete(w.Velocity, id)
	delete(w.Movement, id)
	delete(w.Health, id)
	delete(w.Hitbox, id)
	delete(w.HitboxTrapezoid, id)
	delete(w.Facing, id)
	delete(w.AI, id)
	delete(w.Dash, id)
	delete(w.ProjectileData, id)
	delete(w.GoldData, id)
	delete(w.PlayerData, id)
	delete(w.IsPlayer, id)
	delete(w.IsEnemy, id)
	delete(w.IsProjectile, id)
	delete(w.IsGold, id)
}

// Exists checks if an entity has Position component
func (w *World) Exists(id EntityID) bool {
	_, ok := w.Position[id]
	return ok
}

// CreatePlayer creates a player entity
func (w *World) CreatePlayer(pixelX, pixelY int, hitbox HitboxTrapezoid, maxHealth int) EntityID {
	id := w.NewEntity()

	w.Position[id] = Position{X: pixelX * PositionScale, Y: pixelY * PositionScale}
	w.Velocity[id] = Velocity{}
	w.Movement[id] = Movement{}
	w.Health[id] = Health{Current: maxHealth, Max: maxHealth}
	w.HitboxTrapezoid[id] = hitbox
	w.Facing[id] = Facing{Right: true}
	w.Dash[id] = Dash{CanDash: true}
	w.PlayerData[id] = Player{
		EquippedArrows: [4]ArrowType{ArrowGray, ArrowRed, ArrowBlue, ArrowPurple},
		CurrentArrow:   ArrowGray,
	}
	w.IsPlayer[id] = struct{}{}

	w.PlayerID = id
	return id
}

// EnemyConfig holds configuration for creating an enemy
// Physics values are in IU/substep (pre-converted)
type EnemyConfig struct {
	MaxHealth     int
	ContactDamage int
	MoveSpeed     int // IU/substep
	HitboxOffsetX int
	HitboxOffsetY int
	HitboxWidth   int
	HitboxHeight  int
	AIType        AIType
	DetectRange   int // pixels
	PatrolDist    int // pixels
	AttackRange   int // pixels
	JumpForce     int // IU/substep
	Flying        bool
	GoldDropMin   int
	GoldDropMax   int
}

// CreateEnemy creates an enemy entity
func (w *World) CreateEnemy(pixelX, pixelY int, cfg EnemyConfig, facingRight bool) EntityID {
	id := w.NewEntity()

	w.Position[id] = Position{X: pixelX * PositionScale, Y: pixelY * PositionScale}
	w.Velocity[id] = Velocity{}
	w.Movement[id] = Movement{}
	w.Health[id] = Health{Current: cfg.MaxHealth, Max: cfg.MaxHealth}
	w.Hitbox[id] = Hitbox{
		OffsetX: cfg.HitboxOffsetX,
		OffsetY: cfg.HitboxOffsetY,
		Width:   cfg.HitboxWidth,
		Height:  cfg.HitboxHeight,
	}
	w.Facing[id] = Facing{Right: facingRight}
	w.AI[id] = AI{
		Type:           cfg.AIType,
		DetectRange:    cfg.DetectRange,
		AttackRange:    cfg.AttackRange,
		PatrolDistance: cfg.PatrolDist,
		JumpForce:      cfg.JumpForce,
		MoveSpeed:      cfg.MoveSpeed,
		ContactDamage:  cfg.ContactDamage,
		Flying:         cfg.Flying,
		PatrolStartX:   pixelX,
		PatrolDir:      -1,
		GoldDropMin:    cfg.GoldDropMin,
		GoldDropMax:    cfg.GoldDropMax,
	}
	w.IsEnemy[id] = struct{}{}

	return id
}

// ProjectileConfig holds configuration for creating a projectile
// All velocity values are in IU/substep (pre-converted)
type ProjectileConfig struct {
	GravityAccel  int // IU/substep²
	MaxFallSpeed  int // IU/substep
	MaxRange      int // pixels
	Damage        int
	HitboxOffsetX int
	HitboxOffsetY int
	HitboxWidth   int
	HitboxHeight  int
	StuckDuration int // frames
}

// CreateProjectile creates a projectile entity
// x, y: pixel coordinates
// vx, vy: IU/substep velocity
func (w *World) CreateProjectile(x, y int, vx, vy int, cfg ProjectileConfig, isPlayer bool) EntityID {
	id := w.NewEntity()

	w.Position[id] = Position{X: x * PositionScale, Y: y * PositionScale}
	w.Velocity[id] = Velocity{X: vx, Y: vy}
	w.Hitbox[id] = Hitbox{
		OffsetX: cfg.HitboxOffsetX,
		OffsetY: cfg.HitboxOffsetY,
		Width:   cfg.HitboxWidth,
		Height:  cfg.HitboxHeight,
	}
	w.ProjectileData[id] = Projectile{
		StartX:        x,
		GravityAccel:  cfg.GravityAccel,
		MaxFallSpeed:  cfg.MaxFallSpeed,
		MaxRange:      cfg.MaxRange,
		Damage:        cfg.Damage,
		IsPlayerOwned: isPlayer,
		StuckDuration: cfg.StuckDuration,
	}
	w.IsProjectile[id] = struct{}{}

	return id
}

// GoldConfig holds configuration for creating gold
// All velocity values are in IU/substep (pre-converted)
type GoldConfig struct {
	Gravity       int // IU/substep²
	BouncePercent int // 0-100 (percentage of velocity retained on bounce)
	CollectDelay  int // frames
	HitboxWidth   int // pixels
	HitboxHeight  int // pixels
	CollectRadius int // pixels
}

// CreateGold creates a gold pickup entity
// x, y: pixel coordinates
func (w *World) CreateGold(x, y int, amount int, cfg GoldConfig) EntityID {
	id := w.NewEntity()

	w.Position[id] = Position{X: x * PositionScale, Y: y * PositionScale}
	// Random spread velocity (IU/substep)
	// Approx: 20 pixels/sec * 256 / 600 ≈ 8.5 IU/substep
	spreadVX := ((amount % 10) - 5) * 9 // -45 to +45 IU/substep
	popVelocity := -43                  // -100 pixels/sec ≈ -43 IU/substep
	w.Velocity[id] = Velocity{X: spreadVX, Y: popVelocity}
	w.GoldData[id] = Gold{
		Amount:        amount,
		Grounded:      false,
		CollectDelay:  cfg.CollectDelay,
		Gravity:       cfg.Gravity,
		BouncePercent: cfg.BouncePercent,
		CollectRadius: cfg.CollectRadius,
		HitboxWidth:   cfg.HitboxWidth,
		HitboxHeight:  cfg.HitboxHeight,
	}
	w.IsGold[id] = struct{}{}

	return id
}

// GetPlayerPosition returns the player's position
func (w *World) GetPlayerPosition() Position {
	return w.Position[w.PlayerID]
}

// GetPlayerPixelPos returns the player's pixel position
func (w *World) GetPlayerPixelPos() (int, int) {
	pos := w.Position[w.PlayerID]
	return pos.PixelX(), pos.PixelY()
}

// CountEnemies returns the number of active enemies
func (w *World) CountEnemies() int {
	return len(w.IsEnemy)
}
