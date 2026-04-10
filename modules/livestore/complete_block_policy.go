package livestore

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// completeBlockPolicy keeps complete-block lifecycle decisions local to the
// live-store package. The initial implementation preserves the current Kafka
// behaviour; single-binary-specific behaviour will be added in a follow-up.
type completeBlockPolicy interface {
	onCompletedBlock(ctx context.Context, tenantID string, blockID uuid.UUID) error
	onReloadedBlock(ctx context.Context, tenantID string, blockID uuid.UUID, block *LocalBlock) error
	shouldDeleteCompleteBlock(block *LocalBlock, cutoff time.Time) bool
}

func newCompleteBlockPolicy(_ Config) completeBlockPolicy {
	return kafkaCompleteBlockPolicy{}
}

type kafkaCompleteBlockPolicy struct{}

func (kafkaCompleteBlockPolicy) onCompletedBlock(context.Context, string, uuid.UUID) error {
	return nil
}

func (kafkaCompleteBlockPolicy) onReloadedBlock(context.Context, string, uuid.UUID, *LocalBlock) error {
	return nil
}

func (kafkaCompleteBlockPolicy) shouldDeleteCompleteBlock(block *LocalBlock, cutoff time.Time) bool {
	if block == nil {
		return false
	}

	return block.BlockMeta().EndTime.Before(cutoff)
}
