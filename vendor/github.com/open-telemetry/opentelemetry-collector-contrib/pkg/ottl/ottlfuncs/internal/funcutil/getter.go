// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package funcutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs/internal/funcutil"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

// GetSliceOrMapValue retrieves a pcommon.Slice or pcommon.Map value based on the provided getter logic.
// Returns an error if the value type is unsupported or retrieval fails.
func GetSliceOrMapValue[K any](ctx context.Context, tCtx K, getter ottl.Getter[K]) (any, error) {
	sourceVal, err := getter.Get(ctx, tCtx)
	if err != nil {
		return nil, err
	}

	// short-circuit if the value is already a pcommon.Slice or pcommon.Map
	switch typedVal := sourceVal.(type) {
	case pcommon.Map, pcommon.Slice:
		return typedVal, nil
	case pcommon.Value:
		switch typedVal.Type() {
		case pcommon.ValueTypeMap:
			return typedVal.Map(), nil
		case pcommon.ValueTypeSlice:
			return typedVal.Slice(), nil
		}
	}

	valueGetter := func(context.Context, K) (any, error) { return sourceVal, nil }

	var res any
	res, err = ottl.StandardPSliceGetter[K]{Getter: valueGetter}.Get(ctx, tCtx)
	if err == nil {
		return res, nil
	}
	var typeError ottl.TypeError
	if !errors.As(err, &typeError) {
		return nil, err
	}

	res, err = ottl.StandardPMapGetter[K]{Getter: valueGetter}.Get(ctx, tCtx)
	if err == nil {
		return res, nil
	}
	if !errors.As(err, &typeError) {
		return nil, err
	}

	return nil, fmt.Errorf("unsupported type provided: %T", sourceVal)
}
