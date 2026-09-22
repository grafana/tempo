package backend

import (
	"bytes"
	"unsafe"

	"github.com/maypok86/otter/v2"
)

var (
	dedicatedColumnsCacheNull  = []byte("null")
	dedicatedColumnsCacheEmpty = []byte("[]")
	dedicatedColumnsCacheSize  = 1024
	dedicatedColumnsCache      *otter.Cache[string, DedicatedColumnLayout]
)

func init() {
	dedicatedColumnsCache = otter.Must(&otter.Options[string, DedicatedColumnLayout]{MaximumSize: dedicatedColumnsCacheSize})
}

func getDedicatedColumnsFromCache(marshalled []byte) (DedicatedColumns, bool) {
	if bytes.Equal(marshalled, dedicatedColumnsCacheNull) {
		return nil, true
	}
	if bytes.Equal(marshalled, dedicatedColumnsCacheEmpty) {
		return DedicatedColumns{}, true
	}
	if len(marshalled) == 0 {
		return nil, false
	}
	v, ok := getDedicatedColumnLayoutFromCache(marshalled)
	return v.Columns(), ok
}

func putDedicatedColumnsToCache(marshalled []byte, cols DedicatedColumns) {
	if len(marshalled) == 0 {
		return
	}
	dedicatedColumnsCache.Set(string(marshalled), NewDedicatedColumnLayout(cols))
}

func getDedicatedColumnLayoutFromCache(data []byte) (DedicatedColumnLayout, bool) {
	if len(data) == 0 {
		return DedicatedColumnLayout{}, false
	}
	// The lookup does not retain the input buffer.
	key := unsafe.String(unsafe.SliceData(data), len(data))
	return dedicatedColumnsCache.GetIfPresent(key)
}
