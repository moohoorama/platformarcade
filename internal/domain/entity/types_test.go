package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestStage() *Stage {
	// Create a 3x3 stage with some solid tiles
	tiles := [][]Tile{
		{{Type: TileWall, Solid: true}, {Type: TileEmpty, Solid: false}, {Type: TileWall, Solid: true}},
		{{Type: TileEmpty, Solid: false}, {Type: TileEmpty, Solid: false}, {Type: TileEmpty, Solid: false}},
		{{Type: TileWall, Solid: true}, {Type: TileSpike, Solid: false, Damage: 10}, {Type: TileWall, Solid: true}},
	}

	return &Stage{
		Width:    3,
		Height:   3,
		TileSize: 16,
		Tiles:    tiles,
		SpawnX:   24,
		SpawnY:   24,
	}
}

func TestStage_GetTile(t *testing.T) {
	stage := createTestStage()

	tests := []struct {
		name      string
		tx, ty    int
		wantType  TileType
		wantSolid bool
	}{
		{"top-left wall", 0, 0, TileWall, true},
		{"top-center empty", 1, 0, TileEmpty, false},
		{"center empty", 1, 1, TileEmpty, false},
		{"bottom-center spike", 1, 2, TileSpike, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tile := stage.GetTile(tt.tx, tt.ty)
			assert.Equal(t, tt.wantType, tile.Type)
			assert.Equal(t, tt.wantSolid, tile.Solid)
		})
	}
}

func TestStage_GetTile_OutOfBounds(t *testing.T) {
	stage := createTestStage()

	outOfBoundsCases := []struct {
		name   string
		tx, ty int
	}{
		{"negative x", -1, 0},
		{"negative y", 0, -1},
		{"x too large", 10, 0},
		{"y too large", 0, 10},
		{"both negative", -1, -1},
	}

	for _, tt := range outOfBoundsCases {
		t.Run(tt.name, func(t *testing.T) {
			tile := stage.GetTile(tt.tx, tt.ty)
			assert.Equal(t, TileWall, tile.Type, "out of bounds should return wall")
			assert.True(t, tile.Solid, "out of bounds should be solid")
		})
	}
}

func TestStage_GetTileAtPixel(t *testing.T) {
	stage := createTestStage()

	tests := []struct {
		name      string
		px, py    int
		wantType  TileType
		wantSolid bool
	}{
		{"pixel in top-left tile", 8, 8, TileWall, true},
		{"pixel at tile boundary", 16, 0, TileEmpty, false},
		{"pixel in center tile", 24, 24, TileEmpty, false},
		{"pixel in bottom-center", 24, 40, TileSpike, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tile := stage.GetTileAtPixel(tt.px, tt.py)
			assert.Equal(t, tt.wantType, tile.Type)
			assert.Equal(t, tt.wantSolid, tile.Solid)
		})
	}
}

func TestStage_IsSolidAt(t *testing.T) {
	stage := createTestStage()

	tests := []struct {
		name   string
		px, py int
		want   bool
	}{
		{"solid wall", 0, 0, true},
		{"empty space", 24, 24, false},
		{"spike (not solid)", 24, 40, false},
		{"out of bounds", -5, -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stage.IsSolidAt(tt.px, tt.py))
		})
	}
}

func TestTileTypes(t *testing.T) {
	// Verify tile type constants
	assert.Equal(t, TileType(0), TileEmpty)
	assert.Equal(t, TileType(1), TileWall)
	assert.Equal(t, TileType(2), TileSpike)
}
