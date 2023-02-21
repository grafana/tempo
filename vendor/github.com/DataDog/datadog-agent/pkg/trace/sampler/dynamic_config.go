// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package sampler

import (
	"math"
	"strconv"
	"sync"
	"time"

	"go.uber.org/atomic"
)

// DynamicConfig contains configuration items which may change
// dynamically over time.
type DynamicConfig struct {
	// RateByService contains the rate for each service/env tuple,
	// used in priority sampling by client libs.
	RateByService RateByService
}

// NewDynamicConfig creates a new dynamic config object which maps service signatures
// to their corresponding sampling rates. Each service will have a default assigned
// matching the service rate of the specified env.
func NewDynamicConfig() *DynamicConfig {
	return &DynamicConfig{RateByService: RateByService{}}
}

// State specifies the current state of DynamicConfig
type State struct {
	Rates   map[string]float64
	Version string
}

// rc specifies a pair of rate and color.
// color is used for detecting changes.
type rc struct {
	r float64
	c int8
}

// RateByService stores the sampling rate per service. It is thread-safe, so
// one can read/write on it concurrently, using getters and setters.
type RateByService struct {
	mu sync.RWMutex // guards rates
	// currentColor is either 0 or 1. And, it changes every time `SetAll()` is called.
	// When `SetAll()` is called, we paint affected keys with `currentColor`.
	// If there is a key has a color doesn't match `currentColor`, it means that key no longer exists.
	currentColor int8
	rates        map[string]*rc
	version      string
}

// SetAll the sampling rate for all services. If a service/env is not
// in the map, then the entry is removed.
func (rbs *RateByService) SetAll(rates map[ServiceSignature]float64) {
	rbs.mu.Lock()
	defer rbs.mu.Unlock()

	rbs.currentColor = 1 - rbs.currentColor
	changed := false
	if rbs.rates == nil {
		rbs.rates = make(map[string]*rc, len(rates))
	}
	for s, r := range rates {
		ks := s.String()
		r = math.Min(math.Max(r, 0), 1)
		if oldV, ok := rbs.rates[ks]; !ok || oldV.r != r {
			changed = true
			rbs.rates[ks] = &rc{
				r: r,
			}
		}
		rbs.rates[ks].c = rbs.currentColor
	}
	for k, v := range rbs.rates {
		if v.c != rbs.currentColor {
			changed = true
			delete(rbs.rates, k)
		}
	}
	if changed {
		rbs.version = newVersion()
	}
}

// GetNewState returns the current state if the given version is different from the local version.
func (rbs *RateByService) GetNewState(version string) State {
	rbs.mu.RLock()
	defer rbs.mu.RUnlock()

	if version != "" && version == rbs.version {
		return State{
			Version: version,
		}
	}
	ret := State{
		Rates:   make(map[string]float64, len(rbs.rates)),
		Version: rbs.version,
	}
	for k, v := range rbs.rates {
		ret.Rates[k] = v.r
	}

	return ret
}

var localVersion atomic.Int64

func newVersion() string {
	return strconv.FormatInt(time.Now().Unix(), 16) + "-" + strconv.FormatInt(localVersion.Inc(), 16)
}
