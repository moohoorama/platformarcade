package entity

import "github.com/younwookim/mg/internal/infrastructure/config"

// EntityID is a unique identifier for an entity
type EntityID uint32

// TileType represents the type of a tile
type TileType int

const (
	TileEmpty TileType = iota
	TileWall
	TileSpike
)

// Tile represents a single tile in the stage
type Tile struct {
	Type   TileType
	Solid  bool
	Damage int
}

// Stage represents the current stage's tile data
type Stage struct {
	Width    int
	Height   int
	TileSize int
	Tiles    [][]Tile
	SpawnX   int
	SpawnY   int
}

// GetTile returns the tile at the given tile coordinates
func (s *Stage) GetTile(tx, ty int) Tile {
	if tx < 0 || tx >= s.Width || ty < 0 || ty >= s.Height {
		return Tile{Type: TileWall, Solid: true}
	}
	return s.Tiles[ty][tx]
}

// GetTileAtPixel returns the tile at the given pixel coordinates
func (s *Stage) GetTileAtPixel(px, py int) Tile {
	tx := px / s.TileSize
	ty := py / s.TileSize
	return s.GetTile(tx, ty)
}

// IsSolidAt checks if the tile at pixel coordinates is solid
func (s *Stage) IsSolidAt(px, py int) bool {
	return s.GetTileAtPixel(px, py).Solid
}

// GetTileType returns the tile type at pixel coordinates
func (s *Stage) GetTileType(px, py int) int {
	return int(s.GetTileAtPixel(px, py).Type)
}

// GetTileDamage returns the tile damage at pixel coordinates
func (s *Stage) GetTileDamage(px, py int) int {
	return s.GetTileAtPixel(px, py).Damage
}

// GetWidth returns the stage width in tiles
func (s *Stage) GetWidth() int {
	return s.Width
}

// GetHeight returns the stage height in tiles
func (s *Stage) GetHeight() int {
	return s.Height
}

// GetTileSize returns the tile size in pixels
func (s *Stage) GetTileSize() int {
	return s.TileSize
}

// GetSpawnX returns the player spawn X position
func (s *Stage) GetSpawnX() int {
	return s.SpawnX
}

// GetSpawnY returns the player spawn Y position
func (s *Stage) GetSpawnY() int {
	return s.SpawnY
}

// LoadStage converts a StageConfig into a Stage entity
func LoadStage(cfg *config.StageConfig) *Stage {
	tileWidth := cfg.Size.Width / cfg.Size.TileSize
	tileHeight := len(cfg.Layers.Collision)

	tiles := make([][]Tile, tileHeight)
	for y, row := range cfg.Layers.Collision {
		tiles[y] = make([]Tile, tileWidth)
		for x, char := range row {
			if x >= tileWidth {
				break
			}
			charStr := string(char)
			mapping, ok := cfg.TileMapping[charStr]
			if !ok {
				tiles[y][x] = Tile{Type: TileEmpty, Solid: false}
				continue
			}

			var tileType TileType
			switch mapping.Type {
			case "wall":
				tileType = TileWall
			case "spike":
				tileType = TileSpike
			default:
				tileType = TileEmpty
			}

			tiles[y][x] = Tile{
				Type:   tileType,
				Solid:  mapping.Solid,
				Damage: mapping.Damage,
			}
		}
	}

	return &Stage{
		Width:    tileWidth,
		Height:   tileHeight,
		TileSize: cfg.Size.TileSize,
		Tiles:    tiles,
		SpawnX:   cfg.PlayerSpawn.X,
		SpawnY:   cfg.PlayerSpawn.Y,
	}
}
