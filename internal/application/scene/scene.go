// Package scene defines the Scene interface for game screens.
//
// Each game screen (title, menu, playing, settings, etc.) implements
// the Scene interface to handle its own update logic and rendering.
package scene

import "github.com/hajimehoshi/ebiten/v2"

// Scene represents a game screen (title, menu, playing, settings, etc.)
//
// The game loop delegates Update and Draw calls to the current scene.
// Scene transitions are handled by returning a new Scene from Update.
type Scene interface {
	// Update updates the scene state.
	// dt is the delta time in seconds (typically 1/60).
	// Returns the next scene if a transition is needed, nil to stay on current scene.
	// Returns an error to terminate the game.
	Update(dt float64) (next Scene, err error)

	// Draw renders the scene to the screen.
	Draw(screen *ebiten.Image)

	// OnEnter is called when entering this scene.
	// Use this for initialization that should happen each time the scene is entered.
	OnEnter()

	// OnExit is called when leaving this scene.
	// Use this for cleanup, saving state, or resource release.
	OnExit()
}
