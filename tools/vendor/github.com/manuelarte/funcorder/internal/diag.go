package internal

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

func NewConstructorNotAfterStructType(structSpec *ast.TypeSpec, constructor *ast.FuncDecl) analysis.Diagnostic {
	return analysis.Diagnostic{
		Pos: constructor.Pos(),
		Message: fmt.Sprintf("constructor %q for struct %q should be placed after the struct declaration",
			constructor.Name, structSpec.Name),
		URL: "https://github.com/manuelarte/funcorder?tab=readme-ov-file#check-constructors-functions-are-placed-after-struct-declaration", //nolint:lll // url
	}
}

func NewConstructorNotBeforeStructMethod(
	structSpec *ast.TypeSpec,
	constructor *ast.FuncDecl,
	method *ast.FuncDecl,
) analysis.Diagnostic {
	return analysis.Diagnostic{
		Pos: constructor.Pos(),
		URL: "https://github.com/manuelarte/funcorder?tab=readme-ov-file#check-constructors-functions-are-placed-after-struct-declaration", //nolint:lll // url
		Message: fmt.Sprintf("constructor %q for struct %q should be placed before struct method %q",
			constructor.Name, structSpec.Name, method.Name),
	}
}

func NewAdjacentConstructorsNotSortedAlphabetically(
	structSpec *ast.TypeSpec,
	constructorNotSorted *ast.FuncDecl,
	otherConstructorNotSorted *ast.FuncDecl,
) analysis.Diagnostic {
	return analysis.Diagnostic{
		Pos: otherConstructorNotSorted.Pos(),
		URL: "https://github.com/manuelarte/funcorder?tab=readme-ov-file#check-constructorsmethods-are-sorted-alphabetically",
		Message: fmt.Sprintf("constructor %q for struct %q should be placed before constructor %q",
			otherConstructorNotSorted.Name, structSpec.Name, constructorNotSorted.Name),
	}
}

func NewUnexportedMethodBeforeExportedForStruct(
	structSpec *ast.TypeSpec,
	privateMethod *ast.FuncDecl,
	publicMethod *ast.FuncDecl,
) analysis.Diagnostic {
	return analysis.Diagnostic{
		Pos: privateMethod.Pos(),
		URL: "https://github.com/manuelarte/funcorder?tab=readme-ov-file#check-exported-methods-are-placed-before-unexported-methods", //nolint:lll // url
		Message: fmt.Sprintf("unexported method %q for struct %q should be placed after the exported method %q",
			privateMethod.Name, structSpec.Name, publicMethod.Name),
	}
}

func NewAdjacentStructMethodsNotSortedAlphabetically(
	structSpec *ast.TypeSpec,
	method *ast.FuncDecl,
	otherMethod *ast.FuncDecl,
) analysis.Diagnostic {
	return analysis.Diagnostic{
		Pos: otherMethod.Pos(),
		URL: "https://github.com/manuelarte/funcorder?tab=readme-ov-file#check-constructorsmethods-are-sorted-alphabetically",
		Message: fmt.Sprintf("method %q for struct %q should be placed before method %q",
			otherMethod.Name, structSpec.Name, method.Name),
	}
}
