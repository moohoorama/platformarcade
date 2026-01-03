// Package game provides the main game loop manager that handles Scene transitions.
package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/younwookim/mg/internal/application/scene"
)

// Game implements ebiten.Game and manages Scene transitions.
type Game struct {
	current scene.Scene
	screenW int
	screenH int
	dt      float64
}

// New creates a new Game with the given initial scene.
// The initial scene's OnEnter is called immediately.
func New(initialScene scene.Scene, screenW, screenH int) *Game {
	g := &Game{
		current: initialScene,
		screenW: screenW,
		screenH: screenH,
		dt:      1.0 / 60.0, // Default to 60 FPS
	}
	g.current.OnEnter()
	return g
}

// Update updates the current scene and handles scene transitions.
// Implements ebiten.Game interface.
func (g *Game) Update() error {
	next, err := g.current.Update(g.dt)
	if err != nil {
		return err
	}

	// Handle scene transition
	if next != nil {
		g.current.OnExit()
		g.current = next
		g.current.OnEnter()
	}

	return nil
}

// Draw renders the current scene.
// Implements ebiten.Game interface.
func (g *Game) Draw(screen *ebiten.Image) {
	g.current.Draw(screen)
}

// Layout returns the game's logical screen dimensions.
// Implements ebiten.Game interface.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenW, g.screenH
}

// SetDT sets the delta time used for updates.
// Useful for testing or custom frame rates.
func (g *Game) SetDT(dt float64) {
	g.dt = dt
}
