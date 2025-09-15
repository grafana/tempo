package traceql

import (
	"math"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/util/log"
)

type Sampler interface {
	// Sample is called on every entry. If this returns true the entry should
	// be inspected further. If this returns false then the entry should be skipped.
	Sample() bool

	// Measured is called on matching data coming into the TraceQL engine.
	// This is how samplers gain knowledge of the actual proportion of data matching the query.
	Measured()

	// Expect is called before any sampling and provides knowledge of the total number of entries
	// that are present in the dataset. This allows the sampler to alter behavior if needed.
	Expect(count uint64)

	// FinalScalingFactor is called at the end to and is how much to scale up the
	// results.
	FinalScalingFactor() float64
}

// probabilisticSampler applies a fixed percentage.  This exists primarily
// for benchmarking the adaptive approach.
type probabilisticSampler struct {
	sample float64
	curr   int
	lim    int

	sampled, skipped uint64
}

var _ Sampler = (*probabilisticSampler)(nil)

// newProbablisticSampler creates a sampler with the given ratio between 0 and 1.
// For performance it only supports ratios that are perfect divisors, i.e. 1/2, 1/3, 1/4, etc.
func newProbablisticSampler(sample float64) *probabilisticSampler {
	lim := int(1 / sample)

	return &probabilisticSampler{
		sample: sample,
		curr:   -1, // Always sample first
		lim:    lim,
	}
}

// Sample returns if the incoming data should be sampled. Call this for
// every possible entry.
func (s *probabilisticSampler) Sample() bool {
	s.curr = (s.curr + 1) % s.lim

	if s.curr == 0 {
		s.sampled++
		return true
	}

	s.skipped++
	return false
}

func (s *probabilisticSampler) Measured() {
}

func (s *probabilisticSampler) Expect(_ uint64) {
}

func (s *probabilisticSampler) FinalScalingFactor() float64 {
	if s.sampled == 0 {
		return 1.0
	}

	return float64(s.sampled+s.skipped) / float64(s.sampled)
}

type adaptiveSampler struct {
	expected                   uint64
	measured, skipped, sampled uint64
	lost                       uint64
	info                       bool
	debug                      bool

	// Probabilistic part
	curr int
	lim  int
}

var _ Sampler = (*adaptiveSampler)(nil)

func newAdaptiveSampler() *adaptiveSampler {
	return &adaptiveSampler{
		debug: false,
		curr:  -1,
	}
}

func (s *adaptiveSampler) Expect(count uint64) {
	// Loss are samples that we were told to expect, but were never called against Sample()
	// because the storage layer is skipping over chunks of data or exiting early.
	// We keep track of them for more accurate estimation of the proportion of matches
	// in the datset. But they don't go into the scaling factor at the end.
	s.lost = s.expected - s.sampled - s.skipped

	s.expected += count
	s.recompute()
}

func (s *adaptiveSampler) Measured() {
	s.measured++
	s.recompute()
}

func (s *adaptiveSampler) recompute() {
	if s.sampled == 0 {
		return
	}

	// This is the proportion of samples that turned into matches
	// Lost samples were already filtered out a lower level and never offered
	// to the sampler, so they count as not matching.
	p := float64(s.measured) / float64(s.sampled+s.lost)

	// This turns the proportion into a sampling rate.
	// Something present in 100% of data will be sampled at 10% (1 in 10)
	// Something present in 50% of data will be sampled at 20% (1 in 5)
	// Something present in 1 to 10% of data will be sampled at 100% (1 in 1)
	// The overall target is therefore up to 10% of the entire dataset.
	s.lim = rateFor(p)
}

func rateFor(proportion float64) int {
	l := int(math.Ceil(proportion * 10))
	if l > 10 {
		l = 10
	}
	return l
}

func (s *adaptiveSampler) Sample() bool {
	sample := false
	if s.lim == 0 || s.measured == 0 {
		// 100% sampling until we receive data and have
		// determined the probabilistic sampling rate.
		sample = true
	} else {
		s.curr = (s.curr + 1) % s.lim
		if s.curr == 0 {
			sample = true
		}
	}

	if s.debug {
		var sampleStr string
		if sample {
			sampleStr = "sampled"
		}

		level.Debug(log.Logger).Log(
			"msg", "adaptiveSampler.Sample",
			"expected", s.expected,
			"measured", s.measured,
			"sampled", s.sampled,
			"skipped", s.skipped,
			"lost", s.lost,
			"lim", s.lim,
			"p", float64(s.measured)/float64(s.sampled+s.lost),
			"decision", sampleStr,
		)
	}

	if sample {
		s.sampled++
		return true
	}

	s.skipped++
	return false
}

func (s *adaptiveSampler) FinalScalingFactor() float64 {
	var f float64
	if s.sampled == 0 {
		// Never called.
		f = 1.0
	} else {
		// Scale up to include the population that was specifically skipped
		// by the sampler.
		f = float64(s.sampled+s.skipped) / float64(s.sampled)
	}

	if s.info {
		level.Info(log.Logger).Log(
			"msg", "adaptiveSampler.FinalScalingFactor",
			"expected", s.expected,
			"measured", s.measured,
			"skipped", s.skipped,
			"sampled", s.sampled,
			"lim", s.lim,
			"f", f,
		)
	}
	return f
}
