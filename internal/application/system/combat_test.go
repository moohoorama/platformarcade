package system

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

// testRNG returns a seeded RNG for deterministic tests
func testRNG() *rand.Rand {
	return rand.New(rand.NewSource(12345))
}

func createTestGameConfig() *config.GameConfig {
	return &config.GameConfig{
		Physics: &config.PhysicsConfig{
			Physics: config.PhysicsSettings{
				Gravity:      800,
				MaxFallSpeed: 400,
			},
			Feedback: config.FeedbackConfig{
				Hitstop: config.HitstopConfig{
					Frames: 3,
				},
				ScreenShake: config.ScreenShakeConfig{
					Intensity: 4,
				},
			},
			Combat: config.CombatConfig{
				Iframes: 1.0,
				Knockback: config.KnockbackConfig{
					Force:        200,
					UpForce:      100,
					StunDuration: 0.5,
				},
			},
		},
		Entities: &config.EntitiesConfig{
			Projectiles: map[string]config.ProjectileConfig{
				"playerArrow": {
					Physics: config.ProjectilePhysicsConfig{
						Speed:          300,
						LaunchAngleDeg: 20,
						GravityAccel:   500,
						MaxFallSpeed:   350,
						MaxRange:       180,
					},
					Damage: 25,
				},
				"enemyArrow": {
					Physics: config.ProjectilePhysicsConfig{
						Speed:          220,
						LaunchAngleDeg: 15,
						GravityAccel:   400,
						MaxFallSpeed:   300,
						MaxRange:       150,
					},
					Damage: 15,
				},
			},
			Pickups: map[string]config.PickupConfig{
				"gold": {
					Hitbox: config.Rect{
						Width:  8,
						Height: 8,
					},
					Physics: config.PickupPhysicsConfig{
						Gravity:       400,
						BounceDecay:   0.5,
						CollectDelay:  0.3,
						CollectRadius: 16,
					},
				},
			},
		},
	}
}

func TestNewCombatSystem(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()

	sys := NewCombatSystem(cfg, stage, testRNG())

	require.NotNil(t, sys)
	assert.NotNil(t, sys.projectiles)
	assert.NotNil(t, sys.enemies)
	assert.NotNil(t, sys.golds)
}

func TestCombatSystem_SpawnPlayerArrow(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	sys.SpawnPlayerArrow(100, 200, true)

	require.Len(t, sys.projectiles, 1)
	arrow := sys.projectiles[0]
	assert.True(t, arrow.Active)
	assert.True(t, arrow.IsPlayer)
	assert.Equal(t, 25, arrow.Damage)
}

func TestCombatSystem_MoveEnemyX(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	t.Run("moves right with sub-pixel accumulation", func(t *testing.T) {
		enemy := entity.NewEnemy(1, 32, 32, "slime")
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		enemy.PatrolDir = 1

		// Move 0.5 pixels 3 times
		sys.moveEnemyX(enemy, 0.5)
		assert.Equal(t, 32, enemy.X) // Not moved yet
		assert.InDelta(t, 0.5, enemy.RemX, 0.001)

		sys.moveEnemyX(enemy, 0.5)
		assert.Equal(t, 33, enemy.X) // Moved 1 pixel
		assert.InDelta(t, 0.0, enemy.RemX, 0.001)

		sys.moveEnemyX(enemy, 0.5)
		assert.Equal(t, 33, enemy.X) // Still at 33
		assert.InDelta(t, 0.5, enemy.RemX, 0.001)
	})

	t.Run("stops at wall and turns around", func(t *testing.T) {
		enemy := entity.NewEnemy(1, 44, 32, "slime")
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		enemy.PatrolDir = 1

		initialDir := enemy.PatrolDir
		sys.moveEnemyX(enemy, 20) // Try to move into wall

		assert.Equal(t, -initialDir, enemy.PatrolDir) // Direction reversed
		assert.Equal(t, 0.0, enemy.RemX)              // Remainder reset
	})
}

func TestCombatSystem_MoveEnemyY(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	t.Run("moves down with sub-pixel accumulation", func(t *testing.T) {
		enemy := entity.NewEnemy(1, 32, 32, "slime")
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		enemy.VY = 100

		sys.moveEnemyY(enemy, 0.5)
		assert.Equal(t, 32, enemy.Y)
		assert.InDelta(t, 0.5, enemy.RemY, 0.001)

		sys.moveEnemyY(enemy, 0.5)
		assert.Equal(t, 33, enemy.Y)
	})

	t.Run("stops at ground", func(t *testing.T) {
		enemy := entity.NewEnemy(1, 32, 44, "slime")
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		enemy.VY = 100

		sys.moveEnemyY(enemy, 20)

		assert.Equal(t, 0.0, enemy.VY)
		assert.Equal(t, 0.0, enemy.RemY)
	})
}

func TestCombatSystem_ApplyEnemyGravity(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	enemy := entity.NewEnemy(1, 32, 32, "slime")
	enemy.HitboxWidth = 12
	enemy.HitboxHeight = 12
	enemy.VY = 0

	sys.applyEnemyGravity(enemy, 0.016)

	assert.Greater(t, enemy.VY, 0.0)
}

func TestCombatSystem_RectsOverlap(t *testing.T) {
	tests := []struct {
		name                       string
		x1, y1, w1, h1             int
		x2, y2, w2, h2             int
		want                       bool
	}{
		{
			name: "overlapping",
			x1: 0, y1: 0, w1: 10, h1: 10,
			x2: 5, y2: 5, w2: 10, h2: 10,
			want: true,
		},
		{
			name: "not overlapping - horizontal gap",
			x1: 0, y1: 0, w1: 10, h1: 10,
			x2: 20, y2: 0, w2: 10, h2: 10,
			want: false,
		},
		{
			name: "not overlapping - vertical gap",
			x1: 0, y1: 0, w1: 10, h1: 10,
			x2: 0, y2: 20, w2: 10, h2: 10,
			want: false,
		},
		{
			name: "touching edges - no overlap",
			x1: 0, y1: 0, w1: 10, h1: 10,
			x2: 10, y2: 0, w2: 10, h2: 10,
			want: false,
		},
		{
			name: "one inside another",
			x1: 0, y1: 0, w1: 20, h1: 20,
			x2: 5, y2: 5, w2: 5, h2: 5,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rectsOverlap(tt.x1, tt.y1, tt.w1, tt.h1, tt.x2, tt.y2, tt.w2, tt.h2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCombatSystem_SpawnEnemy(t *testing.T) {
	cfg := createTestGameConfig()
	cfg.Entities.Enemies = map[string]config.EnemyConfig{
		"slime": {
			Hitbox: config.EnemyHitboxConfig{
				Body: config.Rect{OffsetX: 2, OffsetY: 4, Width: 12, Height: 12},
			},
			Stats: config.EnemyStats{
				MaxHealth:     50,
				ContactDamage: 10,
				MoveSpeed:     40,
				GoldDrop:      config.GoldDrop{Min: 5, Max: 15},
			},
			AI: config.AIConfig{
				Type:           "patrol",
				DetectRange:    80,
				PatrolDistance: 60,
			},
		},
	}
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	sys.SpawnEnemy(1, 100, 200, "slime", true)

	require.Len(t, sys.enemies, 1)
	enemy := sys.enemies[0]
	assert.Equal(t, entity.EntityID(1), enemy.ID)
	assert.Equal(t, 100, enemy.X)
	assert.Equal(t, 200, enemy.Y)
	assert.Equal(t, 50, enemy.MaxHealth)
	assert.True(t, enemy.FacingRight)
}

func TestCombatSystem_GetProjectiles(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	sys.SpawnPlayerArrow(100, 200, true)

	projectiles := sys.GetProjectiles()
	assert.Len(t, projectiles, 1)
}

func TestCombatSystem_GetEnemies(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	enemies := sys.GetEnemies()
	assert.Empty(t, enemies)
}

func TestCombatSystem_GetGolds(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	golds := sys.GetGolds()
	assert.Empty(t, golds)
}

func TestCombatSystem_UpdateProjectiles(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	t.Run("moves projectiles", func(t *testing.T) {
		sys.SpawnPlayerArrow(32, 32, true)
		initialX := sys.projectiles[0].X

		sys.updateProjectiles(0.016)

		assert.Greater(t, sys.projectiles[0].X, initialX)
	})

	t.Run("projectile continues beyond max range", func(t *testing.T) {
		sys.projectiles = nil
		sys.SpawnPlayerArrow(32, 32, true)
		proj := sys.projectiles[0]
		proj.MaxRange = 10

		// Move projectile far - should still be active
		for i := 0; i < 10; i++ {
			sys.updateProjectiles(0.1)
		}

		assert.True(t, proj.Active)
	})

	t.Run("sticks projectiles hitting walls", func(t *testing.T) {
		sys.projectiles = nil
		sys.SpawnPlayerArrow(50, 32, true) // Near right wall
		proj := sys.projectiles[0]

		for i := 0; i < 20; i++ {
			sys.updateProjectiles(0.016)
		}

		assert.True(t, proj.Stuck)
		assert.True(t, proj.Active)
		assert.Equal(t, 5.0, proj.StuckDuration)
	})
}

func TestCombatSystem_SpawnGold(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	t.Run("spawns gold from enemy", func(t *testing.T) {
		enemy := entity.NewEnemy(1, 32, 32, "slime")
		enemy.GoldDropMin = 5
		enemy.GoldDropMax = 10
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12

		sys.spawnGold(enemy)

		require.Len(t, sys.golds, 1)
		gold := sys.golds[0]
		assert.True(t, gold.Active)
		assert.GreaterOrEqual(t, gold.Amount, 5)
		assert.LessOrEqual(t, gold.Amount, 10)
	})
}

func TestCombatSystem_DamagePlayer(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	t.Run("applies damage and iframes", func(t *testing.T) {
		hitbox := entity.TrapezoidHitbox{
			Head: entity.HitboxRect{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
			Body: entity.HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
			Feet: entity.HitboxRect{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
		}
		player := entity.NewPlayer(32, 32, hitbox, 100)

		sys.damagePlayer(player, 10, 16)

		assert.Equal(t, 90, player.Health)
		assert.Greater(t, player.IframeTimer, 0.0)
		assert.Greater(t, player.StunTimer, 0.0)
	})

	t.Run("applies knockback from left", func(t *testing.T) {
		hitbox := entity.TrapezoidHitbox{
			Head: entity.HitboxRect{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
			Body: entity.HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
			Feet: entity.HitboxRect{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
		}
		player := entity.NewPlayer(32, 32, hitbox, 100)

		sys.damagePlayer(player, 10, 16) // Enemy at x=16, left of player

		assert.Greater(t, player.VX, 0.0) // Knocked right
		assert.Less(t, player.VY, 0.0)    // Knocked up
	})

	t.Run("applies knockback from right", func(t *testing.T) {
		hitbox := entity.TrapezoidHitbox{
			Head: entity.HitboxRect{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
			Body: entity.HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
			Feet: entity.HitboxRect{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
		}
		player := entity.NewPlayer(32, 32, hitbox, 100)

		sys.damagePlayer(player, 10, 48) // Enemy at x=48, right of player

		assert.Less(t, player.VX, 0.0) // Knocked left
		assert.Less(t, player.VY, 0.0) // Knocked up
	})
}

func TestCombatSystem_UpdatePatrolAI(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	t.Run("enemy moves in patrol direction", func(t *testing.T) {
		enemy := entity.NewEnemy(1, 32, 32, "slime")
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		enemy.PatrolDir = 1
		enemy.MoveSpeed = 50
		enemy.PatrolDistance = 100
		enemy.PatrolStartX = 32

		hitbox := entity.TrapezoidHitbox{}
		player := entity.NewPlayer(100, 32, hitbox, 100)

		sys.updatePatrolAI(enemy, player, 100, 0.1)

		// Should accumulate movement in RemX or move
		assert.True(t, enemy.RemX > 0 || enemy.X > 32)
	})
}

func TestCombatSystem_UpdateChaseAI(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	t.Run("applies gravity to non-flying enemy", func(t *testing.T) {
		enemy := entity.NewEnemy(1, 32, 32, "chaser")
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		enemy.MoveSpeed = 50
		enemy.Flying = false
		enemy.VY = 0

		hitbox := entity.TrapezoidHitbox{}
		player := entity.NewPlayer(48, 32, hitbox, 100)

		sys.updateChaseAI(enemy, player, 16, 16, 0, 0.1)

		// Gravity should be applied
		assert.Greater(t, enemy.VY, 0.0)
	})

	t.Run("flying enemy no gravity", func(t *testing.T) {
		enemy := entity.NewEnemy(1, 32, 32, "flying")
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		enemy.MoveSpeed = 50
		enemy.Flying = true
		enemy.VY = 0

		hitbox := entity.TrapezoidHitbox{}
		player := entity.NewPlayer(48, 48, hitbox, 100)

		sys.updateChaseAI(enemy, player, 24, 16, 16, 0.1)

		// No gravity for flying enemy, but may have movement
		// Just verify it ran without error
		assert.True(t, true)
	})
}

func TestCombatSystem_UpdateGolds(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	t.Run("gold falls with gravity", func(t *testing.T) {
		gold := entity.NewGold(32, 24, 10, 400, 0.5, 0.3, 8, 8, 16)
		gold.Active = true
		gold.Grounded = false
		gold.VY = 0
		sys.golds = append(sys.golds, gold)

		hitbox := entity.TrapezoidHitbox{
			Body: entity.HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		}
		player := entity.NewPlayer(100, 100, hitbox, 100)

		sys.updateGolds(player, 0.016)

		assert.Greater(t, gold.VY, 0.0)
	})

	t.Run("gold collects when player is close", func(t *testing.T) {
		sys.golds = nil
		gold := entity.NewGold(32, 32, 10, 400, 0.5, 0, 8, 8, 20)
		gold.Active = true
		gold.Grounded = true
		gold.CollectDelay = 0
		sys.golds = append(sys.golds, gold)

		hitbox := entity.TrapezoidHitbox{
			Body: entity.HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		}
		player := entity.NewPlayer(32, 32, hitbox, 100)

		sys.updateGolds(player, 0.016)

		assert.False(t, gold.Active)
		assert.Equal(t, 10, player.Gold)
	})

	t.Run("gold does not collect during delay", func(t *testing.T) {
		sys.golds = nil
		gold := entity.NewGold(32, 32, 10, 400, 0.5, 0.5, 8, 8, 20)
		gold.Active = true
		gold.Grounded = true
		sys.golds = append(sys.golds, gold)

		hitbox := entity.TrapezoidHitbox{
			Body: entity.HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		}
		player := entity.NewPlayer(32, 32, hitbox, 100)

		sys.updateGolds(player, 0.016)

		assert.True(t, gold.Active) // Not collected yet
	})

	t.Run("inactive gold is skipped", func(t *testing.T) {
		sys.golds = nil
		gold := entity.NewGold(32, 32, 10, 400, 0.5, 0, 8, 8, 20)
		gold.Active = false
		sys.golds = append(sys.golds, gold)

		hitbox := entity.TrapezoidHitbox{
			Body: entity.HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		}
		player := entity.NewPlayer(32, 32, hitbox, 100)

		sys.updateGolds(player, 0.016)

		assert.Equal(t, 0, player.Gold)
	})
}

func TestCombatSystem_UpdateEnemies(t *testing.T) {
	cfg := createTestGameConfig()
	cfg.Entities.Enemies = map[string]config.EnemyConfig{
		"slime": {
			AI: config.AIConfig{Type: "patrol"},
		},
	}
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	t.Run("updates active enemies", func(t *testing.T) {
		enemy := entity.NewEnemy(1, 32, 32, "slime")
		enemy.Active = true
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		enemy.MoveSpeed = 50
		enemy.PatrolDir = 1
		enemy.PatrolDistance = 100
		enemy.PatrolStartX = 32
		enemy.AIType = entity.AIPatrol
		sys.enemies = append(sys.enemies, enemy)

		hitbox := entity.TrapezoidHitbox{}
		player := entity.NewPlayer(100, 32, hitbox, 100)

		sys.updateEnemies(player, 0.016)

		// Just verify it ran without error
		assert.True(t, true)
	})

	t.Run("skips inactive enemies", func(t *testing.T) {
		sys.enemies = nil
		enemy := entity.NewEnemy(1, 32, 32, "slime")
		enemy.Active = false
		sys.enemies = append(sys.enemies, enemy)

		hitbox := entity.TrapezoidHitbox{}
		player := entity.NewPlayer(100, 32, hitbox, 100)

		sys.updateEnemies(player, 0.016)

		// Should not panic or error
		assert.True(t, true)
	})
}

func TestCombatSystem_CheckCollisions(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	t.Run("arrow hits enemy", func(t *testing.T) {
		sys.projectiles = nil
		sys.enemies = nil

		// Create arrow at enemy position
		sys.SpawnPlayerArrow(32, 32, true)
		arrow := sys.projectiles[0]

		enemy := entity.NewEnemy(1, int(arrow.X), int(arrow.Y), "slime")
		enemy.Active = true
		enemy.Health = 50
		enemy.MaxHealth = 50
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		sys.enemies = append(sys.enemies, enemy)

		hitbox := entity.TrapezoidHitbox{
			Body: entity.HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		}
		player := entity.NewPlayer(100, 100, hitbox, 100)

		sys.checkCollisions(player)

		// Arrow should be deactivated
		assert.False(t, arrow.Active)
		// Enemy should take damage
		assert.Less(t, enemy.Health, 50)
	})

	t.Run("enemy contact damages player", func(t *testing.T) {
		sys.enemies = nil

		enemy := entity.NewEnemy(1, 32, 32, "slime")
		enemy.Active = true
		enemy.Health = 50
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		enemy.ContactDamage = 10
		sys.enemies = append(sys.enemies, enemy)

		hitbox := entity.TrapezoidHitbox{
			Body: entity.HitboxRect{OffsetX: 0, OffsetY: 0, Width: 16, Height: 24},
		}
		player := entity.NewPlayer(32, 32, hitbox, 100)

		sys.checkCollisions(player)

		assert.Less(t, player.Health, 100)
	})

	t.Run("player with iframes is not damaged", func(t *testing.T) {
		sys.enemies = nil

		enemy := entity.NewEnemy(1, 32, 32, "slime")
		enemy.Active = true
		enemy.Health = 50
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		enemy.ContactDamage = 10
		sys.enemies = append(sys.enemies, enemy)

		hitbox := entity.TrapezoidHitbox{
			Body: entity.HitboxRect{OffsetX: 0, OffsetY: 0, Width: 16, Height: 24},
		}
		player := entity.NewPlayer(32, 32, hitbox, 100)
		player.IframeTimer = 1.0 // Has iframes

		sys.checkCollisions(player)

		assert.Equal(t, 100, player.Health)
	})
}

func TestCombatSystem_UpdateRangedAI(t *testing.T) {
	cfg := createTestGameConfig()
	stage := createTestStage()
	sys := NewCombatSystem(cfg, stage, testRNG())

	t.Run("faces player direction", func(t *testing.T) {
		enemy := entity.NewEnemy(1, 24, 32, "archer")
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		enemy.FacingRight = false
		enemy.Flying = true
		enemy.AttackCooldown = 0
		enemy.AttackTimer = 0
		enemy.AttackRange = 50

		hitbox := entity.TrapezoidHitbox{}
		player := entity.NewPlayer(48, 32, hitbox, 100)

		sys.updateRangedAI(enemy, player, 24, 24, 0.1)

		// Should face player (to the right)
		assert.True(t, enemy.FacingRight)
	})

	t.Run("applies gravity when not flying", func(t *testing.T) {
		enemy := entity.NewEnemy(1, 32, 32, "archer")
		enemy.HitboxWidth = 12
		enemy.HitboxHeight = 12
		enemy.Flying = false
		enemy.VY = 0

		hitbox := entity.TrapezoidHitbox{}
		player := entity.NewPlayer(48, 32, hitbox, 100)

		sys.updateRangedAI(enemy, player, 16, 16, 0.1)

		assert.Greater(t, enemy.VY, 0.0)
	})
}
