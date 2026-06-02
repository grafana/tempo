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
	dedicatedColumnsCache      *otter.Cache[string, DedicatedColumns]
)

func init() {
	dedicatedColumnsCache = otter.Must(&otter.Options[string, DedicatedColumns]{MaximumSize: dedicatedColumnsCacheSize})
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
	s := unsafe.String(unsafe.SliceData(marshalled), len(marshalled)) // unsafe conversion is safe for lookups
	return dedicatedColumnsCache.GetIfPresent(s)
}

func putDedicatedColumnsToCache(marshalled []byte, cols DedicatedColumns) {
	if len(marshalled) == 0 {
		return
	}
	dedicatedColumnsCache.Set(string(marshalled), cols)
}
