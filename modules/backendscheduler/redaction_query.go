package backendscheduler

import (
	"errors"
	"fmt"

	"github.com/grafana/tempo/v3/pkg/traceql"
)

// validateRedactionQuery enforces the redaction query subset: a single spanset filter that is an &&
// of conditions on resource.*/span.* attributes, at least one of them positive (see
// validateRedactionExpr). Everything else is rejected at submission.
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

// validateRedactionExpr requires fe to be an && of conditions with at least one positive condition:
// an = comparison, or an || of = comparisons. The other conditions only narrow that match, so a
// typo in one of them cannot widen the deletion beyond the positive anchor.
func validateRedactionExpr(fe traceql.FieldExpression) error {
	hasPositive := false
	for _, cond := range redactionConditions(fe) {
		positive, err := validateRedactionCondition(cond)
		if err != nil {
			return err
		}
		hasPositive = hasPositive || positive
	}
	if !hasPositive {
		return errors.New("redaction query must include at least one = comparison (or an || of = comparisons) joined by &&")
	}
	return nil
}

// redactionConditions flattens a tree of && into its conditions.
func redactionConditions(fe traceql.FieldExpression) []traceql.FieldExpression {
	if bin, ok := fe.(*traceql.BinaryOperation); ok && bin.Op == traceql.OpAnd {
		return append(redactionConditions(bin.LHS), redactionConditions(bin.RHS)...)
	}
	return []traceql.FieldExpression{fe}
}

// validateRedactionCondition checks one &&-joined condition and reports whether it is positive.
func validateRedactionCondition(cond traceql.FieldExpression) (positive bool, err error) {
	// The grammar rewrites `attr != nil` into a unary OpExists, so it never reaches the
	// operator switch below as a negation.
	if un, ok := cond.(traceql.UnaryOperation); ok {
		if un.Op != traceql.OpExists {
			return false, fmt.Errorf("operator %v not allowed in redaction query", un.Op)
		}
		attr, ok := un.Expression.(traceql.Attribute)
		if !ok {
			return false, fmt.Errorf("redaction query existence checks must name an attribute, got %T", un.Expression)
		}
		return false, validateRedactionAttribute(attr)
	}

	bin, ok := cond.(*traceql.BinaryOperation)
	if !ok {
		return false, fmt.Errorf("redaction query supports only comparisons and existence checks joined by &&, got %T", cond)
	}

	switch bin.Op {
	case traceql.OpEqual:
		return true, validateRedactionComparison(bin.LHS, bin.RHS)
	case traceql.OpOr:
		return true, validateRedactionEqualityGroup(bin)
	default:
		// Negation against a value is excluded on purpose: its match set is the complement
		// (potentially all data), as catastrophic as a bad regex on an irreversible delete.
		return false, fmt.Errorf("operator %v not allowed in redaction query; only = and != nil are supported", bin.Op)
	}
}

// validateRedactionEqualityGroup requires every branch of an || to be an = comparison.
func validateRedactionEqualityGroup(fe traceql.FieldExpression) error {
	if bin, ok := fe.(*traceql.BinaryOperation); ok {
		switch bin.Op {
		case traceql.OpOr:
			if err := validateRedactionEqualityGroup(bin.LHS); err != nil {
				return err
			}
			return validateRedactionEqualityGroup(bin.RHS)
		case traceql.OpEqual:
			return validateRedactionComparison(bin.LHS, bin.RHS)
		}
	}
	return errors.New("|| in a redaction query may only join = comparisons")
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
