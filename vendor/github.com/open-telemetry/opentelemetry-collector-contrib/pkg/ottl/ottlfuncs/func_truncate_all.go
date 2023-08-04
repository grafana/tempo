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

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func TruncateAll[K any](target ottl.GetSetter[K], limit int64) (ottl.ExprFunc[K], error) {
	if limit < 0 {
		return nil, fmt.Errorf("invalid limit for truncate_all function, %d cannot be negative", limit)
	}
	return func(ctx context.Context, tCtx K) (interface{}, error) {
		if limit < 0 {
			return nil, nil
		}

		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if val == nil {
			return nil, nil
		}

		attrs, ok := val.(pcommon.Map)
		if !ok {
			return nil, nil
		}

		updated := pcommon.NewMap()
		attrs.CopyTo(updated)
		updated.Range(func(key string, value pcommon.Value) bool {
			stringVal := value.Str()
			if int64(len(stringVal)) > limit {
				value.SetStr(stringVal[:limit])
			}
			return true
		})
		err = target.Set(ctx, tCtx, updated)
		if err != nil {
			return nil, err
		}
		// TODO: Write log when truncation is performed
		// https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/9730
		return nil, nil
	}, nil
}
