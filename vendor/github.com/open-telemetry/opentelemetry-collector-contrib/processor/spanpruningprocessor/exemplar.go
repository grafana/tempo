// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

import (
	"math"
	"math/rand/v2"
	"strings"

	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"
)

// exemplarSampleSize returns the number of exemplars to draw from a group of
// the given size: ceil(multiplier * sqrt(groupSize)), clamped to [0, groupSize].
func exemplarSampleSize(groupSize int, multiplier float64) int {
	if groupSize <= 0 || multiplier <= 0 {
		return 0
	}
	return min(int(math.Ceil(multiplier*math.Sqrt(float64(groupSize)))), groupSize)
}

// sampleExemplars selects ceil(multiplier * sqrt(len(nodes))) nodes uniformly
// at random without replacement and returns (remaining, exemplars). When the
// computed sample size equals or exceeds len(nodes) the entire input becomes
// exemplars and remaining is nil. The remaining slice preserves the input
// order, matching filterOutlierNodes.
func sampleExemplars(nodes []*spanNode, multiplier float64) (remaining, exemplars []*spanNode) {
	groupSize := len(nodes)
	sampleSize := exemplarSampleSize(groupSize, multiplier)
	if sampleSize == 0 {
		return nodes, nil
	}
	if sampleSize >= groupSize {
		return nil, nodes
	}

	// Partial Fisher-Yates over an index slice: pick sampleSize distinct
	// indices in O(sampleSize) swaps without materializing a full permutation.
	indices := make([]int, groupSize)
	for i := range indices {
		indices[i] = i
	}
	for i := range sampleSize {
		j := i + rand.IntN(groupSize-i)
		indices[i], indices[j] = indices[j], indices[i]
	}

	exemplarSet := make(map[int]struct{}, sampleSize)
	for _, idx := range indices[:sampleSize] {
		exemplarSet[idx] = struct{}{}
	}

	remaining = make([]*spanNode, 0, groupSize-sampleSize)
	exemplars = make([]*spanNode, 0, sampleSize)
	for i, node := range nodes {
		if _, ok := exemplarSet[i]; ok {
			exemplars = append(exemplars, node)
		} else {
			remaining = append(remaining, node)
		}
	}
	return remaining, exemplars
}

// updateExemplarThreshold composes the exemplar span's CPS sampling threshold
// to account for the additional sampling step (sampleSize kept out of
// groupSize). The new probability encodes p_old * sampleSize/groupSize so
// consumers can extrapolate via adjusted counts.
//
// If the span carries no prior th-value, the TraceState is left untouched —
// we do not fabricate CPS context where the upstream provided none.
func updateExemplarThreshold(span ptrace.Span, sampleSize, groupSize int) {
	if sampleSize <= 0 || groupSize <= 0 {
		return
	}
	raw := span.TraceState().AsRaw()
	w3c, err := sampling.NewW3CTraceState(raw)
	if err != nil {
		return
	}
	otts := w3c.OTelValue()
	priorThreshold, hasPriorTh := otts.TValueThreshold()
	if !hasPriorTh {
		return
	}

	composedProbability := priorThreshold.Probability() * float64(sampleSize) / float64(groupSize)
	// ProbabilityToThreshold rejects values below MinSamplingProbability. If
	// the composed probability is unrepresentable in 56-bit threshold encoding,
	// leave the prior th untouched — consumers will slightly undercount, which
	// is far better than clamping (which would massively overcount via an
	// inflated adjusted count).
	newThreshold, err := sampling.ProbabilityToThreshold(composedProbability)
	if err != nil {
		return
	}
	if err := otts.UpdateTValueWithSampling(newThreshold); err != nil {
		// ErrInconsistentSampling cannot occur here since sampleSize/groupSize
		// is at most 1, so the new threshold is never smaller than the prior.
		// Any other error indicates a malformed tracestate we don't want to
		// overwrite.
		return
	}

	var serialized strings.Builder
	if err := w3c.Serialize(&serialized); err != nil {
		return
	}
	span.TraceState().FromRaw(serialized.String())
}
