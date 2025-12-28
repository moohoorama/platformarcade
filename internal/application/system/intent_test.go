package system

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/younwookim/mg/internal/domain/entity"
)

func TestMoveIntent(t *testing.T) {
	intent := MoveIntent{
		EntityID: entity.EntityID(1),
		DX:       10,
		DY:       -5,
	}

	// Test that it implements Intent interface
	var i Intent = intent
	i.isIntent() // Should not panic

	assert.Equal(t, entity.EntityID(1), intent.EntityID)
	assert.Equal(t, 10, intent.DX)
	assert.Equal(t, -5, intent.DY)
}

func TestJumpIntent(t *testing.T) {
	intent := JumpIntent{
		EntityID: entity.EntityID(2),
		Force:    280.5,
	}

	// Test that it implements Intent interface
	var i Intent = intent
	i.isIntent() // Should not panic

	assert.Equal(t, entity.EntityID(2), intent.EntityID)
	assert.Equal(t, 280.5, intent.Force)
}

func TestDashIntent(t *testing.T) {
	intent := DashIntent{
		EntityID:  entity.EntityID(3),
		Direction: 1,
	}

	// Test that it implements Intent interface
	var i Intent = intent
	i.isIntent() // Should not panic

	assert.Equal(t, entity.EntityID(3), intent.EntityID)
	assert.Equal(t, 1, intent.Direction)
}

func TestAttackIntent(t *testing.T) {
	intent := AttackIntent{
		EntityID:   entity.EntityID(4),
		DirectionX: -1,
		DirectionY: 0,
	}

	// Test that it implements Intent interface
	var i Intent = intent
	i.isIntent() // Should not panic

	assert.Equal(t, entity.EntityID(4), intent.EntityID)
	assert.Equal(t, -1, intent.DirectionX)
	assert.Equal(t, 0, intent.DirectionY)
}
