package ecs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockStage implements Stage interface for testing
type mockStage struct {
	width, height, tileSize int
	solidTiles              map[[2]int]bool
}

func newMockStage(w, h, tileSize int) *mockStage {
	return &mockStage{
		width:      w,
		height:     h,
		tileSize:   tileSize,
		solidTiles: make(map[[2]int]bool),
	}
}

func (s *mockStage) setSolid(tileX, tileY int) {
	s.solidTiles[[2]int{tileX, tileY}] = true
}

func (s *mockStage) IsSolidAt(px, py int) bool {
	tx := px / s.tileSize
	ty := py / s.tileSize
	return s.solidTiles[[2]int{tx, ty}]
}

func (s *mockStage) GetTileType(px, py int) int   { return TileEmpty }
func (s *mockStage) GetTileDamage(px, py int) int { return 0 }
func (s *mockStage) GetWidth() int                { return s.width }
func (s *mockStage) GetHeight() int               { return s.height }
func (s *mockStage) GetTileSize() int             { return s.tileSize }
func (s *mockStage) GetSpawnX() int               { return 0 }
func (s *mockStage) GetSpawnY() int               { return 0 }

// =============================================================================
// Conversion Function Tests
// =============================================================================

func TestToIUPerSubstep(t *testing.T) {
	tests := []struct {
		name         string
		pixelsPerSec float64
		expectedIU   int
	}{
		{"60 pixels/sec", 60, 25},    // 60 * 256 / 600 = 25.6 ≈ 25
		{"120 pixels/sec", 120, 51},  // 120 * 256 / 600 = 51.2 ≈ 51
		{"300 pixels/sec", 300, 128}, // 300 * 256 / 600 = 128
		{"600 pixels/sec", 600, 256}, // 600 * 256 / 600 = 256 (1 pixel/substep)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToIUPerSubstep(tt.pixelsPerSec)
			assert.Equal(t, tt.expectedIU, result)
		})
	}
}

func TestToIUAccelPerFrame(t *testing.T) {
	tests := []struct {
		name           string
		pixelsPerSecSq float64
		expectedIU     int
	}{
		{"800 pixels/sec² (gravity)", 800, 5},   // 800 * 256 / 36000 = 5.68 ≈ 5
		{"2000 pixels/sec² (accel)", 2000, 14},  // 2000 * 256 / 36000 = 14.2 ≈ 14
		{"400 pixels/sec² (gold)", 400, 2},      // 400 * 256 / 36000 = 2.84 ≈ 2
		{"3600 pixels/sec²", 3600, 25},          // 3600 * 256 / 36000 = 25.6 ≈ 25
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToIUAccelPerFrame(tt.pixelsPerSecSq)
			assert.Equal(t, tt.expectedIU, result)
		})
	}
}

// =============================================================================
// Physics Simulation Tests - 1 Second Movement Validation
// =============================================================================

// TestPlayerMovement_OneSecond verifies player moves expected distance in 1 second
func TestPlayerMovement_OneSecond(t *testing.T) {
	const (
		framesPerSecond = 60
		subStepsPerFrame = 10
		targetSpeedPixels = 120.0 // 120 pixels/sec max speed
	)

	stage := newMockStage(100, 100, 16)
	world := NewWorld()

	// Create player at origin
	hitbox := HitboxTrapezoid{
		Head: Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	world.CreatePlayer(500, 500, hitbox, 100)

	// Set player on ground to avoid gravity affecting horizontal movement
	mov := world.Movement[world.PlayerID]
	mov.OnGround = true
	world.Movement[world.PlayerID] = mov

	cfg := PhysicsConfig{
		MaxSpeed:        ToIUPerSubstep(targetSpeedPixels),
		Acceleration:    ToIUAccelPerFrame(10000), // Very high for instant accel
		Deceleration:    ToIUAccelPerFrame(10000),
		AirControlPct:   100,
		TurnaroundPct:   100,
		Gravity:         ToIUAccelPerFrame(800),
		MaxFallSpeed:    ToIUPerSubstep(400),
	}

	startPos := world.Position[world.PlayerID]
	startPixelX := startPos.PixelX()

	// Simulate 1 second: 60 frames × 10 substeps
	for frame := 0; frame < framesPerSecond; frame++ {
		// Player input: move right
		UpdatePlayerInput(world, InputState{Right: true}, cfg)

		// Apply gravity once per frame
		ApplyPlayerGravity(world, cfg)

		// Physics substeps
		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdatePlayerPhysics(world, stage, cfg)
		}
	}

	endPos := world.Position[world.PlayerID]
	endPixelX := endPos.PixelX()

	distanceMoved := endPixelX - startPixelX

	// Expected: ~120 pixels in 1 second
	// Allow 10% tolerance for acceleration ramp-up
	expectedMin := int(targetSpeedPixels * 0.9)
	expectedMax := int(targetSpeedPixels * 1.1)

	t.Logf("Player moved %d pixels in 1 second (expected ~%d)", distanceMoved, int(targetSpeedPixels))

	assert.GreaterOrEqual(t, distanceMoved, expectedMin,
		"Player should move at least %d pixels, moved %d", expectedMin, distanceMoved)
	assert.LessOrEqual(t, distanceMoved, expectedMax,
		"Player should move at most %d pixels, moved %d", expectedMax, distanceMoved)
}

// TestPlayerGravity_OneSecond verifies gravity accelerates player correctly
func TestPlayerGravity_OneSecond(t *testing.T) {
	const (
		framesPerSecond  = 60
		subStepsPerFrame = 10
		gravityPixelsSec = 800.0 // 800 pixels/sec²
	)

	stage := newMockStage(100, 100, 16)
	world := NewWorld()

	hitbox := HitboxTrapezoid{
		Head: Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	world.CreatePlayer(500, 100, hitbox, 100)

	cfg := PhysicsConfig{
		Gravity:           ToIUAccelPerFrame(gravityPixelsSec),
		MaxFallSpeed:      ToIUPerSubstep(10000), // Very high to not clamp
		MaxSpeed:          ToIUPerSubstep(120),
		FallMultiplierPct: 100, // Normal fall
		ApexModEnabled:    false,
	}

	startPos := world.Position[world.PlayerID]
	startPixelY := startPos.PixelY()

	// Simulate 1 second of free fall
	for frame := 0; frame < framesPerSecond; frame++ {
		UpdatePlayerInput(world, InputState{}, cfg)
		ApplyPlayerGravity(world, cfg)

		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdatePlayerPhysics(world, stage, cfg)
		}
	}

	endPos := world.Position[world.PlayerID]
	endPixelY := endPos.PixelY()
	distanceFallen := endPixelY - startPixelY

	// Physics: d = 0.5 * a * t² = 0.5 * 800 * 1² = 400 pixels
	expectedDistance := int(0.5 * gravityPixelsSec * 1.0 * 1.0)

	t.Logf("Player fell %d pixels in 1 second (expected ~%d)", distanceFallen, expectedDistance)

	// Allow 15% tolerance
	expectedMin := int(float64(expectedDistance) * 0.85)
	expectedMax := int(float64(expectedDistance) * 1.15)

	assert.GreaterOrEqual(t, distanceFallen, expectedMin,
		"Player should fall at least %d pixels, fell %d", expectedMin, distanceFallen)
	assert.LessOrEqual(t, distanceFallen, expectedMax,
		"Player should fall at most %d pixels, fell %d", expectedMax, distanceFallen)
}

// TestEnemyMovement_OneSecond verifies enemy moves expected distance
func TestEnemyMovement_OneSecond(t *testing.T) {
	const (
		framesPerSecond  = 60
		subStepsPerFrame = 10
		moveSpeedPixels  = 60.0 // 60 pixels/sec
	)

	stage := newMockStage(100, 100, 16)
	world := NewWorld()

	// Create enemy at position with large patrol area
	startX := 500
	enemyCfg := EnemyConfig{
		MaxHealth:     100,
		MoveSpeed:     ToIUPerSubstep(moveSpeedPixels),
		HitboxOffsetX: 2,
		HitboxOffsetY: 4,
		HitboxWidth:   12,
		HitboxHeight:  20,
		AIType:        AIPatrol,
		PatrolDist:    1000, // Large patrol distance
	}
	enemyID := world.CreateEnemy(startX, 500, enemyCfg, true) // facingRight=true

	// Ensure patrol direction is positive (right)
	ai := world.AI[enemyID]
	ai.PatrolDir = 1
	ai.PatrolStartX = startX - 500 // Start far to the left so we have room to move right
	world.AI[enemyID] = ai

	// Set on ground
	mov := world.Movement[enemyID]
	mov.OnGround = true
	world.Movement[enemyID] = mov

	arrowCfg := ProjectileConfig{}

	startPos := world.Position[enemyID]
	startPixelX := startPos.PixelX()

	// Simulate 1 second
	for frame := 0; frame < framesPerSecond; frame++ {
		ApplyEnemyGravity(world, stage, ToIUAccelPerFrame(800), ToIUPerSubstep(400))

		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdateEnemyAI(world, stage, arrowCfg, PhysicsConfig{})
		}
	}

	endPos := world.Position[enemyID]
	endPixelX := endPos.PixelX()
	distanceMoved := endPixelX - startPixelX

	t.Logf("Enemy moved %d pixels in 1 second (expected ~%d)", distanceMoved, int(moveSpeedPixels))

	// Allow 10% tolerance
	expectedMin := int(moveSpeedPixels * 0.9)
	expectedMax := int(moveSpeedPixels * 1.1)

	assert.GreaterOrEqual(t, distanceMoved, expectedMin,
		"Enemy should move at least %d pixels, moved %d", expectedMin, distanceMoved)
	assert.LessOrEqual(t, distanceMoved, expectedMax,
		"Enemy should move at most %d pixels, moved %d", expectedMax, distanceMoved)
}

// TestEnemyGravity_OneSecond verifies enemy falls correctly
func TestEnemyGravity_OneSecond(t *testing.T) {
	const (
		framesPerSecond  = 60
		subStepsPerFrame = 10
		gravityPixelsSec = 800.0
	)

	stage := newMockStage(100, 100, 16)
	world := NewWorld()

	enemyCfg := EnemyConfig{
		MaxHealth:     100,
		MoveSpeed:     0, // No horizontal movement
		HitboxOffsetX: 2,
		HitboxOffsetY: 4,
		HitboxWidth:   12,
		HitboxHeight:  20,
		AIType:        AIPatrol,
		Flying:        false,
	}
	enemyID := world.CreateEnemy(500, 100, enemyCfg, true)

	// Ensure not on ground
	mov := world.Movement[enemyID]
	mov.OnGround = false
	world.Movement[enemyID] = mov

	gravity := ToIUAccelPerFrame(gravityPixelsSec)
	maxFall := ToIUPerSubstep(10000) // Very high

	arrowCfg := ProjectileConfig{}

	startPos := world.Position[enemyID]
	startPixelY := startPos.PixelY()

	// Simulate 1 second
	for frame := 0; frame < framesPerSecond; frame++ {
		ApplyEnemyGravity(world, stage, gravity, maxFall)

		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdateEnemyAI(world, stage, arrowCfg, PhysicsConfig{})
		}
	}

	endPos := world.Position[enemyID]
	endPixelY := endPos.PixelY()
	distanceFallen := endPixelY - startPixelY

	expectedDistance := int(0.5 * gravityPixelsSec * 1.0 * 1.0)

	t.Logf("Enemy fell %d pixels in 1 second (expected ~%d)", distanceFallen, expectedDistance)

	// Allow 15% tolerance
	expectedMin := int(float64(expectedDistance) * 0.85)
	expectedMax := int(float64(expectedDistance) * 1.15)

	assert.GreaterOrEqual(t, distanceFallen, expectedMin,
		"Enemy should fall at least %d pixels, fell %d", expectedMin, distanceFallen)
	assert.LessOrEqual(t, distanceFallen, expectedMax,
		"Enemy should fall at most %d pixels, fell %d", expectedMax, distanceFallen)
}

// TestProjectileMovement_OneSecond verifies projectile moves at correct speed
func TestProjectileMovement_OneSecond(t *testing.T) {
	const (
		framesPerSecond  = 60
		subStepsPerFrame = 10
		speedPixels      = 300.0 // 300 pixels/sec
	)

	stage := newMockStage(1000, 100, 16)
	world := NewWorld()

	// Create a horizontal projectile (no gravity for this test)
	projCfg := ProjectileConfig{
		GravityAccel:  0, // No gravity
		MaxFallSpeed:  ToIUPerSubstep(300),
		MaxRange:      10000,
		Damage:        10,
		HitboxOffsetX: 0,
		HitboxOffsetY: 0,
		HitboxWidth:   8,
		HitboxHeight:  4,
		StuckDuration: 300,
	}

	vx := ToIUPerSubstep(speedPixels)
	vy := 0
	projID := world.CreateProjectile(100, 500, vx, vy, projCfg, true)

	startPos := world.Position[projID]
	startPixelX := startPos.PixelX()

	// Simulate 1 second
	for frame := 0; frame < framesPerSecond; frame++ {
		ApplyProjectileGravity(world)

		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdateProjectiles(world, stage)
		}
	}

	endPos := world.Position[projID]
	endPixelX := endPos.PixelX()
	distanceMoved := endPixelX - startPixelX

	t.Logf("Projectile moved %d pixels in 1 second (expected ~%d)", distanceMoved, int(speedPixels))

	// Allow 5% tolerance (projectiles should be precise)
	expectedMin := int(speedPixels * 0.95)
	expectedMax := int(speedPixels * 1.05)

	assert.GreaterOrEqual(t, distanceMoved, expectedMin,
		"Projectile should move at least %d pixels, moved %d", expectedMin, distanceMoved)
	assert.LessOrEqual(t, distanceMoved, expectedMax,
		"Projectile should move at most %d pixels, moved %d", expectedMax, distanceMoved)
}

// TestProjectileGravity_OneSecond verifies projectile falls with gravity
func TestProjectileGravity_OneSecond(t *testing.T) {
	const (
		framesPerSecond  = 60
		subStepsPerFrame = 10
		gravityPixelsSec = 400.0 // Lighter gravity for arrows
	)

	stage := newMockStage(1000, 1000, 16)
	world := NewWorld()

	projCfg := ProjectileConfig{
		GravityAccel:  ToIUAccelPerFrame(gravityPixelsSec),
		MaxFallSpeed:  ToIUPerSubstep(10000), // Very high
		MaxRange:      10000,
		Damage:        10,
		HitboxOffsetX: 0,
		HitboxOffsetY: 0,
		HitboxWidth:   8,
		HitboxHeight:  4,
		StuckDuration: 300,
	}

	// Horizontal shot, gravity will pull down
	vx := ToIUPerSubstep(100)
	vy := 0
	projID := world.CreateProjectile(100, 100, vx, vy, projCfg, true)

	startPos := world.Position[projID]
	startPixelY := startPos.PixelY()

	// Simulate 1 second
	for frame := 0; frame < framesPerSecond; frame++ {
		ApplyProjectileGravity(world)

		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdateProjectiles(world, stage)
		}
	}

	endPos := world.Position[projID]
	endPixelY := endPos.PixelY()
	distanceFallen := endPixelY - startPixelY

	// Note: Due to integer truncation in ToIUAccelPerFrame
	actualGravity := float64(ToIUAccelPerFrame(gravityPixelsSec)) * 36000.0 / float64(PositionScale)
	expectedDistance := int(0.5 * actualGravity * 1.0 * 1.0)

	t.Logf("Projectile fell %d pixels in 1 second (expected ~%d with actual gravity %.1f pixels/sec²)",
		distanceFallen, expectedDistance, actualGravity)

	// Allow 15% tolerance
	expectedMin := int(float64(expectedDistance) * 0.85)
	expectedMax := int(float64(expectedDistance) * 1.15)

	assert.GreaterOrEqual(t, distanceFallen, expectedMin,
		"Projectile should fall at least %d pixels, fell %d", expectedMin, distanceFallen)
	assert.LessOrEqual(t, distanceFallen, expectedMax,
		"Projectile should fall at most %d pixels, fell %d", expectedMax, distanceFallen)
}

// TestGoldGravity_OneSecond verifies gold falls correctly
func TestGoldGravity_OneSecond(t *testing.T) {
	const (
		framesPerSecond  = 60
		subStepsPerFrame = 10
		gravityPixelsSec = 400.0
	)

	stage := newMockStage(100, 1000, 16)
	world := NewWorld()

	goldCfg := GoldConfig{
		Gravity:       ToIUAccelPerFrame(gravityPixelsSec),
		BouncePercent: 0, // No bounce
		CollectDelay:  0,
		HitboxWidth:   8,
		HitboxHeight:  8,
		CollectRadius: 16,
	}

	goldID := world.CreateGold(500, 100, 10, goldCfg)

	// Reset velocity to 0 (CreateGold sets initial pop velocity)
	world.Velocity[goldID] = Velocity{X: 0, Y: 0}

	startPos := world.Position[goldID]
	startPixelY := startPos.PixelY()

	// Simulate 1 second
	for frame := 0; frame < framesPerSecond; frame++ {
		ApplyGoldGravity(world)

		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdateGoldPhysics(world, stage)
		}
	}

	endPos := world.Position[goldID]
	endPixelY := endPos.PixelY()
	distanceFallen := endPixelY - startPixelY

	// Note: Due to integer truncation in ToIUAccelPerFrame:
	// ToIUAccelPerFrame(400) = 2 IU/frame (actual: 281 pixels/sec²)
	// Expected: 0.5 * 281 * 1² = 140 pixels
	actualGravity := float64(ToIUAccelPerFrame(gravityPixelsSec)) * 36000.0 / float64(PositionScale)
	expectedDistance := int(0.5 * actualGravity * 1.0 * 1.0)

	t.Logf("Gold fell %d pixels in 1 second (expected ~%d with actual gravity %.1f pixels/sec²)",
		distanceFallen, expectedDistance, actualGravity)

	// Allow 15% tolerance
	expectedMin := int(float64(expectedDistance) * 0.85)
	expectedMax := int(float64(expectedDistance) * 1.15)

	assert.GreaterOrEqual(t, distanceFallen, expectedMin,
		"Gold should fall at least %d pixels, fell %d", expectedMin, distanceFallen)
	assert.LessOrEqual(t, distanceFallen, expectedMax,
		"Gold should fall at most %d pixels, fell %d", expectedMax, distanceFallen)
}

// =============================================================================
// Velocity Sanity Check - Final velocity after 1 second of gravity
// =============================================================================

func TestFinalVelocity_AfterOneSecondGravity(t *testing.T) {
	const (
		framesPerSecond  = 60
		gravityPixelsSec = 800.0
	)

	world := NewWorld()

	hitbox := HitboxTrapezoid{
		Head: Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	world.CreatePlayer(500, 100, hitbox, 100)

	gravityIU := ToIUAccelPerFrame(gravityPixelsSec)
	cfg := PhysicsConfig{
		Gravity:           gravityIU,
		MaxFallSpeed:      ToIUPerSubstep(10000),
		FallMultiplierPct: 100,
		ApexModEnabled:    false,
	}

	// Apply gravity for 60 frames
	for frame := 0; frame < framesPerSecond; frame++ {
		ApplyPlayerGravity(world, cfg)
	}

	vel := world.Velocity[world.PlayerID]

	// Expected: v = gravity_IU_per_frame * frames = 5 * 60 = 300 IU/substep
	expectedVelIU := gravityIU * framesPerSecond
	actualPixelsSec := float64(expectedVelIU) * 600.0 / float64(PositionScale)

	t.Logf("Final velocity: %d IU/substep (expected %d, actual %.1f pixels/sec)", vel.Y, expectedVelIU, actualPixelsSec)

	// Allow 5% tolerance (should be exact since it's just addition)
	expectedMin := int(float64(expectedVelIU) * 0.95)
	expectedMax := int(float64(expectedVelIU) * 1.05)

	assert.GreaterOrEqual(t, vel.Y, expectedMin,
		"Final velocity should be at least %d, got %d", expectedMin, vel.Y)
	assert.LessOrEqual(t, vel.Y, expectedMax,
		"Final velocity should be at most %d, got %d", expectedMax, vel.Y)
}

// =============================================================================
// Debug: Print actual values for analysis
// =============================================================================

func TestPrintPhysicsValues(t *testing.T) {
	t.Log("=== Physics Conversion Values ===")
	t.Logf("PositionScale: %d", PositionScale)
	t.Log("")

	t.Log("=== Velocity Conversions (ToIUPerSubstep) ===")
	velocities := []float64{60, 120, 280, 300, 400, 600}
	for _, v := range velocities {
		iu := ToIUPerSubstep(v)
		// Back-calculate: what's the actual pixels/sec?
		// IU/substep * 600 / 256 = pixels/sec
		actual := float64(iu) * 600.0 / float64(PositionScale)
		t.Logf("  %.0f pixels/sec → %d IU/substep (actual: %.1f pixels/sec)", v, iu, actual)
	}
	t.Log("")

	t.Log("=== Acceleration Conversions (ToIUAccelPerFrame) ===")
	accels := []float64{400, 800, 2000, 2500}
	for _, a := range accels {
		iu := ToIUAccelPerFrame(a)
		// Back-calculate: IU/frame * 36000 / 256 = pixels/sec²
		actual := float64(iu) * 36000.0 / float64(PositionScale)
		t.Logf("  %.0f pixels/sec² → %d IU/frame (actual: %.1f pixels/sec²)", a, iu, actual)
	}
	t.Log("")

	t.Log("=== 1 Second Simulation Expected Values ===")
	t.Logf("  Distance at 120 pixels/sec for 1 sec: 120 pixels")
	t.Logf("  Distance fallen with 800 pixels/sec² gravity: %.0f pixels (d=0.5*a*t²)", 0.5*800*1*1)
	t.Logf("  Final velocity after 1 sec of 800 pixels/sec² gravity: 800 pixels/sec")
}

// =============================================================================
// Debug Tests - Step by Step Analysis
// =============================================================================

func TestGoldGravity_Debug(t *testing.T) {
	const (
		framesPerSecond  = 60
		subStepsPerFrame = 10
		gravityPixelsSec = 400.0
	)

	stage := newMockStage(100, 1000, 16)
	world := NewWorld()

	gravity := ToIUAccelPerFrame(gravityPixelsSec)
	t.Logf("Gravity: %d IU/frame (from %.0f pixels/sec²)", gravity, gravityPixelsSec)

	goldCfg := GoldConfig{
		Gravity:       gravity,
		BouncePercent: 0,
		CollectDelay:  0,
		HitboxWidth:   8,
		HitboxHeight:  8,
		CollectRadius: 16,
	}

	goldID := world.CreateGold(500, 100, 10, goldCfg)

	startPos := world.Position[goldID]
	t.Logf("Start position: %d IU (%d pixels)", startPos.Y, startPos.PixelY())

	// Simulate just 10 frames and log each
	for frame := 0; frame < 10; frame++ {
		velBefore := world.Velocity[goldID]
		posBefore := world.Position[goldID]

		ApplyGoldGravity(world)

		velAfterGravity := world.Velocity[goldID]

		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdateGoldPhysics(world, stage)
		}

		posAfter := world.Position[goldID]
		velAfter := world.Velocity[goldID]

		movedIU := posAfter.Y - posBefore.Y

		t.Logf("Frame %d: vel %d → %d → %d IU, pos %d → %d (%d IU moved, %.2f px)",
			frame, velBefore.Y, velAfterGravity.Y, velAfter.Y,
			posBefore.Y, posAfter.Y, movedIU, float64(movedIU)/float64(PositionScale))
	}

	endPos := world.Position[goldID]
	totalMoved := endPos.Y - startPos.Y
	t.Logf("Total moved in 10 frames: %d IU (%.2f pixels)", totalMoved, float64(totalMoved)/float64(PositionScale))
}

func TestProjectileGravity_Debug(t *testing.T) {
	const (
		framesPerSecond  = 60
		subStepsPerFrame = 10
		gravityPixelsSec = 400.0
	)

	stage := newMockStage(1000, 1000, 16)
	world := NewWorld()

	gravity := ToIUAccelPerFrame(gravityPixelsSec)
	t.Logf("Gravity: %d IU/frame (from %.0f pixels/sec²)", gravity, gravityPixelsSec)

	projCfg := ProjectileConfig{
		GravityAccel:  gravity,
		MaxFallSpeed:  ToIUPerSubstep(10000),
		MaxRange:      10000,
		Damage:        10,
		HitboxOffsetX: 0,
		HitboxOffsetY: 0,
		HitboxWidth:   8,
		HitboxHeight:  4,
		StuckDuration: 300,
	}

	projID := world.CreateProjectile(100, 100, 0, 0, projCfg, true)

	startPos := world.Position[projID]
	t.Logf("Start position: %d IU (%d pixels)", startPos.Y, startPos.PixelY())

	// Simulate just 10 frames and log each
	for frame := 0; frame < 10; frame++ {
		velBefore := world.Velocity[projID]
		posBefore := world.Position[projID]

		ApplyProjectileGravity(world)

		velAfterGravity := world.Velocity[projID]

		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdateProjectiles(world, stage)
		}

		posAfter := world.Position[projID]
		velAfter := world.Velocity[projID]

		movedIU := posAfter.Y - posBefore.Y

		t.Logf("Frame %d: vel %d → %d → %d IU, pos %d → %d (%d IU moved, %.2f px)",
			frame, velBefore.Y, velAfterGravity.Y, velAfter.Y,
			posBefore.Y, posAfter.Y, movedIU, float64(movedIU)/float64(PositionScale))
	}

	endPos := world.Position[projID]
	totalMoved := endPos.Y - startPos.Y
	t.Logf("Total moved in 10 frames: %d IU (%.2f pixels)", totalMoved, float64(totalMoved)/float64(PositionScale))
}

// =============================================================================
// Enemy Spawn Gravity Bug Reproduction
// =============================================================================

// TestEnemySpawnInAir_ShouldFall reproduces the bug where enemies spawned in
// mid-air don't fall until they jump first.
// Bug symptoms:
// 1. Enemy spawns in mid-air (no ground below)
// 2. Initial state: Movement{OnGround: false}, Velocity{Y: 0}
// 3. Enemy should fall due to gravity, but doesn't
// 4. After jumping, enemy falls normally
func TestEnemySpawnInAir_ShouldFall(t *testing.T) {
	const (
		framesPerSecond  = 60
		subStepsPerFrame = 10
		gravityPixelsSec = 800.0
	)

	// Create stage with NO ground anywhere
	stage := newMockStage(100, 100, 16)
	// Don't set any solid tiles - enemy is in pure air

	world := NewWorld()

	// Create player (required for AI targeting)
	hitbox := HitboxTrapezoid{
		Head: Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	world.CreatePlayer(100, 500, hitbox, 100)

	// Create enemy in mid-air
	enemyCfg := EnemyConfig{
		MaxHealth:     100,
		MoveSpeed:     ToIUPerSubstep(60),
		HitboxOffsetX: 2,
		HitboxOffsetY: 4,
		HitboxWidth:   12,
		HitboxHeight:  20,
		AIType:        AIAggressive, // Aggressive AI chases player
		Flying:        false,
	}
	enemyID := world.CreateEnemy(500, 100, enemyCfg, true) // Spawn at y=100 pixels

	// Verify initial state
	mov := world.Movement[enemyID]
	vel := world.Velocity[enemyID]
	t.Logf("Initial state: OnGround=%v, Velocity.Y=%d", mov.OnGround, vel.Y)

	assert.False(t, mov.OnGround, "Enemy should start with OnGround=false")
	assert.Equal(t, 0, vel.Y, "Enemy should start with zero Y velocity")

	gravity := ToIUAccelPerFrame(gravityPixelsSec)
	maxFall := ToIUPerSubstep(400)
	arrowCfg := ProjectileConfig{}

	startPos := world.Position[enemyID]
	startPixelY := startPos.PixelY()

	// Simulate 30 frames (0.5 seconds) - enemy should fall significantly
	for frame := 0; frame < 30; frame++ {
		ApplyEnemyGravity(world, stage, gravity, maxFall)

		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdateEnemyAI(world, stage, arrowCfg, PhysicsConfig{})
		}

		// Log first few frames for debugging
		if frame < 5 {
			pos := world.Position[enemyID]
			vel := world.Velocity[enemyID]
			mov := world.Movement[enemyID]
			t.Logf("Frame %d: Y=%d pixels, VelY=%d, OnGround=%v",
				frame, pos.PixelY(), vel.Y, mov.OnGround)
		}
	}

	endPos := world.Position[enemyID]
	endPixelY := endPos.PixelY()
	distanceFallen := endPixelY - startPixelY

	t.Logf("Enemy fell %d pixels in 0.5 seconds", distanceFallen)

	// In 0.5 seconds with 800 pixels/sec² gravity:
	// d = 0.5 * 800 * 0.5² = 100 pixels
	// Allow some tolerance
	assert.Greater(t, distanceFallen, 50,
		"Enemy spawned in air should fall at least 50 pixels in 0.5 seconds, but fell %d", distanceFallen)
}

// TestEnemySpawnOnGround_ThenWalkOffEdge tests the scenario where enemy
// starts on ground, then walks off a platform edge into the air
func TestEnemySpawnOnGround_ThenWalkOffEdge(t *testing.T) {
	const (
		subStepsPerFrame = 10
		gravityPixelsSec = 800.0
	)

	// Create stage with a small platform
	stage := newMockStage(100, 100, 16)
	// Platform at tile (31, 10) - enemy will start on top
	stage.setSolid(31, 10)

	world := NewWorld()

	// Create player far away
	hitbox := HitboxTrapezoid{
		Head: Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	world.CreatePlayer(100, 500, hitbox, 100)

	// Spawn enemy on the platform (tile 31 = pixel 496, y=9*16=144)
	// Enemy hitbox: OffsetY=4, Height=20, so feet end at y+4+20=y+24
	// For feet to be at tile y=10 (pixel 160), enemy y should be 160-24=136
	enemyX := 31 * 16 // = 496
	enemyY := 136     // feet will be at y=160 (tile 10)

	enemyCfg := EnemyConfig{
		MaxHealth:     100,
		MoveSpeed:     ToIUPerSubstep(60),
		HitboxOffsetX: 2,
		HitboxOffsetY: 4,
		HitboxWidth:   12,
		HitboxHeight:  20,
		AIType:        AIPatrol,
		PatrolDist:    100,
		Flying:        false,
	}
	enemyID := world.CreateEnemy(enemyX, enemyY, enemyCfg, false) // facing left

	// Set patrol direction to move left (off the platform)
	ai := world.AI[enemyID]
	ai.PatrolDir = -1
	ai.PatrolStartX = enemyX + 50
	world.AI[enemyID] = ai

	gravity := ToIUAccelPerFrame(gravityPixelsSec)
	maxFall := ToIUPerSubstep(400)
	arrowCfg := ProjectileConfig{}

	t.Logf("=== Enemy Walk Off Edge Test ===")
	t.Logf("Platform at tile (31, 10), enemy starting at pixel (%d, %d)", enemyX, enemyY)

	// Simulate several frames
	for frame := 0; frame < 30; frame++ {
		movBefore := world.Movement[enemyID]
		velBefore := world.Velocity[enemyID]

		ApplyEnemyGravity(world, stage, gravity, maxFall)

		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdateEnemyAI(world, stage, arrowCfg, PhysicsConfig{})
		}

		posAfter := world.Position[enemyID]
		movAfter := world.Movement[enemyID]
		velAfter := world.Velocity[enemyID]

		if frame < 10 || movBefore.OnGround != movAfter.OnGround {
			t.Logf("Frame %d: pos=(%d,%d), OnGround=%v→%v, VelY=%d→%d",
				frame,
				posAfter.PixelX(), posAfter.PixelY(),
				movBefore.OnGround, movAfter.OnGround,
				velBefore.Y, velAfter.Y)
		}
	}

	// After walking off edge, enemy should have fallen
	endPos := world.Position[enemyID]
	endMov := world.Movement[enemyID]

	t.Logf("Final: pos=(%d,%d), OnGround=%v", endPos.PixelX(), endPos.PixelY(), endMov.OnGround)

	// Enemy should have moved left off the platform and be falling
	assert.Less(t, endPos.PixelX(), enemyX, "Enemy should have moved left")
	assert.False(t, endMov.OnGround, "Enemy should not be on ground after walking off edge")
}

// TestEnemyStartsWithOnGroundTrue tests if enemy incorrectly starts with OnGround=true
func TestEnemyStartsWithOnGroundTrue(t *testing.T) {
	const (
		subStepsPerFrame = 10
		gravityPixelsSec = 800.0
	)

	// Create stage with NO ground
	stage := newMockStage(100, 100, 16)

	world := NewWorld()

	// Create player
	hitbox := HitboxTrapezoid{
		Head: Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	world.CreatePlayer(100, 500, hitbox, 100)

	// Create enemy in mid-air
	enemyCfg := EnemyConfig{
		MaxHealth:     100,
		MoveSpeed:     ToIUPerSubstep(60),
		HitboxOffsetX: 2,
		HitboxOffsetY: 4,
		HitboxWidth:   12,
		HitboxHeight:  20,
		AIType:        AIAggressive,
		Flying:        false,
	}
	enemyID := world.CreateEnemy(500, 100, enemyCfg, true)

	// *** SIMULATE THE BUG: Force OnGround = true ***
	mov := world.Movement[enemyID]
	mov.OnGround = true // This might be what's happening in real game
	world.Movement[enemyID] = mov

	gravity := ToIUAccelPerFrame(gravityPixelsSec)
	maxFall := ToIUPerSubstep(400)
	arrowCfg := ProjectileConfig{}

	startPos := world.Position[enemyID]
	t.Logf("=== Simulated Bug: OnGround=true in mid-air ===")
	t.Logf("Start: pos=(%d,%d), OnGround=true (forced)", startPos.PixelX(), startPos.PixelY())

	// Simulate 10 frames
	for frame := 0; frame < 10; frame++ {
		posBefore := world.Position[enemyID]
		movBefore := world.Movement[enemyID]
		velBefore := world.Velocity[enemyID]

		ApplyEnemyGravity(world, stage, gravity, maxFall)

		movAfterGravity := world.Movement[enemyID]
		velAfterGravity := world.Velocity[enemyID]

		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdateEnemyAI(world, stage, arrowCfg, PhysicsConfig{})
		}

		posAfter := world.Position[enemyID]
		movAfter := world.Movement[enemyID]
		velAfter := world.Velocity[enemyID]

		movedY := posAfter.Y - posBefore.Y
		movedPixels := float64(movedY) / float64(PositionScale)

		t.Logf("Frame %d: OnGround=%v→%v→%v, VelY=%d→%d→%d, Moved=%.2f px",
			frame,
			movBefore.OnGround, movAfterGravity.OnGround, movAfter.OnGround,
			velBefore.Y, velAfterGravity.Y, velAfter.Y,
			movedPixels)
	}

	endPos := world.Position[enemyID]
	distanceFallen := endPos.PixelY() - startPos.PixelY()

	t.Logf("Total fallen: %d pixels", distanceFallen)

	// Even with OnGround=true, after ApplyEnemyGravity checks there's no ground,
	// it should set OnGround=false and apply gravity
	assert.GreaterOrEqual(t, distanceFallen, 10,
		"Enemy should fall even if started with OnGround=true (bug scenario)")
}

// TestEnemyGroundCheckMismatch tests if ApplyEnemyGravity and moveEnemyY have
// different ground check logic, causing inconsistent behavior
func TestEnemyGroundCheckMismatch(t *testing.T) {
	// Create stage with a single tile platform
	stage := newMockStage(100, 100, 16)
	stage.setSolid(10, 10) // Platform at tile (10, 10) = pixel (160, 160)

	world := NewWorld()

	// Create player
	hitbox := HitboxTrapezoid{
		Head: Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	world.CreatePlayer(100, 500, hitbox, 100)

	// Enemy hitbox: OffsetX=2, OffsetY=4, Width=12, Height=20
	// Feet end at: y + 4 + 20 = y + 24
	// To stand on tile 10 (pixel 160), enemy y should be 160 - 24 = 136

	enemyCfg := EnemyConfig{
		MaxHealth:     100,
		MoveSpeed:     0, // No movement
		HitboxOffsetX: 2,
		HitboxOffsetY: 4,
		HitboxWidth:   12,
		HitboxHeight:  20,
		AIType:        AIPatrol,
		Flying:        false,
	}

	t.Log("=== Ground Check Analysis ===")

	// Test different Y positions relative to the platform
	testPositions := []struct {
		name    string
		x, y    int
		expectOnGround bool
	}{
		{"On platform (feet at y=160)", 160, 136, true},
		{"1px above platform", 160, 135, false},
		{"2px above platform", 160, 134, false},
		{"Off platform to the left", 140, 136, false},
		{"Off platform to the right", 180, 136, false},
	}

	for _, tc := range testPositions {
		t.Run(tc.name, func(t *testing.T) {
			world := NewWorld()
			world.CreatePlayer(100, 500, hitbox, 100)
			enemyID := world.CreateEnemy(tc.x, tc.y, enemyCfg, true)

			// Force OnGround = true to test if ApplyEnemyGravity corrects it
			mov := world.Movement[enemyID]
			mov.OnGround = true
			world.Movement[enemyID] = mov

			pos := world.Position[enemyID]
			hb := world.Hitbox[enemyID]

			// This is what ApplyEnemyGravity checks:
			checkY := pos.PixelY() + hb.OffsetY + hb.Height
			groundExistsLeft := stage.IsSolidAt(pos.PixelX()+hb.OffsetX, checkY)
			groundExistsMid := stage.IsSolidAt(pos.PixelX()+hb.OffsetX+hb.Width/2, checkY)
			groundExistsRight := stage.IsSolidAt(pos.PixelX()+hb.OffsetX+hb.Width-1, checkY)
			groundExists := groundExistsLeft || groundExistsMid || groundExistsRight

			t.Logf("  Enemy at (%d, %d), hitbox feet at y=%d",
				pos.PixelX(), pos.PixelY(), pos.PixelY()+hb.OffsetY+hb.Height)
			t.Logf("  Check Y=%d, ground exists: L=%v M=%v R=%v → %v",
				checkY, groundExistsLeft, groundExistsMid, groundExistsRight, groundExists)

			// Now run ApplyEnemyGravity and check result
			ApplyEnemyGravity(world, stage, 5, 100)

			movAfter := world.Movement[enemyID]
			velAfter := world.Velocity[enemyID]

			t.Logf("  After ApplyEnemyGravity: OnGround=%v, VelY=%d",
				movAfter.OnGround, velAfter.Y)

			if tc.expectOnGround {
				assert.True(t, movAfter.OnGround, "Should be on ground")
				assert.Equal(t, 0, velAfter.Y, "Should not have gravity applied")
			} else {
				assert.False(t, movAfter.OnGround, "Should NOT be on ground")
				assert.Greater(t, velAfter.Y, 0, "Should have gravity applied")
			}
		})
	}
}

// TestEnemySpawnInAir_Debug logs detailed frame-by-frame state to diagnose the bug
func TestEnemySpawnInAir_Debug(t *testing.T) {
	const (
		subStepsPerFrame = 10
		gravityPixelsSec = 800.0
	)

	// Create stage with NO ground
	stage := newMockStage(100, 100, 16)

	world := NewWorld()

	// Create player
	hitbox := HitboxTrapezoid{
		Head: Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	world.CreatePlayer(100, 500, hitbox, 100)

	// Create enemy in mid-air
	enemyCfg := EnemyConfig{
		MaxHealth:     100,
		MoveSpeed:     ToIUPerSubstep(60),
		HitboxOffsetX: 2,
		HitboxOffsetY: 4,
		HitboxWidth:   12,
		HitboxHeight:  20,
		AIType:        AIAggressive,
		Flying:        false,
	}
	enemyID := world.CreateEnemy(500, 100, enemyCfg, true)

	gravity := ToIUAccelPerFrame(gravityPixelsSec)
	maxFall := ToIUPerSubstep(400)
	arrowCfg := ProjectileConfig{}

	t.Logf("=== Enemy Spawn In Air Debug ===")
	t.Logf("Gravity: %d IU/frame", gravity)

	// Simulate 10 frames with detailed logging
	for frame := 0; frame < 10; frame++ {
		posBefore := world.Position[enemyID]
		velBefore := world.Velocity[enemyID]
		movBefore := world.Movement[enemyID]

		// Apply gravity
		ApplyEnemyGravity(world, stage, gravity, maxFall)

		velAfterGravity := world.Velocity[enemyID]
		movAfterGravity := world.Movement[enemyID]

		// Run substeps
		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdateEnemyAI(world, stage, arrowCfg, PhysicsConfig{})
		}

		posAfter := world.Position[enemyID]
		velAfter := world.Velocity[enemyID]
		movAfter := world.Movement[enemyID]

		movedY := posAfter.Y - posBefore.Y
		movedPixels := float64(movedY) / float64(PositionScale)

		t.Logf("Frame %d:", frame)
		t.Logf("  Before: OnGround=%v, VelY=%d", movBefore.OnGround, velBefore.Y)
		t.Logf("  After Gravity: OnGround=%v, VelY=%d", movAfterGravity.OnGround, velAfterGravity.Y)
		t.Logf("  After AI: OnGround=%v, VelY=%d, Moved=%.2f px", movAfter.OnGround, velAfter.Y, movedPixels)
	}
}

// =============================================================================
// Single Frame Movement Check
// =============================================================================

func TestSingleFrameMovement(t *testing.T) {
	t.Run("Player single frame at max speed", func(t *testing.T) {
		const maxSpeedPixels = 120.0

		stage := newMockStage(100, 100, 16)
		world := NewWorld()

		hitbox := HitboxTrapezoid{
			Head: Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
			Body: Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
			Feet: Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
		}
		world.CreatePlayer(500, 500, hitbox, 100)

		// Set velocity directly to max speed
		vel := world.Velocity[world.PlayerID]
		vel.X = ToIUPerSubstep(maxSpeedPixels)
		world.Velocity[world.PlayerID] = vel

		mov := world.Movement[world.PlayerID]
		mov.OnGround = true
		world.Movement[world.PlayerID] = mov

		cfg := PhysicsConfig{
			MaxSpeed:     ToIUPerSubstep(maxSpeedPixels),
			MaxFallSpeed: ToIUPerSubstep(400),
			Gravity:      ToIUAccelPerFrame(800),
		}

		startPos := world.Position[world.PlayerID]

		// Run 10 substeps (1 frame)
		for sub := 0; sub < 10; sub++ {
			UpdatePlayerPhysics(world, stage, cfg)
		}

		endPos := world.Position[world.PlayerID]
		movedIU := endPos.X - startPos.X
		movedPixels := float64(movedIU) / float64(PositionScale)

		// Expected: 120 pixels/sec / 60 fps = 2 pixels per frame
		expectedPixels := maxSpeedPixels / 60.0

		t.Logf("Player moved %.2f pixels in 1 frame (expected %.2f)", movedPixels, expectedPixels)

		assert.InDelta(t, expectedPixels, movedPixels, 0.5,
			"Player should move ~%.2f pixels per frame", expectedPixels)
	})

	t.Run("Projectile single frame", func(t *testing.T) {
		const speedPixels = 300.0

		stage := newMockStage(1000, 100, 16)
		world := NewWorld()

		projCfg := ProjectileConfig{
			GravityAccel:  0,
			MaxFallSpeed:  ToIUPerSubstep(300),
			MaxRange:      10000,
			Damage:        10,
			StuckDuration: 300,
		}

		vx := ToIUPerSubstep(speedPixels)
		projID := world.CreateProjectile(100, 500, vx, 0, projCfg, true)

		startPos := world.Position[projID]

		// Run 10 substeps (1 frame)
		for sub := 0; sub < 10; sub++ {
			UpdateProjectiles(world, stage)
		}

		endPos := world.Position[projID]
		movedIU := endPos.X - startPos.X
		movedPixels := float64(movedIU) / float64(PositionScale)

		expectedPixels := speedPixels / 60.0

		t.Logf("Projectile moved %.2f pixels in 1 frame (expected %.2f)", movedPixels, expectedPixels)

		assert.InDelta(t, expectedPixels, movedPixels, 0.5,
			"Projectile should move ~%.2f pixels per frame", expectedPixels)
	})
}

// =============================================================================
// Enemy Knockback Tests
// =============================================================================

// TestEnemyKnockback_XMovement verifies enemy is pushed horizontally when hit
func TestEnemyKnockback_XMovement(t *testing.T) {
	const (
		subStepsPerFrame = 10
		knockbackForce   = 100 // IU/substep
	)

	stage := newMockStage(100, 100, 16)
	world := NewWorld()

	// Create player (required for AI)
	hitbox := HitboxTrapezoid{
		Head: Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	world.CreatePlayer(100, 500, hitbox, 100)

	// Create enemy
	enemyCfg := EnemyConfig{
		MaxHealth:     100,
		MoveSpeed:     0,
		HitboxOffsetX: 2,
		HitboxOffsetY: 4,
		HitboxWidth:   12,
		HitboxHeight:  20,
		AIType:        AIPatrol,
		Flying:        false,
	}
	enemyID := world.CreateEnemy(500, 500, enemyCfg, true)

	startPos := world.Position[enemyID]
	startPixelX := startPos.PixelX()

	// Simulate knockback: set velocity, HitTimer, and initial knockback values
	vel := world.Velocity[enemyID]
	vel.X = knockbackForce // Push right
	world.Velocity[enemyID] = vel

	ai := world.AI[enemyID]
	ai.HitTimer = 12    // Stun frames
	ai.HitTimerMax = 12 // Initial value for decay calculation
	ai.KnockbackVelX = knockbackForce
	ai.KnockbackVelY = 0
	world.AI[enemyID] = ai

	cfg := PhysicsConfig{}
	arrowCfg := ProjectileConfig{}

	t.Logf("=== Enemy Knockback X Movement Test ===")
	t.Logf("Start: pos=(%d, %d), VelX=%d, HitTimer=%d",
		startPixelX, startPos.PixelY(), knockbackForce, ai.HitTimer)

	// Simulate 10 frames
	for frame := 0; frame < 10; frame++ {
		posBefore := world.Position[enemyID]
		velBefore := world.Velocity[enemyID]

		// Update timers once per frame (includes knockback deceleration)
		UpdateTimers(world)

		for sub := 0; sub < subStepsPerFrame; sub++ {
			UpdateEnemyAI(world, stage, arrowCfg, cfg)
		}

		posAfter := world.Position[enemyID]
		velAfter := world.Velocity[enemyID]
		aiAfter := world.AI[enemyID]

		movedX := posAfter.X - posBefore.X
		movedPixels := float64(movedX) / float64(PositionScale)

		t.Logf("Frame %d: VelX=%d→%d, Moved=%.2f px, HitTimer=%d",
			frame, velBefore.X, velAfter.X, movedPixels, aiAfter.HitTimer)
	}

	endPos := world.Position[enemyID]
	endPixelX := endPos.PixelX()
	totalMoved := endPixelX - startPixelX

	t.Logf("Total X movement: %d pixels", totalMoved)

	// Enemy should have moved right (positive X direction)
	assert.Greater(t, totalMoved, 10,
		"Enemy should move at least 10 pixels from knockback, moved %d", totalMoved)
}

// TestEnemyKnockback_ProportionalDeceleration verifies velocity decreases proportionally to HitTimer
func TestEnemyKnockback_ProportionalDeceleration(t *testing.T) {
	stage := newMockStage(100, 100, 16)
	world := NewWorld()

	hitbox := HitboxTrapezoid{
		Head: Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	world.CreatePlayer(100, 500, hitbox, 100)

	enemyCfg := EnemyConfig{
		MaxHealth:     100,
		MoveSpeed:     0,
		HitboxOffsetX: 2,
		HitboxOffsetY: 4,
		HitboxWidth:   12,
		HitboxHeight:  20,
		AIType:        AIPatrol,
		Flying:        false,
	}
	enemyID := world.CreateEnemy(500, 500, enemyCfg, true)

	// Set initial knockback velocity
	initialVelX := 100
	hitTimerMax := 10

	vel := world.Velocity[enemyID]
	vel.X = initialVelX
	world.Velocity[enemyID] = vel

	ai := world.AI[enemyID]
	ai.HitTimer = hitTimerMax
	ai.HitTimerMax = hitTimerMax
	ai.KnockbackVelX = initialVelX
	ai.KnockbackVelY = 0
	world.AI[enemyID] = ai

	cfg := PhysicsConfig{}
	arrowCfg := ProjectileConfig{}

	velocities := []int{}

	// Record velocity each frame
	for frame := 0; frame < hitTimerMax+1; frame++ {
		vel := world.Velocity[enemyID]
		velocities = append(velocities, vel.X)

		// Update timers once per frame (includes knockback deceleration)
		UpdateTimers(world)

		for sub := 0; sub < 10; sub++ {
			UpdateEnemyAI(world, stage, arrowCfg, cfg)
		}
	}

	t.Logf("Velocities over %d frames: %v", hitTimerMax+1, velocities)

	// Verify proportional deceleration: vel = initialVel * remainingTimer / maxTimer
	// After UpdateTimers in frame 0: HitTimer becomes 9, so vel = 100 * 9/10 = 90
	// After UpdateTimers in frame 1: HitTimer becomes 8, so vel = 100 * 8/10 = 80
	// ...
	expectedVelocities := []int{100, 90, 80, 70, 60, 50, 40, 30, 20, 10, 0}
	for i, expected := range expectedVelocities {
		if i < len(velocities) {
			assert.Equal(t, expected, velocities[i],
				"Frame %d: expected velocity %d, got %d", i, expected, velocities[i])
		}
	}

	// Final velocity should be 0
	finalVel := world.Velocity[enemyID]
	assert.Equal(t, 0, finalVel.X, "Velocity should reach 0 when HitTimer reaches 0")
}

// TestEnemyKnockback_StopsAtWall verifies knockback stops when hitting wall
func TestEnemyKnockback_StopsAtWall(t *testing.T) {
	stage := newMockStage(100, 100, 16)
	// Place wall at tile (33, 31) = pixel (528, 496)
	stage.setSolid(33, 31)

	world := NewWorld()

	hitbox := HitboxTrapezoid{
		Head: Hitbox{OffsetX: 4, OffsetY: 0, Width: 8, Height: 6},
		Body: Hitbox{OffsetX: 2, OffsetY: 6, Width: 12, Height: 12},
		Feet: Hitbox{OffsetX: 0, OffsetY: 18, Width: 16, Height: 6},
	}
	world.CreatePlayer(100, 500, hitbox, 100)

	// Place enemy close to wall (will be pushed into it)
	enemyCfg := EnemyConfig{
		MaxHealth:     100,
		MoveSpeed:     0,
		HitboxOffsetX: 2,
		HitboxOffsetY: 4,
		HitboxWidth:   12,
		HitboxHeight:  20,
		AIType:        AIPatrol,
		Flying:        false,
	}
	enemyID := world.CreateEnemy(510, 496, enemyCfg, true) // Close to wall at x=528

	// Strong knockback pushing right toward wall
	vel := world.Velocity[enemyID]
	vel.X = 200
	world.Velocity[enemyID] = vel

	ai := world.AI[enemyID]
	ai.HitTimer = 20
	ai.HitTimerMax = 20
	ai.KnockbackVelX = 200
	ai.KnockbackVelY = 0
	world.AI[enemyID] = ai

	cfg := PhysicsConfig{}
	arrowCfg := ProjectileConfig{}

	// Simulate several frames
	for frame := 0; frame < 10; frame++ {
		UpdateTimers(world)
		for sub := 0; sub < 10; sub++ {
			UpdateEnemyAI(world, stage, arrowCfg, cfg)
		}
	}

	endPos := world.Position[enemyID]
	endVel := world.Velocity[enemyID]

	t.Logf("Final: pos=(%d, %d), VelX=%d", endPos.PixelX(), endPos.PixelY(), endVel.X)

	// Enemy should have stopped before going through wall
	assert.Less(t, endPos.PixelX(), 528-12, // Wall at 528, enemy width ~12
		"Enemy should stop before wall")
}
