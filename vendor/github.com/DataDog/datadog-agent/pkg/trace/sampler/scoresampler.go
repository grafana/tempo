// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package sampler

import (
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
)

const (
	errorsRateKey     = "_dd.errors_sr"
	noPriorityRateKey = "_dd.no_p_sr"
	// shrinkCardinality is the max Signature cardinality before shrinking
	shrinkCardinality = 200
)

// ErrorsSampler is dedicated to catching traces containing spans with errors.
type ErrorsSampler struct{ ScoreSampler }

// NoPrioritySampler is dedicated to catching traces with no priority set.
type NoPrioritySampler struct{ ScoreSampler }

// ScoreSampler samples pieces of traces by computing a signature based on spans (service, name, rsc, http.status, error.type)
// scoring it and applying a rate.
// The rates are applied on the TraceID to maximize the number of chunks with errors caught for the same traceID.
// For a set traceID: P(chunk1 kept and chunk2 kept) = min(P(chunk1 kept), P(chunk2 kept))
type ScoreSampler struct {
	*Sampler
	samplingRateKey string
	disabled        bool
	mu              sync.Mutex
	shrinkAllowList map[Signature]float64
}

// NewNoPrioritySampler returns an initialized Sampler dedicated to traces with
// no priority set.
func NewNoPrioritySampler(conf *config.AgentConfig) *NoPrioritySampler {
	s := newSampler(conf.ExtraSampleRate, conf.TargetTPS, []string{"sampler:no_priority"})
	return &NoPrioritySampler{ScoreSampler{Sampler: s, samplingRateKey: noPriorityRateKey}}
}

// NewErrorsSampler returns an initialized Sampler dedicate to errors. It behaves
// just like the the normal ScoreEngine except for its GetType method (useful
// for reporting).
func NewErrorsSampler(conf *config.AgentConfig) *ErrorsSampler {
	s := newSampler(conf.ExtraSampleRate, conf.ErrorTPS, []string{"sampler:error"})
	return &ErrorsSampler{ScoreSampler{Sampler: s, samplingRateKey: errorsRateKey, disabled: conf.ErrorTPS == 0}}
}

// Sample counts an incoming trace and tells if it is a sample which has to be kept
func (s *ScoreSampler) Sample(now time.Time, trace pb.Trace, root *pb.Span, env string) bool {
	if s.disabled {
		return false
	}

	// Extra safety, just in case one trace is empty
	if len(trace) == 0 {
		return false
	}
	signature := computeSignatureWithRootAndEnv(trace, root, env)
	signature = s.shrink(signature)
	// Update sampler state by counting this trace
	s.countWeightedSig(now, signature, weightRoot(root))

	rate := s.getSignatureSampleRate(signature)

	return s.applySampleRate(root, rate)
}

func (s *ScoreSampler) UpdateTargetTPS(targetTPS float64) {
	s.Sampler.updateTargetTPS(targetTPS)
}

func (s *ScoreSampler) GetTargetTPS() float64 {
	return s.Sampler.targetTPS.Load()
}

func (s *ScoreSampler) applySampleRate(root *pb.Span, rate float64) bool {
	initialRate := GetGlobalRate(root)
	newRate := initialRate * rate
	traceID := root.TraceID
	sampled := SampleByRate(traceID, newRate)
	if sampled {
		s.countSample()
		setMetric(root, s.samplingRateKey, rate)
	}
	return sampled
}

// shrink limits the number of signatures stored in the sampler.
// After a cardinality above shrinkCardinality/2 is reached
// signatures are spread uniformly on a fixed set of values.
// This ensures that ScoreSamplers are memory capped.
// When the shrink is triggered, previously active signatures
// stay unaffected.
// New signatures may share the same TPS computation.
func (s *ScoreSampler) shrink(sig Signature) Signature {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.size() < shrinkCardinality/2 {
		s.shrinkAllowList = nil
		return sig
	}
	if s.shrinkAllowList == nil {
		rates, _ := s.getAllSignatureSampleRates()
		s.shrinkAllowList = rates
	}
	if _, ok := s.shrinkAllowList[sig]; ok {
		return sig
	}
	return sig % (shrinkCardinality / 2)
}
