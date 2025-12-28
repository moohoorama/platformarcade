package entity

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewArrow(t *testing.T) {
	arrow := NewArrow(100, 200, true, 300, 20, 500, 350, 180, 25, true)

	require.NotNil(t, arrow)
	assert.Equal(t, 100.0, arrow.X)
	assert.Equal(t, 200.0, arrow.Y)
	assert.True(t, arrow.Active)
	assert.True(t, arrow.IsPlayer)
	assert.Equal(t, 25, arrow.Damage)

	// Check velocity components for 20 degree angle
	expectedAngle := 20.0 * math.Pi / 180
	expectedVX := 300 * math.Cos(expectedAngle)
	expectedVY := -300 * math.Sin(expectedAngle) // Negative because up is negative Y

	assert.InDelta(t, expectedVX, arrow.VX, 0.1)
	assert.InDelta(t, expectedVY, arrow.VY, 0.1)
}

func TestNewArrow_FacingLeft(t *testing.T) {
	arrow := NewArrow(100, 200, false, 300, 20, 500, 350, 180, 25, false)

	// Facing left should have negative VX
	assert.Less(t, arrow.VX, 0.0)
	assert.Less(t, arrow.VY, 0.0) // Still going up initially
}

func TestProjectile_Update(t *testing.T) {
	arrow := NewArrow(100, 200, true, 300, 0, 500, 350, 180, 25, true)
	initialX := arrow.X
	initialVY := arrow.VY

	dt := 0.016 // One frame at 60fps

	arrow.Update(dt)

	// X should increase (moving right)
	assert.Greater(t, arrow.X, initialX)

	// VY should increase (gravity pulls down)
	assert.Greater(t, arrow.VY, initialVY)

	// Distance from start should be greater than 0
	assert.Greater(t, math.Abs(arrow.X-arrow.StartX), 0.0)
}

func TestProjectile_Update_ExceedsRange(t *testing.T) {
	arrow := NewArrow(100, 200, true, 300, 0, 500, 350, 10, 25, true) // Very short range

	// Update until it exceeds range
	for i := 0; i < 10; i++ {
		arrow.Update(0.016)
	}

	assert.False(t, arrow.Active)
}

func TestProjectile_Update_MaxFallSpeed(t *testing.T) {
	arrow := NewArrow(100, 200, true, 300, 0, 500, 100, 180, 25, true) // Low max fall speed

	// Update many times to accelerate past max
	for i := 0; i < 100; i++ {
		arrow.Update(0.016)
	}

	assert.LessOrEqual(t, arrow.VY, 100.0)
}

func TestProjectile_Rotation(t *testing.T) {
	tests := []struct {
		name     string
		vx, vy   float64
		expected float64
	}{
		{
			name:     "moving right",
			vx:       100,
			vy:       0,
			expected: 0,
		},
		{
			name:     "moving down",
			vx:       0,
			vy:       100,
			expected: math.Pi / 2,
		},
		{
			name:     "moving left",
			vx:       -100,
			vy:       0,
			expected: math.Pi,
		},
		{
			name:     "moving up",
			vx:       0,
			vy:       -100,
			expected: -math.Pi / 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Projectile{VX: tt.vx, VY: tt.vy}
			assert.InDelta(t, tt.expected, p.Rotation(), 0.001)
		})
	}
}

func TestProjectile_GetHitbox(t *testing.T) {
	p := &Projectile{
		X:            100.5,
		Y:            200.5,
		HitboxWidth:  12,
		HitboxHeight: 4,
	}

	x, y, w, h := p.GetHitbox()

	assert.Equal(t, 100, x)
	assert.Equal(t, 200, y)
	assert.Equal(t, 12, w)
	assert.Equal(t, 4, h)
}

func TestProjectile_Deactivate(t *testing.T) {
	p := &Projectile{Active: true}

	p.Deactivate()

	assert.False(t, p.Active)
}
