package bloomgatewayevents

import (
	"context"

	"github.com/grafana/tempo/tempodb/backend"
)

// Notifier adapts a Publisher to tempodb.CompactionNotifier
// (tempodb/tempodb.go) structurally, without this package importing
// tempodb -- the compile-time assertion that Notifier actually satisfies
// that interface lives in modules/backendworker, which already imports
// both.
//
// Notifier is deliberately stateless: tempodb's own per-job closure
// (tempodb/compactor.go's CompactWithConfig) already resolves every
// attribution decision -- which trace IDs belong to which output block, and
// discarding a failed job's partial capture -- before BlockCompacted is
// ever called. That leaves nothing here for concurrent or failed compaction
// jobs to corrupt.
type Notifier struct {
	pub *Publisher
}

// NewNotifier wraps pub so it can be installed via
// storage.Store.SetCompactionNotifier.
func NewNotifier(pub *Publisher) *Notifier {
	return &Notifier{pub: pub}
}

// BlockCompacted delivers a completed compaction output block plus every
// trace ID written into it.
func (n *Notifier) BlockCompacted(meta *backend.BlockMeta, traceIDs [][]byte) {
	// context.Background() is correct here: this is invoked from deep
	// inside compaction with no caller-supplied ctx to thread through, and
	// PublishAdd applies its own 2x WriteTimeout deadline ceiling
	// internally (Publisher.withDeadlineCeiling), so this call can never
	// hang regardless.
	n.pub.PublishAdd(context.Background(), meta.BlockID, meta.TenantID, meta.StartTime, meta.EndTime, traceIDs)
}

// BlockDeleted fires strictly after a successful physical ClearBlock.
func (n *Notifier) BlockDeleted(meta *backend.CompactedBlockMeta) {
	// Same reasoning as BlockCompacted above: no caller ctx exists this
	// deep inside retention, and PublishDelete is bounded the same way.
	// meta.TenantID is used by PublishDelete only for rate limiting -- it is
	// never put on the wire (BloomGatewayDelete has no tenant field).
	n.pub.PublishDelete(context.Background(), meta.BlockID, meta.TenantID)
}
