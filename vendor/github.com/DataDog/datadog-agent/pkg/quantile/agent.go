// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package quantile

const (
	agentBufCap = 512
)

var agentConfig = Default()

// An Agent sketch is an insert optimized version of the sketch for use in the
// datadog-agent.
type Agent struct {
	Sketch   Sketch
	Buf      []Key
	CountBuf []KeyCount
}

// IsEmpty returns true if the sketch is empty
func (a *Agent) IsEmpty() bool {
	return a.Sketch.Basic.Cnt == 0 && len(a.Buf) == 0
}

// Finish flushes any pending inserts and returns a deep copy of the sketch.
func (a *Agent) Finish() *Sketch {
	a.flush()

	if a.IsEmpty() {
		return nil
	}

	return a.Sketch.Copy()
}

// flush buffered values into the sketch.
func (a *Agent) flush() {
	if len(a.Buf) != 0 {
		a.Sketch.insert(agentConfig, a.Buf)
		a.Buf = nil
	}

	if len(a.CountBuf) != 0 {
		a.Sketch.insertCounts(agentConfig, a.CountBuf)
		a.CountBuf = nil
	}
}

// Reset the agent sketch to the empty state.
func (a *Agent) Reset() {
	a.Sketch.Reset()
	a.Buf = nil // TODO: pool
}

// Insert v into the sketch.
func (a *Agent) Insert(v float64, sampleRate float64) {
	k := agentConfig.key(v)
	// bounds enforcement
	if sampleRate <= 0 || sampleRate > 1 {
		sampleRate = 1
	}

	if sampleRate == 1 {
		a.Sketch.Basic.Insert(v)
		a.Buf = append(a.Buf, k)

		if len(a.Buf) < agentBufCap {
			return
		}
	} else {
		// use truncated 1 / sampleRate as count to match histograms
		n := 1 / sampleRate
		a.Sketch.Basic.InsertN(v, n)
		kc := KeyCount{
			k: k,
			n: uint(n),
		}
		a.CountBuf = append(a.CountBuf, kc)
	}
	a.flush()
}

// InsertInterpolate linearly interpolates a count from the given lower to upper bounds
func (a *Agent) InsertInterpolate(lower float64, upper float64, count uint) {
	keys := make([]Key, 0)
	for k := agentConfig.key(lower); k <= agentConfig.key(upper); k++ {
		keys = append(keys, k)
	}
	whatsLeft := int(count)
	distance := upper - lower
	startIdx := 0
	lowerB := agentConfig.binLow(keys[startIdx])
	endIdx := 1
	var remainder float64
	for endIdx < len(keys) && whatsLeft > 0 {
		upperB := agentConfig.binLow(keys[endIdx])
		// ((upperB - lowerB) / distance) is the ratio of the distance between the current buckets to the total distance
		// which tells us how much of the remaining value to put in this bucket
		fkn := ((upperB - lowerB) / distance) * float64(count)
		// only track the remainder if fkn is >1 because we designed this to not store a bunch of 0 count buckets
		if fkn > 1 {
			remainder += fkn - float64(int(fkn))
		}
		kn := int(fkn)
		if remainder > 1 {
			kn++
			remainder--
		}
		if kn > 0 {
			// Guard against overflow at the end
			if kn > whatsLeft {
				kn = whatsLeft
			}
			a.Sketch.Basic.InsertN(lowerB, float64(kn))
			a.CountBuf = append(a.CountBuf, KeyCount{k: keys[startIdx], n: uint(kn)})
			whatsLeft -= kn
			startIdx = endIdx
			lowerB = upperB
		}
		endIdx++
	}
	if whatsLeft > 0 {
		a.Sketch.Basic.InsertN(agentConfig.binLow(keys[startIdx]), float64(whatsLeft))
		a.CountBuf = append(a.CountBuf, KeyCount{k: keys[startIdx], n: uint(whatsLeft)})
	}
	a.flush()
}
