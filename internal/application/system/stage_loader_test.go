package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/younwookim/mg/internal/domain/entity"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

func TestLoadStage(t *testing.T) {
	t.Run("loads basic stage", func(t *testing.T) {
		cfg := &config.StageConfig{
			Size: config.StageSizeConfig{
				Width:    48,
				Height:   48,
				TileSize: 16,
			},
			PlayerSpawn: config.PositionConfig{
				X: 32,
				Y: 32,
			},
			Layers: config.LayersConfig{
				Collision: []string{
					"###",
					"#.#",
					"###",
				},
			},
			TileMapping: map[string]config.TileMappingConfig{
				"#": {Type: "wall", Solid: true},
				".": {Type: "empty", Solid: false},
			},
		}

		stage := LoadStage(cfg)

		require.NotNil(t, stage)
		assert.Equal(t, 3, stage.Width)
		assert.Equal(t, 3, stage.Height)
		assert.Equal(t, 16, stage.TileSize)
		assert.Equal(t, 32, stage.SpawnX)
		assert.Equal(t, 32, stage.SpawnY)
	})

	t.Run("maps wall tiles correctly", func(t *testing.T) {
		cfg := &config.StageConfig{
			Size: config.StageSizeConfig{
				Width:    32,
				Height:   32,
				TileSize: 16,
			},
			Layers: config.LayersConfig{
				Collision: []string{
					"##",
					"##",
				},
			},
			TileMapping: map[string]config.TileMappingConfig{
				"#": {Type: "wall", Solid: true},
			},
		}

		stage := LoadStage(cfg)

		for y := 0; y < 2; y++ {
			for x := 0; x < 2; x++ {
				tile := stage.GetTile(x, y)
				assert.Equal(t, entity.TileWall, tile.Type)
				assert.True(t, tile.Solid)
			}
		}
	})

	t.Run("maps spike tiles with damage", func(t *testing.T) {
		cfg := &config.StageConfig{
			Size: config.StageSizeConfig{
				Width:    16,
				Height:   16,
				TileSize: 16,
			},
			Layers: config.LayersConfig{
				Collision: []string{
					"^",
				},
			},
			TileMapping: map[string]config.TileMappingConfig{
				"^": {Type: "spike", Solid: false, Damage: 10},
			},
		}

		stage := LoadStage(cfg)

		tile := stage.GetTile(0, 0)
		assert.Equal(t, entity.TileSpike, tile.Type)
		assert.False(t, tile.Solid)
		assert.Equal(t, 10, tile.Damage)
	})

	t.Run("handles unknown tile mapping", func(t *testing.T) {
		cfg := &config.StageConfig{
			Size: config.StageSizeConfig{
				Width:    16,
				Height:   16,
				TileSize: 16,
			},
			Layers: config.LayersConfig{
				Collision: []string{
					"?",
				},
			},
			TileMapping: map[string]config.TileMappingConfig{
				// No mapping for "?"
			},
		}

		stage := LoadStage(cfg)

		tile := stage.GetTile(0, 0)
		assert.Equal(t, entity.TileEmpty, tile.Type)
		assert.False(t, tile.Solid)
	})

	t.Run("handles unknown tile type", func(t *testing.T) {
		cfg := &config.StageConfig{
			Size: config.StageSizeConfig{
				Width:    16,
				Height:   16,
				TileSize: 16,
			},
			Layers: config.LayersConfig{
				Collision: []string{
					"X",
				},
			},
			TileMapping: map[string]config.TileMappingConfig{
				"X": {Type: "unknown_type", Solid: false},
			},
		}

		stage := LoadStage(cfg)

		tile := stage.GetTile(0, 0)
		assert.Equal(t, entity.TileEmpty, tile.Type)
	})

	t.Run("handles row longer than width", func(t *testing.T) {
		cfg := &config.StageConfig{
			Size: config.StageSizeConfig{
				Width:    32, // 2 tiles
				Height:   16,
				TileSize: 16,
			},
			Layers: config.LayersConfig{
				Collision: []string{
					"####", // 4 characters, but only 2 tiles should be used
				},
			},
			TileMapping: map[string]config.TileMappingConfig{
				"#": {Type: "wall", Solid: true},
			},
		}

		stage := LoadStage(cfg)

		assert.Equal(t, 2, stage.Width)
		assert.Equal(t, 1, stage.Height)
	})
}
