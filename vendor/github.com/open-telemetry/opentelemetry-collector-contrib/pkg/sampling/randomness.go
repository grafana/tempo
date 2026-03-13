// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sampling // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"

import (
	"encoding/binary"
	"errors"
	"strconv"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// numRandomnessValues equals MaxAdjustedCount--this variable has been
// introduced to improve readability.  Recall that MaxAdjustedCount is
// 2**56 which is one greater than the maximum RValue
// ("ffffffffffffff", i.e., "100000000000000").
const numRandomnessValues = MaxAdjustedCount

// ErrRValueSize is returned by RValueToRandomess in case of
// unexpected size.
var ErrRValueSize = errors.New("r-value must have 14 hex digits")

// leastHalfTraceIDThresholdMask is the mask to use on the
// least-significant half of the TraceID, i.e., bytes 8-15.
// Because this is a 56 bit mask, the result after masking is
// the unsigned value of bytes 9 through 15.
//
// This helps extract 56 bits of randomness from the second half of
// the TraceID, as specified in https://www.w3.org/TR/trace-context-2/#randomness-of-trace-id
const leastHalfTraceIDThresholdMask = MaxAdjustedCount - 1

// AllProbabilitiesRandomness is sampled at all probabilities.
var AllProbabilitiesRandomness = Randomness{unsigned: numRandomnessValues - 1}

// Randomness may be derived from R-value or TraceID.
//
// Randomness contains 56 bits of randomness, derived in one of two ways, see:
// https://www.w3.org/TR/trace-context-2/#randomness-of-trace-id
type Randomness struct {
	// unsigned is in the range [0, MaxAdjustedCount-1]
	unsigned uint64
}

// TraceIDToRandomness returns randomness from a TraceID (assumes
// the traceparent random flag was set).
func TraceIDToRandomness(id pcommon.TraceID) Randomness {
	// To get the 56 bits we want, take the second half of the trace ID,
	leastHalf := binary.BigEndian.Uint64(id[8:])
	return Randomness{
		// Then apply the mask to get the least-significant 56 bits / 7 bytes.
		// Equivalently stated: zero the most-significant 8 bits.
		unsigned: leastHalf & leastHalfTraceIDThresholdMask,
	}
}

// RValueToRandomness parses NumHexDigits hex bytes into a Randomness.
func RValueToRandomness(s string) (Randomness, error) {
	if len(s) != NumHexDigits {
		return Randomness{}, ErrRValueSize
	}

	unsigned, err := strconv.ParseUint(s, hexBase, 64)
	if err != nil {
		return Randomness{}, err
	}

	return Randomness{
		unsigned: unsigned,
	}, nil
}

// RValue formats the r-value encoding.
func (rnd Randomness) RValue() string {
	// The important part here is to format a full 14-byte hex
	// string, including leading zeros.  We could accomplish the
	// same with custom code or with fmt.Sprintf directives, but
	// here we let strconv.FormatUint fill in leading zeros, as
	// follows:
	//
	//   Format (numRandomnessValues+Randomness) as a hex string
	//   Strip the leading hex digit, which is a "1" by design
	//
	// For example, a randomness that requires two leading zeros
	// (all in hexadecimal):
	//
	//   randomness is 7 bytes:             aabbccddeeff
	//   numRandomnessValues is 2^56:    100000000000000
	//   randomness+numRandomnessValues: 100aabbccddeeff
	//   strip the leading "1":           00aabbccddeeff
	//
	// If the value is out-of-range, the empty string will be
	// returned.
	if rnd.unsigned >= numRandomnessValues {
		return ""
	}
	return strconv.FormatUint(numRandomnessValues+rnd.unsigned, hexBase)[1:]
}

// Unsigned returns the unsigned representation of the random value.
// Items of data SHOULD be sampled when:
//
//	Threshold.Unsigned() <= // Randomness.Unsigned().
func (rnd Randomness) Unsigned() uint64 {
	return rnd.unsigned
}

// UnsignedToRandomness constructs a randomness using 56 random bits
// of unsigned number.  If the input is out of range, an invalid value
// will be returned with an error.
func UnsignedToRandomness(x uint64) (Randomness, error) {
	if x >= MaxAdjustedCount {
		return AllProbabilitiesRandomness, ErrRValueSize
	}
	return Randomness{unsigned: x}, nil
}
