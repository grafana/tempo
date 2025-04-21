// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"
	gosort "sort"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ToKeyValueStringArguments[K any] struct {
	Target        ottl.PMapGetter[K]
	Delimiter     ottl.Optional[string]
	PairDelimiter ottl.Optional[string]
	SortOutput    ottl.Optional[bool]
}

func NewToKeyValueStringFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ToKeyValueString", &ToKeyValueStringArguments[K]{}, createToKeyValueStringFunction[K])
}

func createToKeyValueStringFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ToKeyValueStringArguments[K])

	if !ok {
		return nil, errors.New("ToKeyValueStringFactory args must be of type *ToKeyValueStringArguments[K]")
	}

	return toKeyValueString[K](args.Target, args.Delimiter, args.PairDelimiter, args.SortOutput)
}

func toKeyValueString[K any](target ottl.PMapGetter[K], d ottl.Optional[string], p ottl.Optional[string], s ottl.Optional[bool]) (ottl.ExprFunc[K], error) {
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

	sortOutput := false
	if !s.IsEmpty() {
		sortOutput = s.Get()
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		source, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		return convertMapToKV(source, delimiter, pairDelimiter, sortOutput), nil
	}, nil
}

// convertMapToKV converts a pcommon.Map to a key value string
func convertMapToKV(target pcommon.Map, delimiter string, pairDelimiter string, sortOutput bool) string {
	var kvStrings []string
	if sortOutput {
		var keyValues []struct {
			key string
			val pcommon.Value
		}

		// Sort by keys
		for k, v := range target.All() {
			keyValues = append(keyValues, struct {
				key string
				val pcommon.Value
			}{key: k, val: v})
		}
		gosort.Slice(keyValues, func(i, j int) bool {
			return keyValues[i].key < keyValues[j].key
		})

		// Convert KV pairs
		for _, kv := range keyValues {
			kvStrings = append(kvStrings, buildKVString(kv.key, kv.val, delimiter, pairDelimiter))
		}
	} else {
		for k, v := range target.All() {
			kvStrings = append(kvStrings, buildKVString(k, v, delimiter, pairDelimiter))
		}
	}

	return strings.Join(kvStrings, pairDelimiter)
}

func buildKVString(k string, v pcommon.Value, delimiter string, pairDelimiter string) string {
	key := escapeAndQuoteKV(k, delimiter, pairDelimiter)
	value := escapeAndQuoteKV(v.AsString(), delimiter, pairDelimiter)
	return key + delimiter + value
}

func escapeAndQuoteKV(s string, delimiter string, pairDelimiter string) string {
	s = strings.ReplaceAll(s, `"`, `\"`)
	if strings.Contains(s, pairDelimiter) || strings.Contains(s, delimiter) {
		s = `"` + s + `"`
	}
	return s
}
