package backend

import "sync"

var (
	dedicatedColumnsKeeper = map[string]*DedicatedColumns{}
	dedicatedColumnsMtx    = sync.Mutex{}
)

func getDedicatedColumns(b string) *DedicatedColumns {
	dedicatedColumnsMtx.Lock()
	defer dedicatedColumnsMtx.Unlock()

	if v, ok := dedicatedColumnsKeeper[b]; ok {
		return v
	}
	return nil
}

func putDedicatedColumns(b string, d *DedicatedColumns) {
	dedicatedColumnsMtx.Lock()
	defer dedicatedColumnsMtx.Unlock()

	dedicatedColumnsKeeper[b] = d
}

func ClearDedicatedColumns() {
	dedicatedColumnsMtx.Lock()
	defer dedicatedColumnsMtx.Unlock()

	clear(dedicatedColumnsKeeper)
}
