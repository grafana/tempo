package wal

import (
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type AppendBlock interface {
	Append(id common.ID, b []byte, start, end uint32) error
	BlockID() uuid.UUID
	DataLength() uint64
	Length() int
	Meta() *backend.BlockMeta
	Iterator(combiner model.ObjectCombiner) (common.Iterator, error)
	Find(id common.ID, combiner model.ObjectCombiner) ([]byte, error)
	Clear() error

	// jpe - add search
}
