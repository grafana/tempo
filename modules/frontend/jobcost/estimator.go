package jobcost

// Initial factors of block jobs, used on startup, until enough jobs finished and as the fallback when
// calibration is disabled. The factor of trace-by-ID is the bytes read per block, the others are the share
// of the bytes of the block range that is read. They are first guesses from fleet wide averages.
var initialFactors = [numOps]float64{
	OpTraceByID: 128 * 1024,
	OpSearch:    0.01,
	OpMetadata:  0.01,
	OpMetrics:   0.01,
}

// Config configures the Estimator.
type Config struct {
	// CalibrationEnabled calibrates the factors of block jobs with the bytes that finished jobs read.
	// If disabled the initial factors are used.
	CalibrationEnabled bool `yaml:"calibration_enabled"`

	// LiveStoreCost is the estimate in bytes of a job that reads recent data from the live-stores or the
	// generators. 0 means the job has no estimate.
	LiveStoreCost uint64 `yaml:"live_store_cost"`
}

// Estimator estimates the bytes a job reads and calibrates the estimates with finished jobs.
// It is safe for concurrent use.
type Estimator struct {
	block     [numOps]*calibrator
	liveStore uint64
}

// NewEstimator returns an Estimator for cfg.
func NewEstimator(cfg Config) *Estimator {
	e := &Estimator{liveStore: cfg.LiveStoreCost}
	for op := range numOps {
		e.block[op] = newCalibrator(initialFactors[op], cfg.CalibrationEnabled)
	}
	return e
}

// Block returns the cost of a job that reads blocks in the backend. See Cost.Base for base.
// An unknown op or a nil Estimator has no cost.
func (e *Estimator) Block(op Op, base uint64) Cost {
	c := e.calibrator(op)
	if c == nil {
		return Cost{}
	}
	return Cost{
		Kind:     KindBlock,
		Base:     base,
		Estimate: uint64(float64(base) * c.factor()),
	}
}

// LiveStore returns the cost of a job that reads recent data. A nil Estimator or a configured cost of 0
// has no cost.
func (e *Estimator) LiveStore() Cost {
	if e == nil || e.liveStore == 0 {
		return Cost{}
	}
	return Cost{Kind: KindLiveStore, Estimate: e.liveStore}
}

// Observe adds a finished job with the bytes it actually read. Only block jobs calibrate the factors.
func (e *Estimator) Observe(op Op, cost Cost, actual uint64) {
	c := e.calibrator(op)
	if c == nil || cost.Kind != KindBlock {
		return
	}
	c.observe(cost.Base, actual)
}

// Factor returns the factor in use for the block jobs of op.
// It is the bytes a job is expected to read per unit of Base (Estimate = Base * Factor).
// It is 0 for an unknown op or a nil Estimator.
func (e *Estimator) Factor(op Op) float64 {
	c := e.calibrator(op)
	if c == nil {
		return 0
	}
	return c.factor()
}

func (e *Estimator) calibrator(op Op) *calibrator {
	if e == nil || op >= numOps {
		return nil
	}
	return e.block[op]
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		CalibrationEnabled: true,
		LiveStoreCost:      256 * 1024,
	}
}
