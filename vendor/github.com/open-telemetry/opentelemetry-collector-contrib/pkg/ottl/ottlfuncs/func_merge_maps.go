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

const (
	INSERT = "insert"
	UPDATE = "update"
	UPSERT = "upsert"
)

// MergeMaps function merges the source map into the target map using the supplied strategy to handle conflicts.
// Strategy definitions:
//
//	insert: Insert the value from `source` into `target` where the key does not already exist.
//	update: Update the entry in `target` with the value from `source` where the key does exist
//	upsert: Performs insert or update. Insert the value from `source` into `target` where the key does not already exist and update the entry in `target` with the value from `source` where the key does exist.
func MergeMaps[K any](target ottl.Getter[K], source ottl.Getter[K], strategy string) (ottl.ExprFunc[K], error) {
	if strategy != INSERT && strategy != UPDATE && strategy != UPSERT {
		return nil, fmt.Errorf("invalid value for strategy, %v, must be 'insert', 'update' or 'upsert'", strategy)
	}

	return func(ctx context.Context, tCtx K) (interface{}, error) {
		targetVal, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		if targetMap, ok := targetVal.(pcommon.Map); ok {
			val, err := source.Get(ctx, tCtx)
			if err != nil {
				return nil, err
			}

			if valueMap, ok := val.(pcommon.Map); ok {
				switch strategy {
				case INSERT:
					valueMap.Range(func(k string, v pcommon.Value) bool {
						if _, ok := targetMap.Get(k); !ok {
							tv := targetMap.PutEmpty(k)
							v.CopyTo(tv)
						}
						return true
					})
				case UPDATE:
					valueMap.Range(func(k string, v pcommon.Value) bool {
						if tv, ok := targetMap.Get(k); ok {
							v.CopyTo(tv)
						}
						return true
					})
				case UPSERT:
					valueMap.Range(func(k string, v pcommon.Value) bool {
						tv := targetMap.PutEmpty(k)
						v.CopyTo(tv)
						return true
					})
				default:
					return nil, fmt.Errorf("unknown strategy, %v", strategy)
				}
			}
		}
		return nil, nil
	}, nil
}
