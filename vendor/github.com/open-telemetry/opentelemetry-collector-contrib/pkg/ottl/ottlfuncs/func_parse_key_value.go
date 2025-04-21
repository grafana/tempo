// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/parseutils"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ParseKeyValueArguments[K any] struct {
	Target        ottl.StringGetter[K]
	Delimiter     ottl.Optional[string]
	PairDelimiter ottl.Optional[string]
}

func NewParseKeyValueFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ParseKeyValue", &ParseKeyValueArguments[K]{}, createParseKeyValueFunction[K])
}

func createParseKeyValueFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ParseKeyValueArguments[K])

	if !ok {
		return nil, errors.New("ParseKeyValueFactory args must be of type *ParseKeyValueArguments[K]")
	}

	return parseKeyValue[K](args.Target, args.Delimiter, args.PairDelimiter)
}

func parseKeyValue[K any](target ottl.StringGetter[K], d ottl.Optional[string], p ottl.Optional[string]) (ottl.ExprFunc[K], error) {
	delimiter := "="
	if !d.IsEmpty() {
		if d.Get() == "" {
			return nil, errors.New("delimiter cannot be set to an empty string")
		}
		delimiter = d.Get()
	}

	pairDelimiter := " "
	if !p.IsEmpty() {
		if p.Get() == "" {
			return nil, errors.New("pair delimiter cannot be set to an empty string")
		}
		pairDelimiter = p.Get()
	}

	if pairDelimiter == delimiter {
		return nil, fmt.Errorf("pair delimiter %q cannot be equal to delimiter %q", pairDelimiter, delimiter)
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		source, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		if source == "" {
			return nil, errors.New("cannot parse from empty target")
		}

		pairs, err := parseutils.SplitString(source, pairDelimiter)
		if err != nil {
			return nil, fmt.Errorf("splitting source %q into pairs failed: %w", source, err)
		}

		parsed, err := parseutils.ParseKeyValuePairs(pairs, delimiter)
		if err != nil {
			return nil, fmt.Errorf("failed to split pairs into key-values: %w", err)
		}

		result := pcommon.NewMap()
		err = result.FromRaw(parsed)
		return result, err
	}, nil
}
