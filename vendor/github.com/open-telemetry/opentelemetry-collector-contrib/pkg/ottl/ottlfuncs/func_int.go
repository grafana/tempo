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
	"strconv"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func Int[K any](target ottl.Getter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (interface{}, error) {
		value, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		switch value := value.(type) {
		case int64:
			return value, nil
		case string:
			intValue, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, nil
			}

			return intValue, nil
		case float64:
			return (int64)(value), nil
		case bool:
			if value {
				return int64(1), nil
			}
			return int64(0), nil
		default:
			return nil, nil
		}
	}, nil
}
