package traceql

import (
	"strings"
)

const emptyQuery = "{}"

// ExtractConditions extracts filter conditions from a query string.
// It parses the query using the lenient parser (which handles incomplete matchers like `.foo=`)
// and walks the AST to extract conditions. Conditions with OpNone (from incomplete matchers) are filtered out.
//
// Returns nil if:
//   - The query is empty
//   - Parsing fails completely
//   - The query contains structural operators (multiple spansets)
//   - The conditions use OR (AllConditions is false)
//   - No valid matchers can be extracted
func ExtractConditions(query string) ([][]Condition, *SpansetFilter) {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return nil, nil
	}

	expr, err := ParseLenient(query)
	if err != nil {
		return nil, nil
	}

	// Find the first SpansetFilter in the pipeline.
	// Returns nil for structural operators (SpansetOperation) indicating multiple spansets.
	filter := findSpansetFilter(expr.Pipeline)
	if filter == nil {
		return nil, nil
	}

	return SplitReqConditions(filter.Expression), filter
}

// ExtractMatchers extracts matchers from a query string and returns a string
// that can be parsed by the storage layer. It uses ExtractConditions internally.
func ExtractMatchers(query string) string {
	_, filter := ExtractConditions(query)
	if filter == nil {
		return emptyQuery
	}

	return filter.String()
}

func RemoveUnnecessaryParentheses(query string) string {
	findNextCloseParens := func(s string, start int) int {
		depth := 0
		for i := start; i < len(s); i++ {
			switch s[i] {
			case '(':
				depth++
			case ')':
				if depth == 0 {
					return i
				}
				depth--
			}
		}
		return -1
	}
	for char := 0; char < len(query); char++ {
		if query[char] == '(' {
			closeParensIdx := findNextCloseParens(query, char+1)
			if closeParensIdx != -1 && closeParensIdx != char+1 {
				// Check if the parentheses are around a simple matcher (e.g., (.foo = "bar"))
				inside := query[char+1 : closeParensIdx]
				if !strings.Contains(inside, "&&") && !strings.Contains(inside, "||") {
					if !strings.Contains(inside, "=") && !strings.Contains(inside, ">") &&
						!strings.Contains(inside, "<") && !strings.Contains(inside, "!") &&
						!strings.Contains(inside, "~") {
						continue
					}
					// Remove the parentheses
					query = query[:char] + inside + query[closeParensIdx+1:]
					// Move back the index to account for removed parentheses
					char -= 1
				}
			}
		}
	}
	return query
}

// findSpansetFilter returns the first SpansetFilter in the pipeline.
// Returns nil if the pipeline contains structural operators (SpansetOperation)
// which indicate multiple spansets.
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

type ConditionOperation struct {
	Type       Operator
	Conditions [][]Condition
}

func SplitReqConditions(expr FieldExpression) [][]Condition {
	var ConditionOperations []ConditionOperation
	flattenExprToOperations(expr, &ConditionOperations, nil, OpNone)

	// the idea is every OR group of conditions we see, we will generate a new group
	// ex. { (.attr1 = "a" || .attr2 = "b") && .attr3 = "c" }
	// will generate 2 group, one for .attr1 = "a" && .attr3 = "c" and another for .attr2 = "b" && .attr3 = "c"
	// each group will be an iterator, and the results of all iterators will be merged together

	totalGroups := 1
	for _, op := range ConditionOperations {
		if op.Type == OpOr {
			totalGroups *= len(op.Conditions)
		}
	}

	// each group will be one iterator
	conditionGroups := make([][]Condition, totalGroups)
	repeats := 1
	for _, op := range ConditionOperations {
		opCondIdx := 0
		repeated := 0
		for i := 0; i < totalGroups; i++ {
			if op.Type == OpOr {
				conditionGroups[i] = append(conditionGroups[i], op.Conditions[opCondIdx]...)
				repeated++
				if repeated == repeats {
					repeated = 0
					opCondIdx++
				}
				if opCondIdx >= len(op.Conditions) {
					opCondIdx = 0
				}
			} else {
				for _, conds := range op.Conditions {
					conditionGroups[i] = append(conditionGroups[i], conds...)
				}
			}
		}
		if op.Type == OpOr {
			repeats *= len(op.Conditions)
		}
	}
	return conditionGroups
}

func flattenExprToOperations(expr FieldExpression, operators *[]ConditionOperation, parentConditionOperation *ConditionOperation, parentOp Operator) {
	switch e := expr.(type) {
	case *BinaryOperation:
		if e.Op == OpAnd || e.Op == OpOr {
			if parentOp == OpOr {
				if e.Op == OpAnd {
					LHS := []ConditionOperation{}
					RHS := []ConditionOperation{}
					flattenExprToOperations(e.LHS, &LHS, nil, e.Op)
					flattenExprToOperations(e.RHS, &RHS, nil, e.Op)
					conditions := []Condition{}
					for _, op := range LHS {
						for _, conds := range op.Conditions {
							conditions = append(conditions, conds...)
						}
					}
					for _, op := range RHS {
						for _, conds := range op.Conditions {
							conditions = append(conditions, conds...)
						}
					}
					parentConditionOperation.Conditions = append(parentConditionOperation.Conditions, conditions)
					return
				} else {
					flattenExprToOperations(e.LHS, operators, parentConditionOperation, e.Op)
					flattenExprToOperations(e.RHS, operators, parentConditionOperation, e.Op)
					return
				}
			}
			if e.Op == OpAnd {
				LHS := []ConditionOperation{}
				RHS := []ConditionOperation{}
				flattenExprToOperations(e.LHS, &LHS, nil, e.Op)
				flattenExprToOperations(e.RHS, &RHS, nil, e.Op)
				*operators = append(*operators, LHS...)
				*operators = append(*operators, RHS...)
				return
			}
			if e.Op == OpOr {
				SharedOperation := &ConditionOperation{Type: OpOr}
				flattenExprToOperations(e.LHS, operators, SharedOperation, e.Op)
				flattenExprToOperations(e.RHS, operators, SharedOperation, e.Op)
				*operators = append(*operators, *SharedOperation)
				return
			}
		} else {
			req := &FetchSpansRequest{AllConditions: false}
			expr.extractConditions(req)
			if parentConditionOperation != nil {
				parentConditionOperation.Conditions = append(parentConditionOperation.Conditions, req.Conditions)
			} else {
				*operators = append(*operators, ConditionOperation{Type: OpAnd, Conditions: [][]Condition{req.Conditions}})
			}
		}
	case *UnaryOperation:
		req := &FetchSpansRequest{AllConditions: false}
		expr.extractConditions(req)
		if parentConditionOperation != nil {
			parentConditionOperation.Conditions = append(parentConditionOperation.Conditions, req.Conditions)
		} else {
			*operators = append(*operators, ConditionOperation{Type: OpAnd, Conditions: [][]Condition{req.Conditions}})
		}
	default:
		// Handle other expression types if necessary
	}
}
