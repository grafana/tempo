// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package quantile

import (
	"fmt"
)

const (
	// TODO|DOC: Talk about why I choose a symmetrical number system
	uvinf    = 1<<15 - 1
	uvneginf = -uvinf

	maxKey = uvinf - 1 // 1 spot for +/- inf
)

// A Key represents a quantized version of a float64. See Config for more details
type Key int16

// A KeyCount represents a Key and an associated count
type KeyCount struct {
	k Key
	n uint
}

// IsInf returns true if the key represents +/-Inf
func (k Key) IsInf() bool {
	// TODO: bench http://graphics.stanford.edu/~seander/bithacks.html#IntegerAbs
	return k == uvinf || k == -uvneginf
}

func (k Key) String() string {
	switch k {
	case uvinf:
		return "+Inf"
	case uvneginf:
		return "-Inf"
	}

	return fmt.Sprintf("%d", k)
}

// InfKey returns the Key for +Inf if sign >= 0, -Inf if sign < 0.
func InfKey(sign int) Key {
	if sign >= 0 {
		return uvinf
	}

	return uvneginf
}
