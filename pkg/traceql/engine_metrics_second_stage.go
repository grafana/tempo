package traceql

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/grafana/tempo/pkg/tempopb"
)

type metricsSecondStageElement interface {
	Element
	extractConditions(request *FetchSpansRequest)
	init(req *tempopb.QueryRangeRequest, mode AggregateMode)
	observeSeries([]*tempopb.TimeSeries)
	result() []*tempopb.TimeSeries
}

// MetricsSecondStage handles second stage metrics operations (topK/bottomK)
// it takes output of the first stage pipeline as input and applies the second stage operations like topk, bottomk on the first stage data.
type MetricsSecondStage struct {
	op    SecondStageOp
	limit int
	input []*tempopb.TimeSeries
}

type SecondStageOp int

const (
	OpTopK SecondStageOp = iota
	OpBottomK
)

var errInvalidLimit = fmt.Errorf("limit must be greater than 0")

func (op SecondStageOp) String() string {
	switch op {
	case OpTopK:
		return "topk"
	case OpBottomK:
		return "bottomk"
	}
	return "unknown"
}

func newMetricsTopK(limit int) metricsSecondStageElement {
	return &MetricsSecondStage{op: OpTopK, limit: limit}
}

func newMetricsBottomK(limit int) metricsSecondStageElement {
	return &MetricsSecondStage{op: OpBottomK, limit: limit}
}

// Interface implementation
func (m *MetricsSecondStage) String() string {
	return fmt.Sprintf("%s(%d)", m.op.String(), m.limit)
}

func (m *MetricsSecondStage) validate() error {
	// TODO: should we also enforce a max limit for topk/bottomk?
	if m.limit <= 0 {
		return errInvalidLimit
	}
	return nil
}

func (m *MetricsSecondStage) extractConditions(*FetchSpansRequest) {
	// todo: implement this?? also do we need this??

}

func (m *MetricsSecondStage) init(*tempopb.QueryRangeRequest, AggregateMode) {
	// todo: implement this?? assign input from last stage to m.input??
	// also, do we need this if we are already initializing input in observeSeries??
	m.input = nil
}

func (m *MetricsSecondStage) observeSeries(series []*tempopb.TimeSeries) {
	m.input = series
}

func (m *MetricsSecondStage) result() []*tempopb.TimeSeries {
	if len(m.input) == 0 {
		return nil
	}

	// Create a slice of indices to sort instead of sorting the series directly
	indices := make([]int, len(m.input))
	for i := range indices {
		indices[i] = i
	}

	// Sort indices based on series values
	slices.SortStableFunc(indices, func(i, j int) int {
		aVal := getAvgValue(m.input[i])
		bVal := getAvgValue(m.input[j])

		if m.op == OpTopK {
			if math.IsNaN(aVal) && math.IsNaN(bVal) {
				return 0
			}
			if math.IsNaN(aVal) {
				return 1
			}
			if math.IsNaN(bVal) {
				return -1
			}
			return -cmp.Compare(aVal, bVal)
		}

		if math.IsNaN(aVal) && math.IsNaN(bVal) {
			return 0
		}
		if math.IsNaN(aVal) {
			return 1
		}
		if math.IsNaN(bVal) {
			return -1
		}
		return cmp.Compare(aVal, bVal)
	})

	// Create result using sorted indices
	k := min(m.limit, len(m.input))
	result := make([]*tempopb.TimeSeries, k)
	for i := 0; i < k; i++ {
		result[i] = m.input[indices[i]]
	}
	return result
}

// Helper function
func getAvgValue(series *tempopb.TimeSeries) float64 {
	var sum float64
	count := 0
	for _, s := range series.Samples {
		if !math.IsNaN(s.Value) {
			sum += s.Value
			count++
		}
	}
	if count == 0 {
		return math.NaN() // Return NaN if all values are NaN
	}
	return sum / float64(count)
}

var _ metricsSecondStageElement = (*MetricsSecondStage)(nil)
