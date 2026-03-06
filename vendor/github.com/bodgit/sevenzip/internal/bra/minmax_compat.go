//go:build !1.21

package bra

//nolint:predeclared
func min(x, y int) int {
	if x < y {
		return x
	}

	return y
}

//nolint:predeclared
func max(x, y int) int {
	if x > y {
		return x
	}

	return y
}
