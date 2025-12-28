package main

import "testing"

const N2 = 100_000

// AoS - 큰 구조체 (256 bytes)
type EntityBigAoS struct {
	X, Y, Z    float64
	VX, VY, VZ float64
	Padding    [26]float64 // 캐시라인 여러개 차지하도록
	Health     int
	Type       int
}

var bigAoS [N2]EntityBigAoS

// SoA
type BigSoA struct {
	X, Y, Z    [N2]float64
	VX, VY, VZ [N2]float64
	Padding    [26][N2]float64
	Health     [N2]int
	Type       [N2]int
}

var bigSoA BigSoA

func init() {
	for i := 0; i < N2; i++ {
		bigAoS[i] = EntityBigAoS{X: float64(i), VX: 1.0, Health: 100, Type: i % 4}
		bigSoA.X[i] = float64(i)
		bigSoA.VX[i] = 1.0
		bigSoA.Health[i] = 100
		bigSoA.Type[i] = i % 4
	}
}

// Case 1: X 컬럼만 합산 (SoA 유리해야 함)
func BenchmarkBigSingleCol_AoS(b *testing.B) {
	var sum float64
	for n := 0; n < b.N; n++ {
		sum = 0
		for i := 0; i < N2; i++ {
			sum += bigAoS[i].X
		}
	}
	_ = sum
}

func BenchmarkBigSingleCol_SoA(b *testing.B) {
	var sum float64
	for n := 0; n < b.N; n++ {
		sum = 0
		for i := 0; i < N2; i++ {
			sum += bigSoA.X[i]
		}
	}
	_ = sum
}

// Case 2: X += VX (두 컬럼)
func BenchmarkBigTwoCol_AoS(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for i := 0; i < N2; i++ {
			bigAoS[i].X += bigAoS[i].VX
		}
	}
}

func BenchmarkBigTwoCol_SoA(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for i := 0; i < N2; i++ {
			bigSoA.X[i] += bigSoA.VX[i]
		}
	}
}

// Case 3: Type 필터 후 Health 합
func BenchmarkBigFilter_AoS(b *testing.B) {
	var sum int
	for n := 0; n < b.N; n++ {
		sum = 0
		for i := 0; i < N2; i++ {
			if bigAoS[i].Type == 1 {
				sum += bigAoS[i].Health
			}
		}
	}
	_ = sum
}

func BenchmarkBigFilter_SoA(b *testing.B) {
	var sum int
	for n := 0; n < b.N; n++ {
		sum = 0
		for i := 0; i < N2; i++ {
			if bigSoA.Type[i] == 1 {
				sum += bigSoA.Health[i]
			}
		}
	}
	_ = sum
}
