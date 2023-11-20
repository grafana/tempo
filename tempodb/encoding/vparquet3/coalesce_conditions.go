package vparquet3

import "github.com/grafana/tempo/pkg/traceql"

// coalesceConditions reduces the amount of data pulled from the backend when the same column is used in multiple conditions
//
//	todo: There is a lot that can be done with this method. for example we can coalesce conditions if they have equivalent
//	operands. consider a query like: { span.foo >= 1 } && { span.foo = 1 } the conditions could be coalesced to only
//	the >= condition. it also is currently ignoring situations where allconditions is true, but some of these
//	improvements apply to that case as well.
func coalesceConditions(f *traceql.FetchSpansRequest) {
	// only do this if all conditions is false for now. it's safer and all conditions queries tend to be quite quick
	if !f.AllConditions {
		prevLen := 0
		for len(f.Conditions) != prevLen { // check combinations until we can't coalesce any more
			prevLen = len(f.Conditions)

			// search for conditions with the same attribute name and consider coalescing them
			for i := 0; i < len(f.Conditions); i++ {
				for j := i + 1; j < len(f.Conditions); j++ {
					if c, ok := coalesce(f.Conditions[i], f.Conditions[j]); ok {
						f.Conditions[i] = c
						f.Conditions = append(f.Conditions[:j], f.Conditions[j+1:]...)
						j--
					}
				}
			}
		}
	}
}

// coalesce takes two conditions and turns them into one. it returns a bool to indicate
// if the returned condition is valid or if it should just continue using the original 2 conditions
//
// it is very difficult to coalesce conditions in a way that will always be more performant at the fetch layer
// consider: { resource.service.name = "foo" || resource.service.name = "bar" }
// if foo and bar are rare then it will be faster to double pull the column and intersect the results. if
// foo and bar are common then it will be faster to single pull the column and allow the engine to do the work.
// currently the fetch layer does not support the = condition with multiple operands. we should add capability
// and then extend this method.
//
// therefore! we will only coalesce conditions in the following cases:
//   - they are exactly the same
//   - they will pull every span anyway. example: { span.foo = "bar" } >> { span.foo != "bar" }
func coalesce(c1 traceql.Condition, c2 traceql.Condition) (traceql.Condition, bool) {
	// if the conditions are exactly the same then we can just return one of them
	if c1.Attribute == c2.Attribute &&
		c1.Op == c2.Op &&
		operandsEqual(c1, c2) {
		return c1, true
	}

	// if the operations are != and = and the operands are the same this is going to pull every row. let's
	// collapse to one condition with OpNone and no operands
	if c1.Attribute == c2.Attribute && // attributes equal
		(c1.Op == traceql.OpEqual && c2.Op == traceql.OpNotEqual || c1.Op == traceql.OpNotEqual && c2.Op == traceql.OpEqual) && // operands are != and =
		operandsEqual(c1, c2) { // operands equal
		return traceql.Condition{Attribute: c1.Attribute, Op: traceql.OpNone, Operands: nil}, true
	}

	// if one of the operations is opnone we're already pulling every row. coalesce
	if c1.Attribute == c2.Attribute && // attributes equal
		(c1.Op == traceql.OpNone || c2.Op == traceql.OpNone) { // one operand is opnone
		return traceql.Condition{Attribute: c1.Attribute, Op: traceql.OpNone, Operands: nil}, true
	}

	return traceql.Condition{}, false
}

func operandsEqual(c1 traceql.Condition, c2 traceql.Condition) bool {
	if len(c1.Operands) != len(c2.Operands) {
		return false
	}

	// todo: sort first?
	for i := 0; i < len(c1.Operands); i++ {
		if c1.Operands[i] != c2.Operands[i] {
			return false
		}
	}

	return true
}
