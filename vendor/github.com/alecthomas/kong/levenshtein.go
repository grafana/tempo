package kong

import "unicode/utf8"

// Copied from https://github.com/daviddengcn/go-algs/blob/fe23fabd9d0670e4675326040ba7c285c7117b4c/ed/ed.go#L31
// License: https://github.com/daviddengcn/go-algs/blob/fe23fabd9d0670e4675326040ba7c285c7117b4c/LICENSE
func levenshtein(a, b string) int {
	f := make([]int, utf8.RuneCountInString(b)+1)

	for j := range f {
		f[j] = j
	}

	for _, ca := range a {
		j := 1
		fj1 := f[0] // fj1 is the value of f[j - 1] in last iteration
		f[0]++
		for _, cb := range b {
			mn := min(f[j]+1, f[j-1]+1) // delete & insert
			if cb != ca {
				mn = min(mn, fj1+1) // change
			} else {
				mn = min(mn, fj1) // matched
			}

			fj1, f[j] = f[j], mn // save f[j] to fj1(j is about to increase), update f[j] to mn
			j++
		}
	}

	return f[len(f)-1]
}

func min(a, b int) int { //nolint:predeclared
	if a <= b {
		return a
	}
	return b
}
