// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/antchfx/xmlquery"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ParseSimplifiedXMLArguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewParseSimplifiedXMLFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ParseSimplifiedXML", &ParseSimplifiedXMLArguments[K]{}, createParseSimplifiedXMLFunction[K])
}

func createParseSimplifiedXMLFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ParseSimplifiedXMLArguments[K])

	if !ok {
		return nil, errors.New("ParseSimplifiedXML args must be of type *ParseSimplifiedXMLAguments[K]")
	}

	return parseSimplifiedXML(args.Target), nil
}

// The `ParseSimplifiedXML` Converter returns a `pcommon.Map` struct that is the result of parsing the target
// string without preservation of attributes or extraneous text content.
func parseSimplifiedXML[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		var doc *xmlquery.Node
		if targetVal, err := target.Get(ctx, tCtx); err != nil {
			return nil, err
		} else if doc, err = parseNodesXML(targetVal); err != nil {
			return nil, err
		}

		docMap := pcommon.NewMap()
		parseElement(doc, &docMap)
		return docMap, nil
	}
}

func parseElement(parent *xmlquery.Node, parentMap *pcommon.Map) {
	// Count the number of each element tag so we know whether it will be a member of a slice or not
	childTags := make(map[string]int)
	for child := parent.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != xmlquery.ElementNode {
			continue
		}
		childTags[child.Data]++
	}
	if len(childTags) == 0 {
		return
	}

	// Convert the children, now knowing whether they will be a member of a slice or not
	for child := parent.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != xmlquery.ElementNode || child.FirstChild == nil {
			continue
		}

		leafValue := leafValueFromElement(child)

		// Slice of the same element
		if childTags[child.Data] > 1 {
			// Get or create the slice of children
			var childrenSlice pcommon.Slice
			childrenValue, ok := parentMap.Get(child.Data)
			if ok {
				childrenSlice = childrenValue.Slice()
			} else {
				childrenSlice = parentMap.PutEmptySlice(child.Data)
			}

			// Add the child's text content to the slice
			if leafValue != "" {
				childrenSlice.AppendEmpty().SetStr(leafValue)
				continue
			}

			// Parse the child to make sure there's something to add
			childMap := pcommon.NewMap()
			parseElement(child, &childMap)
			if childMap.Len() == 0 {
				continue
			}

			sliceValue := childrenSlice.AppendEmpty()
			sliceMap := sliceValue.SetEmptyMap()
			childMap.CopyTo(sliceMap)
			continue
		}

		if leafValue != "" {
			parentMap.PutStr(child.Data, leafValue)
			continue
		}

		// Child will be a map
		childMap := pcommon.NewMap()
		parseElement(child, &childMap)
		if childMap.Len() == 0 {
			continue
		}

		childMap.CopyTo(parentMap.PutEmptyMap(child.Data))
	}
}

func leafValueFromElement(node *xmlquery.Node) string {
	// First check if there are any child elements. If there are, ignore any extraneous text.
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xmlquery.ElementNode {
			return ""
		}
	}

	// No child elements, so return the first text or CDATA content
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case xmlquery.TextNode, xmlquery.CharDataNode:
			return child.Data
		}
	}
	return ""
}
