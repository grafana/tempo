package traceql

const HintSample = "sample"

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
	if h == nil {
		return false, 0
	}

	for _, hh := range h.Hints {
		if hh.Name == k && hh.Value.Type == TypeFloat {
			return true, hh.Value.F
		}
	}

	return false, 0
}
