// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxotelcol // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxotelcol"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxerror"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlcommon"
)

func accessClient[K any](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	nextPath := path.Next()
	if nextPath == nil {
		return nil, ctxerror.New(path.Name(), path.String(), Name, DocRef)
	}
	switch nextPath.Name() {
	case "addr":
		return accessClientAddr(nextPath)
	case "auth":
		return accessClientAuth(nextPath)
	case "metadata":
		return accessClientMetadata(nextPath)
	default:
		return nil, ctxerror.New(nextPath.Name(), nextPath.String(), Name, DocRef)
	}
}

func accessClientMetadata[K any](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	nextPath := path.Next()
	if nextPath != nil {
		return nil, ctxerror.New(nextPath.Name(), nextPath.String(), Name, DocRef)
	}
	if path.Keys() == nil {
		return accessClientMetadataKeys[K](), nil
	}
	return accessClientMetadataKey[K](path.Keys()), nil
}

func accessClientAddr[K any](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	nextPath := path.Next()
	if nextPath != nil {
		return nil, ctxerror.New(nextPath.Name(), nextPath.String(), Name, DocRef)
	}
	if path.Keys() != nil {
		return nil, ctxerror.New(path.Name(), path.String(), Name, DocRef)
	}
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, _ K) (any, error) {
			cl := client.FromContext(ctx)
			if cl.Addr == nil {
				return nil, nil
			}
			return cl.Addr.String(), nil
		},
		Setter: func(_ context.Context, _ K, _ any) error {
			return fmt.Errorf(readOnlyPathErrMsg, "otelcol.client.addr")
		},
	}, nil
}

func getAuthAttributeValue(authData client.AuthData, key string) (pcommon.Value, error) {
	attrVal := authData.GetAttribute(key)
	switch typedAttrVal := attrVal.(type) {
	case string:
		return pcommon.NewValueStr(typedAttrVal), nil
	case []string:
		value := pcommon.NewValueSlice()
		slice := value.Slice()
		slice.EnsureCapacity(len(typedAttrVal))
		for _, str := range typedAttrVal {
			slice.AppendEmpty().SetStr(str)
		}
		return value, nil
	default:
		value := pcommon.NewValueEmpty()
		err := value.FromRaw(attrVal)
		if err != nil {
			return pcommon.Value{}, err
		}
		return value, nil
	}
}

func convertAuthDataToMap(authData client.AuthData) pcommon.Map {
	authMap := pcommon.NewMap()
	if authData == nil {
		return authMap
	}
	names := authData.GetAttributeNames()
	authMap.EnsureCapacity(len(names))
	for _, name := range names {
		newKeyValue := authMap.PutEmpty(name)
		if value, err := getAuthAttributeValue(authData, name); err == nil {
			value.MoveTo(newKeyValue)
		}
	}
	return authMap
}

func accessClientAuth[K any](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	nextPath := path.Next()
	if nextPath == nil {
		return nil, ctxerror.New(path.Name(), path.String(), Name, DocRef)
	}
	switch nextPath.Name() {
	case "attributes":
		if nextPath.Keys() == nil {
			return accessClientAuthAttributesKeys[K](), nil
		}
		return accessClientAuthAttributesKey[K](nextPath.Keys()), nil
	default:
		return nil, ctxerror.New(nextPath.Name(), nextPath.String(), Name, DocRef)
	}
}

func accessClientAuthAttributesKeys[K any]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, _ K) (any, error) {
			cl := client.FromContext(ctx)
			return convertAuthDataToMap(cl.Auth), nil
		},
		Setter: func(_ context.Context, _ K, _ any) error {
			return fmt.Errorf(readOnlyPathErrMsg, "otelcol.client.auth.attributes")
		},
	}
}

func accessClientAuthAttributesKey[K any](keys []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			if len(keys) == 0 {
				return nil, errors.New("cannot get map value without keys")
			}
			cl := client.FromContext(ctx)
			key, err := ctxutil.GetMapKeyName(ctx, tCtx, keys[0])
			if err != nil {
				return nil, err
			}
			if cl.Auth == nil {
				return nil, nil
			}
			attrVal, err := getAuthAttributeValue(cl.Auth, *key)
			if err != nil {
				return nil, err
			}
			if len(keys) > 1 {
				switch attrVal.Type() {
				case pcommon.ValueTypeSlice:
					return ctxutil.GetSliceValue[K](ctx, tCtx, attrVal.Slice(), keys[1:])
				case pcommon.ValueTypeMap:
					return ctxutil.GetMapValue[K](ctx, tCtx, attrVal.Map(), keys[1:])
				default:
					return nil, fmt.Errorf("attribute %q value is not indexable: %s", *key, attrVal.Type().String())
				}
			}
			return ottlcommon.GetValue(attrVal), nil
		},
		Setter: func(_ context.Context, _ K, _ any) error {
			return fmt.Errorf(readOnlyPathErrMsg, "otelcol.client.auth.attributes")
		},
	}
}

func convertClientMetadataToMap(md client.Metadata) pcommon.Map {
	mdMap := pcommon.NewMap()
	for k := range md.Keys() {
		convertStringArrToValueSlice(md.Get(k)).MoveTo(mdMap.PutEmpty(k))
	}
	return mdMap
}

func accessClientMetadataKeys[K any]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, _ K) (any, error) {
			cl := client.FromContext(ctx)
			return convertClientMetadataToMap(cl.Metadata), nil
		},
		Setter: func(_ context.Context, _ K, _ any) error {
			return fmt.Errorf(readOnlyPathErrMsg, "otelcol.client.metadata")
		},
	}
}

func accessClientMetadataKey[K any](keys []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			if len(keys) == 0 {
				return nil, errors.New("cannot get map value without keys")
			}

			key, err := ctxutil.GetMapKeyName(ctx, tCtx, keys[0])
			if err != nil {
				return nil, fmt.Errorf("cannot get map value: %w", err)
			}
			cl := client.FromContext(ctx)
			mdVal := cl.Metadata.Get(*key)
			if len(mdVal) == 0 {
				return nil, nil
			}
			return getIndexableValueFromStringArr(ctx, tCtx, keys[1:], mdVal)
		},
		Setter: func(_ context.Context, _ K, _ any) error {
			return fmt.Errorf(readOnlyPathErrMsg, "otelcol.client.metadata")
		},
	}
}
