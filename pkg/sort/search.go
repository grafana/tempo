package sort

// SearchWithErrors is forked from https://golang.org/src/sort/search.go
// but with added support for errors
func SearchWithErrors(n int, f func(int) (bool, error)) (int, error) {
	// Define f(-1) == false and f(n) == true.
	// Invariant: f(i-1) == false, f(j) == true.
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		b, e := f(h)
		if e != nil {
			return -1, e
		}
		if !b {
			i = h + 1 // preserves f(i-1) == false
		} else {
			j = h // preserves f(j) == true
		}
	}
	// i == j, f(i-1) == false, and f(j) (= f(i)) == true  =>  answer is i.
	return i, nil
}
