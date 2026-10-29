// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxcommon // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"

import (
	"encoding/hex"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

func ParseSpanID(spanIDStr string) (pcommon.SpanID, error) {
	var id pcommon.SpanID
	if hex.DecodedLen(len(spanIDStr)) != len(id) {
		return pcommon.SpanID{}, errors.New("span ids must be 16 hex characters")
	}
	_, err := hex.Decode(id[:], []byte(spanIDStr))
	if err != nil {
		return pcommon.SpanID{}, err
	}
	return id, nil
}

func ParseTraceID(traceIDStr string) (pcommon.TraceID, error) {
	var id pcommon.TraceID
	if hex.DecodedLen(len(traceIDStr)) != len(id) {
		return pcommon.TraceID{}, errors.New("trace ids must be 32 hex characters")
	}
	_, err := hex.Decode(id[:], []byte(traceIDStr))
	if err != nil {
		return pcommon.TraceID{}, err
	}
	return id, nil
}

func ParseProfileID(profileIDStr string) (pprofile.ProfileID, error) {
	var id pprofile.ProfileID
	if hex.DecodedLen(len(profileIDStr)) != len(id) {
		return pprofile.ProfileID{}, errors.New("profile ids must be 32 hex characters")
	}
	_, err := hex.Decode(id[:], []byte(profileIDStr))
	if err != nil {
		return pprofile.ProfileID{}, err
	}
	return id, nil
}
