package backendscheduler

import (
	"errors"
	"fmt"

	"github.com/grafana/tempo/pkg/traceql"
)

// validateRedactionQuery enforces the redaction query subset: a single spanset filter
// whose expression is = / != comparisons on resource.*/span.* attributes combined with
// && / ||. Anything else (regex, ordered comparisons, unscoped attributes, pipelines,
// aggregates, multiple/structural filters) is rejected at submission.
//
// Parsing uses ParseNoOptimizations so the optimizer does not fold an OR of equalities
// on the same attribute into a regex, which the subset would then wrongly reject.
func validateRedactionQuery(query string) error {
	if query == "" {
		return errors.New("redaction query must not be empty")
	}

	expr, err := traceql.ParseNoOptimizations(query)
	if err != nil {
		return fmt.Errorf("invalid TraceQL: %w", err)
	}

	pipeline, ok := expr.SinglePipeline()
	if !ok {
		return errors.New("redaction query must be a single spanset filter (no metrics, math, or multiple pipelines)")
	}
	if len(pipeline.Elements) != 1 {
		return errors.New("redaction query must be a single spanset filter (no pipeline stages)")
	}

	filter, ok := pipeline.Elements[0].(*traceql.SpansetFilter)
	if !ok {
		return fmt.Errorf("redaction query must be a single spanset filter, got %T", pipeline.Elements[0])
	}

	return validateRedactionExpr(filter.Expression)
}

// validateRedactionExpr recursively checks that fe is built only from &&/|| combinators
// over = / != comparisons on scoped attributes.
func validateRedactionExpr(fe traceql.FieldExpression) error {
	bin, ok := fe.(*traceql.BinaryOperation)
	if !ok {
		return fmt.Errorf("redaction query supports only =, != comparisons joined by && / ||, got %T", fe)
	}

	switch bin.Op {
	case traceql.OpAnd, traceql.OpOr:
		if err := validateRedactionExpr(bin.LHS); err != nil {
			return err
		}
		return validateRedactionExpr(bin.RHS)
	case traceql.OpEqual, traceql.OpNotEqual:
		return validateRedactionComparison(bin.LHS, bin.RHS)
	default:
		return fmt.Errorf("operator %v not allowed in redaction query; only =, != are supported", bin.Op)
	}
}

// validateRedactionComparison requires one side to be a resource.*/span.* attribute and
// the other a static literal.
func validateRedactionComparison(lhs, rhs traceql.FieldExpression) error {
	lAttr, lIsAttr := lhs.(traceql.Attribute)
	rAttr, rIsAttr := rhs.(traceql.Attribute)
	_, lIsStatic := lhs.(traceql.Static)
	_, rIsStatic := rhs.(traceql.Static)

	var attr traceql.Attribute
	switch {
	case lIsAttr && rIsStatic:
		attr = lAttr
	case rIsAttr && lIsStatic:
		attr = rAttr
	default:
		return errors.New("redaction query comparisons must be <attribute> = <value>")
	}

	if attr.Scope != traceql.AttributeScopeResource && attr.Scope != traceql.AttributeScopeSpan {
		return fmt.Errorf("redaction query attributes must be scoped to resource. or span., got %q", attr.Name)
	}
	return nil
}
