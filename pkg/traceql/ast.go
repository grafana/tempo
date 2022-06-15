package traceql

import (
	"fmt"
	"time"
)

type element interface {
	fmt.Stringer
	validate() error
}

type typedExpression interface {
	impliedType() StaticType
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

// nolint: revive
func (Pipeline) __scalarExpression() {}

// nolint: revive
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

func (p Pipeline) impliedType() StaticType {
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
	op  Operator
	lhs ScalarExpression
	rhs ScalarExpression
}

func newScalarOperation(op Operator, lhs ScalarExpression, rhs ScalarExpression) ScalarOperation {
	return ScalarOperation{
		op:  op,
		lhs: lhs,
		rhs: rhs,
	}
}

// nolint: revive
func (ScalarOperation) __scalarExpression() {}

func (o ScalarOperation) impliedType() StaticType {
	if o.op.isBoolean() {
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

type Aggregate struct {
	agg AggregateOp
	e   FieldExpression
}

func newAggregate(agg AggregateOp, e FieldExpression) Aggregate {
	return Aggregate{
		agg: agg,
		e:   e,
	}
}

// nolint: revive
func (Aggregate) __scalarExpression() {}

func (a Aggregate) impliedType() StaticType {
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
	op  Operator
	lhs SpansetExpression
	rhs SpansetExpression
}

func newSpansetOperation(op Operator, lhs SpansetExpression, rhs SpansetExpression) SpansetOperation {
	return SpansetOperation{
		op:  op,
		lhs: lhs,
		rhs: rhs,
	}
}

// nolint: revive
func (SpansetOperation) __spansetExpression() {}

type SpansetFilter struct {
	e FieldExpression
}

func newSpansetFilter(e FieldExpression) SpansetFilter {
	return SpansetFilter{
		e: e,
	}
}

// nolint: revive
func (SpansetFilter) __spansetExpression() {}

type ScalarFilter struct {
	op  Operator
	lhs ScalarExpression
	rhs ScalarExpression
}

func newScalarFilter(op Operator, lhs ScalarExpression, rhs ScalarExpression) ScalarFilter {
	return ScalarFilter{
		op:  op,
		lhs: lhs,
		rhs: rhs,
	}
}

// nolint: revive
func (ScalarFilter) __spansetExpression() {}

// **********************
// Expressions
// **********************
type FieldExpression interface {
	element
	typedExpression

	// referencesSpan returns true if this field expression has any attributes or intrinsics. i.e. it references the span itself
	referencesSpan() bool
	__fieldExpression()
}

type BinaryOperation struct {
	op  Operator
	lhs FieldExpression
	rhs FieldExpression
}

func newBinaryOperation(op Operator, lhs FieldExpression, rhs FieldExpression) BinaryOperation {
	return BinaryOperation{
		op:  op,
		lhs: lhs,
		rhs: rhs,
	}
}

// nolint: revive
func (BinaryOperation) __fieldExpression() {}

func (o BinaryOperation) impliedType() StaticType {
	if o.op.isBoolean() {
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

func (o BinaryOperation) referencesSpan() bool {
	return o.lhs.referencesSpan() || o.rhs.referencesSpan()
}

type UnaryOperation struct {
	op Operator
	e  FieldExpression
}

func newUnaryOperation(op Operator, e FieldExpression) UnaryOperation {
	return UnaryOperation{
		op: op,
		e:  e,
	}
}

// nolint: revive
func (UnaryOperation) __fieldExpression() {}

func (o UnaryOperation) impliedType() StaticType {
	// both operators (opPower and opNot) will just be based on the operand type
	return o.e.impliedType()
}

func (o UnaryOperation) referencesSpan() bool {
	return o.e.referencesSpan()
}

// **********************
// Statics
// **********************
type Static struct {
	staticType StaticType
	n          int
	f          float64
	s          string
	b          bool
	d          time.Duration
	status     Status
}

// nolint: revive
func (Static) __fieldExpression() {}

// nolint: revive
func (Static) __scalarExpression() {}

func (Static) referencesSpan() bool {
	return false
}

func (s Static) impliedType() StaticType {
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

func newStaticStatus(s Status) Static {
	return Static{
		staticType: typeStatus,
		status:     s,
	}
}

// **********************
// Attributes
// **********************

type Attribute struct {
	scope     AttributeScope
	parent    bool
	name      string
	intrinsic Intrinsic
}

// newAttribute creates a new attribute with the given identifier string. If the identifier
//  string matches an intrinsic use that.
func newAttribute(att string) Attribute {
	intrinsic := intrinsicFromString(att)

	return Attribute{
		scope:     attributeScopeNone,
		parent:    false,
		name:      att,
		intrinsic: intrinsic,
	}
}

// nolint: revive
func (Attribute) __fieldExpression() {}

func (a Attribute) impliedType() StaticType {
	switch a.intrinsic {
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

	return typeAttribute
}

func (Attribute) referencesSpan() bool {
	return true
}

// newScopedAttribute creates a new scopedattribute with the given identifier string.
//  this handles parent, span, and resource scopes.
func newScopedAttribute(scope AttributeScope, parent bool, att string) Attribute {
	intrinsic := intrinsicNone
	// if we are explicitly passed a resource or span scopes then we shouldn't parse for intrinsic
	if scope != attributeScopeResource && scope != attributeScopeSpan {
		intrinsic = intrinsicFromString(att)
	}

	return Attribute{
		scope:     scope,
		parent:    parent,
		name:      att,
		intrinsic: intrinsic,
	}
}

func newIntrinsic(n Intrinsic) Attribute {
	return Attribute{
		scope:     attributeScopeNone,
		parent:    false,
		name:      n.String(),
		intrinsic: n,
	}
}
