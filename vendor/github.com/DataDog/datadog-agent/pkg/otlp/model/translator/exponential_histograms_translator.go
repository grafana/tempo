// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translator

import (
	"context"
	"fmt"
	"math"

	"github.com/DataDog/sketches-go/ddsketch"
	"github.com/DataDog/sketches-go/ddsketch/mapping"
	"github.com/DataDog/sketches-go/ddsketch/store"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/DataDog/datadog-agent/pkg/quantile"
)

func toStore(b pmetric.ExponentialHistogramDataPointBuckets) store.Store {
	offset := b.Offset()
	bucketCounts := b.BucketCounts()

	store := store.NewDenseStore()
	for j := 0; j < bucketCounts.Len(); j++ {
		// Find the real index of the bucket by adding the offset
		index := j + int(offset)

		store.AddWithCount(index, float64(bucketCounts.At(j)))
	}
	return store
}

func (t *Translator) exponentialHistogramToDDSketch(
	p pmetric.ExponentialHistogramDataPoint,
	delta bool,
) (*ddsketch.DDSketch, error) {
	if !delta {
		return nil, fmt.Errorf("cumulative exponential histograms are not supported")
	}

	// Create the DDSketch stores
	positiveStore := toStore(p.Positive())
	negativeStore := toStore(p.Negative())

	// Create the DDSketch mapping that corresponds to the ExponentialHistogram settings
	gamma := math.Pow(2, math.Pow(2, float64(-p.Scale())))
	mapping, err := mapping.NewLogarithmicMappingWithGamma(gamma, 0)
	if err != nil {
		return nil, fmt.Errorf("couldn't create LogarithmicMapping for DDSketch: %w", err)
	}

	// Create DDSketch with the above mapping and stores
	sketch := ddsketch.NewDDSketch(mapping, positiveStore, negativeStore)
	err = sketch.AddWithCount(0, float64(p.ZeroCount()))
	if err != nil {
		return nil, fmt.Errorf("failed to add ZeroCount to DDSketch: %w", err)
	}

	return sketch, nil
}

// mapExponentialHistogramMetrics maps exponential histogram metrics slices to Datadog metrics
//
// An ExponentialHistogram metric has:
// - The count of values in the population
// - The sum of values in the population
// - A scale, from which the base of the exponential histogram is computed
// - Two bucket stores, each with:
//   - an offset
//   - a list of bucket counts
//
// - A count of zero values in the population
func (t *Translator) mapExponentialHistogramMetrics(
	ctx context.Context,
	consumer Consumer,
	dims *Dimensions,
	slice pmetric.ExponentialHistogramDataPointSlice,
	delta bool,
) {
	for i := 0; i < slice.Len(); i++ {
		p := slice.At(i)
		startTs := uint64(p.StartTimestamp())
		ts := uint64(p.Timestamp())
		pointDims := dims.WithAttributeMap(p.Attributes())

		histInfo := histogramInfo{ok: true}

		countDims := pointDims.WithSuffix("count")
		if delta {
			histInfo.count = p.Count()
		} else if dx, ok := t.prevPts.Diff(countDims, startTs, ts, float64(p.Count())); ok {
			histInfo.count = uint64(dx)
		} else { // not ok
			histInfo.ok = false
		}

		sumDims := pointDims.WithSuffix("sum")
		if !t.isSkippable(sumDims.name, p.Sum()) {
			if delta {
				histInfo.sum = p.Sum()
			} else if dx, ok := t.prevPts.Diff(sumDims, startTs, ts, p.Sum()); ok {
				histInfo.sum = dx
			} else { // not ok
				histInfo.ok = false
			}
		} else { // skippable
			histInfo.ok = false
		}

		if t.cfg.SendCountSum && histInfo.ok {
			// We only send the sum and count if both values were ok.
			consumer.ConsumeTimeSeries(ctx, countDims, Count, ts, float64(histInfo.count))
			consumer.ConsumeTimeSeries(ctx, sumDims, Count, ts, histInfo.sum)
		}

		expHistDDSketch, err := t.exponentialHistogramToDDSketch(p, delta)
		if err != nil {
			t.logger.Debug("Failed to convert ExponentialHistogram into DDSketch",
				zap.String("metric name", dims.name),
				zap.Error(err),
			)
			continue
		}

		agentSketch, err := quantile.ConvertDDSketchIntoSketch(expHistDDSketch)
		if err != nil {
			t.logger.Debug("Failed to convert DDSketch into Sketch",
				zap.String("metric name", dims.name),
				zap.Error(err),
			)
		}

		if histInfo.ok {
			// override approximate sum, count and average in sketch with exact values if available.
			agentSketch.Basic.Cnt = int64(histInfo.count)
			agentSketch.Basic.Sum = histInfo.sum
			agentSketch.Basic.Avg = agentSketch.Basic.Sum / float64(agentSketch.Basic.Cnt)
		}
		if delta && p.HasMin() {
			agentSketch.Basic.Min = p.Min()
		}
		if delta && p.HasMax() {
			agentSketch.Basic.Max = p.Max()
		}

		consumer.ConsumeSketch(ctx, pointDims, ts, agentSketch)
	}
}
