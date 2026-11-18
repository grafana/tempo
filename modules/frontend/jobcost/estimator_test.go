package jobcost

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

func TestOpAndKindStrings(t *testing.T) {
	require.Equal(t, "traces", OpTraceByID.String())
	require.Equal(t, "search", OpSearch.String())
	require.Equal(t, "metadata", OpMetadata.String())
	require.Equal(t, "metrics", OpMetrics.String())
	require.Equal(t, "unknown", numOps.String())

	require.Equal(t, "none", KindNone.String())
	require.Equal(t, "block", KindBlock.String())
	require.Equal(t, "live_store", KindLiveStore.String())
}

func TestEstimatorBlock(t *testing.T) {
	tests := []struct {
		name string
		op   Op
		base uint64
		want Cost
	}{
		{"trace by id is the bytes per block", OpTraceByID, 10, Cost{Kind: KindBlock, Base: 10, Estimate: 10 * 128 * 1024}},
		{"search is a share of the block bytes", OpSearch, 1_000_000, Cost{Kind: KindBlock, Base: 1_000_000, Estimate: 10_000}},
		{"metadata is a share of the block bytes", OpMetadata, 1_000_000, Cost{Kind: KindBlock, Base: 1_000_000, Estimate: 10_000}},
		{"metrics is a share of the block bytes", OpMetrics, 1_000_000, Cost{Kind: KindBlock, Base: 1_000_000, Estimate: 10_000}},
		{"no base has no bytes", OpSearch, 0, Cost{Kind: KindBlock}},
		{"large base does not overflow", OpSearch, 1 << 60, Cost{Kind: KindBlock, Base: 1 << 60, Estimate: uint64(float64(1<<60) * 0.01)}},
		{"unknown op has no cost", numOps, 1_000_000, Cost{}},
	}

	e := NewEstimator(DefaultConfig())
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, e.Block(tc.op, tc.base))
		})
	}
}

func TestEstimatorLiveStore(t *testing.T) {
	e := NewEstimator(Config{LiveStoreCost: 7})
	require.Equal(t, Cost{Kind: KindLiveStore, Estimate: 7}, e.LiveStore())

	// a cost of 0 disables the estimate
	e = NewEstimator(Config{})
	require.Equal(t, Cost{}, e.LiveStore())
}

func TestEstimatorCalibration(t *testing.T) {
	// observe finishes n jobs of the given kind that cover base and read actual bytes
	observe := func(e *Estimator, op Op, kind Kind, n int, base, actual uint64) {
		for range n {
			e.Observe(op, Cost{Kind: kind, Base: base}, actual)
		}
	}

	tests := []struct {
		name    string
		cfg     Config
		observe func(e *Estimator)
		// factors expected after the jobs, ops that are not listed keep their initial factor
		want  map[Op]float64
		delta float64
	}{
		{
			name:    "initial factor until enough jobs finished",
			cfg:     DefaultConfig(),
			observe: func(e *Estimator) { observe(e, OpSearch, KindBlock, minSamples-1, 1_000_000, 500_000) },
			want:    map[Op]float64{OpSearch: 0.01},
		},
		{
			name:    "calibrated with exactly the minimum number of jobs",
			cfg:     DefaultConfig(),
			observe: func(e *Estimator) { observe(e, OpSearch, KindBlock, minSamples, 1_000_000, 50_000) },
			want:    map[Op]float64{OpSearch: 0.05},
			delta:   1e-9,
		},
		{
			name:    "converges to the share that is read",
			cfg:     DefaultConfig(),
			observe: func(e *Estimator) { observe(e, OpSearch, KindBlock, 5*minSamples, 1_000_000, 50_000) },
			want:    map[Op]float64{OpSearch: 0.05},
			delta:   1e-9,
		},
		{
			name: "big jobs weigh more than small ones",
			cfg:  DefaultConfig(),
			observe: func(e *Estimator) {
				// per round 9 small jobs reading 10% and 1 big job reading 1%, the mean of the ratios is 9.1%
				for range 200 {
					observe(e, OpSearch, KindBlock, 9, 1_000_000, 100_000)
					observe(e, OpSearch, KindBlock, 1, 10_000_000, 100_000)
				}
			},
			want:  map[Op]float64{OpSearch: 1_000_000.0 / 19_000_000},
			delta: 0.002,
		},
		{
			name:    "jobs that read nothing are samples",
			cfg:     DefaultConfig(),
			observe: func(e *Estimator) { observe(e, OpSearch, KindBlock, 2*minSamples, 1_000_000, 0) },
			want:    map[Op]float64{OpSearch: 0.001}, // limited to initial / maxDeviation
		},
		{
			name:    "factor is limited upwards",
			cfg:     DefaultConfig(),
			observe: func(e *Estimator) { observe(e, OpSearch, KindBlock, 2*minSamples, 1_000_000, 900_000) },
			want:    map[Op]float64{OpSearch: 0.1}, // limited to initial * maxDeviation
		},
		{
			name: "factor can exceed 1 and ops are independent",
			cfg:  DefaultConfig(),
			observe: func(e *Estimator) {
				// 10 blocks per job, each read costs 256 KiB
				observe(e, OpTraceByID, KindBlock, 2*minSamples, 10, 10*256*1024)
			},
			want: map[Op]float64{OpTraceByID: 256 * 1024},
		},
		{
			name:    "disabled calibration keeps the initial factor",
			cfg:     Config{CalibrationEnabled: false},
			observe: func(e *Estimator) { observe(e, OpSearch, KindBlock, 5*minSamples, 1_000_000, 50_000) },
			want:    map[Op]float64{OpSearch: 0.01},
		},
		{
			name:    "jobs without base are ignored",
			cfg:     DefaultConfig(),
			observe: func(e *Estimator) { observe(e, OpSearch, KindBlock, 5*minSamples, 0, 50_000) },
			want:    map[Op]float64{OpSearch: 0.01},
		},
		{
			name:    "live store jobs do not calibrate",
			cfg:     DefaultConfig(),
			observe: func(e *Estimator) { observe(e, OpSearch, KindLiveStore, 5*minSamples, 1_000_000, 50_000) },
			want:    map[Op]float64{OpSearch: 0.01},
		},
		{
			name:    "unknown op is ignored",
			cfg:     DefaultConfig(),
			observe: func(e *Estimator) { observe(e, numOps, KindBlock, 5*minSamples, 1_000_000, 50_000) },
			want:    map[Op]float64{OpSearch: 0.01},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := NewEstimator(tc.cfg)
			tc.observe(e)

			for op := range numOps {
				want, ok := tc.want[op]
				if !ok {
					want = initialFactors[op]
				}
				require.InDelta(t, want, e.Factor(op), max(tc.delta, want*1e-9), op.String())
			}
			require.Zero(t, e.Factor(numOps))
		})
	}
}

func TestEstimatorCalibrationFollowsChanges(t *testing.T) {
	e := NewEstimator(DefaultConfig())
	cost := Cost{Kind: KindBlock, Base: 1_000_000}

	for range 3 * minSamples {
		e.Observe(OpMetrics, cost, 20_000)
	}
	require.InDelta(t, 0.02, e.Factor(OpMetrics), 1e-9)

	// after the jobs change, the factor follows them
	for range 10_000 {
		e.Observe(OpMetrics, cost, 40_000)
	}
	require.InDelta(t, 0.04, e.Factor(OpMetrics), 1e-3)
}

func TestEstimatorUsesCalibratedFactor(t *testing.T) {
	e := NewEstimator(DefaultConfig())
	cost := e.Block(OpSearch, 1_000_000)
	require.Equal(t, uint64(10_000), cost.Estimate)

	for range 3 * minSamples {
		e.Observe(OpSearch, cost, 50_000)
	}

	cost = e.Block(OpSearch, 1_000_000)
	require.Equal(t, uint64(50_000), cost.Estimate)
}

func TestEstimatorConcurrent(t *testing.T) {
	e := NewEstimator(DefaultConfig())

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// half of the goroutines read 2% of the base, the other half 4%
			actual := uint64(20_000 * (1 + i%2))
			for range 1000 {
				cost := e.Block(OpSearch, 1_000_000)
				e.Observe(OpSearch, cost, actual)
				_ = e.Factor(OpSearch)
			}
		}()
	}
	wg.Wait()

	require.Greater(t, e.Factor(OpSearch), 0.02)
	require.Less(t, e.Factor(OpSearch), 0.04)
}

func TestNilEstimator(t *testing.T) {
	var e *Estimator

	require.Equal(t, Cost{}, e.Block(OpSearch, 1_000_000))
	require.Equal(t, Cost{}, e.LiveStore())
	require.Zero(t, e.Factor(OpSearch))
	require.NotPanics(t, func() { e.Observe(OpSearch, Cost{Kind: KindBlock, Base: 1}, 1) })

	// the zero value has no calibrators
	e = &Estimator{}
	require.Equal(t, Cost{}, e.Block(OpSearch, 1_000_000))
	require.Zero(t, e.Factor(OpSearch))
	require.NotPanics(t, func() { e.Observe(OpSearch, Cost{Kind: KindBlock, Base: 1}, 1) })
}

func TestConfigYAML(t *testing.T) {
	cfg := DefaultConfig()
	err := yaml.UnmarshalStrict([]byte(`
calibration_enabled: false
live_store_cost: 100
`), &cfg)
	require.NoError(t, err)

	require.False(t, cfg.CalibrationEnabled)
	require.Equal(t, uint64(100), cfg.LiveStoreCost)
}
