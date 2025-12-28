package entity

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
