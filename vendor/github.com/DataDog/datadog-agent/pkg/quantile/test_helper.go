// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build test
// +build test

package quantile

import (
	"math"
)

func almostEqual(a, b, e float64) bool {
	return math.Abs((a-b)/a) <= e
}

// SketchesApproxEqual checks whether two SketchSeries are equal
func SketchesApproxEqual(exp, act *Sketch, e float64) bool {

	if !almostEqual(exp.Basic.Sum, act.Basic.Sum, e) {
		return false
	}

	if !almostEqual(exp.Basic.Avg, act.Basic.Avg, e) {
		return false
	}

	if !almostEqual(exp.Basic.Max, act.Basic.Max, e) {
		return false
	}

	if !almostEqual(exp.Basic.Min, act.Basic.Min, e) {
		return false
	}

	if exp.Basic.Cnt != exp.Basic.Cnt {
		return false
	}

	if exp.count != act.count {
		return false
	}

	if len(exp.bins) != len(act.bins) {
		return false
	}

	for i := range exp.bins {
		if math.Abs(float64(act.bins[i].k-exp.bins[i].k)) > 1 {
			return false
		}

		if act.bins[i].n != exp.bins[i].n {
			return false
		}
	}

	return true
}

type tHelper interface {
	Helper()
}
