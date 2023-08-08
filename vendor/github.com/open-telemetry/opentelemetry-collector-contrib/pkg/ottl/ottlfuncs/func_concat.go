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
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func Concat[K any](vals []ottl.Getter[K], delimiter string) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (interface{}, error) {
		builder := strings.Builder{}
		for i, rv := range vals {
			val, err := rv.Get(ctx, tCtx)
			if err != nil {
				return nil, err
			}
			switch v := val.(type) {
			case string:
				builder.WriteString(v)
			case []byte:
				builder.WriteString(fmt.Sprintf("%x", v))
			case int64:
				builder.WriteString(fmt.Sprint(v))
			case float64:
				builder.WriteString(fmt.Sprint(v))
			case bool:
				builder.WriteString(fmt.Sprint(v))
			case nil:
				builder.WriteString(fmt.Sprint(v))
			}

			if i != len(vals)-1 {
				builder.WriteString(delimiter)
			}
		}
		return builder.String(), nil
	}, nil
}
