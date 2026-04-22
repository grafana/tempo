// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
import (
	"context"
	"errors"

	"github.com/ua-parser/uap-go/uaparser"
	conventions "go.opentelemetry.io/otel/semconv/v1.38.0"

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
		parsedOS := parser.ParseOs(userAgentString)
		result := map[string]any{
			string(conventions.UserAgentNameKey):     parsedUserAgent.Family,
			string(conventions.UserAgentOriginalKey): userAgentString,
			string(conventions.UserAgentVersionKey):  parsedUserAgent.ToVersionString(),
		}

		osName := parsedOS.Family
		if osName != "" {
			result[string(conventions.OSNameKey)] = osName
		}

		osVersion := parsedOS.ToVersionString()
		if osVersion != "" {
			result[string(conventions.OSVersionKey)] = osVersion
		}

		return result, nil
	}
}
