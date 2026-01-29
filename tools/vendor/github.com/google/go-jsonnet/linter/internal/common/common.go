// Package common provides utilities to be used in multiple linter
// subpackages.
package common

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/errors"
)

// VariableKind allows distinguishing various kinds of variables.
type VariableKind int

const (
	// VarRegular is a "normal" variable with a definition in the code.
	VarRegular VariableKind = iota
	// VarParam is a function parameter.
	VarParam
	// VarStdlib is a special `std` variable.
	VarStdlib
)

// Variable is a representation of a variable somewhere in the code.
type Variable struct {
	Name         ast.Identifier
	BindNode     ast.Node
	Occurences   []ast.Node
	VariableKind VariableKind
	LocRange     ast.LocationRange
}

// VariableInfo holds information about a variables from one file
type VariableInfo struct {
	Variables []*Variable

	// Variable information at every use site.
	// More precisely it maps every *ast.Var to the variable.
	VarAt map[ast.Node]*Variable
}

// ErrCollector is a struct for accumulating warnings / errors from the linter.
// It is slightly more convenient and more clear than passing pointers to slices around.
type ErrCollector struct {
	Errs []errors.StaticError
}

// Collect adds an error to the list
func (ec *ErrCollector) Collect(err errors.StaticError) {
	ec.Errs = append(ec.Errs, err)
}

// StaticErr constructs a static error from msg and loc and adds it to the list.
func (ec *ErrCollector) StaticErr(msg string, loc *ast.LocationRange) {
	ec.Collect(errors.MakeStaticError(msg, *loc))
}
