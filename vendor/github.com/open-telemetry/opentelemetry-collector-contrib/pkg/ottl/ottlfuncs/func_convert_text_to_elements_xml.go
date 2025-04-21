// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/antchfx/xmlquery"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ConvertTextToElementsXMLArguments[K any] struct {
	Target      ottl.StringGetter[K]
	XPath       ottl.Optional[string]
	ElementName ottl.Optional[string]
}

func NewConvertTextToElementsXMLFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ConvertTextToElementsXML", &ConvertTextToElementsXMLArguments[K]{}, createConvertTextToElementsXMLFunction[K])
}

func createConvertTextToElementsXMLFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ConvertTextToElementsXMLArguments[K])

	if !ok {
		return nil, errors.New("ConvertTextToElementsXML args must be of type *ConvertTextToElementsXMLAguments[K]")
	}

	xPath := args.XPath.Get()
	if xPath == "" {
		xPath = "/"
	} else if err := validateXPath(xPath); err != nil {
		return nil, err
	}

	elementName := args.ElementName.Get()
	if elementName == "" {
		elementName = "value"
	}

	return convertTextToElementsXML(args.Target, xPath, elementName), nil
}

// convertTextToElementsXML returns a string that is a result of wrapping any extraneous text nodes with a dedicated element.
func convertTextToElementsXML[K any](target ottl.StringGetter[K], xPath string, elementName string) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		var doc *xmlquery.Node
		if targetVal, err := target.Get(ctx, tCtx); err != nil {
			return nil, err
		} else if doc, err = parseNodesXML(targetVal); err != nil {
			return nil, err
		}
		for _, n := range xmlquery.Find(doc, xPath) {
			convertTextToElementsForNode(n, elementName)
		}
		return doc.OutputXML(false), nil
	}
}

func convertTextToElementsForNode(parent *xmlquery.Node, elementName string) {
	switch parent.Type {
	case xmlquery.ElementNode: // ok
	case xmlquery.DocumentNode: // ok
	default:
		return
	}

	if parent.FirstChild == nil {
		return
	}

	// Convert any child nodes and count text and element nodes.
	var valueCount, elementCount int
	for child := parent.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case xmlquery.ElementNode:
			convertTextToElementsForNode(child, elementName)
			elementCount++
		case xmlquery.TextNode:
			valueCount++
		}
	}

	// If there are no values to wrap, or if there is exactly one value OR one element, this node is all set.
	if valueCount == 0 || elementCount+valueCount <= 1 {
		return
	}

	// At this point, we either have multiple values, or a mix of values and elements.
	// Either way, we need to wrap the values.
	for child := parent.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != xmlquery.TextNode {
			continue
		}
		newTextNode := &xmlquery.Node{
			Type: xmlquery.TextNode,
			Data: child.Data,
		}
		// Change this node into an element
		child.Type = xmlquery.ElementNode
		child.Data = elementName
		child.FirstChild = newTextNode
		child.LastChild = newTextNode
	}
}
