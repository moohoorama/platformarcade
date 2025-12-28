package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnemy(t *testing.T) {
	enemy := NewEnemy(1, 100, 200, "slime")

	require.NotNil(t, enemy)
	assert.Equal(t, EntityID(1), enemy.ID)
	assert.Equal(t, 100, enemy.X)
	assert.Equal(t, 200, enemy.Y)
	assert.Equal(t, "slime", enemy.EnemyType)
	assert.True(t, enemy.Active)
	assert.Equal(t, 100, enemy.PatrolStartX)
	assert.Equal(t, -1, enemy.PatrolDir)
}

func TestEnemy_TakeDamage(t *testing.T) {
	enemy := NewEnemy(1, 0, 0, "slime")
	enemy.Health = 50
	enemy.MaxHealth = 50

	// Take non-lethal damage
	killed := enemy.TakeDamage(20)
	assert.False(t, killed)
	assert.Equal(t, 30, enemy.Health)
	assert.InDelta(t, 0.2, enemy.HitTimer, 0.001)

	// Take lethal damage
	killed = enemy.TakeDamage(30)
	assert.True(t, killed)
	assert.Equal(t, 0, enemy.Health)
}

func TestEnemy_IsAlive(t *testing.T) {
	enemy := NewEnemy(1, 0, 0, "slime")
	enemy.Health = 50

	assert.True(t, enemy.IsAlive())

	enemy.Health = 0
	assert.False(t, enemy.IsAlive())

	enemy.Health = 50
	enemy.Active = false
	assert.False(t, enemy.IsAlive())
}

func TestEnemy_GetHitbox(t *testing.T) {
	enemy := NewEnemy(1, 100, 200, "slime")
	enemy.HitboxOffsetX = 2
	enemy.HitboxOffsetY = 4
	enemy.HitboxWidth = 12
	enemy.HitboxHeight = 12

	x, y, w, h := enemy.GetHitbox()

	assert.Equal(t, 102, x)
	assert.Equal(t, 204, y)
	assert.Equal(t, 12, w)
	assert.Equal(t, 12, h)
}

func TestNewGold(t *testing.T) {
	gold := NewGold(100, 200, 50, 400, 0.5, 0.3, 8, 8, 16)

	require.NotNil(t, gold)
	assert.Equal(t, 100.0, gold.X)
	assert.Equal(t, 200.0, gold.Y)
	assert.Equal(t, 50, gold.Amount)
	assert.Equal(t, 400.0, gold.Gravity)
	assert.Equal(t, 0.5, gold.BounceDecay)
	assert.Equal(t, 0.3, gold.CollectDelay)
	assert.Equal(t, 8, gold.HitboxWidth)
	assert.Equal(t, 8, gold.HitboxHeight)
	assert.Equal(t, 16.0, gold.CollectRadius)
	assert.True(t, gold.Active)
	assert.Equal(t, -100.0, gold.VY) // Pop up
}

func TestGold_CanCollect(t *testing.T) {
	gold := NewGold(0, 0, 10, 400, 0.5, 0.3, 8, 8, 16)

	// Cannot collect while delay is active
	assert.False(t, gold.CanCollect())

	// Can collect after delay expires
	gold.CollectDelay = 0
	assert.True(t, gold.CanCollect())

	// Cannot collect if inactive
	gold.Active = false
	assert.False(t, gold.CanCollect())
}
