# Code Review: master (Initial Commit)

## Status
- Started: 2025-12-28 16:00
- Last Updated: 2025-12-28 16:30
- Status: Completed

## Review Scope
전체 프로젝트 첫 커밋 리뷰 (35개 파일, 15개 Go 소스 파일)

---

## Test Coverage

### internal/infrastructure/config

| Function | Coverage |
|----------|----------|
| `NewLoader` | 100.0% |
| `NewFSLoader` | 0.0% |
| `LoadPhysics` | 71.4% |
| `LoadEntities` | 71.4% |
| `LoadStage` | 75.0% |
| `LoadAll` | 71.4% |

**Overall Coverage**: 71.0%
**Coverage Profile**: `/tmp/cover_config.out`

**Missing Coverage**:
- `NewFSLoader` - embed.FS용 생성자, 테스트 미작성
- Error paths (파일 읽기 실패, JSON 파싱 실패)

---

## Function Reviews

### physics.go - PhysicsSystem

#### resolveOverlap (Lines 204-291) - Push-out 로직
**Summary**: 플레이어가 벽에 끼었을 때 가장 짧은 방향으로 밀어냄
**Concerns**:
1. **[WARNING]** `len(options) == 0`일 때 조용히 실패 - 플레이어가 갇힌 상태로 유지됨
2. **[INFO]** 프레임당 2회 호출 (이동 전/후) - 방어적이지만 중복 가능성

#### isSolidRect (Lines 294-316) - 사각형 충돌 체크
**Summary**: 사각형 내 타일이 solid인지 확인
**Concerns**:
1. **[CRITICAL]** 4개 꼭지점 + 4개 변 중점만 체크 → 32px 이상 히트박스에서 내부 타일 누락 가능
   ```
   +--------+--------+--------+
   | check  |        | check  |  ← 상단 체크됨
   +--------+--------+--------+
   |        | SOLID  |        |  ← 중앙 solid 타일 누락!
   +--------+--------+--------+
   | check  |        | check  |  ← 하단 체크됨
   +--------+--------+--------+
   ```

#### checkCollisionX/Y (Lines 143-165)
**Summary**: 이동 전 충돌 검사
**Concerns**:
1. **[INFO]** Magic number `16` (sprite width) 하드코딩 → 상수 또는 config로 추출 권장

#### moveX/moveY (Lines 93-140)
**Summary**: 1픽셀씩 substep 이동
**Concerns**: None - 올바른 구현

---

### combat.go - CombatSystem

#### moveEnemyX/moveEnemyY (Lines 227-279)
**Summary**: 적 substep 이동 (신규 추가)
**Concerns**:
1. **[WARNING]** Sub-pixel 이동 손실 - `int(math.Abs(moveX))` 절삭으로 0.9px/frame 이동 시 실제 이동 0
   - Player의 `Body.ApplyVelocity()`는 remainder 누적 있음
   - Enemy는 없음 → 느린 적은 움직이지 않을 수 있음

#### applyEnemyGravity (Lines 282-294)
**Summary**: 비행하지 않는 적에게 중력 적용
**Concerns**: None - 올바른 구현

#### updateGolds (Lines 296-369)
**Summary**: 골드 substep 물리 (신규 추가)
**Concerns**:
1. **[WARNING]** 하드코딩된 히트박스 값 (`6`, `4`, `8`, `12`, `16`)
2. **[INFO]** `BounceDecay > 1`이면 바운스마다 가속 (config 검증 필요)

#### updateChaseAI (Lines 196-224)
**Summary**: 추적 AI 이동
**Concerns**:
1. **[INFO]** 대각선 이동 시 속도 정규화 없음 (√2배 빠름)
2. **[INFO]** `moveEnemyX`가 `PatrolDir` 수정 - Chase AI에는 무의미하지만 무해

#### updatePatrolAI (Lines 150-165)
**Summary**: 순찰 AI 이동
**Concerns**:
1. **[INFO]** `player`, `dist` 파라미터 미사용

#### checkCollisions (Lines 371-446)
**Summary**: 엔티티 간 충돌 검사
**Concerns**:
1. **[INFO]** 플레이어 히트박스 중복 계산 (Lines 416-419, 436-439)

#### rectsOverlap (Lines 499-501)
**Summary**: AABB 충돌 테스트
**Concerns**: None - 올바른 구현

---

## Final Review Results

### Overview
| 항목 | 값 |
|------|-----|
| 파일 수 | 35개 (Go 15개, JSON 4개, 기타) |
| 추가 라인 | ~3,500+ |
| 핵심 시스템 | Physics, Combat, Input |
| 테스트 커버리지 | config: 71% |

### Key Changes
- **cmd/game/main.go**: Ebiten 게임 루프, 렌더링, UI
- **physics.go**: Intent & Apply 물리, substep 충돌, push-out
- **combat.go**: 적 AI, 투사체, 골드, 데미지 시스템
- **input.go**: 코요테 타임, 점프 버퍼, 대시
- **entity/*.go**: Player, Enemy, Projectile, Gold 엔티티

### Concerns (Severity Order)

| Severity | Issue | Location | Recommendation |
|----------|-------|----------|----------------|
| **CRITICAL** | `isSolidRect` 대형 히트박스에서 내부 타일 누락 | physics.go:294 | 타일 순회 또는 히트박스 크기 제한 문서화 |
| **WARNING** | `resolveOverlap` 실패 시 무시 | physics.go:262 | 로깅 또는 스폰 리셋 고려 |
| **WARNING** | Enemy sub-pixel 이동 손실 | combat.go:227,256 | remainder 누적 추가 |
| **WARNING** | Gold 히트박스 하드코딩 | combat.go:319-359 | config로 추출 |
| INFO | Magic number 16 (sprite width) | physics.go | 상수 추출 |
| INFO | 대각선 이동 속도 정규화 없음 | combat.go:217-222 | 의도적이면 OK |
| INFO | 미사용 파라미터 | combat.go:150 | 제거 또는 주석 |

### Testing Requirements
1. **Unit Tests 필요**:
   - `resolveOverlap` - 4방향 push-out 검증
   - `moveEnemyX/Y` - 벽 충돌, 방향 전환
   - `updateGolds` - substep 충돌, 바운스

2. **Integration Tests 권장**:
   - 대각선 점프 → 플랫폼 착지 시나리오
   - 빠른 속도 적/골드의 벽 통과 방지 검증

---
## PR Body (Copy below for PR)
---

# 플랫폼 액션 게임 초기 구현

## 개요
Ebiten-Go 기반 2D 플랫폼 액션 게임 프로토타입. Intent & Apply 물리 모델, 사다리꼴 히트박스, 게임 필 요소 (코요테 타임, 점프 버퍼, 대시) 구현.

## 수정내역

### 게임 루프 및 렌더링 (`cmd/game/main.go`)
- Ebiten 게임 구조체, Update/Draw 루프
- 타일맵, 플레이어, 적, 투사체, UI 렌더링
- 일시정지, 게임오버 상태 처리

### 물리 시스템 (`internal/application/system/physics.go`)
- Integer 위치 + subpixel remainder 누적
- 1픽셀 substep 충돌 감지
- `resolveOverlap`: 벽 끼임 시 push-out
- 사다리꼴 히트박스 (Head/Body/Feet)

### 전투 시스템 (`internal/application/system/combat.go`)
- 적 AI: Patrol, Ranged, Chase
- 투사체: 20도 발사각 + 중력 가속
- 골드 드롭: substep 물리 + 바운스
- 피격: 히트스탑, 화면 흔들림, 넉백

### 입력 시스템 (`internal/application/system/input.go`)
- 코요테 타임, 점프 버퍼
- 가변 점프 높이, apex modifier
- 대시 + i-frame

### 엔티티 (`internal/domain/entity/`)
- `Body`: Integer 위치, TrapezoidHitbox
- `Player`, `Enemy`, `Projectile`, `Gold`

### 설정 로더 (`internal/infrastructure/config/`)
- JSON 기반 설정 (physics, entities, stages)
- `embed.FS` 지원 (WebAssembly)

### 빌드/배포
- `Makefile`: native, wasm, run, test
- GitHub Actions: Pages 자동 배포
