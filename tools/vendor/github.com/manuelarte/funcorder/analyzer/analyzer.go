package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/manuelarte/funcorder/internal"
)

const (
	ConstructorCheckName  = "constructor"
	StructMethodCheckName = "struct-method"
	AlphabeticalCheckName = "alphabetical"
)

func NewAnalyzer() *analysis.Analyzer {
	f := funcorder{}

	a := &analysis.Analyzer{
		Name:     "funcorder",
		Doc:      "checks the order of functions, methods, and constructors",
		URL:      "https://github.com/manuelarte/funcorder",
		Run:      f.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}

	a.Flags.BoolVar(&f.constructorCheck, ConstructorCheckName, true,
		"Checks that constructors are placed after the structure declaration.")
	a.Flags.BoolVar(&f.structMethodCheck, StructMethodCheckName, true,
		"Checks if the exported methods of a structure are placed before the unexported ones.")
	a.Flags.BoolVar(&f.alphabeticalCheck, AlphabeticalCheckName, false,
		"Checks if the constructors and/or structure methods are sorted alphabetically.")

	return a
}

type funcorder struct {
	constructorCheck  bool
	structMethodCheck bool
	alphabeticalCheck bool
}

func (f *funcorder) run(pass *analysis.Pass) (any, error) {
	var enabledCheckers internal.Feature
	if f.constructorCheck {
		enabledCheckers.Enable(internal.ConstructorCheck)
	}

	if f.structMethodCheck {
		enabledCheckers.Enable(internal.StructMethodCheck)
	}

	if f.alphabeticalCheck {
		enabledCheckers.Enable(internal.AlphabeticalCheck)
	}

	fp := internal.NewFileProcessor(pass.Fset, enabledCheckers)

	insp, found := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !found {
		//nolint:nilnil // impossible case.
		return nil, nil
	}

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.TypeSpec)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.File:
			for _, report := range fp.Analyze() {
				pass.Report(report)
			}

			fp.NewFileNode(node)

		case *ast.FuncDecl:
			fp.NewFuncDecl(node)

		case *ast.TypeSpec:
			fp.NewTypeSpec(node)
		}
	})

	for _, report := range fp.Analyze() {
		pass.Report(report)
	}

	//nolint:nilnil //any, error
	return nil, nil
}
