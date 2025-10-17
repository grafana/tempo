// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/hex"
	"errors"
	fmt "fmt"

	"github.com/gogo/protobuf/proto"
)

const spanIDSize = 8

var (
	errMarshalSpanID   = errors.New("marshal: invalid buffer length for SpanID")
	errUnmarshalSpanID = errors.New("unmarshal: invalid SpanID length")
)

// SpanID is a custom data type that is used for all span_id fields in OTLP
// Protobuf messages.
type SpanID []byte

var _ proto.Sizer = (*SpanID)(nil)

// Size returns the size of the data to serialize.
func (sid SpanID) Size() int {
	if sid.IsEmpty() {
		return 0
	}
	return spanIDSize
}

// IsEmpty returns true if id contains at least one non-zero byte.
func (sid SpanID) IsEmpty() bool {
	return sid == nil
}

// MarshalTo converts trace ID into a binary representation. Called by Protobuf serialization.
func (sid SpanID) MarshalTo(data []byte) (n int, err error) {
	if sid.IsEmpty() {
		return 0, nil
	}

	if len(data) < spanIDSize {
		return 0, errMarshalSpanID
	}

	return copy(data, sid), nil
}

// Unmarshal inflates this trace ID from binary representation. Called by Protobuf serialization.
func (sid *SpanID) Unmarshal(data []byte) error {
	if len(data) == 0 {
		*sid = []byte{}
		return nil
	}

	if len(data) != spanIDSize {
		return errUnmarshalSpanID
	}

	*sid = data
	return nil
}
func (sid SpanID) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "\"%s\"", hex.EncodeToString(sid)), nil
}

func (sid *SpanID) UnmarshalJSON(data []byte) error {
	if hex.DecodedLen(len(data)-2) != spanIDSize {
		return errors.New("length mismatch")
	}

	*sid = make([]byte, traceIDSize)
	_, err := hex.Decode(*sid, data[1:len(data)-1])
	return err
}
