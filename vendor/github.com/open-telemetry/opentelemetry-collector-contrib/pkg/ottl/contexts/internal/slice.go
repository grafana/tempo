// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func GetSliceValue(s pcommon.Slice, keys []ottl.Key) (interface{}, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("cannot get slice value without key")
	}
	if keys[0].Int == nil {
		return nil, fmt.Errorf("non-integer indexing is not supported")
	}
	idx := int(*keys[0].Int)

	if idx < 0 || idx >= s.Len() {
		return nil, fmt.Errorf("index %d out of bounds", idx)
	}

	return getIndexableValue(s.At(int(*keys[0].Int)), keys[1:])
}

func SetSliceValue(s pcommon.Slice, keys []ottl.Key, val interface{}) error {
	if len(keys) == 0 {
		return fmt.Errorf("cannot set slice value without key")
	}
	if keys[0].Int == nil {
		return fmt.Errorf("non-integer indexing is not supported")
	}
	idx := int(*keys[0].Int)

	if idx < 0 || idx >= s.Len() {
		return fmt.Errorf("index %d out of bounds", idx)
	}

	return setIndexableValue(s.At(int(*keys[0].Int)), val, keys[1:])
}
