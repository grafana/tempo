package traceql

import (
	"fmt"
	"time"
)

type Element interface {
	fmt.Stringer
	validate() error
}

type typedExpression interface {
	impliedType() StaticType
}

type RootExpr struct {
	Pipeline Pipeline
}

func newRootExpr(e Element) *RootExpr {
	p, ok := e.(Pipeline)
	if !ok {
		p = newPipeline(e)
	}

	return &RootExpr{
		Pipeline: p,
	}
}

// **********************
// Pipeline
// **********************

type Pipeline struct {
	Elements []Element
}

// nolint: revive
func (Pipeline) __scalarExpression() {}

// nolint: revive
func (Pipeline) __spansetExpression() {}

func newPipeline(i ...Element) Pipeline {
	return Pipeline{
		Elements: i,
	}
}

func (p Pipeline) addItem(i Element) Pipeline {
	p.Elements = append(p.Elements, i)
	return p
}

func (p Pipeline) impliedType() StaticType {
	if len(p.Elements) == 0 {
		return TypeSpanset
	}

	finalItem := p.Elements[len(p.Elements)-1]
	aggregate, ok := finalItem.(Aggregate)
	if ok {
		return aggregate.impliedType()
	}

	return TypeSpanset
}

type GroupOperation struct {
	Expression FieldExpression
}

func newGroupOperation(e FieldExpression) GroupOperation {
	return GroupOperation{
		Expression: e,
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
	Element
	typedExpression
	__scalarExpression()
}

type ScalarOperation struct {
	Op  Operator
	LHS ScalarExpression
	RHS ScalarExpression
}

func newScalarOperation(op Operator, lhs ScalarExpression, rhs ScalarExpression) ScalarOperation {
	return ScalarOperation{
		Op:  op,
		LHS: lhs,
		RHS: rhs,
	}
}

// nolint: revive
func (ScalarOperation) __scalarExpression() {}

func (o ScalarOperation) impliedType() StaticType {
	if o.Op.isBoolean() {
		return TypeBoolean
	}

	// remaining operators will be based on the operands
	// opAdd, opSub, opDiv, opMod, opMult
	t := o.LHS.impliedType()
	if t != TypeAttribute {
		return t
	}

	return o.RHS.impliedType()
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
		return TypeInt
	}

	return a.e.impliedType()
}

// **********************
// Spansets
// **********************
type SpansetExpression interface {
	Element
	__spansetExpression()
}

type SpansetOperation struct {
	Op  Operator
	LHS SpansetExpression
	RHS SpansetExpression
}

func newSpansetOperation(op Operator, lhs SpansetExpression, rhs SpansetExpression) SpansetOperation {
	return SpansetOperation{
		Op:  op,
		LHS: lhs,
		RHS: rhs,
	}
}

// nolint: revive
func (SpansetOperation) __spansetExpression() {}

type SpansetFilter struct {
	Expression FieldExpression
}

func newSpansetFilter(e FieldExpression) SpansetFilter {
	return SpansetFilter{
		Expression: e,
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
	Element
	typedExpression

	// referencesSpan returns true if this field expression has any attributes or intrinsics. i.e. it references the span itself
	referencesSpan() bool
	__fieldExpression()
}

type BinaryOperation struct {
	Op  Operator
	LHS FieldExpression
	RHS FieldExpression
}

func newBinaryOperation(op Operator, lhs FieldExpression, rhs FieldExpression) BinaryOperation {
	return BinaryOperation{
		Op:  op,
		LHS: lhs,
		RHS: rhs,
	}
}

// nolint: revive
func (BinaryOperation) __fieldExpression() {}

func (o BinaryOperation) impliedType() StaticType {
	if o.Op.isBoolean() {
		return TypeBoolean
	}

	// remaining operators will be based on the operands
	// opAdd, opSub, opDiv, opMod, opMult
	t := o.LHS.impliedType()
	if t != TypeAttribute {
		return t
	}

	return o.RHS.impliedType()
}

func (o BinaryOperation) referencesSpan() bool {
	return o.LHS.referencesSpan() || o.RHS.referencesSpan()
}

type UnaryOperation struct {
	Op         Operator
	Expression FieldExpression
}

func newUnaryOperation(op Operator, e FieldExpression) UnaryOperation {
	return UnaryOperation{
		Op:         op,
		Expression: e,
	}
}

// nolint: revive
func (UnaryOperation) __fieldExpression() {}

func (o UnaryOperation) impliedType() StaticType {
	// both operators (opPower and opNot) will just be based on the operand type
	return o.Expression.impliedType()
}

func (o UnaryOperation) referencesSpan() bool {
	return o.Expression.referencesSpan()
}

// **********************
// Statics
// **********************
type Static struct {
	Type   StaticType
	N      int
	F      float64
	S      string
	B      bool
	D      time.Duration
	Status Status
}

// nolint: revive
func (Static) __fieldExpression() {}

// nolint: revive
func (Static) __scalarExpression() {}

func (Static) referencesSpan() bool {
	return false
}

func (s Static) impliedType() StaticType {
	return s.Type
}

func newStaticInt(n int) Static {
	return Static{
		Type: TypeInt,
		N:    n,
	}
}

func newStaticFloat(f float64) Static {
	return Static{
		Type: TypeFloat,
		F:    f,
	}
}

func newStaticString(s string) Static {
	return Static{
		Type: TypeString,
		S:    s,
	}
}

func newStaticBool(b bool) Static {
	return Static{
		Type: TypeBoolean,
		B:    b,
	}
}

func newStaticNil() Static {
	return Static{
		Type: TypeNil,
	}
}

func newStaticDuration(d time.Duration) Static {
	return Static{
		Type: TypeDuration,
		D:    d,
	}
}

func newStaticStatus(s Status) Static {
	return Static{
		Type:   TypeStatus,
		Status: s,
	}
}

// **********************
// Attributes
// **********************

type Attribute struct {
	Scope     AttributeScope
	Parent    bool
	Name      string
	Intrinsic Intrinsic
}

// newAttribute creates a new attribute with the given identifier string.
func newAttribute(att string) Attribute {
	return Attribute{
		Scope:     AttributeScopeNone,
		Parent:    false,
		Name:      att,
		Intrinsic: IntrinsicNone,
	}
}

// nolint: revive
func (Attribute) __fieldExpression() {}

func (a Attribute) impliedType() StaticType {
	switch a.Intrinsic {
	case IntrinsicDuration:
		return TypeDuration
	case IntrinsicChildCount:
		return TypeInt
	case IntrinsicName:
		return TypeString
	case IntrinsicStatus:
		return TypeStatus
	case IntrinsicParent:
		return TypeNil
	}

	return TypeAttribute
}

func (Attribute) referencesSpan() bool {
	return true
}

// newScopedAttribute creates a new scopedattribute with the given identifier string.
// this handles parent, span, and resource scopes.
func newScopedAttribute(scope AttributeScope, parent bool, att string) Attribute {
	intrinsic := IntrinsicNone
	// if we are explicitly passed a resource or span scopes then we shouldn't parse for intrinsic
	if scope != AttributeScopeResource && scope != AttributeScopeSpan {
		intrinsic = intrinsicFromString(att)
	}

	return Attribute{
		Scope:     scope,
		Parent:    parent,
		Name:      att,
		Intrinsic: intrinsic,
	}
}

func newIntrinsic(n Intrinsic) Attribute {
	return Attribute{
		Scope:     AttributeScopeNone,
		Parent:    false,
		Name:      n.String(),
		Intrinsic: n,
	}
}
