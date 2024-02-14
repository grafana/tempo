package traceql

import (
	"time"
)

const (
	HintSample = "sample"
	HintDedupe = "dedupe"
)

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

func (h *Hints) GetFloat(k string) (bool, float64) {
	return get(h, k, TypeFloat, func(s Static) float64 { return s.F })
}

func (h *Hints) GetInt(k string) (bool, int) {
	return get(h, k, TypeInt, func(s Static) int { return s.N })
}

func (h *Hints) GetDuration(k string) (bool, time.Duration) {
	return get(h, k, TypeDuration, func(s Static) time.Duration { return s.D })
}

func (h *Hints) GetBool(k string) (bool, bool) {
	return get(h, k, TypeBoolean, func(s Static) bool { return s.B })
}

func get[T any](h *Hints, k string, t StaticType, f func(Static) T) (ok bool, value T) {
	if h == nil {
		return
	}

	for _, hh := range h.Hints {
		if hh.Name == k && hh.Value.Type == t {
			return true, f(hh.Value)
		}
	}

	return
}
