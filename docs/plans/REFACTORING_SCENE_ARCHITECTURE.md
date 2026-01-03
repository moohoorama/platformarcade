# Scene Architecture Refactoring Plan

## 개요

`cmd/game/main.go`에 집중된 게임 로직을 Scene 패턴으로 분리하여 확장성과 테스트 용이성을 개선한다.

---

## 현재 구조 (Before)

```
cmd/game/
├── main.go              # Game 구조체 + 게임 루프 + 렌더링 + 모든 상태 처리
├── replay.go            # Recorder, Replayer, ReplayData
├── replay_test.go       # 테스트 (package main이라 import 불가)
└── embed.go             # 리소스 임베딩

internal/
├── application/
│   ├── state/           # GameState enum만
│   └── system/          # Physics, Input, Combat
├── domain/entity/
└── infrastructure/config/
```

### 문제점

| 문제 | 설명 |
|------|------|
| `package main` | 다른 패키지에서 import 불가, 테스트 어려움 |
| `main.go` 비대 | 렌더링 + 상태 관리 + 입력 처리가 한 파일에 |
| 확장성 부족 | 메뉴, 설정, 스토리 등 화면 추가 시 main.go가 계속 커짐 |
| Replay 위치 | 테스트에서 Replayer import 불가능 |

---

## 목표 구조 (After)

```
cmd/game/
└── main.go                        # 진입점만 (Game 생성, ebiten.RunGame)

internal/
├── application/
│   ├── game/
│   │   └── game.go                # Game 구조체, Scene 관리
│   │
│   ├── scene/
│   │   ├── scene.go               # Scene 인터페이스 정의
│   │   │
│   │   ├── playing/               # 전투 화면
│   │   │   ├── playing.go         # Playing scene (현재 main.go의 핵심 로직)
│   │   │   ├── playing_test.go
│   │   │   ├── renderer.go        # 렌더링 로직 분리
│   │   │   └── recorder.go        # Recorder (Playing 전용)
│   │   │
│   │   ├── title/                 # 타이틀 화면 (향후)
│   │   │   └── title.go
│   │   │
│   │   ├── menu/                  # 메인 메뉴 (향후)
│   │   │   └── menu.go
│   │   │
│   │   └── settings/              # 설정 화면 (향후)
│   │       └── settings.go
│   │
│   ├── replay/
│   │   ├── data.go                # ReplayData, FrameInput (공용)
│   │   ├── replayer.go            # Replayer (테스트에서 import 가능)
│   │   └── replay_test.go         # Replay 테스트
│   │
│   ├── system/                    # 기존 유지
│   │   ├── physics.go
│   │   ├── physics_test.go
│   │   ├── input.go
│   │   ├── combat.go
│   │   └── combat_test.go
│   │
│   └── state/                     # 기존 유지 (또는 제거 - Scene으로 대체)
│
├── domain/entity/                 # 기존 유지
│
└── infrastructure/config/         # 기존 유지
```

---

## 핵심 컴포넌트

### 1. Scene 인터페이스

```go
// internal/application/scene/scene.go
package scene

import "github.com/hajimehoshi/ebiten/v2"

// Scene represents a game screen (title, menu, playing, etc.)
type Scene interface {
    // Update updates the scene state
    // Returns the next scene if transition needed, nil to stay
    Update(dt float64) (next Scene, err error)

    // Draw renders the scene
    Draw(screen *ebiten.Image)

    // OnEnter is called when entering this scene
    OnEnter()

    // OnExit is called when leaving this scene
    OnExit()
}
```

### 2. Game (Scene 관리자)

```go
// internal/application/game/game.go
package game

import (
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/younwookim/mg/internal/application/scene"
    "github.com/younwookim/mg/internal/application/scene/playing"
)

type Game struct {
    current scene.Scene
}

func New() *Game {
    g := &Game{}
    // 초기 Scene 설정 (Playing 또는 Title)
    g.current = playing.New()
    g.current.OnEnter()
    return g
}

func (g *Game) Update() error {
    dt := 1.0 / 60.0

    next, err := g.current.Update(dt)
    if err != nil {
        return err
    }

    // Scene 전환
    if next != nil {
        g.current.OnExit()
        g.current = next
        g.current.OnEnter()
    }

    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    g.current.Draw(screen)
}

func (g *Game) Layout(w, h int) (int, int) {
    return 320, 240 // 또는 config에서 로드
}
```

### 3. Playing Scene

```go
// internal/application/scene/playing/playing.go
package playing

import (
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/younwookim/mg/internal/application/replay"
    "github.com/younwookim/mg/internal/application/scene"
    "github.com/younwookim/mg/internal/application/system"
    "github.com/younwookim/mg/internal/domain/entity"
    "github.com/younwookim/mg/internal/infrastructure/config"
)

type Playing struct {
    // Config
    cfg *config.GameConfig

    // Entities
    player *entity.Player
    stage  *entity.Stage

    // Systems
    inputSystem   *system.InputSystem
    physicsSystem *system.PhysicsSystem
    combatSystem  *system.CombatSystem

    // State
    paused bool

    // Recording (optional)
    recorder *Recorder

    // Callbacks for scene transition
    onGameOver func() scene.Scene
}

func New(/* dependencies */) *Playing {
    return &Playing{
        // 초기화
    }
}

func (p *Playing) Update(dt float64) (scene.Scene, error) {
    if p.paused {
        return p.updatePaused(dt)
    }
    return p.updatePlaying(dt)
}

func (p *Playing) Draw(screen *ebiten.Image) {
    // 렌더링 (renderer.go로 분리 가능)
}

func (p *Playing) OnEnter() {
    // Scene 진입 시 초기화
}

func (p *Playing) OnExit() {
    // Scene 이탈 시 정리 (녹화 저장 등)
    if p.recorder != nil {
        p.recorder.Save("replay.json")
    }
}
```

### 4. Replay 패키지

```go
// internal/application/replay/data.go
package replay

// FrameInput records input state for a single frame
type FrameInput struct {
    F   int  `json:"f"`
    L   bool `json:"l,omitempty"`
    R   bool `json:"r,omitempty"`
    // ... 기존과 동일
}

// ReplayData contains all data needed to replay a game session
type ReplayData struct {
    Version   string       `json:"version"`
    Seed      int64        `json:"seed"`
    Stage     string       `json:"stage"`
    StartTime string       `json:"startTime"`
    Frames    []FrameInput `json:"frames"`
}
```

```go
// internal/application/replay/replayer.go
package replay

import "github.com/younwookim/mg/internal/application/system"

// Replayer handles input playback from recorded data
type Replayer struct {
    data  ReplayData
    frame int
}

func NewReplayer(data ReplayData) *Replayer {
    return &Replayer{data: data, frame: 0}
}

func (r *Replayer) GetInput() (system.InputState, bool) {
    // 기존 로직
}

// ... 나머지 메서드
```

### 5. Recorder (Playing 전용)

```go
// internal/application/scene/playing/recorder.go
package playing

import (
    "github.com/younwookim/mg/internal/application/replay"
    "github.com/younwookim/mg/internal/application/system"
)

// Recorder handles input recording during gameplay
type Recorder struct {
    data      replay.ReplayData
    recording bool
    frame     int
}

func NewRecorder(seed int64, stage string) *Recorder {
    return &Recorder{
        data: replay.ReplayData{
            Version: "1.0",
            Seed:    seed,
            Stage:   stage,
            Frames:  make([]replay.FrameInput, 0, 3600),
        },
        recording: true,
    }
}

func (r *Recorder) RecordFrame(input system.InputState) {
    // 기존 로직
}

func (r *Recorder) GetReplayData() replay.ReplayData {
    return r.data
}
```

### 6. 진입점

```go
// cmd/game/main.go
package main

import (
    "log"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/younwookim/mg/internal/application/game"
)

func main() {
    ebiten.SetWindowSize(640, 480)
    ebiten.SetWindowTitle("MG")

    g := game.New()

    if err := ebiten.RunGame(g); err != nil {
        log.Fatal(err)
    }
}
```

---

## 의존성 다이어그램

```
cmd/game/main.go
    │
    └── application/game.Game
            │
            ├── scene.Scene (interface)
            │
            └── scene/playing.Playing
                    │
                    ├── replay.ReplayData     ← import
                    ├── replay.Replayer       ← (테스트에서 사용)
                    │
                    ├── system.InputSystem    ← import
                    ├── system.PhysicsSystem  ← import
                    ├── system.CombatSystem   ← import
                    │
                    ├── entity.Player         ← import
                    ├── entity.Stage          ← import
                    │
                    └── config.GameConfig     ← import
```

---

## Replay 시스템 배치 이유

| 컴포넌트 | 위치 | 이유 |
|----------|------|------|
| `ReplayData` | `application/replay/` | Recorder, Replayer, 테스트 모두 사용 |
| `FrameInput` | `application/replay/` | ReplayData의 일부 |
| `Replayer` | `application/replay/` | 테스트에서 import 필요, 전투 로직 모름 |
| `Recorder` | `scene/playing/` | Playing scene에서만 사용, 전투 중 녹화 |

**Replayer가 전투 로직을 모르는 이유:**
- Replayer는 입력 데이터를 순차적으로 반환하는 Iterator일 뿐
- 실제 시뮬레이션은 테스트 코드 또는 Playing scene에서 수행
- 따라서 System들에 의존하지 않음

---

## 마이그레이션 단계

### Phase 1: 패키지 구조 생성
1. `internal/application/game/` 디렉토리 생성
2. `internal/application/scene/` 디렉토리 생성
3. `internal/application/replay/` 디렉토리 생성

### Phase 2: Replay 분리
1. `cmd/game/replay.go` → `internal/application/replay/`로 이동
2. `ReplayData`, `FrameInput` → `replay/data.go`
3. `Replayer` → `replay/replayer.go`
4. `Recorder`는 나중에 이동 (Phase 4에서)

### Phase 3: Scene 인터페이스 정의
1. `scene/scene.go` 작성
2. `game/game.go` 작성 (Scene 관리자)

### Phase 4: Playing Scene 분리
1. `cmd/game/main.go`의 Game 구조체 → `scene/playing/playing.go`
2. 렌더링 로직 → `scene/playing/renderer.go`
3. `Recorder` → `scene/playing/recorder.go`
4. 테스트 이동 및 수정

### Phase 5: 진입점 정리
1. `cmd/game/main.go` 최소화
2. 통합 테스트 실행
3. 기존 테스트 수정

---

## 고려 사항

### Config/Embed 처리
- `cmd/game/embed.go`의 `//go:embed`는 유지
- `game.New()`에서 embed.FS를 받아 config 로드
- 또는 별도의 `loader` 패키지로 분리

### State 패키지
- 현재 `application/state/`에 `GameState` enum이 있음
- Scene 패턴 도입 시 불필요해질 수 있음
- Playing 내부 상태(Paused, GameOver)로 대체 가능

### 테스트 전략
- `replay/replay_test.go`: Replayer 단위 테스트
- `scene/playing/playing_test.go`: Playing scene 통합 테스트
- `game/game_test.go`: Scene 전환 테스트

---

## 향후 확장

Scene 패턴 도입 후 쉽게 추가 가능:

```
scene/
├── title/        # 타이틀 화면 (Press Start)
├── menu/         # 메인 메뉴 (New Game, Continue, Settings)
├── settings/     # 설정 화면 (Sound, Controls)
├── saveload/     # 세이브/로드 화면
├── story/        # 스토리 컷씬
├── playing/      # 전투 (현재)
├── gameover/     # 게임 오버 화면
└── credits/      # 크레딧
```

각 Scene은 독립적으로 개발/테스트 가능하며, `next Scene` 반환으로 전환 처리.
