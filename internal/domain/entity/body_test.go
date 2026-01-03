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
	// With 100x scale, velocities are in 100x units.
	// No remainder accumulation needed as precision is built-in.
	tests := []struct {
		name   string
		vx, vy float64
		dt     float64
		wantDX int
		wantDY int
	}{
		{
			name:   "positive velocity (100 units/sec = 1 pixel/sec)",
			vx:     100,
			vy:     50,
			dt:     0.016,
			wantDX: 1, // 100 * 0.016 = 1.6 → 1
			wantDY: 0, // 50 * 0.016 = 0.8 → 0
		},
		{
			name:   "negative velocity",
			vx:     -100,
			vy:     -50,
			dt:     0.016,
			wantDX: -1,
			wantDY: 0,
		},
		{
			name:   "zero velocity",
			vx:     0,
			vy:     0,
			dt:     0.016,
			wantDX: 0,
			wantDY: 0,
		},
		{
			name:   "high velocity (12000 units/sec = 120 pixels/sec)",
			vx:     12000,
			vy:     12000,
			dt:     1.0 / 60.0,
			wantDX: 200, // 12000 / 60 = 200 units
			wantDY: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Body{
				VX: tt.vx,
				VY: tt.vy,
			}

			dx, dy := b.ApplyVelocity(tt.dt)

			assert.Equal(t, tt.wantDX, dx, "dx mismatch")
			assert.Equal(t, tt.wantDY, dy, "dy mismatch")
		})
	}
}

func TestNewPlayer(t *testing.T) {
	hitbox := TrapezoidHitbox{
		Head: HitboxRect{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: HitboxRect{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}

	// Create player at pixel position (100, 200)
	player := NewPlayer(100, 200, hitbox, 100)

	require.NotNil(t, player)
	// Internal position is 100x scaled
	assert.Equal(t, 10000, player.X, "X should be 100x scaled")
	assert.Equal(t, 20000, player.Y, "Y should be 100x scaled")
	// PixelX/PixelY return pixel position
	assert.Equal(t, 100, player.PixelX(), "PixelX should return pixel position")
	assert.Equal(t, 200, player.PixelY(), "PixelY should return pixel position")
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

// ============================================================
// 100x Scale Position Tests (Phase 1)
// ============================================================

func TestBody_PositionScale(t *testing.T) {
	// PositionScale constant should be 100
	// 1 pixel = 100 internal units
	assert.Equal(t, 100, PositionScale, "PositionScale should be 100")
}

func TestBody_PixelPosition(t *testing.T) {
	tests := []struct {
		name       string
		x, y       int // Internal 100x scaled position
		wantPixelX int
		wantPixelY int
	}{
		{
			name:       "zero position",
			x:          0,
			y:          0,
			wantPixelX: 0,
			wantPixelY: 0,
		},
		{
			name:       "exact pixel boundary",
			x:          100,
			y:          200,
			wantPixelX: 1,
			wantPixelY: 2,
		},
		{
			name:       "sub-pixel position rounds down",
			x:          150,
			y:          250,
			wantPixelX: 1, // 150/100 = 1
			wantPixelY: 2, // 250/100 = 2
		},
		{
			name:       "large position",
			x:          32000,
			y:          24000,
			wantPixelX: 320,
			wantPixelY: 240,
		},
		{
			name:       "negative position",
			x:          -100,
			y:          -200,
			wantPixelX: -1,
			wantPixelY: -2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Body{X: tt.x, Y: tt.y}

			assert.Equal(t, tt.wantPixelX, b.PixelX(), "PixelX mismatch")
			assert.Equal(t, tt.wantPixelY, b.PixelY(), "PixelY mismatch")
		})
	}
}

func TestBody_SetPixelPos(t *testing.T) {
	tests := []struct {
		name   string
		pixelX int
		pixelY int
		wantX  int // Expected internal 100x scaled position
		wantY  int
	}{
		{
			name:   "zero position",
			pixelX: 0,
			pixelY: 0,
			wantX:  0,
			wantY:  0,
		},
		{
			name:   "positive position",
			pixelX: 100,
			pixelY: 200,
			wantX:  10000,
			wantY:  20000,
		},
		{
			name:   "screen center",
			pixelX: 160,
			pixelY: 120,
			wantX:  16000,
			wantY:  12000,
		},
		{
			name:   "negative position",
			pixelX: -10,
			pixelY: -20,
			wantX:  -1000,
			wantY:  -2000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Body{}
			b.SetPixelPos(tt.pixelX, tt.pixelY)

			assert.Equal(t, tt.wantX, b.X, "X mismatch")
			assert.Equal(t, tt.wantY, b.Y, "Y mismatch")
		})
	}
}

func TestBody_ApplyVelocity_Scaled(t *testing.T) {
	// With 100x scale, velocities are also 100x
	// Original: 120 pixels/sec -> Now: 12000 units/sec
	tests := []struct {
		name   string
		vx, vy float64 // 100x scaled velocity
		dt     float64
		wantDX int // Expected movement in 100x units
		wantDY int
	}{
		{
			name:   "stationary",
			vx:     0,
			vy:     0,
			dt:     1.0 / 60.0,
			wantDX: 0,
			wantDY: 0,
		},
		{
			name:   "normal speed 120 pixels/sec (12000 units/sec)",
			vx:     12000, // 120 * 100 = 12000 units/sec
			vy:     0,
			dt:     1.0 / 60.0, // ~0.0167 sec
			wantDX: 200,        // 12000 * 0.0167 = 200 units = 2 pixels
			wantDY: 0,
		},
		{
			name:   "gravity fall 800 pixels/sec (80000 units/sec)",
			vx:     0,
			vy:     80000, // 800 * 100 = 80000 units/sec
			dt:     1.0 / 60.0,
			wantDX: 0,
			wantDY: 1333, // 80000 / 60 ≈ 1333 units
		},
		{
			name:   "diagonal movement",
			vx:     6000, // 60 pixels/sec
			vy:     6000,
			dt:     1.0 / 60.0,
			wantDX: 100, // 6000 / 60 = 100 units = 1 pixel
			wantDY: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Body{VX: tt.vx, VY: tt.vy}

			dx, dy := b.ApplyVelocity(tt.dt)

			assert.Equal(t, tt.wantDX, dx, "dx mismatch")
			assert.Equal(t, tt.wantDY, dy, "dy mismatch")
		})
	}
}

func TestNewPlayer_ScaledPosition(t *testing.T) {
	hitbox := TrapezoidHitbox{
		Head: HitboxRect{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: HitboxRect{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: HitboxRect{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}

	// Create player at pixel position (100, 200)
	// Internally should be stored as (10000, 20000)
	player := NewPlayer(100, 200, hitbox, 100)

	require.NotNil(t, player)
	// Internal position should be 100x scaled
	assert.Equal(t, 10000, player.X, "X should be 100x scaled")
	assert.Equal(t, 20000, player.Y, "Y should be 100x scaled")
	// Pixel position helpers should return original values
	assert.Equal(t, 100, player.PixelX(), "PixelX should return pixel position")
	assert.Equal(t, 200, player.PixelY(), "PixelY should return pixel position")
}
