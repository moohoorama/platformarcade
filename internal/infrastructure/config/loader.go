package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
)

// GameConfig holds all loaded configurations
type GameConfig struct {
	Physics  *PhysicsConfig
	Entities *EntitiesConfig
}

// Loader loads game configuration from JSON files using fs.FS interface
type Loader struct {
	fsys     fs.FS
	basePath string
}

// NewLoader creates a new config loader from filesystem path
func NewLoader(basePath string) *Loader {
	return &Loader{
		fsys:     os.DirFS(basePath),
		basePath: basePath,
	}
}

// NewFSLoader creates a new config loader from fs.FS
func NewFSLoader(fsys fs.FS, basePath string) *Loader {
	return &Loader{
		fsys:     fsys,
		basePath: basePath,
	}
}

// LoadPhysics loads physics.json
func (l *Loader) LoadPhysics() (*PhysicsConfig, error) {
	data, err := fs.ReadFile(l.fsys, "physics.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read physics.json: %w", err)
	}

	var cfg PhysicsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse physics.json: %w", err)
	}

	return &cfg, nil
}

// LoadEntities loads entities.json
func (l *Loader) LoadEntities() (*EntitiesConfig, error) {
	data, err := fs.ReadFile(l.fsys, "entities.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read entities.json: %w", err)
	}

	var cfg EntitiesConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse entities.json: %w", err)
	}

	return &cfg, nil
}

// LoadStage loads a stage JSON file
func (l *Loader) LoadStage(name string) (*StageConfig, error) {
	path := "stages/" + name + ".json"
	data, err := fs.ReadFile(l.fsys, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read stage %s: %w", name, err)
	}

	var cfg StageConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse stage %s: %w", name, err)
	}

	return &cfg, nil
}

// LoadAll loads all base configurations (physics, entities)
func (l *Loader) LoadAll() (*GameConfig, error) {
	physics, err := l.LoadPhysics()
	if err != nil {
		return nil, err
	}

	entities, err := l.LoadEntities()
	if err != nil {
		return nil, err
	}

	return &GameConfig{
		Physics:  physics,
		Entities: entities,
	}, nil
}
