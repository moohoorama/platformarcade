package config

// StageConfig is the root config for stage JSON files
type StageConfig struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Size        StageSizeConfig          `json:"size"`
	Tileset     string                   `json:"tileset"`
	Background  BackgroundConfig         `json:"background"`
	Connections ConnectionsConfig        `json:"connections"`
	PlayerSpawn PositionConfig           `json:"playerSpawn"`
	Layers      LayersConfig             `json:"layers"`
	TileMapping map[string]TileMappingConfig `json:"tileMapping"`
	Enemies     []EnemySpawnConfig       `json:"enemies"`
	Pickups     []PickupSpawnConfig      `json:"pickups"`
	Triggers    []TriggerConfig          `json:"triggers"`
	Decorations []DecorationConfig       `json:"decorations"`
}

type StageSizeConfig struct {
	Width    int `json:"width"`
	Height   int `json:"height"`
	TileSize int `json:"tileSize"`
}

type BackgroundConfig struct {
	Color    string  `json:"color"`
	Image    string  `json:"image"`
	Parallax float64 `json:"parallax"`
}

type ConnectionsConfig struct {
	Right *string `json:"right"`
	Left  *string `json:"left"`
	Up    *string `json:"up"`
	Down  *string `json:"down"`
}

type PositionConfig struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type LayersConfig struct {
	Collision []string `json:"collision"`
}

type TileMappingConfig struct {
	Type      string `json:"type"`
	Solid     bool   `json:"solid"`
	Damage    int    `json:"damage,omitempty"`
	TileIndex int    `json:"tileIndex"`
}

type EnemySpawnConfig struct {
	Type        string `json:"type"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	FacingRight bool   `json:"facingRight"`
}

type PickupSpawnConfig struct {
	Type string `json:"type"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

type TriggerConfig struct {
	Type       string     `json:"type"`
	Rect       RectConfig `json:"rect"`
	Target     string     `json:"target"`
	SpawnPoint string     `json:"spawnPoint"`
}

type RectConfig struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type DecorationConfig struct {
	Sprite    string `json:"sprite"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Animation string `json:"animation"`
}
