// Package traversal provides relatively lightweight checks
// which can all fit within one traversal of the AST.
// Currently available checks:
// * Loop detection
// TODO(sbarzowski) add more
package traversal

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/linter/internal/common"

	"github.com/google/go-jsonnet/internal/parser"
)

func findLoopingInChildren(node ast.Node, vars map[ast.Identifier]ast.Node, runOf map[ast.Identifier]int, currentRun int, ec *common.ErrCollector) bool {
	for _, c := range parser.DirectChildren(node) {
		found := findLooping(c, vars, runOf, currentRun, ec)
		if found {
			return found
		}
	}
	return false
}

func findLooping(node ast.Node, vars map[ast.Identifier]ast.Node, runOf map[ast.Identifier]int, currentRun int, ec *common.ErrCollector) bool {
	switch node := node.(type) {
	case *ast.Var:
		_, varFromThisLocal := vars[node.Id]
		if !varFromThisLocal {
			return false
		}
		firstRun, reachedBefore := runOf[node.Id]
		if !reachedBefore {
			runOf[node.Id] = currentRun
			return findLooping(vars[node.Id], vars, runOf, currentRun, ec)
		} else if firstRun == currentRun {
			// TODO(sbarzowski) Maybe report the whole path of the looping, rather than just the last element
			ec.StaticErr("Endless loop in local definition", node.Loc())
			return true
		}
	}
	return findLoopingInChildren(node, vars, runOf, currentRun, ec)
}

func findLoopingInLocal(node *ast.Local, ec *common.ErrCollector) {
	vars := make(map[ast.Identifier]ast.Node)
	runOf := make(map[ast.Identifier]int)
	for _, b := range node.Binds {
		if b.Body == nil {
			panic("Body cannot be nil")
		}
		vars[b.Variable] = b.Body
	}
	for i, b := range node.Binds {
		found := findLooping(b.Body, vars, runOf, i, ec)
		if found {
			return
		}
	}
}

// Traverse visits all nodes in the AST and runs appropriate
// checks.
func Traverse(node ast.Node, ec *common.ErrCollector) {
	switch node := node.(type) {
	case *ast.Local:
		findLoopingInLocal(node, ec)
	}
	for _, c := range parser.Children(node) {
		Traverse(c, ec)
	}
}
