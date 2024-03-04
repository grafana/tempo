package traceql

import (
	"time"
)

const (
	HintSample            = "sample"
	HintDedupe            = "dedupe"
	HintJobInterval       = "job_interval"
	HintJobSize           = "job_size"
	HintTimeOverlapCutoff = "time_overlap_cutoff"
	HintBlockConcurrency  = "block_concurrency"
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

func (h *Hints) GetFloat(k string, allowUnsafe bool) (ok bool, v float64) {
	if ok, v := h.Get(k, TypeFloat, allowUnsafe); ok {
		return ok, v.F
	}

	// If float not found, then try integer.
	if ok, v := h.Get(k, TypeInt, allowUnsafe); ok {
		return ok, float64(v.N)
	}

	return
}

func (h *Hints) GetInt(k string, allowUnsafe bool) (ok bool, v int) {
	if ok, v := h.Get(k, TypeInt, allowUnsafe); ok {
		return ok, v.N
	}

	return
}

func (h *Hints) GetDuration(k string, allowUnsafe bool) (ok bool, v time.Duration) {
	if ok, v := h.Get(k, TypeDuration, allowUnsafe); ok {
		return ok, v.D
	}

	return
}

func (h *Hints) GetBool(k string, allowUnsafe bool) (ok bool, v bool) {
	if ok, v := h.Get(k, TypeBoolean, allowUnsafe); ok {
		return ok, v.B
	}

	return
}

func (h *Hints) Get(k string, t StaticType, allowUnsafe bool) (ok bool, v Static) {
	if h == nil {
		return
	}

	if isUnsafe(k) && !allowUnsafe {
		return
	}

	for _, hh := range h.Hints {
		if hh.Name == k && hh.Value.Type == t {
			return true, hh.Value
		}
	}

	return
}
