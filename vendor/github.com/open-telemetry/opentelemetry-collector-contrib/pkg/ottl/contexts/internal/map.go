// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func GetMapValue(m pcommon.Map, keys []ottl.Key) (interface{}, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("cannot get map value without key")
	}
	if keys[0].String == nil {
		return nil, fmt.Errorf("non-string indexing is not supported")
	}

	val, ok := m.Get(*keys[0].String)
	if !ok {
		return nil, nil
	}
	return getIndexableValue(val, keys[1:])
}

func SetMapValue(m pcommon.Map, keys []ottl.Key, val interface{}) error {
	if len(keys) == 0 {
		return fmt.Errorf("cannot set map value without key")
	}
	if keys[0].String == nil {
		return fmt.Errorf("non-string indexing is not supported")
	}

	currentValue, ok := m.Get(*keys[0].String)
	if !ok {
		currentValue = m.PutEmpty(*keys[0].String)
	}

	return setIndexableValue(currentValue, val, keys[1:])
}
