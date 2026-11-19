package jobcost

import "fmt"

const (
	// defaultMinFactor and defaultMaxFactor are the default bounds of the calibrated factors of search,
	// metadata and metrics jobs, a tenth and ten times their initial factor.
	defaultMinFactor = 0.001
	defaultMaxFactor = 0.1
)

// Initial factors of block jobs, used on startup, until enough jobs finished and as the fallback when
// calibration is disabled. The factor of trace-by-ID is the bytes read per block, the others are the share
// of the bytes of the block range that is read and are moved into the configured limits. They are first
// guesses from fleet wide averages.
var initialFactors = [numOps]float64{
	OpTraceByID: 128 * 1024,
	OpSearch:    0.01,
	OpMetadata:  0.01,
	OpMetrics:   0.01,
}

// Config configures the Estimator.
type Config struct {
	// Calibration configures how the factors of block jobs are calibrated with the bytes that finished jobs read.
	Calibration CalibrationConfig `yaml:"calibration"`

	// LiveStoreCost is the estimate in bytes of a job that reads recent data from the live-stores or the
	// generators. 0 means the job has no estimate.
	LiveStoreCost uint64 `yaml:"live_store_cost"`
}

// Validate returns an error if the configuration is invalid.
func (c Config) Validate() error {
	return c.Calibration.validate()
}

// CalibrationConfig configures the calibration of the factors of block jobs.
type CalibrationConfig struct {
	// Enabled calibrates the factors with the bytes that finished jobs read. If disabled the initial factors
	// are used.
	Enabled bool `yaml:"enabled"`

	// MinFactor and MaxFactor limit the calibrated factors of search, metadata and metrics jobs, which are the
	// share of the bytes of a block range that a job reads, so 0 < MinFactor <= MaxFactor <= 1. A factor never
	// leaves this range, which also moves the initial factor into it if it lies outside. The factor of trace by id
	// jobs is the bytes read per block and is limited to a tenth and ten times its initial factor.
	MinFactor float64 `yaml:"min_factor"`
	MaxFactor float64 `yaml:"max_factor"`
}

func (c CalibrationConfig) validate() error {
	if !c.Enabled || c.boundsValid() {
		return nil
	}
	return fmt.Errorf("query_frontend.job_cost.calibration: min_factor (%v) and max_factor (%v) must satisfy 0 < min_factor <= max_factor <= 1", c.MinFactor, c.MaxFactor)
}

// boundsValid is false for NaN, infinite and negative values, as every comparison with them fails.
func (c CalibrationConfig) boundsValid() bool {
	return c.MinFactor > 0 && c.MinFactor <= c.MaxFactor && c.MaxFactor <= 1
}

// bounds returns the limits of the calibrated factors of search, metadata and metrics jobs. Invalid limits
// are replaced by the defaults, so a zero Config works. Validate rejects invalid limits of an enabled calibration.
func (c CalibrationConfig) bounds() (minFactor, maxFactor float64) {
	if !c.boundsValid() {
		return defaultMinFactor, defaultMaxFactor
	}
	return c.MinFactor, c.MaxFactor
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
	minFactor, maxFactor := cfg.Calibration.bounds()

	for op := range numOps {
		initial := initialFactors[op]
		lo, hi := initial/maxDeviation, initial*maxDeviation
		if op != OpTraceByID {
			lo, hi = minFactor, maxFactor
			initial = min(max(initial, lo), hi)
		}
		e.block[op] = newCalibrator(initial, lo, hi, cfg.Calibration.Enabled)
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
		Calibration: CalibrationConfig{
			Enabled:   true,
			MinFactor: defaultMinFactor,
			MaxFactor: defaultMaxFactor,
		},
		LiveStoreCost: 256 * 1024,
	}
}
