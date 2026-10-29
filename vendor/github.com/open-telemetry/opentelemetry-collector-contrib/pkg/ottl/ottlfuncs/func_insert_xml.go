// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"github.com/antchfx/xmlquery"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type InsertXMLArguments[K any] struct {
	Target      ottl.StringGetter[K]
	XPath       string
	SubDocument ottl.StringGetter[K]
}

func NewInsertXMLFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("InsertXML", &InsertXMLArguments[K]{}, createInsertXMLFunction[K])
}

func createInsertXMLFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*InsertXMLArguments[K])

	if !ok {
		return nil, errors.New("InsertXML args must be of type *InsertXMLAguments[K]")
	}

	if err := validateXPath(args.XPath); err != nil {
		return nil, err
	}

	return insertXML(args.Target, args.XPath, args.SubDocument), nil
}

// insertXML returns a XML formatted string that is a result of inserting another XML document into
// the content of each selected target element.
func insertXML[K any](target ottl.StringGetter[K], xPath string, subGetter ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		var doc *xmlquery.Node
		targetVal, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if targetVal == "" {
			doc = &xmlquery.Node{
				Type: xmlquery.ElementNode,
				Data: targetVal,
			}
		} else if doc, err = parseNodesXML(targetVal); err != nil {
			return nil, err
		}

		var subDoc *xmlquery.Node
		if subDocVal, err := subGetter.Get(ctx, tCtx); err != nil {
			return nil, err
		} else if subDoc, err = parseNodesXML(subDocVal); err != nil {
			return nil, err
		}

		nodes, errs := xmlquery.QueryAll(doc, xPath)
		for _, n := range nodes {
			switch n.Type {
			case xmlquery.ElementNode, xmlquery.DocumentNode:
				var nextSibling *xmlquery.Node
				for c := subDoc.FirstChild; c != nil; c = nextSibling {
					// AddChild updates c.NextSibling but not subDoc.FirstChild
					// so we need to get the handle to it prior to the update.
					nextSibling = c.NextSibling
					xmlquery.AddChild(n, c)
				}
			default:
				errs = errors.Join(errs, fmt.Errorf("InsertXML XPath selected non-element: %q", n.Data))
			}
		}
		return doc.OutputXML(false), errs
	}
}
