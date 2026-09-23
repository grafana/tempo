package backendscheduler

import (
	"errors"
	"fmt"

	"github.com/grafana/tempo/v3/pkg/traceql"
)

// validateRedactionQuery enforces the redaction query subset: a single spanset filter whose
// expression combines, with && / ||, either an = comparison on a resource.*/span.* attribute or an
// existence check on one (`!= nil`, `!= ""`, `> ""`). Anything else (value negation, regex, ordered comparisons, unscoped
// attributes, pipelines, aggregates, multiple/structural filters) is rejected at submission.
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
	case traceql.OpNotEqual:
		// != is allowed against the empty string only, where it is an existence check rather than a
		// negation. TraceQL resolves a missing attribute to nil, and Static.NotEquals returns false
		// whenever either side is nil, so `attr != ""` selects spans where attr is present and
		// non-empty; it does not select the spans that lack it.
		//
		// Against any other value the match set is the complement, potentially all data, so a typo
		// is as catastrophic as a bad regex.
		if err := validateRedactionComparison(bin.LHS, bin.RHS); err != nil {
			return err
		}
		if !isEmptyString(bin.LHS) && !isEmptyString(bin.RHS) {
			return errors.New(`!= is allowed only against "" (attr != "" selects spans where attr is present and non-empty); use = to match a value`)
		}
		return nil
	case traceql.OpGreater:
		// `attr > ""` spells the same existence check by a different route: every non-empty string
		// sorts after "", and an absent attribute resolves to nil, which binaryTypeValid refuses for
		// ordered operators outright, so the comparison is false rather than matching. A present
		// non-string operand fails the matching-operand check for the same reason.
		//
		// Orientation matters here in a way it does not for !=, which is symmetric: `"" > attr`
		// compares the other way and matches nothing at all, so it is refused rather than accepted as
		// a query that silently selects no traces.
		if err := validateRedactionComparison(bin.LHS, bin.RHS); err != nil {
			return err
		}
		if !isEmptyString(bin.RHS) {
			return errors.New(`> is allowed only as attr > "" (which selects spans where attr is present and non-empty)`)
		}
		return nil
	default:
		// >= and <= against "" are bounded the same way, but they are not existence checks: `attr >= ""`
		// also matches an empty value, which `attr != nil` already says, and `attr <= ""` matches only
		// the empty value. `< ""` matches nothing. None are worth widening the subset for.
		return fmt.Errorf(`operator %v not allowed in redaction query; only =, != "", > "" and != nil are supported`, bin.Op)
	}
}

// isEmptyString reports whether fe is the empty string literal.
func isEmptyString(fe traceql.FieldExpression) bool {
	st, ok := fe.(traceql.Static)
	return ok && st.Type == traceql.TypeString && st.EncodeToString(false) == ""
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
