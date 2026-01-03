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
	"github.com/younwookim/mg/internal/application/system"
	"github.com/younwookim/mg/internal/domain/entity"
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
	config        *config.GameConfig
	stageCfg      *config.StageConfig
	stage         *entity.Stage
	state         state.GameState
	player        *entity.Player
	physicsSystem *system.PhysicsSystem
	inputSystem   *system.InputSystem
	combatSystem  *system.CombatSystem
	screenW       int
	screenH       int
	tileSize      int
	dt            float64

	// Feedback
	hitstopFrames int
	screenShakeX  float64
	screenShakeY  float64
	shakeDecay    float64

	// Mouse aiming
	mouseWorldX float64
	mouseWorldY float64

	// Arrow selection UI
	arrowSelectUI *entity.ArrowSelectUI

	// Deterministic RNG
	rng  *rand.Rand
	seed int64

	// Input recording
	recorder       *Recorder
	recordFilename string
}

// New creates a new Playing scene.
// If recordPath is not empty, gameplay will be recorded.
func New(cfg *config.GameConfig, stageCfg *config.StageConfig, stage *entity.Stage, recordPath string) *Playing {
	// Initialize seeded RNG for deterministic randomness
	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))

	// Create player hitbox from config
	playerCfg := cfg.Entities.Player
	hitbox := entity.TrapezoidHitbox{
		Head: entity.HitboxRect{
			OffsetX: playerCfg.Hitbox.Head.OffsetX,
			OffsetY: playerCfg.Hitbox.Head.OffsetY,
			Width:   playerCfg.Hitbox.Head.Width,
			Height:  playerCfg.Hitbox.Head.Height,
		},
		Body: entity.HitboxRect{
			OffsetX: playerCfg.Hitbox.Body.OffsetX,
			OffsetY: playerCfg.Hitbox.Body.OffsetY,
			Width:   playerCfg.Hitbox.Body.Width,
			Height:  playerCfg.Hitbox.Body.Height,
		},
		Feet: entity.HitboxRect{
			OffsetX: playerCfg.Hitbox.Feet.OffsetX,
			OffsetY: playerCfg.Hitbox.Feet.OffsetY,
			Width:   playerCfg.Hitbox.Feet.Width,
			Height:  playerCfg.Hitbox.Feet.Height,
		},
	}

	player := entity.NewPlayer(stage.SpawnX, stage.SpawnY, hitbox, playerCfg.Stats.MaxHealth)

	// Create combat system with seeded RNG
	combatSystem := system.NewCombatSystem(cfg, stage, rng)

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
		player:         player,
		physicsSystem:  system.NewPhysicsSystem(cfg.Physics, stage),
		inputSystem:    system.NewInputSystem(cfg.Physics),
		combatSystem:   combatSystem,
		screenW:        cfg.Physics.Display.ScreenWidth,
		screenH:        cfg.Physics.Display.ScreenHeight,
		tileSize:       stage.TileSize,
		dt:             1.0 / float64(cfg.Physics.Display.Framerate),
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

	// Set up combat callbacks
	combatSystem.OnHitstop = func(frames int) {
		p.hitstopFrames = frames
	}
	combatSystem.OnScreenShake = func(intensity float64) {
		p.screenShakeX = intensity
		p.screenShakeY = intensity
	}

	// Spawn enemies from stage config
	for i, spawn := range stageCfg.Enemies {
		combatSystem.SpawnEnemy(entity.EntityID(i+1), spawn.X, spawn.Y, spawn.Type, spawn.FacingRight)
	}

	return p
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
	input := p.inputSystem.GetInput()

	// Record input if recording is enabled
	if p.recorder != nil {
		p.recorder.RecordFrame(input)
	}

	// Update arrow selection UI (always, for animation)
	p.arrowSelectUI.Update(input.RightClickPressed, input.RightClickReleased, input.MouseX, input.MouseY, p.screenW, p.screenH)

	// Update highlight based on mouse position
	if p.arrowSelectUI.IsActive() {
		selectedDir := p.arrowSelectUI.UpdateHighlight(input.MouseX, input.MouseY)

		// On right click release, confirm selection
		if input.RightClickReleased && selectedDir != entity.DirNone {
			p.player.CurrentArrow = p.player.EquippedArrows[int(selectedDir)]
		}
	}

	// Calculate camera offset for mouse world position (use pixel coordinates)
	camX := p.player.PixelX() - p.screenW/2 + 8
	camY := p.player.PixelY() - p.screenH/2 + 12
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

	// Convert mouse screen position to world position
	p.mouseWorldX = float64(input.MouseX + camX)
	p.mouseWorldY = float64(input.MouseY + camY)

	// Handle attack (mouse click) - only when arrow selection UI is not active
	if input.MouseClick && !p.arrowSelectUI.IsActive() {
		arrowX := float64(p.player.PixelX() + 8)
		arrowY := float64(p.player.PixelY() + 10)
		// Convert player velocity from 100x scaled to pixels/sec
		// Zero out VY when on ground (VY oscillates due to gravity/collision cycle)
		playerVX := p.player.VX / entity.PositionScale
		playerVY := p.player.VY / entity.PositionScale
		if p.player.OnGround {
			playerVY = 0
		}
		p.combatSystem.SpawnPlayerArrowToward(arrowX, arrowY, p.mouseWorldX, p.mouseWorldY, playerVX, playerVY)
	}

	// Update player with input
	p.inputSystem.UpdatePlayer(p.player, input, p.dt)

	// Update physics with sub-steps
	// Normal: 10 sub-steps = full speed
	// Slow motion: 1 sub-step = 1/10 speed
	subSteps := 10
	if p.arrowSelectUI.IsActive() {
		subSteps = 1
	}
	p.physicsSystem.Update(p.player, p.dt, subSteps)

	// Update combat
	p.combatSystem.Update(p.player, p.dt)

	// Check spike damage
	p.checkSpikeDamage()

	// Decay screen shake
	p.screenShakeX *= p.shakeDecay
	p.screenShakeY *= p.shakeDecay

	// Check game over
	if p.player.Health <= 0 {
		p.state = state.StateGameOver
		// Auto-save recording on game over
		if p.recorder != nil {
			p.saveRecording()
		}
	}
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
	if p.player.IsInvincible() {
		return
	}

	// Check feet hitbox against spikes (use pixel coordinates)
	fx, fy, fw, fh := p.player.Hitbox.Feet.GetWorldRect(p.player.PixelX(), p.player.PixelY(), p.player.FacingRight, 16)

	for py := fy; py < fy+fh; py++ {
		for px := fx; px < fx+fw; px++ {
			tile := p.stage.GetTileAtPixel(px, py)
			if tile.Type == entity.TileSpike {
				p.player.Health -= tile.Damage
				p.player.IframeTimer = p.config.Physics.Combat.Iframes
				p.player.VY = -150 * entity.PositionScale // Bounce up (100x scaled)
				p.screenShakeX = p.config.Physics.Feedback.ScreenShake.Intensity
				p.screenShakeY = p.config.Physics.Feedback.ScreenShake.Intensity
				return
			}
		}
	}
}

func (p *Playing) restart() {
	// Reset RNG with new seed
	p.seed = time.Now().UnixNano()
	p.rng = rand.New(rand.NewSource(p.seed))

	p.player.SetPixelPos(p.stage.SpawnX, p.stage.SpawnY)
	p.player.VX = 0
	p.player.VY = 0
	p.player.Health = p.player.MaxHealth
	p.player.Gold = 0
	p.player.IframeTimer = 1.0
	p.player.CurrentArrow = entity.ArrowGray
	p.state = state.StatePlaying

	// Reset UI with config
	p.arrowSelectUI = entity.NewArrowSelectUIWithConfig(entity.ArrowSelectConfig{
		Radius:      p.config.Physics.ArrowSelect.Radius,
		MinDistance: p.config.Physics.ArrowSelect.MinDistance,
		MaxFrame:    p.config.Physics.ArrowSelect.MaxFrame,
	})

	// Respawn enemies with new RNG
	p.combatSystem = system.NewCombatSystem(p.config, p.stage, p.rng)
	p.combatSystem.OnHitstop = func(frames int) {
		p.hitstopFrames = frames
	}
	p.combatSystem.OnScreenShake = func(intensity float64) {
		p.screenShakeX = intensity
		p.screenShakeY = intensity
	}
	for i, spawn := range p.stageCfg.Enemies {
		p.combatSystem.SpawnEnemy(entity.EntityID(i+1), spawn.X, spawn.Y, spawn.Type, spawn.FacingRight)
	}

	// Reset recorder if recording
	if p.recordFilename != "" {
		p.recorder = NewRecorder(p.seed, p.stageCfg.Name)
		log.Printf("Recording restarted (seed: %d)", p.seed)
	}
}

// Draw renders the game screen
func (p *Playing) Draw(screen *ebiten.Image) {
	// Fill background
	screen.Fill(colorBG)

	// Calculate camera offset (use pixel coordinates)
	camX := p.player.PixelX() - p.screenW/2 + 8
	camY := p.player.PixelY() - p.screenH/2 + 12

	// Apply screen shake
	camX += int(p.screenShakeX * (2*randFloat() - 1))
	camY += int(p.screenShakeY * (2*randFloat() - 1))

	// Clamp camera to stage bounds
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
	playerScreenX := float64(p.player.PixelX() - camX)
	playerScreenY := float64(p.player.PixelY() - camY)

	playerW := float64(p.config.Entities.Player.Sprite.FrameWidth)
	playerH := float64(p.config.Entities.Player.Sprite.FrameHeight)

	// Flash when invincible
	playerColor := colorPlayer
	if p.player.IsInvincible() && int(p.player.IframeTimer*10)%2 == 0 {
		playerColor = color.RGBA{255, 255, 255, 200}
	}

	ebitenutil.DrawRect(screen, playerScreenX, playerScreenY, playerW, playerH, playerColor)

	// Draw hitbox debug (use pixel coordinates)
	if ebiten.IsKeyPressed(ebiten.KeyTab) {
		hx, hy, hw, hh := p.player.Hitbox.Head.GetWorldRect(p.player.PixelX(), p.player.PixelY(), p.player.FacingRight, 16)
		ebitenutil.DrawRect(screen, float64(hx-camX), float64(hy-camY), float64(hw), float64(hh), colorHead)

		fx, fy, fw, fh := p.player.Hitbox.Feet.GetWorldRect(p.player.PixelX(), p.player.PixelY(), p.player.FacingRight, 16)
		ebitenutil.DrawRect(screen, float64(fx-camX), float64(fy-camY), float64(fw), float64(fh), colorFeet)
	}
}

func (p *Playing) drawEnemies(screen *ebiten.Image, camX, camY int) {
	for _, enemy := range p.combatSystem.GetEnemies() {
		if !enemy.Active {
			continue
		}

		x := float64(enemy.X - camX)
		y := float64(enemy.Y - camY)

		// Flash on hit
		c := colorEnemy
		if enemy.HitTimer > 0 {
			c = color.RGBA{255, 255, 255, 255}
		}

		ebitenutil.DrawRect(screen, x, y, float64(enemy.HitboxWidth+4), float64(enemy.HitboxHeight+4), c)
	}
}

func (p *Playing) drawProjectiles(screen *ebiten.Image, camX, camY int) {
	for _, proj := range p.combatSystem.GetProjectiles() {
		if !proj.Active {
			continue
		}

		x := proj.X - float64(camX)
		y := proj.Y - float64(camY)

		// Use current arrow color for player projectiles
		var c color.RGBA
		if proj.IsPlayer {
			c = entity.ArrowColors[p.player.CurrentArrow]
		} else {
			c = colorEnemyArrow
		}

		// Apply alpha for fading (pre-multiplied alpha)
		alpha := proj.GetAlpha()
		c = color.RGBA{
			uint8(float64(c.R) * alpha),
			uint8(float64(c.G) * alpha),
			uint8(float64(c.B) * alpha),
			uint8(float64(c.A) * alpha),
		}

		// Draw rotated arrow: p.X, p.Y is the arrow tip
		rot := proj.Rotation()
		length := 12.0
		prevX := x - math.Cos(rot)*length
		prevY := y - math.Sin(rot)*length

		ebitenutil.DrawRect(screen, x-2, y-2, 4, 4, c)
		ebitenutil.DrawLine(screen, x, y, prevX, prevY, c)
	}
}

func (p *Playing) drawGolds(screen *ebiten.Image, camX, camY int) {
	for _, gold := range p.combatSystem.GetGolds() {
		if !gold.Active {
			continue
		}

		x := gold.X - float64(camX)
		y := gold.Y - float64(camY)

		ebitenutil.DrawRect(screen, x, y, 8, 8, colorGold)
	}
}

func (p *Playing) drawUI(screen *ebiten.Image) {
	// Health bar
	barX := 10.0
	barY := float64(p.screenH - 20)
	barW := 100.0
	barH := 10.0

	// Background
	ebitenutil.DrawRect(screen, barX, barY, barW, barH, colorHealthBG)

	// Foreground
	healthRatio := float64(p.player.Health) / float64(p.player.MaxHealth)
	if healthRatio < 0 {
		healthRatio = 0
	}
	ebitenutil.DrawRect(screen, barX, barY, barW*healthRatio, barH, colorHealthFG)

	// Current arrow indicator (right side of HP bar)
	p.drawArrowIcon(screen, barX+barW+10, barY+barH/2, p.player.CurrentArrow, 1.0, true)

	// Gold
	goldText := fmt.Sprintf("Gold: %d", p.player.Gold)
	ebitenutil.DebugPrintAt(screen, goldText, 10, p.screenH-35)

	// Controls
	debugText := "A/D: Move | W: Jump | Space: Dash | LClick: Attack | RClick: Arrow Select | ESC: Pause"
	ebitenutil.DebugPrint(screen, debugText)
}

func (p *Playing) drawPauseOverlay(screen *ebiten.Image) {
	// Semi-transparent overlay
	overlay := color.RGBA{0, 0, 0, 128}
	ebitenutil.DrawRect(screen, 0, 0, float64(p.screenW), float64(p.screenH), overlay)

	text := "PAUSED\n\nPress ESC to resume"
	ebitenutil.DebugPrintAt(screen, text, p.screenW/2-50, p.screenH/2-20)
}

func (p *Playing) drawGameOverOverlay(screen *ebiten.Image) {
	overlay := color.RGBA{100, 0, 0, 180}
	ebitenutil.DrawRect(screen, 0, 0, float64(p.screenW), float64(p.screenH), overlay)

	text := fmt.Sprintf("GAME OVER\n\nGold collected: %d\n\nPress Z to restart", p.player.Gold)
	ebitenutil.DebugPrintAt(screen, text, p.screenW/2-60, p.screenH/2-30)
}

// drawArrowSelectOverlay draws the dark overlay for arrow selection
func (p *Playing) drawArrowSelectOverlay(screen *ebiten.Image) {
	progress := p.arrowSelectUI.GetProgress()
	easedProgress := math.Sin(progress * math.Pi / 2) // sin easing

	// Darken based on progress (max 50% opacity)
	alpha := uint8(128 * easedProgress)
	overlay := color.RGBA{0, 0, 0, alpha}
	ebitenutil.DrawRect(screen, 0, 0, float64(p.screenW), float64(p.screenH), overlay)
}

// drawArrowSelectUI draws the radial arrow selection menu
func (p *Playing) drawArrowSelectUI(screen *ebiten.Image) {
	progress := p.arrowSelectUI.GetProgress()
	easedProgress := math.Sin(progress * math.Pi / 2) // sin easing

	// Draw each arrow icon in the 4 directions
	for dir := entity.DirRight; dir <= entity.DirDown; dir++ {
		arrowType := p.player.EquippedArrows[int(dir)]
		x, y := p.arrowSelectUI.GetIconPosition(dir, easedProgress)

		// Determine brightness
		// Selected (current) arrow = 100%, unselected = 70%, highlighted = 100%
		brightness := 0.7
		if arrowType == p.player.CurrentArrow {
			brightness = 1.0
		}
		if dir == p.arrowSelectUI.Highlighted {
			brightness = 1.0
		}

		// Determine scale (highlighted = larger)
		scale := 1.0
		if dir == p.arrowSelectUI.Highlighted {
			scale = 1.3
		}

		p.drawArrowIcon(screen, x, y, arrowType, brightness*easedProgress, scale > 1.0)
	}
}

// drawArrowIcon draws an arrow icon (line with tip) at the given position
func (p *Playing) drawArrowIcon(screen *ebiten.Image, x, y float64, arrowType entity.ArrowType, brightness float64, large bool) {
	baseColor := entity.ArrowColors[arrowType]

	// Apply brightness (and alpha based on easedProgress for animation)
	c := color.RGBA{
		uint8(float64(baseColor.R) * brightness),
		uint8(float64(baseColor.G) * brightness),
		uint8(float64(baseColor.B) * brightness),
		uint8(float64(baseColor.A) * brightness),
	}

	// Arrow dimensions
	length := 12.0
	if large {
		length = 16.0
	}

	// Arrow pointing right (0 degrees)
	tipX := x + length/2
	tipY := y
	tailX := x - length/2
	tailY := y

	// Draw arrow line
	ebitenutil.DrawLine(screen, tailX, tailY, tipX, tipY, c)

	// Draw arrow tip (small triangle approximated with lines)
	tipSize := 4.0
	if large {
		tipSize = 5.0
	}
	ebitenutil.DrawLine(screen, tipX, tipY, tipX-tipSize, tipY-tipSize/2, c)
	ebitenutil.DrawLine(screen, tipX, tipY, tipX-tipSize, tipY+tipSize/2, c)

	// Draw small square at tip for visibility
	ebitenutil.DrawRect(screen, tipX-1, tipY-1, 2, 2, c)
}

func (p *Playing) drawTrajectory(screen *ebiten.Image, camX, camY int) {
	// Get arrow physics config
	speed, gravity, maxFall, maxRange, velocityInfluence := p.combatSystem.GetArrowConfig()

	// Arrow start position (use pixel coordinates)
	startX := float64(p.player.PixelX() + 8)
	startY := float64(p.player.PixelY() + 10)

	// Calculate initial velocity toward mouse
	dx := p.mouseWorldX - startX
	dy := p.mouseWorldY - startY
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1 {
		dist = 1
	}
	vx := (dx / dist) * speed
	vy := (dy / dist) * speed

	// Add player velocity with influence multiplier
	// Note: When on ground, VY oscillates due to gravity/collision cycle,
	// so we zero it out for stable trajectory prediction
	playerVX := p.player.VX / entity.PositionScale
	playerVY := p.player.VY / entity.PositionScale
	if p.player.OnGround {
		playerVY = 0
	}
	vx += playerVX * velocityInfluence
	vy += playerVY * velocityInfluence

	// Trajectory color - white with slight tint of current arrow color
	arrowColor := entity.ArrowColors[p.player.CurrentArrow]
	trajectoryColor := color.RGBA{
		uint8((int(arrowColor.R) + 255) / 2),
		uint8((int(arrowColor.G) + 255) / 2),
		uint8((int(arrowColor.B) + 255) / 2),
		200,
	}

	// Simulate trajectory
	x, y := startX, startY
	dt := 1.0 / 60.0
	dotSpacing := 8.0
	accumulated := 0.0
	dotSize := 3.0

	for traveled := 0.0; traveled < maxRange; {
		// Apply gravity
		vy += gravity * dt
		if vy > maxFall {
			vy = maxFall
		}

		// Previous position
		prevX, prevY := x, y

		// Move
		x += vx * dt
		y += vy * dt

		// Calculate distance moved this step
		stepDx := x - prevX
		stepDy := y - prevY
		stepDist := math.Sqrt(stepDx*stepDx + stepDy*stepDy)
		traveled += stepDist
		accumulated += stepDist

		// Check wall collision
		if p.stage.IsSolidAt(int(x), int(y)) {
			break
		}

		// Draw dot at intervals
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

// Layout returns the game's screen dimensions (used by game.Game)
func (p *Playing) Layout(outsideWidth, outsideHeight int) (int, int) {
	return p.screenW, p.screenH
}

var randState uint32 = 1

func randFloat() float64 {
	randState = randState*1103515245 + 12345
	return float64(randState&0x7fffffff) / float64(0x7fffffff)
}
