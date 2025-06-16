// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlcommon"
)

func SetValue(value pcommon.Value, val any) error {
	var err error
	switch v := val.(type) {
	case string:
		value.SetStr(v)
	case bool:
		value.SetBool(v)
	case int64:
		value.SetInt(v)
	case float64:
		value.SetDouble(v)
	case []byte:
		value.SetEmptyBytes().FromRaw(v)
	case []string:
		value.SetEmptySlice().EnsureCapacity(len(v))
		for _, str := range v {
			value.Slice().AppendEmpty().SetStr(str)
		}
	case []bool:
		value.SetEmptySlice().EnsureCapacity(len(v))
		for _, b := range v {
			value.Slice().AppendEmpty().SetBool(b)
		}
	case []int64:
		value.SetEmptySlice().EnsureCapacity(len(v))
		for _, i := range v {
			value.Slice().AppendEmpty().SetInt(i)
		}
	case []float64:
		value.SetEmptySlice().EnsureCapacity(len(v))
		for _, f := range v {
			value.Slice().AppendEmpty().SetDouble(f)
		}
	case [][]byte:
		value.SetEmptySlice().EnsureCapacity(len(v))
		for _, b := range v {
			value.Slice().AppendEmpty().SetEmptyBytes().FromRaw(b)
		}
	case []any:
		value.SetEmptySlice().EnsureCapacity(len(v))
		for _, a := range v {
			pval := value.Slice().AppendEmpty()
			err = SetValue(pval, a)
		}
	case pcommon.Slice:
		v.CopyTo(value.SetEmptySlice())
	case pcommon.Map:
		v.CopyTo(value.SetEmptyMap())
	case map[string]any:
		err = value.FromRaw(v)
	}
	return err
}

func getIndexableValue[K any](ctx context.Context, tCtx K, value pcommon.Value, keys []ottl.Key[K]) (any, error) {
	val := value
	var ok bool
	for index := 0; index < len(keys); index++ {
		switch val.Type() {
		case pcommon.ValueTypeMap:
			s, err := GetMapKeyName(ctx, tCtx, keys[index])
			if err != nil {
				return nil, err
			}
			val, ok = val.Map().Get(*s)
			if !ok {
				return nil, nil
			}
		case pcommon.ValueTypeSlice:
			i, err := keys[index].Int(ctx, tCtx)
			if err != nil {
				return nil, err
			}
			if i == nil {
				resInt, err := FetchValueFromExpression[K, int64](ctx, tCtx, keys[index])
				if err != nil {
					return nil, fmt.Errorf("unable to resolve an integer index in slice: %w", err)
				}
				i = resInt
			}
			if int(*i) >= val.Slice().Len() || int(*i) < 0 {
				return nil, fmt.Errorf("index %v out of bounds", *i)
			}
			val = val.Slice().At(int(*i))
		default:
			return nil, fmt.Errorf("type %v does not support string indexing", val.Type())
		}
	}
	return ottlcommon.GetValue(val), nil
}

func SetIndexableValue[K any](ctx context.Context, tCtx K, currentValue pcommon.Value, val any, keys []ottl.Key[K]) error {
	var newValue pcommon.Value
	switch val.(type) {
	case []string, []bool, []int64, []float64, [][]byte, []any:
		newValue = pcommon.NewValueSlice()
	default:
		newValue = pcommon.NewValueEmpty()
	}
	err := SetValue(newValue, val)
	if err != nil {
		return err
	}

	for index := 0; index < len(keys); index++ {
		switch currentValue.Type() {
		case pcommon.ValueTypeMap:
			s, err := GetMapKeyName(ctx, tCtx, keys[index])
			if err != nil {
				return err
			}
			potentialValue, ok := currentValue.Map().Get(*s)
			if !ok {
				currentValue = currentValue.Map().PutEmpty(*s)
			} else {
				currentValue = potentialValue
			}
		case pcommon.ValueTypeSlice:
			i, err := keys[index].Int(ctx, tCtx)
			if err != nil {
				return err
			}
			if i == nil {
				resInt, err := FetchValueFromExpression[K, int64](ctx, tCtx, keys[index])
				if err != nil {
					return fmt.Errorf("unable to resolve an integer index in slice: %w", err)
				}
				i = resInt
			}
			if int(*i) >= currentValue.Slice().Len() || int(*i) < 0 {
				return fmt.Errorf("index %v out of bounds", *i)
			}
			currentValue = currentValue.Slice().At(int(*i))
		case pcommon.ValueTypeEmpty:
			s, err := keys[index].String(ctx, tCtx)
			if err != nil {
				return err
			}
			i, err := keys[index].Int(ctx, tCtx)
			if err != nil {
				return err
			}
			switch {
			case s != nil:
				currentValue = currentValue.SetEmptyMap().PutEmpty(*s)
			case i != nil:
				currentValue.SetEmptySlice()
				for k := 0; k < int(*i); k++ {
					currentValue.Slice().AppendEmpty()
				}
				currentValue = currentValue.Slice().AppendEmpty()
			default:
				resString, errString := FetchValueFromExpression[K, string](ctx, tCtx, keys[index])
				resInt, errInt := FetchValueFromExpression[K, int64](ctx, tCtx, keys[index])
				switch {
				case errInt == nil:
					currentValue.SetEmptySlice()
					for k := 0; k < int(*resInt); k++ {
						currentValue.Slice().AppendEmpty()
					}
					currentValue = currentValue.Slice().AppendEmpty()
				case errString == nil:
					currentValue = currentValue.SetEmptyMap().PutEmpty(*resString)
				default:
					return errors.New("neither a string nor an int index was given, this is an error in the OTTL")
				}
			}
		default:
			return fmt.Errorf("type %v does not support string indexing", currentValue.Type())
		}
	}
	newValue.CopyTo(currentValue)
	return nil
}
