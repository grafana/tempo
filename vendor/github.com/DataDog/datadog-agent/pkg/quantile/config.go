// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package quantile

import (
	"fmt"
	"math"
)

const (
	defaultBinLimit = 4096
	defaultEps      = 1.0 / 128.0
	defaultMin      = 1e-9
)

// A Config struct is passed around to many sketches (read-only).
type Config struct {
	binLimit int

	// TODO: interpolation type enum (e.g. https://github.com/gee-go/util/blob/ec29b7754/vec/quantile.go#L13-L29)

	gamma struct {
		// ln is the natural log of v, used to speed up calculating log base gamma
		v, ln float64
	}

	norm struct {
		// min and max values representable by a sketch with these params.
		//
		// key(x) =
		//    0 : -min > x < min
		//    1 : x == min
		//   -1 : x == -min
		// +Inf : x > max
		// -Inf : x < -max.
		min, max float64
		emin     int

		// bias of the exponent, used to ensure key(x) >= 1
		bias int
	}
}

// MaxCount returns the max number of values you can insert.
// This is limited by using a uint16 for bin.n
func (c *Config) MaxCount() int {
	return c.binLimit * math.MaxUint16
}

// f64 returns the lower bound for the given key: γ^k
func (c *Config) f64(k Key) float64 {
	switch {
	case k < 0:
		return -c.f64(-k)
	case k.IsInf():
		return math.Inf(int(k))
	case k == 0:
		return 0
	}

	exp := float64(int(k) - c.norm.bias)
	return c.powGamma(exp)
}

func (c *Config) binLow(k Key) float64 {
	switch {
	case k < 0:
		return -c.f64(-k)
	case k.IsInf():
		return math.Inf(int(k))
	case k == 0:
		return 0
	}

	exp := float64(int(k) - c.norm.bias)
	return c.powGamma(exp)
}

// key returns a value k such that:
//
//	γ^k <= v < γ^(k+1)
func (c *Config) key(v float64) Key {
	switch {
	case v < 0:
		return -c.key(-v)
	case v == 0, v > 0 && v < c.norm.min, v < 0 && v > -c.norm.min:
		return 0
	}

	// RoundToEven is used so that key(f64(k)) == k.
	i := int(math.RoundToEven(c.logGamma(v))) + c.norm.bias
	if i > maxKey {
		return InfKey(1)
	}

	if i < 1 {
		// this should not happen, but better be safe than sorry
		return Key(1)
	}

	return Key(i)
}

func (c *Config) logGamma(v float64) float64 {
	return math.Log(v) / c.gamma.ln
}

func (c *Config) powGamma(y float64) float64 {
	return math.Pow(c.gamma.v, y)
}

// Default returns the default config.
func Default() *Config {
	c, err := NewConfig(0, 0, 0)
	if err != nil {
		panic(err)
	}

	return c
}

func (c *Config) refresh(eps, minValue float64) error {
	// (1) gamma
	// TODO: Calc via epsilon:
	//  (1) gamma.v = 1 + 2 * eps
	//  (2) gamma.log = math.Log1p(2 * eps) // more accurate for numbers close to 1.
	switch {
	case eps == 0:
		eps = defaultEps
	case eps > 1 || eps < 0:
		return fmt.Errorf("%g: eps must be between 0 and 1", eps)
	}

	eps *= 2
	c.gamma.v = 1 + eps
	c.gamma.ln = math.Log1p(eps)

	// (2) norm
	// pick the next smaller power of gamma for min
	// this value will have a key of 1.
	switch {
	case minValue == 0:
		minValue = defaultMin
	case minValue < 0:
		return fmt.Errorf("%g: min must be > 0", minValue)
	}

	emin := int(math.Floor(c.logGamma(minValue)))

	// TODO: error when bias doesn't fit in an int16
	c.norm.bias = -emin + 1
	c.norm.emin = emin

	// TODO: double check c.key(c.f64(1)) == 1
	c.norm.min = c.f64(1)
	c.norm.max = c.f64(maxKey)

	// sanity check, should never happen
	if c.norm.min > minValue {
		return fmt.Errorf("%g > %g", c.norm.min, minValue)
	}

	return nil
}

// NewConfig creates a config object with.
// TODO|DOC: describe params
func NewConfig(eps, min float64, binLimit int) (*Config, error) {
	c := &Config{}

	switch {
	case binLimit == 0:
		binLimit = defaultBinLimit
	case binLimit < 0:
		return nil, fmt.Errorf("binLimit can't be negative: %d", binLimit)
	}
	c.binLimit = binLimit

	if err := c.refresh(eps, min); err != nil {
		return nil, err
	}

	return c, nil
}
