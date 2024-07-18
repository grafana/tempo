package traceql

import (
	"bytes"
	"cmp"
	"fmt"
	"hash/fnv"
	"math"
	"regexp"
	"slices"
	"time"
	"unsafe"

	"github.com/grafana/tempo/pkg/tempopb"
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
	impliedType() StaticType
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

	groupBuffer map[StaticMapKey]*Spanset
}

func newGroupOperation(e FieldExpression) GroupOperation {
	return GroupOperation{
		Expression:  e,
		groupBuffer: make(map[StaticMapKey]*Spanset),
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

			if b, ok := result.Bool(); !ok || !b {
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

func (o *BinaryOperation) impliedType() StaticType {
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
	Type StaticType

	valScalar  uint64   // used for int, float64, bool, time.Duration, Kind, and Status
	valBytes   []byte   // used for string, []int, []float64, []bool
	valStrings []string // used for []string
}

type StaticMapKey struct {
	typ  StaticType
	code uint64
	str  string
}

func NewStaticNil() Static {
	return Static{Type: TypeNil}
}

func NewStaticInt(i int) Static {
	return Static{
		Type:      TypeInt,
		valScalar: uint64(i),
	}
}

func NewStaticFloat(f float64) Static {
	return Static{
		Type:      TypeFloat,
		valScalar: math.Float64bits(f),
	}
}

func NewStaticString(s string) Static {
	return Static{
		Type:     TypeString,
		valBytes: unsafe.Slice(unsafe.StringData(s), len(s)),
	}
}

func NewStaticBool(b bool) Static {
	var val uint64
	if b {
		val = 1
	}
	return Static{
		Type:      TypeBoolean,
		valScalar: val,
	}
}

func NewStaticDuration(d time.Duration) Static {
	return Static{
		Type:      TypeDuration,
		valScalar: uint64(d),
	}
}

func NewStaticStatus(s Status) Static {
	return Static{
		Type:      TypeStatus,
		valScalar: uint64(s),
	}
}

func NewStaticKind(k Kind) Static {
	return Static{
		Type:      TypeKind,
		valScalar: uint64(k),
	}
}

func NewStaticIntArray(i []int) Static {
	if i == nil {
		return Static{Type: TypeIntArray}
	}
	if len(i) == 0 {
		return Static{Type: TypeIntArray, valBytes: []byte{}}
	}

	numBytes := uintptr(len(i)) * unsafe.Sizeof(i[0])
	return Static{
		Type:     TypeIntArray,
		valBytes: unsafe.Slice((*byte)(unsafe.Pointer(&i[0])), numBytes),
	}
}

func NewStaticFloatArray(f []float64) Static {
	if f == nil {
		return Static{Type: TypeFloatArray}
	}
	if len(f) == 0 {
		return Static{Type: TypeFloatArray, valBytes: []byte{}}
	}

	numBytes := uintptr(len(f)) * unsafe.Sizeof(f[0])
	return Static{
		Type:     TypeFloatArray,
		valBytes: unsafe.Slice((*byte)(unsafe.Pointer(&f[0])), numBytes),
	}
}

func NewStaticStringArray(s []string) Static {
	if s == nil {
		return Static{Type: TypeStringArray}
	}
	if len(s) == 0 {
		return Static{Type: TypeStringArray, valStrings: []string{}}
	}

	return Static{
		Type:       TypeStringArray,
		valStrings: s,
	}
}

func NewStaticBooleanArray(b []bool) Static {
	if b == nil {
		return Static{Type: TypeBooleanArray}
	}
	if len(b) == 0 {
		return Static{Type: TypeBooleanArray, valBytes: []byte{}}
	}

	return Static{
		Type:     TypeBooleanArray,
		valBytes: unsafe.Slice((*byte)(unsafe.Pointer(&b[0])), len(b)),
	}
}

var seedBytes = []byte{204, 38, 247, 160, 15, 37, 67, 77}

func (s Static) MapKey() StaticMapKey {
	switch s.Type {
	case TypeNil:
		return StaticMapKey{typ: TypeNil}
	case TypeString:
		var str string
		if len(s.valBytes) > 0 {
			str = unsafe.String(unsafe.SliceData(s.valBytes), len(s.valBytes))
		}
		return StaticMapKey{typ: s.Type, str: str}
	case TypeIntArray, TypeFloatArray, TypeBooleanArray:
		if len(s.valBytes) == 0 {
			return StaticMapKey{typ: s.Type}
		}

		h := fnv.New64a()
		_, _ = h.Write(s.valBytes)
		return StaticMapKey{typ: s.Type, code: h.Sum64()}
	case TypeStringArray:
		if len(s.valStrings) == 0 {
			return StaticMapKey{typ: s.Type}
		}

		h := fnv.New64a()
		_, _ = h.Write(seedBytes) // avoid collisions with values like []string{""}
		for _, str := range s.valStrings {
			_, _ = h.Write(unsafe.Slice(unsafe.StringData(str), len(str)))
		}
		return StaticMapKey{typ: s.Type, code: h.Sum64()}
	default:
		return StaticMapKey{typ: s.Type, code: s.valScalar}
	}
}

func (s Static) Equals(o *Static) bool {
	switch s.Type {
	case TypeInt, TypeDuration:
		switch o.Type {
		case TypeInt, TypeDuration:
			return s.valScalar == o.valScalar
		case TypeStatus:
			// only int can be compared to status
			return s.Type == TypeInt && s.valScalar == o.valScalar
		case TypeFloat:
			of := math.Float64frombits(o.valScalar)
			return s.Float() == of
		default:
			return false
		}
	case TypeStatus:
		switch o.Type {
		case TypeInt, TypeStatus:
			return s.valScalar == o.valScalar
		default:
			return false
		}
	case TypeFloat:
		sf := math.Float64frombits(s.valScalar)
		return sf == o.Float()
	case TypeKind, TypeBoolean:
		return s.Type == o.Type && s.valScalar == o.valScalar
	case TypeString, TypeIntArray, TypeFloatArray, TypeBooleanArray:
		return s.Type == o.Type && bytes.Equal(s.valBytes, o.valBytes)
	case TypeStringArray:
		return s.Type == o.Type && slices.Equal(s.valStrings, o.valStrings)
	case TypeNil:
		return o.Type == TypeNil
	default:
		// should not be reached
		return false
	}
}

func (s Static) StrictEquals(o *Static) bool {
	if s.Type != o.Type {
		return false
	}

	switch s.Type {
	case TypeFloat:
		sf := math.Float64frombits(s.valScalar)
		of := math.Float64frombits(o.valScalar)
		return sf == of
	case TypeString, TypeIntArray, TypeFloatArray, TypeBooleanArray:
		return bytes.Equal(s.valBytes, o.valBytes)
	case TypeStringArray:
		return slices.Equal(s.valStrings, o.valStrings)
	case TypeNil:
		return true
	default:
		return s.valScalar == o.valScalar
	}
}

func (s Static) compare(o *Static) int {
	if s.Type != o.Type {
		if s.isNumeric() && o.isNumeric() {
			return cmp.Compare(s.Float(), o.Float())
		}
		return cmp.Compare(s.Type, o.Type)
	}

	switch s.Type {
	case TypeString, TypeBooleanArray:
		return bytes.Compare(s.valBytes, o.valBytes)
	case TypeIntArray:
		sa, _ := s.IntArray()
		oa, _ := o.IntArray()
		return slices.Compare(sa, oa)
	case TypeFloatArray:
		sa, _ := s.FloatArray()
		oa, _ := o.FloatArray()
		return slices.Compare(sa, oa)
	case TypeStringArray:
		return slices.Compare(s.valStrings, o.valStrings)
	case TypeNil:
		return 0
	default:
		return cmp.Compare(int64(s.valScalar), int64(o.valScalar))
	}
}

func (s Static) Int() (int, bool) {
	if s.Type != TypeInt {
		return 0, false
	}
	return int(s.valScalar), true
}

func (s Static) Float() float64 {
	switch s.Type {
	case TypeFloat:
		return math.Float64frombits(s.valScalar)
	case TypeInt:
		return float64(int(s.valScalar))
	case TypeDuration:
		return float64(int64(s.valScalar))
	default:
		return math.NaN()
	}
}

func (s Static) Bool() (bool, bool) {
	if s.Type != TypeBoolean {
		return false, false
	}
	return s.valScalar != 0, true
}

func (s Static) Duration() (time.Duration, bool) {
	if s.Type != TypeDuration {
		return 0, false
	}
	return time.Duration(s.valScalar), true
}

func (s Static) Status() (Status, bool) {
	if s.Type != TypeStatus {
		return 0, false
	}
	return Status(s.valScalar), true
}

func (s Static) Kind() (Kind, bool) {
	if s.Type != TypeKind {
		return 0, false
	}
	return Kind(s.valScalar), true
}

func (s Static) IntArray() ([]int, bool) {
	if s.Type != TypeIntArray {
		return nil, false
	}

	if s.valBytes == nil {
		return nil, true
	}
	if len(s.valBytes) == 0 {
		return []int{}, true
	}
	numInts := uintptr(len(s.valBytes)) / unsafe.Sizeof(int(0))
	return unsafe.Slice((*int)(unsafe.Pointer(&s.valBytes[0])), numInts), true
}

func (s Static) FloatArray() ([]float64, bool) {
	if s.Type != TypeFloatArray {
		return nil, false
	}

	if s.valBytes == nil {
		return nil, true
	}
	if len(s.valBytes) == 0 {
		return []float64{}, true
	}
	numFloats := uintptr(len(s.valBytes)) / unsafe.Sizeof(float64(0))
	return unsafe.Slice((*float64)(unsafe.Pointer(&s.valBytes[0])), numFloats), true
}

func (s Static) StringArray() ([]string, bool) {
	if s.Type != TypeStringArray {
		return nil, false
	}

	return s.valStrings, true
}

func (s Static) BooleanArray() ([]bool, bool) {
	if s.Type != TypeBooleanArray {
		return nil, false
	}

	if s.valBytes == nil {
		return nil, true
	}
	if len(s.valBytes) == 0 {
		return []bool{}, true
	}
	return unsafe.Slice((*bool)(unsafe.Pointer(&s.valBytes[0])), len(s.valBytes)), true
}

func (s Static) isNumeric() bool {
	return s.Type.isNumeric()
}

func (s *Static) sumInto(o *Static) {
	if s.Type != o.Type {
		return
	}
	switch s.Type {
	case TypeInt:
		s.valScalar = uint64(int(s.valScalar) + int(o.valScalar))
	case TypeDuration:
		s.valScalar = uint64(time.Duration(s.valScalar) + time.Duration(o.valScalar))
	case TypeFloat:
		sf := math.Float64frombits(s.valScalar)
		of := math.Float64frombits(o.valScalar)
		s.valScalar = math.Float64bits(sf + of)
	}
}

func (s Static) divideBy(f float64) Static {
	switch s.Type {
	case TypeInt:
		return NewStaticFloat(float64(s.valScalar) / f) // there's no integer division in traceql
	case TypeDuration:
		return NewStaticDuration(time.Duration(s.valScalar) / time.Duration(f))
	case TypeFloat:
		sf := math.Float64frombits(s.valScalar)
		return NewStaticFloat(sf / f)
	}
	return s
}

func (Static) referencesSpan() bool {
	return false
}

func (s Static) impliedType() StaticType {
	return s.Type
}

// nolint: revive
func (Static) __fieldExpression() {}

// nolint: revive
func (Static) __scalarExpression() {}

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

	case metricsAggregateHistogramOverTime, metricsAggregateQuantileOverTime:
		// Histograms and quantiles are implemented as count_over_time() by(2^log2(attr)) for now
		// I.e. a duration of 500ms will be in __bucket==0.512s
		innerAgg = func() VectorAggregator { return NewCountOverTimeAggregator() }
		byFuncLabel = internalLabelBucket
		switch a.attr {
		case IntrinsicDurationAttribute:
			// Optimal implementation for duration attribute
			byFunc = a.bucketizeSpanDuration
		default:
			// Basic implementation for all other attributes
			byFunc = a.bucketizeAttribute
		}
	}

	a.agg = NewGroupingAggregator(a.op.String(), func() RangeAggregator {
		return NewStepAggregator(q.Start, q.End, q.Step, innerAgg)
	}, a.by, byFunc, byFuncLabel)
}

func (a *MetricsAggregate) bucketizeSpanDuration(s Span) (Static, bool) {
	d := s.DurationNanos()
	if d < 2 {
		return NewStaticNil(), false
	}
	// Bucket is in seconds
	return NewStaticFloat(Log2Bucketize(d) / float64(time.Second)), true
}

func (a *MetricsAggregate) bucketizeAttribute(s Span) (Static, bool) {
	v, ok := s.AttributeFor(a.attr)
	if !ok {
		return NewStaticNil(), false
	}

	switch v.Type {
	case TypeInt:
		n, _ := v.Int()
		if n < 2 {
			return NewStaticNil(), false
		}
		// Bucket is the value rounded up to the nearest power of 2
		return NewStaticFloat(Log2Bucketize(uint64(n))), true
	case TypeDuration:
		d, _ := v.Duration()
		n := d.Nanoseconds()
		if n < 2 {
			return NewStaticNil(), false
		}
		// Bucket is log2(nanos) converted to float seconds
		return NewStaticFloat(Log2Bucketize(uint64(n)) / float64(time.Second)), true
	default:
		// TODO(mdisibio) - Add support for floats, we need to map them into buckets.
		// Because of the range of floats, we need a native histogram approach.
		return NewStaticNil(), false
	}
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
