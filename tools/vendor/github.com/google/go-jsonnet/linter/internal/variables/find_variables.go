// Package variables allows collecting the information about how variables
// are used.
package variables

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/parser"
	"github.com/google/go-jsonnet/linter/internal/common"
)

// Environment is mapping from variable names to information about variables.
// It represents variables in a specific scope.
type Environment map[ast.Identifier]*common.Variable

func addVar(name ast.Identifier, loc ast.LocationRange, bindNode ast.Node, info *common.VariableInfo, scope Environment, varKind common.VariableKind) {
	v := &common.Variable{
		Name:         name,
		BindNode:     bindNode,
		Occurences:   nil,
		VariableKind: varKind,
		LocRange:     loc,
	}
	info.Variables = append(info.Variables, v)
	scope[name] = v
}

func cloneScope(oldScope Environment) Environment {
	new := make(Environment)
	for k, v := range oldScope {
		new[k] = v
	}
	return new
}

func findVariablesInFunc(node *ast.Function, info *common.VariableInfo, scope Environment) {
	for _, param := range node.Parameters {
		addVar(param.Name, param.LocRange, nil, info, scope, common.VarParam)
	}
	for _, param := range node.Parameters {
		if param.DefaultArg != nil {
			findVariables(param.DefaultArg, info, scope)
		}
	}
	findVariables(node.Body, info, scope)
}

func findVariablesInLocal(node *ast.Local, info *common.VariableInfo, scope Environment) {
	for _, bind := range node.Binds {
		addVar(bind.Variable, bind.LocRange, bind.Body, info, scope, common.VarRegular)
	}
	for _, bind := range node.Binds {
		if bind.Fun != nil {
			newScope := cloneScope(scope)
			findVariablesInFunc(bind.Fun, info, newScope)
		} else {
			findVariables(bind.Body, info, scope)
		}
	}
	findVariables(node.Body, info, scope)
}

func findVariablesInObject(node *ast.DesugaredObject, info *common.VariableInfo, scopeOutside Environment) {
	scopeInside := cloneScope(scopeOutside)
	for _, local := range node.Locals {
		addVar(local.Variable, local.LocRange, local.Body, info, scopeInside, common.VarRegular)
	}
	for _, local := range node.Locals {
		findVariables(local.Body, info, scopeInside)
	}
	for _, field := range node.Fields {
		findVariables(field.Body, info, scopeInside)
		findVariables(field.Name, info, scopeOutside)
	}
}

func findVariables(node ast.Node, info *common.VariableInfo, scope Environment) {
	switch node := node.(type) {
	case *ast.Function:
		newScope := cloneScope(scope)
		findVariablesInFunc(node, info, newScope)
	case *ast.Local:
		newScope := cloneScope(scope)
		findVariablesInLocal(node, info, newScope)
	case *ast.DesugaredObject:
		newScope := cloneScope(scope)
		findVariablesInObject(node, info, newScope)
	case *ast.Var:
		if v, ok := scope[node.Id]; ok {
			v.Occurences = append(v.Occurences, node)
		} else {
			panic("Undeclared variable " + string(node.Id) + " - it should be caught earlier")
		}

	default:
		for _, child := range parser.Children(node) {
			findVariables(child, info, scope)
		}
	}
}

// FindVariables builds common.VariableInfo based on the AST from a file.
func FindVariables(node ast.Node, scope Environment) *common.VariableInfo {
	info := common.VariableInfo{
		Variables: nil,
		VarAt:     make(map[ast.Node]*common.Variable),
	}
	// Add variables from the initial scope (e.g. std)
	for _, v := range scope {
		info.Variables = append(info.Variables, v)
	}
	findVariables(node, &info, scope)
	for _, v := range info.Variables {
		for _, u := range v.Occurences {
			info.VarAt[u] = v
		}
	}
	return &info
}
