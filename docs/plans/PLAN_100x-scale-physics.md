# Implementation Plan: 100x Scale Position + Unified Sub-step Physics (Player)

**Status**: 🔄 In Progress
**Started**: 2026-01-03
**Last Updated**: 2026-01-03
**Scope**: Player entity only (1단계)

---

**CRITICAL INSTRUCTIONS**: After completing each phase:
1. Check off completed task checkboxes
2. Run all quality gate validation commands
3. Verify ALL quality gate items pass
4. Update "Last Updated" date above
5. Document learnings in Notes section
6. Only then proceed to next phase

**DO NOT skip quality gates or proceed with failing checks**

---

## Overview

### Feature Description
현재 int 위치 + float RemX/RemY 방식을 100x 스케일 int 위치로 변경하고,
프레임 스킵 방식의 슬로우모션을 통합 sub-step 방식으로 교체한다.

**핵심 변경**:
- 1 픽셀 = 100 단위 (PositionScale = 100)
- 평상시: 10 sub-step/프레임
- 슬로우모션: 1 sub-step/프레임
- RemX, RemY 제거 (정밀도가 위치에 내장)

### Success Criteria
- [ ] 플레이어가 100x 스케일 좌표로 동작
- [ ] 슬로우모션이 부드럽게 1/10 속도로 동작
- [ ] 기존 게임플레이 동일하게 유지
- [ ] 모든 테스트 통과

### User Impact
- 슬로우모션이 "뚝뚝 끊기는" 대신 "부드럽게 느려지는" 느낌으로 개선
- 물리 정밀도 향상 (0.01픽셀 단위)

---

## Architecture Decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| 100x 스케일 사용 | 0.01픽셀 정밀도, RemX/RemY 불필요 | 모든 위치 코드 수정 필요 |
| Sub-step 기반 슬로우 | 부드러운 슬로우모션, 일관된 물리 | 프레임 스킵보다 복잡 |
| Player만 먼저 적용 | 리스크 최소화, 점진적 적용 | 임시로 두 시스템 공존 |

---

## Constants & Formulas

```go
const PositionScale = 100  // 1 pixel = 100 units

// 좌표 변환
pixelX := entity.X / PositionScale
scaledX := pixelX * PositionScale

// 속도도 100x 스케일
// 기존: VX = 120 (pixels/sec)
// 변경: VX = 12000 (units/sec, 120 * 100)

// Sub-step 계산
subSteps := 10  // 평상시
if slowMotion {
    subSteps = 1
}
dtPerStep := dt / 10  // 항상 1/10 dt
for i := 0; i < subSteps; i++ {
    updatePhysics(dtPerStep)
}
```

---

## Implementation Phases

### Phase 1: Body 100x Scale 변환
**Goal**: Body 구조체를 100x 스케일로 변환, RemX/RemY 제거
**Estimated Time**: 2-3 hours
**Status**: ✅ Complete

#### Tasks

**RED: Write Failing Tests First**
- [x] **Test 1.1**: Body 100x 스케일 테스트 작성
  - File: `internal/domain/entity/body_test.go`
  - Test cases:
    - `TestBody_PositionScale`: X=100은 1픽셀
    - `TestBody_ApplyVelocity_Scaled`: 속도 100x에서 올바른 이동
    - `TestBody_PixelPosition`: PixelX(), PixelY() 헬퍼 함수

- [x] **Test 1.2**: Player 100x 스케일 생성 테스트
  - File: `internal/domain/entity/body_test.go`
  - Test cases:
    - `TestNewPlayer_ScaledPosition`: 픽셀 좌표로 생성 시 100x로 저장

**GREEN: Implement to Make Tests Pass**
- [x] **Task 1.3**: Body 구조체 수정
  - File: `internal/domain/entity/body.go`
  - Changes:
    ```go
    const PositionScale = 100

    type Body struct {
        X, Y       int     // 100x scaled position
        VX, VY     float64 // 100x scaled velocity
        // RemX, RemY 제거
        // ... 나머지 동일
    }

    // 헬퍼 함수 추가
    func (b *Body) PixelX() int { return b.X / PositionScale }
    func (b *Body) PixelY() int { return b.Y / PositionScale }
    func (b *Body) SetPixelPos(x, y int) {
        b.X = x * PositionScale
        b.Y = y * PositionScale
    }
    ```

- [x] **Task 1.4**: ApplyVelocity 수정
  - File: `internal/domain/entity/body.go`
  - 이제 remainder 없이 직접 위치에 누적
  ```go
  func (b *Body) ApplyVelocity(dt float64) (dx, dy int) {
      // 100x 스케일에서 직접 계산
      dx = int(b.VX * dt)
      dy = int(b.VY * dt)
      return dx, dy
  }
  ```

- [x] **Task 1.5**: NewPlayer 수정
  - File: `internal/domain/entity/body.go`
  - 픽셀 좌표 입력 → 100x 스케일로 저장

**REFACTOR: Clean Up**
- [x] **Task 1.6**: 기존 테스트 수정
  - 기존 body_test.go 테스트들이 100x 스케일 반영

#### Quality Gate

**Build & Tests**:
- [x] `go build ./...` 성공
- [x] `go test ./internal/domain/entity/...` 통과

**Validation Commands**:
```bash
go build ./...
go test -v ./internal/domain/entity/... -run TestBody
go test -v ./internal/domain/entity/... -run TestPlayer
```

---

### Phase 2: PhysicsSystem Sub-step 통합
**Goal**: PhysicsSystem이 sub-step 기반으로 동작, 슬로우모션 파라미터 수용
**Estimated Time**: 2-3 hours
**Status**: ✅ Complete (2026-01-03)

#### Tasks

**RED: Write Failing Tests First**
- [x] **Test 2.1**: PhysicsSystem sub-step 테스트
  - File: `internal/application/system/physics_test.go`
  - Test cases:
    - `TestPhysicsSystem_SubSteps`: N회 sub-step 호출 시 정확한 이동
    - `TestPhysicsSystem_SlowMotion`: subSteps=1일 때 1/10 속도

- [x] **Test 2.2**: 충돌 판정 100x 스케일 테스트
  - Test cases:
    - `TestPhysicsSystem_CollisionScaled`: 100x 좌표에서 타일 충돌 정상

**GREEN: Implement to Make Tests Pass**
- [x] **Task 2.3**: PhysicsSystem.Update 수정
  - File: `internal/application/system/physics.go`
  - Changes:
    ```go
    func (s *PhysicsSystem) Update(player *entity.Player, dt float64, subSteps int) {
        dtPerStep := dt / 10.0  // 항상 1/10 단위

        for i := 0; i < subSteps; i++ {
            s.updateStep(player, dtPerStep)
        }
    }

    func (s *PhysicsSystem) updateStep(player *entity.Player, dt float64) {
        // 기존 Update 로직을 여기로 이동
        // 단, dt는 1/10 프레임 단위
    }
    ```

- [x] **Task 2.4**: 충돌 판정 100x 스케일 적용
  - File: `internal/application/system/physics.go`
  - checkCollisionX/Y에서 PixelX()/PixelY()로 픽셀 변환 후 타일 체크

- [x] **Task 2.5**: moveX, moveY는 100x 단위씩 이동
  - dx, dy는 100x 단위로 전달됨

**REFACTOR: Clean Up**
- [x] **Task 2.6**: combat.go에서 player.X/Y → PixelX()/PixelY() 변경
  - damagePlayer, updateGolds, checkCollisions 함수 수정

#### Quality Gate

**Build & Tests**:
- [x] `go build ./...` 성공
- [x] `go test ./internal/application/system/...` 통과

**Manual Test**:
- [ ] 플레이어 이동/점프 정상 동작
- [ ] 벽 충돌 정상

**Validation Commands**:
```bash
go build ./...
go test -v ./internal/application/system/... -run TestPhysics
```

---

### Phase 3: main.go 슬로우모션 연동 + 렌더링
**Goal**: 프레임 스킵 제거, sub-step 기반 슬로우모션, 100x 렌더링
**Estimated Time**: 2-3 hours
**Status**: ✅ Complete (2026-01-03)

#### Tasks

**RED: Integration Test 개념 정리**
- [x] **Test 3.1**: 수동 테스트 시나리오 정의
  - 슬로우모션 진입 시 부드러운 감속 확인
  - 슬로우모션 해제 시 정상 속도 복귀

**GREEN: Implement**
- [x] **Task 3.2**: main.go updatePlaying 수정
  - File: `cmd/game/main.go`
  - frameSkipCounter 로직 제거, subSteps 기반 슬로우모션 적용

- [x] **Task 3.3**: 렌더링 100x 스케일 적용
  - File: `cmd/game/main.go`
  - `drawPlayer`, `drawTrajectory` 등에서 PixelX(), PixelY() 사용

- [x] **Task 3.4**: 카메라 좌표 100x 스케일 적용
  - updatePlaying, Draw에서 PixelX(), PixelY() 사용

- [x] **Task 3.5**: Stage spawn 좌표 처리
  - NewPlayer에서 자동 100x 변환 (Phase 1에서 완료)
  - restart()에서 SetPixelPos() 사용

**REFACTOR: Clean Up**
- [x] **Task 3.6**: frameSkipCounter 관련 코드 제거
- [x] **Task 3.7**: PixelX()/PixelY() 헬퍼 일관되게 사용

#### Quality Gate

**Build & Tests**:
- [x] `go build ./...` 성공
- [x] `go test ./...` 전체 통과

**Manual Test**:
- [ ] 게임 실행 정상
- [ ] 플레이어 이동/점프/대시 정상
- [ ] 우클릭 슬로우모션 부드럽게 동작
- [ ] 적과 충돌/데미지 정상
- [ ] 화살 발사 정상

**Validation Commands**:
```bash
go build ./...
go test ./...
make run  # 수동 테스트
```

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| 충돌 판정 오류 | Medium | High | 기존 테스트 유지, 수동 테스트 철저히 |
| 속도 값 불일치 | Medium | Medium | 기존 설정값 100x 스케일 변환 확인 |
| 렌더링 위치 오류 | Low | Medium | PixelX()/PixelY() 일관되게 사용 |
| 성능 저하 | Low | Low | sub-step 10회는 기존과 유사 |

---

## Rollback Strategy

### If Phase 1 Fails
- `git checkout` body.go 원복
- 기존 RemX/RemY 방식 유지

### If Phase 2 Fails
- Phase 1 상태로 복귀
- PhysicsSystem 원복

### If Phase 3 Fails
- Phase 2 상태로 복귀
- main.go 프레임 스킵 방식 복원

---

## Progress Tracking

### Completion Status
- **Phase 1**: 100% ✅
- **Phase 2**: 100% ✅
- **Phase 3**: 100% ✅ (수동 테스트 필요)

**Overall Progress**: 100% complete (수동 테스트 후 최종 완료)

---

## Notes & Learnings

### Implementation Notes

**Phase 1 (2026-01-03)**:
- Body 구조체에서 RemX/RemY 제거 완료
- PositionScale=100 상수 추가
- PixelX(), PixelY(), SetPixelPos() 헬퍼 함수 추가
- NewPlayer가 픽셀 좌표를 100x 스케일로 변환하도록 수정
- physics.go에서 player.RemX=0, player.RemY=0 참조 제거

**Phase 2 (2026-01-03)**:
- PhysicsSystem.Update에 subSteps 파라미터 추가
- updateStep() 분리하여 sub-step 반복 처리
- applyGravity, checkCollision 함수들 100x 스케일 적용
- physics_test.go, combat_test.go 100x 스케일로 수정 완료
- combat.go의 damagePlayer, updateGolds, checkCollisions 100x 적용

**Phase 3 (2026-01-03)**:
- frameSkipCounter 로직 완전 제거
- sub-step 기반 슬로우모션 구현 (arrowSelectUI 활성 시 subSteps=1)
- 모든 렌더링 함수에서 PixelX()/PixelY() 사용
- restart()에서 SetPixelPos() 사용
- checkSpikeDamage에서 VY 100x 스케일 적용

### Key Formulas Reference
```
# 좌표 변환
pixel = scaled / 100
scaled = pixel * 100

# 속도 변환 (설정값이 pixel/sec이면)
scaledVelocity = pixelVelocity * 100

# Sub-step
dtPerStep = dt / 10
totalDt = dtPerStep * subSteps
# 평상시: 10 * (dt/10) = dt (정상 속도)
# 슬로우: 1 * (dt/10) = dt/10 (1/10 속도)
```

---

## Next Steps After Completion

1. **2단계**: Enemy, Projectile, Gold로 100x 스케일 확장
2. **3단계**: 모든 엔티티 통합 sub-step 물리 시스템
3. **최종**: RemX/RemY 완전 제거, 코드 정리

---

**Plan Status**: All Phases Complete ✅
**Next Action**: 수동 테스트 후 최종 확인 (`make run`)
**Blocked By**: None
**Final Notes**: Enemy, Projectile은 아직 RemX/RemY 사용 중 (다음 단계에서 적용 예정)
