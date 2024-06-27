package traceql

const (
	initialMapSize = 8

	// limits for interned values
	maxMapSize      = 1024 * 10
	maxInternStrLen = 128
	maxGlobalInt    = 1024

	// constants for globally interned values
	globalIntCount    = maxGlobalInt * 2
	globalStatusCount = 3
	globalKindCount   = 9
)

var (
	globalInts   [globalIntCount]Static
	globalStatus [globalStatusCount]Static
	globalKind   [globalKindCount]Static // 9 is the number of kinds
	globalBool   = struct {
		true  Static
		false Static
	}{true: NewStaticBool(true), false: NewStaticBool(false)}
)

func init() {
	for i := 0; i < len(globalInts); i++ {
		globalInts[i] = NewStaticInt(i - maxGlobalInt)
	}
	for i := 0; i < len(globalStatus); i++ {
		globalStatus[i] = NewStaticStatus(Status(i))
	}
	for i := 0; i < len(globalKind); i++ {
		globalKind[i] = NewStaticKind(Kind(i))
	}
}

// StaticInterner helps to intern static values. This can help to reduce allocations
// in cases where a large number of static values are created and passed as interface
// type traceql.Static.
type StaticInterner struct {
	strings map[string]Static
	ints    map[int]Static
	floats  map[float64]Static
}

func NewStaticInterner() *StaticInterner {
	return &StaticInterner{
		strings: make(map[string]Static, initialMapSize),
		ints:    make(map[int]Static, initialMapSize),
		floats:  make(map[float64]Static, initialMapSize),
	}
}

func (si *StaticInterner) StaticString(s string) Static {
	if len(s) > maxInternStrLen {
		return NewStaticString(s)
	}

	ss, ok := si.strings[s]
	if !ok {
		ss = NewStaticString(s)
		if len(si.strings) <= maxMapSize {
			si.strings[s] = ss
		}
	}
	return ss
}

func (si *StaticInterner) StaticInt(i int) Static {
	if i < maxGlobalInt && i >= -maxGlobalInt {
		return globalInts[i+maxGlobalInt]
	}

	ii, ok := si.ints[i]
	if !ok {
		ii = NewStaticInt(i)
		if len(si.ints) <= maxMapSize {
			si.ints[i] = ii
		}
	}
	return ii
}

func (si *StaticInterner) StaticFloat(f float64) Static {
	ff, ok := si.floats[f]
	if !ok {
		ff = NewStaticFloat(f)
		if len(si.floats) <= maxMapSize {
			si.floats[f] = ff
		}
	}
	return ff
}

func (si *StaticInterner) StaticStatus(s Status) Static {
	return globalStatus[s]
}

func (si *StaticInterner) StaticKind(k Kind) Static {
	return globalKind[k]
}

func (si *StaticInterner) StaticBool(b bool) Static {
	if b {
		return globalBool.true
	}
	return globalBool.false
}
