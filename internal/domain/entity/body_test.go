package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHitboxRect_GetWorldRect(t *testing.T) {
	tests := []struct {
		name        string
		hr          HitboxRect
		bodyX       int
		bodyY       int
		facingRight bool
		spriteWidth int
		wantX       int
		wantY       int
		wantW       int
		wantH       int
	}{
		{
			name:        "facing right",
			hr:          HitboxRect{OffsetX: 2, OffsetY: 4, Width: 12, Height: 8},
			bodyX:       100,
			bodyY:       200,
			facingRight: true,
			spriteWidth: 16,
			wantX:       102,
			wantY:       204,
			wantW:       12,
			wantH:       8,
		},
		{
			name:        "facing left - mirrored",
			hr:          HitboxRect{OffsetX: 2, OffsetY: 4, Width: 12, Height: 8},
			bodyX:       100,
			bodyY:       200,
			facingRight: false,
			spriteWidth: 16,
			wantX:       102, // 16 - 2 - 12 = 2, so 100 + 2 = 102
			wantY:       204,
			wantW:       12,
			wantH:       8,
		},
		{
			name:        "zero offset",
			hr:          HitboxRect{OffsetX: 0, OffsetY: 0, Width: 16, Height: 16},
			bodyX:       50,
			bodyY:       50,
			facingRight: true,
			spriteWidth: 16,
			wantX:       50,
			wantY:       50,
			wantW:       16,
			wantH:       16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y, w, h := tt.hr.GetWorldRect(tt.bodyX, tt.bodyY, tt.facingRight, tt.spriteWidth)
			assert.Equal(t, tt.wantX, x, "X mismatch")
			assert.Equal(t, tt.wantY, y, "Y mismatch")
			assert.Equal(t, tt.wantW, w, "Width mismatch")
			assert.Equal(t, tt.wantH, h, "Height mismatch")
		})
	}
}

func TestBody_ApplyVelocity(t *testing.T) {
	tests := []struct {
		name    string
		vx, vy  float64
		dt      float64
		remX    float64
		remY    float64
		wantDX  int
		wantDY  int
		wantRemX float64
		wantRemY float64
	}{
		{
			name:     "positive velocity",
			vx:       100,
			vy:       50,
			dt:       0.016,
			wantDX:   1,
			wantDY:   0,
			wantRemX: 0.6,
			wantRemY: 0.8,
		},
		{
			name:     "negative velocity",
			vx:       -100,
			vy:       -50,
			dt:       0.016,
			wantDX:   -1,
			wantDY:   0,
			wantRemX: -0.6,
			wantRemY: -0.8,
		},
		{
			name:     "accumulates remainder",
			vx:       100,
			vy:       100,
			dt:       0.016,
			remX:     0.5,
			remY:     0.5,
			wantDX:   2,
			wantDY:   2,
			wantRemX: 0.1,
			wantRemY: 0.1,
		},
		{
			name:     "zero velocity",
			vx:       0,
			vy:       0,
			dt:       0.016,
			wantDX:   0,
			wantDY:   0,
			wantRemX: 0,
			wantRemY: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Body{
				VX:   tt.vx,
				VY:   tt.vy,
				RemX: tt.remX,
				RemY: tt.remY,
			}

			dx, dy := b.ApplyVelocity(tt.dt)

			assert.Equal(t, tt.wantDX, dx, "dx mismatch")
			assert.Equal(t, tt.wantDY, dy, "dy mismatch")
			assert.InDelta(t, tt.wantRemX, b.RemX, 0.001, "RemX mismatch")
			assert.InDelta(t, tt.wantRemY, b.RemY, 0.001, "RemY mismatch")
		})
	}
}

func TestNewPlayer(t *testing.T) {
	hitbox := TrapezoidHitbox{
		Head: HitboxRect{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: HitboxRect{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}

	player := NewPlayer(100, 200, hitbox, 100)

	require.NotNil(t, player)
	assert.Equal(t, 100, player.X)
	assert.Equal(t, 200, player.Y)
	assert.Equal(t, 100, player.Health)
	assert.Equal(t, 100, player.MaxHealth)
	assert.True(t, player.FacingRight)
	assert.True(t, player.CanDash)
}

func TestPlayer_IsInvincible(t *testing.T) {
	player := &Player{}

	// Not invincible by default
	assert.False(t, player.IsInvincible())

	// Invincible during i-frames
	player.IframeTimer = 0.5
	assert.True(t, player.IsInvincible())

	// Invincible during dash
	player.IframeTimer = 0
	player.Dashing = true
	assert.True(t, player.IsInvincible())
}

func TestPlayer_IsStunned(t *testing.T) {
	player := &Player{}

	assert.False(t, player.IsStunned())

	player.StunTimer = 0.5
	assert.True(t, player.IsStunned())
}
