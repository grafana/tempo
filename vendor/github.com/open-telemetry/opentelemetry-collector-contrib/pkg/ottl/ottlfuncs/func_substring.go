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

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func Substring[K any](target ottl.Getter[K], start int64, length int64) (ottl.ExprFunc[K], error) {
	if start < 0 {
		return nil, fmt.Errorf("invalid start for substring function, %d cannot be negative", start)
	}
	if length <= 0 {
		return nil, fmt.Errorf("invalid length for substring function, %d cannot be negative or zero", length)
	}

	return func(ctx context.Context, tCtx K) (interface{}, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if valStr, ok := val.(string); ok {
			if (start + length) > int64(len(valStr)) {
				return nil, fmt.Errorf("invalid range for substring function, %d cannot be greater than the length of target string(%d)", start+length, len(valStr))
			}
			return valStr[start : start+length], nil
		}
		return nil, nil
	}, nil
}
