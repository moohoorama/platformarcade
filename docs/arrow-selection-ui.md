# Arrow Selection UI (화살 변환 인터페이스)

## 개요

마우스 우클릭으로 화살 종류를 선택하는 인터페이스. 우클릭 동안 게임 업데이트가 10프레임당 1번으로 느려지며, 방사형 메뉴에서 상하좌우 방향으로 화살을 선택할 수 있다.

> **의도된 메카닉**: 우클릭을 빠르게 반복하여 슬로우 상태를 유지하면서 이동하는 플레이가 가능함

## 기본 동작

1. **우클릭 누름**: 게임 속도 1/10, UI 나타남 애니메이션 시작
2. **우클릭 유지**: 마우스 위치에 따라 아이콘 하이라이트
3. **우클릭 뗌**: 선택 확정 (또는 취소 시 현재 화살 유지), UI 사라짐 애니메이션 시작
4. **현재 화살 표시**: HP바 주변에 현재 선택된 화살 아이콘 표시

## 화살 종류

4종류의 화살이 존재하며, 최대 4개까지 장착 가능. 동시에 1개만 사용.

| 방향 | 색상 | 설명 |
|------|------|------|
| 오른쪽 | 회색 (Gray) | 기본 화살 |
| 위쪽 | 빨간색 (Red) | - |
| 왼쪽 | 파란색 (Blue) | - |
| 아래쪽 | 보라색 (Purple) | - |

> **장착 시스템**: 여러 종류의 화살을 소유할 수 있으나, 장착은 4개까지. 정비 UI에서 장착할 화살 선택 (미구현, 현재는 4종류 모두 장착된 상태)

## 아이콘 배치

```
        [위]
         |
         | 80px
         |
[왼쪽]---●---[오른쪽]    ● = 우클릭 시작 위치 (centerX, centerY)
         |
         |
         |
        [아래]
```

- **반지름**: 80px
- **최종 위치 (iconDX, iconDY)**:
  - 오른쪽: (80, 0)
  - 위쪽: (0, -80)
  - 왼쪽: (-80, 0)
  - 아래쪽: (0, 80)

## 애니메이션

### 나타남 (30프레임)

1. **화면 어두워짐**: UI 외 모든 전투 표현이 어두워짐
2. **아이콘 이동**: 중앙에서 목표 위치로 sin 곡선 이동 + 투명→불투명

```
progress = frame / maxFrame  (0.0 ~ 1.0)
easedProgress = sin(progress * 90도)

iconX = centerX + iconDX * easedProgress
iconY = centerY + iconDY * easedProgress
alpha = easedProgress
```

### 사라짐 (역방향)

- 나타남 애니메이션의 정확히 역방향
- 화면이 밝아지며, 아이콘이 중앙으로 수렴하며 투명해짐

### 중간 전환

- 나타남 애니메이션 도중 우클릭 떼면 → 즉시 사라짐으로 전환
- 사라짐 애니메이션 도중 우클릭 누르면 → 즉시 나타남으로 전환
- 현재 progress 값에서 반대 방향으로 진행

> **참고**: 사용자는 우클릭을 빠르게 누렀다 떼는 것을 반복하여 게임 속도를 1/10로 유지하면서 이동하는 꼼수가 가능함

## 선택 판정

### 최소 거리 조건

```
abs(deltaX) + abs(deltaY) >= 40
```

- 마름모 모양 (성능상 원 대신 사용)
- 40px 미만이면 선택 없음

### 방향 판정

각도 대신 deltaX, deltaY 비교로 판정 (성능상 이유):

```
오른쪽 (315° ~ 45°):   deltaX > 0  && abs(deltaX) >= abs(deltaY)
위쪽   (45° ~ 135°):   deltaY < 0  && abs(deltaY) > abs(deltaX)
왼쪽   (135° ~ 225°):  deltaX < 0  && abs(deltaX) >= abs(deltaY)
아래쪽 (225° ~ 315°):  deltaY > 0  && abs(deltaY) > abs(deltaX)
```

### 의사 코드

```go
func getSelectedDirection(deltaX, deltaY int) Direction {
    // 최소 거리 체크
    if abs(deltaX) + abs(deltaY) < 40 {
        return None
    }

    // 방향 판정
    if abs(deltaX) >= abs(deltaY) {
        if deltaX > 0 {
            return Right
        }
        return Left
    } else {
        if deltaY < 0 {
            return Up
        }
        return Down
    }
}
```

## 아이콘 색상

### 기본 상태
- **현재 선택된 화살**: 원색 (RGB 100%)
- **선택되지 않은 화살**: RGB 70% (어둡게)

### 하이라이트 상태
마우스 커서가 특정 아이콘을 선택할 수 있는 위치에 있을 때:
- 해당 아이콘이 **약간 크게** 표시
- 해당 아이콘이 **RGB 100%** (밝게)

## 화면 경계 처리

우클릭 시작 위치가 화면 가장자리일 경우, 중심점을 화면 안쪽으로 clamp:

```go
centerX = clamp(mouseX, 80, screenWidth - 80)
centerY = clamp(mouseY, 80, screenHeight - 80)
```

## 상태 다이어그램

```
[Idle] ---(우클릭 누름)---> [Appearing] ---(30프레임 완료)---> [Shown]
   ^                            |                                |
   |                            | (우클릭 뗌)                    | (우클릭 뗌)
   |                            v                                v
   +---(frame=0)--- [Disappearing] <-----------------------------+
                         |
                         | (우클릭 누름)
                         v
                    [Appearing] (현재 frame에서 정방향 재개)
```

### 중간 전환 동작

**핵심**: Appearing ↔ Disappearing 전환 시 현재 frame 값 유지

예시: 5프레임 진행 후 우클릭 뗌
```
Appearing (frame: 0→5) → 우클릭 뗌 → Disappearing (frame: 5→0)
```

- 5프레임 동안 아이콘이 살짝 나타남
- 5프레임 동안 아이콘이 다시 사라짐
- 총 10프레임의 자연스러운 전환

## 구현 참고

### 화살 종류

```go
type ArrowType int

const (
    ArrowGray   ArrowType = iota  // 회색 (기본)
    ArrowRed                      // 빨간색
    ArrowBlue                     // 파란색
    ArrowPurple                   // 보라색
)

var ArrowColors = map[ArrowType]color.RGBA{
    ArrowGray:   {128, 128, 128, 255},
    ArrowRed:    {255, 80, 80, 255},
    ArrowBlue:   {80, 80, 255, 255},
    ArrowPurple: {180, 80, 255, 255},
}
```

### 장착 시스템

```go
type Player struct {
    // ...
    OwnedArrows   []ArrowType    // 소유한 화살 (여러 종류 가능)
    EquippedArrows [4]ArrowType  // 장착된 화살 (최대 4개, 방향별)
    CurrentArrow  ArrowType      // 현재 사용 중인 화살
}

// 방향별 장착 위치
// EquippedArrows[0] = 오른쪽
// EquippedArrows[1] = 위쪽
// EquippedArrows[2] = 왼쪽
// EquippedArrows[3] = 아래쪽
```

### UI 상태

```go
type ArrowSelectState int

const (
    ArrowSelectIdle ArrowSelectState = iota
    ArrowSelectAppearing
    ArrowSelectShown
    ArrowSelectDisappearing
)

type ArrowSelectUI struct {
    State        ArrowSelectState
    Frame        int           // 현재 애니메이션 프레임 (0~30)
    MaxFrame     int           // 30
    CenterX      int           // 우클릭 시작 위치 (clamped)
    CenterY      int
    Highlighted  Direction     // 현재 하이라이트된 방향 (-1 = 없음)
}
```

### 상태 전환 로직

```go
func (ui *ArrowSelectUI) Update(rightClickPressed, rightClickReleased bool) {
    switch ui.State {
    case ArrowSelectIdle:
        if rightClickPressed {
            ui.State = ArrowSelectAppearing
            ui.Frame = 0
            // CenterX, CenterY 설정 (clamped)
        }

    case ArrowSelectAppearing:
        if rightClickReleased {
            // 바로 Disappearing으로 전환, frame 유지
            ui.State = ArrowSelectDisappearing
        } else {
            ui.Frame++
            if ui.Frame >= ui.MaxFrame {
                ui.State = ArrowSelectShown
            }
        }

    case ArrowSelectShown:
        if rightClickReleased {
            ui.State = ArrowSelectDisappearing
            ui.Frame = ui.MaxFrame
        }

    case ArrowSelectDisappearing:
        if rightClickPressed {
            // 바로 Appearing으로 전환, frame 유지
            ui.State = ArrowSelectAppearing
        } else {
            ui.Frame--
            if ui.Frame <= 0 {
                ui.State = ArrowSelectIdle
            }
        }
    }
}
```

### 게임 속도 조절

```go
var frameSkipCounter int

func (g *Game) Update() {
    if g.arrowSelectUI.State != ArrowSelectIdle {
        frameSkipCounter++
        if frameSkipCounter < 10 {
            // UI 애니메이션만 업데이트
            g.arrowSelectUI.Update(...)
            return
        }
        frameSkipCounter = 0
    }

    // 게임 로직 업데이트 (10프레임당 1번)
    // ...
}
```

### 렌더링 순서

1. 배경/타일 (어두움 효과 적용)
2. 적/플레이어/화살 (어두움 효과 적용)
3. **어두움 오버레이** (State != Idle일 때)
4. **화살 선택 아이콘** (State != Idle일 때)
5. UI (HP바, 현재 화살 아이콘 등) - 어두움 효과 미적용
