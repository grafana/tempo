package jobcost

import (
	"math"
	"sync"
	"sync/atomic"
)

const (
	// maxDeviation limits how far calibration moves a factor away from its initial value, in both directions.
	maxDeviation = 10.0

	// minSamples is the number of jobs a factor must have seen before it is calibrated.
	minSamples = 100

	// decay is applied to the samples seen so far with every new sample, so a factor follows the
	// last ~1000 jobs.
	decay = 0.999
)

// calibrator tracks the factor between the bytes jobs actually read and their Base. The factor is
// the ratio of the decayed sums of both, so big jobs weigh more than small ones, which fits an estimate
// that is meant to represent bytes.
type calibrator struct {
	enabled              bool
	factorMin, factorMax float64

	factorBits atomic.Uint64 // float64 factorBits of the factor in use

	mu        sync.Mutex
	sumActual float64
	sumBase   float64
	samples   uint64
}

func newCalibrator(initial float64, enabled bool) *calibrator {
	c := &calibrator{
		enabled:   enabled,
		factorMin: initial / maxDeviation,
		factorMax: initial * maxDeviation,
	}
	c.factorBits.Store(math.Float64bits(initial))
	return c
}

// factor returns the factor in use, the bytes a job is expected to read per unit of Base.
// Once calibrated, it is sumActual/sumBase limited to [factorMin, factorMax].
func (c *calibrator) factor() float64 {
	return math.Float64frombits(c.factorBits.Load())
}

// observe adds a finished job. Jobs that read nothing are valid samples, jobs without a Base are ignored.
func (c *calibrator) observe(base, actual uint64) {
	if !c.enabled || base == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.sumActual = c.sumActual*decay + float64(actual)
	c.sumBase = c.sumBase*decay + float64(base)
	c.samples++

	if c.samples < minSamples {
		return
	}

	f := min(max(c.sumActual/c.sumBase, c.factorMin), c.factorMax)
	c.factorBits.Store(math.Float64bits(f))
}
