package traceql

import "fmt"

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
	weighted                   bool
	fpc                        bool
	p                          float64
}

var _ Sampler = (*cochraneSampler)(nil)

func newCochraneSampler(weighted bool, fpc bool, p float64) *cochraneSampler {
	return &cochraneSampler{
		weighted: weighted,
		fpc:      fpc,
		p:        p,
	}
}

func (s *cochraneSampler) Expect(weight uint64) {
	s.expected += weight
}

func (s *cochraneSampler) Sample(weight uint64) bool {
	if !s.weighted {
		weight = 1
	}

	ideal := s.idealSampleSize()

	// Proportion of measured vs total encountered
	p2 := float64(s.measured) / float64(s.sampled+s.skipped)

	// Total samples we expect at this rate
	expectedTotalSamples := float64(s.expected-s.sampled-s.skipped)*p2 + float64(s.measured)

	// If the estimate is over ideal, then skip this next sample.
	// Else, if the estimate is coming in too few, then sample.
	skip := ideal > 0 && (expectedTotalSamples > ideal || float64(s.measured) >= ideal)

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
		//"p", p,
		skipS)

	if skip {
		// We're either already over the ideal or we're about to be.
		// Don't sample.
		s.skipped += weight
		return false
	}

	// Keep going
	s.sampled += weight
	return true
}

func (s *cochraneSampler) idealSampleSize() float64 {
	if s.sampled == 0 {
		return 0
	}

	p := s.p
	if p == 0 {
		p = float64(s.measured) / float64(s.sampled)

		maxP := 0.9    // Don't allow us to go less than 13K samples for very common conditions
		minP := 0.0001 // Allow us to go as low as 7 samples for very rare conditions
		if p > maxP {
			p = maxP
		}
		if p < minP {
			p = minP
		}
	}

	// z := 1.96 // 95% confidence interval
	z := 2.576 // 99% confidence interval
	z2 := z * z
	e := 0.01 // Margin of error
	// e := 0.05
	e2 := e * e

	n0 := z2 * p * (1 - p) / e2

	// If the population size is known, use it to adjust the sample size
	if s.fpc {
		if s.expected > 0 {
			n0 = n0 / (1 + (n0-1)/float64(s.expected))
		}
	}
	return n0
}

func (s *cochraneSampler) Measured(weight uint64) {
	if !s.weighted {
		weight = 1
	}

	s.measured += weight
}

func (s *cochraneSampler) FinalScalingFactor() float64 {
	// No data was skipped so no scaling needed.
	if s.skipped == 0 {
		fmt.Println("cochraneSampler.FinalScalingFactor nothing skipped", "expected", s.expected, "measured", s.measured, "sampled", s.sampled, "f", 1.0)
		return 1.0
	}

	// p := float64(s.sampled) / float64(s.sampled+s.skipped)

	// Scale up to include the population that was skipped
	// f := 1.0 + float64(s.skipped)/p
	f := float64(s.sampled+s.skipped) / float64(s.sampled)

	// f := float64(s.total) / (p*float64(s.sampled) + p*float64(s.skipped))

	fmt.Println("cochraneSampler.FinalScalingFactor", "expected", s.expected, "measured", s.measured, "skipped", s.skipped, "sampled", s.sampled, "f", f)
	return f
}

type minimumSampler struct {
	min               uint64
	measured, skipped uint64
}

var _ Sampler = (*minimumSampler)(nil)

func newMinimumSampler(min uint64) *minimumSampler {
	return &minimumSampler{min: min}
}

func (s *minimumSampler) Sample(weight uint64) bool {
	if s.measured < s.min {
		return true
	}

	s.skipped += weight
	return false
}

func (s *minimumSampler) Measured(weight uint64) {
	s.measured += weight
}

func (s *minimumSampler) FinalScalingFactor() float64 {
	return float64(s.measured+s.skipped) / float64(s.measured)
}

func (s *minimumSampler) Expect(weight uint64) {
}
