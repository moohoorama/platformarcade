package system

import (
	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

// LoadStage converts a StageConfig into a Stage entity
func LoadStage(cfg *config.StageConfig) *entity.Stage {
	tileWidth := cfg.Size.Width / cfg.Size.TileSize
	tileHeight := len(cfg.Layers.Collision)

	tiles := make([][]entity.Tile, tileHeight)
	for y, row := range cfg.Layers.Collision {
		tiles[y] = make([]entity.Tile, tileWidth)
		for x, char := range row {
			if x >= tileWidth {
				break
			}
			charStr := string(char)
			mapping, ok := cfg.TileMapping[charStr]
			if !ok {
				tiles[y][x] = entity.Tile{Type: entity.TileEmpty, Solid: false}
				continue
			}

			var tileType entity.TileType
			switch mapping.Type {
			case "wall":
				tileType = entity.TileWall
			case "spike":
				tileType = entity.TileSpike
			default:
				tileType = entity.TileEmpty
			}

			tiles[y][x] = entity.Tile{
				Type:   tileType,
				Solid:  mapping.Solid,
				Damage: mapping.Damage,
			}
		}
	}

	return &entity.Stage{
		Width:    tileWidth,
		Height:   tileHeight,
		TileSize: cfg.Size.TileSize,
		Tiles:    tiles,
		SpawnX:   cfg.PlayerSpawn.X,
		SpawnY:   cfg.PlayerSpawn.Y,
	}
}
