package traceql

import (
	"time"
)

// The list of all traceql query hints.  Although most of these are implementation-specific
// and not part of the language or engine, we organize them here in one place.
const (
	HintSample            = "sample"
	HintDedupe            = "dedupe"
	HintJobInterval       = "job_interval"
	HintJobSize           = "job_size"
	HintTimeOverlapCutoff = "time_overlap_cutoff"
	HintConcurrentBlocks  = "concurrent_blocks"
)

func isUnsafe(h string) bool {
	switch h {
	case HintSample:
		return false
	default:
		return true
	}
}

type Hint struct {
	Name  string
	Value Static
}

func newHint(k string, v Static) *Hint {
	return &Hint{k, v}
}

type Hints struct {
	Hints []*Hint
}

func newHints(h []*Hint) *Hints {
	return &Hints{h}
}

func (h *Hints) GetFloat(k string, allowUnsafe bool) (float64, bool) {
	if v, ok := h.Get(k, TypeFloat, allowUnsafe); ok {
		f := v.(StaticFloat)
		return f.Float, ok
	}

	// If float not found, then try integer.
	if v, ok := h.Get(k, TypeInt, allowUnsafe); ok {
		return v.asFloat(), ok
	}

	return 0, false
}

func (h *Hints) GetInt(k string, allowUnsafe bool) (int, bool) {
	if v, ok := h.Get(k, TypeInt, allowUnsafe); ok {
		n := v.(StaticInt)
		return n.Int, ok
	}

	return 0, false
}

func (h *Hints) GetDuration(k string, allowUnsafe bool) (time.Duration, bool) {
	if v, ok := h.Get(k, TypeDuration, allowUnsafe); ok {
		d := v.(StaticDuration)
		return d.Duration, ok
	}

	return 0, false
}

func (h *Hints) GetBool(k string, allowUnsafe bool) (bool, bool) {
	if v, ok := h.Get(k, TypeBoolean, allowUnsafe); ok {
		b := v.(StaticBool)
		return b.Bool, ok
	}

	return false, false
}

func (h *Hints) Get(k string, t StaticType, allowUnsafe bool) (v Static, ok bool) {
	if h == nil {
		return
	}

	if isUnsafe(k) && !allowUnsafe {
		return
	}

	for _, hh := range h.Hints {
		if hh.Name == k && hh.Value.Type() == t {
			return hh.Value, true
		}
	}

	return
}

var _ Element = (*Hints)(nil)
