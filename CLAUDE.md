# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

2D platformer action game built with Go and Ebitengine. Features arrow projectiles with physics, trapezoid hitboxes, and game feel mechanics (coyote time, dash with i-frames, hitstop).

## Build & Run Commands

```bash
make build          # Build native binary to bin/mg
make run            # Build and run the game
make dev            # Run directly without building to bin (go run)
make wasm           # Build WebAssembly to web/game.wasm
make serve          # Build WASM and serve at http://localhost:8080
make test           # Run all tests
make test-cover     # Run tests with coverage
make fmt            # Format code
make lint           # Run golangci-lint
```

### Running Specific Tests
```bash
go test -v -run TestFunctionName ./internal/...
go test -v ./internal/infrastructure/config/...
```

## Architecture

```
cmd/game/main.go    ← Entry point, dependency injection, Ebiten game loop
     │
     ├── domain/entity/       ← Core entities (Body, Player, Enemy, Projectile)
     ├── application/system/  ← Game systems (Physics, Input, Combat)
     ├── application/state/   ← Game state enum
     └── infrastructure/config/ ← JSON config loader
```

### Intent & Apply Physics Model

Physics uses a two-phase approach for fair simultaneous updates:
1. **Intent phase**: Collect movement intentions from all entities
2. **Apply phase**: Resolve collisions with 1-pixel substeps

Positions are integers for deterministic collision; velocities are floats with remainder accumulation.

### Trapezoid Hitbox System

Player has three hitbox regions:
- **Head**: Narrow - forgiving ceiling collision (corner correction)
- **Body**: Standard hitbox for damage detection
- **Feet**: Wide - forgiving ground collision (coyote time)

### Systems

- **PhysicsSystem** (`internal/application/system/physics.go`): Gravity, substep collision, corner correction, overlap resolution
- **InputSystem** (`internal/application/system/input.go`): Keyboard input, coyote time, jump buffer, dash handling
- **CombatSystem** (`internal/application/system/combat.go`): Projectiles, enemy AI, gold drops, damage + knockback

## Configuration

All game parameters are data-driven via JSON in `configs/`:
- `physics.json` - Gravity, jump, dash, feedback (hitstop, screen shake)
- `entities.json` - Player, enemies, projectiles, pickups definitions
- `stages/demo.json` - Stage layout with ASCII tilemap

Configs are embedded via `cmd/game/embed.go` for WebAssembly builds.

## Key Mechanics

| Mechanic | Implementation |
|----------|----------------|
| Coyote time | `player.CoyoteTimer` - allows jump after leaving ground |
| Jump buffer | `player.JumpBufferTimer` - queues jump before landing |
| Variable jump | Release jump early → `VY *= 0.4` for lower jumps |
| Dash | Fixed duration with i-frames, cooldown reset on ground |
| Arrow physics | 20° launch angle, gravity acceleration, sprite rotation |

## Tile Types

- `#` Wall - solid collision
- `S` Spike - damages player
- `.` Empty

## Controls

Arrow/WASD: Move | Z/Space: Jump | X: Attack | C: Dash | Tab: Hitbox debug | ESC: Pause
