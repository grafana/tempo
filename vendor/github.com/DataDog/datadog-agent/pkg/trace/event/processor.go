// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package event

import (
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/sampler"
)

// Processor is responsible for all the logic surrounding extraction and sampling of APM events from processed traces.
type Processor struct {
	extractors    []Extractor
	maxEPSSampler eventSampler
}

// NewProcessor returns a new instance of Processor configured with the provided extractors and max eps limitation.
//
// Extractors will look at each span in the trace and decide whether it should be converted to an APM event or not. They
// will be tried in the provided order, with the first one returning an event stopping the chain.
//
// All extracted APM events are then submitted to sampling. This sampling is 2-fold:
//   - A first sampling step is done based on the extraction sampling rate returned by an Extractor. If an Extractor
//     returns an event accompanied with a 0.1 extraction rate, then there's a 90% chance that this event will get
//     discarded.
//   - A max events per second maxEPSSampler is applied to all non-PriorityUserKeep events that survived the first step
//     and will ensure that, in average, the total rate of events returned by the processor is not bigger than maxEPS.
func NewProcessor(extractors []Extractor, maxEPS float64) *Processor {
	return newProcessor(extractors, newMaxEPSSampler(maxEPS))
}

func newProcessor(extractors []Extractor, maxEPSSampler eventSampler) *Processor {
	return &Processor{
		extractors:    extractors,
		maxEPSSampler: maxEPSSampler,
	}
}

// Start starts the processor.
func (p *Processor) Start() {
	p.maxEPSSampler.Start()
}

// Stop stops the processor.
func (p *Processor) Stop() {
	p.maxEPSSampler.Stop()
}

// Process takes a processed trace, extracts events from it and samples them, returning a collection of
// sampled events along with the total count of extracted events.
func (p *Processor) Process(root *pb.Span, t *pb.TraceChunk) (numEvents, numExtracted int64) {
	clientSampleRate := sampler.GetClientRate(root)
	preSampleRate := sampler.GetPreSampleRate(root)
	priority := sampler.SamplingPriority(t.Priority)
	events := []*pb.Span{}

	for _, span := range t.Spans {
		extractionRate, ok := p.extract(span, priority)
		if !ok {
			continue
		}
		if !sampler.SampleByRate(span.TraceID, extractionRate) {
			continue
		}

		numExtracted++

		sampled, epsRate := p.maxEPSSample(span, priority)
		if !sampled {
			continue
		}
		// event analytics tags shouldn't be set on sampled single spans
		sampler.SetMaxEPSRate(span, epsRate)
		sampler.SetClientRate(span, clientSampleRate)
		sampler.SetPreSampleRate(span, preSampleRate)
		sampler.SetEventExtractionRate(span, extractionRate)
		sampler.SetAnalyzedSpan(span)
		if t.DroppedTrace {
			events = append(events, span)
		}
		numEvents++
	}
	if t.DroppedTrace {
		// we are not keeping anything out of this trace, except the events
		t.Spans = events
	}
	return numEvents, numExtracted
}

func (p *Processor) extract(span *pb.Span, priority sampler.SamplingPriority) (float64, bool) {
	for _, extractor := range p.extractors {
		if rate, ok := extractor.Extract(span, priority); ok {
			return rate, ok
		}
	}
	return 0, false
}

func (p *Processor) maxEPSSample(event *pb.Span, priority sampler.SamplingPriority) (sampled bool, rate float64) {
	if priority == sampler.PriorityUserKeep {
		return true, 1
	}
	return p.maxEPSSampler.Sample(event)
}

type eventSampler interface {
	Start()
	Sample(event *pb.Span) (sampled bool, rate float64)
	Stop()
}
