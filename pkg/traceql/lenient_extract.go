package traceql

import (
	"strings"
)

const emptyQuery = "{}"

// ExtractConditionGroups parses a query string using the lenient parser and returns
// a groups of conditions. Conditions with OpNone (from incomplete matchers) are filtered out.
//
// Returns nil if the query is empty or parsing fails completely.
func ExtractConditionGroups(query string) [][]Condition {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return nil
	}

	expr, err := ParseLenient(query)
	if err != nil {
		return nil
	}
	if err := expr.validate(); err != nil {
		return nil
	}

	s := expr.String()
	if s == "{ true }" {
		return nil
	}

	// Find the first SpansetFilter in the pipeline.
	// Returns nil for structural operators (SpansetOperation) indicating multiple spansets.
	filter := findSpansetFilter(expr.Pipeline)
	if filter == nil {
		return nil
	}

	groups := splitReqConditions(filter.Expression)
	for i := range groups {
		// Filter out OpNone conditions — these are column-fetch hints (bare attributes,
		// structural intrinsics), not filterable conditions.
		conditions := make([]Condition, 0, len(groups[i]))
		for _, cond := range groups[i] {
			if cond.Op != OpNone {
				conditions = append(conditions, cond)
			}
		}
		groups[i] = conditions
	}

	// if even one group has zero conditions after filtering, treat the whole query as empty (e.g. `{.attr || .foo}`)
	for _, g := range groups {
		if len(g) == 0 {
			return nil
		}
	}

	return groups
}

// NormalizeQuery parses a query string using the lenient parser and returns
// a normalized string representation. Used for cache key generation.
// Match-all queries (e.g. "{}", "{ }", "{ true }") are normalized to "{}"
// to avoid cache fragmentation.
func NormalizeQuery(query string) string {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return emptyQuery
	}

	expr, err := ParseLenient(query)
	if err != nil {
		return emptyQuery
	}

	if err := expr.validate(); err != nil {
		return emptyQuery
	}

	s := expr.String()
	if s == "{ true }" {
		return emptyQuery
	}
	return s
}

func findSpansetFilter(p Pipeline) *SpansetFilter {
	if len(p.Elements) == 0 {
		return nil
	}

	switch e := p.Elements[0].(type) {
	case *SpansetFilter:
		return e
	case Pipeline:
		return findSpansetFilter(e)
	default:
		// SpansetOperation, ScalarFilter, etc. - not a simple spanset filter
		return nil
	}
}

type conditionOperation struct {
	opType     Operator
	conditions [][]Condition
}

// splitReqConditions converts a field expression into condition groups by expanding
// OR clauses into separate groups. Each group represents one AND-connected set of
// conditions that must all be satisfied together.
//
// Example: { (.a="1" || .b="2") && .c="3" } produces two groups:
//   - [.a="1", .c="3"]
//   - [.b="2", .c="3"]
func splitReqConditions(expr FieldExpression) [][]Condition {
	var ops []conditionOperation
	flattenExprToOperations(expr, &ops, nil, OpNone)

	totalGroups := 1
	for _, op := range ops {
		if op.opType == OpOr {
			totalGroups *= len(op.conditions)
		}
	}

	conditionGroups := make([][]Condition, totalGroups)
	repeats := 1
	for _, op := range ops {
		opCondIdx := 0
		repeated := 0
		for i := 0; i < totalGroups; i++ {
			if op.opType == OpOr {
				conditionGroups[i] = append(conditionGroups[i], op.conditions[opCondIdx]...)
				repeated++
				if repeated == repeats {
					repeated = 0
					opCondIdx++
				}
				if opCondIdx >= len(op.conditions) {
					opCondIdx = 0
				}
			} else {
				for _, conds := range op.conditions {
					conditionGroups[i] = append(conditionGroups[i], conds...)
				}
			}
		}
		if op.opType == OpOr {
			repeats *= len(op.conditions)
		}
	}

	// Drop empty groups (e.g. from an empty filter `{}`).
	result := conditionGroups[:0]
	for _, g := range conditionGroups {
		if len(g) > 0 {
			result = append(result, g)
		}
	}
	return result
}

// flattenExprToOperations walks a FieldExpression AST and collects conditionOperations:
//   - OpAnd operations collect leaf conditions into a single AND group
//   - OpOr operations collect their branches as separate condition slices in an OR group
//
// The parent context (operators list, parentOp, parentCondOp) tracks whether we are
// inside an OR branch so nested ANDs can be collected into a single OR branch entry.
func flattenExprToOperations(expr FieldExpression, operators *[]conditionOperation, parentCondOp *conditionOperation, parentOp Operator) {
	e, ok := expr.(*BinaryOperation)
	if !ok {
		// Leaf node (UnaryOperation or other): extract its conditions.
		req := &FetchSpansRequest{AllConditions: false}
		expr.extractConditions(req)
		if parentCondOp != nil {
			parentCondOp.conditions = append(parentCondOp.conditions, req.Conditions)
		} else {
			*operators = append(*operators, conditionOperation{opType: OpAnd, conditions: [][]Condition{req.Conditions}})
		}
		return
	}

	switch {
	case (e.Op == OpAnd || e.Op == OpOr) && parentOp == OpOr && e.Op == OpAnd:
		// AND nested directly inside an OR branch: collect both sides into one flat
		// condition slice so they become a single OR branch entry.
		var lhs, rhs []conditionOperation
		flattenExprToOperations(e.LHS, &lhs, nil, e.Op)
		flattenExprToOperations(e.RHS, &rhs, nil, e.Op)
		var combined []Condition
		for _, op := range lhs {
			for _, conds := range op.conditions {
				combined = append(combined, conds...)
			}
		}
		for _, op := range rhs {
			for _, conds := range op.conditions {
				combined = append(combined, conds...)
			}
		}
		parentCondOp.conditions = append(parentCondOp.conditions, combined)

	case (e.Op == OpAnd || e.Op == OpOr) && parentOp == OpOr && e.Op == OpOr:
		// OR nested inside an OR branch: keep recursing into the same parent OR group.
		flattenExprToOperations(e.LHS, operators, parentCondOp, e.Op)
		flattenExprToOperations(e.RHS, operators, parentCondOp, e.Op)

	case e.Op == OpAnd:
		// Top-level AND: distribute both sides as independent AND operations.
		var lhs, rhs []conditionOperation
		flattenExprToOperations(e.LHS, &lhs, nil, e.Op)
		flattenExprToOperations(e.RHS, &rhs, nil, e.Op)
		*operators = append(*operators, lhs...)
		*operators = append(*operators, rhs...)

	case e.Op == OpOr:
		// Top-level OR: create a new OR group and collect branches into it.
		sharedOp := &conditionOperation{opType: OpOr}
		flattenExprToOperations(e.LHS, operators, sharedOp, e.Op)
		flattenExprToOperations(e.RHS, operators, sharedOp, e.Op)
		*operators = append(*operators, *sharedOp)

	default:
		// Leaf binary operation (comparison etc.): extract its conditions.
		req := &FetchSpansRequest{AllConditions: false}
		expr.extractConditions(req)
		if parentCondOp != nil {
			parentCondOp.conditions = append(parentCondOp.conditions, req.Conditions)
		} else {
			*operators = append(*operators, conditionOperation{opType: OpAnd, conditions: [][]Condition{req.Conditions}})
		}
	}
}
