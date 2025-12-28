package state

// GameState represents the current state of the game
type GameState int

const (
	StateMenu GameState = iota
	StateLoading
	StatePlaying
	StatePaused
	StateGameOver
	StateStageClear
)

// String returns the string representation of the game state
func (s GameState) String() string {
	switch s {
	case StateMenu:
		return "Menu"
	case StateLoading:
		return "Loading"
	case StatePlaying:
		return "Playing"
	case StatePaused:
		return "Paused"
	case StateGameOver:
		return "GameOver"
	case StateStageClear:
		return "StageClear"
	default:
		return "Unknown"
	}
}
