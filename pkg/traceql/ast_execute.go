package traceql

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/grafana/tempo/pkg/regexp"
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
			group, ok := groups[result.MapKey()]
			if !ok {
				// If not, create a new group and add it to the map
				group = &Spanset{}
				// copy all existing attributes forward
				group.Attributes = append(group.Attributes, spanset.Attributes...)
				group.AddAttribute(g.String(), result)
				groups[result.MapKey()] = group
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

		var relFn func(s Span, l, r []Span) []Span

		switch o.Op {
		case OpSpansetAnd:
			if len(lhs) > 0 && len(rhs) > 0 {
				output = addSpanset(input[i], uniqueSpans(lhs, rhs), output)
			}

		case OpSpansetUnion:
			if len(lhs) > 0 || len(rhs) > 0 {
				output = addSpanset(input[i], uniqueSpans(lhs, rhs), output)
			}
		// relationship operators all set relFn which is used by below code
		// to perform the operation
		case OpSpansetNotDescendant: // !>>
			fallthrough
		case OpSpansetNotAncestor: // !<<
			fallthrough
		case OpSpansetAncestor: // <<
			fallthrough
		case OpSpansetDescendant: // >>
			fallthrough
		case OpSpansetUnionAncestor: // &<<
			fallthrough
		case OpSpansetUnionDescendant: // &>>
			falseForAll := o.Op == OpSpansetNotDescendant || o.Op == OpSpansetNotAncestor
			invert := o.Op == OpSpansetAncestor || o.Op == OpSpansetNotAncestor || o.Op == OpSpansetUnionAncestor
			union := o.Op == OpSpansetUnionAncestor || o.Op == OpSpansetUnionDescendant
			relFn = func(s Span, l, r []Span) []Span {
				return s.DescendantOf(l, r, falseForAll, invert, union, o.matchingSpansBuffer)
			}

		case OpSpansetNotChild: // !>
			fallthrough
		case OpSpansetChild: // >
			fallthrough
		case OpSpansetNotParent: // !<
			fallthrough
		case OpSpansetParent: // <
			fallthrough
		case OpSpansetUnionParent: // &<
			fallthrough
		case OpSpansetUnionChild: // &>
			falseForAll := o.Op == OpSpansetNotParent || o.Op == OpSpansetNotChild
			invert := o.Op == OpSpansetParent || o.Op == OpSpansetNotParent || o.Op == OpSpansetUnionParent
			union := o.Op == OpSpansetUnionParent || o.Op == OpSpansetUnionChild
			relFn = func(s Span, l, r []Span) []Span {
				return s.ChildOf(l, r, falseForAll, invert, union, o.matchingSpansBuffer)
			}

		case OpSpansetNotSibling: // !~
			fallthrough
		case OpSpansetSibling: // ~
			fallthrough
		case OpSpansetUnionSibling: // &~
			falseForAll := o.Op == OpSpansetNotSibling
			union := o.Op == OpSpansetUnionSibling
			relFn = func(s Span, l, r []Span) []Span {
				return s.SiblingOf(l, r, falseForAll, union, o.matchingSpansBuffer)
			}

		default:
			return nil, fmt.Errorf("spanset operation (%v) not supported", o.Op)
		}

		// if relFn was set up above we are doing a relationship operation.
		if relFn != nil {
			o.matchingSpansBuffer, err = o.joinSpansets(lhs, rhs, relFn) // o.matchingSpansBuffer is passed into the functions above and is stored here
			if err != nil {
				return nil, err
			}
			output = addSpanset(input[i], o.matchingSpansBuffer, output)
		}
	}

	return output, nil
}

// joinSpansets compares all pairwise combinations of the inputs and returns the right-hand side
// where the eval callback returns true.  For now the behavior is only defined when there is exactly one
// spanset on both sides and will return an error if multiple spansets are present.
func (o *SpansetOperation) joinSpansets(lhs, rhs []*Spanset, eval func(s Span, l, r []Span) []Span) ([]Span, error) {
	if len(rhs) < 1 {
		return nil, nil
	}

	if len(lhs) > 1 || len(rhs) > 1 {
		return nil, errSpansetOperationMultiple
	}

	// if rhs side is empty then no spans match and we can bail out here
	if len(rhs[0].Spans) == 0 {
		return nil, nil
	}

	var lspans []Span
	if len(lhs) >= 1 {
		lspans = lhs[0].Spans
	}

	return eval(rhs[0].Spans[0], lspans, rhs[0].Spans), nil
}

// addSpanset is a helper function that adds a new spanset to the output. it clones
// the input to prevent modifying the original spanset.
func addSpanset(input *Spanset, matching []Span, output []*Spanset) []*Spanset {
	if len(matching) == 0 {
		return output
	}

	matchingSpanset := input.clone()
	matchingSpanset.Spans = matching

	return append(output, matchingSpanset)
}

// SelectOperation evaluate is a no-op b/c the fetch layer has already decorated the spans with the requested attributes
func (o SelectOperation) evaluate(input []*Spanset) (output []*Spanset, err error) {
	return input, nil
}

func (f ScalarFilter) evaluate(input []*Spanset) (output []*Spanset, err error) {
	// TODO we solve this gap where pipeline elements and scalar binary
	// operations meet in a generic way. For now we only support well-defined
	// case: aggregate binop static
	switch l := f.LHS.(type) {
	case Aggregate:
		switch r := f.RHS.(type) {
		case Static:
			input, err = l.evaluate(input)
			if err != nil {
				return nil, err
			}

			for _, ss := range input {
				res, err := binOp(f.Op, ss.Scalar, r)
				if err != nil {
					return nil, fmt.Errorf("scalar filter (%v) failed: %v", f, err)
				}
				if res {
					output = append(output, ss)
				}
			}

		default:
			return nil, fmt.Errorf("scalar filter lhs (%v) not supported", f.LHS)
		}

	default:
		return nil, fmt.Errorf("scalar filter lhs (%v) not supported", f.LHS)
	}

	return output, nil
}

func (a Aggregate) evaluate(input []*Spanset) ([]*Spanset, error) {
	var output []*Spanset

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
				if val.IsNil() {
					continue
				}

				if sum == nil {
					sum = &val
				} else {
					sum.sumInto(&val)
				}
				count++
			}
			if sum == nil {
				continue
			}

			cpy := ss.clone()
			cpy.Scalar = sum.divideBy(float64(count))
			cpy.AddAttribute(a.String(), cpy.Scalar)
			output = append(output, cpy)

		case aggregateMax:
			var maxS *Static
			for _, s := range ss.Spans {
				val, err := a.e.execute(s)
				if err != nil {
					return nil, err
				}
				if val.IsNil() {
					continue
				}

				if maxS == nil || val.compare(maxS) > 0 {
					maxS = &val
				}
			}
			if maxS == nil {
				continue
			}

			cpy := ss.clone()
			cpy.Scalar = *maxS
			cpy.AddAttribute(a.String(), cpy.Scalar)
			output = append(output, cpy)

		case aggregateMin:
			var minS *Static
			for _, s := range ss.Spans {
				val, err := a.e.execute(s)
				if err != nil {
					return nil, err
				}
				if val.IsNil() {
					continue
				}

				if minS == nil || val.compare(minS) == -1 {
					minS = &val
				}
			}
			if minS == nil {
				continue
			}

			cpy := ss.clone()
			cpy.Scalar = *minS
			cpy.AddAttribute(a.String(), cpy.Scalar)
			output = append(output, cpy)

		case aggregateSum:
			var sum *Static
			for _, s := range ss.Spans {
				val, err := a.e.execute(s)
				if err != nil {
					return nil, err
				}
				if val.IsNil() {
					continue
				}

				if sum == nil {
					sum = &val
				} else {
					sum.sumInto(&val)
				}
			}
			if sum == nil {
				continue
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

func (o *BinaryOperation) execute(span Span) (Static, error) {
	recording := o.b.Recording
	if recording {
		o.b.Start()
	}

	lhs, err := o.LHS.execute(span)
	if err != nil {
		return NewStaticNil(), err
	}

	if recording {
		o.b.Finish(leftBranch)
	}

	// Look for cases where we don't even need to evaluate the RHS
	// But wait until we have enough samples so we can optimize
	if !recording {
		if lhsB, ok := lhs.Bool(); ok {
			if o.Op == OpAnd && !lhsB {
				// x && y
				// x is false so we don't need to evaluate y
				return StaticFalse, nil
			}
			if o.Op == OpOr && lhsB {
				// x || y
				// x is true so we don't need to evaluate y
				return StaticTrue, nil
			}
		}
	}

	if recording {
		o.b.Start()
	}

	rhs, err := o.RHS.execute(span)
	if err != nil {
		return NewStaticNil(), err
	}

	if recording {
		o.b.Finish(rightBranch)
	}

	lhsT := lhs.Type
	rhsT := rhs.Type

	// ensure the resolved types are still valid
	if !lhsT.isMatchingOperand(rhsT) {
		return StaticFalse, nil
	}
	if !o.Op.binaryTypesValid(lhsT, rhsT) {
		return StaticFalse, nil
	}

	switch {
	case lhsT == TypeString && rhsT == TypeString:
		// if both types are strings, execute string operations ...
		lhsS := lhs.EncodeToString(false)
		rhsS := rhs.EncodeToString(false)

		switch o.Op {
		case OpGreater:
			return NewStaticBool(strings.Compare(lhsS, rhsS) > 0), nil
		case OpGreaterEqual:
			return NewStaticBool(strings.Compare(lhsS, rhsS) >= 0), nil
		case OpLess:
			return NewStaticBool(strings.Compare(lhsS, rhsS) < 0), nil
		case OpLessEqual:
			return NewStaticBool(strings.Compare(lhsS, rhsS) <= 0), nil
		case OpRegex, OpNotRegex:
			shouldMatch := o.Op == OpRegex

			if len(o.compiledExpressions) == 0 {
				exp, err := regexp.NewRegexp([]string{rhsS}, shouldMatch)
				if err != nil {
					return NewStaticNil(), err
				}
				o.compiledExpressions = append(o.compiledExpressions, exp)
			}
			if len(o.compiledExpressions) != 1 {
				return NewStaticNil(), errors.New("unexpected numbers of pre-compiled regexp")
			}

			matched := o.compiledExpressions[0].MatchString(lhsS)
			return NewStaticBool(matched), nil
		}
		// if not executed, fall through to the catch-all below

	case lhsT == TypeInt && rhsT == TypeInt:
		// if both types are ints, execute int operations
		lhsN, _ := lhs.Int()
		rhsN, _ := rhs.Int()

		switch o.Op {
		case OpAdd:
			return NewStaticInt(lhsN + rhsN), nil
		case OpSub:
			return NewStaticInt(lhsN - rhsN), nil
		case OpDiv:
			return NewStaticInt(lhsN / rhsN), nil
		case OpMod:
			return NewStaticInt(lhsN % rhsN), nil
		case OpMult:
			return NewStaticInt(lhsN * rhsN), nil
		case OpGreater:
			return NewStaticBool(lhsN > rhsN), nil
		case OpGreaterEqual:
			return NewStaticBool(lhsN >= rhsN), nil
		case OpLess:
			return NewStaticBool(lhsN < rhsN), nil
		case OpLessEqual:
			return NewStaticBool(lhsN <= rhsN), nil
		case OpPower:
			return NewStaticInt(intPow(rhsN, lhsN)), nil
		}
		// if not executed, fall through to the catch-all below

	case lhsT == TypeBoolean && rhsT == TypeBoolean:
		// if both types are bools, execute boolean operations
		lhsB, _ := lhs.Bool()
		rhsB, _ := rhs.Bool()

		if recording {
			switch o.Op {
			case OpAnd:
				if !lhsB {
					// Record cost of wasted rhs execution
					o.b.Penalize(rightBranch)
				}
				if !rhsB {
					// Record cost of wasted lhs execution
					o.b.Penalize(leftBranch)
				}
			case OpOr:
				if rhsB {
					// Record cost of wasted lhs execution
					o.b.Penalize(rightBranch)
				}
				if lhsB {
					// Record cost of wasted rhs execution
					o.b.Penalize(leftBranch)
				}
			}

			if done := o.b.Sampled(); done {
				if o.b.OptimalBranch() == rightBranch {
					// RHS is the optimal starting branch,
					// so swap the elements now.
					o.LHS, o.RHS = o.RHS, o.LHS
				}
			}
		}

		switch o.Op {
		case OpAnd:
			return NewStaticBool(lhsB && rhsB), nil
		case OpOr:
			return NewStaticBool(lhsB || rhsB), nil
		}
		// if not executed, fall through to the catch-all below

	case lhsT.isMatchingArrayElement(rhsT):
		// if operands matching arrays / array elements, execute array operations

		// we only support boolean op in the arrays
		if !o.Op.isBoolean() {
			return NewStaticNil(), errors.ErrUnsupported
		}

		var (
			// the operator to apply to each element of the array
			elemOp Operator

			// the array and scalar side of the operation
			array  Static
			scalar Static

			flipOperands bool
		)

		if lhs.Type.isArray() && rhs.Type.isArray() {
			return NewStaticNil(), errors.New("array operators must consist of a scalar and an array operand")
		}

		// determine which side is the array and which is the scalar
		if o.Op.isArrayOp() {
			elemOp = o.Op.toElementOp()
			array = rhs
			scalar = lhs

			// array operations like IN, NOT IN, REGEX_MATCH_ANY, REGEX_MATCH_NONE are not symmetric
			if !array.Type.isArray() {
				return NewStaticNil(), errors.New("array operators require array on the right side")
			}
		} else {
			elemOp = o.Op
			array = rhs
			scalar = lhs

			if scalar.Type.isArray() {
				elemOp = getFlippedOp(o.Op)
				array, scalar = scalar, array

				// for regex operations we assume the RHS is a regex: must flip back operands later
				if elemOp == OpRegex || elemOp == OpNotRegex {
					flipOperands = true
				}
			}
		}

		// operators like OpNotEqual and OpNotRegex have the semantics of 'not in' / 'match none': must check all elements
		matchAll := elemOp == OpNotEqual || elemOp == OpNotRegex

		// apply operation to each element of the array
		var elemCount, matchCount int
		for i, elem := range array.Elements() {
			elemCount++

			l := scalar
			r := elem
			if flipOperands {
				l, r = r, l
			}

			var exp []*regexp.Regexp
			if len(o.compiledExpressions) > i {
				exp = o.compiledExpressions[i : i+1]
			}

			res, exp, err := binOpExecuteScalar(elemOp, l, r, exp)
			if err != nil {
				return NewStaticNil(), err
			}

			if len(o.compiledExpressions) == i {
				o.compiledExpressions = append(o.compiledExpressions, exp...)
			}

			match, ok := res.Bool()
			if ok && match {
				matchCount++
				if !matchAll {
					break
				}
			}
		}

		var result Static
		if matchAll {
			result = NewStaticBool(matchCount == elemCount)
		} else {
			result = NewStaticBool(matchCount > 0)
		}

		return result, nil
	}

	// if no operation was executed, execute float operations as a catch-all
	var result Static
	switch o.Op {
	case OpAdd:
		result = NewStaticFloat(lhs.Float() + rhs.Float())
	case OpSub:
		result = NewStaticFloat(lhs.Float() - rhs.Float())
	case OpDiv:
		result = NewStaticFloat(lhs.Float() / rhs.Float())
	case OpMod:
		result = NewStaticFloat(math.Mod(lhs.Float(), rhs.Float()))
	case OpMult:
		result = NewStaticFloat(lhs.Float() * rhs.Float())
	case OpGreater:
		result = NewStaticBool(lhs.Float() > rhs.Float())
	case OpGreaterEqual:
		result = NewStaticBool(lhs.Float() >= rhs.Float())
	case OpLess:
		result = NewStaticBool(lhs.Float() < rhs.Float())
	case OpLessEqual:
		result = NewStaticBool(lhs.Float() <= rhs.Float())
	case OpPower:
		result = NewStaticFloat(math.Pow(lhs.Float(), rhs.Float()))
	case OpEqual:
		result = NewStaticBool(lhs.Equals(&rhs))
	case OpNotEqual:
		result = NewStaticBool(lhs.NotEquals(&rhs))
	default:
		return NewStaticNil(), errors.New("unexpected operator " + o.Op.String())
	}

	return result, nil
}

// binOpExecuteScalar executes binary operations on scalar values only. This is used for comparing array elements
// and duplicates large parts of BinaryOperation.execute().
// NOTE: The code duplication with BinaryOperation.execute() IS intentional as the inlining has shown to be relevant
// for query performance.
func binOpExecuteScalar(op Operator, lhs, rhs Static, expressions []*regexp.Regexp) (Static, []*regexp.Regexp, error) {
	lhsT := lhs.Type
	rhsT := rhs.Type

	// Fast validation - we know these are scalar types from array iteration
	if !lhsT.isMatchingOperand(rhsT) || !op.binaryTypesValid(lhsT, rhsT) {
		return StaticFalse, expressions, nil
	}

	switch {
	case lhsT == TypeString && rhsT == TypeString:
		// String operations
		lhsS := lhs.EncodeToString(false)
		rhsS := rhs.EncodeToString(false)

		switch op {
		case OpGreater:
			return NewStaticBool(strings.Compare(lhsS, rhsS) > 0), expressions, nil
		case OpGreaterEqual:
			return NewStaticBool(strings.Compare(lhsS, rhsS) >= 0), expressions, nil
		case OpLess:
			return NewStaticBool(strings.Compare(lhsS, rhsS) < 0), expressions, nil
		case OpLessEqual:
			return NewStaticBool(strings.Compare(lhsS, rhsS) <= 0), expressions, nil
		case OpRegex, OpNotRegex:
			shouldMatch := op == OpRegex

			if len(expressions) == 0 {
				exp, err := regexp.NewRegexp([]string{rhsS}, shouldMatch)
				if err != nil {
					return NewStaticNil(), nil, err
				}
				expressions = append(expressions, exp)
			}
			if len(expressions) != 1 {
				return NewStaticNil(), expressions, errors.New("unexpected numbers of pre-compiled regexp")
			}

			matched := expressions[0].MatchString(lhsS)
			return NewStaticBool(matched), expressions, nil
		}

	case lhsT == TypeInt && rhsT == TypeInt:
		// Integer operations
		lhsN, _ := lhs.Int()
		rhsN, _ := rhs.Int()

		switch op {
		case OpAdd:
			return NewStaticInt(lhsN + rhsN), expressions, nil
		case OpSub:
			return NewStaticInt(lhsN - rhsN), expressions, nil
		case OpDiv:
			return NewStaticInt(lhsN / rhsN), expressions, nil
		case OpMod:
			return NewStaticInt(lhsN % rhsN), expressions, nil
		case OpMult:
			return NewStaticInt(lhsN * rhsN), expressions, nil
		case OpGreater:
			return NewStaticBool(lhsN > rhsN), expressions, nil
		case OpGreaterEqual:
			return NewStaticBool(lhsN >= rhsN), expressions, nil
		case OpLess:
			return NewStaticBool(lhsN < rhsN), expressions, nil
		case OpLessEqual:
			return NewStaticBool(lhsN <= rhsN), expressions, nil
		case OpPower:
			return NewStaticInt(intPow(rhsN, lhsN)), expressions, nil
		}

	case lhsT == TypeBoolean && rhsT == TypeBoolean:
		// Boolean operations
		lhsB, _ := lhs.Bool()
		rhsB, _ := rhs.Bool()

		switch op {
		case OpAnd:
			return NewStaticBool(lhsB && rhsB), expressions, nil
		case OpOr:
			return NewStaticBool(lhsB || rhsB), expressions, nil
		}
	}

	// Float catch-all
	switch op {
	case OpAdd:
		return NewStaticFloat(lhs.Float() + rhs.Float()), expressions, nil
	case OpSub:
		return NewStaticFloat(lhs.Float() - rhs.Float()), expressions, nil
	case OpDiv:
		return NewStaticFloat(lhs.Float() / rhs.Float()), expressions, nil
	case OpMod:
		return NewStaticFloat(math.Mod(lhs.Float(), rhs.Float())), expressions, nil
	case OpMult:
		return NewStaticFloat(lhs.Float() * rhs.Float()), expressions, nil
	case OpGreater:
		return NewStaticBool(lhs.Float() > rhs.Float()), expressions, nil
	case OpGreaterEqual:
		return NewStaticBool(lhs.Float() >= rhs.Float()), expressions, nil
	case OpLess:
		return NewStaticBool(lhs.Float() < rhs.Float()), expressions, nil
	case OpLessEqual:
		return NewStaticBool(lhs.Float() <= rhs.Float()), expressions, nil
	case OpPower:
		return NewStaticFloat(math.Pow(lhs.Float(), rhs.Float())), expressions, nil
	case OpEqual:
		return NewStaticBool(lhs.Equals(&rhs)), expressions, nil
	case OpNotEqual:
		return NewStaticBool(lhs.NotEquals(&rhs)), expressions, nil
	default:
		return NewStaticNil(), nil, errors.New("unexpected operator " + op.String())
	}
}

// getFlippedOp will return the flipped op, used when flipping the LHS and RHS of a BinaryOperation
func getFlippedOp(op Operator) Operator {
	switch op {
	case OpGreater:
		return OpLess
	case OpGreaterEqual:
		return OpLessEqual
	case OpLess:
		return OpGreater
	case OpLessEqual:
		return OpGreaterEqual
	default:
		return op
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

	if lhsT == TypeBoolean && rhsT == TypeBoolean {
		lhsB, _ := lhs.Bool()
		rhsB, _ := rhs.Bool()

		switch op {
		case OpAnd:
			return lhsB && rhsB, nil
		case OpOr:
			return lhsB || rhsB, nil
		}
	}

	switch op {
	case OpGreater:
		return lhs.Float() > rhs.Float(), nil
	case OpGreaterEqual:
		return lhs.Float() >= rhs.Float(), nil
	case OpLess:
		return lhs.Float() < rhs.Float(), nil
	case OpLessEqual:
		return lhs.Float() <= rhs.Float(), nil
	case OpEqual:
		return lhs.Equals(&rhs), nil
	case OpNotEqual:
		return lhs.NotEquals(&rhs), nil
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
		b, _ := static.Bool()
		return NewStaticBool(!b), nil
	}
	if o.Op == OpSub {
		if !static.Type.isNumeric() {
			return NewStaticNil(), fmt.Errorf("expression (%v) expected a numeric, but got %v", o, static.Type)
		}
		switch static.Type {
		case TypeInt:
			n, _ := static.Int()
			return NewStaticInt(-1 * n), nil
		case TypeFloat:
			return NewStaticFloat(-1 * static.Float()), nil
		case TypeDuration:
			d, _ := static.Duration()
			return NewStaticDuration(-1 * d), nil
		}
	}
	if o.Op == OpExists {
		return NewStaticBool(static.Type != TypeNil), nil
	}
	if o.Op == OpNotExists {
		staticNilString := NewStaticString("nil")
		return NewStaticBool(static.Equals(&staticNilString)), nil
	}

	return NewStaticNil(), fmt.Errorf("UnaryOperation has invalid operator %v", o.Op)
}

func (s Static) execute(Span) (Static, error) {
	return s, nil
}

func (a Attribute) execute(span Span) (Static, error) {
	static, ok := span.AttributeFor(a)
	if ok {
		return static, nil
	}

	return StaticNil, nil
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

func intPow(m, n int) int {
	result := 1
	for i := 0; i < n; i++ {
		result *= m
	}
	return result
}
