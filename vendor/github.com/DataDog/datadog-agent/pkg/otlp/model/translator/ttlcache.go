// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translator

import (
	"time"

	gocache "github.com/patrickmn/go-cache"
)

type ttlCache struct {
	cache *gocache.Cache
}

// numberCounter keeps the value of a number
// monotonic counter at a given point in time
type numberCounter struct {
	ts      uint64
	startTs uint64
	value   float64
}

func newTTLCache(sweepInterval int64, deltaTTL int64) *ttlCache {
	cache := gocache.New(time.Duration(deltaTTL)*time.Second, time.Duration(sweepInterval)*time.Second)
	return &ttlCache{cache}
}

// Diff submits a new value for a given non-monotonic metric and returns the difference with the
// last submitted value (ordered by timestamp). The diff value is only valid if `ok` is true.
func (t *ttlCache) Diff(dimensions *Dimensions, startTs, ts uint64, val float64) (float64, bool) {
	return t.putAndGetDiff(dimensions, false, startTs, ts, val)
}

// MonotonicDiff submits a new value for a given monotonic metric and returns the difference with the
// last submitted value (ordered by timestamp). The diff value is only valid if `ok` is true.
func (t *ttlCache) MonotonicDiff(dimensions *Dimensions, startTs, ts uint64, val float64) (float64, bool) {
	return t.putAndGetDiff(dimensions, true, startTs, ts, val)
}

// putAndGetDiff submits a new value for a given metric and returns the difference with the
// last submitted value (ordered by timestamp). The diff value is only valid if `ok` is true.
func (t *ttlCache) putAndGetDiff(
	dimensions *Dimensions,
	monotonic bool,
	startTs, ts uint64,
	val float64,
) (dx float64, ok bool) {
	key := dimensions.String()
	if c, found := t.cache.Get(key); found {
		cnt := c.(numberCounter)
		if cnt.ts > ts {
			// We were given a point older than the one in memory so we drop it
			// We keep the existing point in memory since it is the most recent
			return 0, false
		}
		dx = val - cnt.value

		// Determine if this is the first point on a cumulative series:
		// https://github.com/open-telemetry/opentelemetry-specification/blob/v1.7.0/specification/metrics/datamodel.md#resets-and-gaps
		//
		// This is written down as an 'if' because I feel it is easier to understand than with a boolean expression.
		if startTs == 0 {
			// We don't know the start time, assume the sequence has not been restarted.
			ok = true
		} else if startTs != ts && startTs == cnt.startTs {
			// Since startTs != 0 we know the start time, thus we apply the following rules from the spec:
			//  - "When StartTimeUnixNano equals TimeUnixNano, a new unbroken sequence of observations begins with a reset at an unknown start time."
			//  - "[for cumulative series] the StartTimeUnixNano of each point matches the StartTimeUnixNano of the initial observation."
			ok = true
		}

		// If sequence is monotonic and diff is negative, there has been a reset.
		// This must never happen if we know the startTs; we also override the value in this case.
		if monotonic && dx < 0 {
			ok = false
		}
	}

	t.cache.Set(
		key,
		numberCounter{
			startTs: startTs,
			ts:      ts,
			value:   val,
		},
		gocache.DefaultExpiration,
	)
	return
}
