package util

// Clone returns a copy of the slice.
// The elements are copied using assignment, so this is a shallow clone.
// The difference with slices.Clone is that the result will not have additional unused capacity.
func Clone[S ~[]E, E any](s S) S {
	// Preserve nilness in case it matters.
	if s == nil {
		return nil
	}
	return append(make(S, 0, len(s)), s...)
}
