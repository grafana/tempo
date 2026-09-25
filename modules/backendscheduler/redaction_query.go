package backendscheduler

import (
	"errors"
	"fmt"

	"github.com/grafana/tempo/pkg/traceql"
)

// validateRedactionQuery enforces the redaction query subset: a single spanset filter whose
// expression combines, with && / ||, either an = comparison on a resource.*/span.* attribute or an
// `attr != nil` existence check on one. Anything else (value negation, regex, ordered comparisons,
// unscoped attributes, pipelines, aggregates, multiple/structural filters) is rejected at submission.
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

// validateRedactionExpr recursively checks that fe is built only from &&/|| combinators over the
// comparisons the subset allows.
func validateRedactionExpr(fe traceql.FieldExpression) error {
	// `attr != nil` never reaches the operator switch below. The grammar rewrites it into a unary
	// existence check, so it arrives as OpExists rather than as a negation. It selects spans that
	// have the attribute, which is a positive match.
	if un, ok := fe.(traceql.UnaryOperation); ok {
		if un.Op != traceql.OpExists {
			return fmt.Errorf("operator %v not allowed in redaction query", un.Op)
		}
		attr, ok := un.Expression.(traceql.Attribute)
		if !ok {
			return fmt.Errorf("redaction query existence checks must name an attribute, got %T", un.Expression)
		}
		return validateRedactionAttribute(attr)
	}

	bin, ok := fe.(*traceql.BinaryOperation)
	if !ok {
		return fmt.Errorf("redaction query supports only = comparisons and existence checks joined by && / ||, got %T", fe)
	}

	switch bin.Op {
	case traceql.OpAnd, traceql.OpOr:
		if err := validateRedactionExpr(bin.LHS); err != nil {
			return err
		}
		return validateRedactionExpr(bin.RHS)
	case traceql.OpEqual:
		return validateRedactionComparison(bin.LHS, bin.RHS)
	default:
		// Only positive equality and the != nil existence check are allowed. Negation against a
		// value is excluded on purpose: its match set is the complement (potentially all data), so
		// a typo is as catastrophic as a bad regex on an irreversible delete.
		return fmt.Errorf("operator %v not allowed in redaction query; only = and != nil are supported", bin.Op)
	}
}

// validateRedactionAttribute requires an attribute the subset can act on: scoped to resource. or
// span., and not parent-scoped.
func validateRedactionAttribute(attr traceql.Attribute) error {
	if attr.Scope != traceql.AttributeScopeResource && attr.Scope != traceql.AttributeScopeSpan {
		return fmt.Errorf("redaction query attributes must be scoped to resource. or span., got %q", attr.Name)
	}
	// Reject parent.resource.* / parent.span.*: these reference the ancestor span, i.e.
	// structural filtering, which is outside the single-span subset and could select
	// traces far beyond the operator's intent.
	if attr.Parent {
		return fmt.Errorf("redaction query attributes must not be parent-scoped (parent.), got %q", attr.Name)
	}
	return nil
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

	return validateRedactionAttribute(attr)
}
