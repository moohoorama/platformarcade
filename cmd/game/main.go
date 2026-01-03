package main

import (
	"flag"
	"fmt"
	"image/color"
	"io/fs"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	colorArrow      = color.RGBA{255, 200, 100, 255}
	colorEnemyArrow = color.RGBA{255, 100, 100, 255}
	colorGold       = color.RGBA{255, 215, 0, 255}
	colorHealthBG   = color.RGBA{60, 60, 60, 255}
	colorHealthFG   = color.RGBA{100, 200, 100, 255}
	colorTrajectory = color.RGBA{255, 255, 255, 200}
)

// Game implements ebiten.Game interface
type Game struct {
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

// NewGame creates a new game instance
func NewGame(cfg *config.GameConfig, stageCfg *config.StageConfig, stage *entity.Stage, recordFilename string) *Game {
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

	game := &Game{
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
		recordFilename: recordFilename,
	}

	// Initialize recorder if recording is enabled
	if recordFilename != "" {
		game.recorder = NewRecorder(seed, stageCfg.Name)
		log.Printf("Recording enabled: %s (seed: %d)", recordFilename, seed)
	}

	// Set up combat callbacks
	combatSystem.OnHitstop = func(frames int) {
		game.hitstopFrames = frames
	}
	combatSystem.OnScreenShake = func(intensity float64) {
		game.screenShakeX = intensity
		game.screenShakeY = intensity
	}

	// Spawn enemies from stage config
	for i, spawn := range stageCfg.Enemies {
		combatSystem.SpawnEnemy(entity.EntityID(i+1), spawn.X, spawn.Y, spawn.Type, spawn.FacingRight)
	}

	return game
}

// Update proceeds the game state
func (g *Game) Update() error {
	// Handle hitstop
	if g.hitstopFrames > 0 {
		g.hitstopFrames--
		return nil
	}

	switch g.state {
	case state.StatePlaying:
		g.updatePlaying()
	case state.StatePaused:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.state = state.StatePlaying
		}
	case state.StateGameOver:
		if inpututil.IsKeyJustPressed(ebiten.KeyZ) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.restart()
		}
	}

	return nil
}

func (g *Game) updatePlaying() {
	// Check for pause
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = state.StatePaused
		return
	}

	// F5: Save recording manually
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) && g.recorder != nil {
		g.saveRecording()
	}

	// Get input
	input := g.inputSystem.GetInput()

	// Record input if recording is enabled
	if g.recorder != nil {
		g.recorder.RecordFrame(input)
	}

	// Update arrow selection UI (always, for animation)
	g.arrowSelectUI.Update(input.RightClickPressed, input.RightClickReleased, input.MouseX, input.MouseY, g.screenW, g.screenH)

	// Update highlight based on mouse position
	if g.arrowSelectUI.IsActive() {
		selectedDir := g.arrowSelectUI.UpdateHighlight(input.MouseX, input.MouseY)

		// On right click release, confirm selection
		if input.RightClickReleased && selectedDir != entity.DirNone {
			g.player.CurrentArrow = g.player.EquippedArrows[int(selectedDir)]
		}
	}

	// Calculate camera offset for mouse world position (use pixel coordinates)
	camX := g.player.PixelX() - g.screenW/2 + 8
	camY := g.player.PixelY() - g.screenH/2 + 12
	if camX < 0 {
		camX = 0
	}
	if camY < 0 {
		camY = 0
	}
	maxCamX := g.stage.Width*g.tileSize - g.screenW
	maxCamY := g.stage.Height*g.tileSize - g.screenH
	if camX > maxCamX {
		camX = maxCamX
	}
	if camY > maxCamY {
		camY = maxCamY
	}

	// Convert mouse screen position to world position
	g.mouseWorldX = float64(input.MouseX + camX)
	g.mouseWorldY = float64(input.MouseY + camY)

	// Handle attack (mouse click) - only when arrow selection UI is not active
	if input.MouseClick && !g.arrowSelectUI.IsActive() {
		arrowX := float64(g.player.PixelX() + 8)
		arrowY := float64(g.player.PixelY() + 10)
		// Convert player velocity from 100x scaled to pixels/sec
		// Zero out VY when on ground (VY oscillates due to gravity/collision cycle)
		playerVX := g.player.VX / entity.PositionScale
		playerVY := g.player.VY / entity.PositionScale
		if g.player.OnGround {
			playerVY = 0
		}
		g.combatSystem.SpawnPlayerArrowToward(arrowX, arrowY, g.mouseWorldX, g.mouseWorldY, playerVX, playerVY)
	}

	// Update player with input
	g.inputSystem.UpdatePlayer(g.player, input, g.dt)

	// Update physics with sub-steps
	// Normal: 10 sub-steps = full speed
	// Slow motion: 1 sub-step = 1/10 speed
	subSteps := 10
	if g.arrowSelectUI.IsActive() {
		subSteps = 1
	}
	g.physicsSystem.Update(g.player, g.dt, subSteps)

	// Update combat
	g.combatSystem.Update(g.player, g.dt)

	// Check spike damage
	g.checkSpikeDamage()

	// Decay screen shake
	g.screenShakeX *= g.shakeDecay
	g.screenShakeY *= g.shakeDecay

	// Check game over
	if g.player.Health <= 0 {
		g.state = state.StateGameOver
		// Auto-save recording on game over
		if g.recorder != nil {
			g.saveRecording()
		}
	}
}

// saveRecording saves the current recording to file
func (g *Game) saveRecording() {
	if g.recorder == nil {
		return
	}

	filename := g.recordFilename
	if filename == "" {
		filename = GenerateFilename()
	}

	if err := g.recorder.Save(filename); err != nil {
		log.Printf("Failed to save recording: %v", err)
	} else {
		log.Printf("Recording saved: %s (%d frames)", filename, g.recorder.FrameCount())
	}
}

func (g *Game) checkSpikeDamage() {
	if g.player.IsInvincible() {
		return
	}

	// Check feet hitbox against spikes (use pixel coordinates)
	fx, fy, fw, fh := g.player.Hitbox.Feet.GetWorldRect(g.player.PixelX(), g.player.PixelY(), g.player.FacingRight, 16)

	for py := fy; py < fy+fh; py++ {
		for px := fx; px < fx+fw; px++ {
			tile := g.stage.GetTileAtPixel(px, py)
			if tile.Type == entity.TileSpike {
				g.player.Health -= tile.Damage
				g.player.IframeTimer = g.config.Physics.Combat.Iframes
				g.player.VY = -150 * entity.PositionScale // Bounce up (100x scaled)
				g.screenShakeX = g.config.Physics.Feedback.ScreenShake.Intensity
				g.screenShakeY = g.config.Physics.Feedback.ScreenShake.Intensity
				return
			}
		}
	}
}

func (g *Game) restart() {
	// Reset RNG with new seed
	g.seed = time.Now().UnixNano()
	g.rng = rand.New(rand.NewSource(g.seed))

	g.player.SetPixelPos(g.stage.SpawnX, g.stage.SpawnY)
	g.player.VX = 0
	g.player.VY = 0
	g.player.Health = g.player.MaxHealth
	g.player.Gold = 0
	g.player.IframeTimer = 1.0
	g.player.CurrentArrow = entity.ArrowGray
	g.state = state.StatePlaying

	// Reset UI with config
	g.arrowSelectUI = entity.NewArrowSelectUIWithConfig(entity.ArrowSelectConfig{
		Radius:      g.config.Physics.ArrowSelect.Radius,
		MinDistance: g.config.Physics.ArrowSelect.MinDistance,
		MaxFrame:    g.config.Physics.ArrowSelect.MaxFrame,
	})

	// Respawn enemies with new RNG
	g.combatSystem = system.NewCombatSystem(g.config, g.stage, g.rng)
	g.combatSystem.OnHitstop = func(frames int) {
		g.hitstopFrames = frames
	}
	g.combatSystem.OnScreenShake = func(intensity float64) {
		g.screenShakeX = intensity
		g.screenShakeY = intensity
	}
	for i, spawn := range g.stageCfg.Enemies {
		g.combatSystem.SpawnEnemy(entity.EntityID(i+1), spawn.X, spawn.Y, spawn.Type, spawn.FacingRight)
	}

	// Reset recorder if recording
	if g.recordFilename != "" {
		g.recorder = NewRecorder(g.seed, g.stageCfg.Name)
		log.Printf("Recording restarted (seed: %d)", g.seed)
	}
}

// Draw renders the game screen
func (g *Game) Draw(screen *ebiten.Image) {
	// Fill background
	screen.Fill(colorBG)

	// Calculate camera offset (use pixel coordinates)
	camX := g.player.PixelX() - g.screenW/2 + 8
	camY := g.player.PixelY() - g.screenH/2 + 12

	// Apply screen shake
	camX += int(g.screenShakeX * (2*randFloat() - 1))
	camY += int(g.screenShakeY * (2*randFloat() - 1))

	// Clamp camera to stage bounds
	maxCamX := g.stage.Width*g.tileSize - g.screenW
	maxCamY := g.stage.Height*g.tileSize - g.screenH
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
	g.drawTiles(screen, camX, camY)
	g.drawGolds(screen, camX, camY)
	g.drawEnemies(screen, camX, camY)
	g.drawProjectiles(screen, camX, camY)
	g.drawPlayer(screen, camX, camY)
	g.drawTrajectory(screen, camX, camY)

	// Draw dark overlay when arrow selection UI is active
	if g.arrowSelectUI.IsActive() {
		g.drawArrowSelectOverlay(screen)
	}

	// Draw arrow selection UI
	if g.arrowSelectUI.IsActive() {
		g.drawArrowSelectUI(screen)
	}

	// Draw UI (HP bar, current arrow, etc.) - always on top
	g.drawUI(screen)

	// Draw state overlays
	switch g.state {
	case state.StatePaused:
		g.drawPauseOverlay(screen)
	case state.StateGameOver:
		g.drawGameOverOverlay(screen)
	}
}

func (g *Game) drawTiles(screen *ebiten.Image, camX, camY int) {
	startTileX := camX / g.tileSize
	startTileY := camY / g.tileSize
	endTileX := (camX + g.screenW) / g.tileSize + 1
	endTileY := (camY + g.screenH) / g.tileSize + 1

	for ty := startTileY; ty <= endTileY && ty < g.stage.Height; ty++ {
		for tx := startTileX; tx <= endTileX && tx < g.stage.Width; tx++ {
			if tx < 0 || ty < 0 {
				continue
			}
			tile := g.stage.GetTile(tx, ty)
			if tile.Type == entity.TileEmpty {
				continue
			}

			x := float64(tx*g.tileSize - camX)
			y := float64(ty*g.tileSize - camY)

			var c color.Color
			switch tile.Type {
			case entity.TileWall:
				c = colorWall
			case entity.TileSpike:
				c = colorSpike
			}

			ebitenutil.DrawRect(screen, x, y, float64(g.tileSize), float64(g.tileSize), c)
		}
	}
}

func (g *Game) drawPlayer(screen *ebiten.Image, camX, camY int) {
	playerScreenX := float64(g.player.PixelX() - camX)
	playerScreenY := float64(g.player.PixelY() - camY)

	playerW := float64(g.config.Entities.Player.Sprite.FrameWidth)
	playerH := float64(g.config.Entities.Player.Sprite.FrameHeight)

	// Flash when invincible
	playerColor := colorPlayer
	if g.player.IsInvincible() && int(g.player.IframeTimer*10)%2 == 0 {
		playerColor = color.RGBA{255, 255, 255, 200}
	}

	ebitenutil.DrawRect(screen, playerScreenX, playerScreenY, playerW, playerH, playerColor)

	// Draw hitbox debug (use pixel coordinates)
	if ebiten.IsKeyPressed(ebiten.KeyTab) {
		hx, hy, hw, hh := g.player.Hitbox.Head.GetWorldRect(g.player.PixelX(), g.player.PixelY(), g.player.FacingRight, 16)
		ebitenutil.DrawRect(screen, float64(hx-camX), float64(hy-camY), float64(hw), float64(hh), colorHead)

		fx, fy, fw, fh := g.player.Hitbox.Feet.GetWorldRect(g.player.PixelX(), g.player.PixelY(), g.player.FacingRight, 16)
		ebitenutil.DrawRect(screen, float64(fx-camX), float64(fy-camY), float64(fw), float64(fh), colorFeet)
	}
}

func (g *Game) drawEnemies(screen *ebiten.Image, camX, camY int) {
	for _, enemy := range g.combatSystem.GetEnemies() {
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

func (g *Game) drawProjectiles(screen *ebiten.Image, camX, camY int) {
	for _, proj := range g.combatSystem.GetProjectiles() {
		if !proj.Active {
			continue
		}

		x := proj.X - float64(camX)
		y := proj.Y - float64(camY)

		// Use current arrow color for player projectiles
		var c color.RGBA
		if proj.IsPlayer {
			c = entity.ArrowColors[g.player.CurrentArrow]
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

func (g *Game) drawGolds(screen *ebiten.Image, camX, camY int) {
	for _, gold := range g.combatSystem.GetGolds() {
		if !gold.Active {
			continue
		}

		x := gold.X - float64(camX)
		y := gold.Y - float64(camY)

		ebitenutil.DrawRect(screen, x, y, 8, 8, colorGold)
	}
}

func (g *Game) drawUI(screen *ebiten.Image) {
	// Health bar
	barX := 10.0
	barY := float64(g.screenH - 20)
	barW := 100.0
	barH := 10.0

	// Background
	ebitenutil.DrawRect(screen, barX, barY, barW, barH, colorHealthBG)

	// Foreground
	healthRatio := float64(g.player.Health) / float64(g.player.MaxHealth)
	if healthRatio < 0 {
		healthRatio = 0
	}
	ebitenutil.DrawRect(screen, barX, barY, barW*healthRatio, barH, colorHealthFG)

	// Current arrow indicator (right side of HP bar)
	g.drawArrowIcon(screen, barX+barW+10, barY+barH/2, g.player.CurrentArrow, 1.0, true)

	// Gold
	goldText := fmt.Sprintf("Gold: %d", g.player.Gold)
	ebitenutil.DebugPrintAt(screen, goldText, 10, g.screenH-35)

	// Controls
	debugText := "A/D: Move | W: Jump | Space: Dash | LClick: Attack | RClick: Arrow Select | ESC: Pause"
	ebitenutil.DebugPrint(screen, debugText)
}

func (g *Game) drawPauseOverlay(screen *ebiten.Image) {
	// Semi-transparent overlay
	overlay := color.RGBA{0, 0, 0, 128}
	ebitenutil.DrawRect(screen, 0, 0, float64(g.screenW), float64(g.screenH), overlay)

	text := "PAUSED\n\nPress ESC to resume"
	ebitenutil.DebugPrintAt(screen, text, g.screenW/2-50, g.screenH/2-20)
}

func (g *Game) drawGameOverOverlay(screen *ebiten.Image) {
	overlay := color.RGBA{100, 0, 0, 180}
	ebitenutil.DrawRect(screen, 0, 0, float64(g.screenW), float64(g.screenH), overlay)

	text := fmt.Sprintf("GAME OVER\n\nGold collected: %d\n\nPress Z to restart", g.player.Gold)
	ebitenutil.DebugPrintAt(screen, text, g.screenW/2-60, g.screenH/2-30)
}

// drawArrowSelectOverlay draws the dark overlay for arrow selection
func (g *Game) drawArrowSelectOverlay(screen *ebiten.Image) {
	progress := g.arrowSelectUI.GetProgress()
	easedProgress := math.Sin(progress * math.Pi / 2) // sin easing

	// Darken based on progress (max 50% opacity)
	alpha := uint8(128 * easedProgress)
	overlay := color.RGBA{0, 0, 0, alpha}
	ebitenutil.DrawRect(screen, 0, 0, float64(g.screenW), float64(g.screenH), overlay)
}

// drawArrowSelectUI draws the radial arrow selection menu
func (g *Game) drawArrowSelectUI(screen *ebiten.Image) {
	progress := g.arrowSelectUI.GetProgress()
	easedProgress := math.Sin(progress * math.Pi / 2) // sin easing

	// Draw each arrow icon in the 4 directions
	for dir := entity.DirRight; dir <= entity.DirDown; dir++ {
		arrowType := g.player.EquippedArrows[int(dir)]
		x, y := g.arrowSelectUI.GetIconPosition(dir, easedProgress)

		// Determine brightness
		// Selected (current) arrow = 100%, unselected = 70%, highlighted = 100%
		brightness := 0.7
		if arrowType == g.player.CurrentArrow {
			brightness = 1.0
		}
		if dir == g.arrowSelectUI.Highlighted {
			brightness = 1.0
		}

		// Determine scale (highlighted = larger)
		scale := 1.0
		if dir == g.arrowSelectUI.Highlighted {
			scale = 1.3
		}

		g.drawArrowIcon(screen, x, y, arrowType, brightness*easedProgress, scale > 1.0)
	}
}

// drawArrowIcon draws an arrow icon (line with tip) at the given position
func (g *Game) drawArrowIcon(screen *ebiten.Image, x, y float64, arrowType entity.ArrowType, brightness float64, large bool) {
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

// Layout returns the game's screen dimensions
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenW, g.screenH
}

func (g *Game) drawTrajectory(screen *ebiten.Image, camX, camY int) {
	// Get arrow physics config
	speed, gravity, maxFall, maxRange, velocityInfluence := g.combatSystem.GetArrowConfig()

	// Arrow start position (use pixel coordinates)
	startX := float64(g.player.PixelX() + 8)
	startY := float64(g.player.PixelY() + 10)

	// Calculate initial velocity toward mouse
	dx := g.mouseWorldX - startX
	dy := g.mouseWorldY - startY
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1 {
		dist = 1
	}
	vx := (dx / dist) * speed
	vy := (dy / dist) * speed

	// Add player velocity with influence multiplier
	// Note: When on ground, VY oscillates due to gravity/collision cycle,
	// so we zero it out for stable trajectory prediction
	playerVX := g.player.VX / entity.PositionScale
	playerVY := g.player.VY / entity.PositionScale
	if g.player.OnGround {
		playerVY = 0
	}
	vx += playerVX * velocityInfluence
	vy += playerVY * velocityInfluence

	// Trajectory color - white with slight tint of current arrow color
	arrowColor := entity.ArrowColors[g.player.CurrentArrow]
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
		if g.stage.IsSolidAt(int(x), int(y)) {
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

var randState uint32 = 1

func randFloat() float64 {
	randState = randState*1103515245 + 12345
	return float64(randState&0x7fffffff) / float64(0x7fffffff)
}

func main() {
	// Parse command line flags
	recordFlag := flag.String("record", "", "Record input to file (e.g., -record replay.json)")
	flag.Parse()

	recordFilename := *recordFlag

	// Load configurations using embedded filesystem
	fsys, err := fs.Sub(configFS, "configs")
	if err != nil {
		log.Fatalf("Failed to get config subfs: %v", err)
	}
	loader := config.NewFSLoader(fsys, "configs")
	cfg, err := loader.LoadAll()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Load stage
	stageCfg, err := loader.LoadStage("demo")
	if err != nil {
		log.Fatalf("Failed to load stage: %v", err)
	}
	stage := system.LoadStage(stageCfg)

	// Create game
	game := NewGame(cfg, stageCfg, stage, recordFilename)

	// Set up ebiten
	ebiten.SetWindowSize(cfg.Physics.Display.ScreenWidth*cfg.Physics.Display.Scale,
		cfg.Physics.Display.ScreenHeight*cfg.Physics.Display.Scale)
	ebiten.SetWindowTitle("Platform Action Game")
	ebiten.SetTPS(cfg.Physics.Display.Framerate)

	// Run game
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
