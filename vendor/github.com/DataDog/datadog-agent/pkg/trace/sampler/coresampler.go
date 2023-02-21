// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package sampler

import (
	"sort"
	"sync"
	"time"

	"go.uber.org/atomic"

	"github.com/DataDog/datadog-agent/pkg/trace/metrics"
	"github.com/DataDog/datadog-agent/pkg/trace/watchdog"
)

const (
	bucketDuration  = 5 * time.Second
	numBuckets      = 6
	maxRateIncrease = 1.2
)

// Sampler is the main component of the sampling logic
// Seen traces are counted per signature in a circular buffer
// of numBuckets.
// The sampler distributes uniformly on all signature
// a targetTPS. The bucket with the maximum counts over the period
// of the buffer is used to compute the sampling rates.
type Sampler struct {
	// seen counts seen signatures by Signature in a circular buffer of numBuckets of bucketDuration.
	// In the case of the PrioritySampler, chunks dropped in the Client are also taken in account.
	seen map[Signature][numBuckets]float32
	// allSigsSeen counts all signatures in a circular buffer of numBuckets of bucketDuration
	allSigsSeen [numBuckets]float32
	// lastBucketID is the index of the last bucket on which traces were counted
	lastBucketID int64
	// rates maps sampling rate in %
	rates map[Signature]float64
	// lowestRate is the lowest rate of all signatures
	lowestRate float64

	// muSeen is a lock protecting seen map and totalSeen count
	muSeen sync.RWMutex
	// muRates is a lock protecting rates map
	muRates sync.RWMutex

	// Maximum limit to the total number of traces per second to sample
	targetTPS *atomic.Float64
	// extraRate is an extra raw sampling rate to apply on top of the sampler rate
	extraRate float64

	totalSeen float32
	totalKept *atomic.Int64

	tags    []string
	exit    chan struct{}
	stopped chan struct{}
}

// newSampler returns an initialized Sampler
func newSampler(extraRate float64, targetTPS float64, tags []string) *Sampler {
	s := &Sampler{
		seen: make(map[Signature][numBuckets]float32),

		extraRate: extraRate,
		targetTPS: atomic.NewFloat64(targetTPS),
		tags:      tags,

		totalKept: atomic.NewInt64(0),

		exit:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	return s
}

// updateTargetTPS updates the targetTPS and all rates
func (s *Sampler) updateTargetTPS(targetTPS float64) {
	previousTargetTPS := s.targetTPS.Load()
	s.targetTPS.Store(targetTPS)

	if previousTargetTPS == 0 {
		return
	}
	ratio := targetTPS / previousTargetTPS

	s.muRates.Lock()
	for sig, rate := range s.rates {
		newRate := rate * ratio
		if newRate > 1 {
			newRate = 1
		}
		s.rates[sig] = newRate
	}
	s.muRates.Unlock()
}

// Start runs and the Sampler main loop
func (s *Sampler) Start() {
	go func() {
		defer watchdog.LogOnPanic()
		statsTicker := time.NewTicker(10 * time.Second)
		defer statsTicker.Stop()
		for {
			select {
			case <-statsTicker.C:
				s.report()
			case <-s.exit:
				close(s.stopped)
				return
			}
		}
	}()
}

// countWeightedSig counts a trace sampled by the sampler and update rates
// if buckets are rotated
func (s *Sampler) countWeightedSig(now time.Time, signature Signature, n float32) bool {
	bucketID := now.Unix() / int64(bucketDuration.Seconds())
	s.muSeen.Lock()
	prevBucketID := s.lastBucketID
	s.lastBucketID = bucketID

	// pass through each bucket, zero expired ones and adjust sampling rates
	updateRates := prevBucketID != bucketID
	if updateRates {
		s.updateRates(prevBucketID, bucketID)
	}

	buckets, ok := s.seen[signature]
	if !ok {
		buckets = [numBuckets]float32{}
	}
	s.allSigsSeen[bucketID%numBuckets] += n
	buckets[bucketID%numBuckets] += n
	s.seen[signature] = buckets

	s.totalSeen += n
	s.muSeen.Unlock()
	return updateRates
}

// updateRates distributes TPS on each signature and apply it to the moving
// max of seen buckets.
// Rates increase are bounded by 20% increases, it requires 13 evaluations (1.2**13 = 10.6)
// to increase a sampling rate by 10 fold in about 1min.
// A caller of updateRates must hold a lock on s.muSeen (e.g. as used by countWeightedSig).
func (s *Sampler) updateRates(previousBucket, newBucket int64) {
	if len(s.seen) == 0 {
		return
	}
	rates := make(map[Signature]float64, len(s.seen))

	seenTPSs := make([]float64, 0, len(s.seen))
	sigs := make([]Signature, 0, len(s.seen))
	for sig, buckets := range s.seen {
		maxBucket, buckets := zeroAndGetMax(buckets, previousBucket, newBucket)
		s.seen[sig] = buckets
		seenTPSs = append(seenTPSs, float64(maxBucket)/bucketDuration.Seconds())
		sigs = append(sigs, sig)
	}
	_, allSigsSeen := zeroAndGetMax(s.allSigsSeen, previousBucket, newBucket)
	s.allSigsSeen = allSigsSeen

	tpsPerSig := computeTPSPerSig(s.targetTPS.Load(), seenTPSs)

	s.muRates.Lock()
	defer s.muRates.Unlock()
	s.lowestRate = 1
	for i, sig := range sigs {
		seenTPS := seenTPSs[i]
		rate := 1.0
		if tpsPerSig < seenTPS && seenTPS > 0 {
			rate = tpsPerSig / seenTPS
		}
		// capping increase rate to 20%
		if prevRate, ok := s.rates[sig]; ok && prevRate != 0 {
			if rate/prevRate > maxRateIncrease {
				rate = prevRate * maxRateIncrease
			}
		}
		if rate > 1.0 {
			rate = 1.0
		}
		// no traffic on this signature, clean it up from the sampler
		if rate == 1.0 && seenTPS == 0 {
			delete(s.seen, sig)
			continue
		}
		if rate < s.lowestRate {
			s.lowestRate = rate
		}
		rates[sig] = rate
	}
	s.rates = rates
}

// computeTPSPerSig distributes TPS looking at the seenTPS of all signatures.
// By default it spreads uniformly the TPS on all signatures. If a signature
// is low volume and does not use all of its TPS, the remaining is spread uniformly
// on all other signatures.
func computeTPSPerSig(targetTPS float64, seen []float64) float64 {
	sorted := make([]float64, len(seen))
	copy(sorted, seen)
	sort.Float64s(sorted)

	sigTarget := targetTPS / float64(len(sorted))

	for i, c := range sorted {
		if c >= sigTarget || i == len(sorted)-1 {
			break
		}
		targetTPS -= c
		sigTarget = targetTPS / float64((len(sorted) - i - 1))
	}
	return sigTarget
}

// zeroAndGetMax zeroes expired buckets and returns the max count
func zeroAndGetMax(buckets [numBuckets]float32, previousBucket, newBucket int64) (float32, [numBuckets]float32) {
	maxBucket := float32(0)
	for i := previousBucket + 1; i <= previousBucket+numBuckets; i++ {
		index := i % numBuckets

		// if a complete rotation happened between previousBucket and newBucket
		// all buckets will be zeroed
		if i < newBucket {
			buckets[index] = 0
			continue
		}

		value := buckets[index]
		if value > maxBucket {
			maxBucket = value
		}

		// zeroing after taking in account the previous value of the bucket
		// overridden by this rotation. This allows to take in account all buckets
		if i == newBucket {
			buckets[index] = 0
		}
	}
	return maxBucket, buckets
}

// countSample counts a trace sampled by the sampler.
func (s *Sampler) countSample() {
	s.totalKept.Inc()
}

// getSignatureSampleRate returns the sampling rate to apply to a signature
func (s *Sampler) getSignatureSampleRate(sig Signature) float64 {
	s.muRates.RLock()
	rate, ok := s.rates[sig]
	s.muRates.RUnlock()
	if !ok {
		return s.defaultRate()
	}
	return rate * s.extraRate
}

// getAllSignatureSampleRates returns the sampling rate to apply to each signature
func (s *Sampler) getAllSignatureSampleRates() (map[Signature]float64, float64) {
	s.muRates.RLock()
	rates := make(map[Signature]float64, len(s.rates))
	for sig, val := range s.rates {
		rates[sig] = val * s.extraRate
	}
	s.muRates.RUnlock()
	return rates, s.defaultRate()
}

// defaultRate returns the rate to apply to unknown signatures. It's computed by considering
// the moving max of all Sigs seen by the sampler, and the lowest rate stored.
// Callers of defaultRate must hold a RLock on s.muRates
func (s *Sampler) defaultRate() float64 {
	targetTPS := s.targetTPS.Load()
	if targetTPS == 0 {
		return 0
	}

	var maxSeen float32
	s.muSeen.RLock()
	defer s.muSeen.RUnlock()
	for _, c := range s.allSigsSeen {
		if c > maxSeen {
			maxSeen = c
		}
	}
	seenTPS := float64(maxSeen) / bucketDuration.Seconds()

	rate := 1.0
	if targetTPS < seenTPS && seenTPS > 0 {
		rate = targetTPS / seenTPS
	}
	if s.lowestRate < rate && s.lowestRate != 0 {
		return s.lowestRate
	}
	return rate
}

func (s *Sampler) size() int64 {
	s.muSeen.RLock()
	defer s.muSeen.RUnlock()
	return int64(len(s.seen))
}

func (s *Sampler) report() {
	s.muSeen.Lock()
	seen := int64(s.totalSeen)
	s.totalSeen = 0
	s.muSeen.Unlock()
	kept := s.totalKept.Swap(0)
	metrics.Count("datadog.trace_agent.sampler.kept", kept, s.tags, 1)
	metrics.Count("datadog.trace_agent.sampler.seen", seen, s.tags, 1)
	metrics.Gauge("datadog.trace_agent.sampler.size", float64(s.size()), s.tags, 1)
}

// Stop stops the main Run loop
func (s *Sampler) Stop() {
	close(s.exit)
	<-s.stopped
}
