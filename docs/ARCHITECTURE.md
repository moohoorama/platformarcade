# Architecture Analysis

현재 구현의 아키텍처와 설계 결정에 대한 분석 문서.

---

## 1. Overall Architecture

### 1.1 레이어 구조

```
┌─────────────────────────────────────────────────────────────┐
│                      cmd/game/main.go                       │
│                   (Composition Root)                        │
│            Game struct, Ebiten 게임 루프, 렌더링              │
└─────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│     Domain      │  │   Application   │  │  Infrastructure │
│                 │  │                 │  │                 │
│ entity/         │  │ system/         │  │ config/         │
│ - Body          │  │ - PhysicsSystem │  │ - Loader        │
│ - Player        │  │ - InputSystem   │  │ - Types         │
│ - Enemy         │  │ - CombatSystem  │  │ - Entities      │
│ - Projectile    │  │ - StageLoader   │  │ - Stage         │
│ - Stage, Tile   │  │                 │  │                 │
│                 │  │ state/          │  │                 │
│                 │  │ - GameState     │  │                 │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### 1.2 의존성 방향

```
Infrastructure → Application → Domain
     ↓               ↓            ↓
  config/         system/      entity/
```

**현재 상태:**
- `Domain (entity/)`: 외부 의존성 없음 (순수 Go)
- `Application (system/)`: Domain과 Infrastructure에 의존
- `Infrastructure (config/)`: 표준 라이브러리만 사용 (`encoding/json`, `io/fs`)

**문제점:**
- `system/` 패키지가 `config/` 패키지에 직접 의존 → Clean Architecture 위반
- 이상적으로는 `system/`이 인터페이스에 의존하고, `config/`가 이를 구현해야 함

### 1.3 Composition Root (main.go)

```go
// 의존성 조립 순서
1. config.Loader → GameConfig 로드
2. StageConfig → entity.Stage 변환
3. Game 구조체 생성 (모든 시스템 주입)
4. Ebiten.RunGame()
```

현재 `main.go`가 너무 많은 책임을 가짐:
- 설정 로딩
- 엔티티 생성
- 시스템 조립
- **렌더링 (Draw 메서드 전체)**
- UI 그리기

---

## 2. ECS vs Current Architecture

### 2.1 ECS란?

Entity-Component-System 패턴:
- **Entity**: 고유 ID만 가진 컨테이너
- **Component**: 순수 데이터 (Position, Velocity, Health)
- **System**: 특정 컴포넌트 조합을 가진 엔티티들을 처리

### 2.2 현재 구현: Object-Oriented with Systems

```go
// 현재: 객체 지향 + 시스템 하이브리드
type Player struct {
    Body                    // 상속 (임베딩)
    Hitbox TrapezoidHitbox  // 컴포지션
    Health, MaxHealth int   // 데이터
    CoyoteTimer float64     // 상태
}

type PhysicsSystem struct {
    config *config.PhysicsConfig
    stage  *entity.Stage
}

func (s *PhysicsSystem) Update(player *entity.Player, dt float64)
```

### 2.3 비교

| 측면 | ECS | 현재 구현 |
|------|-----|----------|
| 엔티티 정의 | ID + 컴포넌트 맵 | 구조체 (Player, Enemy) |
| 데이터 레이아웃 | 컴포넌트별 배열 (SoA) | 엔티티별 구조체 (AoS) |
| 시스템 쿼리 | `world.Query(Position, Velocity)` | 타입별 슬라이스 순회 |
| 확장성 | 컴포넌트 추가로 확장 | 구조체 필드 추가 |
| 복잡도 | 높음 (World, Query 필요) | 낮음 (직접적) |

### 2.4 현재 접근법의 장단점

**장점:**
- 단순함: 별도의 ECS 라이브러리 불필요
- 타입 안전: Go 컴파일러가 필드 접근 검증
- 디버깅 용이: 구조체 필드 직접 확인 가능
- 작은 규모에 적합: 엔티티 타입이 제한적 (Player, Enemy, Projectile, Gold)

**단점:**
- 코드 중복: Enemy와 Player 모두 `X, Y, VX, VY` 가짐
- 확장 어려움: 새 엔티티 타입마다 별도 처리 필요
- 시스템 간 결합: CombatSystem이 모든 엔티티 타입을 알아야 함

### 2.5 권장 사항

현재 규모에서는 ECS 도입이 과도함. 다만 개선 가능:

```go
// 공통 인터페이스 정의
type PhysicsBody interface {
    GetPosition() (int, int)
    SetPosition(x, y int)
    GetVelocity() (float64, float64)
    SetVelocity(vx, vy float64)
}

// Body가 인터페이스 구현
func (b *Body) GetPosition() (int, int) { return b.X, b.Y }
```

---

## 3. Intent & Apply Pattern

### 3.1 개념

전통적 게임 루프 문제:
```
A가 B를 밀침 → B 위치 변경 → B가 A를 밀침 → A만 영향받음 (불공평)
```

Intent & Apply 해결:
```
Phase 1 (Intent): 모든 엔티티가 "의도" 수집
Phase 2 (Apply): 모든 의도를 동시에 적용
```

### 3.2 현재 구현 분석

**CONCEPT.md에서 설계:**
```go
type MoveIntent struct {
    EntityID EntityID
    DX, DY   int
}

func (s *PhysicsSystem) Apply(intents []Intent) {
    for _, intent := range intents {
        // ...
    }
}
```

**실제 구현 (physics.go):**
```go
func (s *PhysicsSystem) Update(player *entity.Player, dt float64) {
    // Intent 단계 없음 - 직접 계산
    dx, dy := player.ApplyVelocity(dt)

    // Apply 단계
    s.applyMovement(player, dx, dy)
}
```

### 3.3 Gap 분석

| 설계 | 구현 | 상태 |
|------|------|------|
| Intent 구조체 | 없음 | ❌ 미구현 |
| 의도 수집 단계 | 없음 | ❌ 미구현 |
| 동시 적용 | Player만 처리 | ⚠️ 부분 구현 |

**현재 흐름:**
```
main.Update()
  ├→ inputSystem.UpdatePlayer()    # Player 속도 변경
  ├→ physicsSystem.Update()        # Player 이동 (즉시 적용)
  └→ combatSystem.Update()         # Enemy 이동 (별도 처리)
```

**문제점:**
- Player와 Enemy가 서로 다른 타이밍에 업데이트
- 동시 충돌 시 순서에 따라 결과 달라짐

### 3.4 개선 방향

```go
// 1. Intent 인터페이스 정의
type MoveIntent struct {
    Body *Body
    DX, DY int
}

// 2. 모든 엔티티의 Intent 수집
func (g *Game) collectIntents() []MoveIntent {
    intents := []MoveIntent{}

    // Player intent
    dx, dy := g.player.ApplyVelocity(dt)
    intents = append(intents, MoveIntent{&g.player.Body, dx, dy})

    // Enemy intents
    for _, enemy := range enemies {
        dx, dy := calculateEnemyMove(enemy)
        intents = append(intents, MoveIntent{&enemy.Body, dx, dy})
    }

    return intents
}

// 3. 동시 적용
func (g *Game) applyIntents(intents []MoveIntent) {
    for _, intent := range intents {
        s.physicsSystem.ApplyMove(intent.Body, intent.DX, intent.DY)
    }
}
```

---

## 4. Substep Collision

### 4.1 문제

고속 이동 시 터널링:
```
Frame N:   Player [■]............Wall
Frame N+1: Player ............[■]Wall  ← 벽 통과!
```

### 4.2 해결: 1픽셀 서브스텝

```go
func (s *PhysicsSystem) moveX(player *entity.Player, dx int) {
    step := sign(dx)  // +1 or -1
    for i := 0; i < abs(dx); i++ {
        if s.checkCollisionX(player, step) {
            player.VX = 0
            return  // 충돌 시 중단
        }
        player.X += step  // 1픽셀씩 이동
    }
}
```

### 4.3 성능 고려

- dx=10이면 10번 충돌 체크
- 최대 이동 속도 제한으로 완화 (MaxFallSpeed: 400px/s → 60fps에서 ~7px/frame)

### 4.4 현재 구현의 특징

```go
// X축과 Y축 분리 처리
s.moveX(player, dx)  // 먼저 X 이동
s.moveY(player, dy)  // 그 다음 Y 이동
```

**장점:** 대각선 이동 시 각 축 독립적으로 충돌 처리
**주의:** 순서가 결과에 영향 (X→Y vs Y→X)

---

## 5. Trapezoid Hitbox System

### 5.1 개념

```
     ┌───┐          ← Head (8px) - 좁음
   ┌─┴───┴─┐        ← Body (12px)
  ┌┴───────┴┐       ← Feet (16px) - 넓음
```

### 5.2 목적별 히트박스

| 히트박스 | 용도 | 효과 |
|----------|------|------|
| Head | 천장 충돌 | 좁아서 corner correction 가능 |
| Body | 피격 판정, 수평 충돌 | 표준 판정 |
| Feet | 바닥 충돌 | 넓어서 플랫폼 착지 관대 |

### 5.3 구현

```go
type TrapezoidHitbox struct {
    Head HitboxRect  // {OffsetX: 4, Width: 8}
    Body HitboxRect  // {OffsetX: 2, Width: 12}
    Feet HitboxRect  // {OffsetX: 0, Width: 16}
}

func (s *PhysicsSystem) checkCollisionY(player *entity.Player, dy int) bool {
    if dy > 0 {
        // 아래로 이동 → Feet 사용 (관대)
        hb := player.Hitbox.Feet
    } else {
        // 위로 이동 → Head 사용 (좁음)
        hb := player.Hitbox.Head
    }
    // ...
}
```

### 5.4 Corner Correction

```go
func (s *PhysicsSystem) tryCornerCorrection(player *entity.Player) {
    margin := s.config.Collision.CornerCorrection.Margin  // 4px

    // Head가 천장에 걸렸을 때
    for i := 1; i <= margin; i++ {
        // 왼쪽으로 밀어보기
        if !s.checkCollisionYAt(player, player.X - i, -1) {
            player.X -= i
            return
        }
        // 오른쪽으로 밀어보기
        if !s.checkCollisionYAt(player, player.X + i, -1) {
            player.X += i
            return
        }
    }
}
```

---

## 6. Integer Position with Remainder

### 6.1 설계 결정

```go
type Body struct {
    X, Y       int     // 정수 위치 (픽셀)
    VX, VY     float64 // 속도 (픽셀/초)
    RemX, RemY float64 // 서브픽셀 나머지
}
```

**왜 정수 위치?**
- 결정론적 충돌 판정
- 타일 기반 게임과 자연스러운 호환
- 부동소수점 오차 누적 방지

### 6.2 Remainder 시스템

```go
func (b *Body) ApplyVelocity(dt float64) (dx, dy int) {
    // 속도를 거리로 변환 + 이전 나머지 추가
    moveX := b.VX*dt + b.RemX
    moveY := b.VY*dt + b.RemY

    // 정수 부분 추출
    dx = int(moveX)
    dy = int(moveY)

    // 소수 부분 저장 (다음 프레임에 누적)
    b.RemX = moveX - float64(dx)
    b.RemY = moveY - float64(dy)

    return dx, dy
}
```

**예시:**
```
VX = 50px/s, dt = 1/60 → moveX = 0.833px
Frame 1: dx=0, RemX=0.833
Frame 2: moveX = 0.833 + 0.833 = 1.666, dx=1, RemX=0.666
Frame 3: moveX = 0.833 + 0.666 = 1.499, dx=1, RemX=0.499
```

---

## 7. Game Feel Systems

### 7.1 Coyote Time

```go
// InputSystem.updateTimers()
if player.OnGround {
    player.CoyoteTimer = s.config.Jump.CoyoteTime  // 0.1초
} else if player.CoyoteTimer > 0 {
    player.CoyoteTimer -= dt
}

// 점프 가능 조건
canJump := player.OnGround || player.CoyoteTimer > 0
```

플랫폼에서 떨어진 직후에도 점프 가능 → 관대한 조작감.

### 7.2 Jump Buffer

```go
if input.JumpPressed {
    player.JumpBufferTimer = s.config.Jump.JumpBuffer  // 0.1초
}

// 착지 전 미리 점프 입력 → 착지 즉시 점프
wantsJump := player.JumpBufferTimer > 0
if canJump && wantsJump {
    // 점프 실행
}
```

### 7.3 Variable Jump Height

```go
// 점프 버튼 떼면 상승 속도 감소
if input.JumpReleased && player.VY < 0 {
    player.VY *= s.config.Jump.VariableJumpMultiplier  // 0.4
}
```

짧게 누르면 낮은 점프, 길게 누르면 높은 점프.

### 7.4 Apex Modifier

```go
// 점프 정점에서 중력 감소 → 공중 제어 용이
if absFloat(player.VY) < s.config.Jump.ApexModifier.Threshold {
    gravity *= s.config.Jump.ApexModifier.GravityMultiplier  // 0.5
}
```

### 7.5 Dash with I-Frames

```go
func (s *InputSystem) handleDash(player *entity.Player, input InputState) {
    player.Dashing = true
    player.DashTimer = s.config.Dash.Duration      // 0.15초
    player.IframeTimer = s.config.Dash.IframesDuration  // 무적
    player.VX = dir * s.config.Dash.Speed          // 300px/s
    player.VY = 0                                   // 수직 속도 무시
}

// PhysicsSystem: 대쉬 중 중력 무시
if player.Dashing {
    return  // 중력 적용 안함
}
```

### 7.6 Hitstop & Screen Shake

```go
// main.go Update
if g.hitstopFrames > 0 {
    g.hitstopFrames--
    return nil  // 모든 업데이트 중단
}

// CombatSystem: 타격 시 콜백
if s.OnHitstop != nil {
    s.OnHitstop(s.config.Physics.Feedback.Hitstop.Frames)  // 3프레임
}
if s.OnScreenShake != nil {
    s.OnScreenShake(s.config.Physics.Feedback.ScreenShake.Intensity)
}
```

---

## 8. Combat System

### 8.1 구조

```go
type CombatSystem struct {
    config      *config.GameConfig
    stage       *entity.Stage
    projectiles []*entity.Projectile
    enemies     []*entity.Enemy
    golds       []*entity.Gold

    OnHitstop     func(frames int)
    OnScreenShake func(intensity float64)
}
```

### 8.2 Enemy AI Types

| AI Type | 행동 |
|---------|------|
| Patrol | 좌우 왕복 이동 |
| Ranged | 감지 범위 내 플레이어 향해 발사 |
| Chase | 감지 범위 내 플레이어 추적 |

```go
switch enemy.AIType {
case entity.AIPatrol:
    s.updatePatrolAI(enemy, player, dist, dt)
case entity.AIRanged:
    s.updateRangedAI(enemy, player, dist, dx, dt)
case entity.AIChase:
    s.updateChaseAI(enemy, player, dist, dx, dy, dt)
}
```

### 8.3 Collision Detection

```go
func rectsOverlap(x1, y1, w1, h1, x2, y2, w2, h2 int) bool {
    return x1 < x2+w2 && x1+w1 > x2 && y1 < y2+h2 && y1+h1 > y2
}
```

AABB (Axis-Aligned Bounding Box) 충돌 검사.

### 8.4 Damage Flow

```
Player Arrow → Enemy
    │
    ├→ proj.Deactivate()
    ├→ enemy.TakeDamage(damage)
    ├→ OnHitstop(3 frames)
    ├→ OnScreenShake(intensity)
    │
    └→ if killed:
         ├→ enemy.Active = false
         └→ spawnGold(enemy)
```

---

## 9. Data-Driven Design

### 9.1 Config 구조

```
configs/
├── physics.json     → PhysicsConfig
├── entities.json    → EntitiesConfig
└── stages/
    └── demo.json    → StageConfig
```

### 9.2 JSON → Go 매핑

```go
// entities.json의 enemies 맵
"enemies": {
    "slime": { ... },
    "archer": { ... }
}

// Go에서 map으로 로드
type EntitiesConfig struct {
    Enemies map[string]EnemyConfig `json:"enemies"`
}

// 사용
cfg.Enemies["slime"].Stats.MaxHealth
```

### 9.3 WebAssembly 지원

```go
// cmd/game/embed.go
//go:embed configs
var configFS embed.FS

// main.go
fsys, _ := fs.Sub(configFS, "configs")
loader := config.NewFSLoader(fsys, "configs")
```

`embed.FS`를 통해 WASM 빌드에서도 설정 파일 접근 가능.

---

## 10. 문제점 및 개선 방향

### 10.1 아키텍처

| 문제 | 현재 | 개선 방향 |
|------|------|----------|
| main.go 비대 | 렌더링 + 로직 + 설정 | RenderSystem 분리 |
| 의존성 방향 | system → config 직접 의존 | 인터페이스 도입 |
| 엔티티 중복 | Body 필드 반복 | 공통 인터페이스 |

### 10.2 게임플레이

| 문제 | 현재 | 개선 방향 |
|------|------|----------|
| Intent 미구현 | 순차 업데이트 | Intent 구조체 도입 |
| 카메라 분리 안됨 | main.go에 하드코딩 | CameraSystem 분리 |
| Overlap Resolution | Push-out만 | 이전 위치 복원 고려 |

### 10.3 코드 품질

| 문제 | 현재 | 개선 방향 |
|------|------|----------|
| 매직 넘버 | `8`, `12` 하드코딩 | config에서 로드 |
| 에러 처리 | 무시 또는 log.Fatal | 에러 전파 |
| 테스트 부족 | loader_test.go만 | System 테스트 추가 |

---

## 11. 결론

현재 구현은 **작은 규모의 플랫포머에 적합한 실용적 설계**를 가지고 있음:

- ✅ Clean Architecture 레이어 분리 (domain/application/infrastructure)
- ✅ Data-Driven 설계 (JSON 설정)
- ✅ 서브스텝 충돌 (터널링 방지)
- ✅ 게임필 요소 구현 (Coyote Time, Jump Buffer, Dash)
- ⚠️ Intent & Apply 설계했으나 미구현
- ⚠️ main.go에 렌더링 로직 집중
- ❌ 공통 인터페이스 부재 (PhysicsBody 등)

**다음 단계 우선순위:**
1. RenderSystem 분리 (main.go 정리)
2. Intent & Apply 실제 구현
3. PhysicsBody 인터페이스 도입
4. CameraSystem 분리
