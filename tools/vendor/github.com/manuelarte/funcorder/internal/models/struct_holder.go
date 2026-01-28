package models

import (
	"cmp"
	"go/ast"
	"slices"

	"golang.org/x/tools/go/analysis"

	"github.com/manuelarte/funcorder/internal/diag"
	"github.com/manuelarte/funcorder/internal/features"
)

// StructHolder contains all the information around a Go struct.
type StructHolder struct {
	// The features to be analyzed
	Features features.Feature

	// The struct declaration
	Struct *ast.TypeSpec

	// A Struct constructor is considered if starts with `New...` and the 1st output parameter is a struct
	Constructors []*ast.FuncDecl

	// Struct methods
	StructMethods []*ast.FuncDecl
}

func (sh *StructHolder) AddConstructor(fn *ast.FuncDecl) {
	sh.Constructors = append(sh.Constructors, fn)
}

func (sh *StructHolder) AddMethod(fn *ast.FuncDecl) {
	sh.StructMethods = append(sh.StructMethods, fn)
}

//nolint:gocognit,nestif // refactor later
func (sh *StructHolder) Analyze() []analysis.Diagnostic {
	// TODO maybe sort constructors and then report also, like NewXXX before MustXXX

	slices.SortFunc(sh.StructMethods, func(a, b *ast.FuncDecl) int {
		return cmp.Compare(a.Pos(), b.Pos())
	})

	var reports []analysis.Diagnostic

	if sh.Features.IsEnabled(features.ConstructorCheck) {
		structPos := sh.Struct.Pos()

		for _, c := range sh.Constructors {
			if c.Pos() < structPos {
				reports = append(reports, diag.NewConstructorNotAfterStructType(sh.Struct, c))
			}

			if len(sh.StructMethods) > 0 && c.Pos() > sh.StructMethods[0].Pos() {
				reports = append(reports, diag.NewConstructorNotBeforeStructMethod(sh.Struct, c, sh.StructMethods[0]))
			}
		}
	}

	if sh.Features.IsEnabled(features.StructMethodCheck) {
		var lastExportedMethod *ast.FuncDecl

		for _, m := range sh.StructMethods {
			if !m.Name.IsExported() {
				continue
			}

			if lastExportedMethod == nil {
				lastExportedMethod = m
			}

			if lastExportedMethod.Pos() < m.Pos() {
				lastExportedMethod = m
			}
		}

		if lastExportedMethod != nil {
			for _, m := range sh.StructMethods {
				if m.Name.IsExported() || m.Pos() >= lastExportedMethod.Pos() {
					continue
				}

				reports = append(reports, diag.NewNonExportedMethodBeforeExportedForStruct(sh.Struct, m, lastExportedMethod))
			}
		}
	}

	// TODO also check that the methods are declared after the struct
	return reports
}
