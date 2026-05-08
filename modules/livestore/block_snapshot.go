package livestore

import (
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

func (s *blockSnapshot) clone() *blockSnapshot {
	out := &blockSnapshot{
		headBlock:      s.headBlock,
		walBlocks:      make(map[uuid.UUID]common.WALBlock, len(s.walBlocks)),
		completeBlocks: make(map[uuid.UUID]*LocalBlock, len(s.completeBlocks)),
	}
	for k, v := range s.walBlocks {
		out.walBlocks[k] = v
	}
	for k, v := range s.completeBlocks {
		out.completeBlocks[k] = v
	}
	return out
}

func (s *blockSnapshot) withHeadBlock(b common.WALBlock) *blockSnapshot {
	out := s.clone()
	out.headBlock = b
	return out
}

func (s *blockSnapshot) withWALBlockAdded(id uuid.UUID, b common.WALBlock) *blockSnapshot {
	out := s.clone()
	out.walBlocks[id] = b
	return out
}

func (s *blockSnapshot) withWALBlockRemoved(id uuid.UUID) *blockSnapshot {
	out := s.clone()
	delete(out.walBlocks, id)
	return out
}

func (s *blockSnapshot) withCompleteBlockAdded(id uuid.UUID, b *LocalBlock) *blockSnapshot {
	out := s.clone()
	out.completeBlocks[id] = b
	return out
}

func (s *blockSnapshot) withCompleteBlockRemoved(id uuid.UUID) *blockSnapshot {
	out := s.clone()
	delete(out.completeBlocks, id)
	return out
}
