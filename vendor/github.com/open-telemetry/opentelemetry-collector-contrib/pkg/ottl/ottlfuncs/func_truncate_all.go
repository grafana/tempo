// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type TruncateAllArguments[K any] struct {
	Target   ottl.PMapGetSetter[K]
	Limit    int64
	Utf8Safe ottl.Optional[bool]
}

func NewTruncateAllFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("truncate_all", &TruncateAllArguments[K]{}, createTruncateAllFunction[K])
}

func createTruncateAllFunction[K any](fCtx ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*TruncateAllArguments[K])

	if !ok {
		return nil, errors.New("TruncateAllFactory args must be of type *TruncateAllArguments[K]")
	}

	return TruncateAll(args.Target, args.Limit, args.Utf8Safe, fCtx.Set.Logger)
}

func TruncateAll[K any](target ottl.PMapGetSetter[K], limit int64, utf8Safe ottl.Optional[bool], logger *zap.Logger) (ottl.ExprFunc[K], error) {
	if limit < 0 {
		return nil, fmt.Errorf("invalid limit for truncate_all function, %d cannot be negative", limit)
	}

	useUTF8Safe := utf8Safe.GetOr(true)

	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		var truncated []string
		debugEnabled := logger.Core().Enabled(zap.DebugLevel)
		for key, value := range val.All() {
			stringVal := value.Str()
			if int64(len(stringVal)) > limit {
				truncateAt := int(limit)
				if useUTF8Safe {
					// Back up to a valid UTF-8 boundary if we're in the middle of a rune
					for truncateAt > 0 && !utf8.RuneStart(stringVal[truncateAt]) {
						truncateAt--
					}
				}
				value.SetStr(stringVal[:truncateAt])
				if debugEnabled {
					truncated = append(truncated, key)
				}
			}
		}
		if len(truncated) != 0 {
			logger.Debug(fmt.Sprintf("truncated %d values", len(truncated)), zap.Strings("keys", truncated))
		}
		return nil, target.Set(ctx, tCtx, val)
	}, nil
}
