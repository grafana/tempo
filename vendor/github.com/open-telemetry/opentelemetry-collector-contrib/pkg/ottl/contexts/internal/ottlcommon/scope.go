// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ottlcommon // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ottlcommon"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type InstrumentationScopeContext interface {
	GetInstrumentationScope() pcommon.InstrumentationScope
}

func ScopePathGetSetter[K InstrumentationScopeContext](path []ottl.Field) (ottl.GetSetter[K], error) {
	if len(path) == 0 {
		return accessInstrumentationScope[K](), nil
	}

	switch path[0].Name {
	case "name":
		return accessInstrumentationScopeName[K](), nil
	case "version":
		return accessInstrumentationScopeVersion[K](), nil
	case "attributes":
		mapKey := path[0].MapKey
		if mapKey == nil {
			return accessInstrumentationScopeAttributes[K](), nil
		}
		return accessInstrumentationScopeAttributesKey[K](mapKey), nil
	case "dropped_attributes_count":
		return accessInstrumentationScopeDroppedAttributesCount[K](), nil
	}

	return nil, fmt.Errorf("invalid scope path expression %v", path)
}

func accessInstrumentationScope[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetInstrumentationScope(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if newIl, ok := val.(pcommon.InstrumentationScope); ok {
				newIl.CopyTo(tCtx.GetInstrumentationScope())
			}
			return nil
		},
	}
}

func accessInstrumentationScopeAttributes[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetInstrumentationScope().Attributes(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if attrs, ok := val.(pcommon.Map); ok {
				attrs.CopyTo(tCtx.GetInstrumentationScope().Attributes())
			}
			return nil
		},
	}
}

func accessInstrumentationScopeAttributesKey[K InstrumentationScopeContext](mapKey *string) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return GetMapValue(tCtx.GetInstrumentationScope().Attributes(), *mapKey), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			SetMapValue(tCtx.GetInstrumentationScope().Attributes(), *mapKey, val)
			return nil
		},
	}
}

func accessInstrumentationScopeName[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetInstrumentationScope().Name(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if str, ok := val.(string); ok {
				tCtx.GetInstrumentationScope().SetName(str)
			}
			return nil
		},
	}
}

func accessInstrumentationScopeVersion[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetInstrumentationScope().Version(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if str, ok := val.(string); ok {
				tCtx.GetInstrumentationScope().SetVersion(str)
			}
			return nil
		},
	}
}

func accessInstrumentationScopeDroppedAttributesCount[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return int64(tCtx.GetInstrumentationScope().DroppedAttributesCount()), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetInstrumentationScope().SetDroppedAttributesCount(uint32(i))
			}
			return nil
		},
	}
}
