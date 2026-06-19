package livestore

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBlockSnapshot_EmptyHasInitializedMaps(t *testing.T) {
	s := emptyBlockSnapshot()
	assert.NotNil(t, s.walBlocks)
	assert.NotNil(t, s.completeBlocks)
	assert.Empty(t, s.walBlocks)
	assert.Empty(t, s.completeBlocks)
	assert.Nil(t, s.headBlock)
}

func TestBlockSnapshot_WithWALBlockAddedRemovedDoesNotMutateOriginal(t *testing.T) {
	s := emptyBlockSnapshot()
	id := uuid.New()

	s2 := s.withWALBlockAdded(id, nil)
	assert.Empty(t, s.walBlocks, "original snapshot unchanged after add")
	assert.Contains(t, s2.walBlocks, id)

	s3 := s2.withWALBlockRemoved(id)
	assert.Contains(t, s2.walBlocks, id, "second snapshot unchanged after remove")
	assert.Empty(t, s3.walBlocks)
}

func TestBlockSnapshot_WithCompleteBlockAddedRemovedDoesNotMutateOriginal(t *testing.T) {
	s := emptyBlockSnapshot()
	id := uuid.New()

	s2 := s.withCompleteBlockAdded(id, nil)
	assert.Empty(t, s.completeBlocks, "original snapshot unchanged after add")
	assert.Contains(t, s2.completeBlocks, id)

	s3 := s2.withCompleteBlockRemoved(id)
	assert.Contains(t, s2.completeBlocks, id, "second snapshot unchanged after remove")
	assert.Empty(t, s3.completeBlocks)
}

func TestBlockSnapshot_WithHeadBlockReplaces(t *testing.T) {
	s := emptyBlockSnapshot()
	s2 := s.withHeadBlock(nil) // identity check via map equality
	assert.Nil(t, s.headBlock)
	assert.Nil(t, s2.headBlock)

	// Setting head block does not affect maps in original.
	id := uuid.New()
	s3 := s.withWALBlockAdded(id, nil).withHeadBlock(nil)
	assert.Empty(t, s.walBlocks)
	assert.Contains(t, s3.walBlocks, id)
}
