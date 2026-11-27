package vparquet5

import (
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/tracefingerprint"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// newTraceSimilarity returns the stored columns for r. The columns cannot be
// null, so empty and unavailable results are stored as zeros. A computed
// fingerprint always has at least one shingle, so a stored ShingleCount of 0
// means the trace has no fingerprint.
func newTraceSimilarity(r tracefingerprint.Result) TraceSimilarity {
	if r.Status != tracefingerprint.StatusComputed {
		return TraceSimilarity{}
	}
	return TraceSimilarity{Signature: r.Fingerprint, ShingleCount: r.ShingleCount}
}

// estimatedSimilarityBytes returns how much the Similarity columns add to the
// size estimate of a trace written with cfg. A computed fingerprint is random
// and does not compress, so it counts at full size rather than the 1 byte per
// 20 used for other values. Unsigned rows store zeros, which compress away.
func estimatedSimilarityBytes(cfg *common.BlockConfig) int {
	if !cfg.ComputeTraceFingerprints {
		return 0
	}
	return tracefingerprint.Size + 4 // the uint32 shingle count
}

// traceFingerprinter computes the Similarity columns when CreateBlock is
// asked to with BlockConfig.ComputeTraceFingerprints.
type traceFingerprinter struct {
	f    tracefingerprint.Fingerprinter
	meta *backend.BlockMeta
	// readBack is set when the block has a dedicated resource column named
	// service.name. That column reads a one-element array back as a plain
	// string, so the fingerprint must come from the trace as it will be read.
	readBack bool
}

// newTraceFingerprinter returns a fingerprinter for blocks with meta.
// resMapping is the block's mapping of resource dedicated columns.
func newTraceFingerprinter(meta *backend.BlockMeta, resMapping dedicatedColumnMapping) *traceFingerprinter {
	_, readBack := resMapping.get(LabelServiceName)
	return &traceFingerprinter{meta: meta, readBack: readBack}
}

// similarity returns the Similarity columns for tr, which was converted to row.
func (t *traceFingerprinter) similarity(tr *tempopb.Trace, row *Trace) TraceSimilarity {
	if t.readBack {
		tr = ParquetTraceToTempopbTrace(t.meta, row)
	}
	return newTraceSimilarity(t.f.Fingerprint(tr))
}
