package backend

import (
	"bytes"
	"unsafe"

	lru "github.com/hashicorp/golang-lru/v2"
)

var (
	dedicatedColumnsCacheNull  = []byte("null")
	dedicatedColumnsCacheEmpty = []byte("[]")
	dedicatedColumnsCacheSize  = 1024
	dedicatedColumnsCache      *lru.Cache[string, DedicatedColumns]
)

func init() {
	var err error
	dedicatedColumnsCache, err = lru.New[string, DedicatedColumns](dedicatedColumnsCacheSize)
	if err != nil { // only errors if dedicatedColumnsCacheSize <= 0
		panic(err)
	}
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
	return dedicatedColumnsCache.Get(s)
}

func putDedicatedColumnsToCache(marshalled []byte, cols DedicatedColumns) {
	if len(marshalled) == 0 {
		return
	}
	_ = dedicatedColumnsCache.Add(string(marshalled), cols)
}
