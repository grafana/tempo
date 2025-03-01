package traceql

import "math"

func sumOverTime() func(curr float64, n float64) (res float64) {
	var comp float64 // Kahan compensation
	return func(curr, n float64) (res float64) {
		if math.IsNaN(curr) {
			return n
		}
		sum, c := kahanSumInc(n, curr, comp)
		comp = c
		if math.IsInf(sum, 0) {
			return sum
		}
		return sum + c
	}
}

func minOverTime() func(curr float64, n float64) (res float64) {
	return func(curr, n float64) (res float64) {
		if math.IsNaN(curr) || n < curr {
			return n
		}
		return curr
	}
}

func maxOverTime() func(curr float64, n float64) (res float64) {
	return func(curr, n float64) (res float64) {
		if math.IsNaN(curr) || n > curr {
			return n
		}
		return curr
	}
}
