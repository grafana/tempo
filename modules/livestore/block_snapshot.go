package livestore

import (
	"maps"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// blockSnapshot is an immutable view of an instance's blocks. Readers obtain
// a snapshot via atomic.Load; writers build a new one under blocksMtx and
// atomic.Store it.
type blockSnapshot struct {
	headBlock      common.WALBlock
	walBlocks      map[uuid.UUID]common.WALBlock
	completeBlocks map[uuid.UUID]*LocalBlock
}

func emptyBlockSnapshot() *blockSnapshot {
	return &blockSnapshot{
		walBlocks:      map[uuid.UUID]common.WALBlock{},
		completeBlocks: map[uuid.UUID]*LocalBlock{},
	}
}

func (s *blockSnapshot) withHeadBlock(b common.WALBlock) *blockSnapshot {
	return &blockSnapshot{
		headBlock:      b,
		walBlocks:      s.walBlocks,
		completeBlocks: s.completeBlocks,
	}
}

func (s *blockSnapshot) withWALBlockAdded(id uuid.UUID, b common.WALBlock) *blockSnapshot {
	w := maps.Clone(s.walBlocks)
	w[id] = b
	return &blockSnapshot{
		headBlock:      s.headBlock,
		walBlocks:      w,
		completeBlocks: s.completeBlocks,
	}
}

func (s *blockSnapshot) withWALBlockRemoved(id uuid.UUID) *blockSnapshot {
	w := maps.Clone(s.walBlocks)
	delete(w, id)
	return &blockSnapshot{
		headBlock:      s.headBlock,
		walBlocks:      w,
		completeBlocks: s.completeBlocks,
	}
}

func (s *blockSnapshot) withCompleteBlockAdded(id uuid.UUID, b *LocalBlock) *blockSnapshot {
	c := maps.Clone(s.completeBlocks)
	c[id] = b
	return &blockSnapshot{
		headBlock:      s.headBlock,
		walBlocks:      s.walBlocks,
		completeBlocks: c,
	}
}

func (s *blockSnapshot) withCompleteBlockRemoved(id uuid.UUID) *blockSnapshot {
	c := maps.Clone(s.completeBlocks)
	delete(c, id)
	return &blockSnapshot{
		headBlock:      s.headBlock,
		walBlocks:      s.walBlocks,
		completeBlocks: c,
	}
}
