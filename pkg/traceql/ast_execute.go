package traceql

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
)

var errSpansetOperationMultiple = errors.New("spanset operators are not supported for multiple spansets per trace. consider using coalesce()")

func (g GroupOperation) evaluate(ss []*Spanset) ([]*Spanset, error) {
	result := make([]*Spanset, 0, len(ss))
	groups := g.groupBuffer

	// Iterate over each spanset in the input slice
	for _, spanset := range ss {
		// clear out the groups
		for k := range groups {
			delete(groups, k)
		}

		// Iterate over each span in the spanset
		for _, span := range spanset.Spans {
			// Execute the FieldExpression for the span
			result, err := g.Expression.execute(span)
			if err != nil {
				return nil, err
			}

			// Check if the result already has a group in the map
			group, ok := groups[result]
			if !ok {
				// If not, create a new group and add it to the map
				group = &Spanset{}
				// copy all existing attributes forward
				group.Attributes = append(group.Attributes, spanset.Attributes...)
				group.AddAttribute(g.String(), result)
				groups[result] = group
			}

			// Add the current spanset to the group
			group.Spans = append(group.Spans, span)
		}

		// add all groups created by this spanset to the result
		for _, group := range groups {
			result = append(result, group)
		}
	}

	return result, nil
}

// CoalesceOperation undoes grouping. It takes spansets and recombines them into
// one by trace id. Since all spansets are guaranteed to be from the same traceid
// due to the structure of the engine we can cheat and just recombine all spansets
// in ss into one without checking.
func (CoalesceOperation) evaluate(ss []*Spanset) ([]*Spanset, error) {
	l := 0
	for _, spanset := range ss {
		l += len(spanset.Spans)
	}
	result := &Spanset{
		Spans: make([]Span, 0, l),
	}
	for _, spanset := range ss {
		result.Spans = append(result.Spans, spanset.Spans...)
	}
	return []*Spanset{result}, nil
}

func (o SpansetOperation) evaluate(input []*Spanset) (output []*Spanset, err error) {
	for i := range input {
		curr := input[i : i+1]

		lhs, err := o.LHS.evaluate(curr)
		if err != nil {
			return nil, err
		}

		rhs, err := o.RHS.evaluate(curr)
		if err != nil {
			return nil, err
		}

		var relFn func(l, r Span) bool
		var falseForAll bool

		switch o.Op {
		case OpSpansetAnd:
			if len(lhs) > 0 && len(rhs) > 0 {
				matchingSpanset := input[i].clone()
				matchingSpanset.Spans = uniqueSpans(lhs, rhs)
				output = append(output, matchingSpanset)
			}

		case OpSpansetUnion:
			if len(lhs) > 0 || len(rhs) > 0 {
				matchingSpanset := input[i].clone()
				matchingSpanset.Spans = uniqueSpans(lhs, rhs)
				output = append(output, matchingSpanset)
			}

		// relationship operators all set relFn which is used by below code
		// to perform the operation
		case OpSpansetDescendant:
			fallthrough
		case OpSpansetNotDescendant:
			relFn = func(l, r Span) bool {
				return r.DescendantOf(l)
			}
			falseForAll = o.Op == OpSpansetNotDescendant

		case OpSpansetAncestor:
			fallthrough
		case OpSpansetNotAncestor:
			relFn = func(l, r Span) bool {
				return l.DescendantOf(r)
			}
			falseForAll = o.Op == OpSpansetNotAncestor

		case OpSpansetChild:
			fallthrough
		case OpSpansetNotChild:
			relFn = func(l, r Span) bool {
				return r.ChildOf(l)
			}
			falseForAll = o.Op == OpSpansetNotChild

		case OpSpansetParent:
			fallthrough
		case OpSpansetNotParent:
			relFn = func(l, r Span) bool {
				return l.ChildOf(r)
			}
			falseForAll = o.Op == OpSpansetNotParent

		case OpSpansetSibling:
			fallthrough
		case OpSpansetNotSibling:
			relFn = func(l, r Span) bool {
				return r.SiblingOf(l)
			}
			falseForAll = o.Op == OpSpansetNotSibling
		default:
			return nil, fmt.Errorf("spanset operation (%v) not supported", o.Op)
		}

		// if relFn was set up above we are doing a relationship operation.
		if relFn != nil {
			spans, err := o.joinSpansets(lhs, rhs, falseForAll, relFn)
			if err != nil {
				return nil, err
			}

			if len(spans) > 0 {
				// Clone here to capture previously computed aggregates, grouped attrs, etc.
				// Copy spans to new slice because of internal buffering.
				matchingSpanset := input[i].clone()
				matchingSpanset.Spans = append([]Span(nil), spans...)
				output = append(output, matchingSpanset)
			}
		}
	}

	return output, nil
}

// joinSpansets compares all pairwise combinations of the inputs and returns the right-hand side
// where the eval callback returns true.  For now the behavior is only defined when there is exactly one
// spanset on both sides and will return an error if multiple spansets are present.
func (o *SpansetOperation) joinSpansets(lhs, rhs []*Spanset, falseForAll bool, eval func(l, r Span) bool) ([]Span, error) {
	if len(lhs) < 1 || len(rhs) < 1 {
		return nil, nil
	}

	if len(lhs) > 1 || len(rhs) > 1 {
		return nil, errSpansetOperationMultiple
	}

	return o.joinSpansAndReturnRHS(lhs[0].Spans, rhs[0].Spans, falseForAll, eval), nil
}

// joinSpansAndReturnRHS compares all pairwise combinations of the inputs and returns the right-hand side
// spans where the eval callback returns true.  Uses and internal buffer and output is only valid until
// the next call.  Destructively edits the RHS slice for performance.
// falseForAll indicates that the spans on the RHS should only be returned if relFn returns
// false for all on the LHS. otherwise spans on the RHS are returned if there are any matches on the lhs
func (o *SpansetOperation) joinSpansAndReturnRHS(lhs, rhs []Span, falseForAll bool, eval func(l, r Span) bool) []Span {
	if len(lhs) == 0 || len(rhs) == 0 {
		return nil
	}

	o.matchingSpansBuffer = o.matchingSpansBuffer[:0]

	for _, r := range rhs {
		matches := false
		for _, l := range lhs {
			if eval(l, r) {
				// Returns RHS
				matches = true
				break
			}
		}
		if matches && !falseForAll || // return RHS if there are any matches on the LHS
			!matches && falseForAll { // return RHS if there are no matches on the LHS
			o.matchingSpansBuffer = append(o.matchingSpansBuffer, r)
		}
	}

	return o.matchingSpansBuffer
}

// SelectOperation evaluate is a no-op b/c the fetch layer has already decorated the spans with the requested attributes
func (o SelectOperation) evaluate(input []*Spanset) (output []*Spanset, err error) {
	return input, nil
}

func (f ScalarFilter) evaluate(input []*Spanset) (output []*Spanset, err error) {
	// TODO we solve this gap where pipeline elements and scalar binary
	// operations meet in a generic way. For now we only support well-defined
	// case: aggregate binop static
	switch l := f.lhs.(type) {
	case Aggregate:
		switch r := f.rhs.(type) {
		case Static:
			input, err = l.evaluate(input)
			if err != nil {
				return nil, err
			}

			for _, ss := range input {
				res, err := binOp(f.op, ss.Scalar, r)
				if err != nil {
					return nil, fmt.Errorf("scalar filter (%v) failed: %v", f, err)
				}
				if res {
					output = append(output, ss)
				}
			}

		default:
			return nil, fmt.Errorf("scalar filter lhs (%v) not supported", f.lhs)
		}

	default:
		return nil, fmt.Errorf("scalar filter lhs (%v) not supported", f.lhs)
	}

	return output, nil
}

func (a Aggregate) evaluate(input []*Spanset) (output []*Spanset, err error) {
	for _, ss := range input {
		switch a.op {
		case aggregateCount:
			cpy := ss.clone()
			cpy.Scalar = NewStaticInt(len(ss.Spans))
			cpy.AddAttribute(a.String(), cpy.Scalar)
			output = append(output, cpy)

		case aggregateAvg:
			var sum *Static
			count := 0
			for _, s := range ss.Spans {
				val, err := a.e.execute(s)
				if err != nil {
					return nil, err
				}

				if sum == nil {
					sum = &val
				} else {
					sum.sumInto(val)
				}
				count++
			}

			cpy := ss.clone()
			cpy.Scalar = sum.divideBy(float64(count))
			cpy.AddAttribute(a.String(), cpy.Scalar)
			output = append(output, cpy)

		case aggregateMax:
			var max *Static
			for _, s := range ss.Spans {
				val, err := a.e.execute(s)
				if err != nil {
					return nil, err
				}
				if max == nil || val.compare(max) == 1 {
					max = &val
				}
			}
			cpy := ss.clone()
			cpy.Scalar = *max
			cpy.AddAttribute(a.String(), cpy.Scalar)
			output = append(output, cpy)

		case aggregateMin:
			var min *Static
			for _, s := range ss.Spans {
				val, err := a.e.execute(s)
				if err != nil {
					return nil, err
				}
				if min == nil || val.compare(min) == -1 {
					min = &val
				}
			}
			cpy := ss.clone()
			cpy.Scalar = *min
			cpy.AddAttribute(a.String(), cpy.Scalar)
			output = append(output, cpy)

		case aggregateSum:
			var sum *Static
			for _, s := range ss.Spans {
				val, err := a.e.execute(s)
				if err != nil {
					return nil, err
				}
				if sum == nil {
					sum = &val
				} else {
					sum.sumInto(val)
				}
			}
			cpy := ss.clone()
			cpy.Scalar = *sum
			cpy.AddAttribute(a.String(), cpy.Scalar)
			output = append(output, cpy)

		default:
			return nil, fmt.Errorf("aggregate operation (%v) not supported", a.op)
		}
	}

	return output, nil
}

func (o BinaryOperation) execute(span Span) (Static, error) {
	lhs, err := o.LHS.execute(span)
	if err != nil {
		return NewStaticNil(), err
	}

	rhs, err := o.RHS.execute(span)
	if err != nil {
		return NewStaticNil(), err
	}

	// Ensure the resolved types are still valid
	lhsT := lhs.impliedType()
	rhsT := rhs.impliedType()
	if !lhsT.isMatchingOperand(rhsT) {
		return NewStaticBool(false), nil
	}

	if !o.Op.binaryTypesValid(lhsT, rhsT) {
		return NewStaticBool(false), nil
	}

	if lhsT == TypeString && rhsT == TypeString {
		switch o.Op {
		case OpGreater:
			return NewStaticBool(strings.Compare(lhs.String(), rhs.String()) > 0), nil
		case OpGreaterEqual:
			return NewStaticBool(strings.Compare(lhs.String(), rhs.String()) >= 0), nil
		case OpLess:
			return NewStaticBool(strings.Compare(lhs.String(), rhs.String()) < 0), nil
		case OpLessEqual:
			return NewStaticBool(strings.Compare(lhs.String(), rhs.String()) <= 0), nil
		default:
		}
	}

	switch o.Op {
	case OpAdd:
		return NewStaticFloat(lhs.asFloat() + rhs.asFloat()), nil
	case OpSub:
		return NewStaticFloat(lhs.asFloat() - rhs.asFloat()), nil
	case OpDiv:
		return NewStaticFloat(lhs.asFloat() / rhs.asFloat()), nil
	case OpMod:
		return NewStaticFloat(math.Mod(lhs.asFloat(), rhs.asFloat())), nil
	case OpMult:
		return NewStaticFloat(lhs.asFloat() * rhs.asFloat()), nil
	case OpGreater:
		return NewStaticBool(lhs.asFloat() > rhs.asFloat()), nil
	case OpGreaterEqual:
		return NewStaticBool(lhs.asFloat() >= rhs.asFloat()), nil
	case OpLess:
		return NewStaticBool(lhs.asFloat() < rhs.asFloat()), nil
	case OpLessEqual:
		return NewStaticBool(lhs.asFloat() <= rhs.asFloat()), nil
	case OpPower:
		return NewStaticFloat(math.Pow(lhs.asFloat(), rhs.asFloat())), nil
	case OpEqual:
		return NewStaticBool(lhs.Equals(rhs)), nil
	case OpNotEqual:
		return NewStaticBool(!lhs.Equals(rhs)), nil
	case OpRegex:
		matched, err := regexp.MatchString(rhs.S, lhs.S)
		return NewStaticBool(matched), err
	case OpNotRegex:
		matched, err := regexp.MatchString(rhs.S, lhs.S)
		return NewStaticBool(!matched), err
	case OpAnd:
		return NewStaticBool(lhs.B && rhs.B), nil
	case OpOr:
		return NewStaticBool(lhs.B || rhs.B), nil
	default:
		return NewStaticNil(), errors.New("unexpected operator " + o.Op.String())
	}
}

// why does this and the above exist?
func binOp(op Operator, lhs, rhs Static) (bool, error) {
	lhsT := lhs.impliedType()
	rhsT := rhs.impliedType()
	if !lhsT.isMatchingOperand(rhsT) {
		return false, nil
	}

	if !op.binaryTypesValid(lhsT, rhsT) {
		return false, nil
	}

	switch op {
	case OpGreater:
		return lhs.asFloat() > rhs.asFloat(), nil
	case OpGreaterEqual:
		return lhs.asFloat() >= rhs.asFloat(), nil
	case OpLess:
		return lhs.asFloat() < rhs.asFloat(), nil
	case OpLessEqual:
		return lhs.asFloat() <= rhs.asFloat(), nil
	case OpEqual:
		return lhs.Equals(rhs), nil
	case OpNotEqual:
		return !lhs.Equals(rhs), nil
	case OpAnd:
		return lhs.B && rhs.B, nil
	case OpOr:
		return lhs.B || rhs.B, nil
	}

	return false, errors.New("unexpected operator " + op.String())
}

func (o UnaryOperation) execute(span Span) (Static, error) {
	static, err := o.Expression.execute(span)
	if err != nil {
		return NewStaticNil(), err
	}

	if o.Op == OpNot {
		if static.Type != TypeBoolean {
			return NewStaticNil(), fmt.Errorf("expression (%v) expected a boolean, but got %v", o, static.Type)
		}
		return NewStaticBool(!static.B), nil
	}
	if o.Op == OpSub {
		if !static.Type.isNumeric() {
			return NewStaticNil(), fmt.Errorf("expression (%v) expected a numeric, but got %v", o, static.Type)
		}
		switch static.Type {
		case TypeInt:
			return NewStaticInt(-1 * static.N), nil
		case TypeFloat:
			return NewStaticFloat(-1 * static.F), nil
		case TypeDuration:
			return NewStaticDuration(-1 * static.D), nil
		}
	}

	return NewStaticNil(), errors.New("UnaryOperation has Op different from Not and Sub")
}

func (s Static) execute(Span) (Static, error) {
	return s, nil
}

func (a Attribute) execute(span Span) (Static, error) {
	atts := span.Attributes()
	static, ok := atts[a]
	if ok {
		return static, nil
	}

	if a.Scope == AttributeScopeNone {
		for attribute, static := range atts {
			if a.Name == attribute.Name && attribute.Scope == AttributeScopeSpan {
				return static, nil
			}
		}
		for attribute, static := range atts {
			if a.Name == attribute.Name {
				return static, nil
			}
		}
	}

	return NewStaticNil(), nil
}

func uniqueSpans(ss1 []*Spanset, ss2 []*Spanset) []Span {
	ss1Count := 0
	ss2Count := 0

	for _, ss1 := range ss1 {
		ss1Count += len(ss1.Spans)
	}
	for _, ss2 := range ss2 {
		ss2Count += len(ss2.Spans)
	}
	output := make([]Span, 0, ss1Count+ss2Count)

	ssCount := ss2Count
	ssSmaller := ss2
	ssLarger := ss1
	if ss1Count < ss2Count {
		ssCount = ss1Count
		ssSmaller = ss1
		ssLarger = ss2
	}

	// make the map with ssSmaller
	spans := make(map[Span]struct{}, ssCount)
	for _, ss := range ssSmaller {
		for _, span := range ss.Spans {
			spans[span] = struct{}{}
			output = append(output, span)
		}
	}

	// only add the spans from ssLarger that aren't in the map
	for _, ss := range ssLarger {
		for _, span := range ss.Spans {
			if _, ok := spans[span]; !ok {
				output = append(output, span)
			}
		}
	}

	return output
}
