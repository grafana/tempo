// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package sampler contains all the logic of the agent-side trace sampling
//
// Currently implementation is based on the scoring of the "signature" of each trace
// Based on the score, we get a sample rate to apply to the given trace
//
// Current score implementation is super-simple, it is a counter with polynomial decay per signature.
// We increment it for each incoming trace then we periodically divide the score by two every X seconds.
// Right after the division, the score is an approximation of the number of received signatures over X seconds.
// It is different from the scoring in the Agent.
//
// Since the sampling can happen at different levels (client, agent, server) or depending on different rules,
// we have to track the sample rate applied at previous steps. This way, sampling twice at 50% can result in an
// effective 25% sampling. The rate is stored as a metric in the trace root.
package sampler

import (
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
)

const (
	deprecatedRateKey = "_sampling_priority_rate_v1"
	agentRateKey      = "_dd.agent_psr"
	ruleRateKey       = "_dd.rule_psr"
)

// PrioritySampler computes priority rates per tracerEnv, service to apply in a feedback loop with trace-agent clients.
// Computed rates are sent in http responses to trace-agent. The rates are continuously adjusted in function
// of the received traffic to match a targetTPS (target traces per second).
type PrioritySampler struct {
	agentEnv string
	// sampler targetTPS is defined locally on the agent
	// This sampler tries to get the received number of sampled trace chunks/s to match its targetTPS.
	sampler *Sampler

	// rateByService contains the sampling rates in % to communicate with trace-agent clients.
	// This struct is shared with the agent API which sends the rates in http responses to spans post requests
	rateByService *RateByService
	catalog       *serviceKeyCatalog
	exit          chan struct{}
}

// NewPrioritySampler returns an initialized Sampler
func NewPrioritySampler(conf *config.AgentConfig, dynConf *DynamicConfig) *PrioritySampler {
	s := &PrioritySampler{
		agentEnv:      conf.DefaultEnv,
		sampler:       newSampler(conf.ExtraSampleRate, conf.TargetTPS, []string{"sampler:priority"}),
		rateByService: &dynConf.RateByService,
		catalog:       newServiceLookup(conf.MaxCatalogEntries),
		exit:          make(chan struct{}),
	}
	return s
}

// Start runs and block on the Sampler main loop
func (s *PrioritySampler) Start() {
	go func() {
		statsTicker := time.NewTicker(10 * time.Second)
		defer statsTicker.Stop()
		for {
			select {
			case <-statsTicker.C:
				s.sampler.report()
			case <-s.exit:
				return
			}
		}
	}()
}

func (s *PrioritySampler) UpdateTargetTPS(targetTPS float64) {
	s.sampler.updateTargetTPS(targetTPS)
}

func (s *PrioritySampler) GetTargetTPS() float64 {
	return s.sampler.targetTPS.Load()
}

// update sampling rates
func (s *PrioritySampler) updateRates() {
	s.rateByService.SetAll(s.ratesByService())
}

// Stop stops the sampler main loop
func (s *PrioritySampler) Stop() {
	close(s.exit)
}

// Sample counts an incoming trace and returns the trace sampling decision and the applied sampling rate
func (s *PrioritySampler) Sample(now time.Time, trace *pb.TraceChunk, root *pb.Span, tracerEnv string, clientDroppedP0sWeight float64) bool {
	// Extra safety, just in case one trace is empty
	if len(trace.Spans) == 0 {
		return false
	}

	samplingPriority, _ := GetSamplingPriority(trace)
	// Regardless of rates, sampling here is based on the metadata set
	// by the client library. Which, is turn, is based on agent hints,
	// but the rule of thumb is: respect client choice.
	sampled := samplingPriority > 0

	// Short-circuit and return without counting the trace in the sampling rate logic
	// if its value has not been set automatically by the client lib.
	// The feedback loop should be scoped to the values it can act upon.
	if samplingPriority < 0 {
		return sampled
	}
	if samplingPriority > 1 {
		return sampled
	}

	signature := s.catalog.register(ServiceSignature{Name: root.Service, Env: toSamplerEnv(tracerEnv, s.agentEnv)})

	// Update sampler state by counting this trace
	s.countSignature(now, root, signature, clientDroppedP0sWeight)

	if sampled {
		s.applyRate(root, signature)
		s.sampler.countSample()
	}
	return sampled
}

func (s *PrioritySampler) applyRate(root *pb.Span, signature Signature) float64 {
	if root.ParentID != 0 {
		return 1.0
	}
	// recent tracers annotate roots with applied priority rate
	// agentRateKey is set when the agent computed rate is applied
	if rate, ok := getMetric(root, agentRateKey); ok {
		return rate
	}
	// ruleRateKey is set when a tracer rule rate is applied
	if rate, ok := getMetric(root, ruleRateKey); ok {
		return rate
	}
	// slow path used by older tracer versions
	// dd-trace-go used to set the rate in deprecatedRateKey
	if rate, ok := getMetric(root, deprecatedRateKey); ok {
		return rate
	}
	rate := s.sampler.getSignatureSampleRate(signature)

	setMetric(root, deprecatedRateKey, rate)

	return rate
}

// countSignature counts all chunks received with local chunk root signature.
func (s *PrioritySampler) countSignature(now time.Time, root *pb.Span, signature Signature, clientDroppedP0Weight float64) {
	rootWeight := weightRoot(root)
	newRates := s.sampler.countWeightedSig(now, signature, rootWeight+float32(clientDroppedP0Weight))

	if newRates {
		s.updateRates()
	}
}

// ratesByService returns all rates by service, this information is useful for
// agents to pick the right service rate.
func (s *PrioritySampler) ratesByService() map[ServiceSignature]float64 {
	rates, defaultRate := s.sampler.getAllSignatureSampleRates()
	return s.catalog.ratesByService(s.agentEnv, rates, defaultRate)
}
