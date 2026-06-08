package internal

import (
	"cmp"
	"go/ast"
	"go/token"
	"slices"

	"golang.org/x/tools/go/analysis"
)

type (
	ExportedMethods   []*ast.FuncDecl
	UnexportedMethods []*ast.FuncDecl
)

// StructHolder contains all the information around a Go struct.
type StructHolder struct {
	// The fileset
	Fset *token.FileSet
	// The features to be analyzed
	Features Feature

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

// Analyze applies the linter to the struct holder.
func (sh *StructHolder) Analyze() []analysis.Diagnostic {
	// TODO maybe sort constructors and then report also, like NewXXX before MustXXX
	slices.SortFunc(sh.StructMethods, func(a, b *ast.FuncDecl) int {
		return cmp.Compare(a.Pos(), b.Pos())
	})

	var reports []analysis.Diagnostic

	if sh.Features.IsEnabled(ConstructorCheck) {
		reports = append(reports, sh.analyzeConstructor()...)
	}

	if sh.Features.IsEnabled(StructMethodCheck) {
		reports = append(reports, sh.analyzeStructMethod()...)
	}

	// TODO also check that the methods are declared after the struct
	return reports
}

func (sh *StructHolder) analyzeConstructor() []analysis.Diagnostic {
	var reports []analysis.Diagnostic

	for i, constructor := range sh.Constructors {
		if constructor.Pos() < sh.Struct.Pos() {
			reports = append(reports, NewConstructorNotAfterStructType(sh.Struct, constructor))
		}

		if len(sh.StructMethods) > 0 && constructor.Pos() > sh.StructMethods[0].Pos() {
			reports = append(reports, NewConstructorNotBeforeStructMethod(sh.Struct, constructor, sh.StructMethods[0]))
		}

		if sh.Features.IsEnabled(AlphabeticalCheck) &&
			i < len(sh.Constructors)-1 && sh.Constructors[i].Name.Name > sh.Constructors[i+1].Name.Name {
			reports = append(reports,
				NewAdjacentConstructorsNotSortedAlphabetically(sh.Struct, sh.Constructors[i], sh.Constructors[i+1]),
			)
		}
	}

	return reports
}

func (sh *StructHolder) analyzeStructMethod() []analysis.Diagnostic {
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

	var reports []analysis.Diagnostic

	if lastExportedMethod != nil {
		for _, m := range sh.StructMethods {
			if m.Name.IsExported() || m.Pos() >= lastExportedMethod.Pos() {
				continue
			}

			reports = append(reports, NewUnexportedMethodBeforeExportedForStruct(sh.Struct, m, lastExportedMethod))
		}
	}

	if sh.Features.IsEnabled(AlphabeticalCheck) {
		exported, unexported := SplitExportedUnexported(sh.StructMethods)
		reports = slices.Concat(reports,
			sortDiagnostics(sh.Struct, exported),
			sortDiagnostics(sh.Struct, unexported),
		)
	}

	return reports
}

func sortDiagnostics(typeSpec *ast.TypeSpec, funcDecls []*ast.FuncDecl) []analysis.Diagnostic {
	var reports []analysis.Diagnostic

	for i := range funcDecls {
		if i >= len(funcDecls)-1 {
			continue
		}

		if funcDecls[i].Name.Name > funcDecls[i+1].Name.Name {
			reports = append(reports,
				NewAdjacentStructMethodsNotSortedAlphabetically(typeSpec, funcDecls[i], funcDecls[i+1]))
		}
	}

	return reports
}
