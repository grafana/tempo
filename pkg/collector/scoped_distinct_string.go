package collector

type ScopedDistinctString struct {
	cols     map[string]*DistinctString
	maxLen   int
	curLen   int
	exceeded bool
}

func NewScopedDistinctString(sz int) *ScopedDistinctString {
	return &ScopedDistinctString{
		cols:   map[string]*DistinctString{},
		maxLen: sz,
	}
}

func (d *ScopedDistinctString) Collect(scope string, val string) {
	// can it fit?
	if d.maxLen > 0 && d.curLen+len(val) > d.maxLen {
		d.exceeded = true
		// No
		return
	}

	// get or create collector
	col, ok := d.cols[scope]
	if !ok {
		col = NewDistinctString(0)
		d.cols[scope] = col
	}

	added := col.Collect(val)
	if added {
		d.curLen += len(val)
	}
}

// Strings returns the final list of distinct values collected and sorted.
func (d *ScopedDistinctString) Strings() map[string][]string {
	ss := map[string][]string{}

	for k, v := range d.cols {
		ss[k] = v.Strings()
	}

	return ss
}

// Exceeded indicates if some values were lost because the maximum size limit was met.
func (d *ScopedDistinctString) Exceeded() bool {
	return d.exceeded
}

// Diff returns all new strings collected since the last time diff was called
func (d *ScopedDistinctString) Diff() map[string][]string {
	ss := map[string][]string{}

	for k, v := range d.cols {
		diff := v.Diff()
		if len(diff) > 0 {
			ss[k] = diff
		}
	}

	return ss
}
