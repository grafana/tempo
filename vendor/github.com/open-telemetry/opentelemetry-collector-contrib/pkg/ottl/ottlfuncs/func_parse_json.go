// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"github.com/goccy/go-json"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ParseJSONArguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewParseJSONFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ParseJSON", &ParseJSONArguments[K]{}, createParseJSONFunction[K])
}

func createParseJSONFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ParseJSONArguments[K])

	if !ok {
		return nil, errors.New("ParseJSONFactory args must be of type *ParseJSONArguments[K]")
	}

	return parseJSON(args.Target), nil
}

// parseJSON returns a `pcommon.Map` or `pcommon.Slice` struct that is a result of parsing the target string as JSON
// Each JSON type is converted into a `pdata.Value` using the following map:
//
//	JSON boolean -> bool
//	JSON number  -> float64
//	JSON string  -> string
//	JSON null    -> nil
//	JSON arrays  -> pdata.SliceValue
//	JSON objects -> map[string]any
func parseJSON[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		targetVal, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		var parsedValue any
		err = json.Unmarshal([]byte(targetVal), &parsedValue)
		if err != nil {
			return nil, err
		}
		switch v := parsedValue.(type) {
		case []any:
			result := pcommon.NewSlice()
			err = result.FromRaw(v)
			return result, err
		case map[string]any:
			result := pcommon.NewMap()
			err = result.FromRaw(v)
			return result, err
		default:
			return nil, fmt.Errorf("could not convert parsed value of type %T to JSON object", v)
		}
	}
}
