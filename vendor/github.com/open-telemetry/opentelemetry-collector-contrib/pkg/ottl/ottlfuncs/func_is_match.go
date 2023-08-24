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
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func IsMatch[K any](target ottl.Getter[K], pattern string) (ottl.ExprFunc[K], error) {
	compiledPattern, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("the pattern supplied to IsMatch is not a valid regexp pattern: %w", err)
	}
	return func(ctx context.Context, tCtx K) (interface{}, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if val == nil {
			return false, nil
		}

		switch v := val.(type) {
		case string:
			return compiledPattern.MatchString(v), nil
		case bool:
			return compiledPattern.MatchString(strconv.FormatBool(v)), nil
		case int64:
			return compiledPattern.MatchString(strconv.FormatInt(v, 10)), nil
		case float64:
			return compiledPattern.MatchString(strconv.FormatFloat(v, 'f', -1, 64)), nil
		case []byte:
			return compiledPattern.MatchString(base64.StdEncoding.EncodeToString(v)), nil
		case pcommon.Map:
			result, err := jsoniter.MarshalToString(v.AsRaw())
			if err != nil {
				return nil, err
			}
			return compiledPattern.MatchString(result), nil
		case pcommon.Slice:
			result, err := jsoniter.MarshalToString(v.AsRaw())
			if err != nil {
				return nil, err
			}
			return compiledPattern.MatchString(result), nil
		case pcommon.Value:
			return compiledPattern.MatchString(v.AsString()), nil
		default:
			return nil, errors.New("unsupported type")
		}
	}, nil
}
