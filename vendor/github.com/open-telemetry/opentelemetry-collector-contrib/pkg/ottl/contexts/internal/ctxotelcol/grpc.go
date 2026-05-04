// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxotelcol // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxotelcol"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"google.golang.org/grpc/metadata"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxerror"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"
)

func accessGRPC[K any](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	nextPath := path.Next()
	if nextPath == nil {
		return nil, ctxerror.New(path.Name(), path.String(), Name, DocRef)
	}
	switch nextPath.Name() {
	case "metadata":
		if nextPath.Keys() == nil {
			return accessGRPCMetadataKeys[K](), nil
		}
		return accessGRPCMetadataKey[K](nextPath.Keys()), nil
	default:
		return nil, ctxerror.New(nextPath.Name(), nextPath.String(), Name, DocRef)
	}
}

func convertGRPCMetadataToMap(md metadata.MD) pcommon.Map {
	mdMap := pcommon.NewMap()
	mdMap.EnsureCapacity(len(md))
	for k, v := range md {
		convertStringArrToValueSlice(v).MoveTo(mdMap.PutEmpty(k))
	}
	return mdMap
}

func accessGRPCMetadataKeys[K any]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, _ K) (any, error) {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return pcommon.NewMap(), nil
			}
			return convertGRPCMetadataToMap(md), nil
		},
		Setter: func(_ context.Context, _ K, _ any) error {
			return fmt.Errorf(readOnlyPathErrMsg, "otelcol.grpc.metadata")
		},
	}
}

func accessGRPCMetadataKey[K any](keys []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			if len(keys) == 0 {
				return nil, errors.New("cannot get map value without keys")
			}
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return nil, nil
			}
			key, err := ctxutil.GetMapKeyName(ctx, tCtx, keys[0])
			if err != nil {
				return nil, err
			}
			mdVal := md.Get(*key)
			if len(mdVal) == 0 {
				return nil, nil
			}
			return getIndexableValueFromStringArr(ctx, tCtx, keys[1:], mdVal)
		},
		Setter: func(_ context.Context, _ K, _ any) error {
			return fmt.Errorf(readOnlyPathErrMsg, "otelcol.grpc.metadata")
		},
	}
}
