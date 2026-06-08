package types

import (
	"fmt"
	"strconv"

	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/parser"
	"github.com/google/go-jsonnet/linter/internal/common"
)

// Maximum number of array elements for which we track
// the values individually. We do that, because in Jsonnet
// there is no separate tuple type, so we treat arrays as
// potential tuples.
const maxKnownCount = 5

// ImportFunc should provide an AST node from a given location.
// If a node is not available it should return nil.$
type ImportFunc func(currentPath, importedPath string) ast.Node

func (g *typeGraph) getExprPlaceholder(node ast.Node) placeholderID {
	if g.exprPlaceholder[node] == noType {
		// fmt.Fprintf(os.Stderr, "------------------------------------------------------------------\n")
		// spew.Dump(node)
		panic("Bug - placeholder for a dependent node cannot be noType")
		// It will be possible in later stages, after some simplifications
		// but for now (i.e. during generation) it means that something was not initialized
		// at the appropriate time.
	}
	return g.exprPlaceholder[node]
}

// prepareTP recursively creates type placeholders for all expressions
// in a subtree and calculates the definitions for them.
func prepareTP(node ast.Node, varAt map[ast.Node]*common.Variable, g *typeGraph) {
	if node == nil {
		panic("Node cannot be nil")
	}
	p := g.newPlaceholder()
	g.exprPlaceholder[node] = p
	prepareTPWithPlaceholder(node, varAt, g, p)
}

// prepareTPWithPlaceholder recursively creates type placeholders for all
// expressions in a subtree.
//
// The type placeholder for the root of the subtree is not created, but already provided.
// This allows us to express mutually recursive relationships.
func prepareTPWithPlaceholder(node ast.Node, varAt map[ast.Node]*common.Variable, g *typeGraph, p placeholderID) {
	if node == nil {
		panic("Node cannot be nil")
	}
	switch node := node.(type) {
	case *ast.Local:
		bindPlaceholders := make([]placeholderID, len(node.Binds))
		for i := range node.Binds {
			bindPlaceholders[i] = g.newPlaceholder()
			g.exprPlaceholder[node.Binds[i].Body] = bindPlaceholders[i]
		}
		for i := range node.Binds {
			prepareTPWithPlaceholder(node.Binds[i].Body, varAt, g, bindPlaceholders[i])
		}
		prepareTP(node.Body, varAt, g)
	case *ast.DesugaredObject:
		localPlaceholders := make([]placeholderID, len(node.Locals))
		for i := range node.Locals {
			localPlaceholders[i] = g.newPlaceholder()
			g.exprPlaceholder[node.Locals[i].Body] = localPlaceholders[i]
		}
		for i := range node.Locals {
			prepareTPWithPlaceholder(node.Locals[i].Body, varAt, g, localPlaceholders[i])
		}
		for i := range node.Fields {
			prepareTP(node.Fields[i].Name, varAt, g)
			prepareTP(node.Fields[i].Body, varAt, g)
		}
	default:
		for _, child := range parser.Children(node) {
			if child == nil {
				panic("Bug - child cannot be nil")
			}
			prepareTP(child, varAt, g)
		}
	}
	*(g.placeholder(p)) = calcTP(node, varAt, g)
}

func (g *typeGraph) addRoots(roots map[string]ast.Node, vars map[string]map[ast.Node]*common.Variable) {
	for _, rootNode := range roots {
		p := g.newPlaceholder()
		g.exprPlaceholder[rootNode] = p
	}

	for importedPath, rootNode := range roots {
		prepareTPWithPlaceholder(rootNode, vars[importedPath], g, g.getExprPlaceholder(rootNode))
	}
}

func countRequiredParameters(params []ast.Parameter) int {
	count := 0
	for _, p := range params {
		if p.DefaultArg != nil {
			count++
		}
	}
	return count
}

// calcTP calculates a definition for a type placeholder.
func calcTP(node ast.Node, varAt map[ast.Node]*common.Variable, g *typeGraph) typePlaceholder {
	switch node := node.(type) {
	case *ast.Array:
		knownCount := len(node.Elements)
		if knownCount > maxKnownCount {
			knownCount = maxKnownCount
		}

		desc := &arrayDesc{
			furtherContain:  make([]placeholderID, 0, len(node.Elements)-knownCount),
			elementContains: make([][]placeholderID, knownCount, maxKnownCount),
		}

		for i, el := range node.Elements {
			if i < knownCount {
				desc.elementContains[i] = []placeholderID{g.getExprPlaceholder(el.Expr)}
			} else {
				desc.furtherContain = append(desc.furtherContain, g.getExprPlaceholder(el.Expr))
			}
		}

		return concreteTP(TypeDesc{ArrayDesc: desc})
	case *ast.Binary:
		if node.Op == ast.BopPlus {
			return typePlaceholder{
				builtinOp: &builtinOpDesc{
					args: []placeholderID{g.getExprPlaceholder(node.Left), g.getExprPlaceholder(node.Right)},
					f:    builtinPlus,
				},
			}
		}
		return tpRef(anyType)
	case *ast.Unary:
		switch node.Op {
		case ast.UopNot:
			return tpRef(boolType)
		case ast.UopBitwiseNot, ast.UopPlus, ast.UopMinus:
			return tpRef(numberType)
		default:
			panic(fmt.Sprintf("Unrecognized unary operator %v", node.Op))
		}
	case *ast.Conditional:
		return tpSum(g.getExprPlaceholder(node.BranchTrue), g.getExprPlaceholder(node.BranchFalse))
	case *ast.Var:
		v := varAt[node]
		if v == nil {
			panic("Could not find variable")
		}
		switch v.VariableKind {
		case common.VarStdlib:
			return tpRef(stdlibType)
		case common.VarParam:
			return tpRef(anyType)
		case common.VarRegular:

			return tpRef(g.getExprPlaceholder(v.BindNode))
		}

	case *ast.DesugaredObject:
		obj := &objectDesc{
			allFieldsKnown: true,
			fieldContains:  make(map[string][]placeholderID),
		}
		for _, field := range node.Fields {
			switch fieldName := field.Name.(type) {
			case *ast.LiteralString:
				if field.PlusSuper {
					obj.fieldContains[fieldName.Value] = []placeholderID{anyType}
				} else {
					obj.fieldContains[fieldName.Value] = append(obj.fieldContains[fieldName.Value], g.getExprPlaceholder(field.Body))
				}
			default:
				obj.allFieldsKnown = false
				if field.PlusSuper {
					obj.unknownContain = []placeholderID{anyType}
				} else {
					obj.unknownContain = append(obj.unknownContain, g.getExprPlaceholder(field.Body))
				}
			}
		}
		return concreteTP(TypeDesc{ObjectDesc: obj})
	case *ast.Error:
		return concreteTP(voidTypeDesc())
	case *ast.Index:
		switch index := node.Index.(type) {
		case *ast.LiteralString:
			return tpIndex(knownObjectIndex(g.getExprPlaceholder(node.Target), index.Value))
		case *ast.LiteralNumber:
			// Since the lexer ensures that OriginalString is of
			// the right form, this will only fail if the number is
			// too large to fit in a double.
			valFloat, err := strconv.ParseFloat(index.OriginalString, 64)
			if err != nil {
				return tpIndex(unknownIndexSpec(g.getExprPlaceholder(node.Target)))
			}
			if valFloat >= 0 && valFloat < maxKnownCount && valFloat == float64(int64(valFloat)) {
				return tpIndex(arrayIndex(g.getExprPlaceholder(node.Target), int(valFloat)))
			}
		}
		return tpIndex(unknownIndexSpec(g.getExprPlaceholder(node.Target)))
	case *ast.Import:
		codePath := node.Loc().FileName
		imported := g.importFunc(codePath, node.File.Value)
		if imported == nil {
			return tpRef(anyType)
		}
		return tpRef(g.getExprPlaceholder(imported))
	case *ast.ImportStr:
		return tpRef(stringType)
	case *ast.ImportBin:
		return tpRef(numberArrayType)
	case *ast.LiteralBoolean:
		return tpRef(boolType)
	case *ast.LiteralNull:
		return tpRef(nullType)

	case *ast.LiteralNumber:
		return tpRef(numberType)

	case *ast.LiteralString:
		return tpRef(stringType)

	case *ast.Local:
		return tpRef(g.getExprPlaceholder(node.Body))
	case *ast.Self:
		// no recursion yet
		return tpRef(anyObjectType)
	case *ast.SuperIndex:
		return tpRef(anyType)
	case *ast.InSuper:
		return tpRef(boolType)
	case *ast.Function:
		return concreteTP(TypeDesc{FunctionDesc: &functionDesc{
			minArity:       countRequiredParameters(node.Parameters),
			maxArity:       len(node.Parameters),
			params:         node.Parameters,
			resultContains: []placeholderID{g.getExprPlaceholder(node.Body)},
		}})
	case *ast.Apply:
		return tpIndex(functionCallIndex(g.getExprPlaceholder(node.Target)))
	}
	panic(fmt.Sprintf("Unexpected %#v", node))
}
