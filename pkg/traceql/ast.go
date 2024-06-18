package traceql

import (
	"cmp"
	"fmt"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"hash/crc32"
	"math"
	"regexp"
	"time"
)

type Element interface {
	fmt.Stringer
	validate() error
}

type metricsFirstStageElement interface {
	Element
	extractConditions(request *FetchSpansRequest)
	init(req *tempopb.QueryRangeRequest, mode AggregateMode)
	observe(Span)                        // TODO - batching?
	observeSeries([]*tempopb.TimeSeries) // Re-entrant metrics on the query-frontend.  Using proto version for efficiency
	result() SeriesSet
}

type pipelineElement interface {
	Element
	extractConditions(request *FetchSpansRequest)
	evaluate([]*Spanset) ([]*Spanset, error)
}

type typedExpression interface {
	Type() StaticType
}

type RootExpr struct {
	Pipeline        Pipeline
	MetricsPipeline metricsFirstStageElement
	Hints           *Hints
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

func newRootExprWithMetrics(e pipelineElement, m metricsFirstStageElement) *RootExpr {
	p, ok := e.(Pipeline)
	if !ok {
		p = newPipeline(e)
	}

	return &RootExpr{
		Pipeline:        p,
		MetricsPipeline: m,
	}
}

func (r *RootExpr) withHints(h *Hints) *RootExpr {
	r.Hints = h
	return r
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

func (p Pipeline) Type() StaticType {
	if len(p.Elements) == 0 {
		return TypeSpanset
	}

	finalItem := p.Elements[len(p.Elements)-1]
	aggregate, ok := finalItem.(Aggregate)
	if ok {
		return aggregate.Type()
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

	groupBuffer map[StaticHashCode]*Spanset
}

func newGroupOperation(e FieldExpression) GroupOperation {
	return GroupOperation{
		Expression:  e,
		groupBuffer: make(map[StaticHashCode]*Spanset),
	}
}

func (o GroupOperation) extractConditions(request *FetchSpansRequest) {
	o.Expression.extractConditions(request)
}

type CoalesceOperation struct{}

func newCoalesceOperation() CoalesceOperation {
	return CoalesceOperation{}
}

func (o CoalesceOperation) extractConditions(*FetchSpansRequest) {
}

type SelectOperation struct {
	attrs []Attribute
}

func newSelectOperation(exprs []Attribute) SelectOperation {
	return SelectOperation{
		attrs: exprs,
	}
}

// **********************
// Scalars
// **********************
type ScalarExpression interface {
	// pipelineElement
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

func newScalarOperation(op Operator, lhs, rhs ScalarExpression) ScalarOperation {
	return ScalarOperation{
		Op:  op,
		LHS: lhs,
		RHS: rhs,
	}
}

// nolint: revive
func (ScalarOperation) __scalarExpression() {}

func (o ScalarOperation) Type() StaticType {
	if o.Op.isBoolean() {
		return TypeBoolean
	}

	// remaining operators will be based on the operands
	// opAdd, opSub, opDiv, opMod, opMult
	t := o.LHS.Type()
	if t != TypeAttribute {
		return t
	}

	return o.RHS.Type()
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

func (a Aggregate) Type() StaticType {
	if a.op == aggregateCount || a.e == nil {
		return TypeInt
	}

	return a.e.Type()
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
	Op                  Operator
	LHS                 SpansetExpression
	RHS                 SpansetExpression
	matchingSpansBuffer []Span
}

func (o SpansetOperation) extractConditions(request *FetchSpansRequest) {
	switch o.Op {
	case OpSpansetDescendant, OpSpansetAncestor, OpSpansetNotDescendant, OpSpansetNotAncestor, OpSpansetUnionDescendant, OpSpansetUnionAncestor:
		request.Conditions = append(request.Conditions, Condition{
			Attribute: NewIntrinsic(IntrinsicStructuralDescendant),
		})
	case OpSpansetChild, OpSpansetParent, OpSpansetNotChild, OpSpansetNotParent, OpSpansetUnionChild, OpSpansetUnionParent:
		request.Conditions = append(request.Conditions, Condition{
			Attribute: NewIntrinsic(IntrinsicStructuralChild),
		})
	case OpSpansetSibling, OpSpansetNotSibling, OpSpansetUnionSibling:
		request.Conditions = append(request.Conditions, Condition{
			Attribute: NewIntrinsic(IntrinsicStructuralSibling),
		})
	}

	o.LHS.extractConditions(request)
	o.RHS.extractConditions(request)

	request.AllConditions = false
}

func newSpansetOperation(op Operator, lhs, rhs SpansetExpression) SpansetOperation {
	return SpansetOperation{
		Op:  op,
		LHS: lhs,
		RHS: rhs,
	}
}

// nolint: revive
func (SpansetOperation) __spansetExpression() {}

type SpansetFilter struct {
	Expression          FieldExpression
	matchingSpansBuffer []Span
}

func newSpansetFilter(e FieldExpression) *SpansetFilter {
	return &SpansetFilter{
		Expression: e,
	}
}

// nolint: revive
func (*SpansetFilter) __spansetExpression() {}

func (f *SpansetFilter) evaluate(input []*Spanset) ([]*Spanset, error) {
	var outputBuffer []*Spanset

	for _, ss := range input {
		if len(ss.Spans) == 0 {
			continue
		}

		f.matchingSpansBuffer = f.matchingSpansBuffer[:0]

		for _, s := range ss.Spans {
			result, err := f.Expression.execute(s)
			if err != nil {
				return nil, err
			}

			boolResult, ok := result.(StaticBool)
			if !ok || !boolResult.Bool {
				continue
			}

			f.matchingSpansBuffer = append(f.matchingSpansBuffer, s)
		}

		if len(f.matchingSpansBuffer) == 0 {
			continue
		}

		if len(f.matchingSpansBuffer) == len(ss.Spans) {
			// All matched, so we return the input as-is
			// and preserve the local buffer.
			outputBuffer = append(outputBuffer, ss)
			continue
		}

		matchingSpanset := ss.clone()
		matchingSpanset.Spans = append([]Span(nil), f.matchingSpansBuffer...)
		outputBuffer = append(outputBuffer, matchingSpanset)
	}

	return outputBuffer, nil
}

type ScalarFilter struct {
	op  Operator
	lhs ScalarExpression
	rhs ScalarExpression
}

func newScalarFilter(op Operator, lhs, rhs ScalarExpression) ScalarFilter {
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

	compiledExpression *regexp.Regexp
}

func newBinaryOperation(op Operator, lhs, rhs FieldExpression) FieldExpression {
	binop := &BinaryOperation{
		Op:  op,
		LHS: lhs,
		RHS: rhs,
	}

	if !binop.referencesSpan() && binop.validate() == nil {
		if simplified, err := binop.execute(nil); err == nil {
			return simplified
		}
	}

	return binop
}

// nolint: revive
func (BinaryOperation) __fieldExpression() {}

func (o *BinaryOperation) Type() StaticType {
	if o.Op.isBoolean() {
		return TypeBoolean
	}

	// remaining operators will be based on the operands
	// opAdd, opSub, opDiv, opMod, opMult
	t := o.LHS.Type()
	if t != TypeAttribute {
		return t
	}

	return o.RHS.Type()
}

func (o *BinaryOperation) referencesSpan() bool {
	return o.LHS.referencesSpan() || o.RHS.referencesSpan()
}

type UnaryOperation struct {
	Op         Operator
	Expression FieldExpression
}

func newUnaryOperation(op Operator, e FieldExpression) FieldExpression {
	unop := UnaryOperation{
		Op:         op,
		Expression: e,
	}

	if !unop.referencesSpan() && unop.validate() == nil {
		if simplified, err := unop.execute(nil); err == nil {
			return simplified
		}
	}

	return unop
}

// nolint: revive
func (UnaryOperation) __fieldExpression() {}

func (o UnaryOperation) Type() StaticType {
	// both operators (opPower and opNot) will just be based on the operand type
	return o.Expression.Type()
}

func (o UnaryOperation) referencesSpan() bool {
	return o.Expression.referencesSpan()
}

// **********************
// Statics
// **********************

type Static interface {
	ScalarExpression
	FieldExpression

	AsAnyValue() *common_v1.AnyValue
	EncodeToString(quotes bool) string
	equals(o Static) bool
	compare(o Static) int
	asFloat() float64
	add(o Static) Static
	divide(f float64) Static
	hashCode() StaticHashCode
}

type StaticHashCode struct {
	Type StaticType
	Hash uint64
}

// StaticBase partially implements Static and is meant to be embedded in other static types
type StaticBase struct{}

func (s StaticBase) asFloat() float64 {
	return math.NaN()
}

func (StaticBase) referencesSpan() bool {
	return false
}

func (StaticBase) __fieldExpression() {}

func (StaticBase) __scalarExpression() {}

// StaticNil a nil representation of a static value
type StaticNil struct {
	StaticBase
}

var staticNil Static = StaticNil{}

func NewStaticNil() Static {
	return staticNil
}

func (StaticNil) Type() StaticType {
	return TypeNil
}

func (StaticNil) equals(o Static) bool {
	_, ok := o.(StaticNil)
	return ok
}

func (StaticNil) compare(o Static) int {
	_, ok := o.(StaticNil)
	if !ok {
		return cmp.Compare(TypeNil, o.Type())
	}
	return 0
}

func (StaticNil) add(_ Static) Static {
	return NewStaticNil()
}

func (StaticNil) divide(_ float64) Static {
	return NewStaticNil()
}

func (StaticNil) hashCode() StaticHashCode {
	return StaticHashCode{Type: TypeNil}
}

// StaticInt represents a Static implementation based on int
type StaticInt struct {
	StaticBase
	Int int
}

var _ Static = StaticInt{}

func NewStaticInt(n int) StaticInt {
	return StaticInt{Int: n}
}

func (s StaticInt) Type() StaticType {
	return TypeInt
}

func (s StaticInt) equals(o Static) bool {
	switch o := o.(type) {
	case StaticInt:
		return s.Int == o.Int
	case StaticDuration:
		return s.Int == int(o.Duration)
	case StaticFloat:
		return float64(s.Int) == o.Float
	case StaticStatus:
		return s.Int == int(o.Status)
	default:
		return false
	}
}

func (s StaticInt) compare(o Static) int {
	switch o := o.(type) {
	case StaticInt:
		return cmp.Compare(s.Int, o.Int)
	case StaticDuration:
		return cmp.Compare(s.Int, int(o.Duration))
	case StaticFloat:
		return cmp.Compare(float64(s.Int), o.Float)
	case StaticStatus:
		return cmp.Compare(s.Int, int(o.Status))
	default:
		return cmp.Compare(TypeInt, o.Type())
	}
}

func (s StaticInt) add(o Static) Static {
	n, ok := o.(StaticInt)
	if ok {
		s.Int += n.Int
	}
	return s
}

func (s StaticInt) divide(f float64) Static {
	return NewStaticFloat(float64(s.Int) / f)
}

func (s StaticInt) asFloat() float64 {
	return float64(s.Int)
}

func (s StaticInt) hashCode() StaticHashCode {
	return StaticHashCode{Type: TypeInt, Hash: uint64(s.Int)}
}

// StaticFloat represents a Static implementation based on float64
type StaticFloat struct {
	StaticBase
	Float float64
}

var _ Static = StaticFloat{}

func NewStaticFloat(f float64) StaticFloat {
	return StaticFloat{Float: f}
}

func (s StaticFloat) Type() StaticType {
	return TypeFloat
}

func (s StaticFloat) equals(o Static) bool {
	switch o := o.(type) {
	case StaticFloat:
		return s.Float == o.Float
	case StaticInt:
		return s.Float == float64(o.Int)
	case StaticDuration:
		return s.Float == float64(o.Duration)
	default:
		return false
	}
}

func (s StaticFloat) compare(o Static) int {
	switch o := o.(type) {
	case StaticFloat:
		return cmp.Compare(s.Float, o.Float)
	case StaticInt:
		return cmp.Compare(s.Float, float64(o.Int))
	case StaticDuration:
		return cmp.Compare(s.Float, float64(o.Duration))
	default:
		return cmp.Compare(TypeFloat, o.Type())
	}
}

func (s StaticFloat) asFloat() float64 {
	return s.Float
}

func (s StaticFloat) add(o Static) Static {
	n, ok := o.(StaticFloat)
	if ok {
		s.Float += n.Float
	}
	return s
}

func (s StaticFloat) divide(f float64) Static {
	s.Float /= f
	return s
}

func (s StaticFloat) hashCode() StaticHashCode {
	return StaticHashCode{Type: TypeFloat, Hash: math.Float64bits(s.Float)}
}

// StaticString represents a Static implementation based on string
type StaticString struct {
	StaticBase
	Str string
}

var _ Static = StaticString{}

func NewStaticString(s string) StaticString {
	return StaticString{Str: s}
}

func (s StaticString) Type() StaticType {
	return TypeString
}

func (s StaticString) equals(o Static) bool {
	v, ok := o.(StaticString)
	if !ok {
		return false
	}
	return s.Str == v.Str
}

func (s StaticString) compare(o Static) int {
	v, ok := o.(StaticString)
	if !ok {
		return cmp.Compare(TypeString, o.Type())
	}
	return cmp.Compare(s.Str, v.Str)
}

func (s StaticString) add(_ Static) Static {
	return s
}

func (s StaticString) divide(_ float64) Static {
	return s
}

func (s StaticString) hashCode() StaticHashCode {
	return StaticHashCode{Type: TypeString, Hash: uint64(crc32.ChecksumIEEE([]byte(s.Str)))}
}

// StaticBool represents a Static implementation based on bool
type StaticBool struct {
	StaticBase
	Bool bool
}

var _ Static = StaticBool{}

func NewStaticBool(b bool) StaticBool {
	return StaticBool{Bool: b}
}

func (s StaticBool) Type() StaticType {
	return TypeBoolean
}

func (s StaticBool) equals(o Static) bool {
	v, ok := o.(StaticBool)
	if !ok {
		return false
	}
	return s.Bool == v.Bool
}

func (s StaticBool) compare(o Static) int {
	v, ok := o.(StaticBool)
	if !ok {
		return cmp.Compare(TypeBoolean, o.Type())
	}
	if s.Bool && !v.Bool {
		return 1
	} else if !s.Bool && v.Bool {
		return -1
	}
	return 0
}

func (s StaticBool) add(_ Static) Static {
	return s
}

func (s StaticBool) divide(_ float64) Static {
	return s
}

func (s StaticBool) hashCode() StaticHashCode {
	if s.Bool {
		return StaticHashCode{Type: TypeBoolean, Hash: 1}
	}
	return StaticHashCode{Type: TypeBoolean, Hash: 0}
}

// StaticDuration represents a Static implementation based on time.Duration
type StaticDuration struct {
	StaticBase
	Duration time.Duration
}

var _ Static = StaticDuration{}

func NewStaticDuration(d time.Duration) StaticDuration {
	return StaticDuration{Duration: d}
}

func (s StaticDuration) Type() StaticType {
	return TypeDuration
}

func (s StaticDuration) equals(o Static) bool {
	switch o := o.(type) {
	case StaticDuration:
		return s.Duration == o.Duration
	case StaticInt:
		return int(s.Duration) == o.Int
	case StaticFloat:
		return float64(s.Duration) == o.Float
	default:
		return false
	}
}

func (s StaticDuration) compare(o Static) int {
	switch o := o.(type) {
	case StaticDuration:
		return cmp.Compare(s.Duration, o.Duration)
	case StaticInt:
		return cmp.Compare(int(s.Duration), o.Int)
	case StaticFloat:
		return cmp.Compare(float64(s.Duration), o.Float)
	default:
		return cmp.Compare(TypeDuration, o.Type())
	}
}

func (s StaticDuration) asFloat() float64 {
	return float64(s.Duration)
}

func (s StaticDuration) add(o Static) Static {
	d, ok := o.(StaticDuration)
	if ok {
		s.Duration += d.Duration
	}
	return s
}

func (s StaticDuration) divide(f float64) Static {
	d := time.Duration(float64(s.Duration) / f)
	return NewStaticDuration(d)
}

func (s StaticDuration) hashCode() StaticHashCode {
	return StaticHashCode{Type: TypeDuration, Hash: uint64(s.Duration)}
}

// StaticStatus represents a Static implementation based on Status
type StaticStatus struct {
	StaticBase
	Status Status
}

var _ Static = StaticStatus{}

func NewStaticStatus(s Status) StaticStatus {
	return StaticStatus{Status: s}
}

func (s StaticStatus) Type() StaticType {
	return TypeStatus
}

func (s StaticStatus) equals(o Static) bool {
	switch o := o.(type) {
	case StaticStatus:
		return s.Status == o.Status
	case StaticInt:
		return s.Status == Status(o.Int)
	default:
		return false
	}
}

func (s StaticStatus) compare(o Static) int {
	switch o := o.(type) {
	case StaticStatus:
		return cmp.Compare(s.Status, o.Status)
	case StaticInt:
		return cmp.Compare(s.Status, Status(o.Int))
	default:
		return cmp.Compare(TypeStatus, o.Type())
	}
}

func (s StaticStatus) add(_ Static) Static {
	return s
}

func (s StaticStatus) divide(_ float64) Static {
	return s
}

func (s StaticStatus) hashCode() StaticHashCode {
	return StaticHashCode{Type: TypeStatus, Hash: uint64(s.Status)}
}

// StaticKind represents a Static implementation based on Kind
type StaticKind struct {
	StaticBase
	Kind Kind
}

var _ Static = StaticKind{}

func NewStaticKind(k Kind) StaticKind {
	return StaticKind{Kind: k}
}

func (s StaticKind) Type() StaticType {
	return TypeKind
}

func (s StaticKind) equals(o Static) bool {
	v, ok := o.(StaticKind)
	if !ok {
		return false
	}
	return s.Kind == v.Kind
}

func (s StaticKind) compare(o Static) int {
	v, ok := o.(StaticKind)
	if !ok {
		return cmp.Compare(TypeKind, o.Type())
	}
	return cmp.Compare(s.Kind, v.Kind)
}

func (s StaticKind) add(_ Static) Static {
	return s
}

func (s StaticKind) divide(_ float64) Static {
	return s
}

func (s StaticKind) hashCode() StaticHashCode {
	return StaticHashCode{Type: TypeKind, Hash: uint64(s.Kind)}
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

func (a Attribute) Type() StaticType {
	switch a.Intrinsic {
	case IntrinsicDuration:
		return TypeDuration
	case IntrinsicChildCount:
		return TypeInt
	case IntrinsicName:
		return TypeString
	case IntrinsicStatus:
		return TypeStatus
	case IntrinsicStatusMessage:
		return TypeString
	case IntrinsicKind:
		return TypeKind
	case IntrinsicEventName:
		return TypeString
	case IntrinsicLinkTraceID:
		return TypeString
	case IntrinsicLinkSpanID:
		return TypeString
	case IntrinsicParent:
		return TypeNil
	case IntrinsicTraceDuration:
		return TypeDuration
	case IntrinsicTraceRootService:
		return TypeString
	case IntrinsicTraceRootSpan:
		return TypeString
	case IntrinsicNestedSetLeft:
		return TypeInt
	case IntrinsicNestedSetRight:
		return TypeInt
	case IntrinsicNestedSetParent:
		return TypeInt
	case IntrinsicTraceID:
		return TypeString
	case IntrinsicSpanID:
		return TypeString
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
	if scope == AttributeScopeNone && !parent {
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

var (
	_ pipelineElement = (*Pipeline)(nil)
	_ pipelineElement = (*Aggregate)(nil)
	_ pipelineElement = (*SpansetOperation)(nil)
	_ pipelineElement = (*SpansetFilter)(nil)
	_ pipelineElement = (*CoalesceOperation)(nil)
	_ pipelineElement = (*ScalarFilter)(nil)
	_ pipelineElement = (*GroupOperation)(nil)
)

// MetricsAggregate is a placeholder in the AST for a metrics aggregation
// pipeline element. It has a superset of the properties of them all, and
// builds them later via init() so that appropriate buffers can be allocated
// for the query time range and step, and different implementations for
// shardable and unshardable pipelines.
type MetricsAggregate struct {
	op        MetricsAggregateOp
	by        []Attribute
	attr      Attribute
	floats    []float64
	agg       SpanAggregator
	seriesAgg SeriesAggregator
}

func newMetricsAggregate(agg MetricsAggregateOp, by []Attribute) *MetricsAggregate {
	return &MetricsAggregate{
		op: agg,
		by: by,
	}
}

func newMetricsAggregateQuantileOverTime(attr Attribute, qs []float64, by []Attribute) *MetricsAggregate {
	return &MetricsAggregate{
		op:     metricsAggregateQuantileOverTime,
		floats: qs,
		attr:   attr,
		by:     by,
	}
}

func newMetricsAggregateHistogramOverTime(attr Attribute, by []Attribute) *MetricsAggregate {
	return &MetricsAggregate{
		op:   metricsAggregateHistogramOverTime,
		by:   by,
		attr: attr,
	}
}

func (a *MetricsAggregate) extractConditions(request *FetchSpansRequest) {
	switch a.op {
	case metricsAggregateRate, metricsAggregateCountOverTime:
		// No extra conditions, start time is already enough
	case metricsAggregateQuantileOverTime, metricsAggregateHistogramOverTime:
		if !request.HasAttribute(a.attr) {
			request.SecondPassConditions = append(request.SecondPassConditions, Condition{
				Attribute: a.attr,
			})
		}
	}

	for _, b := range a.by {
		if !request.HasAttribute(b) {
			request.SecondPassConditions = append(request.SecondPassConditions, Condition{
				Attribute: b,
			})
		}
	}
}

func (a *MetricsAggregate) init(q *tempopb.QueryRangeRequest, mode AggregateMode) {
	switch mode {
	case AggregateModeSum:
		a.initSum(q)
		return

	case AggregateModeFinal:
		a.initFinal(q)
		return
	}

	// Raw mode:

	var innerAgg func() VectorAggregator
	var byFunc func(Span) (Static, bool)
	var byFuncLabel string

	switch a.op {
	case metricsAggregateCountOverTime:
		innerAgg = func() VectorAggregator { return NewCountOverTimeAggregator() }

	case metricsAggregateRate:
		innerAgg = func() VectorAggregator { return NewRateAggregator(1.0 / time.Duration(q.Step).Seconds()) }

	case metricsAggregateHistogramOverTime:
		// Histograms are implemented as count_over_time() by(2^log2(attr)) for now
		// This is very similar to quantile_over_time except the bucket values are the true
		// underlying value in scale, i.e. a duration of 500ms will be in __bucket==0.512s
		// The difference is that quantile_over_time has to calculate the final quantiles
		// so in that case the log2 bucket number is more useful.  We can clean it up later
		// when updating quantiles to be smarter and more customizable range of buckets.
		innerAgg = func() VectorAggregator { return NewCountOverTimeAggregator() }
		byFuncLabel = internalLabelBucket
		switch a.attr {
		case IntrinsicDurationAttribute:
			// Optimal implementation for duration attribute
			byFunc = func(s Span) (Static, bool) {
				d := s.DurationNanos()
				if d < 2 {
					return NewStaticNil(), false
				}
				// Bucket is log2(nanos) converted to float seconds
				return NewStaticFloat(Log2Bucketize(d) / float64(time.Second)), true
			}
		default:
			// Basic implementation for all other attributes
			byFunc = func(s Span) (Static, bool) {
				v, ok := s.AttributeFor(a.attr)
				if !ok {
					return NewStaticNil(), false
				}

				// TODO(mdisibio) - Add support for floats, we need to map them into buckets.
				// Because of the range of floats, we need a native histogram approach.
				n, ok := v.(StaticInt)
				if !ok || n.Int < 2 {
					return NewStaticNil(), false
				}

				// Bucket is the value rounded up to the nearest power of 2
				return NewStaticFloat(Log2Bucketize(uint64(n.Int))), true
			}
		}

	case metricsAggregateQuantileOverTime:
		// Quantiles are implemented as count_over_time() by(log2(attr)) for now
		innerAgg = func() VectorAggregator { return NewCountOverTimeAggregator() }
		byFuncLabel = internalLabelBucket
		switch a.attr {
		case IntrinsicDurationAttribute:
			// Optimal implementation for duration attribute
			byFunc = func(s Span) (Static, bool) {
				d := s.DurationNanos()
				if d < 2 {
					return NewStaticNil(), false
				}
				// Bucket is in seconds
				return NewStaticFloat(Log2Bucketize(d) / float64(time.Second)), true
			}
		default:
			// Basic implementation for all other attributes
			byFunc = func(s Span) (Static, bool) {
				v, ok := s.AttributeFor(a.attr)
				if !ok {
					return NewStaticNil(), false
				}

				// TODO(mdisibio) - Add support for floats, we need to map them into buckets.
				// Because of the range of floats, we need a native histogram approach.
				n, ok := v.(StaticInt)
				if !ok || n.Int < 2 {
					return NewStaticNil(), false
				}

				return NewStaticFloat(Log2Bucketize(uint64(n.Int))), true
			}
		}
	}

	a.agg = NewGroupingAggregator(a.op.String(), func() RangeAggregator {
		return NewStepAggregator(q.Start, q.End, q.Step, innerAgg)
	}, a.by, byFunc, byFuncLabel)
}

func (a *MetricsAggregate) initSum(q *tempopb.QueryRangeRequest) {
	// Currently all metrics are summed by job to produce
	// intermediate results. This will change when adding min/max/topk/etc
	a.seriesAgg = NewSimpleAdditionCombiner(q)
}

func (a *MetricsAggregate) initFinal(q *tempopb.QueryRangeRequest) {
	switch a.op {
	case metricsAggregateQuantileOverTime:
		a.seriesAgg = NewHistogramAggregator(q, a.floats)
	default:
		// These are simple additions by series
		a.seriesAgg = NewSimpleAdditionCombiner(q)
	}
}

func (a *MetricsAggregate) observe(span Span) {
	a.agg.Observe(span)
}

func (a *MetricsAggregate) observeSeries(ss []*tempopb.TimeSeries) {
	a.seriesAgg.Combine(ss)
}

func (a *MetricsAggregate) result() SeriesSet {
	if a.agg != nil {
		return a.agg.Series()
	}

	// In the frontend-version the results come from
	// the job-level aggregator
	return a.seriesAgg.Results()
}

func (a *MetricsAggregate) validate() error {
	switch a.op {
	case metricsAggregateCountOverTime:
	case metricsAggregateRate:
	case metricsAggregateHistogramOverTime:
		if len(a.by) >= maxGroupBys {
			// We reserve a spot for the bucket so quantile has 1 less group by
			return newUnsupportedError(fmt.Sprintf("metrics group by %v values", len(a.by)))
		}
	case metricsAggregateQuantileOverTime:
		if len(a.by) >= maxGroupBys {
			// We reserve a spot for the bucket so quantile has 1 less group by
			return newUnsupportedError(fmt.Sprintf("metrics group by %v values", len(a.by)))
		}
		for _, q := range a.floats {
			if q < 0 || q > 1 {
				return fmt.Errorf("quantile must be between 0 and 1: %v", q)
			}
		}
	default:
		return newUnsupportedError(fmt.Sprintf("metrics aggregate operation (%v)", a.op))
	}

	if len(a.by) > maxGroupBys {
		return newUnsupportedError(fmt.Sprintf("metrics group by %v values", len(a.by)))
	}

	return nil
}

var _ metricsFirstStageElement = (*MetricsAggregate)(nil)
