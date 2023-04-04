package traceql

import (
	"fmt"
	"math"
	"time"
)

type Element interface {
	fmt.Stringer
	validate() error
}

type pipelineElement interface {
	Element
	extractConditions(request *FetchSpansRequest)
	evaluate([]*Spanset) ([]*Spanset, error)
}

type typedExpression interface {
	impliedType() StaticType
}

type RootExpr struct {
	Pipeline Pipeline
}

func newRootExpr(e pipelineElement) *RootExpr {
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
	Elements []pipelineElement
}

// nolint: revive
func (Pipeline) __scalarExpression() {}

// nolint: revive
func (Pipeline) __spansetExpression() {}

func newPipeline(i ...pipelineElement) Pipeline {
	return Pipeline{
		Elements: i,
	}
}

func (p Pipeline) addItem(i pipelineElement) Pipeline {
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

func (p Pipeline) extractConditions(req *FetchSpansRequest) {
	for _, element := range p.Elements {
		element.extractConditions(req)
	}
	// TODO this needs to be fine-tuned a bit, e.g. { .foo = "bar" } | by(.namespace), AllConditions can still be true
	if len(p.Elements) > 1 {
		req.AllConditions = false
	}
}

func (p Pipeline) evaluate(input []*Spanset) (result []*Spanset, err error) {
	result = input

	for _, element := range p.Elements {
		result, err = element.evaluate(result)
		if err != nil {
			return nil, err
		}

		if len(result) == 0 {
			return []*Spanset{}, nil
		}
	}

	return result, nil
}

type GroupOperation struct {
	Expression FieldExpression
}

func newGroupOperation(e FieldExpression) GroupOperation {
	return GroupOperation{
		Expression: e,
	}
}

func (o GroupOperation) extractConditions(request *FetchSpansRequest) {
	o.Expression.extractConditions(request)
}

func (GroupOperation) evaluate(ss []*Spanset) ([]*Spanset, error) {
	return ss, nil
}

type CoalesceOperation struct {
}

func newCoalesceOperation() CoalesceOperation {
	return CoalesceOperation{}
}

func (o CoalesceOperation) extractConditions(request *FetchSpansRequest) {
}

func (CoalesceOperation) evaluate(ss []*Spanset) ([]*Spanset, error) {
	return ss, nil
}

// **********************
// Scalars
// **********************
type ScalarExpression interface {
	//pipelineElement
	Element
	typedExpression
	__scalarExpression()

	extractConditions(request *FetchSpansRequest)
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

func (o ScalarOperation) extractConditions(request *FetchSpansRequest) {
	o.LHS.extractConditions(request)
	o.RHS.extractConditions(request)
	request.AllConditions = false
}

type Aggregate struct {
	op AggregateOp
	e  FieldExpression
}

func newAggregate(agg AggregateOp, e FieldExpression) Aggregate {
	return Aggregate{
		op: agg,
		e:  e,
	}
}

// nolint: revive
func (Aggregate) __scalarExpression() {}

func (a Aggregate) impliedType() StaticType {
	if a.op == aggregateCount || a.e == nil {
		return TypeInt
	}

	return a.e.impliedType()
}

func (a Aggregate) extractConditions(request *FetchSpansRequest) {
	if a.e != nil {
		a.e.extractConditions(request)
	}
}

// **********************
// Spansets
// **********************
type SpansetExpression interface {
	pipelineElement
	__spansetExpression()
}

type SpansetOperation struct {
	Op  Operator
	LHS SpansetExpression
	RHS SpansetExpression
}

func (o SpansetOperation) extractConditions(request *FetchSpansRequest) {
	o.LHS.extractConditions(request)
	o.RHS.extractConditions(request)
	request.AllConditions = false
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

func (f SpansetFilter) evaluate(input []*Spanset) ([]*Spanset, error) {
	var output []*Spanset

	for _, ss := range input {
		if len(ss.Spans) == 0 {
			continue
		}

		var matchingSpans []Span
		for _, s := range ss.Spans {
			result, err := f.Expression.execute(s)
			if err != nil {
				return nil, err
			}

			if result.Type != TypeBoolean {
				continue
			}

			if !result.B {
				continue
			}

			matchingSpans = append(matchingSpans, s)
		}

		if len(matchingSpans) == 0 {
			continue
		}

		matchingSpanset := *ss
		matchingSpanset.Spans = matchingSpans
		output = append(output, &matchingSpanset)
	}

	return output, nil
}

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

func (f ScalarFilter) extractConditions(request *FetchSpansRequest) {
	f.lhs.extractConditions(request)
	f.rhs.extractConditions(request)
	request.AllConditions = false
}

// **********************
// Expressions
// **********************
type FieldExpression interface {
	Element
	typedExpression

	// referencesSpan returns true if this field expression has any attributes or intrinsics. i.e. it references the span itself
	referencesSpan() bool
	__fieldExpression()

	extractConditions(request *FetchSpansRequest)
	execute(span Span) (Static, error)
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
	Status Status // todo: can we just use the N member for status and kind?
	Kind   Kind
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

func (s Static) Equals(other Static) bool {
	// if they are different number types. compare them as floats. however, if they are the same type just fall through to
	// a normal comparison which should be more efficient
	differentNumberTypes := (s.Type == TypeInt || s.Type == TypeFloat || s.Type == TypeDuration) &&
		(other.Type == TypeInt || other.Type == TypeFloat || other.Type == TypeDuration) &&
		s.Type != other.Type
	if differentNumberTypes {
		return s.asFloat() == other.asFloat()
	}

	eitherIsTypeStatus := (s.Type == TypeStatus && other.Type == TypeInt) || (other.Type == TypeStatus && s.Type == TypeInt)
	if eitherIsTypeStatus {
		if s.Type == TypeStatus {
			return s.Status == Status(other.N)
		}
		return Status(s.N) == other.Status
	}

	// no special cases, just compare directly
	return s == other
}

func (s Static) asFloat() float64 {
	switch s.Type {
	case TypeInt:
		return float64(s.N)
	case TypeFloat:
		return s.F
	case TypeDuration:
		return float64(s.D.Nanoseconds())
	default:
		return math.NaN()
	}
}

func NewStaticInt(n int) Static {
	return Static{
		Type: TypeInt,
		N:    n,
	}
}

func NewStaticFloat(f float64) Static {
	return Static{
		Type: TypeFloat,
		F:    f,
	}
}

func NewStaticString(s string) Static {
	return Static{
		Type: TypeString,
		S:    s,
	}
}

func NewStaticBool(b bool) Static {
	return Static{
		Type: TypeBoolean,
		B:    b,
	}
}

func NewStaticNil() Static {
	return Static{
		Type: TypeNil,
	}
}

func NewStaticDuration(d time.Duration) Static {
	return Static{
		Type: TypeDuration,
		D:    d,
	}
}

func NewStaticStatus(s Status) Static {
	return Static{
		Type:   TypeStatus,
		Status: s,
	}
}

func NewStaticKind(k Kind) Static {
	return Static{
		Type: TypeKind,
		Kind: k,
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

// NewAttribute creates a new attribute with the given identifier string.
func NewAttribute(att string) Attribute {
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
	case IntrinsicKind:
		return TypeKind
	case IntrinsicParent:
		return TypeNil
	}

	return TypeAttribute
}

func (Attribute) referencesSpan() bool {
	return true
}

// NewScopedAttribute creates a new scopedattribute with the given identifier string.
// this handles parent, span, and resource scopes.
func NewScopedAttribute(scope AttributeScope, parent bool, att string) Attribute {
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

func NewIntrinsic(n Intrinsic) Attribute {
	return Attribute{
		Scope:     AttributeScopeNone,
		Parent:    false,
		Name:      n.String(),
		Intrinsic: n,
	}
}

var _ pipelineElement = (*Pipeline)(nil)
var _ pipelineElement = (*Aggregate)(nil)
var _ pipelineElement = (*SpansetOperation)(nil)
var _ pipelineElement = (*SpansetFilter)(nil)
var _ pipelineElement = (*CoalesceOperation)(nil)
var _ pipelineElement = (*ScalarFilter)(nil)
var _ pipelineElement = (*GroupOperation)(nil)
