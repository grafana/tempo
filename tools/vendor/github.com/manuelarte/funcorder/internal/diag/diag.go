package diag

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

func NewConstructorNotAfterStructType(structSpec *ast.TypeSpec, constructor *ast.FuncDecl) analysis.Diagnostic {
	return analysis.Diagnostic{
		Pos: constructor.Pos(),
		Message: fmt.Sprintf("function %q for struct %q should be placed after the struct declaration",
			constructor.Name, structSpec.Name),
	}
}

func NewConstructorNotBeforeStructMethod(
	structSpec *ast.TypeSpec,
	constructor *ast.FuncDecl,
	method *ast.FuncDecl,
) analysis.Diagnostic {
	return analysis.Diagnostic{
		Pos: constructor.Pos(),
		Message: fmt.Sprintf("constructor %q for struct %q should be placed before struct method %q",
			constructor.Name, structSpec.Name, method.Name),
	}
}

func NewNonExportedMethodBeforeExportedForStruct(
	structSpec *ast.TypeSpec,
	privateMethod *ast.FuncDecl,
	publicMethod *ast.FuncDecl,
) analysis.Diagnostic {
	return analysis.Diagnostic{
		Pos: privateMethod.Pos(),
		Message: fmt.Sprintf("unexported method %q for struct %q should be placed after the exported method %q",
			privateMethod.Name, structSpec.Name, publicMethod.Name),
	}
}
