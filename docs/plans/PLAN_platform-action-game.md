# Platform Action Game Implementation Plan

**Status**: In Progress
**Created**: 2024-12-28
**Last Updated**: 2024-12-28

---

**CRITICAL INSTRUCTIONS**: After completing each phase:
1. Check off completed task checkboxes
2. Run all quality gate validation commands
3. Verify ALL quality gate items pass
4. Update "Last Updated" date
5. Document learnings in Notes section
6. Only then proceed to next phase

DO NOT skip quality gates or proceed with failing checks

---

## Overview

ebiten-go 기반 2D 플랫폼 액션 게임 구현.
- Intent & Apply 물리 시스템
- 평행사변형 히트박스 (head/body/feet)
- 화살 포물선 궤적 (20도 발사, 중력 가속도)
- 게임 필 (코요테 타임, 점프 버퍼, 대쉬 등)
- JSON 기반 설정
- 네이티브 바이너리 + WebAssembly (GitHub Pages)

---

## Phase 1: 프로젝트 기반 + JSON 로더

**Goal**: ebiten 게임 루프 동작, JSON 설정 로딩, 타일맵 렌더링

### Tasks

- [x] Go 모듈 초기화 (`go mod init`)
- [x] 디렉토리 구조 생성
- [x] JSON 설정 구조체 정의 (physics, entities, stages)
- [x] JSON 로더 구현
- [x] ebiten 기본 게임 루프
- [x] 타일맵 파싱 및 색상 사각형 렌더링
- [x] 플레이어 스폰 위치 표시

### Quality Gate

- [x] `go build ./...` 성공
- [x] `go test ./...` 통과
- [x] 게임 실행 시 타일맵 표시
- [x] 플레이어 위치에 사각형 표시

---

## Phase 2: 물리 시스템 + 플레이어 이동

**Goal**: 플레이어가 좌우 이동하고 벽에 충돌

### Tasks

- [ ] Body 구조체 (정수 위치 + 서브픽셀)
- [ ] Intent 타입 정의
- [ ] PhysicsSystem (Intent 수집 + Apply)
- [ ] Substep 이동 (1픽셀 단위)
- [ ] 평행사변형 히트박스 구현
- [ ] 타일맵 충돌 감지
- [ ] 중력 적용 + 바닥 착지

### Quality Gate

- [ ] 좌우 이동 동작
- [ ] 벽에서 멈춤
- [ ] 바닥에 착지
- [ ] 빠른 이동에도 벽 통과 없음

---

## Phase 3: 플레이어 액션 + 게임 필

**Goal**: 점프, 대쉬, 게임 필 요소 동작

### Tasks

- [ ] 점프 구현 (Z키)
- [ ] 가변 점프 높이
- [ ] 코요테 타임
- [ ] 점프 버퍼
- [ ] 에이펙스 모디파이어
- [ ] 하강 가속
- [ ] 대쉬 구현 (C키)
- [ ] 코너 보정
- [ ] 레지 어시스트

### Quality Gate

- [ ] 점프 높이 조절 가능
- [ ] 절벽에서 떨어진 직후 점프 가능
- [ ] 착지 직전 점프 입력 인식
- [ ] 대쉬 동작

---

## Phase 4: 전투 시스템

**Goal**: 화살 발사, 적 AI, 데미지, 골드

### Tasks

- [ ] 화살 발사 (X키)
- [ ] 화살 포물선 (20도 발사, 중력 가속도)
- [ ] 화살 스프라이트 회전
- [ ] 적 엔티티 생성
- [ ] Slime AI (patrol)
- [ ] Archer AI (ranged)
- [ ] Bat AI (chase)
- [ ] 데미지 시스템
- [ ] 플레이어 i-frames + 넉백
- [ ] 골드 드롭 + 수집

### Quality Gate

- [ ] 화살이 포물선으로 날아감
- [ ] 적이 패턴대로 움직임
- [ ] 적에게 데미지
- [ ] 플레이어 피격 시 무적

---

## Phase 5: 피드백 + 카메라

**Goal**: 게임 피드백, 카메라 시스템

### Tasks

- [ ] 히트스탑 (프레임 정지)
- [ ] 스크린 쉐이크
- [ ] 스쿼시 & 스트레치
- [ ] 카메라 플레이어 추적
- [ ] 카메라 경계 제한

### Quality Gate

- [ ] 타격 시 일시 정지 느낌
- [ ] 피격 시 화면 흔들림
- [ ] 점프/착지 시 스프라이트 변형
- [ ] 카메라가 플레이어 따라감

---

## Phase 6: UI + 게임 상태

**Goal**: UI 표시, 메뉴/일시정지

### Tasks

- [ ] 체력바 UI
- [ ] 골드 표시 UI
- [ ] State Stack 구현
- [ ] 메뉴 화면
- [ ] 일시정지 (ESC)
- [ ] 게임오버 화면
- [ ] 재시작

### Quality Gate

- [ ] 체력/골드 표시
- [ ] ESC로 일시정지
- [ ] 사망 시 게임오버
- [ ] 재시작 동작

---

## Phase 7: 빌드 + 배포

**Goal**: 네이티브/WASM 빌드, GitHub Pages 배포

### Tasks

- [ ] Makefile 작성
- [ ] 네이티브 빌드 테스트
- [ ] WebAssembly 빌드
- [ ] index.html 작성
- [ ] wasm_exec.js 복사
- [ ] GitHub Pages 설정
- [ ] README 업데이트

### Quality Gate

- [ ] `make build` 성공
- [ ] `make wasm` 성공
- [ ] 로컬 웹서버에서 WASM 실행
- [ ] GitHub Pages에서 플레이 가능

---

## Notes & Learnings

(구현 중 발견한 내용 기록)

---

## Rollback Strategy

각 Phase는 독립적으로 롤백 가능:
- Git commit을 Phase 단위로 생성
- 문제 발생 시 해당 Phase commit으로 reset
