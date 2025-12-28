# Platform Action Game

2D platformer action game built with Go and [Ebitengine](https://ebitengine.org/).

## Play Online

Play the game directly in your browser: [GitHub Pages](https://<username>.github.io/mg/)

## Features

- **Intent & Apply Physics**: Fair simultaneous entity updates with integer positions
- **Trapezoid Hitbox**: Head narrow (forgiving ceilings), feet wide (forgiving ground)
- **Arrow Projectiles**: 20-degree launch angle with gravity acceleration
- **Game Feel**: Coyote time, jump buffer, variable jump height, dash with i-frames
- **Feedback**: Hitstop and screen shake on hits
- **JSON Configuration**: All physics and entity parameters are data-driven

## Controls

| Key | Action |
|-----|--------|
| Arrow / WASD | Move |
| Z / Space | Jump |
| X | Attack (Arrow) |
| C | Dash |
| Tab | Show Hitbox |
| ESC | Pause |

## Build

### Prerequisites

- Go 1.23+
- Make

### Native Build

```bash
make build
# or
go build -o bin/game ./cmd/game
```

### WebAssembly Build

```bash
make wasm
```

Output files are in `web/` directory.

### Run Locally

```bash
make run
```

### Test

```bash
make test
```

## Project Structure

```
mg/
├── cmd/game/           # Main application
├── configs/            # JSON configuration files
│   ├── physics.json    # Physics parameters
│   ├── entities.json   # Entity definitions
│   └── stages/         # Stage definitions
├── internal/
│   ├── domain/         # Core domain entities
│   │   └── entity/     # Body, Hitbox, Player, Enemy, Projectile
│   ├── application/    # Application logic
│   │   └── system/     # Physics, Input, Combat systems
│   └── infrastructure/ # External concerns
│       └── config/     # JSON config loader
└── web/                # WebAssembly output
```

## Configuration

All game parameters are in JSON files under `configs/`:

- `physics.json` - Gravity, jump force, coyote time, dash settings, feedback
- `entities.json` - Player, enemies, projectiles, pickups
- `stages/demo.json` - Stage layout with ASCII tilemap

## Deployment

Push to `main` branch to automatically deploy to GitHub Pages via GitHub Actions.

To enable GitHub Pages:
1. Go to repository Settings > Pages
2. Set Source to "GitHub Actions"

## License

MIT
