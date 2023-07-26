package math

// Max returns the maximum of two ints
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min returns the minimum of two ints
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max64 returns the maximum uint64s
func Max64(values ...uint64) uint64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}

	x := values[0]
	for i := 1; i < len(values); i++ {
		if values[i] > x {
			x = values[i]
		}
	}
	return x
}

// Min64 returns the minimum of uint64s
func Min64(values ...uint64) uint64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}

	x := values[0]
	for i := 1; i < len(values); i++ {
		if values[i] < x {
			x = values[i]
		}
	}
	return x
}
