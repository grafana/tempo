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

func booleanOperator(op int) bool {
	return op == opOr ||
		op == opAnd ||
		op == opEqual ||
		op == opNotEqual ||
		op == opRegex ||
		op == opNotRegex ||
		op == opGreater ||
		op == opGreaterEqual ||
		op == opLess ||
		op == opLessEqual
}

type element interface {
	fmt.Stringer
	validate() error
}

type typedExpression interface {
	impliedType() int
}

type RootExpr struct {
	p Pipeline
}

func newRootExpr(e element) *RootExpr {
	p, ok := e.(Pipeline)
	if !ok {
		p = newPipeline(e)
	}

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

func (p Pipeline) impliedType() int {
	if len(p.p) == 0 {
		return typeSpanset
	}

	finalItem := p.p[len(p.p)-1]
	aggregate, ok := finalItem.(Aggregate)
	if ok {
		return aggregate.impliedType()
	}

	return typeSpanset
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
	typedExpression
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
func (o ScalarOperation) impliedType() int {
	if booleanOperator(o.op) {
		return typeBoolean
	}

	// remaining operators will be based on the operands
	// opAdd, opSub, opDiv, opMod, opMult
	t := o.lhs.impliedType()
	if t != typeAttribute {
		return t
	}

	return o.rhs.impliedType()
}

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
func (a Aggregate) impliedType() int {
	if a.agg == aggregateCount || a.e == nil {
		return typeInt
	}

	return a.e.impliedType()
}

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
	typedExpression
	__fieldExpression()
}

type BinaryOperation struct {
	op  int
	lhs FieldExpression
	rhs FieldExpression
}

func (BinaryOperation) __fieldExpression() {}

func (o BinaryOperation) impliedType() int {
	if booleanOperator(o.op) {
		return typeBoolean
	}

	// remaining operators will be based on the operands
	// opAdd, opSub, opDiv, opMod, opMult
	t := o.lhs.impliedType()
	if t != typeAttribute {
		return t
	}

	return o.rhs.impliedType()
}

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

func (o UnaryOperation) impliedType() int {
	// both operators (opPower and opNot) will just be based on the operand type
	return o.e.impliedType()
}

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
	typeSpanset   = iota // type used by spanset pipelines
	typeAttribute        // a special constant that indicates the type is determined at query time by the attribute
	typeInt
	typeFloat
	typeString
	typeBoolean
	typeNil
	typeDuration
	typeStatus
	typeIntrinsic
)

const (
	statusError = iota
	statusOk
	statusUnset
)

const (
	intrinsicDuration = iota
	intrinsicChildCount
	intrinsicName
	intrinsicStatus
	intrinsicParent
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

func (s Static) impliedType() int {
	if s.staticType == typeIntrinsic {
		switch s.n {
		case intrinsicDuration:
			return typeDuration
		case intrinsicChildCount:
			return typeInt
		case intrinsicName:
			return typeString
		case intrinsicStatus:
			return typeStatus
		case intrinsicParent:
			return typeNil
		}
	}

	return s.staticType
}

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

func newStaticStatus(s int) Static {
	return Static{
		staticType: typeStatus,
		n:          s,
	}
}

func newIntrinsic(n int) Static {
	return Static{
		staticType: typeIntrinsic,
		n:          n,
	}
}

// **********************
// Attributes
// **********************

const (
	attributeScopeNone = iota
	attributeScopeParent
	attributeScopeParentResource
	attributeScopeParentSpan
	attributeScopeResource
	attributeScopeSpan
)

type Attribute struct {
	scope int
	att   string
}

func (Attribute) __fieldExpression() {}

func (a Attribute) impliedType() int {
	return typeAttribute
}

func newAttribute(att string) Attribute {
	return Attribute{
		scope: attributeScopeNone,
		att:   att,
	}
}

func newScopedAttribute(scope int, att string) Attribute {
	return Attribute{
		scope: scope,
		att:   att,
	}
}

func appendAttribute(existing Attribute, att string) Attribute {
	return Attribute{
		scope: existing.scope,
		att:   existing.att + "." + att,
	}
}
