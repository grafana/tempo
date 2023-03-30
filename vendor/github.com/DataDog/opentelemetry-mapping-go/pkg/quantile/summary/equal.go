// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package summary

import (
	"fmt"
	"math"
)

const (
	ulpLimit = 128
)

// ulpDistance is the absolute difference in units of least precision.
//
// Special cases are (order of arguments doesn't matter):
//  ulpDistance(NaN, b) = max
//  ulpDistance(+Inf, -Inf) = max
//  ulpDistance(+Inf, +Inf) = 0
//  ulpDistance(-Inf, -Inf) = 0
func ulpDistance(a, b float64) uint64 {
	switch {
	case a == b:
		return 0
	case math.IsInf(a, 0) || math.IsInf(b, 0):
		return math.MaxUint64
	case math.IsNaN(a) || math.IsNaN(b):
		return math.MaxUint64
	case math.Signbit(a) != math.Signbit(b):
		return math.Float64bits(math.Abs(a)) + math.Float64bits(math.Abs(b))
	}

	x, y := math.Float64bits(a), math.Float64bits(b)
	if x > y {
		return x - y
	}

	return y - x
}

func checkFloat64Equal(name string, a, e float64) error {
	ulp := ulpDistance(a, e)
	if ulp <= ulpLimit {
		return nil
	}

	return fmt.Errorf("%s: (act) %g != %g (exp) ❌ ulp=%d limit=%d",
		name, a, e, ulp, ulpLimit)

}

func checkIntEqual(name string, a, e int) error {
	if a != e {
		return fmt.Errorf("%s: (act) %v != %v (exp) ❌", name, a, e)
	}

	return nil
}

// CheckEqual returns an error if the summaries are not equal
func CheckEqual(a, e Summary) error {
	if err := checkIntEqual("Count", int(a.Cnt), int(e.Cnt)); err != nil {
		return err
	}

	if err := checkFloat64Equal("Min", a.Min, e.Min); err != nil {
		return err
	}

	if err := checkFloat64Equal("Max", a.Max, e.Max); err != nil {
		return err
	}

	if err := checkFloat64Equal("Sum", a.Sum, e.Sum); err != nil {
		return err
	}

	return checkFloat64Equal("Avg", a.Avg, e.Avg)
}
