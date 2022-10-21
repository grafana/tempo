package traceql

import (
	"fmt"
	"regexp"

	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/pkg/util/log"
)

func appendSpans(buffer []Span, input []Spanset) []Span {
	for _, i := range input {
		buffer = append(buffer, i.Spans...)
	}
	return buffer
}

func (o SpansetOperation) evaluate(input []Spanset) (output []Spanset, err error) {

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

		switch o.Op {
		case OpSpansetAnd:
			if len(lhs) > 0 && len(rhs) > 0 {
				matchingSpanset := input[i]
				matchingSpanset.Spans = appendSpans(nil, lhs)
				matchingSpanset.Spans = appendSpans(matchingSpanset.Spans, rhs)
				output = append(output, matchingSpanset)
			}

		case OpSpansetUnion:
			if len(lhs) > 0 || len(rhs) > 0 {
				matchingSpanset := input[i]
				matchingSpanset.Spans = appendSpans(nil, lhs)
				matchingSpanset.Spans = appendSpans(matchingSpanset.Spans, rhs)
				output = append(output, matchingSpanset)
			}

		default:
			return nil, fmt.Errorf("spanset operation (%v) not supported", o.Op)
		}
	}

	return output, nil
}

func (f SpansetFilter) matches(span Span) (bool, error) {
	static, err := f.Expression.execute(span)
	if err != nil {
		level.Debug(log.Logger).Log("msg", "SpanSetFilter.matches failed", "err", err)
		return false, err
	}
	if static.Type != TypeBoolean {
		level.Debug(log.Logger).Log("msg", "SpanSetFilter.matches did not return a boolean", "err", err)
		return false, fmt.Errorf("result of SpanSetFilter (%v) is %v", f, static.Type)
	}
	return static.B, nil
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
		return NewStaticNil(), fmt.Errorf("binary operations must operate on the same type: %s", o.String())
	}

	if !o.Op.binaryTypesValid(lhsT, rhsT) {
		return NewStaticNil(), fmt.Errorf("illegal operation for the given types: %s", o.String())
	}

	switch o.Op {
	// TODO implement arithmetics
	case OpAdd:
	case OpSub:
	case OpDiv:
	case OpMod:
	case OpMult:
	case OpGreater:
		return NewStaticBool(lhs.asFloat() > rhs.asFloat()), nil
	case OpGreaterEqual:
		return NewStaticBool(lhs.asFloat() >= rhs.asFloat()), nil
	case OpLess:
		return NewStaticBool(lhs.asFloat() < rhs.asFloat()), nil
	case OpLessEqual:
		return NewStaticBool(lhs.asFloat() <= rhs.asFloat()), nil
	case OpPower:
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
		panic("unexpected operator " + o.Op.String())
	}

	panic("operator " + o.Op.String() + " is not yet implemented")
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

	panic("UnaryOperation has Op different from Not and Sub")
}

func (s Static) execute(span Span) (Static, error) {
	return s, nil
}

func (a Attribute) execute(span Span) (Static, error) {
	static, ok := span.Attributes[a]
	if ok {
		return static, nil
	}

	if a.Scope == AttributeScopeNone {
		for attribute, static := range span.Attributes {
			if a.Name == attribute.Name && attribute.Scope == AttributeScopeSpan {
				return static, nil
			}
		}
		for attribute, static := range span.Attributes {
			if a.Name == attribute.Name {
				return static, nil
			}
		}
	}

	return NewStaticNil(), nil
}
