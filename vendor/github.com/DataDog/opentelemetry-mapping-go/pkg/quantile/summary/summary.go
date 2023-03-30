// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package summary

import (
	"fmt"
)

// A Summary stores basic incremental stats.
type Summary struct {
	// TODO: store min/max in the entries array of the summary.
	Min, Max, Sum, Avg float64

	// TODO: cnt is a duplicate of sketch.n.
	Cnt int64
}

// Reset the summary
func (s *Summary) Reset() {
	*s = Summary{}
}

func (s *Summary) String() string {
	return fmt.Sprintf("min=%.4f max=%.4f avg=%.4f sum=%.4f cnt=%d",
		s.Min, s.Max, s.Avg, s.Sum, s.Cnt)
}

// InsertN is equivalent to calling Insert(v) n times (but faster).
func (s *Summary) InsertN(v float64, n float64) {
	s.Merge(Summary{
		Cnt: int64(n),
		Sum: n * v,
		Min: v,
		Max: v,
		Avg: v,
	})
}

// Insert adds a single value to the summary.
func (s *Summary) Insert(v float64) {
	if v > s.Max || s.Cnt == 0 {
		s.Max = v
	}

	if v < s.Min || s.Cnt == 0 {
		s.Min = v
	}

	s.Cnt++
	s.Sum += v

	// incremental avg to reduce precision errors.
	s.Avg += (v - s.Avg) / float64(s.Cnt)
}

// Merge another summary into this one.
func (s *Summary) Merge(o Summary) {
	switch {
	case s.Cnt == 0:
		*s = o
		return
	case o.Cnt == 0:
		return
	}

	if o.Max > s.Max {
		s.Max = o.Max
	}

	if o.Min < s.Min {
		s.Min = o.Min
	}

	s.Cnt += o.Cnt
	s.Sum += o.Sum

	// TODO: Is there a numerically stable way of doing this.
	//   - When o.Avg and s.Avg are close in value, we lose precision.
	s.Avg = s.Avg + (o.Avg-s.Avg)*float64(o.Cnt)/float64(s.Cnt)
}
