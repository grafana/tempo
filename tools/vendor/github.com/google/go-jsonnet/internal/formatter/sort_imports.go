/*
Copyright 2019 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package formatter

import (
	"sort"

	"github.com/google/go-jsonnet/ast"
)

type importElem struct {
	adjacentFodder ast.Fodder
	key            string
	bind           ast.LocalBind
}

func sortGroup(imports []importElem) {
	if !duplicatedVariables(imports) {
		sort.Slice(imports, func(i, j int) bool {
			return imports[i].key < imports[j].key
		})
	}
}

// Check if `local` expression is used for importing.
func isGoodLocal(local *ast.Local) bool {
	for _, bind := range local.Binds {
		if bind.Fun != nil {
			return false
		}
		_, ok := bind.Body.(*ast.Import)
		if !ok {
			return false
		}
	}
	return true
}

func goodLocalOrNull(node ast.Node) *ast.Local {
	local, ok := node.(*ast.Local)
	if ok && isGoodLocal(local) {
		return local
	}
	return nil
}

/** Split fodder after the first new line / paragraph fodder,
 * leaving blank lines after the newline in the second half.
 *
 * The two returned fodders can be concatenated using concat_fodder to get the original fodder.
 *
 * It's a heuristic that given two consecutive tokens `prev_token`, `next_token`
 * with some fodder between them, decides which part of the fodder logically belongs
 * to `prev_token` and which part belongs to the `next_token`.
 *
 * Example:
 * prev_token // prev_token is awesome!
 *
 * // blah blah
 * next_token
 *
 * In such case "// prev_token is awesome!\n" part of the fodder belongs
 * to the `prev_token` and "\n//blah blah\n" to the `next_token`.
 */
func splitFodder(fodder ast.Fodder) (ast.Fodder, ast.Fodder) {
	var afterPrev, beforeNext ast.Fodder
	inSecondPart := false
	for _, fodderElem := range fodder {
		if inSecondPart {
			ast.FodderAppend(&beforeNext, fodderElem)
		} else {
			afterPrev = append(afterPrev, fodderElem)
		}
		if fodderElem.Kind != ast.FodderInterstitial && !inSecondPart {
			inSecondPart = true
			if fodderElem.Blanks > 0 {
				// If there are any blank lines at the end of afterPrev, move them
				// to beforeNext.
				afterPrev[len(afterPrev)-1].Blanks = 0
				if len(beforeNext) != 0 {
					panic("beforeNext should still be empty.")
				}
				beforeNext = append(beforeNext, ast.FodderElement{
					Kind:   ast.FodderLineEnd,
					Blanks: fodderElem.Blanks,
					Indent: fodderElem.Indent,
				})
			}
		}
	}
	return afterPrev, beforeNext
}

func extractImportElems(binds ast.LocalBinds, after ast.Fodder) []importElem {
	var result []importElem
	before := binds[0].VarFodder
	for i, bind := range binds {
		last := i == len(binds)-1
		var adjacent ast.Fodder
		var beforeNext ast.Fodder
		if !last {
			next := &binds[i+1]
			adjacent, beforeNext = splitFodder(next.VarFodder)
		} else {
			adjacent = after
		}
		ast.FodderEnsureCleanNewline(&adjacent)
		newBind := bind
		newBind.VarFodder = before
		theImport := bind.Body.(*ast.Import)
		result = append(result,
			importElem{key: theImport.File.Value, adjacentFodder: adjacent, bind: newBind})
		before = beforeNext
	}
	return result
}

func buildGroupAST(imports []importElem, body ast.Node, groupOpenFodder ast.Fodder) ast.Node {
	for i := len(imports) - 1; i >= 0; i-- {
		theImport := &(imports)[i]
		var fodder ast.Fodder
		if i == 0 {
			fodder = groupOpenFodder
		} else {
			fodder = imports[i-1].adjacentFodder
		}
		local := &ast.Local{
			NodeBase: ast.NodeBase{Fodder: fodder},
			Binds:    []ast.LocalBind{theImport.bind},
			Body:     body}
		body = local
	}
	return body
}

func duplicatedVariables(elems []importElem) bool {
	idents := make(map[string]bool)
	for _, elem := range elems {
		idents[string(elem.bind.Variable)] = true
	}
	return len(idents) < len(elems)
}

func groupEndsAfter(local *ast.Local) bool {
	next := goodLocalOrNull(local.Body)
	if next == nil {
		return true
	}
	newlineReached := false
	for _, fodderElem := range *openFodder(next) {
		if newlineReached || fodderElem.Blanks > 0 {
			return true
		}
		if fodderElem.Kind != ast.FodderInterstitial {
			newlineReached = true
		}
	}
	return false
}

func topLevelImport(local *ast.Local, imports *[]importElem, groupOpenFodder ast.Fodder) ast.Node {
	if !isGoodLocal(local) {
		panic("topLevelImport called with bad local.")
	}
	adjacentCommentFodder, beforeNextFodder :=
		splitFodder(*openFodder(local.Body))
	ast.FodderEnsureCleanNewline(&adjacentCommentFodder)
	newImports := extractImportElems(local.Binds, adjacentCommentFodder)
	*imports = append(*imports, newImports...)

	if groupEndsAfter(local) {
		sortGroup(*imports)
		afterGroup := (*imports)[len(*imports)-1].adjacentFodder
		ast.FodderEnsureCleanNewline(&beforeNextFodder)
		nextOpenFodder := ast.FodderConcat(afterGroup, beforeNextFodder)
		var bodyAfterGroup ast.Node
		// Process the code after the current group:
		next := goodLocalOrNull(local.Body)
		if next != nil {
			// Another group of imports
			nextImports := make([]importElem, 0)
			bodyAfterGroup = topLevelImport(next, &nextImports, nextOpenFodder)
		} else {
			// Something else
			bodyAfterGroup = local.Body
			*openFodder(bodyAfterGroup) = nextOpenFodder
		}

		return buildGroupAST(*imports, bodyAfterGroup, groupOpenFodder)
	}

	if len(beforeNextFodder) > 0 {
		panic("Expected beforeNextFodder to be empty")
	}
	return topLevelImport(local.Body.(*ast.Local), imports, groupOpenFodder)
}

// SortImports sorts imports at the top of the file into alphabetical order
// by path.
//
// Top-level imports are `local x = import 'xxx.jsonnet` expressions
// that go before anything else in the file (more precisely all such imports
// that are either the root of AST or a direct child (body) of a top-level
// import.  Top-level imports are therefore more top-level than top-level
// functions.
//
// Grouping of imports is preserved. Groups of imports are separated by blank
// lines or lines containing comments.
func SortImports(file *ast.Node) {
	imports := make([]importElem, 0)
	local := goodLocalOrNull(*file)
	if local != nil {
		*file = topLevelImport(local, &imports, *openFodder(local))
	}
}
