// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

var _ localIdentifierScopeVisitor = (*grammarPathVisitor)(nil)

// grammarPathVisitor is used to extract all paths from a parsedStatement or booleanExpression
// localIdentifierDecl paths are not included on the grammarPathVisitor.paths results.
type grammarPathVisitor struct {
	paths  []path
	scopes localScopeStack
}

func (*grammarPathVisitor) visitEditor(*editor)                   {}
func (*grammarPathVisitor) visitConverter(*converter)             {}
func (*grammarPathVisitor) visitValue(*value)                     {}
func (*grammarPathVisitor) visitMathExprLiteral(*mathExprLiteral) {}
func (*grammarPathVisitor) visitLambdaBody(*lambdaBody)           {}

func (v *grammarPathVisitor) pushLocalIdentifiers(params []localIdentifierDecl) {
	v.scopes.push(localIdentifiersDeclToFrame(params))
}

func (v *grammarPathVisitor) popLocalIdentifiers() {
	v.scopes.pop()
}

func (v *grammarPathVisitor) visitPath(value *path) {
	if !value.inScope(v.scopes) {
		v.paths = append(v.paths, *value)
	}
}

func getParsedStatementPaths(ps *parsedStatement) []path {
	visitor := &grammarPathVisitor{}
	ps.Editor.accept(visitor)
	if ps.WhereClause != nil {
		ps.WhereClause.accept(visitor)
	}
	return visitor.paths
}

func getBooleanExpressionPaths(be *booleanExpression) []path {
	visitor := &grammarPathVisitor{}
	be.accept(visitor)
	return visitor.paths
}

func getValuePaths(v *value) []path {
	visitor := &grammarPathVisitor{}
	v.accept(visitor)
	return visitor.paths
}
