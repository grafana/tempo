package types

import (
	"math"
	"strings"

	"github.com/google/go-jsonnet/ast"
)

// Technically on 64-bit system, if someone really tries, maybe they can
// go over that and get strange errors. At this point I don't care.
const maxPossibleArity = math.MaxInt32

type arrayDesc struct {
	// TODO(sbarzowski) Explicit size – if known. It will help with array plus
	// and it will allow catching some out of bounds errors.

	furtherContain []placeholderID

	elementContains [][]placeholderID
}

func (a *arrayDesc) widen(other *arrayDesc) {
	if other == nil {
		return
	}
	for i := range other.elementContains {
		if len(a.elementContains) <= i {
			a.elementContains = append(a.elementContains, copyPlaceholders(a.furtherContain))
		}
		a.elementContains[i] = append(a.elementContains[i], other.elementContains[i]...)
	}
	for i := len(other.elementContains); i < len(a.elementContains); i++ {
		a.elementContains[i] = append(a.elementContains[i], other.furtherContain...)
	}
	a.furtherContain = append(a.furtherContain, other.furtherContain...)
}

func (a *arrayDesc) normalize() {
	for index, ps := range a.elementContains {
		a.elementContains[index] = normalizePlaceholders(ps)
	}
	a.furtherContain = normalizePlaceholders(a.furtherContain)
}

type objectDesc struct {
	unknownContain []placeholderID
	fieldContains  map[string][]placeholderID
	allFieldsKnown bool
}

func (o *objectDesc) widen(other *objectDesc) {
	if other == nil {
		return
	}
	o.unknownContain = append(o.unknownContain, other.unknownContain...)
	for name, placeholders := range other.fieldContains {
		o.fieldContains[name] = append(o.fieldContains[name], placeholders...)
	}
	if !other.allFieldsKnown {
		for name, placeholders := range o.fieldContains {
			if _, present := other.fieldContains[name]; !present {
				o.fieldContains[name] = append(placeholders, other.unknownContain...)
			}
		}
		o.allFieldsKnown = false
	}
}

func (o *objectDesc) normalize() {
	o.unknownContain = normalizePlaceholders(o.unknownContain)
	for f, ps := range o.fieldContains {
		o.fieldContains[f] = normalizePlaceholders(ps)
	}
}

type functionDesc struct {
	resultContains []placeholderID

	// TODO(sbarzowski) instead of keeping "real" parameters here,
	// maybe keep only what we care about in the linter desc
	// (names and required-or-not).
	params []ast.Parameter

	minArity, maxArity int
}

func sameParameters(a, b []ast.Parameter) bool {
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].Name != b[i].Name {
			return false
		}
		// We only check that either both are optional or both are the required
		// We don't care about the specific default arg, because we assume nothing
		// about the argument value anyway.
		if (a[i].DefaultArg != nil) != (b[i].DefaultArg != nil) {
			return false
		}
	}

	return true
}

func (f *functionDesc) widen(other *functionDesc) {
	if other == nil {
		return
	}

	if other.minArity < f.minArity {
		f.minArity = other.minArity
	}
	if other.maxArity > f.maxArity {
		f.maxArity = other.maxArity
	}

	if !sameParameters(f.params, other.params) {
		f.params = nil
	}

	f.resultContains = append(f.resultContains, other.resultContains...)
}

func (f *functionDesc) normalize() {
	f.resultContains = normalizePlaceholders(f.resultContains)
}

// TypeDesc is a representation of a type.
// This is (way) richer than the basic Jsonnet type system
// with seven types.
type TypeDesc struct {
	Bool         bool
	Number       bool
	String       bool
	Null         bool
	FunctionDesc *functionDesc
	ObjectDesc   *objectDesc
	ArrayDesc    *arrayDesc
}

// Any returns whether all values are allowed (i.e. we know nothing about it).
func (t *TypeDesc) Any() bool {
	return t.Bool && t.Number && t.String && t.Null && t.AnyFunction() && t.AnyObject() && t.AnyArray()
}

// Void returns whether the type is empty (no values are possible).
func (t *TypeDesc) Void() bool {
	return !t.Bool && !t.Number && !t.String && !t.Null && !t.Function() && !t.Object() && !t.Array()
}

// Function returns whether the types contains a function.
func (t *TypeDesc) Function() bool {
	return t.FunctionDesc != nil
}

// AnyFunction returns whether the types contain all functions.
func (t *TypeDesc) AnyFunction() bool {
	if !t.Function() || t.FunctionDesc.maxArity < maxPossibleArity || t.FunctionDesc.minArity > 0 || t.FunctionDesc.params != nil {
		return false
	}
	for _, elemType := range t.FunctionDesc.resultContains {
		if elemType == anyType {
			return true
		}
	}
	return false
}

// Object returns whether the types contains an object.
func (t *TypeDesc) Object() bool {
	return t.ObjectDesc != nil
}

// AnyObject returns whether the type contains all objects.
func (t *TypeDesc) AnyObject() bool {
	if !t.Object() || t.ObjectDesc.allFieldsKnown {
		return false
	}
	for _, elemType := range t.ObjectDesc.unknownContain {
		if elemType == anyType {
			return true
		}
	}
	return false
}

// Array returns whether the types contains an array.
func (t *TypeDesc) Array() bool {
	return t.ArrayDesc != nil
}

// AnyArray returns whether the types contain all arrays.
func (t *TypeDesc) AnyArray() bool {
	if !t.Array() {
		return false
	}
	for _, elem := range t.ArrayDesc.elementContains {
		for _, elemType := range elem {
			if elemType == anyType {
				break
			}
			return false
		}
	}
	for _, elemType := range t.ArrayDesc.furtherContain {
		if elemType == anyType {
			break
		}
		return false
	}
	return true
}

func voidTypeDesc() TypeDesc {
	return TypeDesc{}
}

// Describe provides incomplete, but human-readable
// representation of a type.
func Describe(t *TypeDesc) string {
	if t.Any() {
		return "any"
	}
	if t.Void() {
		return "void"
	}
	parts := []string{}
	if t.Bool {
		parts = append(parts, "a bool")
	}
	if t.Number {
		parts = append(parts, "a number")
	}
	if t.String {
		parts = append(parts, "a string")
	}
	if t.Null {
		parts = append(parts, "a null")
	}
	if t.Function() {
		parts = append(parts, "a function")
	}
	if t.Object() {
		parts = append(parts, "an object")
	}
	if t.Array() {
		parts = append(parts, "an array")
	}
	return strings.Join(parts, " or ")
}

func (t *TypeDesc) widen(b *TypeDesc) {
	t.Bool = t.Bool || b.Bool
	t.Number = t.Number || b.Number
	t.String = t.String || b.String
	t.Null = t.Null || b.Null

	if t.FunctionDesc != nil {
		t.FunctionDesc.widen(b.FunctionDesc)
	} else if t.FunctionDesc == nil && b.FunctionDesc != nil {
		copy := *b.FunctionDesc
		t.FunctionDesc = &copy
	}

	if t.ObjectDesc != nil {
		t.ObjectDesc.widen(b.ObjectDesc)
	} else if t.ObjectDesc == nil && b.ObjectDesc != nil {
		copy := *b.ObjectDesc
		t.ObjectDesc = &copy
	}

	if t.ArrayDesc != nil {
		t.ArrayDesc.widen(b.ArrayDesc)
	} else if t.ArrayDesc == nil && b.ArrayDesc != nil {
		copy := *b.ArrayDesc
		t.ArrayDesc = &copy
	}
}

func (t *TypeDesc) normalize() {
	if t.ArrayDesc != nil {
		t.ArrayDesc.normalize()
	}
	if t.FunctionDesc != nil {
		t.FunctionDesc.normalize()
	}
	if t.ObjectDesc != nil {
		t.ObjectDesc.normalize()
	}
}

type indexSpec struct {
	indexType indexType

	indexed placeholderID

	// Known string with which a container is indexed. E.g. "bar" in foo.bar.
	knownStringIndex string
	// Known int with which a container is indexed, e.g. 3 in foo[3].
	knownIntIndex int
}

type indexType int

const (
	genericIndex     = iota
	knownIntIndex    = iota
	knownStringIndex = iota
	functionIndex    = iota
)

func unknownIndexSpec(indexed placeholderID) *indexSpec {
	return &indexSpec{
		indexType:        genericIndex,
		indexed:          indexed,
		knownStringIndex: "",
	}
}

func knownObjectIndex(indexed placeholderID, index string) *indexSpec {
	return &indexSpec{
		indexType:        knownStringIndex,
		indexed:          indexed,
		knownStringIndex: index}
}

func functionCallIndex(function placeholderID) *indexSpec {
	return &indexSpec{
		indexType: functionIndex,
		indexed:   function,
	}
}

func arrayIndex(indexed placeholderID, index int) *indexSpec {
	return &indexSpec{
		indexType:     knownIntIndex,
		indexed:       indexed,
		knownIntIndex: index,
	}
}

type elementDesc struct {
	genericIndex     placeholderID
	knownStringIndex map[string]placeholderID
	knownIntIndex    []placeholderID
	callIndex        placeholderID
}

type builtinOpResult struct {
	contained []placeholderID
	concrete  TypeDesc
}

// builtinOpFunc represents an operation requring custom type calculations
// such as operator+. This operation can often take advantage of having some types
// already concretized. So all arguments are passed as placeholders, but available
// upper bounds are passed too as concreteArgs. For unavailable ones nil is put there.
type builtinOpFunc func(concreteArgs []*TypeDesc, pArgs []placeholderID) builtinOpResult

type builtinOpDesc struct {
	args []placeholderID
	f    builtinOpFunc
}

func (b *builtinOpDesc) withUnknown() builtinOpResult {
	var concrete []*TypeDesc
	for range b.args {
		concrete = append(concrete, nil)
	}
	return b.f(concrete, b.args)
}

func plusObjects(left, right *objectDesc) *objectDesc {
	if left == nil || right == nil {
		return nil
	}
	var res objectDesc
	res.unknownContain = append(res.unknownContain, left.unknownContain...)
	res.unknownContain = append(res.unknownContain, right.unknownContain...)
	res.fieldContains = make(map[string][]placeholderID)
	for k, v := range left.fieldContains {
		res.fieldContains[k] = copyPlaceholders(v)
		if !right.allFieldsKnown {
			if _, present := right.fieldContains[k]; !present {
				res.fieldContains[k] = append(v, right.unknownContain...)
			}
		}
	}
	// From the external point of view, new fields simply replace the old ones
	for k, v := range right.fieldContains {
		res.fieldContains[k] = copyPlaceholders(v)
		if !left.allFieldsKnown {
			if _, present := left.fieldContains[k]; !present {
				res.fieldContains[k] = append(v, left.unknownContain...)
			}
		}
	}
	res.allFieldsKnown = left.allFieldsKnown && right.allFieldsKnown
	return &res
}

func plusArrays(left, right *arrayDesc) *arrayDesc {
	if left == nil || right == nil {
		return nil
	}

	var res arrayDesc

	// Known from the left array
	for _, placeholders := range left.elementContains {
		res.elementContains = append(res.elementContains, copyPlaceholders(placeholders))
	}

	// Further elements from the left array
	res.furtherContain = append(res.furtherContain, left.furtherContain...)

	// Known elements from the right array
	for _, v := range right.elementContains {
		// Since we do not know the size of the left array, we cannot do much else
		res.furtherContain = append(res.furtherContain, v...)
	}

	// Unknown elements from the right array
	res.furtherContain = append(res.furtherContain, right.furtherContain...)
	return &res
}

func builtinPlus(concreteArgs []*TypeDesc, pArgs []placeholderID) builtinOpResult {
	if concreteArgs[0] != nil && concreteArgs[1] != nil {
		// We have concrete arguments available - we can provide a concrete result.
		left := concreteArgs[0]
		right := concreteArgs[1]
		return builtinOpResult{
			concrete: TypeDesc{
				Bool:         left.Bool && right.Bool,
				Number:       left.Number && right.Number,
				String:       left.String || right.String,
				Null:         false,
				FunctionDesc: nil,
				ObjectDesc:   plusObjects(left.ObjectDesc, right.ObjectDesc),
				ArrayDesc:    plusArrays(left.ArrayDesc, right.ArrayDesc),
			},
			contained: pArgs,
		}
	}
	// We do now know what the arguments are yet, so we cannot provide any concrete
	// result without more context.
	return builtinOpResult{
		contained: pArgs,
	}
}
