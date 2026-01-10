// Package playing provides the main gameplay scene.
package playing

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/younwookim/mg/internal/application/scene"
	"github.com/younwookim/mg/internal/application/state"
	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/ecs"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

// Colors for rendering
var (
	colorWall       = color.RGBA{80, 80, 100, 255}
	colorSpike      = color.RGBA{200, 50, 50, 255}
	colorPlayer     = color.RGBA{100, 200, 100, 255}
	colorHead       = color.RGBA{100, 100, 200, 128}
	colorFeet       = color.RGBA{200, 200, 100, 128}
	colorBG         = color.RGBA{26, 26, 46, 255}
	colorEnemy      = color.RGBA{200, 100, 100, 255}
	colorEnemyArrow = color.RGBA{255, 100, 100, 255}
	colorGold       = color.RGBA{255, 215, 0, 255}
	colorHealthBG   = color.RGBA{60, 60, 60, 255}
	colorHealthFG   = color.RGBA{100, 200, 100, 255}
)

// Playing is the main gameplay scene
type Playing struct {
	config   *config.GameConfig
	stageCfg *config.StageConfig
	stage    *entity.Stage
	state    state.GameState
	world    *ecs.World
	screenW  int
	screenH  int
	tileSize int

	// Physics config for ECS systems
	physicsCfg ecs.PhysicsConfig
	arrowCfg   ecs.ProjectileConfig

	// Feedback
	hitstopFrames int
	screenShakeX  float64
	screenShakeY  float64
	shakeDecay    float64

	// Mouse aiming
	mouseWorldX float64
	mouseWorldY float64

	// Arrow selection UI (keep entity package for UI)
	arrowSelectUI *entity.ArrowSelectUI

	// Deterministic RNG
	rng  *rand.Rand
	seed int64

	// Input recording
	recorder       *Recorder
	recordFilename string

	// Enemy spawner
	spawnTimer  int
	nextEnemyID ecs.EntityID
}

// New creates a new Playing scene.
// If recordPath is not empty, gameplay will be recorded.
func New(cfg *config.GameConfig, stageCfg *config.StageConfig, stage *entity.Stage, recordPath string) *Playing {
	// Initialize seeded RNG for deterministic randomness
	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))

	// Create ECS world
	world := ecs.NewWorld()

	// Create player hitbox from config
	playerCfg := cfg.Entities.Player
	hitbox := ecs.HitboxTrapezoid{
		Head: ecs.Hitbox{
			OffsetX: playerCfg.Hitbox.Head.OffsetX,
			OffsetY: playerCfg.Hitbox.Head.OffsetY,
			Width:   playerCfg.Hitbox.Head.Width,
			Height:  playerCfg.Hitbox.Head.Height,
		},
		Body: ecs.Hitbox{
			OffsetX: playerCfg.Hitbox.Body.OffsetX,
			OffsetY: playerCfg.Hitbox.Body.OffsetY,
			Width:   playerCfg.Hitbox.Body.Width,
			Height:  playerCfg.Hitbox.Body.Height,
		},
		Feet: ecs.Hitbox{
			OffsetX: playerCfg.Hitbox.Feet.OffsetX,
			OffsetY: playerCfg.Hitbox.Feet.OffsetY,
			Width:   playerCfg.Hitbox.Feet.Width,
			Height:  playerCfg.Hitbox.Feet.Height,
		},
	}

	// Create player entity
	world.CreatePlayer(stage.SpawnX, stage.SpawnY, hitbox, playerCfg.Stats.MaxHealth)

	// Build physics config for ECS
	physicsCfg := buildPhysicsConfig(cfg)

	// Build arrow config
	arrowCfg := buildArrowConfig(cfg)

	// Create arrow select UI with config
	arrowSelectCfg := entity.ArrowSelectConfig{
		Radius:      cfg.Physics.ArrowSelect.Radius,
		MinDistance: cfg.Physics.ArrowSelect.MinDistance,
		MaxFrame:    cfg.Physics.ArrowSelect.MaxFrame,
	}

	p := &Playing{
		config:         cfg,
		stageCfg:       stageCfg,
		stage:          stage,
		state:          state.StatePlaying,
		world:          world,
		screenW:        cfg.Physics.Display.ScreenWidth,
		screenH:        cfg.Physics.Display.ScreenHeight,
		tileSize:       stage.TileSize,
		physicsCfg:     physicsCfg,
		arrowCfg:       arrowCfg,
		shakeDecay:     cfg.Physics.Feedback.ScreenShake.Decay,
		arrowSelectUI:  entity.NewArrowSelectUIWithConfig(arrowSelectCfg),
		rng:            rng,
		seed:           seed,
		recordFilename: recordPath,
	}

	// Initialize recorder if recording is enabled
	if recordPath != "" {
		p.recorder = NewRecorder(seed, stageCfg.Name)
		log.Printf("Recording enabled: %s (seed: %d)", recordPath, seed)
	}

	// Spawn enemies from stage config
	for _, spawn := range stageCfg.Enemies {
		p.spawnEnemy(spawn.X, spawn.Y, spawn.Type, spawn.FacingRight)
	}

	// Initialize enemy ID counter for spawner
	p.nextEnemyID = ecs.EntityID(len(stageCfg.Enemies) + 2) // +2 because player is ID 1

	return p
}

func buildPhysicsConfig(cfg *config.GameConfig) ecs.PhysicsConfig {
	return ecs.PhysicsConfig{
		// Physics
		// Gravity: acceleration (pixels/sec²) → IU velocity change per frame
		Gravity:      ecs.ToIUAccelPerFrame(cfg.Physics.Physics.Gravity),
		MaxFallSpeed: ecs.ToIUPerSubstep(cfg.Physics.Physics.MaxFallSpeed),

		// Movement
		MaxSpeed: ecs.ToIUPerSubstep(cfg.Physics.Movement.MaxSpeed),
		// Acceleration/Deceleration: pixels/sec² → IU velocity change per frame
		Acceleration:  ecs.ToIUAccelPerFrame(cfg.Physics.Movement.Acceleration),
		Deceleration:  ecs.ToIUAccelPerFrame(cfg.Physics.Movement.Deceleration),
		AirControlPct: ecs.PctToInt(cfg.Physics.Movement.AirControl),
		TurnaroundPct: ecs.PctToInt(cfg.Physics.Movement.TurnaroundBoost),

		// Jump
		JumpForce:         ecs.ToIUPerSubstep(cfg.Physics.Jump.Force),
		VarJumpPct:        ecs.PctToInt(cfg.Physics.Jump.VariableJumpMultiplier),
		CoyoteFrames:      int(cfg.Physics.Jump.CoyoteTime * 60),
		JumpBufferFrames:  int(cfg.Physics.Jump.JumpBuffer * 60),
		ApexModEnabled:    cfg.Physics.Jump.ApexModifier.Enabled,
		ApexThreshold:     ecs.ToIUPerSubstep(cfg.Physics.Jump.ApexModifier.Threshold),
		ApexGravityPct:    ecs.PctToInt(cfg.Physics.Jump.ApexModifier.GravityMultiplier),
		FallMultiplierPct: ecs.PctToInt(cfg.Physics.Jump.FallMultiplier),

		// Dash
		DashSpeed:          ecs.ToIUPerSubstep(cfg.Physics.Dash.Speed),
		DashFrames:         int(cfg.Physics.Dash.Duration * 60),
		DashCooldownFrames: int(cfg.Physics.Dash.Cooldown * 60),
		DashIframes:        int(cfg.Physics.Dash.IframesDuration * 60),

		// Collision
		CornerCorrectionMargin:  cfg.Physics.Collision.CornerCorrection.Margin,
		CornerCorrectionEnabled: cfg.Physics.Collision.CornerCorrection.Enabled,
	}
}

func buildArrowConfig(cfg *config.GameConfig) ecs.ProjectileConfig {
	arrowCfg := cfg.Entities.Projectiles["playerArrow"]
	return ecs.ProjectileConfig{
		GravityAccel:  ecs.ToIUAccelPerFrame(arrowCfg.Physics.GravityAccel),
		MaxFallSpeed:  ecs.ToIUPerSubstep(arrowCfg.Physics.MaxFallSpeed),
		MaxRange:      int(arrowCfg.Physics.MaxRange),
		Damage:        arrowCfg.Damage,
		HitboxOffsetX: 2,
		HitboxOffsetY: 2,
		HitboxWidth:   12,
		HitboxHeight:  4,
		StuckDuration: 300, // 5 seconds at 60fps
	}
}

func (p *Playing) spawnEnemy(x, y int, enemyType string, facingRight bool) {
	enemyCfg, ok := p.config.Entities.Enemies[enemyType]
	if !ok {
		return
	}

	aiType := ecs.AIPatrol
	switch enemyCfg.AI.Type {
	case "patrol":
		aiType = ecs.AIPatrol
	case "ranged":
		aiType = ecs.AIRanged
	case "chase":
		aiType = ecs.AIChase
	case "aggressive":
		aiType = ecs.AIAggressive
	}

	ecsCfg := ecs.EnemyConfig{
		MaxHealth:     enemyCfg.Stats.MaxHealth,
		ContactDamage: enemyCfg.Stats.ContactDamage,
		MoveSpeed:     ecs.ToIUPerSubstep(enemyCfg.Stats.MoveSpeed),
		HitboxOffsetX: enemyCfg.Hitbox.Body.OffsetX,
		HitboxOffsetY: enemyCfg.Hitbox.Body.OffsetY,
		HitboxWidth:   enemyCfg.Hitbox.Body.Width,
		HitboxHeight:  enemyCfg.Hitbox.Body.Height,
		AIType:        aiType,
		DetectRange:   int(enemyCfg.AI.DetectRange),
		PatrolDist:    int(enemyCfg.AI.PatrolDistance),
		AttackRange:   int(enemyCfg.AI.AttackRange),
		JumpForce:     ecs.ToIUPerSubstep(enemyCfg.AI.JumpForce),
		Flying:        enemyCfg.AI.Flying,
		GoldDropMin:   enemyCfg.Stats.GoldDrop.Min,
		GoldDropMax:   enemyCfg.Stats.GoldDrop.Max,
	}

	p.world.CreateEnemy(x, y, ecsCfg, facingRight)
}

// Update proceeds the game state (implements scene.Scene)
func (p *Playing) Update(_ float64) (scene.Scene, error) {
	// Handle hitstop
	if p.hitstopFrames > 0 {
		p.hitstopFrames--
		return nil, nil
	}

	switch p.state {
	case state.StatePlaying:
		p.updatePlaying()
	case state.StatePaused:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			p.state = state.StatePlaying
		}
	case state.StateGameOver:
		if inpututil.IsKeyJustPressed(ebiten.KeyZ) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			p.restart()
		}
	}

	return nil, nil // nil = stay on this scene
}

func (p *Playing) updatePlaying() {
	// Check for pause
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.state = state.StatePaused
		return
	}

	// F5: Save recording manually
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) && p.recorder != nil {
		p.saveRecording()
	}

	// Get input
	input := p.getInput()

	// Record input if recording is enabled
	if p.recorder != nil {
		p.recorder.RecordFrame(RecordableInput{
			Left:               input.Left,
			Right:              input.Right,
			Up:                 input.Up,
			Down:               input.Down,
			Jump:               input.Up, // W key is both up and jump
			JumpPressed:        input.JumpPressed,
			JumpReleased:       input.JumpReleased,
			Dash:               input.Dash,
			MouseX:             input.MouseX,
			MouseY:             input.MouseY,
			MouseClick:         inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
			RightClickPressed:  inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
			RightClickReleased: inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight),
		})
	}

	// Update arrow selection UI (always, for animation)
	p.arrowSelectUI.Update(
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
		inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight),
		input.MouseX, input.MouseY, p.screenW, p.screenH,
	)

	// Get player data for arrow selection
	playerData := p.world.PlayerData[p.world.PlayerID]

	// Update highlight based on mouse position
	if p.arrowSelectUI.IsActive() {
		selectedDir := p.arrowSelectUI.UpdateHighlight(input.MouseX, input.MouseY)

		// On right click release, confirm selection
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight) && selectedDir != entity.DirNone {
			playerData.CurrentArrow = ecs.ArrowType(selectedDir)
			p.world.PlayerData[p.world.PlayerID] = playerData
		}
	}

	// Calculate camera offset for mouse world position
	camX, camY := p.getCameraOffset()

	// Convert mouse screen position to world position
	p.mouseWorldX = float64(input.MouseX + camX)
	p.mouseWorldY = float64(input.MouseY + camY)

	// Handle attack (mouse click) - only when arrow selection UI is not active
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !p.arrowSelectUI.IsActive() {
		pos := p.world.Position[p.world.PlayerID]
		vel := p.world.Velocity[p.world.PlayerID]
		mov := p.world.Movement[p.world.PlayerID]

		arrowX := pos.PixelX() + 8
		arrowY := pos.PixelY() + 10

		// Player velocity is already in IU/substep
		playerVX := vel.X
		playerVY := vel.Y
		if mov.OnGround {
			playerVY = 0
		}

		p.spawnPlayerArrow(arrowX, arrowY, int(p.mouseWorldX), int(p.mouseWorldY), playerVX, playerVY)
	}

	// Update ECS systems
	subSteps := 10
	if p.arrowSelectUI.IsActive() {
		subSteps = 1 // Slow motion during arrow select
	}

	// Update timers (once per frame)
	ecs.UpdateTimers(p.world)

	// Update player input (once per frame)
	ecs.UpdatePlayerInput(p.world, ecs.InputState{
		Left:         input.Left,
		Right:        input.Right,
		Up:           input.Up,
		Down:         input.Down,
		JumpPressed:  input.JumpPressed,
		JumpReleased: input.JumpReleased,
		Dash:         input.Dash,
	}, p.physicsCfg)

	// Apply gravity once per frame (before substep loop)
	ecs.ApplyPlayerGravity(p.world, p.physicsCfg)
	ecs.ApplyEnemyGravity(p.world, p.stage, p.physicsCfg.Gravity, p.physicsCfg.MaxFallSpeed)
	ecs.ApplyProjectileGravity(p.world)
	ecs.ApplyGoldGravity(p.world)

	// Substep loop: movement and collision per substep
	// subSteps=10 is normal speed, subSteps=1 is 10x slow motion
	for i := 0; i < subSteps; i++ {
		ecs.UpdatePlayerPhysics(p.world, p.stage, p.physicsCfg)
		ecs.UpdateEnemyAI(p.world, p.stage, p.arrowCfg, p.physicsCfg)
		ecs.UpdateProjectiles(p.world, p.stage)
		ecs.UpdateGoldPhysics(p.world, p.stage)
	}

	// Collect gold
	ecs.CollectGold(p.world)

	// Update damage
	knockbackForce := ecs.ToIUPerSubstep(p.config.Physics.Combat.Knockback.Force)
	knockbackUp := ecs.ToIUPerSubstep(p.config.Physics.Combat.Knockback.UpForce)
	iframeFrames := int(p.config.Physics.Combat.Iframes * 60)
	result := ecs.UpdateDamage(p.world, knockbackForce, knockbackUp, iframeFrames)

	// Handle damage feedback
	if result.HitstopFrames > 0 {
		p.hitstopFrames = result.HitstopFrames
	}
	if result.ScreenShake > 0 {
		p.screenShakeX = result.ScreenShake
		p.screenShakeY = result.ScreenShake
	}

	// Resolve enemy collisions
	ecs.ResolveEnemyCollisions(p.world)

	// Check spike damage
	p.checkSpikeDamage()

	// Decay screen shake
	p.screenShakeX *= p.shakeDecay
	p.screenShakeY *= p.shakeDecay

	// Spawn enemies periodically (max 10 active enemies)
	p.spawnTimer++
	if p.spawnTimer >= 30 {
		p.spawnTimer = 0
		if p.world.CountEnemies() < 10 {
			p.spawnEnemyOnRight()
		}
	}

	// Check game over
	health := p.world.Health[p.world.PlayerID]
	if health.Current <= 0 {
		p.state = state.StateGameOver
		// Auto-save recording on game over
		if p.recorder != nil {
			p.saveRecording()
		}
	}
}

type inputState struct {
	Left, Right, Up, Down bool
	JumpPressed           bool
	JumpReleased          bool
	Dash                  bool
	MouseX, MouseY        int
}

func (p *Playing) getInput() inputState {
	mx, my := ebiten.CursorPosition()
	return inputState{
		Left:         ebiten.IsKeyPressed(ebiten.KeyA),
		Right:        ebiten.IsKeyPressed(ebiten.KeyD),
		Up:           ebiten.IsKeyPressed(ebiten.KeyW),
		Down:         ebiten.IsKeyPressed(ebiten.KeyS),
		JumpPressed:  inpututil.IsKeyJustPressed(ebiten.KeyW),
		JumpReleased: inpututil.IsKeyJustReleased(ebiten.KeyW),
		Dash:         inpututil.IsKeyJustPressed(ebiten.KeySpace),
		MouseX:       mx,
		MouseY:       my,
	}
}

func (p *Playing) spawnPlayerArrow(x, y, targetX, targetY int, playerVX, playerVY int) {
	arrowCfg := p.config.Entities.Projectiles["playerArrow"]
	velocityInfluence := p.config.Physics.Projectile.VelocityInfluence

	// Calculate direction (use float for normalization, convert to int at end)
	dx := float64(targetX - x)
	dy := float64(targetY - y)
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1 {
		dist = 1
	}

	// Convert speed to IU/substep
	speedIU := ecs.ToIUPerSubstep(arrowCfg.Physics.Speed)

	// Calculate velocity components
	vxf := (dx / dist) * float64(speedIU)
	vyf := (dy / dist) * float64(speedIU)

	// Add player velocity influence (velocityInfluence is 0.0-1.0)
	vxf += float64(playerVX) * velocityInfluence
	vyf += float64(playerVY) * velocityInfluence

	// Convert to int
	vx := int(vxf)
	vy := int(vyf)

	cfg := ecs.ProjectileConfig{
		GravityAccel:  ecs.ToIUAccelPerFrame(arrowCfg.Physics.GravityAccel),
		MaxFallSpeed:  ecs.ToIUPerSubstep(arrowCfg.Physics.MaxFallSpeed),
		MaxRange:      int(arrowCfg.Physics.MaxRange),
		Damage:        arrowCfg.Damage,
		HitboxOffsetX: 2,
		HitboxOffsetY: 2,
		HitboxWidth:   12,
		HitboxHeight:  4,
		StuckDuration: 300, // 5 seconds
	}

	p.world.CreateProjectile(x, y, vx, vy, cfg, true)
}

func (p *Playing) getCameraOffset() (int, int) {
	pos := p.world.Position[p.world.PlayerID]
	camX := pos.PixelX() - p.screenW/2 + 8
	camY := pos.PixelY() - p.screenH/2 + 12
	if camX < 0 {
		camX = 0
	}
	if camY < 0 {
		camY = 0
	}
	maxCamX := p.stage.Width*p.tileSize - p.screenW
	maxCamY := p.stage.Height*p.tileSize - p.screenH
	if camX > maxCamX {
		camX = maxCamX
	}
	if camY > maxCamY {
		camY = maxCamY
	}
	return camX, camY
}

// saveRecording saves the current recording to file
func (p *Playing) saveRecording() {
	if p.recorder == nil {
		return
	}

	filename := p.recordFilename
	if filename == "" {
		filename = GenerateFilename()
	}

	if err := p.recorder.Save(filename); err != nil {
		log.Printf("Failed to save recording: %v", err)
	} else {
		log.Printf("Recording saved: %s (%d frames)", filename, p.recorder.FrameCount())
	}
}

func (p *Playing) checkSpikeDamage() {
	playerID := p.world.PlayerID
	playerData := p.world.PlayerData[playerID]
	dash := p.world.Dash[playerID]

	if playerData.IsInvincible(dash.Active) {
		return
	}

	pos := p.world.Position[playerID]
	hitbox := p.world.HitboxTrapezoid[playerID]
	facing := p.world.Facing[playerID]

	fx, fy, fw, fh := hitbox.Feet.GetWorldRect(pos.PixelX(), pos.PixelY(), facing.Right, 16)

	for py := fy; py < fy+fh; py++ {
		for px := fx; px < fx+fw; px++ {
			tile := p.stage.GetTileAtPixel(px, py)
			if tile.Type == entity.TileSpike {
				health := p.world.Health[playerID]
				health.Current -= tile.Damage
				p.world.Health[playerID] = health

				playerData.IframeTimer = int(p.config.Physics.Combat.Iframes * 60)
				p.world.PlayerData[playerID] = playerData

				vel := p.world.Velocity[playerID]
				vel.Y = -150 * ecs.PositionScale
				p.world.Velocity[playerID] = vel

				p.screenShakeX = p.config.Physics.Feedback.ScreenShake.Intensity
				p.screenShakeY = p.config.Physics.Feedback.ScreenShake.Intensity
				return
			}
		}
	}
}

func (p *Playing) spawnEnemyOnRight() {
	spawnX := (p.stage.Width - 3) * p.tileSize

	maxAttempts := 20
	for i := 0; i < maxAttempts; i++ {
		tileY := 1 + p.rng.Intn(p.stage.Height-2)
		spawnY := tileY * p.tileSize

		if !p.stage.IsSolidAt(spawnX, spawnY) && !p.stage.IsSolidAt(spawnX, spawnY+p.tileSize-1) {
			hasGround := false
			for checkY := spawnY + p.tileSize; checkY < p.stage.Height*p.tileSize; checkY += p.tileSize {
				if p.stage.IsSolidAt(spawnX, checkY) {
					hasGround = true
					break
				}
			}
			if hasGround {
				p.spawnEnemy(spawnX, spawnY, "berserker", false)
				p.nextEnemyID++
				return
			}
		}
	}
}

func (p *Playing) restart() {
	// Reset RNG with new seed
	p.seed = time.Now().UnixNano()
	p.rng = rand.New(rand.NewSource(p.seed))

	// Create new world
	p.world = ecs.NewWorld()

	// Create player
	playerCfg := p.config.Entities.Player
	hitbox := ecs.HitboxTrapezoid{
		Head: ecs.Hitbox{
			OffsetX: playerCfg.Hitbox.Head.OffsetX,
			OffsetY: playerCfg.Hitbox.Head.OffsetY,
			Width:   playerCfg.Hitbox.Head.Width,
			Height:  playerCfg.Hitbox.Head.Height,
		},
		Body: ecs.Hitbox{
			OffsetX: playerCfg.Hitbox.Body.OffsetX,
			OffsetY: playerCfg.Hitbox.Body.OffsetY,
			Width:   playerCfg.Hitbox.Body.Width,
			Height:  playerCfg.Hitbox.Body.Height,
		},
		Feet: ecs.Hitbox{
			OffsetX: playerCfg.Hitbox.Feet.OffsetX,
			OffsetY: playerCfg.Hitbox.Feet.OffsetY,
			Width:   playerCfg.Hitbox.Feet.Width,
			Height:  playerCfg.Hitbox.Feet.Height,
		},
	}
	p.world.CreatePlayer(p.stage.SpawnX, p.stage.SpawnY, hitbox, playerCfg.Stats.MaxHealth)

	p.state = state.StatePlaying

	// Reset UI
	p.arrowSelectUI = entity.NewArrowSelectUIWithConfig(entity.ArrowSelectConfig{
		Radius:      p.config.Physics.ArrowSelect.Radius,
		MinDistance: p.config.Physics.ArrowSelect.MinDistance,
		MaxFrame:    p.config.Physics.ArrowSelect.MaxFrame,
	})

	// Respawn enemies
	for _, spawn := range p.stageCfg.Enemies {
		p.spawnEnemy(spawn.X, spawn.Y, spawn.Type, spawn.FacingRight)
	}

	// Reset spawner
	p.spawnTimer = 0
	p.nextEnemyID = ecs.EntityID(len(p.stageCfg.Enemies) + 2)

	// Reset recorder if recording
	if p.recordFilename != "" {
		p.recorder = NewRecorder(p.seed, p.stageCfg.Name)
		log.Printf("Recording restarted (seed: %d)", p.seed)
	}
}

// Draw renders the game screen
func (p *Playing) Draw(screen *ebiten.Image) {
	screen.Fill(colorBG)

	camX, camY := p.getCameraOffset()

	// Apply screen shake
	camX += int(p.screenShakeX * (2*randFloat() - 1))
	camY += int(p.screenShakeY * (2*randFloat() - 1))

	// Clamp camera
	maxCamX := p.stage.Width*p.tileSize - p.screenW
	maxCamY := p.stage.Height*p.tileSize - p.screenH
	if camX < 0 {
		camX = 0
	}
	if camY < 0 {
		camY = 0
	}
	if camX > maxCamX {
		camX = maxCamX
	}
	if camY > maxCamY {
		camY = maxCamY
	}

	// Draw world
	p.drawTiles(screen, camX, camY)
	p.drawGolds(screen, camX, camY)
	p.drawEnemies(screen, camX, camY)
	p.drawProjectiles(screen, camX, camY)
	p.drawPlayer(screen, camX, camY)
	p.drawTrajectory(screen, camX, camY)

	// Draw dark overlay when arrow selection UI is active
	if p.arrowSelectUI.IsActive() {
		p.drawArrowSelectOverlay(screen)
	}

	// Draw arrow selection UI
	if p.arrowSelectUI.IsActive() {
		p.drawArrowSelectUI(screen)
	}

	// Draw UI (HP bar, current arrow, etc.) - always on top
	p.drawUI(screen)

	// Draw state overlays
	switch p.state {
	case state.StatePaused:
		p.drawPauseOverlay(screen)
	case state.StateGameOver:
		p.drawGameOverOverlay(screen)
	}
}

func (p *Playing) drawTiles(screen *ebiten.Image, camX, camY int) {
	startTileX := camX / p.tileSize
	startTileY := camY / p.tileSize
	endTileX := (camX + p.screenW) / p.tileSize + 1
	endTileY := (camY + p.screenH) / p.tileSize + 1

	for ty := startTileY; ty <= endTileY && ty < p.stage.Height; ty++ {
		for tx := startTileX; tx <= endTileX && tx < p.stage.Width; tx++ {
			if tx < 0 || ty < 0 {
				continue
			}
			tile := p.stage.GetTile(tx, ty)
			if tile.Type == entity.TileEmpty {
				continue
			}

			x := float64(tx*p.tileSize - camX)
			y := float64(ty*p.tileSize - camY)

			var c color.Color
			switch tile.Type {
			case entity.TileWall:
				c = colorWall
			case entity.TileSpike:
				c = colorSpike
			}

			ebitenutil.DrawRect(screen, x, y, float64(p.tileSize), float64(p.tileSize), c)
		}
	}
}

func (p *Playing) drawPlayer(screen *ebiten.Image, camX, camY int) {
	pos := p.world.Position[p.world.PlayerID]
	playerData := p.world.PlayerData[p.world.PlayerID]
	facing := p.world.Facing[p.world.PlayerID]
	dash := p.world.Dash[p.world.PlayerID]

	playerScreenX := float64(pos.PixelX() - camX)
	playerScreenY := float64(pos.PixelY() - camY)

	playerW := float64(p.config.Entities.Player.Sprite.FrameWidth)
	playerH := float64(p.config.Entities.Player.Sprite.FrameHeight)

	// Flash when invincible
	playerColor := colorPlayer
	if playerData.IsInvincible(dash.Active) && playerData.IframeTimer%6 < 3 {
		playerColor = color.RGBA{255, 255, 255, 200}
	}

	ebitenutil.DrawRect(screen, playerScreenX, playerScreenY, playerW, playerH, playerColor)

	// Draw hitbox debug
	if ebiten.IsKeyPressed(ebiten.KeyTab) {
		hitbox := p.world.HitboxTrapezoid[p.world.PlayerID]
		hx, hy, hw, hh := hitbox.Head.GetWorldRect(pos.PixelX(), pos.PixelY(), facing.Right, 16)
		ebitenutil.DrawRect(screen, float64(hx-camX), float64(hy-camY), float64(hw), float64(hh), colorHead)

		fx, fy, fw, fh := hitbox.Feet.GetWorldRect(pos.PixelX(), pos.PixelY(), facing.Right, 16)
		ebitenutil.DrawRect(screen, float64(fx-camX), float64(fy-camY), float64(fw), float64(fh), colorFeet)
	}
}

func (p *Playing) drawEnemies(screen *ebiten.Image, camX, camY int) {
	for id := range p.world.IsEnemy {
		pos := p.world.Position[id]
		ai := p.world.AI[id]
		hitbox := p.world.Hitbox[id]

		x := float64(pos.PixelX() - camX)
		y := float64(pos.PixelY() - camY)

		// Flash on hit
		c := colorEnemy
		if ai.HitTimer > 0 {
			c = color.RGBA{255, 255, 255, 255}
		}

		ebitenutil.DrawRect(screen, x, y, float64(hitbox.Width+4), float64(hitbox.Height+4), c)
	}
}

func (p *Playing) drawProjectiles(screen *ebiten.Image, camX, camY int) {
	playerData := p.world.PlayerData[p.world.PlayerID]

	for id := range p.world.IsProjectile {
		pos := p.world.Position[id]
		vel := p.world.Velocity[id]
		proj := p.world.ProjectileData[id]

		x := float64(pos.PixelX() - camX)
		y := float64(pos.PixelY() - camY)

		// Determine color
		var c color.RGBA
		if proj.IsPlayerOwned {
			c = ecs.ArrowColors[playerData.CurrentArrow]
		} else {
			c = colorEnemyArrow
		}

		// Apply alpha for fading
		alpha := proj.GetAlpha()
		c = color.RGBA{
			uint8(float64(c.R) * alpha),
			uint8(float64(c.G) * alpha),
			uint8(float64(c.B) * alpha),
			uint8(float64(c.A) * alpha),
		}

		// Draw rotated arrow
		rot := proj.Rotation(vel.X, vel.Y)
		length := 12.0
		prevX := x - math.Cos(rot)*length
		prevY := y - math.Sin(rot)*length

		ebitenutil.DrawRect(screen, x-2, y-2, 4, 4, c)
		ebitenutil.DrawLine(screen, x, y, prevX, prevY, c)
	}
}

func (p *Playing) drawGolds(screen *ebiten.Image, camX, camY int) {
	for id := range p.world.IsGold {
		pos := p.world.Position[id]

		x := float64(pos.PixelX() - camX)
		y := float64(pos.PixelY() - camY)

		ebitenutil.DrawRect(screen, x, y, 8, 8, colorGold)
	}
}

func (p *Playing) drawUI(screen *ebiten.Image) {
	health := p.world.Health[p.world.PlayerID]
	playerData := p.world.PlayerData[p.world.PlayerID]

	// Health bar
	barX := 10.0
	barY := float64(p.screenH - 20)
	barW := 100.0
	barH := 10.0

	ebitenutil.DrawRect(screen, barX, barY, barW, barH, colorHealthBG)

	healthRatio := float64(health.Current) / float64(health.Max)
	if healthRatio < 0 {
		healthRatio = 0
	}
	ebitenutil.DrawRect(screen, barX, barY, barW*healthRatio, barH, colorHealthFG)

	// Current arrow indicator
	p.drawArrowIcon(screen, barX+barW+10, barY+barH/2, playerData.CurrentArrow, 1.0, true)

	// Gold
	goldText := fmt.Sprintf("Gold: %d", playerData.Gold)
	ebitenutil.DebugPrintAt(screen, goldText, 10, p.screenH-35)

	// Controls
	debugText := "A/D: Move | W: Jump | Space: Dash | LClick: Attack | RClick: Arrow Select | ESC: Pause"
	ebitenutil.DebugPrint(screen, debugText)
}

func (p *Playing) drawPauseOverlay(screen *ebiten.Image) {
	overlay := color.RGBA{0, 0, 0, 128}
	ebitenutil.DrawRect(screen, 0, 0, float64(p.screenW), float64(p.screenH), overlay)

	text := "PAUSED\n\nPress ESC to resume"
	ebitenutil.DebugPrintAt(screen, text, p.screenW/2-50, p.screenH/2-20)
}

func (p *Playing) drawGameOverOverlay(screen *ebiten.Image) {
	playerData := p.world.PlayerData[p.world.PlayerID]

	overlay := color.RGBA{100, 0, 0, 180}
	ebitenutil.DrawRect(screen, 0, 0, float64(p.screenW), float64(p.screenH), overlay)

	text := fmt.Sprintf("GAME OVER\n\nGold collected: %d\n\nPress Z to restart", playerData.Gold)
	ebitenutil.DebugPrintAt(screen, text, p.screenW/2-60, p.screenH/2-30)
}

func (p *Playing) drawArrowSelectOverlay(screen *ebiten.Image) {
	progress := p.arrowSelectUI.GetProgress()
	easedProgress := math.Sin(progress * math.Pi / 2)

	alpha := uint8(128 * easedProgress)
	overlay := color.RGBA{0, 0, 0, alpha}
	ebitenutil.DrawRect(screen, 0, 0, float64(p.screenW), float64(p.screenH), overlay)
}

func (p *Playing) drawArrowSelectUI(screen *ebiten.Image) {
	progress := p.arrowSelectUI.GetProgress()
	easedProgress := math.Sin(progress * math.Pi / 2)
	playerData := p.world.PlayerData[p.world.PlayerID]

	for dir := entity.DirRight; dir <= entity.DirDown; dir++ {
		arrowType := playerData.EquippedArrows[int(dir)]
		x, y := p.arrowSelectUI.GetIconPosition(dir, easedProgress)

		brightness := 0.7
		if arrowType == playerData.CurrentArrow {
			brightness = 1.0
		}
		if dir == p.arrowSelectUI.Highlighted {
			brightness = 1.0
		}

		p.drawArrowIcon(screen, x, y, arrowType, brightness*easedProgress, dir == p.arrowSelectUI.Highlighted)
	}
}

func (p *Playing) drawArrowIcon(screen *ebiten.Image, x, y float64, arrowType ecs.ArrowType, brightness float64, large bool) {
	baseColor := ecs.ArrowColors[arrowType]

	c := color.RGBA{
		uint8(float64(baseColor.R) * brightness),
		uint8(float64(baseColor.G) * brightness),
		uint8(float64(baseColor.B) * brightness),
		uint8(float64(baseColor.A) * brightness),
	}

	length := 12.0
	if large {
		length = 16.0
	}

	tipX := x + length/2
	tipY := y
	tailX := x - length/2
	tailY := y

	ebitenutil.DrawLine(screen, tailX, tailY, tipX, tipY, c)

	tipSize := 4.0
	if large {
		tipSize = 5.0
	}
	ebitenutil.DrawLine(screen, tipX, tipY, tipX-tipSize, tipY-tipSize/2, c)
	ebitenutil.DrawLine(screen, tipX, tipY, tipX-tipSize, tipY+tipSize/2, c)

	ebitenutil.DrawRect(screen, tipX-1, tipY-1, 2, 2, c)
}

func (p *Playing) drawTrajectory(screen *ebiten.Image, camX, camY int) {
	arrowCfg := p.config.Entities.Projectiles["playerArrow"]
	speed := arrowCfg.Physics.Speed
	gravity := arrowCfg.Physics.GravityAccel
	maxFall := arrowCfg.Physics.MaxFallSpeed
	maxRange := arrowCfg.Physics.MaxRange
	velocityInfluence := p.config.Physics.Projectile.VelocityInfluence

	pos := p.world.Position[p.world.PlayerID]
	vel := p.world.Velocity[p.world.PlayerID]
	mov := p.world.Movement[p.world.PlayerID]
	playerData := p.world.PlayerData[p.world.PlayerID]

	startX := float64(pos.PixelX() + 8)
	startY := float64(pos.PixelY() + 10)

	dx := p.mouseWorldX - startX
	dy := p.mouseWorldY - startY
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1 {
		dist = 1
	}
	vx := (dx / dist) * speed
	vy := (dy / dist) * speed

	playerVX := float64(vel.X) / float64(ecs.PositionScale)
	playerVY := float64(vel.Y) / float64(ecs.PositionScale)
	if mov.OnGround {
		playerVY = 0
	}
	vx += playerVX * velocityInfluence
	vy += playerVY * velocityInfluence

	arrowColor := ecs.ArrowColors[playerData.CurrentArrow]
	trajectoryColor := color.RGBA{
		uint8((int(arrowColor.R) + 255) / 2),
		uint8((int(arrowColor.G) + 255) / 2),
		uint8((int(arrowColor.B) + 255) / 2),
		200,
	}

	x, y := startX, startY
	dt := 1.0 / 60.0
	dotSpacing := 8.0
	accumulated := 0.0
	dotSize := 3.0

	for traveled := 0.0; traveled < maxRange; {
		vy += gravity * dt
		if vy > maxFall {
			vy = maxFall
		}

		prevX, prevY := x, y

		x += vx * dt
		y += vy * dt

		stepDx := x - prevX
		stepDy := y - prevY
		stepDist := math.Sqrt(stepDx*stepDx + stepDy*stepDy)
		traveled += stepDist
		accumulated += stepDist

		if p.stage.IsSolidAt(int(x), int(y)) {
			break
		}

		if accumulated >= dotSpacing {
			accumulated -= dotSpacing
			screenX := x - float64(camX) - dotSize/2
			screenY := y - float64(camY) - dotSize/2
			ebitenutil.DrawRect(screen, screenX, screenY, dotSize, dotSize, trajectoryColor)
		}
	}
}

// OnEnter is called when entering this scene
func (p *Playing) OnEnter() {
	// Scene is already initialized in New
}

// OnExit is called when leaving this scene
func (p *Playing) OnExit() {
	p.saveRecording()
}

// Layout returns the game's screen dimensions
func (p *Playing) Layout(outsideWidth, outsideHeight int) (int, int) {
	return p.screenW, p.screenH
}

var randState uint32 = 1

func randFloat() float64 {
	randState = randState*1103515245 + 12345
	return float64(randState&0x7fffffff) / float64(0x7fffffff)
}

