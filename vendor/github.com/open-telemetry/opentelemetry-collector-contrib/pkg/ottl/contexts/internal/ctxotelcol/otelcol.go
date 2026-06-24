// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxotelcol // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxotelcol"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxerror"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/metadata"
)

var errOTelColContextDisabled = errors.New("OTTL `otelcol` context requires the `ottl.contexts.enableOTelColContext` feature gate to be enabled")

func PathGetSetter[K any](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	if !metadata.OttlContextsEnableOTelColContextFeatureGate.IsEnabled() {
		return nil, errOTelColContextDisabled
	}
	switch path.Name() {
	case "client":
		return accessClient[K](path)
	case "grpc":
		return accessGRPC[K](path)
	default:
		return nil, ctxerror.New(path.Name(), path.String(), Name, DocRef)
	}
}

func convertStringArrToValueSlice(vals []string) pcommon.Value {
	val := pcommon.NewValueSlice()
	sl := val.Slice()
	sl.EnsureCapacity(len(vals))
	for _, v := range vals {
		sl.AppendEmpty().SetStr(v)
	}
	return val
}

func getIndexableValueFromStringArr[K any](ctx context.Context, tCtx K, keys []ottl.Key[K], strSlice []string) (any, error) {
	if len(keys) == 0 {
		slice := pcommon.NewSlice()
		slice.EnsureCapacity(len(strSlice))
		for _, str := range strSlice {
			slice.AppendEmpty().SetStr(str)
		}
		return slice, nil
	}
	if len(keys) > 1 {
		return nil, errors.New("cannot index into string slice more than once")
	}
	index, err := ctxutil.GetSliceIndexFromKeys(ctx, tCtx, len(strSlice), keys)
	if err != nil {
		return nil, err
	}
	return strSlice[index], nil
}
