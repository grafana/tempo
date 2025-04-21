// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type IsValidLuhnArguments[K any] struct {
	Target ottl.StringLikeGetter[K]
}

func NewIsValidLuhnFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("IsValidLuhn", &IsValidLuhnArguments[K]{}, createIsValidLuhnFunction[K])
}

func createIsValidLuhnFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*IsValidLuhnArguments[K])

	if !ok {
		return nil, errors.New("IsValidLuhnFactory args must be of type *IsValidLuhnArguments[K]")
	}

	return isValidLuhnFunc(args.Target), nil
}

func isValidLuhnFunc[K any](target ottl.StringLikeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		value, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		if value == nil {
			return nil, fmt.Errorf("invalid input: %v", value)
		}

		// first trim all spaces
		trimmedNumber := strings.ReplaceAll(*value, " ", "")

		// return false if the value is an empty string
		if len(trimmedNumber) == 0 {
			return false, nil
		}

		// extract the check digit (the right most digit)
		checkDigit, err := strconv.Atoi(string(trimmedNumber[len(trimmedNumber)-1]))
		if err != nil {
			return nil, err
		}

		sum := 0
		alternate := true
		for i := len(trimmedNumber) - 2; i >= 0; i-- {
			n, err := strconv.Atoi(string(trimmedNumber[i]))
			if err != nil {
				return nil, err
			}

			if alternate {
				// double the digit
				n *= 2
				// subtract 9 if the number is greater than 9
				if n > 9 {
					n -= 9
				}
			}
			sum += n
			alternate = !alternate
		}
		// calculate the check sum
		actualChecksum := (10 - sum%10) % 10

		return actualChecksum == checkDigit, nil
	}
}
