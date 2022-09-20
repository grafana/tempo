package common

import (
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

// extracts a time range from an object. start/end times returned are unix epoch
// seconds
type RangeFunc func(obj []byte, dataEncoding string) (uint32, uint32, error)

type AppendBlock interface { // jpe cooler name like Appender
	// jpe add common.Finder and common.Searcher

	Append(id ID, b []byte, start, end uint32) error
	BlockID() uuid.UUID
	DataLength() uint64
	Length() int
	Meta() *backend.BlockMeta
	Iterator() (Iterator, error)
	Find(id ID) ([]byte, error)
	Clear() error
}
