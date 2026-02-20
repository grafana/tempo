// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
import (
	"context"
	"errors"
	"net"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type IsInCIDRArguments[K any] struct {
	Target   ottl.StringGetter[K]
	Networks []ottl.StringGetter[K]
}

func NewIsInCIDRFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("IsInCIDR", &IsInCIDRArguments[K]{}, createIsInCIDRFunction[K])
}

func createIsInCIDRFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*IsInCIDRArguments[K])
	if !ok {
		return nil, errors.New("IsInCIDRFactory args must be of type *IsInCIDRArguments[K]")
	}

	return isInCIDR(args.Target, args.Networks)
}

func isInCIDR[K any](target ottl.StringGetter[K], networks []ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {
	// Check if all networks are literals and pre-parse them if so.
	literalNetworks := make([]*net.IPNet, 0, len(networks))
	for _, network := range networks {
		literal, isLiteral := ottl.GetLiteralValue[K, string](network)
		if !isLiteral {
			literalNetworks = nil
			break
		}
		_, subnet, err := net.ParseCIDR(literal)
		if err != nil {
			return nil, err
		}
		literalNetworks = append(literalNetworks, subnet)
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		ip := net.ParseIP(val)
		if ip == nil {
			return false, nil
		}

		if literalNetworks != nil {
			// Use pre-parsed networks for literal values.
			for _, subnet := range literalNetworks {
				if subnet.Contains(ip) {
					return true, nil
				}
			}
		} else {
			// Parse networks at runtime for dynamic values.
			for _, network := range networks {
				networkValue, err := network.Get(ctx, tCtx)
				if err != nil {
					return nil, err
				}

				_, subnet, err := net.ParseCIDR(networkValue)
				if err != nil {
					return nil, err
				}
				if subnet.Contains(ip) {
					return true, nil
				}
			}
		}

		return false, nil
	}, nil
}
