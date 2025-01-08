// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/antchfx/xmlquery"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type GetXMLArguments[K any] struct {
	Target ottl.StringGetter[K]
	XPath  string
}

func NewGetXMLFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("GetXML", &GetXMLArguments[K]{}, createGetXMLFunction[K])
}

func createGetXMLFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*GetXMLArguments[K])

	if !ok {
		return nil, fmt.Errorf("GetXML args must be of type *GetXMLAguments[K]")
	}

	if err := validateXPath(args.XPath); err != nil {
		return nil, err
	}

	return getXML(args.Target, args.XPath), nil
}

// getXML returns a XML formatted string that is a result of matching elements from the target XML.
func getXML[K any](target ottl.StringGetter[K], xPath string) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		var doc *xmlquery.Node
		if targetVal, err := target.Get(ctx, tCtx); err != nil {
			return nil, err
		} else if doc, err = parseNodesXML(targetVal); err != nil {
			return nil, err
		}

		nodes, err := xmlquery.QueryAll(doc, xPath)
		if err != nil {
			return nil, err
		}

		result := &xmlquery.Node{Type: xmlquery.DocumentNode}
		for _, n := range nodes {
			switch n.Type {
			case xmlquery.ElementNode, xmlquery.TextNode:
				xmlquery.AddChild(result, n)
			case xmlquery.AttributeNode, xmlquery.CharDataNode:
				// get the value
				xmlquery.AddChild(result, &xmlquery.Node{
					Type: xmlquery.TextNode,
					Data: n.InnerText(),
				})
			default:
				continue
			}
		}
		return result.OutputXML(false), nil
	}
}
