package intern

import "sync"

// A Value pointer is the handle to an underlying comparable value.
type Value struct {
	cmpVal string
}

// Get returns the comparable value passed to the Get func that returned v.
func (v *Value) Get() string { return v.cmpVal }

// Our pool of interned values and a lock to serialize access.
var (
	mu  sync.Mutex
	val = map[string]*Value{}
)

// Get returns a pointer representing the comparable value cmpVal.
//
// The returned pointer will be the same for Get(v) and Get(v2)
// if and only if v == v2, and can be used as a map key.
//
// Note that Get returns a *Value so we only return one word of data
// to the caller, despite potentially storing a large amount of data
// within the Value itself.
func Get(cmpVal string) *Value {
	mu.Lock()
	defer mu.Unlock()

	v := val[cmpVal]
	if v != nil {
		// Value is already interned in the pool.
		return v
	}

	// Value must be created now and then interned.
	v = &Value{cmpVal: cmpVal}
	val[cmpVal] = v
	return v
}
