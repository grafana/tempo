package types

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/parser"
	"github.com/google/go-jsonnet/linter/internal/common"
)

func checkSubexpr(node ast.Node, typeOf exprTypes, ec *common.ErrCollector) {
	for _, child := range parser.Children(node) {
		check(child, typeOf, ec)
	}
}

// check verifies that the types are valid for a given program, given
// the previously resolved types.
func check(node ast.Node, typeOf exprTypes, ec *common.ErrCollector) {
	checkSubexpr(node, typeOf, ec)
	switch node := node.(type) {
	case *ast.Apply:
		t := typeOf[node.Target]
		if !t.Function() {
			ec.StaticErr("Called value must be a function, but it is assumed to be "+Describe(&t), node.Loc())
		} else if t.FunctionDesc.params != nil {
			checkArgs(t.FunctionDesc.params, &node.Arguments, node.Loc(), ec)
		} else {
			argsCount := len(node.Arguments.Named) + len(node.Arguments.Positional)
			minArity := t.FunctionDesc.minArity
			maxArity := t.FunctionDesc.maxArity
			if minArity > argsCount {
				ec.StaticErr(fmt.Sprintf("Too few arguments: got %d, but expected at least %d", argsCount, minArity), node.Loc())
			}
			if maxArity < argsCount {
				ec.StaticErr(fmt.Sprintf("Too many arguments: got %d, but expected at most %d", argsCount, maxArity), node.Loc())
			}
		}
	case *ast.Index:
		targetType := typeOf[node.Target]
		indexType := typeOf[node.Index]

		if !targetType.Array() && !targetType.Object() && !targetType.String {
			ec.StaticErr("Indexed value is neither an array nor an object nor a string", node.Loc())
		} else if !targetType.Object() {
			// It's not an object, so it must be an array or a string
			var assumedType string
			if targetType.Array() && targetType.String {
				assumedType = "an array or a string"
			} else if targetType.Array() {
				assumedType = "an array"
			} else {
				assumedType = "a string"
			}
			if !indexType.Number {
				ec.StaticErr("Indexed value is assumed to be "+assumedType+", but index is not a number", node.Loc())
			}
		} else if !targetType.Array() && !targetType.String {
			// It's not an array or a string so it must be an object
			if !indexType.String {
				ec.StaticErr("Indexed value is assumed to be an object, but index is not a string", node.Loc())
			}
			if targetType.ObjectDesc.allFieldsKnown {
				switch indexNode := node.Index.(type) {
				case *ast.LiteralString:
					if _, hasField := targetType.ObjectDesc.fieldContains[indexNode.Value]; !hasField {
						ec.StaticErr(fmt.Sprintf("Indexed object has no field %#v", indexNode.Value), node.Loc())
					}
				}
			}
		} else if !indexType.Number && !indexType.String {
			// We don't know what the target is, but we sure cannot index it with that
			ec.StaticErr("Index is neither a number (for indexing arrays and string) nor a string (for indexing objects)", node.Loc())
		}
	case *ast.Unary:
		operandType := typeOf[node.Expr]
		switch node.Op {
		case ast.UopBitwiseNot, ast.UopMinus, ast.UopPlus:
			if !operandType.Number {
				ec.StaticErr(fmt.Sprintf("Operand is not a number, it is assumed to be %s", Describe(&operandType)), node.Loc())
			}
		case ast.UopNot:
			if !operandType.Bool {
				ec.StaticErr(fmt.Sprintf("Operand is not a boolean, it is assumed to be %s", Describe(&operandType)), node.Loc())
			}
		}
	}
}

// TODO(sbarzowski) eliminate duplication with the interpreter maybe (this is AST-level and there it's value-level)
func checkArgs(params []ast.Parameter, args *ast.Arguments, loc *ast.LocationRange, ec *common.ErrCollector) {
	received := make(map[ast.Identifier]bool)
	accepted := make(map[ast.Identifier]bool)

	numPassed := len(args.Positional)
	numExpected := len(params)

	for _, param := range params {
		accepted[param.Name] = true
	}

	for i := range args.Positional {
		if i < len(params) {
			received[params[i].Name] = true
		} else {
			ec.StaticErr(fmt.Sprintf("Too many arguments, there can be at most %d, but %d provided", numExpected, numPassed), args.Positional[i].Expr.Loc())
		}
	}

	for _, arg := range args.Named {
		if _, present := received[arg.Name]; present {
			ec.StaticErr(fmt.Sprintf("Argument %v already provided", arg.Name), arg.Arg.Loc())
			return
		}
		if _, present := accepted[arg.Name]; !present {
			ec.StaticErr(fmt.Sprintf("function has no parameter %v", arg.Name), arg.Arg.Loc())
			return
		}
		received[arg.Name] = true
	}

	for _, param := range params {
		if _, present := received[param.Name]; !present && param.DefaultArg == nil {
			ec.StaticErr(fmt.Sprintf("Missing argument: %v", param.Name), loc)
			return
		}
	}
}

// Check finds type problems in a given program.
// It require passing some previously processed data:
// * root nodes of all (transitively) imported Jsonnet files
// * resolution of variables in all files
// * importFunc which allows resolving imports
func Check(mainNode ast.Node, roots map[string]ast.Node, vars map[string]map[ast.Node]*common.Variable, importFunc ImportFunc, ec *common.ErrCollector) {
	et := make(exprTypes)
	g := newTypeGraph(importFunc)
	g.addRoots(roots, vars)
	g.prepareTypes(mainNode, et)

	// TODO(sbarzowski) Useful for debugging – expose it in CLI?
	// t := et[node.node]
	// fmt.Fprintf(os.Stderr, "%v\n", types.Describe(&t))

	check(mainNode, et, ec)
}
