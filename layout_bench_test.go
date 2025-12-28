package main

import "testing"

const N = 100_000

// AoS (Array of Structures) - Row Oriented
type EntityAoS struct {
	X, Y, Z    float64
	VX, VY, VZ float64
	Health     int
	Type       int
}

var entitiesAoS [N]EntityAoS

// SoA (Structure of Arrays) - Column Oriented
type EntitiesSoA struct {
	X, Y, Z    [N]float64
	VX, VY, VZ [N]float64
	Health     [N]int
	Type       [N]int
}

var entitiesSoA EntitiesSoA

func init() {
	for i := 0; i < N; i++ {
		entitiesAoS[i] = EntityAoS{
			X: float64(i), Y: float64(i), Z: float64(i),
			VX: 1.0, VY: 1.0, VZ: 1.0,
			Health: 100, Type: i % 4,
		}
		entitiesSoA.X[i] = float64(i)
		entitiesSoA.Y[i] = float64(i)
		entitiesSoA.Z[i] = float64(i)
		entitiesSoA.VX[i] = 1.0
		entitiesSoA.VY[i] = 1.0
		entitiesSoA.VZ[i] = 1.0
		entitiesSoA.Health[i] = 100
		entitiesSoA.Type[i] = i % 4
	}
}

// Case 1: 단일 컬럼 접근 (SoA 유리)
// 예: 모든 엔티티의 X 좌표 합계

func BenchmarkSingleColumn_AoS(b *testing.B) {
	var sum float64
	for n := 0; n < b.N; n++ {
		sum = 0
		for i := 0; i < N; i++ {
			sum += entitiesAoS[i].X
		}
	}
	_ = sum
}

func BenchmarkSingleColumn_SoA(b *testing.B) {
	var sum float64
	for n := 0; n < b.N; n++ {
		sum = 0
		for i := 0; i < N; i++ {
			sum += entitiesSoA.X[i]
		}
	}
	_ = sum
}

// Case 2: 다중 컬럼 접근 (차이 줄어듦)
// 예: position += velocity (X, Y, Z, VX, VY, VZ 모두 접근)

func BenchmarkMultiColumn_AoS(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for i := 0; i < N; i++ {
			entitiesAoS[i].X += entitiesAoS[i].VX
			entitiesAoS[i].Y += entitiesAoS[i].VY
			entitiesAoS[i].Z += entitiesAoS[i].VZ
		}
	}
}

func BenchmarkMultiColumn_SoA(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for i := 0; i < N; i++ {
			entitiesSoA.X[i] += entitiesSoA.VX[i]
			entitiesSoA.Y[i] += entitiesSoA.VY[i]
			entitiesSoA.Z[i] += entitiesSoA.VZ[i]
		}
	}
}

// Case 3: 조건부 필터링 (SoA 유리)
// 예: Type == 1인 엔티티만 Health 합계

func BenchmarkFilter_AoS(b *testing.B) {
	var sum int
	for n := 0; n < b.N; n++ {
		sum = 0
		for i := 0; i < N; i++ {
			if entitiesAoS[i].Type == 1 {
				sum += entitiesAoS[i].Health
			}
		}
	}
	_ = sum
}

func BenchmarkFilter_SoA(b *testing.B) {
	var sum int
	for n := 0; n < b.N; n++ {
		sum = 0
		for i := 0; i < N; i++ {
			if entitiesSoA.Type[i] == 1 {
				sum += entitiesSoA.Health[i]
			}
		}
	}
	_ = sum
}
