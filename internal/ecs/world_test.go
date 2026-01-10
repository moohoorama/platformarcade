package ecs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorld(t *testing.T) {
	w := NewWorld()

	assert.NotNil(t, w)
	assert.Equal(t, EntityID(1), w.nextID)
	assert.NotNil(t, w.Position)
	assert.NotNil(t, w.Velocity)
	assert.NotNil(t, w.IsPlayer)
}

func TestNewEntity(t *testing.T) {
	w := NewWorld()

	id1 := w.NewEntity()
	id2 := w.NewEntity()
	id3 := w.NewEntity()

	assert.Equal(t, EntityID(1), id1)
	assert.Equal(t, EntityID(2), id2)
	assert.Equal(t, EntityID(3), id3)
	assert.Equal(t, EntityID(4), w.nextID)
}

func TestEntityIDNeverRecycled(t *testing.T) {
	w := NewWorld()

	id1 := w.NewEntity()
	w.Position[id1] = Position{X: 100, Y: 200}

	w.DestroyEntity(id1)

	id2 := w.NewEntity()
	assert.NotEqual(t, id1, id2, "Entity IDs should never be recycled")
	assert.Equal(t, EntityID(2), id2)
}

func TestDestroyEntity(t *testing.T) {
	w := NewWorld()
	id := w.NewEntity()

	// Add components
	w.Position[id] = Position{X: 100, Y: 200}
	w.Velocity[id] = Velocity{X: 10, Y: 20}
	w.Health[id] = Health{Current: 100, Max: 100}
	w.IsEnemy[id] = struct{}{}

	require.True(t, w.Exists(id))

	// Destroy
	w.DestroyEntity(id)

	assert.False(t, w.Exists(id))
	_, hasPos := w.Position[id]
	assert.False(t, hasPos)
	_, hasVel := w.Velocity[id]
	assert.False(t, hasVel)
	_, hasHealth := w.Health[id]
	assert.False(t, hasHealth)
	_, isEnemy := w.IsEnemy[id]
	assert.False(t, isEnemy)
}

func TestExists(t *testing.T) {
	w := NewWorld()
	id := w.NewEntity()

	assert.False(t, w.Exists(id), "Entity without Position should not exist")

	w.Position[id] = Position{X: 0, Y: 0}
	assert.True(t, w.Exists(id), "Entity with Position should exist")
}

func TestPosition(t *testing.T) {
	pos := Position{X: 150 * PositionScale, Y: 200 * PositionScale} // 150px, 200px

	assert.Equal(t, 150, pos.PixelX())
	assert.Equal(t, 200, pos.PixelY())
}

func TestHealth(t *testing.T) {
	t.Run("TakeDamage", func(t *testing.T) {
		h := Health{Current: 100, Max: 100}

		dead := h.TakeDamage(30)
		assert.False(t, dead)
		assert.Equal(t, 70, h.Current)

		dead = h.TakeDamage(80)
		assert.True(t, dead)
		assert.Equal(t, -10, h.Current)
	})

	t.Run("TakeDamage with Iframe", func(t *testing.T) {
		h := Health{Current: 100, Max: 100, Iframe: 60}

		dead := h.TakeDamage(50)
		assert.False(t, dead)
		assert.Equal(t, 100, h.Current, "Should not take damage during iframe")
	})

	t.Run("Heal", func(t *testing.T) {
		h := Health{Current: 50, Max: 100}

		h.Heal(30)
		assert.Equal(t, 80, h.Current)

		h.Heal(50)
		assert.Equal(t, 100, h.Current, "Should not exceed max")
	})

	t.Run("IsAlive", func(t *testing.T) {
		h := Health{Current: 1, Max: 100}
		assert.True(t, h.IsAlive())

		h.Current = 0
		assert.False(t, h.IsAlive())

		h.Current = -10
		assert.False(t, h.IsAlive())
	})
}
