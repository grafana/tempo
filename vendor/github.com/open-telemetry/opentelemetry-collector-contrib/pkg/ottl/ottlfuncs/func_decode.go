// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"encoding/base64"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/textutils"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type DecodeArguments[K any] struct {
	Target   ottl.Getter[K]
	Encoding string
}

func NewDecodeFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Decode", &DecodeArguments[K]{}, createDecodeFunction[K])
}

func createDecodeFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*DecodeArguments[K])
	if !ok {
		return nil, fmt.Errorf("DecodeFactory args must be of type *DecodeArguments[K]")
	}

	return Decode(args.Target, args.Encoding)
}

func Decode[K any](target ottl.Getter[K], encoding string) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		var stringValue string

		switch v := val.(type) {
		case []byte:
			stringValue = string(v)
		case *string:
			stringValue = *v
		case string:
			stringValue = v
		case pcommon.ByteSlice:
			stringValue = string(v.AsRaw())
		case *pcommon.ByteSlice:
			stringValue = string(v.AsRaw())
		case pcommon.Value:
			stringValue = v.AsString()
		case *pcommon.Value:
			stringValue = v.AsString()
		default:
			return nil, fmt.Errorf("unsupported type provided to Decode function: %T", v)
		}

		switch encoding {
		// base64 is not in IANA index, so we have to deal with this encoding separately
		case "base64":
			return decodeBase64(base64.StdEncoding, stringValue)
		case "base64-raw":
			return decodeBase64(base64.RawStdEncoding, stringValue)
		case "base64-url":
			return decodeBase64(base64.URLEncoding, stringValue)
		case "base64-raw-url":
			return decodeBase64(base64.RawURLEncoding, stringValue)
		default:
			e, err := textutils.LookupEncoding(encoding)
			if err != nil {
				return nil, err
			}

			decodedString, err := e.NewDecoder().String(stringValue)
			if err != nil {
				return nil, fmt.Errorf("could not decode: %w", err)
			}

			return decodedString, nil
		}
	}, nil
}

func decodeBase64(encoding *base64.Encoding, stringValue string) (any, error) {
	decodedBytes, err := encoding.DecodeString(stringValue)
	if err != nil {
		return nil, fmt.Errorf("could not decode: %w", err)
	}
	return string(decodedBytes), nil
}
