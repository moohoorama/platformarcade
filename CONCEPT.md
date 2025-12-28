# Platform Action Game - Concept Document

## Overview

2D 플랫폼 액션 게임. 화살을 쏘며 적을 처치하고 스테이지를 탐험한다.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         main.go                             │
│                    (조립, 의존성 주입)                        │
└─────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│     Domain      │  │   Application   │  │  Infrastructure │
│                 │  │                 │  │                 │
│ - Entity        │  │ - GameState     │  │ - Ebiten        │
│ - Combat Rules  │  │ - Systems       │  │ - JSON Loader   │
│ - Physics Rules │  │ - Intent/Apply  │  │ - Assets        │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

## Directory Structure

```
mg/
├── cmd/
│   └── game/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── entity/
│   │   │   ├── body.go         # 물리 바디 (위치, 속도, 히트박스)
│   │   │   ├── player.go
│   │   │   ├── enemy.go
│   │   │   └── projectile.go
│   │   ├── combat/
│   │   │   └── damage.go
│   │   └── physics/
│   │       └── collision.go
│   ├── application/
│   │   ├── state/
│   │   │   ├── state.go        # GameState enum
│   │   │   └── manager.go      # State Stack
│   │   ├── system/
│   │   │   ├── input.go
│   │   │   ├── physics.go      # Intent 수집 + Apply
│   │   │   ├── combat.go
│   │   │   ├── camera.go
│   │   │   └── render.go
│   │   └── ecs/
│   │       └── world.go
│   └── infrastructure/
│       ├── config/
│       │   └── loader.go       # JSON 로더
│       └── asset/
│           └── sprite.go
├── configs/
│   ├── physics.json            # 물리/게임필 설정
│   ├── entities.json           # 캐릭터/적/투사체 정의
│   └── stages/
│       ├── demo.json           # 데모 스테이지
│       └── stage1.json
└── assets/
    └── sprites/
```

## Core Mechanics

### Controls

| Key | Action | Description |
|-----|--------|-------------|
| ← → | Move | 좌우 이동 |
| ↑ ↓ | Aim | 조준 (선택적) |
| Z | Jump | 점프 (가변 높이) |
| X | Arrow | 화살 발사 (중력 영향) |
| C | Dash | 대쉬 (i-frame 포함) |
| ESC | Pause | 일시정지 |

### Player Hitbox (평행사변형 근사)

```
     ┌───┐          ← Head (좁음, 8px)
   ┌─┴───┴─┐        ← Body (중간, 12px)
  ┌┴───────┴┐       ← Feet (넓음, 16px)

코너 보정: Head 영역
코요테 타임: Feet 영역 확장
```

```go
type Hitbox struct {
    Head Rect  // 좁음 - 천장 충돌 관대
    Body Rect  // 기본 피격 판정
    Feet Rect  // 넓음 - 바닥 충돌 관대
}
```

### Arrow Projectile

- 발사 시 초기 속도: 수평 + 약간 위로 (자연스러운 시작)
- 중력 가속도: 매 프레임 VY += gravityAccel * dt
- 스프라이트 회전: 속도 벡터 방향으로 자동 회전
- 사거리: 화면 중간 정도 (~50% 화면 너비)
- 벽 충돌 시 소멸
- 적 충돌 시 데미지 + 소멸

```
포물선 궤적 (launchAngleDeg: 20도):

발사 각도: 20° 위쪽으로 시작
중력에 의해 점점 아래로 꺾임

진행 방향 (스프라이트 회전):
  t=0.0s:  +20°  ↗ (위쪽 20도로 발사)
  t=0.1s:  +10°  ↗ (조금 수평에 가까워짐)
  t=0.2s:   0°   → (수평)
  t=0.3s:  -15°  ↘ (아래로 꺾임)
  t=0.4s:  -30°  ↘ (더 급하게)
  t=0.5s:  -45°  ↓ (착탄)

초기 속도 계산:
  VX = speed * cos(20°) = 300 * 0.94 = 282
  VY = speed * -sin(20°) = 300 * -0.34 = -102 (위쪽이 음수)
```

```go
// 화살 발사
func NewArrow(x, y float64, facingRight bool, config ArrowConfig) *Arrow {
    angleRad := config.LaunchAngleDeg * math.Pi / 180
    dir := 1.0
    if !facingRight {
        dir = -1.0
    }

    return &Arrow{
        X:      x,
        Y:      y,
        VX:     dir * config.Speed * math.Cos(angleRad),
        VY:     -config.Speed * math.Sin(angleRad),  // 위쪽이 음수
        StartX: x,
        Config: config,
    }
}

// 화살 물리 업데이트
func (a *Arrow) Update(dt float64) {
    // 중력 가속도 적용
    a.VY += a.Config.GravityAccel * dt

    // 최대 낙하 속도 제한
    if a.VY > a.Config.MaxFallSpeed {
        a.VY = a.Config.MaxFallSpeed
    }

    // 위치 업데이트
    a.X += a.VX * dt
    a.Y += a.VY * dt
}

func (a *Arrow) Rotation() float64 {
    // 속도 벡터 방향으로 스프라이트 회전
    return math.Atan2(a.VY, a.VX)
}
```

### Stage System

```
전체 맵 구조:

┌─────────┬─────────┐
│ Stage 1 │ Stage 2 │
│ (demo)  │         │
├─────────┼─────────┤
│ Stage 3 │ Stage 4 │
│         │ (boss)  │
└─────────┴─────────┘

각 스테이지: 200% x 200% (화면 대비)
화면: 320x240 기준 → 스테이지: 640x480
```

### Collision Layers

| Layer | Collides With | Description |
|-------|---------------|-------------|
| Wall | All | 모든 것 차단 |
| Spike | Player | 플레이어에게 데미지 |
| Player | Wall, Spike, Enemy, EnemyProjectile | |
| Enemy | Wall, PlayerProjectile | |
| PlayerProjectile | Wall, Enemy | |
| EnemyProjectile | Wall, Player | |
| Gold | Wall | 바닥에 착지 후 수집 가능 |

### Combat

```
Player → Arrow → Enemy
                   │
                   ▼
              TakeDamage()
                   │
                   ▼
              SpawnGold() → 바닥으로 낙하
```

## Intent & Apply Model

```go
// Phase 1: 모든 엔티티의 의도 수집
type Intent interface{}

type MoveIntent struct {
    EntityID EntityID
    DX, DY   int  // 정수 사용
}

type AttackIntent struct {
    EntityID  EntityID
    Direction Vec2
}

type DashIntent struct {
    EntityID  EntityID
    Direction int // -1 or 1
}

// Phase 2: 충돌 해결 및 적용
func (s *PhysicsSystem) Apply(intents []Intent) {
    for _, intent := range intents {
        switch i := intent.(type) {
        case MoveIntent:
            s.applyMoveWithSubsteps(i)
        case DashIntent:
            s.applyDash(i)
        }
    }
}
```

## Substep Movement (Integer-based)

```go
const SubstepSize = 1  // 1픽셀 단위

func (s *PhysicsSystem) applyMoveWithSubsteps(body *Body, dx, dy int) {
    // X축 이동
    stepX := sign(dx)
    for i := 0; i < abs(dx); i++ {
        if s.checkCollision(body, stepX, 0) {
            body.VX = 0
            break
        }
        body.X += stepX
    }

    // Y축 이동
    stepY := sign(dy)
    for i := 0; i < abs(dy); i++ {
        if s.checkCollision(body, 0, stepY) {
            if stepY > 0 {
                body.OnGround = true
            }
            body.VY = 0
            break
        }
        body.Y += stepY
    }
}
```

## Game States

```go
type GameState int

const (
    StateMenu GameState = iota
    StateLoading
    StatePlaying
    StatePaused
    StateGameOver
    StateStageClear
)

// State Stack for overlays
// [Playing] → ESC → [Playing, Paused]
```

## JSON Configuration Files

### 1. configs/physics.json

게임 필(Game Feel) 관련 모든 수치.

### 2. configs/entities.json

플레이어, 적, 투사체 등 모든 엔티티 정의.

### 3. configs/stages/demo.json

스테이지 레이아웃 (타일, 적 배치, 연결 정보).

---

## JSON Schemas

### physics.json

```json
{
  "display": {
    "screenWidth": 320,
    "screenHeight": 240,
    "scale": 2,
    "framerate": 60
  },
  "physics": {
    "substeps": 1,
    "gravity": 800,
    "maxFallSpeed": 400,
    "useIntegerPosition": true
  },
  "movement": {
    "acceleration": 2000,
    "deceleration": 2500,
    "maxSpeed": 120,
    "airControl": 0.8,
    "turnaroundBoost": 1.5
  },
  "jump": {
    "force": 280,
    "variableJumpMultiplier": 0.4,
    "coyoteTime": 0.1,
    "jumpBuffer": 0.1,
    "apexModifier": {
      "enabled": true,
      "threshold": 20,
      "gravityMultiplier": 0.5,
      "speedBoost": 1.1
    },
    "fallMultiplier": 1.6
  },
  "dash": {
    "speed": 300,
    "duration": 0.15,
    "cooldown": 0.5,
    "iframesDuration": 0.15
  },
  "collision": {
    "cornerCorrection": {
      "enabled": true,
      "margin": 4
    },
    "ledgeAssist": {
      "enabled": true,
      "margin": 3
    }
  },
  "combat": {
    "iframes": 1.0,
    "knockback": {
      "force": 150,
      "upForce": 80,
      "stunDuration": 0.2
    }
  },
  "feedback": {
    "hitstop": {
      "enabled": true,
      "frames": 3
    },
    "screenShake": {
      "enabled": true,
      "intensity": 4,
      "decay": 0.9
    },
    "squashStretch": {
      "enabled": true,
      "landSquash": {"x": 1.3, "y": 0.7},
      "jumpStretch": {"x": 0.8, "y": 1.2},
      "duration": 0.1
    }
  }
}
```

### entities.json

```json
{
  "player": {
    "id": "player",
    "sprite": {
      "sheet": "player.png",
      "frameWidth": 16,
      "frameHeight": 24,
      "animations": {
        "idle": {"row": 0, "frames": 4, "fps": 8},
        "run": {"row": 1, "frames": 6, "fps": 12},
        "jump": {"row": 2, "frames": 2, "fps": 8},
        "fall": {"row": 3, "frames": 2, "fps": 8},
        "dash": {"row": 4, "frames": 3, "fps": 20},
        "attack": {"row": 5, "frames": 4, "fps": 16}
      }
    },
    "hitbox": {
      "head": {"offsetX": 4, "offsetY": 0, "width": 8, "height": 6},
      "body": {"offsetX": 2, "offsetY": 6, "width": 12, "height": 12},
      "feet": {"offsetX": 0, "offsetY": 18, "width": 16, "height": 6}
    },
    "hurtbox": {"offsetX": 3, "offsetY": 2, "width": 10, "height": 20},
    "stats": {
      "maxHealth": 100,
      "attackDamage": 25
    }
  },
  "projectiles": {
    "playerArrow": {
      "id": "playerArrow",
      "sprite": {
        "sheet": "projectiles.png",
        "frameWidth": 16,
        "frameHeight": 8,
        "animations": {
          "fly": {"row": 0, "frames": 2, "fps": 12}
        }
      },
      "hitbox": {"offsetX": 2, "offsetY": 2, "width": 12, "height": 4},
      "physics": {
        "speed": 250,
        "gravity": 200,
        "maxRange": 160,
        "piercing": false
      },
      "damage": 25
    }
  },
  "enemies": {
    "slime": {
      "id": "slime",
      "sprite": {
        "sheet": "enemies.png",
        "frameWidth": 16,
        "frameHeight": 16,
        "animations": {
          "idle": {"row": 0, "frames": 4, "fps": 6},
          "move": {"row": 1, "frames": 4, "fps": 8},
          "hit": {"row": 2, "frames": 2, "fps": 10}
        }
      },
      "hitbox": {
        "body": {"offsetX": 2, "offsetY": 4, "width": 12, "height": 12}
      },
      "hurtbox": {"offsetX": 2, "offsetY": 4, "width": 12, "height": 12},
      "stats": {
        "maxHealth": 50,
        "attackDamage": 10,
        "moveSpeed": 40,
        "goldDrop": {"min": 5, "max": 15}
      },
      "ai": {
        "type": "patrol",
        "detectRange": 80,
        "patrolDistance": 60
      }
    },
    "archer": {
      "id": "archer",
      "sprite": {
        "sheet": "enemies.png",
        "frameWidth": 16,
        "frameHeight": 24,
        "animations": {
          "idle": {"row": 3, "frames": 4, "fps": 6},
          "attack": {"row": 4, "frames": 4, "fps": 8}
        }
      },
      "hitbox": {
        "body": {"offsetX": 2, "offsetY": 4, "width": 12, "height": 20}
      },
      "hurtbox": {"offsetX": 3, "offsetY": 4, "width": 10, "height": 18},
      "stats": {
        "maxHealth": 30,
        "attackDamage": 15,
        "goldDrop": {"min": 10, "max": 25}
      },
      "ai": {
        "type": "ranged",
        "detectRange": 120,
        "attackRange": 100,
        "attackCooldown": 2.0
      }
    }
  },
  "pickups": {
    "gold": {
      "id": "gold",
      "sprite": {
        "sheet": "items.png",
        "frameWidth": 8,
        "frameHeight": 8,
        "animations": {
          "idle": {"row": 0, "frames": 4, "fps": 8}
        }
      },
      "hitbox": {"offsetX": 0, "offsetY": 0, "width": 8, "height": 8},
      "physics": {
        "gravity": 400,
        "bounceDecay": 0.5,
        "collectDelay": 0.3
      }
    }
  }
}
```

### stages/demo.json

```json
{
  "id": "demo",
  "name": "Demo Stage",
  "size": {
    "width": 640,
    "height": 480,
    "tileSize": 16
  },
  "tileset": "tileset.png",
  "background": {
    "image": "bg_forest.png",
    "parallax": 0.5
  },
  "connections": {
    "right": "stage1",
    "down": "stage3"
  },
  "playerSpawn": {"x": 32, "y": 400},
  "layers": {
    "background": [
      "........................................",
      "........................................",
      "........................................"
    ],
    "collision": [
      "########################################",
      "#......................................#",
      "#......................................#",
      "#......................................#",
      "#......................................#",
      "#..........####........................#",
      "#......................................#",
      "#......................................#",
      "#...####...............................#",
      "#......................................#",
      "#..............####....................#",
      "#......................................#",
      "#......................................#",
      "#.....####.............................#",
      "#......................................#",
      "#...............S..S..S................#",
      "#..####................................#",
      "#......................................#",
      "#......................................#",
      "#...........#######....................#",
      "#......................................#",
      "#......................................#",
      "#......................................#",
      "#.....SSSSS............................#",
      "#..####....####........................#",
      "#......................................#",
      "#......................................#",
      "#......................................#",
      "########################################",
      "########################################"
    ]
  },
  "tileMapping": {
    "#": {"type": "wall", "tileIndex": 1},
    "S": {"type": "spike", "tileIndex": 5},
    ".": {"type": "empty", "tileIndex": 0}
  },
  "enemies": [
    {"type": "slime", "x": 200, "y": 400},
    {"type": "slime", "x": 350, "y": 300},
    {"type": "archer", "x": 500, "y": 200}
  ],
  "triggers": [
    {
      "type": "stageTransition",
      "rect": {"x": 624, "y": 0, "w": 16, "h": 480},
      "target": "stage1",
      "spawnPoint": "left"
    }
  ]
}
```

## Implementation Order

### Phase 1: Core Systems
1. ECS 기본 구조
2. 물리 시스템 (Substep, 충돌)
3. 플레이어 이동/점프
4. 카메라 시스템

### Phase 2: Combat
5. 화살 발사/물리
6. 적 기본 AI
7. 데미지 시스템
8. 골드 드롭

### Phase 3: Game Feel
9. 코요테 타임, 점프 버퍼
10. 대쉬 + i-frames
11. 히트스탑, 스크린 쉐이크
12. 스쿼시 & 스트레치

### Phase 4: Content
13. 스테이지 로더
14. 스테이지 전환
15. UI (체력, 골드)
16. 메뉴/일시정지

---

## Technical Notes

### Integer Physics

```go
// 내부 위치는 정수 (서브픽셀 없음)
type Body struct {
    X, Y   int     // 픽셀 단위 정수
    VX, VY float64 // 속도는 float (누적 후 정수 변환)
    RemX, RemY float64 // 나머지 (다음 프레임에 반영)
}

func (b *Body) ApplyVelocity(dt float64) {
    // 속도를 픽셀로 변환하면서 나머지 보존
    moveX := b.VX*dt + b.RemX
    moveY := b.VY*dt + b.RemY

    pixelsX := int(moveX)
    pixelsY := int(moveY)

    b.RemX = moveX - float64(pixelsX)
    b.RemY = moveY - float64(pixelsY)

    // pixelsX, pixelsY를 substep으로 적용
}
```

### Collision Priority

```
1. Wall (절대 통과 불가)
2. Spike (데미지 + 통과 불가)
3. Enemy (데미지 후 통과)
4. Pickup (수집 후 제거)
```

### Camera

```go
type Camera struct {
    X, Y          int
    TargetX, TargetY int
    Smoothing     float64
    ShakeOffset   Vec2

    // 스테이지 경계 제한
    MinX, MaxX    int
    MinY, MaxY    int
}

func (c *Camera) Update(target Vec2, dt float64) {
    c.TargetX = target.X - ScreenWidth/2
    c.TargetY = target.Y - ScreenHeight/2

    // 부드러운 추적
    c.X = lerp(c.X, c.TargetX, c.Smoothing*dt)
    c.Y = lerp(c.Y, c.TargetY, c.Smoothing*dt)

    // 경계 제한
    c.X = clamp(c.X, c.MinX, c.MaxX)
    c.Y = clamp(c.Y, c.MinY, c.MaxY)
}
```
