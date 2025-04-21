// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
import (
	"context"
	"errors"

	"github.com/ua-parser/uap-go/uaparser"
	semconv "go.opentelemetry.io/collector/semconv/v1.25.0"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type UserAgentArguments[K any] struct {
	UserAgent ottl.StringGetter[K]
}

func NewUserAgentFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("UserAgent", &UserAgentArguments[K]{}, createUserAgentFunction[K])
}

func createUserAgentFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*UserAgentArguments[K])
	if !ok {
		return nil, errors.New("URLFactory args must be of type *URLArguments[K]")
	}

	return userAgent[K](args.UserAgent), nil
}

func userAgent[K any](userAgentSource ottl.StringGetter[K]) ottl.ExprFunc[K] { //revive:disable-line:var-naming
	parser := uaparser.NewFromSaved()

	return func(ctx context.Context, tCtx K) (any, error) {
		userAgentString, err := userAgentSource.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		parsedUserAgent := parser.ParseUserAgent(userAgentString)
		return map[string]any{
			semconv.AttributeUserAgentName:     parsedUserAgent.Family,
			semconv.AttributeUserAgentOriginal: userAgentString,
			semconv.AttributeUserAgentVersion:  parsedUserAgent.ToVersionString(),
		}, nil
	}
}
