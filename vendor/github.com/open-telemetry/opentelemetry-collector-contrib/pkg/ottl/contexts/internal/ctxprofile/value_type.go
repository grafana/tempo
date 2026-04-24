// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxprofile // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxprofile"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pprofile"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxerror"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"
)

type valueTypeSource[K Context] = func(ctx K) pprofile.ValueType

func valueTypeGetterSetter[K Context](
	path ottl.Path[K],
	source valueTypeSource[K],
) (ottl.GetSetter[K], error) {
	if path == nil || path.Next() == nil {
		return accessValueType(path, source), nil
	}

	nextPath := path.Next()
	switch nextPath.Name() {
	case "type":
		return accessValueTypeType[K](nextPath, source), nil
	case "unit":
		return accessValueTypeUnit[K](nextPath, source), nil
	default:
		return nil, ctxerror.New(path.Name(), nextPath.String(), Name, DocRef)
	}
}

func accessValueType[K Context](path ottl.Path[K], getValueType valueTypeSource[K]) ottl.GetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return getValueType(tCtx), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			newValue, err := ctxutil.ExpectType[pprofile.ValueType](val)
			if err != nil {
				return fmt.Errorf("path %q %w", path.String(), err)
			}
			newValue.CopyTo(getValueType(tCtx))
			return nil
		},
	}
}

func accessValueTypeType[K Context](path ottl.Path[K], getValueType valueTypeSource[K]) ottl.GetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			valueType := getValueType(tCtx)
			return getValueTypeString(path, tCtx.GetProfilesDictionary(), valueType.TypeStrindex())
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			valueType := getValueType(tCtx)
			newIndex, err := setValueTypeString(path, tCtx.GetProfilesDictionary(), valueType.TypeStrindex(), val)
			if err != nil {
				return err
			}
			valueType.SetTypeStrindex(newIndex)
			return nil
		},
	}
}

func accessValueTypeUnit[K Context](path ottl.Path[K], getValueType valueTypeSource[K]) ottl.GetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			valueType := getValueType(tCtx)
			return getValueTypeString(path, tCtx.GetProfilesDictionary(), valueType.UnitStrindex())
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			valueType := getValueType(tCtx)
			newIndex, err := setValueTypeString(path, tCtx.GetProfilesDictionary(), valueType.UnitStrindex(), val)
			if err != nil {
				return err
			}
			valueType.SetUnitStrindex(newIndex)
			return nil
		},
	}
}

func getValueTypeString[K Context](
	path ottl.Path[K],
	dict pprofile.ProfilesDictionary,
	currIndex int32,
) (string, error) {
	if currIndex < 0 || int(currIndex) >= dict.StringTable().Len() {
		return "", fmt.Errorf("path %q with strindex %d is out of range", path.String(), currIndex)
	}
	return dict.StringTable().At(int(currIndex)), nil
}

func setValueTypeString[K Context](
	path ottl.Path[K],
	dict pprofile.ProfilesDictionary,
	currIndex int32,
	val any,
) (int32, error) {
	newValue, err := ctxutil.ExpectType[string](val)
	if err != nil {
		return 0, fmt.Errorf("path %q %w", path.String(), err)
	}
	if currIndex != 0 && int(currIndex) < dict.StringTable().Len() {
		currValue := dict.StringTable().At(int(currIndex))
		if currValue == newValue {
			return currIndex, nil
		}
	}
	newIndex, err := pprofile.SetString(dict.StringTable(), newValue)
	if err != nil {
		return 0, err
	}
	return newIndex, nil
}
