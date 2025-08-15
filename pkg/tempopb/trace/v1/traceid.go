// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/hex"
	"errors"
	fmt "fmt"

	"github.com/gogo/protobuf/proto"
)

const traceIDSize = 16

var (
	errMarshalTraceID   = errors.New("marshal: invalid buffer length for TraceID")
	errUnmarshalTraceID = errors.New("unmarshal: invalid TraceID length")
)

// TraceID is a custom data type that is used for all trace_id fields in OTLP
// Protobuf messages.
type TraceID []byte

var _ proto.Sizer = (*TraceID)(nil)

// Size returns the size of the data to serialize.
func (tid TraceID) Size() int {
	if tid.IsEmpty() {
		return 0
	}
	return traceIDSize
}

// IsEmpty returns true if id contains at leas one non-zero byte.
func (tid TraceID) IsEmpty() bool {
	return tid == nil
}

// MarshalTo converts trace ID into a binary representation. Called by Protobuf serialization.
func (tid TraceID) MarshalTo(data []byte) (n int, err error) {
	if tid.IsEmpty() {
		return 0, nil
	}

	if len(data) < traceIDSize {
		return 0, errMarshalTraceID
	}

	return copy(data, tid), nil
}

// Unmarshal inflates this trace ID from binary representation. Called by Protobuf serialization.
func (tid *TraceID) Unmarshal(data []byte) error {
	if len(data) == 0 {
		*tid = []byte{}
		return nil
	}

	if len(data) != traceIDSize {
		return errUnmarshalTraceID
	}

	*tid = data
	return nil
}

func (tid TraceID) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "\"%s\"", hex.EncodeToString(tid)), nil
}

func (tid *TraceID) UnmarshalJSON(data []byte) error {
	if hex.DecodedLen(len(data)-2) != traceIDSize {
		return errors.New("length mismatch")
	}

	*tid = make([]byte, traceIDSize)
	_, err := hex.Decode(*tid, data[1:len(data)-1])
	return err
}
