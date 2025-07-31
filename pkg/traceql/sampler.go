package traceql

import (
	"fmt"
)

type Sampler interface {
	Expect(weight uint64)
	Sample(weight uint64) bool
	Measured(weight uint64)
	FinalScalingFactor() float64
}

type probabilisticSampler struct {
	sample float64
	curr   int
	lim    int

	found, skipped uint64
}

var _ Sampler = (*probabilisticSampler)(nil)

func newProbablisticSampler(sample float64) *probabilisticSampler {
	lim := int(1 / sample)

	return &probabilisticSampler{
		sample: sample,
		curr:   -1, // Always sample first
		lim:    lim,
	}
}

// Sample returns if the incoming data should be sampled. Call this for
// every possible trace.
func (s *probabilisticSampler) Sample(_ uint64) bool {
	s.found++

	s.curr = (s.curr + 1) % s.lim

	if s.curr == 0 {
		return true
	}

	s.skipped++
	return false
}

// Measured is called on matching data
func (s *probabilisticSampler) Measured(_ uint64) {
}

func (s *probabilisticSampler) Expect(weight uint64) {
}

func (s *probabilisticSampler) FinalScalingFactor() float64 {
	return float64(s.found) / float64(s.found-s.skipped)
}

type cochraneSampler struct {
	expected                   uint64
	measured, skipped, sampled uint64
	lost                       uint64
	weighted                   bool
	fpc                        bool
	p                          float64
	e                          float64
	z                          float64
	info                       bool
	debug                      bool

	targetSampleCount uint64

	// Probabilistic part
	curr int
	lim  int
}

var _ Sampler = (*cochraneSampler)(nil)

func newCochraneSampler() *cochraneSampler {
	return &cochraneSampler{
		weighted: false,
		fpc:      true,
		p:        0,
		// z := 1.96 // 95% confidence interval
		z:     2.576, // 99% confidence interval
		e:     0.01,  // Margin of error
		debug: false,
		curr:  -1,
	}
}

func (s *cochraneSampler) Expect(weight uint64) {
	// Loss is samples that we were told to expect, but were never called against Sample()
	// Because we are skipping over chunks of data. We keep track of them for better proportion calculations.
	// But they don't go into the scaling factor at the end.
	lost := s.expected - s.sampled - s.skipped - s.lost
	if lost > 0 {
		s.lost += lost
	}

	s.expected += weight
	s.recompute()
}

func (s *cochraneSampler) Measured(weight uint64) {
	if !s.weighted {
		weight = 1
	}

	s.measured += weight
	s.recompute()
}

func (s *cochraneSampler) recompute() {
	// This is the proportion of samples that turned into matches
	p := float64(s.measured) / float64(s.sampled+s.lost)

	// Something present in 100% of data will be sampled at 10% (1 in 10)
	// Something present in 50% of data will be sampled at 20% (1 in 5)
	// Something present in 1% of data will be sampled at 100% (1 in 1)

	s.lim = int(p * 10)
	if s.lim > 10 {
		s.lim = 10
	}
	return

	/*if p > 0.5 {
		// Very common, used fixed probability instead.
		// Target 10%
		s.lim = 10
		return
	}

	s.targetSampleCount = uint64(s.idealSampleSize())
	if s.targetSampleCount == 0 {
		s.targetSampleCount = 16000
	}

	// Recompute sampling rate to get the target number of samples

	// If we don't have real data to work off yet,
	// so go with probabilistic sampling.
	if s.sampled == 0 || s.measured == 0 {
		s.lim = int(s.expected / s.targetSampleCount)
		return
	}

	// This is the number of remaining entries
	remaining := s.expected - s.sampled - s.skipped - s.lost

	// This is the number of matches still needed.
	needed := float64(s.targetSampleCount) - float64(s.measured)

	if needed <= 0 {
		// We have enough samples, instead of stopping altogether,
		// keep sampling at the current rate. Better to oversample than undersample.
		return
	}

	// This the sampling rate that the remaining needs to be to
	// get the number of needed matches.
	p2 := float64(needed) / (float64(remaining) * p)
	s.lim = int(1 / p2)*/
}

func (s *cochraneSampler) Sample(weight uint64) bool {
	if !s.weighted {
		weight = 1
	}

	/*if s.measured == 0 || s.sampled == 0 {
		// We don't have data to make a calculation yet.
		s.lim = int(s.expected / s.min)
	}*/

	// ideal := s.idealSampleSize()

	// Proportion of measured vs total encountered
	// p2 := float64(s.measured) / float64(s.sampled+s.skipped)

	// Total samples we expect at this rate
	// expectedTotalSamples := float64(s.expected-s.sampled-s.skipped)*p2 + float64(s.measured)

	// If the estimate is over ideal, then skip this next sample.
	// Else, if the estimate is coming in too few, then sample.
	// skip := ideal > 0 && (expectedTotalSamples > ideal || float64(s.measured) >= ideal)

	/*if s.debug {
		var skipS string
		if skip {
			skipS = "skipping"
		}

		fmt.Println("cochraneSampler.Sample",
			"weight", weight,
			"expected", s.expected,
			"measured", s.measured,
			"ideal", ideal,
			"sampled", s.sampled,
			"skipped", s.skipped,
			"expectedTotalSamples", expectedTotalSamples,
			"p", float64(s.measured)/float64(s.sampled),
			skipS)
	}

	if skip {
		// We're either already over the ideal or we're about to be.
		// Don't sample.
		s.skipped += weight
		return false
	}

	// Keep going
	s.sampled += weight
	return true*/

	sample := false
	if s.lim == 0 || s.measured == 0 {
		sample = true
	} else {
		s.curr = (s.curr + 1) % s.lim
		if s.curr == 0 {
			sample = true
		}
	}

	if s.debug {
		var sampleS string
		if sample {
			sampleS = "sampling"
		}

		fmt.Println("cochraneSampler.Sample",
			"weight", weight,
			"expected", s.expected,
			"measured", s.measured,
			"sampled", s.sampled,
			"skipped", s.skipped,
			"targetSampleCount", s.targetSampleCount,
			"lim", s.lim,
			"p", float64(s.measured)/float64(s.sampled),
			sampleS)
	}

	if sample {
		s.sampled += weight
		return true
	}

	s.skipped += weight
	return false
}

func (s *cochraneSampler) idealSampleSize() float64 {
	if s.sampled == 0 {
		return 0
	}

	p := s.p
	if p == 0 {
		p = float64(s.measured) / float64(s.sampled)
		maxP := 0.5    // Maximum selection for common things.
		minP := 0.0001 // Allow us to go as low as 7 samples for very rare conditions
		if p > maxP {
			p = maxP
		}
		if p < minP {
			p = minP
		}
	}

	z2 := s.z * s.z
	e2 := s.e * s.e
	n0 := z2 * p * (1 - p) / e2

	// If the population size is known, use it to adjust the sample size
	if s.fpc {
		if s.expected > 0 {
			n0 = n0 / (1 + (n0-1)/float64(s.expected))
		}
	}

	return n0
}

func (s *cochraneSampler) FinalScalingFactor() float64 {
	// No data was skipped so no scaling needed.
	if s.skipped == 0 {
		if s.debug {
			fmt.Println("cochraneSampler.FinalScalingFactor nothing skipped", "expected", s.expected, "measured", s.measured, "sampled", s.sampled, "f", 1.0)
		}
		return 1.0
	}

	// Scale up to include the population that was skipped
	f := float64(s.sampled+s.skipped) / float64(s.sampled)

	if s.debug {
		fmt.Println("cochraneSampler.FinalScalingFactor", "expected", s.expected, "measured", s.measured, "skipped", s.skipped, "sampled", s.sampled, "lim", s.lim, "f", f)
	}
	return f
}

type minimumSampler struct {
	min               uint64
	expected          uint64
	measured, skipped uint64
	sampled           uint64

	curr  int
	lim   int
	debug bool
}

var _ Sampler = (*minimumSampler)(nil)

func newMinimumSampler(min uint64) *minimumSampler {
	return &minimumSampler{min: min, curr: -1}
}

func (s *minimumSampler) Sample(weight uint64) bool {
	if s.debug {
		fmt.Println("cochraneSampler.Sample",
			"weight", weight,
			"expected", s.expected,
			"measured", s.measured,
			"sampled", s.sampled,
			"skipped", s.skipped)
	}

	if s.expected == 0 || s.lim == 0 {
		// We don't have data to make a calculation yet.
		s.sampled += weight
		return true
	}

	s.curr = (s.curr + 1) % s.lim
	if s.curr == 0 {
		s.sampled += weight
		return true
	}

	s.skipped += weight
	return false
}

func (s *minimumSampler) Measured(weight uint64) {
	s.measured += weight
	s.recompute()
}

func (s *minimumSampler) FinalScalingFactor() float64 {
	return float64(s.sampled+s.skipped) / float64(s.sampled)
}

func (s *minimumSampler) Expect(weight uint64) {
	s.expected += weight
	s.recompute()
}

func (s *minimumSampler) recompute() {
	// Recompute sampling rate to get the minimum number of samples

	// If we don't have real data to work off yet,
	// so go with probabilistic sampling.
	if s.sampled == 0 || s.measured == 0 {
		s.lim = int(s.expected / s.min)
		return
	}

	// This is the number of remaining entries
	remaining := s.expected - s.sampled

	// This is the proportion of samples that turned into matches
	p := float64(s.measured) / float64(s.sampled)

	// This is the number of matches still needed.
	needed := float64(s.min) - float64(s.measured)

	// This the sampling rate that the remaining needs to be to
	// get the number of needed matches.
	p2 := float64(needed) / (float64(remaining) * p)

	s.lim = int(1 / p2)
}
