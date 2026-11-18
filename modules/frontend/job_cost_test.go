package frontend

import (
	"testing"

	"github.com/grafana/tempo/v3/modules/frontend/jobcost"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/stretchr/testify/require"
)

// newTestEstimator returns an estimator with its initial factors and a different live store cost per op.
func newTestEstimator() *jobcost.Estimator {
	return jobcost.NewEstimator(jobcost.Config{
		LiveStoreCost: jobcost.LiveStoreCostConfig{
			TraceByID: 1000,
			Search:    2000,
			Metadata:  3000,
			Metrics:   4000,
		},
	})
}

func TestBlockJobCost(t *testing.T) {
	m := &backend.BlockMeta{Size_: 1000, TotalRecords: 100}

	tcs := []struct {
		name      string
		estimator *jobcost.Estimator
		startPage int
		pages     int
		expected  jobcost.Cost
	}{
		{
			name:      "whole block",
			estimator: newTestEstimator(),
			pages:     100,
			expected:  jobcost.Cost{Kind: jobcost.KindBlock, Base: 1000, Estimate: 10},
		},
		{
			name:      "part of the block",
			estimator: newTestEstimator(),
			startPage: 10,
			pages:     20,
			expected:  jobcost.Cost{Kind: jobcost.KindBlock, Base: 200, Estimate: 2},
		},
		{
			name:      "last job is cut off at the end of the block",
			estimator: newTestEstimator(),
			startPage: 90,
			pages:     40,
			expected:  jobcost.Cost{Kind: jobcost.KindBlock, Base: 100, Estimate: 1},
		},
		{
			name:      "no estimator",
			estimator: nil,
			pages:     100,
			expected:  jobcost.Cost{},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, blockJobCost(tc.estimator, jobcost.OpSearch, m, tc.startPage, tc.pages))
		})
	}
}
