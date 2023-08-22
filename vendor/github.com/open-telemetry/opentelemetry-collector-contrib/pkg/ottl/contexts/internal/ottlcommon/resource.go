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

type ResourceContext interface {
	GetResource() pcommon.Resource
}

func ResourcePathGetSetter[K ResourceContext](path []ottl.Field) (ottl.GetSetter[K], error) {
	if len(path) == 0 {
		return accessResource[K](), nil
	}
	switch path[0].Name {
	case "attributes":
		mapKey := path[0].MapKey
		if mapKey == nil {
			return accessResourceAttributes[K](), nil
		}
		return accessResourceAttributesKey[K](mapKey), nil
	case "dropped_attributes_count":
		return accessResourceDroppedAttributesCount[K](), nil
	}

	return nil, fmt.Errorf("invalid resource path expression %v", path)
}

func accessResource[K ResourceContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetResource(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if newRes, ok := val.(pcommon.Resource); ok {
				newRes.CopyTo(tCtx.GetResource())
			}
			return nil
		},
	}
}

func accessResourceAttributes[K ResourceContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetResource().Attributes(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if attrs, ok := val.(pcommon.Map); ok {
				attrs.CopyTo(tCtx.GetResource().Attributes())
			}
			return nil
		},
	}
}

func accessResourceAttributesKey[K ResourceContext](mapKey *string) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return GetMapValue(tCtx.GetResource().Attributes(), *mapKey), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			SetMapValue(tCtx.GetResource().Attributes(), *mapKey, val)
			return nil
		},
	}
}

func accessResourceDroppedAttributesCount[K ResourceContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return int64(tCtx.GetResource().DroppedAttributesCount()), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetResource().SetDroppedAttributesCount(uint32(i))
			}
			return nil
		},
	}
}
