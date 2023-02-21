// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package event

import (
	"sync"
	"time"
)

// memoryBackend storing any state required to run the sampling algorithms.
//
// Current implementation is only based on counters with polynomial decay.
// Its bias with steady counts is 1 * decayFactor.
// The stored scores represent approximation of the real count values (with a countScaleFactor factor).
type memoryBackend struct {
	// sampledScore is the score of all sampled traces.
	sampledScore float64

	// mu is a lock protecting all the scores.
	mu sync.RWMutex

	// DecayPeriod is the time period between each score decay.
	// A lower value is more reactive, but forgets quicker.
	decayPeriod time.Duration

	// decayFactor is how much we reduce/divide the score at every decay run.
	// A lower value is more reactive, but forgets quicker.
	decayFactor float64

	// countScaleFactor is the factor to apply to move from the score
	// to the representing number of traces per second.
	// By definition of the decay formula is:
	// countScaleFactor = (decayFactor / (decayFactor - 1)) * DecayPeriod
	// It also represents by how much a spike is smoothed: if we instantly
	// receive N times the same signature, its immediate count will be
	// increased by N / countScaleFactor.
	countScaleFactor float64
}

// newMemoryBackend returns an initialized Backend.
func newMemoryBackend() *memoryBackend {
	decayPeriod := 1 * time.Second
	decayFactor := 1.125
	return &memoryBackend{
		decayPeriod:      decayPeriod,
		decayFactor:      decayFactor,
		countScaleFactor: (decayFactor / (decayFactor - 1)) * decayPeriod.Seconds(),
	}
}

// countSample counts a trace sampled by the sampler.
func (b *memoryBackend) countSample() {
	b.mu.Lock()
	b.sampledScore++
	b.mu.Unlock()
}

// getSampledScore returns the global score of all sampled traces.
func (b *memoryBackend) getSampledScore() float64 {
	b.mu.RLock()
	score := b.sampledScore / b.countScaleFactor
	b.mu.RUnlock()

	return score
}

// getUpperSampledScore returns a certain upper bound of the global count of all sampled traces.
func (b *memoryBackend) getUpperSampledScore() float64 {
	// Overestimate the real score with the high limit of the backend bias.
	return b.getSampledScore() * b.decayFactor
}

// decayScore applies the decay to the rolling counters.
func (b *memoryBackend) decayScore() {
	b.mu.Lock()
	b.sampledScore /= b.decayFactor
	b.mu.Unlock()
}
