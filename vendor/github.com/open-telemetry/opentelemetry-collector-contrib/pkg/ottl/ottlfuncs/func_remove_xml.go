// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type RemoveXMLArguments[K any] struct {
	Target ottl.StringGetter[K]
	XPath  string
}

func NewRemoveXMLFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("RemoveXML", &RemoveXMLArguments[K]{}, createRemoveXMLFunction[K])
}

func createRemoveXMLFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*RemoveXMLArguments[K])

	if !ok {
		return nil, errors.New("RemoveXML args must be of type *RemoveXMLAguments[K]")
	}

	if err := validateXPath(args.XPath); err != nil {
		return nil, err
	}

	return removeXML(args.Target, args.XPath), nil
}

// removeXML returns a XML formatted string that is a result of removing all matching nodes from the target XML.
// This currently supports removal of elements, attributes, text values, comments, and CharData.
func removeXML[K any](target ottl.StringGetter[K], xPath string) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		var doc *xmlquery.Node
		targetVal, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if targetVal == "" {
			return "", nil
		}
		if doc, err = parseNodesXML(targetVal); err != nil {
			return nil, err
		}

		nodes, err := xmlquery.QueryAll(doc, xPath)
		if err != nil {
			return nil, err
		}

		for _, n := range nodes {
			switch n.Type {
			case xmlquery.ElementNode:
				xmlquery.RemoveFromTree(n)
			case xmlquery.AttributeNode:
				n.Parent.RemoveAttr(n.Data)
			case xmlquery.TextNode:
				n.Data = ""
			case xmlquery.CommentNode:
				xmlquery.RemoveFromTree(n)
			case xmlquery.CharDataNode:
				xmlquery.RemoveFromTree(n)
			}
		}
		return doc.OutputXML(false), nil
	}
}

func validateXPath(xPath string) error {
	_, err := xpath.Compile(xPath)
	if err != nil {
		return fmt.Errorf("invalid xpath: %w", err)
	}
	return nil
}

// Aside from parsing the XML document, this function also ensures that
// the XML declaration is included in the result only if it was present in
// the original document.
func parseNodesXML(targetVal string) (*xmlquery.Node, error) {
	preserveDeclearation := strings.HasPrefix(targetVal, "<?xml")
	top, err := xmlquery.Parse(strings.NewReader(targetVal))
	if err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}
	if !preserveDeclearation && top.FirstChild != nil {
		xmlquery.RemoveFromTree(top.FirstChild)
	}
	return top, nil
}
