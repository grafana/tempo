// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ConvertCaseArguments[K any] struct {
	Target ottl.StringGetter[K]
	ToCase string
}

func NewConvertCaseFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ConvertCase", &ConvertCaseArguments[K]{}, createConvertCaseFunction[K])
}

func createConvertCaseFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ConvertCaseArguments[K])

	if !ok {
		return nil, fmt.Errorf("ConvertCaseFactory args must be of type *ConvertCaseArguments[K]")
	}

	return convertCase(args.Target, args.ToCase)
}

func convertCase[K any](target ottl.StringGetter[K], toCase string) (ottl.ExprFunc[K], error) {
	if toCase != "lower" && toCase != "upper" && toCase != "snake" && toCase != "camel" {
		return nil, fmt.Errorf("invalid case: %s, allowed cases are: lower, upper, snake, camel", toCase)
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		if val == "" {
			return val, nil
		}

		switch toCase {
		// Convert string to lowercase (SOME_NAME -> some_name)
		case "lower":
			return strings.ToLower(val), nil

		// Convert string to uppercase (some_name -> SOME_NAME)
		case "upper":
			return strings.ToUpper(val), nil

		// Convert string to snake case (someName -> some_name)
		case "snake":
			return strcase.ToSnake(val), nil

		// Convert string to camel case (some_name -> SomeName)
		case "camel":
			return strcase.ToCamel(val), nil

		default:
			return nil, fmt.Errorf("error handling unexpected case: %s", toCase)
		}
	}, nil
}
