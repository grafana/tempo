// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"

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
	for i := 0; i < len(keys); i++ {
		switch val.Type() {
		case pcommon.ValueTypeMap:
			s, err := keys[i].String(ctx, tCtx)
			if err != nil {
				return nil, err
			}
			if s == nil {
				return nil, fmt.Errorf("map must be indexed by a string")
			}
			val, ok = val.Map().Get(*s)
			if !ok {
				return nil, nil
			}
		case pcommon.ValueTypeSlice:
			i, err := keys[i].Int(ctx, tCtx)
			if err != nil {
				return nil, err
			}
			if i == nil {
				return nil, fmt.Errorf("slice must be indexed by an int")
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

func setIndexableValue[K any](ctx context.Context, tCtx K, currentValue pcommon.Value, val any, keys []ottl.Key[K]) error {
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

	for i := 0; i < len(keys); i++ {
		switch currentValue.Type() {
		case pcommon.ValueTypeMap:
			s, err := keys[i].String(ctx, tCtx)
			if err != nil {
				return err
			}
			if s == nil {
				return errors.New("map must be indexed by a string")
			}
			potentialValue, ok := currentValue.Map().Get(*s)
			if !ok {
				currentValue = currentValue.Map().PutEmpty(*s)
			} else {
				currentValue = potentialValue
			}
		case pcommon.ValueTypeSlice:
			i, err := keys[i].Int(ctx, tCtx)
			if err != nil {
				return err
			}
			if i == nil {
				return errors.New("slice must be indexed by an int")
			}
			if int(*i) >= currentValue.Slice().Len() || int(*i) < 0 {
				return fmt.Errorf("index %v out of bounds", *i)
			}
			currentValue = currentValue.Slice().At(int(*i))
		case pcommon.ValueTypeEmpty:
			s, err := keys[i].String(ctx, tCtx)
			if err != nil {
				return err
			}
			i, err := keys[i].Int(ctx, tCtx)
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
				return errors.New("neither a string nor an int index was given, this is an error in the OTTL")
			}
		default:
			return fmt.Errorf("type %v does not support string indexing", currentValue.Type())
		}
	}
	newValue.CopyTo(currentValue)
	return nil
}
