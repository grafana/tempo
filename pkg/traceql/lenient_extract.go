package traceql

import (
	"errors"
	"fmt"
	"strings"
)

// ErrMaxConditionGroupsPerTagQueryReached is returned when OR expansion exceeds the configured limit.
// Use errors.Is to check for this error; the message includes the configured limit for operator visibility.
var ErrMaxConditionGroupsPerTagQueryReached = errors.New("maximum condition groups reached")

const emptyQuery = "{}"

// DefaultMaxConditionGroupsPerTagQuery is the default cap on the number of OR-expanded condition groups to prevent
// exponential blowup from queries with many OR clauses. Configurable via overrides.
const DefaultMaxConditionGroupsPerTagQuery = 100

// ExtractConditionGroups parses a query string using the lenient parser and returns
// groups of conditions. Conditions with OpNone (from incomplete matchers) are filtered out.
//
// Returns nil if the query is empty or parsing fails completely.
// Returns nil, ErrMaxConditionGroupsPerTagQueryReached if the number of groups exceeds maxGroups.
func ExtractConditionGroups(query string, maxGroups int) ([][]Condition, error) {
	if maxGroups <= 0 {
		maxGroups = DefaultMaxConditionGroupsPerTagQuery
	}
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return nil, nil
	}

	expr, err := ParseLenient(query)
	if err != nil {
		return nil, nil
	}
	if err := expr.validate(); err != nil {
		return nil, nil
	}

	// Find the first SpansetFilter in the pipeline.
	// Returns nil for structural operators (SpansetOperation) indicating multiple spansets.
	filter := findSpansetFilter(expr.Pipeline)
	if filter == nil {
		return nil, nil
	}

	groups, reachedMaxGroupsInSplitConditions := splitReqConditions(filter.Expression, maxGroups)
	for i := range groups {
		// Filter out OpNone conditions — these are column-fetch hints (bare attributes,
		// structural intrinsics), not filterable conditions.
		conditions := make([]Condition, 0, len(groups[i]))
		for _, cond := range groups[i] {
			if cond.Op != OpNone {
				conditions = append(conditions, cond)
			}
		}

		// if even one group has zero conditions after filtering, treat the whole query as empty (e.g. `{.attr || .foo}`)
		if len(conditions) == 0 {
			return nil, nil
		}
		groups[i] = conditions
	}

	if reachedMaxGroupsInSplitConditions {
		return nil, fmt.Errorf("%w (limit: %d). Reduce the number of OR conditions in the query", ErrMaxConditionGroupsPerTagQueryReached, maxGroups)
	}

	return groups, nil
}

// IsEmptyQuery returns true if the query is empty or a match-all (e.g. "{}", "{ }", "{ true }").
func IsEmptyQuery(query string) bool {
	query = strings.ReplaceAll(query, " ", "")
	return query == "" || query == "{}" || query == "{true}"
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
func splitReqConditions(expr FieldExpression, maxGroups int) ([][]Condition, bool) {
	var reachedMaxGroups bool
	ops := flattenExprToOperations(expr, OpNone)

	totalGroups := 1
	for _, op := range ops {
		if op.opType == OpOr {
			totalGroups *= len(op.conditions)
			if totalGroups > maxGroups {
				totalGroups = maxGroups
				break
			}
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
			if repeats > maxGroups {
				repeats = maxGroups
				reachedMaxGroups = true
			}
		}
	}

	// Drop empty groups (e.g. from an empty filter `{}`).
	result := conditionGroups[:0]
	for _, g := range conditionGroups {
		if len(g) > 0 {
			result = append(result, g)
		}
	}

	return result, reachedMaxGroups
}

// deduplicateConditionBranches removes duplicate branches from an OR group.
// Two branches are considered equal if they contain the same conditions in the same order.
// Fields are separated by \x00 and conditions by \x01 to avoid key collisions.
func deduplicateConditionBranches(branches [][]Condition) [][]Condition {
	seen := make(map[string]struct{}, len(branches))
	result := branches[:0]
	for _, branch := range branches {
		var sb strings.Builder
		for _, c := range branch {
			sb.WriteString(c.Attribute.String())
			sb.WriteByte('\x00')
			sb.WriteString(c.Op.String())
			for _, o := range c.Operands {
				// Use %q to escape any bytes (including \x00/\x01 separators) so operand
				// values cannot cause false key collisions.
				fmt.Fprintf(&sb, "\x00%q", o.String())
			}
			sb.WriteByte('\x01')
		}
		key := sb.String()
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			result = append(result, branch)
		}
	}
	return result
}

// flattenExprToOperations walks a FieldExpression AST and returns conditionOperations:
//   - OpAnd operations collect leaf conditions into a single AND group
//   - OpOr operations collect their branches as separate condition slices in an OR group
//
// parentOp tracks whether we are inside an OR branch so nested ANDs can be collected
// into a single OR branch entry. When parentOp == OpOr each returned conditionOperation
// has exactly one conditions entry representing one OR branch.
func flattenExprToOperations(expr FieldExpression, parentOp Operator) []conditionOperation {
	e, ok := expr.(*BinaryOperation)
	if !ok {
		// Leaf node (UnaryOperation or other): extract its conditions.
		req := &FetchSpansRequest{AllConditions: false}
		expr.extractConditions(req)
		return []conditionOperation{{opType: OpAnd, conditions: [][]Condition{req.Conditions}}}
	}

	switch {
	case parentOp == OpOr && e.Op == OpAnd:
		// AND nested directly inside an OR branch: collect both sides into one flat
		// condition slice so they become a single OR branch entry.
		lhs := flattenExprToOperations(e.LHS, e.Op)
		rhs := flattenExprToOperations(e.RHS, e.Op)
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
		return []conditionOperation{{opType: OpAnd, conditions: [][]Condition{combined}}}

	case parentOp == OpOr && e.Op == OpOr:
		// OR nested inside an OR branch: keep recursing, accumulating branches.
		lhs := flattenExprToOperations(e.LHS, e.Op)
		rhs := flattenExprToOperations(e.RHS, e.Op)
		return append(lhs, rhs...)

	case e.Op == OpAnd:
		// Top-level AND: distribute both sides as independent AND operations.
		lhs := flattenExprToOperations(e.LHS, e.Op)
		rhs := flattenExprToOperations(e.RHS, e.Op)
		return append(lhs, rhs...)

	case e.Op == OpOr:
		// Top-level OR: collect branches from both sides into a single OR group.
		lhs := flattenExprToOperations(e.LHS, e.Op)
		rhs := flattenExprToOperations(e.RHS, e.Op)
		orOp := conditionOperation{opType: OpOr}
		for _, op := range append(lhs, rhs...) {
			orOp.conditions = append(orOp.conditions, op.conditions...)
		}
		orOp.conditions = deduplicateConditionBranches(orOp.conditions)
		return []conditionOperation{orOp}

	default:
		// Leaf binary operation (comparison etc.): extract its conditions.
		req := &FetchSpansRequest{AllConditions: false}
		expr.extractConditions(req)
		return []conditionOperation{{opType: OpAnd, conditions: [][]Condition{req.Conditions}}}
	}
}
