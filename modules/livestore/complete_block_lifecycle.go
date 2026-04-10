package livestore

import (
	"context"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/tempodb"
	"github.com/prometheus/client_golang/prometheus"
)

// completeBlockFlusher is the minimal write capability needed by the
// single-binary complete-block lifecycle.
type completeBlockFlusher interface {
	WriteBlock(ctx context.Context, block tempodb.WriteableBlock) error
}

// completeBlockLifecycle owns mode-specific handling for locally completed
// blocks. The initial implementation preserves the current Kafka behaviour;
// single-binary-specific background flushing will be added in a follow-up.
type completeBlockLifecycle interface {
	start(ctx context.Context)
	stop()
	onCompletedBlock(ctx context.Context, tenantID string, block *LocalBlock) error
	onReloadedBlock(ctx context.Context, tenantID string, block *LocalBlock) error
	shouldDeleteCompleteBlock(block *LocalBlock, cutoff time.Time) bool
}

func newCompleteBlockLifecycle(_ Config, _ completeBlockFlusher, _ log.Logger, _ prometheus.Registerer) completeBlockLifecycle {
	return kafkaCompleteBlockLifecycle{}
}

type kafkaCompleteBlockLifecycle struct{}

func (kafkaCompleteBlockLifecycle) start(context.Context) {}

func (kafkaCompleteBlockLifecycle) stop() {}

func (kafkaCompleteBlockLifecycle) onCompletedBlock(context.Context, string, *LocalBlock) error {
	return nil
}

func (kafkaCompleteBlockLifecycle) onReloadedBlock(context.Context, string, *LocalBlock) error {
	return nil
}

func (kafkaCompleteBlockLifecycle) shouldDeleteCompleteBlock(block *LocalBlock, cutoff time.Time) bool {
	if block == nil {
		return false
	}

	return block.BlockMeta().EndTime.Before(cutoff)
}
