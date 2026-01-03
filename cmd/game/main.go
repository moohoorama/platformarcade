package main

import (
	"flag"
	"io/fs"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/younwookim/mg/internal/application/game"
	"github.com/younwookim/mg/internal/application/scene/playing"
	"github.com/younwookim/mg/internal/application/system"
	"github.com/younwookim/mg/internal/infrastructure/config"
)

func main() {
	// Parse command line flags
	recordFlag := flag.String("record", "", "Record input to file (e.g., -record replay.json)")
	flag.Parse()

	recordFilename := *recordFlag

	// Load configurations using embedded filesystem
	fsys, err := fs.Sub(configFS, "configs")
	if err != nil {
		log.Fatalf("Failed to get config subfs: %v", err)
	}
	loader := config.NewFSLoader(fsys, "configs")
	cfg, err := loader.LoadAll()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Load stage
	stageCfg, err := loader.LoadStage("demo")
	if err != nil {
		log.Fatalf("Failed to load stage: %v", err)
	}
	stage := system.LoadStage(stageCfg)

	// Create initial scene (Playing)
	playingScene := playing.New(cfg, stageCfg, stage, recordFilename)

	// Create game manager with scene
	screenW := cfg.Physics.Display.ScreenWidth
	screenH := cfg.Physics.Display.ScreenHeight
	gameManager := game.New(playingScene, screenW, screenH)

	// Set up ebiten
	ebiten.SetWindowSize(screenW*cfg.Physics.Display.Scale, screenH*cfg.Physics.Display.Scale)
	ebiten.SetWindowTitle("Platform Action Game")
	ebiten.SetTPS(cfg.Physics.Display.Framerate)

	// Run game
	if err := ebiten.RunGame(gameManager); err != nil {
		log.Fatal(err)
	}
}
