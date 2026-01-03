package system

import (
	"math"
	"math/rand"

	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

// CombatSystem handles combat interactions
type CombatSystem struct {
	config      *config.GameConfig
	stage       *entity.Stage
	projectiles []*entity.Projectile
	enemies     []*entity.Enemy
	golds       []*entity.Gold

	// Event callbacks
	OnHitstop    func(frames int)
	OnScreenShake func(intensity float64)
}

// NewCombatSystem creates a new combat system
func NewCombatSystem(cfg *config.GameConfig, stage *entity.Stage) *CombatSystem {
	return &CombatSystem{
		config:      cfg,
		stage:       stage,
		projectiles: make([]*entity.Projectile, 0, 32),
		enemies:     make([]*entity.Enemy, 0, 32),
		golds:       make([]*entity.Gold, 0, 64),
	}
}

// SpawnPlayerArrow spawns an arrow from the player
func (s *CombatSystem) SpawnPlayerArrow(x, y float64, facingRight bool) {
	arrowCfg := s.config.Entities.Projectiles["playerArrow"]

	arrow := entity.NewArrow(
		x, y,
		facingRight,
		arrowCfg.Physics.Speed,
		arrowCfg.Physics.LaunchAngleDeg,
		arrowCfg.Physics.GravityAccel,
		arrowCfg.Physics.MaxFallSpeed,
		arrowCfg.Physics.MaxRange,
		arrowCfg.Damage,
		true, // isPlayer
	)

	s.projectiles = append(s.projectiles, arrow)
}

// SpawnPlayerArrowToward spawns an arrow toward a target position
func (s *CombatSystem) SpawnPlayerArrowToward(x, y, targetX, targetY float64) {
	arrowCfg := s.config.Entities.Projectiles["playerArrow"]

	arrow := entity.NewArrowDirected(
		x, y,
		targetX, targetY,
		arrowCfg.Physics.Speed,
		arrowCfg.Physics.GravityAccel,
		arrowCfg.Physics.MaxFallSpeed,
		arrowCfg.Physics.MaxRange,
		arrowCfg.Damage,
		true, // isPlayer
	)

	s.projectiles = append(s.projectiles, arrow)
}

// GetArrowConfig returns the player arrow configuration
func (s *CombatSystem) GetArrowConfig() (speed, gravity, maxFall, maxRange float64) {
	arrowCfg := s.config.Entities.Projectiles["playerArrow"]
	return arrowCfg.Physics.Speed, arrowCfg.Physics.GravityAccel, arrowCfg.Physics.MaxFallSpeed, arrowCfg.Physics.MaxRange
}

// SpawnEnemy spawns an enemy at the given position
func (s *CombatSystem) SpawnEnemy(id entity.EntityID, x, y int, enemyType string, facingRight bool) {
	enemyCfg, ok := s.config.Entities.Enemies[enemyType]
	if !ok {
		return
	}

	enemy := entity.NewEnemy(id, x, y, enemyType)
	enemy.MaxHealth = enemyCfg.Stats.MaxHealth
	enemy.Health = enemyCfg.Stats.MaxHealth
	enemy.ContactDamage = enemyCfg.Stats.ContactDamage
	enemy.MoveSpeed = enemyCfg.Stats.MoveSpeed
	enemy.FacingRight = facingRight
	enemy.GoldDropMin = enemyCfg.Stats.GoldDrop.Min
	enemy.GoldDropMax = enemyCfg.Stats.GoldDrop.Max

	enemy.HitboxOffsetX = enemyCfg.Hitbox.Body.OffsetX
	enemy.HitboxOffsetY = enemyCfg.Hitbox.Body.OffsetY
	enemy.HitboxWidth = enemyCfg.Hitbox.Body.Width
	enemy.HitboxHeight = enemyCfg.Hitbox.Body.Height

	switch enemyCfg.AI.Type {
	case "patrol":
		enemy.AIType = entity.AIPatrol
	case "ranged":
		enemy.AIType = entity.AIRanged
	case "chase":
		enemy.AIType = entity.AIChase
	}

	enemy.DetectRange = enemyCfg.AI.DetectRange
	enemy.PatrolDistance = enemyCfg.AI.PatrolDistance
	enemy.AttackRange = enemyCfg.AI.AttackRange
	enemy.AttackCooldown = enemyCfg.AI.AttackCooldown
	enemy.Flying = enemyCfg.AI.Flying

	s.enemies = append(s.enemies, enemy)
}

// Update updates all combat entities
func (s *CombatSystem) Update(player *entity.Player, dt float64) {
	s.updateProjectiles(dt)
	s.updateEnemies(player, dt)
	s.updateGolds(player, dt)
	s.checkCollisions(player)
}

func (s *CombatSystem) updateProjectiles(dt float64) {
	for _, proj := range s.projectiles {
		if !proj.Active {
			continue
		}

		proj.Update(dt)

		// Skip movement if stuck
		if proj.Stuck {
			continue
		}

		// Get pixels to move this frame
		dx, dy := proj.ApplyVelocity(dt)

		// Move with 1-pixel substeps
		s.moveProjectile(proj, dx, dy)
	}
}

// moveProjectile moves projectile with 1-pixel substeps
func (s *CombatSystem) moveProjectile(proj *entity.Projectile, dx, dy int) {
	// Determine the number of steps (use the larger of abs(dx) or abs(dy))
	stepsX := dx
	stepsY := dy
	if stepsX < 0 {
		stepsX = -stepsX
	}
	if stepsY < 0 {
		stepsY = -stepsY
	}

	totalSteps := stepsX
	if stepsY > totalSteps {
		totalSteps = stepsY
	}

	if totalSteps == 0 {
		return
	}

	// Calculate step direction
	stepX := 0.0
	stepY := 0.0
	if dx != 0 || dy != 0 {
		stepX = float64(dx) / float64(totalSteps)
		stepY = float64(dy) / float64(totalSteps)
	}

	// Move 1 step at a time
	accumX := 0.0
	accumY := 0.0
	for i := 0; i < totalSteps; i++ {
		accumX += stepX
		accumY += stepY

		// Move by integer pixels
		moveX := int(accumX)
		moveY := int(accumY)
		accumX -= float64(moveX)
		accumY -= float64(moveY)

		proj.X += float64(moveX)
		proj.Y += float64(moveY)

		// Check collision at arrow tip
		if s.stage.IsSolidAt(int(proj.X), int(proj.Y)) {
			proj.StickToWall(5.0)
			return
		}
	}
}

func (s *CombatSystem) updateEnemies(player *entity.Player, dt float64) {
	for _, enemy := range s.enemies {
		if !enemy.Active {
			continue
		}

		// Hit stun
		if enemy.HitTimer > 0 {
			enemy.HitTimer -= dt
			continue
		}

		// Attack cooldown
		if enemy.AttackTimer > 0 {
			enemy.AttackTimer -= dt
		}

		// Calculate distance to player
		dx := float64(player.X - enemy.X)
		dy := float64(player.Y - enemy.Y)
		dist := math.Sqrt(dx*dx + dy*dy)

		switch enemy.AIType {
		case entity.AIPatrol:
			s.updatePatrolAI(enemy, player, dist, dt)
		case entity.AIRanged:
			s.updateRangedAI(enemy, player, dist, dx, dt)
		case entity.AIChase:
			s.updateChaseAI(enemy, player, dist, dx, dy, dt)
		}
	}
}

func (s *CombatSystem) updatePatrolAI(enemy *entity.Enemy, player *entity.Player, dist float64, dt float64) {
	// Simple patrol: walk back and forth with substep collision
	moveX := float64(enemy.PatrolDir) * enemy.MoveSpeed * dt
	s.moveEnemyX(enemy, moveX)

	// Turn around at patrol bounds
	if math.Abs(float64(enemy.X-enemy.PatrolStartX)) > enemy.PatrolDistance {
		enemy.PatrolDir *= -1
		enemy.FacingRight = enemy.PatrolDir > 0
	}

	// Apply gravity if not flying
	if !enemy.Flying {
		s.applyEnemyGravity(enemy, dt)
	}
}

func (s *CombatSystem) updateRangedAI(enemy *entity.Enemy, player *entity.Player, dist, dx float64, dt float64) {
	// Face player
	enemy.FacingRight = dx > 0

	// Apply gravity if not flying
	if !enemy.Flying {
		s.applyEnemyGravity(enemy, dt)
	}

	// Shoot if in range and cooldown ready
	if dist < enemy.AttackRange && enemy.AttackTimer <= 0 {
		// Spawn enemy arrow
		arrowCfg := s.config.Entities.Projectiles["enemyArrow"]
		arrow := entity.NewArrow(
			float64(enemy.X+8), float64(enemy.Y+8),
			enemy.FacingRight,
			arrowCfg.Physics.Speed,
			arrowCfg.Physics.LaunchAngleDeg,
			arrowCfg.Physics.GravityAccel,
			arrowCfg.Physics.MaxFallSpeed,
			arrowCfg.Physics.MaxRange,
			arrowCfg.Damage,
			false, // isPlayer
		)
		s.projectiles = append(s.projectiles, arrow)
		enemy.AttackTimer = enemy.AttackCooldown
	}
}

func (s *CombatSystem) updateChaseAI(enemy *entity.Enemy, player *entity.Player, dist, dx, dy float64, dt float64) {
	// Apply gravity if not flying
	if !enemy.Flying {
		s.applyEnemyGravity(enemy, dt)
	}

	if dist > enemy.DetectRange {
		return
	}

	// Chase player with substep collision
	speed := enemy.MoveSpeed * dt

	if dx > 0 {
		s.moveEnemyX(enemy, speed)
		enemy.FacingRight = true
	} else if dx < 0 {
		s.moveEnemyX(enemy, -speed)
		enemy.FacingRight = false
	}

	if enemy.Flying {
		if dy > 0 {
			s.moveEnemyY(enemy, speed)
		} else if dy < 0 {
			s.moveEnemyY(enemy, -speed)
		}
	}
}

// moveEnemyX moves enemy horizontally with substep collision and sub-pixel accumulation
func (s *CombatSystem) moveEnemyX(enemy *entity.Enemy, moveX float64) {
	// Accumulate sub-pixel movement
	moveX += enemy.RemX
	pixelsToMove := int(moveX)
	enemy.RemX = moveX - float64(pixelsToMove)

	if pixelsToMove == 0 {
		return
	}

	step := 1
	if pixelsToMove < 0 {
		step = -1
		pixelsToMove = -pixelsToMove
	}

	for i := 0; i < pixelsToMove; i++ {
		// Check collision at new position
		newX := enemy.X + step
		ex, ey := newX+enemy.HitboxOffsetX, enemy.Y+enemy.HitboxOffsetY
		ew, eh := enemy.HitboxWidth, enemy.HitboxHeight

		// Check leading edge
		checkX := ex
		if step > 0 {
			checkX = ex + ew - 1
		}

		if s.stage.IsSolidAt(checkX, ey) || s.stage.IsSolidAt(checkX, ey+eh-1) || s.stage.IsSolidAt(checkX, ey+eh/2) {
			// Hit wall, turn around for patrol AI
			enemy.PatrolDir *= -1
			enemy.FacingRight = enemy.PatrolDir > 0
			enemy.RemX = 0 // Reset remainder on collision
			return
		}
		enemy.X = newX
	}
}

// moveEnemyY moves enemy vertically with substep collision and sub-pixel accumulation
func (s *CombatSystem) moveEnemyY(enemy *entity.Enemy, moveY float64) {
	// Accumulate sub-pixel movement
	moveY += enemy.RemY
	pixelsToMove := int(moveY)
	enemy.RemY = moveY - float64(pixelsToMove)

	if pixelsToMove == 0 {
		return
	}

	step := 1
	if pixelsToMove < 0 {
		step = -1
		pixelsToMove = -pixelsToMove
	}

	for i := 0; i < pixelsToMove; i++ {
		newY := enemy.Y + step
		ex, ey := enemy.X+enemy.HitboxOffsetX, newY+enemy.HitboxOffsetY
		ew, eh := enemy.HitboxWidth, enemy.HitboxHeight

		// Check leading edge
		checkY := ey
		if step > 0 {
			checkY = ey + eh - 1
		}

		if s.stage.IsSolidAt(ex, checkY) || s.stage.IsSolidAt(ex+ew-1, checkY) || s.stage.IsSolidAt(ex+ew/2, checkY) {
			enemy.VY = 0
			enemy.RemY = 0 // Reset remainder on collision
			return
		}
		enemy.Y = newY
	}
}

// applyEnemyGravity applies gravity to non-flying enemies
func (s *CombatSystem) applyEnemyGravity(enemy *entity.Enemy, dt float64) {
	gravity := s.config.Physics.Physics.Gravity
	maxFall := s.config.Physics.Physics.MaxFallSpeed

	enemy.VY += gravity * dt
	if enemy.VY > maxFall {
		enemy.VY = maxFall
	}

	// Apply Y movement with collision
	moveY := enemy.VY * dt
	s.moveEnemyY(enemy, moveY)
}

func (s *CombatSystem) updateGolds(player *entity.Player, dt float64) {
	for _, gold := range s.golds {
		if !gold.Active {
			continue
		}

		// Update collect delay timer
		if gold.CollectDelay > 0 {
			gold.CollectDelay -= dt
		}

		hbW := gold.HitboxWidth
		hbH := gold.HitboxHeight

		if !gold.Grounded {
			// Apply gravity
			gold.VY += gold.Gravity * dt

			// Substep movement for X
			moveX := gold.VX * dt
			stepX := 1.0
			if moveX < 0 {
				stepX = -1.0
			}
			for math.Abs(moveX) >= 1.0 {
				newX := gold.X + stepX
				if s.stage.IsSolidAt(int(newX), int(gold.Y)) ||
					s.stage.IsSolidAt(int(newX), int(gold.Y)+hbH-1) {
					// Hit wall, bounce
					gold.VX = -gold.VX * gold.BounceDecay
					break
				}
				gold.X = newX
				moveX -= stepX
			}

			// Substep movement for Y
			moveY := gold.VY * dt
			stepY := 1.0
			if moveY < 0 {
				stepY = -1.0
			}
			for math.Abs(moveY) >= 1.0 {
				newY := gold.Y + stepY
				if s.stage.IsSolidAt(int(gold.X), int(newY)+hbH-1) ||
					s.stage.IsSolidAt(int(gold.X)+hbW-1, int(newY)+hbH-1) {
					if stepY > 0 {
						// Hit ground
						gold.Grounded = true
						gold.VY = 0
						gold.VX = 0
					} else {
						// Hit ceiling, bounce
						gold.VY = -gold.VY * gold.BounceDecay
					}
					break
				}
				gold.Y = newY
				moveY -= stepY
			}
		}

		// Collect if player touches
		if gold.CanCollect() {
			// Player center (using body hitbox) - use PixelX/PixelY since gold is in pixels
			px := float64(player.PixelX() + player.Hitbox.Body.OffsetX + player.Hitbox.Body.Width/2)
			py := float64(player.PixelY() + player.Hitbox.Body.OffsetY + player.Hitbox.Body.Height/2)
			// Gold center
			gx := gold.X + float64(hbW)/2
			gy := gold.Y + float64(hbH)/2

			dist := math.Sqrt((px-gx)*(px-gx) + (py-gy)*(py-gy))
			if dist < gold.CollectRadius {
				player.Gold += gold.Amount
				gold.Active = false
			}
		}
	}
}

func (s *CombatSystem) checkCollisions(player *entity.Player) {
	// Player arrows vs enemies
	for _, proj := range s.projectiles {
		if !proj.Active || !proj.IsPlayer {
			continue
		}

		ax, ay, aw, ah := proj.GetHitbox()

		for _, enemy := range s.enemies {
			if !enemy.Active {
				continue
			}

			ex, ey, ew, eh := enemy.GetHitbox()

			if rectsOverlap(ax, ay, aw, ah, ex, ey, ew, eh) {
				proj.Deactivate()
				killed := enemy.TakeDamage(proj.Damage)

				// Feedback
				if s.OnHitstop != nil {
					s.OnHitstop(s.config.Physics.Feedback.Hitstop.Frames)
				}
				if s.OnScreenShake != nil {
					s.OnScreenShake(s.config.Physics.Feedback.ScreenShake.Intensity)
				}

				if killed {
					enemy.Active = false
					s.spawnGold(enemy)
				}
				break
			}
		}
	}

	// Enemy projectiles vs player
	if !player.IsInvincible() {
		for _, proj := range s.projectiles {
			if !proj.Active || proj.IsPlayer {
				continue
			}

			ax, ay, aw, ah := proj.GetHitbox()
			// Use PixelX/PixelY since projectiles are in pixels
			px := player.PixelX() + player.Hitbox.Body.OffsetX
			py := player.PixelY() + player.Hitbox.Body.OffsetY
			pw := player.Hitbox.Body.Width
			ph := player.Hitbox.Body.Height

			if rectsOverlap(ax, ay, aw, ah, px, py, pw, ph) {
				proj.Deactivate()
				s.damagePlayer(player, proj.Damage, int(proj.X))
			}
		}
	}

	// Enemy contact vs player
	if !player.IsInvincible() {
		for _, enemy := range s.enemies {
			if !enemy.Active {
				continue
			}

			ex, ey, ew, eh := enemy.GetHitbox()
			// Use PixelX/PixelY since enemies are in pixels
			px := player.PixelX() + player.Hitbox.Body.OffsetX
			py := player.PixelY() + player.Hitbox.Body.OffsetY
			pw := player.Hitbox.Body.Width
			ph := player.Hitbox.Body.Height

			if rectsOverlap(ex, ey, ew, eh, px, py, pw, ph) {
				s.damagePlayer(player, enemy.ContactDamage, enemy.X)
			}
		}
	}
}

func (s *CombatSystem) damagePlayer(player *entity.Player, damage int, fromX int) {
	player.Health -= damage
	player.IframeTimer = s.config.Physics.Combat.Iframes
	player.StunTimer = s.config.Physics.Combat.Knockback.StunDuration

	// Knockback - compare in same scale (fromX is in pixels, player.X is 100x scaled)
	dir := 1.0
	if fromX > player.PixelX() {
		dir = -1.0
	}
	// VX is in 100x scale, config force is in pixels/sec
	player.VX = dir * s.config.Physics.Combat.Knockback.Force * entity.PositionScale
	player.VY = -s.config.Physics.Combat.Knockback.UpForce * entity.PositionScale

	if s.OnScreenShake != nil {
		s.OnScreenShake(s.config.Physics.Feedback.ScreenShake.Intensity * 1.5)
	}
}

func (s *CombatSystem) spawnGold(enemy *entity.Enemy) {
	amount := enemy.GoldDropMin
	if enemy.GoldDropMax > enemy.GoldDropMin {
		amount += rand.Intn(enemy.GoldDropMax - enemy.GoldDropMin)
	}

	pickupCfg := s.config.Entities.Pickups["gold"]
	gold := entity.NewGold(
		float64(enemy.X+8), float64(enemy.Y),
		amount,
		pickupCfg.Physics.Gravity,
		pickupCfg.Physics.BounceDecay,
		pickupCfg.Physics.CollectDelay,
		pickupCfg.Hitbox.Width,
		pickupCfg.Hitbox.Height,
		pickupCfg.Physics.CollectRadius,
	)
	s.golds = append(s.golds, gold)
}

// GetProjectiles returns all active projectiles
func (s *CombatSystem) GetProjectiles() []*entity.Projectile {
	return s.projectiles
}

// GetEnemies returns all active enemies
func (s *CombatSystem) GetEnemies() []*entity.Enemy {
	return s.enemies
}

// GetGolds returns all active golds
func (s *CombatSystem) GetGolds() []*entity.Gold {
	return s.golds
}

// Helper function
func rectsOverlap(x1, y1, w1, h1, x2, y2, w2, h2 int) bool {
	return x1 < x2+w2 && x1+w1 > x2 && y1 < y2+h2 && y1+h1 > y2
}
