package wal

import (
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type AppendBlock interface {
	Append(id common.ID, b []byte, start, end uint32) error
	BlockID() uuid.UUID
	DataLength() uint64
	Length() int
	Meta() *backend.BlockMeta
	Iterator() (common.Iterator, error)
	Find(id common.ID) ([]byte, error)
	Clear() error
}
