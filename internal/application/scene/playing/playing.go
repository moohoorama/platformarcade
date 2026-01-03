// Package playing provides the main gameplay scene.
package playing

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/younwookim/mg/internal/application/scene"
	"github.com/younwookim/mg/internal/application/system"
	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

// Playing is the main gameplay scene that manages player, enemies, and game state.
type Playing struct {
	cfg         *config.GameConfig
	stageCfg    *config.StageConfig
	stage       *entity.Stage
	player      *entity.Player
	recorder    *Recorder
	inputSystem *system.InputSystem
}

// New creates a new Playing scene.
// If recordPath is not empty, gameplay will be recorded.
func New(cfg *config.GameConfig, stageCfg *config.StageConfig, stage *entity.Stage, recordPath string) *Playing {
	p := &Playing{
		cfg:         cfg,
		stageCfg:    stageCfg,
		stage:       stage,
		inputSystem: system.NewInputSystem(cfg.Physics),
	}

	// Convert config hitbox to entity hitbox
	hitbox := configToHitbox(cfg.Entities.Player.Hitbox)

	// Create player
	p.player = entity.NewPlayer(
		stageCfg.PlayerSpawn.X,
		stageCfg.PlayerSpawn.Y,
		hitbox,
		cfg.Entities.Player.Stats.MaxHealth,
	)

	// Setup recorder if path provided
	if recordPath != "" {
		p.recorder = NewRecorder(stageCfg.Name, recordPath)
	}

	return p
}

// configToHitbox converts config.HitboxConfig to entity.TrapezoidHitbox
func configToHitbox(hc config.HitboxConfig) entity.TrapezoidHitbox {
	return entity.TrapezoidHitbox{
		Head: entity.HitboxRect{
			OffsetX: hc.Head.OffsetX,
			OffsetY: hc.Head.OffsetY,
			Width:   hc.Head.Width,
			Height:  hc.Head.Height,
		},
		Body: entity.HitboxRect{
			OffsetX: hc.Body.OffsetX,
			OffsetY: hc.Body.OffsetY,
			Width:   hc.Body.Width,
			Height:  hc.Body.Height,
		},
		Feet: entity.HitboxRect{
			OffsetX: hc.Feet.OffsetX,
			OffsetY: hc.Feet.OffsetY,
			Width:   hc.Feet.Width,
			Height:  hc.Feet.Height,
		},
	}
}

// Update updates the game state.
// Returns the next scene if transitioning, nil to stay.
func (p *Playing) Update(_ float64) (scene.Scene, error) {
	// Get current input state
	input := p.inputSystem.GetInput()

	// Record frame if recording
	if p.recorder != nil {
		p.recorder.RecordFrame(input)
	}

	// TODO: Implement game logic

	return nil, nil
}

// Draw renders the game.
func (p *Playing) Draw(screen *ebiten.Image) {
	// TODO: Implement rendering
}

// OnEnter is called when entering this scene.
func (p *Playing) OnEnter() {
	// TODO: Initialize scene state
}

// OnExit is called when leaving this scene.
func (p *Playing) OnExit() {
	// Save recording if recording
	if p.recorder != nil {
		_ = p.recorder.Save()
	}
}
