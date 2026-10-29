// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/antchfx/xmlquery"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ConvertAttributesToElementsXMLArguments[K any] struct {
	Target ottl.StringGetter[K]
	XPath  ottl.Optional[string]
}

func NewConvertAttributesToElementsXMLFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ConvertAttributesToElementsXML", &ConvertAttributesToElementsXMLArguments[K]{}, createConvertAttributesToElementsXMLFunction[K])
}

func createConvertAttributesToElementsXMLFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ConvertAttributesToElementsXMLArguments[K])

	if !ok {
		return nil, errors.New("ConvertAttributesToElementsXML args must be of type *ConvertAttributesToElementsXMLAguments[K]")
	}

	xPath := args.XPath.Get()
	if xPath == "" {
		xPath = "//@*" // All attributes in the document
	}
	if err := validateXPath(xPath); err != nil {
		return nil, err
	}

	return convertAttributesToElementsXML(args.Target, xPath), nil
}

// convertAttributesToElementsXML returns a string that is a result of converting all attributes of the
// target XML into child elements. These new elements are added as the last child elements of the parent.
// e.g. <a foo="bar" hello="world"><b/></a> -> <a><hello>world</hello><foo>bar</foo><b/></a>
func convertAttributesToElementsXML[K any](target ottl.StringGetter[K], xPath string) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		var doc *xmlquery.Node
		if targetVal, err := target.Get(ctx, tCtx); err != nil {
			return nil, err
		} else if doc, err = parseNodesXML(targetVal); err != nil {
			return nil, err
		}
		for _, n := range xmlquery.Find(doc, xPath) {
			if n.Type != xmlquery.AttributeNode {
				continue
			}
			xmlquery.AddChild(n.Parent, &xmlquery.Node{
				Type: xmlquery.ElementNode,
				Data: n.Data,
				FirstChild: &xmlquery.Node{
					Type: xmlquery.TextNode,
					Data: n.InnerText(),
				},
			})
			n.Parent.RemoveAttr(n.Data)
		}
		return doc.OutputXML(false), nil
	}
}
