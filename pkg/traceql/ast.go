package traceql

import (
	"fmt"
	"time"
)

const (
	opAdd = iota
	opSub
	opDiv
	opMod
	opMult
	opEqual
	opNotEqual
	opRegex
	opNotRegex
	opGreater
	opGreaterEqual
	opLess
	opLessEqual
	opPower
	opAnd
	opOr
	opNot
	opSpansetChild
	opSpansetDescendant
	opSpansetAnd
	opSpansetUnion
	opSpansetSibling
)

type element interface {
	fmt.Stringer
}

type RootExpr struct {
	p Pipeline
}

func newRootExpr(p Pipeline) *RootExpr {
	return &RootExpr{
		p: p,
	}
}

// **********************
// Pipeline
// **********************

type Pipeline struct {
	p []element
}

func (Pipeline) __scalarExpression()  {}
func (Pipeline) __spansetExpression() {}

func newPipeline(i ...element) Pipeline {
	return Pipeline{
		p: i,
	}
}

func (p Pipeline) addItem(i element) Pipeline {
	p.p = append(p.p, i)
	return p
}

type GroupOperation struct {
	e FieldExpression
}

func newGroupOperation(e FieldExpression) GroupOperation {
	return GroupOperation{
		e: e,
	}
}

type CoalesceOperation struct {
}

func newCoalesceOperation() CoalesceOperation {
	return CoalesceOperation{}
}

// **********************
// Scalars
// **********************
type ScalarExpression interface {
	element
	__scalarExpression()
}

type ScalarOperation struct {
	op  int
	lhs ScalarExpression
	rhs ScalarExpression
}

func newScalarOperation(op int, lhs ScalarExpression, rhs ScalarExpression) ScalarOperation {
	return ScalarOperation{
		op:  op,
		lhs: lhs,
		rhs: rhs,
	}
}

func (ScalarOperation) __scalarExpression() {}

const (
	aggregateCount = iota
	aggregateMax
	aggregateMin
	aggregateSum
	aggregateAvg
)

type Aggregate struct {
	agg int
	e   FieldExpression
}

func newAggregate(agg int, e FieldExpression) Aggregate {
	return Aggregate{
		agg: agg,
		e:   e,
	}
}

func (Aggregate) __scalarExpression() {}

// **********************
// Spansets
// **********************
type SpansetExpression interface {
	element
	__spansetExpression()
}

type SpansetOperation struct {
	op  int
	lhs SpansetExpression
	rhs SpansetExpression
}

func newSpansetOperation(op int, lhs SpansetExpression, rhs SpansetExpression) SpansetOperation {
	return SpansetOperation{
		op:  op,
		lhs: lhs,
		rhs: rhs,
	}
}

func (SpansetOperation) __spansetExpression() {}

type SpansetFilter struct {
	e FieldExpression
}

func newSpansetFilter(e FieldExpression) SpansetFilter {
	return SpansetFilter{
		e: e,
	}
}

func (SpansetFilter) __spansetExpression() {}

type ScalarFilter struct {
	op  int
	lhs ScalarExpression
	rhs ScalarExpression
}

func newScalarFilter(op int, lhs ScalarExpression, rhs ScalarExpression) ScalarFilter {
	return ScalarFilter{
		op:  op,
		lhs: lhs,
		rhs: rhs,
	}
}

func (ScalarFilter) __spansetExpression() {}

// **********************
// Expressions
// **********************
type FieldExpression interface {
	element
	__fieldExpression()
}

type BinaryOperation struct {
	op  int
	lhs FieldExpression
	rhs FieldExpression
}

func (BinaryOperation) __fieldExpression() {}

func newBinaryOperation(op int, lhs FieldExpression, rhs FieldExpression) BinaryOperation {
	return BinaryOperation{
		op:  op,
		lhs: lhs,
		rhs: rhs,
	}
}

type UnaryOperation struct {
	op int
	e  FieldExpression
}

func (UnaryOperation) __fieldExpression() {}

func newUnaryOperation(op int, e FieldExpression) UnaryOperation {
	return UnaryOperation{
		op: op,
		e:  e,
	}
}

// **********************
// Statics
// **********************
const (
	typeInt = iota
	typeFloat
	typeString
	typeBoolean
	typeIdentifier
	typeNil
	typeDuration
)

type Static struct {
	staticType int
	n          int
	f          float64
	s          string
	b          bool
	d          time.Duration
}

func (Static) __fieldExpression()  {}
func (Static) __scalarExpression() {}

func newStaticInt(n int) Static {
	return Static{
		staticType: typeInt,
		n:          n,
	}
}

func newStaticFloat(f float64) Static {
	return Static{
		staticType: typeFloat,
		f:          f,
	}
}

func newStaticString(s string) Static {
	return Static{
		staticType: typeString,
		s:          s,
	}
}

func newStaticBool(b bool) Static {
	return Static{
		staticType: typeBoolean,
		b:          b,
	}
}

func newStaticNil() Static {
	return Static{
		staticType: typeNil,
	}
}

func newStaticDuration(d time.Duration) Static {
	return Static{
		staticType: typeDuration,
		d:          d,
	}
}

func newStaticIdentifier(s string) Static {
	return Static{
		staticType: typeIdentifier,
		s:          s,
	}
}

func newNamespacedIdentifier(ns Static, s string) Static {
	return Static{
		staticType: typeIdentifier,
		s:          ns.s + "." + s,
	}
}
