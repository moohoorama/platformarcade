package system

import "github.com/younwookim/mg/internal/domain/entity"

// Intent represents an action that an entity wants to perform
type Intent interface {
	isIntent()
}

// MoveIntent represents a movement intention
type MoveIntent struct {
	EntityID entity.EntityID
	DX, DY   int // Pixels to move
}

func (MoveIntent) isIntent() {}

// JumpIntent represents a jump intention
type JumpIntent struct {
	EntityID entity.EntityID
	Force    float64
}

func (JumpIntent) isIntent() {}

// DashIntent represents a dash intention
type DashIntent struct {
	EntityID  entity.EntityID
	Direction int // -1 for left, 1 for right
}

func (DashIntent) isIntent() {}

// AttackIntent represents an attack intention
type AttackIntent struct {
	EntityID    entity.EntityID
	DirectionX  int // -1 for left, 1 for right
	DirectionY  int // -1 for up, 1 for down, 0 for horizontal
}

func (AttackIntent) isIntent() {}
