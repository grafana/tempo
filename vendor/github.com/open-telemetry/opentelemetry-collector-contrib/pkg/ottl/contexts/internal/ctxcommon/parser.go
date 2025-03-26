// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxcommon // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"

import (
	"go.opentelemetry.io/collector/component"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcache"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxerror"
)

func NewParser[K any](
	functions map[string]ottl.Factory[K],
	telemetrySettings component.TelemetrySettings,
	pathExpressionParser ottl.PathExpressionParser[K],
	enumParser ottl.EnumParser,
	options ...ottl.Option[K],
) (ottl.Parser[K], error) {
	p, err := ottl.NewParser(
		functions,
		pathExpressionParser,
		telemetrySettings,
		ottl.WithEnumParser[K](enumParser),
	)
	if err != nil {
		return ottl.Parser[K]{}, err
	}
	for _, opt := range options {
		opt(&p)
	}
	return p, nil
}

func PathExpressionParser[K any](
	contextName string,
	contextDocRef string,
	cacheGetter ctxcache.Getter[K],
	contextParsers map[string]ottl.PathExpressionParser[K],
) ottl.PathExpressionParser[K] {
	return func(path ottl.Path[K]) (ottl.GetSetter[K], error) {
		if path == nil {
			return nil, ctxerror.New("nil", "nil", contextName, contextDocRef)
		}

		fullPath := path.String()

		// Normalize context and segment name
		pathContext := path.Context()
		if pathContext == "" {
			if contextParsers[path.Name()] == nil {
				pathContext = contextName
			} else {
				pathContext = path.Name()
				path = path.Next()
				if path == nil {
					return nil, ctxerror.New(pathContext, fullPath, contextName, contextDocRef)
				}
			}
		}

		// Allow cache access only on this context
		if path.Name() == ctxcache.Name {
			if pathContext == contextName {
				return ctxcache.PathExpressionParser(cacheGetter)(path)
			}
			return nil, ctxcache.NewError(contextName, pathContext, fullPath)
		}

		parser, ok := contextParsers[pathContext]
		if ok {
			return parser(path)
		}
		return nil, ctxerror.New(pathContext, fullPath, contextName, contextDocRef)
	}
}
