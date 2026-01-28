/*
Copyright 2017 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jsonnet

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-jsonnet/ast"
)

func builtinPlus(i *interpreter, x, y value) (value, error) {
	// TODO(sbarzowski) perhaps a more elegant way to dispatch
	switch right := y.(type) {
	case valueString:
		left, err := builtinToString(i, x)
		if err != nil {
			return nil, err
		}
		return concatStrings(left.(valueString), right), nil

	}
	switch left := x.(type) {
	case *valueNumber:
		right, err := i.getNumber(y)
		if err != nil {
			return nil, err
		}
		return makeDoubleCheck(i, left.value+right.value)
	case valueString:
		right, err := builtinToString(i, y)
		if err != nil {
			return nil, err
		}
		return concatStrings(left, right.(valueString)), nil
	case *valueObject:
		switch right := y.(type) {
		case *valueObject:
			return makeValueExtendedObject(left, right), nil
		default:
			return nil, i.typeErrorSpecific(y, &valueObject{})
		}

	case *valueArray:
		right, err := i.getArray(y)
		if err != nil {
			return nil, err
		}
		return concatArrays(left, right), nil
	default:
		return nil, i.typeErrorGeneral(x)
	}
}

func builtinMinus(i *interpreter, xv, yv value) (value, error) {
	x, err := i.getNumber(xv)
	if err != nil {
		return nil, err
	}
	y, err := i.getNumber(yv)
	if err != nil {
		return nil, err
	}
	return makeDoubleCheck(i, x.value-y.value)
}

func builtinMult(i *interpreter, xv, yv value) (value, error) {
	x, err := i.getNumber(xv)
	if err != nil {
		return nil, err
	}
	y, err := i.getNumber(yv)
	if err != nil {
		return nil, err
	}
	return makeDoubleCheck(i, x.value*y.value)
}

func builtinDiv(i *interpreter, xv, yv value) (value, error) {
	x, err := i.getNumber(xv)
	if err != nil {
		return nil, err
	}
	y, err := i.getNumber(yv)
	if err != nil {
		return nil, err
	}
	if y.value == 0 {
		return nil, i.Error("Division by zero.")
	}
	return makeDoubleCheck(i, x.value/y.value)
}

func builtinModulo(i *interpreter, xv, yv value) (value, error) {
	x, err := i.getNumber(xv)
	if err != nil {
		return nil, err
	}
	y, err := i.getNumber(yv)
	if err != nil {
		return nil, err
	}
	if y.value == 0 {
		return nil, i.Error("Division by zero.")
	}
	return makeDoubleCheck(i, math.Mod(x.value, y.value))
}

func valueCmp(i *interpreter, x, y value) (int, error) {
	switch left := x.(type) {
	case *valueNumber:
		right, err := i.getNumber(y)
		if err != nil {
			return 0, err
		}
		return float64Cmp(left.value, right.value), nil
	case valueString:
		right, err := i.getString(y)
		if err != nil {
			return 0, err
		}
		return stringCmp(left, right), nil
	case *valueArray:
		right, err := i.getArray(y)
		if err != nil {
			return 0, err
		}
		return arrayCmp(i, left, right)
	default:
		return 0, i.typeErrorGeneral(x)
	}
}

func arrayCmp(i *interpreter, x, y *valueArray) (int, error) {
	for index := 0; index < minInt(x.length(), y.length()); index++ {
		left, err := x.index(i, index)
		if err != nil {
			return 0, err
		}
		right, err := y.index(i, index)
		if err != nil {
			return 0, err
		}
		cmp, err := valueCmp(i, left, right)
		if err != nil {
			return 0, err
		}
		if cmp != 0 {
			return cmp, nil
		}
	}
	return intCmp(x.length(), y.length()), nil
}

func builtinLess(i *interpreter, x, y value) (value, error) {
	r, err := valueCmp(i, x, y)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(r == -1), nil
}

func builtinGreater(i *interpreter, x, y value) (value, error) {
	r, err := valueCmp(i, x, y)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(r == 1), nil
}

func builtinGreaterEq(i *interpreter, x, y value) (value, error) {
	r, err := valueCmp(i, x, y)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(r >= 0), nil
}

func builtinLessEq(i *interpreter, x, y value) (value, error) {
	r, err := valueCmp(i, x, y)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(r <= 0), nil
}

func builtinLength(i *interpreter, x value) (value, error) {
	var num int
	switch x := x.(type) {
	case *valueObject:
		num = len(objectFields(x, withoutHidden))
	case *valueArray:
		num = len(x.elements)
	case valueString:
		num = x.length()
	case *valueFunction:
		for _, param := range x.parameters() {
			if param.defaultArg == nil {
				num++
			}
		}
	default:
		return nil, i.typeErrorGeneral(x)
	}
	return makeValueNumber(float64(num)), nil
}

func builtinToString(i *interpreter, x value) (value, error) {
	switch x := x.(type) {
	case valueString:
		return x, nil
	}
	var buf bytes.Buffer
	err := i.manifestAndSerializeJSON(&buf, x, false, "")
	if err != nil {
		return nil, err
	}
	return makeValueString(buf.String()), nil
}

func builtinTrace(i *interpreter, x value, y value) (value, error) {
	xStr, err := i.getString(x)
	if err != nil {
		return nil, err
	}
	trace := i.stack.currentTrace
	filename := trace.loc.File.DiagnosticFileName
	line := trace.loc.Begin.Line
	fmt.Fprintf(
		i.traceOut, "TRACE: %s:%d %s\n", filename, line, xStr.getGoString())
	return y, nil
}

// astMakeArrayElement wraps the function argument of std.makeArray so that
// it can be embedded in cachedThunk without needing to execute it ahead of
// time.  It is equivalent to `local i = 42; func(i)`.  It therefore has no
// free variables and needs only an empty environment to execute.
type astMakeArrayElement struct {
	function *valueFunction
	ast.NodeBase
	index int
}

func builtinMakeArray(i *interpreter, szv, funcv value) (value, error) {
	sz, err := i.getInt(szv)
	if err != nil {
		return nil, err
	}
	fun, err := i.getFunction(funcv)
	if err != nil {
		return nil, err
	}
	var elems []*cachedThunk
	for i := 0; i < sz; i++ {
		elem := &cachedThunk{
			env: &environment{},
			body: &astMakeArrayElement{
				NodeBase: ast.NodeBase{},
				function: fun,
				index:    i,
			},
		}
		elems = append(elems, elem)
	}
	return makeValueArray(elems), nil
}

func builtinFlatMap(i *interpreter, funcv, arrv value) (value, error) {
	fun, err := i.getFunction(funcv)
	if err != nil {
		return nil, err
	}
	switch arrv := arrv.(type) {
	case *valueArray:
		num := arrv.length()
		// Start with capacity of the original array.
		// This may spare us a few reallocations.
		// TODO(sbarzowski) verify that it actually helps
		elems := make([]*cachedThunk, 0, num)
		for counter := 0; counter < num; counter++ {
			returnedValue, err := fun.call(i, args(arrv.elements[counter]))
			if err != nil {
				return nil, err
			}
			returned, err := i.getArray(returnedValue)
			if err != nil {
				return nil, err
			}
			elems = append(elems, returned.elements...)
		}
		return makeValueArray(elems), nil
	case valueString:
		var str strings.Builder
		for _, elem := range arrv.getRunes() {
			returnedValue, err := fun.call(i, args(readyThunk(makeValueString(string(elem)))))
			if err != nil {
				return nil, err
			}
			returned, err := i.getString(returnedValue)
			if err != nil {
				return nil, err
			}
			str.WriteString(returned.getGoString())
		}
		return makeValueString(str.String()), nil
	default:
		return nil, i.Error("std.flatMap second param must be array / string, got " + arrv.getType().name)
	}
}

func joinArrays(i *interpreter, sep *valueArray, arr *valueArray) (value, error) {
	result := make([]*cachedThunk, 0, arr.length())
	first := true
	for _, elem := range arr.elements {
		elemValue, err := i.evaluatePV(elem)
		if err != nil {
			return nil, err
		}
		switch v := elemValue.(type) {
		case *valueNull:
			continue
		case *valueArray:
			if !first {
				result = append(result, sep.elements...)
			}
			result = append(result, v.elements...)
		default:
			return nil, i.typeErrorSpecific(elemValue, &valueArray{})
		}
		first = false

	}
	return makeValueArray(result), nil
}

func joinStrings(i *interpreter, sep valueString, arr *valueArray) (value, error) {
	result := make([]rune, 0, arr.length())
	first := true
	for _, elem := range arr.elements {
		elemValue, err := i.evaluatePV(elem)
		if err != nil {
			return nil, err
		}
		switch v := elemValue.(type) {
		case *valueNull:
			continue
		case valueString:
			if !first {
				result = append(result, sep.getRunes()...)
			}
			result = append(result, v.getRunes()...)
		default:
			return nil, i.typeErrorSpecific(elemValue, emptyString())
		}
		first = false
	}
	return makeStringFromRunes(result), nil
}

func builtinJoin(i *interpreter, sep, arrv value) (value, error) {
	arr, err := i.getArray(arrv)
	if err != nil {
		return nil, err
	}
	switch sep := sep.(type) {
	case valueString:
		return joinStrings(i, sep, arr)
	case *valueArray:
		return joinArrays(i, sep, arr)
	default:
		return nil, i.Error("join first parameter should be string or array, got " + sep.getType().name)
	}
}

func builtinFoldl(i *interpreter, funcv, arrv, initv value) (value, error) {
	fun, err := i.getFunction(funcv)
	if err != nil {
		return nil, err
	}
	var numElements int
	var elements []*cachedThunk
	switch arrType := arrv.(type) {
	case valueString:
		for _, item := range arrType.getRunes() {
			elements = append(elements, readyThunk(makeStringFromRunes([]rune{item})))
		}
		numElements = len(elements)
	case *valueArray:
		numElements = arrType.length()
		elements = arrType.elements
	default:
		return nil, i.Error("foldl second parameter should be string or array, got " + arrType.getType().name)
	}

	accValue := initv
	for counter := 0; counter < numElements; counter++ {
		accValue, err = fun.call(i, args([]*cachedThunk{readyThunk(accValue), elements[counter]}...))
		if err != nil {
			return nil, err
		}
	}

	return accValue, nil
}

func builtinFoldr(i *interpreter, funcv, arrv, initv value) (value, error) {
	fun, err := i.getFunction(funcv)
	if err != nil {
		return nil, err
	}
	var numElements int
	var elements []*cachedThunk
	switch arrType := arrv.(type) {
	case valueString:
		for _, item := range arrType.getRunes() {
			elements = append(elements, readyThunk(makeStringFromRunes([]rune{item})))
		}
		numElements = len(elements)
	case *valueArray:
		numElements = arrType.length()
		elements = arrType.elements
	default:
		return nil, i.Error("foldr second parameter should be string or array, got " + arrType.getType().name)
	}

	accValue := initv
	for counter := numElements - 1; counter >= 0; counter-- {
		accValue, err = fun.call(i, args([]*cachedThunk{elements[counter], readyThunk(accValue)}...))
		if err != nil {
			return nil, err
		}
	}

	return accValue, nil
}

func builtinReverse(i *interpreter, arrv value) (value, error) {
	arr, err := i.getArray(arrv)
	if err != nil {
		return nil, err
	}

	lenArr := len(arr.elements)                   // lenx holds the original array length
	reversedArray := make([]*cachedThunk, lenArr) // creates a slice that refer to a new array of length lenx

	for i := 0; i < lenArr; i++ {
		j := lenArr - (i + 1) // j initially holds (lenx - 1) and decreases to 0 while i initially holds 0 and increase to (lenx - 1)
		reversedArray[i] = arr.elements[j]
	}

	return makeValueArray(reversedArray), nil
}

func builtinFilter(i *interpreter, funcv, arrv value) (value, error) {
	arr, err := i.getArray(arrv)
	if err != nil {
		return nil, err
	}
	fun, err := i.getFunction(funcv)
	if err != nil {
		return nil, err
	}
	num := arr.length()
	// Start with capacity of the original array.
	// This may spare us a few reallocations.
	// TODO(sbarzowski) verify that it actually helps
	elems := make([]*cachedThunk, 0, num)
	for counter := 0; counter < num; counter++ {
		includedValue, err := fun.call(i, args(arr.elements[counter]))
		if err != nil {
			return nil, err
		}
		included, err := i.getBoolean(includedValue)
		if err != nil {
			return nil, err
		}
		if included.value {
			elems = append(elems, arr.elements[counter])
		}
	}
	return makeValueArray(elems), nil
}
func builtinLstripChars(i *interpreter, str, chars value) (value, error) {
	switch strType := str.(type) {
	case valueString:
		if strType.length() > 0 {
			index, err := strType.index(i, 0)
			if err != nil {
				return nil, err
			}
			member, err := rawMember(i, chars, index)
			if err != nil {
				return nil, err
			}
			if member {
				runes := strType.getRunes()
				s := string(runes[1:])
				return builtinLstripChars(i, makeValueString(s), chars)
			} else {
				return str, nil
			}
		}
		return str, nil
	default:
		return nil, i.Error(fmt.Sprintf("Unexpected type %s, expected string", strType.getType().name))
	}
}

func builtinRstripChars(i *interpreter, str, chars value) (value, error) {
	switch strType := str.(type) {
	case valueString:
		if strType.length() > 0 {
			index, err := strType.index(i, strType.length()-1)
			if err != nil {
				return nil, err
			}
			member, err := rawMember(i, chars, index)
			if err != nil {
				return nil, err
			}
			if member {
				runes := strType.getRunes()
				s := string(runes[:len(runes)-1])
				return builtinRstripChars(i, makeValueString(s), chars)
			} else {
				return str, nil
			}
		}
		return str, nil
	default:
		return nil, i.Error(fmt.Sprintf("Unexpected type %s, expected string", strType.getType().name))
	}
}

func builtinStripChars(i *interpreter, str, chars value) (value, error) {
	lstripChars, err := builtinLstripChars(i, str, chars)
	if err != nil {
		return nil, err
	}
	rstripChars, err := builtinRstripChars(i, lstripChars, chars)
	if err != nil {
		return nil, err
	}
	return rstripChars, nil
}

func rawMember(i *interpreter, arrv, value value) (bool, error) {
	switch arrType := arrv.(type) {
	case valueString:
		valString, err := i.getString(value)
		if err != nil {
			return false, err
		}

		arrString, err := i.getString(arrv)
		if err != nil {
			return false, err
		}

		return strings.Contains(arrString.getGoString(), valString.getGoString()), nil
	case *valueArray:
		for _, elem := range arrType.elements {
			cachedThunkValue, err := elem.getValue(i)
			if err != nil {
				return false, err
			}
			equal, err := rawEquals(i, cachedThunkValue, value)
			if err != nil {
				return false, err
			}
			if equal {
				return true, nil
			}
		}
	default:
		return false, i.Error("std.member first argument must be an array or a string")
	}
	return false, nil
}

func builtinMember(i *interpreter, arrv, value value) (value, error) {
	eq, err := rawMember(i, arrv, value)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(eq), nil
}

type sortData struct {
	err    error
	i      *interpreter
	thunks []*cachedThunk
	keys   []value
}

func (d *sortData) Len() int {
	return len(d.thunks)
}

func (d *sortData) Less(i, j int) bool {
	r, err := valueCmp(d.i, d.keys[i], d.keys[j])
	if err != nil {
		d.err = err
		panic("Error while comparing elements")
	}
	return r == -1
}

func (d *sortData) Swap(i, j int) {
	d.thunks[i], d.thunks[j] = d.thunks[j], d.thunks[i]
	d.keys[i], d.keys[j] = d.keys[j], d.keys[i]
}

func (d *sortData) Sort() (err error) {
	defer func() {
		if d.err != nil {
			if r := recover(); r != nil {
				err = d.err
			}
		}
	}()
	sort.Stable(d)
	return
}

func builtinSort(i *interpreter, arguments []value) (value, error) {
	arrv := arguments[0]
	keyFv := arguments[1]

	arr, err := i.getArray(arrv)
	if err != nil {
		return nil, err
	}
	keyF, err := i.getFunction(keyFv)
	if err != nil {
		return nil, err
	}
	num := arr.length()

	data := sortData{i: i, thunks: make([]*cachedThunk, num), keys: make([]value, num)}

	for counter := 0; counter < num; counter++ {
		var err error
		data.thunks[counter] = arr.elements[counter]
		data.keys[counter], err = keyF.call(i, args(arr.elements[counter]))
		if err != nil {
			return nil, err
		}
	}

	err = data.Sort()
	if err != nil {
		return nil, err
	}

	return makeValueArray(data.thunks), nil
}

func builtinRange(i *interpreter, fromv, tov value) (value, error) {
	from, err := i.getInt(fromv)
	if err != nil {
		return nil, err
	}
	to, err := i.getInt(tov)
	if err != nil {
		return nil, err
	}
	elems := make([]*cachedThunk, to-from+1)
	for i := from; i <= to; i++ {
		elems[i-from] = readyThunk(intToValue(i))
	}
	return makeValueArray(elems), nil
}

func builtinNegation(i *interpreter, x value) (value, error) {
	b, err := i.getBoolean(x)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(!b.value), nil
}

func builtinXnor(i *interpreter, xv, yv value) (value, error) {
	p, err := i.getBoolean(xv)
	if err != nil {
		return nil, err
	}
	q, err := i.getBoolean(yv)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(p.value == q.value), nil
}

func builtinXor(i *interpreter, xv, yv value) (value, error) {
	p, err := i.getBoolean(xv)
	if err != nil {
		return nil, err
	}
	q, err := i.getBoolean(yv)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(p.value != q.value), nil
}

func builtinBitNeg(i *interpreter, x value) (value, error) {
	n, err := i.getNumber(x)
	if err != nil {
		return nil, err
	}
	intValue := int64(n.value)
	return int64ToValue(^intValue), nil
}

func builtinIdentity(i *interpreter, x value) (value, error) {
	return x, nil
}

func builtinUnaryPlus(i *interpreter, x value) (value, error) {
	n, err := i.getNumber(x)
	if err != nil {
		return nil, err
	}

	return makeValueNumber(n.value), nil
}

func builtinUnaryMinus(i *interpreter, x value) (value, error) {
	n, err := i.getNumber(x)
	if err != nil {
		return nil, err
	}
	return makeValueNumber(-n.value), nil
}

// TODO(sbarzowski) since we have a builtin implementation of equals it's no longer really
// needed and we should deprecate it eventually
func primitiveEquals(i *interpreter, x, y value) (value, error) {
	if x.getType() != y.getType() {
		return makeValueBoolean(false), nil
	}
	switch left := x.(type) {
	case *valueBoolean:
		right, err := i.getBoolean(y)
		if err != nil {
			return nil, err
		}
		return makeValueBoolean(left.value == right.value), nil
	case *valueNumber:
		right, err := i.getNumber(y)
		if err != nil {
			return nil, err
		}
		return makeValueBoolean(left.value == right.value), nil
	case valueString:
		right, err := i.getString(y)
		if err != nil {
			return nil, err
		}
		return makeValueBoolean(stringEqual(left, right)), nil
	case *valueNull:
		return makeValueBoolean(true), nil
	case *valueFunction:
		return nil, i.Error("Cannot test equality of functions")
	default:
		return nil, i.Error(
			"primitiveEquals operates on primitive types, got " + x.getType().name,
		)
	}
}

func rawEquals(i *interpreter, x, y value) (bool, error) {
	if x.getType() != y.getType() {
		return false, nil
	}
	switch left := x.(type) {
	case *valueBoolean:
		right, err := i.getBoolean(y)
		if err != nil {
			return false, err
		}
		return left.value == right.value, nil
	case *valueNumber:
		right, err := i.getNumber(y)
		if err != nil {
			return false, err
		}
		return left.value == right.value, nil
	case valueString:
		right, err := i.getString(y)
		if err != nil {
			return false, err
		}
		return stringEqual(left, right), nil
	case *valueNull:
		return true, nil
	case *valueArray:
		right, err := i.getArray(y)
		if err != nil {
			return false, err
		}
		if left.length() != right.length() {
			return false, nil
		}
		for j := range left.elements {
			leftElem, err := i.evaluatePV(left.elements[j])
			if err != nil {
				return false, err
			}
			rightElem, err := i.evaluatePV(right.elements[j])
			if err != nil {
				return false, err
			}
			eq, err := rawEquals(i, leftElem, rightElem)
			if err != nil {
				return false, err
			}
			if !eq {
				return false, nil
			}
		}
		return true, nil
	case *valueObject:
		right, err := i.getObject(y)
		if err != nil {
			return false, err
		}
		leftFields := objectFields(left, withoutHidden)
		rightFields := objectFields(right, withoutHidden)
		sort.Strings(leftFields)
		sort.Strings(rightFields)
		if len(leftFields) != len(rightFields) {
			return false, nil
		}
		for i := range leftFields {
			if leftFields[i] != rightFields[i] {
				return false, nil
			}
		}
		for j := range leftFields {
			fieldName := leftFields[j]
			leftField, err := left.index(i, fieldName)
			if err != nil {
				return false, err
			}
			rightField, err := right.index(i, fieldName)
			if err != nil {
				return false, err
			}
			eq, err := rawEquals(i, leftField, rightField)
			if err != nil {
				return false, err
			}
			if !eq {
				return false, nil
			}
		}
		return true, nil
	case *valueFunction:
		return false, i.Error("Cannot test equality of functions")
	}
	panic(fmt.Sprintf("Unhandled case in equals %#+v %#+v", x, y))
}

func builtinEquals(i *interpreter, x, y value) (value, error) {
	eq, err := rawEquals(i, x, y)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(eq), nil
}

func builtinNotEquals(i *interpreter, x, y value) (value, error) {
	eq, err := rawEquals(i, x, y)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(!eq), nil
}

func builtinType(i *interpreter, x value) (value, error) {
	return makeValueString(x.getType().name), nil
}

func builtinMd5(i *interpreter, x value) (value, error) {
	str, err := i.getString(x)
	if err != nil {
		return nil, err
	}
	hash := md5.Sum([]byte(str.getGoString()))
	return makeValueString(hex.EncodeToString(hash[:])), nil
}

func builtinBase64(i *interpreter, input value) (value, error) {
	var byteArr []byte

	var sanityCheck = func(v int) (string, bool) {
		if v < 0 || 255 < v {
			msg := fmt.Sprintf("base64 encountered invalid codepoint value in the array (must be 0 <= X <= 255), got %d", v)
			return msg, false
		}

		return "", true
	}

	switch input.(type) {
	case valueString:
		vStr, err := i.getString(input)
		if err != nil {
			return nil, err
		}

		str := vStr.getGoString()
		for _, r := range str {
			n := int(r)
			msg, ok := sanityCheck(n)
			if !ok {
				return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
			}
		}

		byteArr = []byte(str)
	case *valueArray:
		vArr, err := i.getArray(input)
		if err != nil {
			return nil, err
		}

		for _, cThunk := range vArr.elements {
			cTv, err := cThunk.getValue(i)
			if err != nil {
				return nil, err
			}

			vInt, err := i.getInt(cTv)
			if err != nil {
				msg := fmt.Sprintf("base64 encountered a non-integer value in the array, got %s", cTv.getType().name)
				return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
			}

			msg, ok := sanityCheck(vInt)
			if !ok {
				return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
			}

			byteArr = append(byteArr, byte(vInt))
		}
	default:
		msg := fmt.Sprintf("base64 can only base64 encode strings / arrays of single bytes, got %s", input.getType().name)
		return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
	}

	sEnc := base64.StdEncoding.EncodeToString(byteArr)
	return makeValueString(sEnc), nil
}

func builtinEncodeUTF8(i *interpreter, x value) (value, error) {
	str, err := i.getString(x)
	if err != nil {
		return nil, err
	}
	s := str.getGoString()
	elems := make([]*cachedThunk, 0, len(s)) // it will be longer if characters fall outside of ASCII
	for _, c := range []byte(s) {
		elems = append(elems, readyThunk(makeValueNumber(float64(c))))
	}
	return makeValueArray(elems), nil
}

func builtinDecodeUTF8(i *interpreter, x value) (value, error) {
	arr, err := i.getArray(x)
	if err != nil {
		return nil, err
	}
	bs := make([]byte, len(arr.elements)) // it will be longer if characters fall outside of ASCII
	for pos := range arr.elements {
		v, err := i.evaluateInt(arr.elements[pos])
		if err != nil {
			return nil, err
		}
		if v < 0 || v > 255 {
			return nil, i.Error(fmt.Sprintf("Bytes must be integers in range [0, 255], got %d", v))
		}
		bs[pos] = byte(v)
	}
	return makeValueString(string(bs)), nil
}

// Maximum allowed unicode codepoint
// https://en.wikipedia.org/wiki/Unicode#Architecture_and_terminology
const codepointMax = 0x10FFFF

func builtinChar(i *interpreter, x value) (value, error) {
	n, err := i.getNumber(x)
	if err != nil {
		return nil, err
	}
	if n.value > codepointMax {
		return nil, i.Error(fmt.Sprintf("Invalid unicode codepoint, got %v", n.value))
	} else if n.value < 0 {
		return nil, i.Error(fmt.Sprintf("Codepoints must be >= 0, got %v", n.value))
	}
	return makeValueString(string(rune(n.value))), nil
}

func builtinCodepoint(i *interpreter, x value) (value, error) {
	str, err := i.getString(x)
	if err != nil {
		return nil, err
	}
	if str.length() != 1 {
		return nil, i.Error(fmt.Sprintf("codepoint takes a string of length 1, got length %v", str.length()))
	}
	return makeValueNumber(float64(str.getRunes()[0])), nil
}

func makeDoubleCheck(i *interpreter, x float64) (value, error) {
	if math.IsNaN(x) {
		return nil, i.Error("Not a number")
	}
	if math.IsInf(x, 0) {
		return nil, i.Error("Overflow")
	}
	return makeValueNumber(x), nil
}

func liftNumeric(f func(float64) float64) func(*interpreter, value) (value, error) {
	return func(i *interpreter, x value) (value, error) {
		n, err := i.getNumber(x)
		if err != nil {
			return nil, err
		}
		return makeDoubleCheck(i, f(n.value))
	}
}

var builtinSqrt = liftNumeric(math.Sqrt)
var builtinCeil = liftNumeric(math.Ceil)
var builtinFloor = liftNumeric(math.Floor)
var builtinSin = liftNumeric(math.Sin)
var builtinCos = liftNumeric(math.Cos)
var builtinTan = liftNumeric(math.Tan)
var builtinAsin = liftNumeric(math.Asin)
var builtinAcos = liftNumeric(math.Acos)
var builtinAtan = liftNumeric(math.Atan)
var builtinLog = liftNumeric(math.Log)
var builtinExp = liftNumeric(func(f float64) float64 {
	res := math.Exp(f)
	if res == 0 && f > 0 {
		return math.Inf(1)
	}
	return res
})
var builtinMantissa = liftNumeric(func(f float64) float64 {
	mantissa, _ := math.Frexp(f)
	return mantissa
})
var builtinExponent = liftNumeric(func(f float64) float64 {
	_, exponent := math.Frexp(f)
	return float64(exponent)
})
var builtinRound = liftNumeric(math.Round)

func liftBitwise(f func(int64, int64) int64, positiveRightArg bool) func(*interpreter, value, value) (value, error) {
	return func(i *interpreter, xv, yv value) (value, error) {
		x, err := i.getNumber(xv)
		if err != nil {
			return nil, err
		}
		y, err := i.getNumber(yv)
		if err != nil {
			return nil, err
		}
		if x.value < math.MinInt64 || x.value > math.MaxInt64 {
			msg := fmt.Sprintf("Bitwise operator argument %v outside of range [%v, %v]", x.value, int64(math.MinInt64), int64(math.MaxInt64))
			return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
		}
		if y.value < math.MinInt64 || y.value > math.MaxInt64 {
			msg := fmt.Sprintf("Bitwise operator argument %v outside of range [%v, %v]", y.value, int64(math.MinInt64), int64(math.MaxInt64))
			return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
		}
		if positiveRightArg && y.value < 0 {
			return nil, makeRuntimeError("Shift by negative exponent.", i.getCurrentStackTrace())
		}
		return makeDoubleCheck(i, float64(f(int64(x.value), int64(y.value))))
	}
}

var builtinShiftL = liftBitwise(func(x, y int64) int64 { return x << uint(y%64) }, true)
var builtinShiftR = liftBitwise(func(x, y int64) int64 { return x >> uint(y%64) }, true)
var builtinBitwiseAnd = liftBitwise(func(x, y int64) int64 { return x & y }, false)
var builtinBitwiseOr = liftBitwise(func(x, y int64) int64 { return x | y }, false)
var builtinBitwiseXor = liftBitwise(func(x, y int64) int64 { return x ^ y }, false)

func builtinObjectFieldsEx(i *interpreter, objv, includeHiddenV value) (value, error) {
	obj, err := i.getObject(objv)
	if err != nil {
		return nil, err
	}
	includeHidden, err := i.getBoolean(includeHiddenV)
	if err != nil {
		return nil, err
	}
	fields := objectFields(obj, withHiddenFromBool(includeHidden.value))
	sort.Strings(fields)
	elems := []*cachedThunk{}
	for _, fieldname := range fields {
		elems = append(elems, readyThunk(makeValueString(fieldname)))
	}
	return makeValueArray(elems), nil
}

func builtinObjectHasEx(i *interpreter, objv value, fnamev value, includeHiddenV value) (value, error) {
	obj, err := i.getObject(objv)
	if err != nil {
		return nil, err
	}
	fname, err := i.getString(fnamev)
	if err != nil {
		return nil, err
	}
	includeHidden, err := i.getBoolean(includeHiddenV)
	if err != nil {
		return nil, err
	}
	h := withHiddenFromBool(includeHidden.value)
	hasField := objectHasField(objectBinding(obj), string(fname.getRunes()), h)
	return makeValueBoolean(hasField), nil
}

func builtinPow(i *interpreter, basev value, expv value) (value, error) {
	base, err := i.getNumber(basev)
	if err != nil {
		return nil, err
	}
	exp, err := i.getNumber(expv)
	if err != nil {
		return nil, err
	}
	return makeDoubleCheck(i, math.Pow(base.value, exp.value))
}

func builtinSubstr(i *interpreter, inputStr, inputFrom, inputLen value) (value, error) {
	strV, err := i.getString(inputStr)
	if err != nil {
		msg := fmt.Sprintf("substr first parameter should be a string, got %s", inputStr.getType().name)
		return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
	}

	fromV, err := i.getNumber(inputFrom)
	if err != nil {
		msg := fmt.Sprintf("substr second parameter should be a number, got %s", inputFrom.getType().name)
		return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
	}

	if math.Mod(fromV.value, 1) != 0 {
		msg := fmt.Sprintf("substr second parameter should be an integer, got %f", fromV.value)
		return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
	}

	if fromV.value < 0 {
		msg := fmt.Sprintf("substr second parameter should be greater than zero, got %f", fromV.value)
		return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
	}

	lenV, err := i.getNumber(inputLen)
	if err != nil {
		msg := fmt.Sprintf("substr third parameter should be a number, got %s", inputLen.getType().name)
		return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
	}

	lenInt, err := i.getInt(lenV)

	if err != nil {
		msg := fmt.Sprintf("substr third parameter should be an integer, got %f", lenV.value)
		return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
	}

	if lenInt < 0 {
		msg := fmt.Sprintf("substr third parameter should be greater than zero, got %d", lenInt)
		return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
	}

	fromInt := int(fromV.value)
	strStr := strV.getRunes()

	endIndex := fromInt + lenInt

	if endIndex > len(strStr) {
		endIndex = len(strStr)
	}

	if fromInt > len(strStr) {
		return makeValueString(""), nil
	}
	return makeValueString(string(strStr[fromInt:endIndex])), nil
}

func builtinSplitLimit(i *interpreter, strv, cv, maxSplitsV value) (value, error) {
	str, err := i.getString(strv)
	if err != nil {
		return nil, err
	}
	c, err := i.getString(cv)
	if err != nil {
		return nil, err
	}
	maxSplits, err := i.getInt(maxSplitsV)
	if err != nil {
		return nil, err
	}
	if maxSplits < -1 {
		return nil, i.Error(fmt.Sprintf("std.splitLimit third parameter should be -1 or non-negative, got %v", maxSplits))
	}
	sStr := str.getGoString()
	sC := c.getGoString()
	if len(sC) < 1 {
		return nil, i.Error(fmt.Sprintf("std.splitLimit second parameter should have length 1 or greater, got %v", len(sC)))
	}

	// the convention is slightly different from strings.splitN in Go (the meaning of non-negative values is shifted by one)
	var strs []string
	if maxSplits == -1 {
		strs = strings.SplitN(sStr, sC, -1)
	} else {
		strs = strings.SplitN(sStr, sC, maxSplits+1)
	}
	res := make([]*cachedThunk, len(strs))
	for i := range strs {
		res[i] = readyThunk(makeValueString(strs[i]))
	}

	return makeValueArray(res), nil
}

func builtinStrReplace(i *interpreter, strv, fromv, tov value) (value, error) {
	str, err := i.getString(strv)
	if err != nil {
		return nil, err
	}
	from, err := i.getString(fromv)
	if err != nil {
		return nil, err
	}
	to, err := i.getString(tov)
	if err != nil {
		return nil, err
	}
	sStr := str.getGoString()
	sFrom := from.getGoString()
	sTo := to.getGoString()
	if len(sFrom) == 0 {
		return nil, i.Error("'from' string must not be zero length.")
	}
	return makeValueString(strings.Replace(sStr, sFrom, sTo, -1)), nil
}

func builtinIsEmpty(i *interpreter, strv value) (value, error) {
	str, err := i.getString(strv)
	if err != nil {
		return nil, err
	}
	sStr := str.getGoString()
	return makeValueBoolean(len(sStr) == 0), nil
}

func base64DecodeGoBytes(i *interpreter, str string) ([]byte, error) {
	strLen := len(str)
	if strLen%4 != 0 {
		msg := fmt.Sprintf("input string appears not to be a base64 encoded string. Wrong length found (%d)", strLen)
		return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, i.Error(fmt.Sprintf("failed to decode: %s", err))
	}

	return decodedBytes, nil
}

func builtinBase64DecodeBytes(i *interpreter, input value) (value, error) {
	vStr, err := i.getString(input)
	if err != nil {
		msg := fmt.Sprintf("base64DecodeBytes requires a string, got %s", input.getType().name)
		return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
	}

	decodedBytes, err := base64DecodeGoBytes(i, vStr.getGoString())
	if err != nil {
		return nil, err
	}

	res := make([]*cachedThunk, len(decodedBytes))
	for i := range decodedBytes {
		res[i] = readyThunk(makeValueNumber(float64(int(decodedBytes[i]))))
	}

	return makeValueArray(res), nil
}

func builtinBase64Decode(i *interpreter, input value) (value, error) {
	vStr, err := i.getString(input)
	if err != nil {
		msg := fmt.Sprintf("base64DecodeBytes requires a string, got %s", input.getType().name)
		return nil, makeRuntimeError(msg, i.getCurrentStackTrace())
	}

	decodedBytes, err := base64DecodeGoBytes(i, vStr.getGoString())
	if err != nil {
		return nil, err
	}

	return makeValueString(string(decodedBytes)), nil
}

func builtinUglyObjectFlatMerge(i *interpreter, x value) (value, error) {
	// TODO(sbarzowski) consider keeping comprehensions in AST
	// It will probably be way less hacky, with better error messages and better performance

	objarr, err := i.getArray(x)
	if err != nil {
		return nil, err
	}
	newFields := make(simpleObjectFieldMap)
	for _, elem := range objarr.elements {
		obj, err := i.evaluateObject(elem)
		if err != nil {
			return nil, err
		}

		// starts getting ugly - we mess with object internals
		simpleObj := obj.uncached.(*simpleObject)

		if len(simpleObj.locals) > 0 {
			panic("Locals should have been desugared in object comprehension.")
		}

		// there is only one field, really
		for fieldName, fieldVal := range simpleObj.fields {
			if _, alreadyExists := newFields[fieldName]; alreadyExists {
				return nil, i.Error(duplicateFieldNameErrMsg(fieldName))
			}

			newFields[fieldName] = simpleObjectField{
				hide: fieldVal.hide,
				field: &bindingsUnboundField{
					inner:    fieldVal.field,
					bindings: simpleObj.upValues,
				},
			}
		}
	}

	return makeValueSimpleObject(
		nil,
		newFields,
		[]unboundField{}, // No asserts allowed
		nil,
	), nil
}

func builtinParseJSON(i *interpreter, str value) (value, error) {
	sval, err := i.getString(str)
	if err != nil {
		return nil, err
	}
	s := sval.getGoString()
	var parsedJSON interface{}
	err = json.Unmarshal([]byte(s), &parsedJSON)
	if err != nil {
		return nil, i.Error(fmt.Sprintf("failed to parse JSON: %v", err.Error()))
	}
	return jsonToValue(i, parsedJSON)
}

func builtinParseYAML(i *interpreter, str value) (value, error) {
	sval, err := i.getString(str)
	if err != nil {
		return nil, err
	}
	s := sval.getGoString()

	isYamlStream := strings.Contains(s, "---")

	elems := []interface{}{}
	d := NewYAMLToJSONDecoder(strings.NewReader(s))
	for {
		var elem interface{}
		if err := d.Decode(&elem); err != nil {
			if err == io.EOF {
				break
			}
			return nil, i.Error(fmt.Sprintf("failed to parse YAML: %v", err.Error()))
		}
		elems = append(elems, elem)
	}

	if isYamlStream {
		return jsonToValue(i, elems)
	}
	return jsonToValue(i, elems[0])
}

func jsonEncode(v interface{}) (string, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(v)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(buf.String(), "\n"), nil
}

// tomlIsSection checks whether an object or array is a section - a TOML section is an
// object or an an array has all of its children being objects
func tomlIsSection(i *interpreter, val value) (bool, error) {
	switch v := val.(type) {
	case *valueObject:
		return true, nil
	case *valueArray:
		if v.length() == 0 {
			return false, nil
		}

		for _, thunk := range v.elements {
			thunkValue, err := thunk.getValue(i)
			if err != nil {
				return false, err
			}

			switch thunkValue.(type) {
			case *valueObject:
				// this is expected, return true if all children are objects
			default:
				// return false if at least one child is not an object
				return false, nil
			}
		}

		return true, nil
	default:
		return false, nil
	}
}

// tomlEncodeString encodes a string as quoted TOML string
func tomlEncodeString(s string) string {
	res := "\""

	for _, c := range s {
		// escape specific characters, rendering non-ASCII ones as \uXXXX,
		// appending remaining characters as is
		if c == '"' {
			res = res + "\\\""
		} else if c == '\\' {
			res = res + "\\\\"
		} else if c == '\b' {
			res = res + "\\b"
		} else if c == '\f' {
			res = res + "\\f"
		} else if c == '\n' {
			res = res + "\\n"
		} else if c == '\r' {
			res = res + "\\r"
		} else if c == '\t' {
			res = res + "\\t"
		} else if c < 32 || (c >= 127 && c <= 159) {
			res = res + fmt.Sprintf("\\u%04x", c)
		} else {
			res = res + string(c)
		}
	}

	res = res + "\""

	return res
}

// tomlEncodeKey encodes a key - returning same string if it does not need quoting,
// otherwise return it quoted; returns empty key as ”
func tomlEncodeKey(s string) string {
	bareAllowed := true

	// for empty string, return ''
	if len(s) == 0 {
		return "''"
	}

	for _, c := range s {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			continue
		}

		bareAllowed = false
		break
	}

	if bareAllowed {
		return s
	}
	return tomlEncodeString(s)
}

func tomlAddToPath(path []string, tail string) []string {
	result := make([]string, 0, len(path)+1)
	result = append(result, path...)
	result = append(result, tail)
	return result
}

// tomlRenderValue returns a rendered value as string, with proper indenting
func tomlRenderValue(i *interpreter, val value, sindent string, indexedPath []string, inline bool, cindent string) (string, error) {
	switch v := val.(type) {
	case *valueNull:
		return "", i.Error(fmt.Sprintf("Tried to manifest \"null\" at %v", indexedPath))
	case *valueBoolean:
		return fmt.Sprintf("%t", v.value), nil
	case *valueNumber:
		return unparseNumber(v.value), nil
	case valueString:
		return tomlEncodeString(v.getGoString()), nil
	case *valueFunction:
		return "", i.Error(fmt.Sprintf("Tried to manifest function at %v", indexedPath))
	case *valueArray:
		if len(v.elements) == 0 {
			return "[]", nil
		}

		// initialize indenting and separators based on whether this is added inline or not
		newIndent := cindent + sindent
		separator := "\n"
		if inline {
			newIndent = ""
			separator = " "
		}

		// open the square bracket to start array values
		res := "[" + separator

		// iterate over elents and add their values to result
		for j, thunk := range v.elements {
			thunkValue, err := thunk.getValue(i)
			if err != nil {
				return "", err
			}

			childIndexedPath := tomlAddToPath(indexedPath, strconv.FormatInt(int64(j), 10))

			if j > 0 {
				res = res + "," + separator
			}

			res = res + newIndent
			value, err := tomlRenderValue(i, thunkValue, sindent, childIndexedPath, true, "")
			if err != nil {
				return "", err
			}
			res = res + value
		}

		res = res + separator
		if inline {
			res = res + cindent
		}

		// close the array and return it
		res = res + "]"

		return res, nil
	case *valueObject:
		res := ""

		fields := objectFields(v, withoutHidden)
		sort.Strings(fields)

		// iterate over sorted field keys and render their values
		for j, fieldName := range fields {
			fieldValue, err := v.index(i, fieldName)
			if err != nil {
				return "", err
			}

			childIndexedPath := tomlAddToPath(indexedPath, fieldName)

			value, err := tomlRenderValue(i, fieldValue, sindent, childIndexedPath, true, "")
			if err != nil {
				return "", err
			}

			if j > 0 {
				res = res + ", "
			}
			res = res + tomlEncodeKey(fieldName) + " = " + value
		}

		// wrap fields in an array
		return "{ " + res + " }", nil
	default:
		return "", i.Error(fmt.Sprintf("Unknown object type %v at %v", reflect.TypeOf(v), indexedPath))
	}
}

func tomlRenderTableArray(i *interpreter, v *valueArray, sindent string, path []string, indexedPath []string, cindent string) (string, error) {

	sections := make([]string, 0, len(v.elements))

	// render all elements of an array
	for j, thunk := range v.elements {
		thunkValue, err := thunk.getValue(i)
		if err != nil {
			return "", err
		}

		switch tv := thunkValue.(type) {
		case *valueObject:
			// render the entire path as section name
			section := cindent + "[["

			for i, element := range path {
				if i > 0 {
					section = section + "."
				}
				section = section + tomlEncodeKey(element)
			}

			section = section + "]]"

			// add newline if the table has elements
			if len(objectFields(tv, withoutHidden)) > 0 {
				section = section + "\n"
			}

			childIndexedPath := tomlAddToPath(indexedPath, strconv.FormatInt(int64(j), 10))

			// render the table and add it to result
			table, err := tomlTableInternal(i, tv, sindent, path, childIndexedPath, cindent+sindent)
			if err != nil {
				return "", err
			}
			section = section + table

			sections = append(sections, section)
		default:
			return "", i.Error(fmt.Sprintf("invalid type for section: %v", reflect.TypeOf(thunkValue)))
		}
	}

	// combine all sections
	return strings.Join(sections, "\n\n"), nil
}

func tomlRenderTable(i *interpreter, v *valueObject, sindent string, path []string, indexedPath []string, cindent string) (string, error) {
	res := cindent + "["
	for i, element := range path {
		if i > 0 {
			res = res + "."
		}
		res = res + tomlEncodeKey(element)
	}
	res = res + "]"
	if len(objectFields(v, withoutHidden)) > 0 {
		res = res + "\n"
	}

	table, err := tomlTableInternal(i, v, sindent, path, indexedPath, cindent+sindent)
	if err != nil {
		return "", err
	}
	res = res + table

	return res, nil
}

func tomlTableInternal(i *interpreter, v *valueObject, sindent string, path []string, indexedPath []string, cindent string) (string, error) {
	resFields := []string{}
	resSections := []string{""}
	fields := objectFields(v, withoutHidden)
	sort.Strings(fields)

	// iterate over non-section items
	for _, fieldName := range fields {
		fieldValue, err := v.index(i, fieldName)
		if err != nil {
			return "", err
		}

		isSection, err := tomlIsSection(i, fieldValue)
		if err != nil {
			return "", err
		}

		childIndexedPath := tomlAddToPath(indexedPath, fieldName)

		if isSection {
			// render as section and add to array of sections

			childPath := tomlAddToPath(path, fieldName)

			switch fv := fieldValue.(type) {
			case *valueObject:
				section, err := tomlRenderTable(i, fv, sindent, childPath, childIndexedPath, cindent)
				if err != nil {
					return "", err
				}
				resSections = append(resSections, section)
			case *valueArray:
				section, err := tomlRenderTableArray(i, fv, sindent, childPath, childIndexedPath, cindent)
				if err != nil {
					return "", err
				}
				resSections = append(resSections, section)
			default:
				return "", i.Error(fmt.Sprintf("invalid type for section: %v", reflect.TypeOf(fieldValue)))
			}
		} else {
			// render as value and append to result fields

			renderedValue, err := tomlRenderValue(i, fieldValue, sindent, childIndexedPath, false, "")
			if err != nil {
				return "", err
			}
			resFields = append(resFields, strings.Split(tomlEncodeKey(fieldName)+" = "+renderedValue, "\n")...)
		}
	}

	// create the result string
	res := ""

	if len(resFields) > 0 {
		res = "" + cindent
	}
	res = res + strings.Join(resFields, "\n"+cindent) + strings.Join(resSections, "\n\n")
	return res, nil
}

func builtinManifestTomlEx(i *interpreter, arguments []value) (value, error) {
	val := arguments[0]
	vindent, err := i.getString(arguments[1])
	if err != nil {
		return nil, err
	}
	sindent := vindent.getGoString()

	switch v := val.(type) {
	case *valueObject:
		res, err := tomlTableInternal(i, v, sindent, []string{}, []string{}, "")
		if err != nil {
			return nil, err
		}
		return makeValueString(res), nil
	default:
		return nil, i.Error(fmt.Sprintf("TOML body must be an object. Got %s", v.getType().name))
	}
}

// We have a very similar logic here /interpreter.go@v0.16.0#L695 and here: /interpreter.go@v0.16.0#L627
// These should ideally be unified
// For backwards compatibility reasons, we are manually marshalling to json so we can control formatting
// In the future, it might be apt to use a library [pretty-printing] function
func builtinManifestJSONEx(i *interpreter, arguments []value) (value, error) {
	val := arguments[0]

	vindent, err := i.getString(arguments[1])
	if err != nil {
		return nil, err
	}

	vnewline, err := i.getString(arguments[2])
	if err != nil {
		return nil, err
	}

	vkvSep, err := i.getString(arguments[3])
	if err != nil {
		return nil, err
	}

	sindent := vindent.getGoString()
	newline := vnewline.getGoString()
	kvSep := vkvSep.getGoString()

	var path []string

	var aux func(ov value, path []string, cindent string) (string, error)
	aux = func(ov value, path []string, cindent string) (string, error) {
		if ov == nil {
			fmt.Println("value is nil")
			return "null", nil
		}

		switch v := ov.(type) {
		case *valueNull:
			return "null", nil
		case valueString:
			jStr, err := jsonEncode(v.getGoString())
			if err != nil {
				return "", i.Error(fmt.Sprintf("failed to marshal valueString to JSON: %v", err.Error()))
			}
			return jStr, nil
		case *valueNumber:
			return strconv.FormatFloat(v.value, 'f', -1, 64), nil
		case *valueBoolean:
			return fmt.Sprintf("%t", v.value), nil
		case *valueFunction:
			return "", i.Error(fmt.Sprintf("tried to manifest function at %s", path))
		case *valueArray:
			newIndent := cindent + sindent
			lines := []string{"[" + newline}

			var arrayLines []string
			for aI, cThunk := range v.elements {
				cTv, err := cThunk.getValue(i)
				if err != nil {
					return "", err
				}

				newPath := append(path, strconv.FormatInt(int64(aI), 10))
				s, err := aux(cTv, newPath, newIndent)
				if err != nil {
					return "", err
				}
				arrayLines = append(arrayLines, newIndent+s)
			}
			lines = append(lines, strings.Join(arrayLines, ","+newline))
			lines = append(lines, newline+cindent+"]")
			return strings.Join(lines, ""), nil
		case *valueObject:
			newIndent := cindent + sindent
			lines := []string{"{" + newline}

			fields := objectFields(v, withoutHidden)
			sort.Strings(fields)
			var objectLines []string
			for _, fieldName := range fields {
				fieldValue, err := v.index(i, fieldName)
				if err != nil {
					return "", err
				}

				fieldNameMarshalled, err := jsonEncode(fieldName)
				if err != nil {
					return "", i.Error(fmt.Sprintf("failed to marshal object fieldname to JSON: %v", err.Error()))
				}

				newPath := append(path, fieldName)
				mvs, err := aux(fieldValue, newPath, newIndent)
				if err != nil {
					return "", err
				}

				line := newIndent + string(fieldNameMarshalled) + kvSep + mvs
				objectLines = append(objectLines, line)
			}
			lines = append(lines, strings.Join(objectLines, ","+newline))
			lines = append(lines, newline+cindent+"}")
			return strings.Join(lines, ""), nil
		default:
			return "", i.Error(fmt.Sprintf("unknown type to marshal to JSON: %s", reflect.TypeOf(v)))
		}
	}

	finalString, err := aux(val, path, "")
	if err != nil {
		return nil, err
	}

	return makeValueString(finalString), nil
}

func builtinExtVar(i *interpreter, name value) (value, error) {
	str, err := i.getString(name)
	if err != nil {
		return nil, err
	}
	index := str.getGoString()
	if pv, ok := i.extVars[index]; ok {
		return i.evaluatePV(pv)
	}
	return nil, i.Error("Undefined external variable: " + string(index))
}

func builtinNative(i *interpreter, name value) (value, error) {
	str, err := i.getString(name)
	if err != nil {
		return nil, err
	}
	index := str.getGoString()
	if f, exists := i.nativeFuncs[index]; exists {
		return &valueFunction{ec: f}, nil
	}
	return &valueNull{}, nil
}

func builtinSum(i *interpreter, arrv value) (value, error) {
	arr, err := i.getArray(arrv)
	if err != nil {
		return nil, err
	}
	sum := 0.0
	for _, elem := range arr.elements {
		elemValue, err := i.evaluateNumber(elem)
		if err != nil {
			return nil, err
		}
		sum += elemValue.value
	}
	return makeValueNumber(sum), nil
}

// Utils for builtins - TODO(sbarzowski) move to a separate file in another commit

type builtin interface {
	evalCallable
	Name() ast.Identifier
}

func flattenArgs(args callArguments, params []namedParameter, defaults []value) []*cachedThunk {
	positions := make(map[ast.Identifier]int, len(params))
	for i, param := range params {
		positions[param.name] = i
	}

	flatArgs := make([]*cachedThunk, len(params))

	// Bind positional arguments
	copy(flatArgs, args.positional)
	// Bind named arguments
	for _, arg := range args.named {
		flatArgs[positions[arg.name]] = arg.pv
	}
	// Bind defaults for unsatisfied named parameters
	for i := range params {
		if flatArgs[i] == nil {
			flatArgs[i] = readyThunk(defaults[i])
		}
	}
	return flatArgs
}

type unaryBuiltinFunc func(*interpreter, value) (value, error)

type unaryBuiltin struct {
	name     ast.Identifier
	function unaryBuiltinFunc
	params   ast.Identifiers
}

func (b *unaryBuiltin) evalCall(args callArguments, i *interpreter) (value, error) {
	flatArgs := flattenArgs(args, b.parameters(), []value{})

	x, err := flatArgs[0].getValue(i)
	if err != nil {
		return nil, err
	}
	return b.function(i, x)
}

func (b *unaryBuiltin) parameters() []namedParameter {
	ret := make([]namedParameter, len(b.params))
	for i := range ret {
		ret[i].name = b.params[i]
	}
	return ret
}

func (b *unaryBuiltin) Name() ast.Identifier {
	return b.name
}

type binaryBuiltinFunc func(*interpreter, value, value) (value, error)

type binaryBuiltin struct {
	name     ast.Identifier
	function binaryBuiltinFunc
	params   ast.Identifiers
}

func (b *binaryBuiltin) evalCall(args callArguments, i *interpreter) (value, error) {
	flatArgs := flattenArgs(args, b.parameters(), []value{})

	x, err := flatArgs[0].getValue(i)
	if err != nil {
		return nil, err
	}
	y, err := flatArgs[1].getValue(i)
	if err != nil {
		return nil, err
	}
	return b.function(i, x, y)
}

func (b *binaryBuiltin) parameters() []namedParameter {
	ret := make([]namedParameter, len(b.params))
	for i := range ret {
		ret[i].name = b.params[i]
	}
	return ret
}

func (b *binaryBuiltin) Name() ast.Identifier {
	return b.name
}

type ternaryBuiltinFunc func(*interpreter, value, value, value) (value, error)

type ternaryBuiltin struct {
	name     ast.Identifier
	function ternaryBuiltinFunc
	params   ast.Identifiers
}

func (b *ternaryBuiltin) evalCall(args callArguments, i *interpreter) (value, error) {
	flatArgs := flattenArgs(args, b.parameters(), []value{})

	x, err := flatArgs[0].getValue(i)
	if err != nil {
		return nil, err
	}
	y, err := flatArgs[1].getValue(i)
	if err != nil {
		return nil, err
	}
	z, err := flatArgs[2].getValue(i)
	if err != nil {
		return nil, err
	}
	return b.function(i, x, y, z)
}

func (b *ternaryBuiltin) parameters() []namedParameter {
	ret := make([]namedParameter, len(b.params))
	for i := range ret {
		ret[i].name = b.params[i]
	}
	return ret
}

func (b *ternaryBuiltin) Name() ast.Identifier {
	return b.name
}

type generalBuiltinFunc func(*interpreter, []value) (value, error)

type generalBuiltinParameter struct {
	// Note that the defaults are passed as values rather than AST nodes like in Parameters.
	// This spares us unnecessary evaluation.
	defaultValue value
	name         ast.Identifier
}

// generalBuiltin covers cases that other builtin structures do not,
// in particular it can have any number of parameters. It can also
// have optional parameters.  The optional ones have non-nil defaultValues
// at the same index.
type generalBuiltin struct {
	name     ast.Identifier
	function generalBuiltinFunc
	params   []generalBuiltinParameter
}

func (b *generalBuiltin) parameters() []namedParameter {
	ret := make([]namedParameter, len(b.params))
	for i := range ret {
		ret[i].name = b.params[i].name
		if b.params[i].defaultValue != nil {
			// This is not actually used because the defaultValue is used instead.
			// The only reason we don't leave it nil is because the checkArguments
			// function uses the non-nil status to indicate that the parameter
			// is optional.
			ret[i].defaultArg = &ast.LiteralNull{}
		}
	}
	return ret
}

func (b *generalBuiltin) defaultValues() []value {
	ret := make([]value, len(b.params))
	for i := range ret {
		ret[i] = b.params[i].defaultValue
	}
	return ret
}

func (b *generalBuiltin) Name() ast.Identifier {
	return b.name
}

func (b *generalBuiltin) evalCall(args callArguments, i *interpreter) (value, error) {
	flatArgs := flattenArgs(args, b.parameters(), b.defaultValues())
	values := make([]value, len(flatArgs))
	for j := 0; j < len(values); j++ {
		var err error
		values[j], err = flatArgs[j].getValue(i)
		if err != nil {
			return nil, err
		}
	}
	return b.function(i, values)
}

// End of builtin utils

var builtinID = &unaryBuiltin{name: "id", function: builtinIdentity, params: ast.Identifiers{"x"}}
var functionID = &valueFunction{ec: builtinID}

var bopBuiltins = []*binaryBuiltin{
	// Note that % and `in` are desugared instead of being handled here
	ast.BopMult: &binaryBuiltin{name: "operator*", function: builtinMult, params: ast.Identifiers{"x", "y"}},
	ast.BopDiv:  &binaryBuiltin{name: "operator/", function: builtinDiv, params: ast.Identifiers{"x", "y"}},

	ast.BopPlus:  &binaryBuiltin{name: "operator+", function: builtinPlus, params: ast.Identifiers{"x", "y"}},
	ast.BopMinus: &binaryBuiltin{name: "operator-", function: builtinMinus, params: ast.Identifiers{"x", "y"}},

	ast.BopShiftL: &binaryBuiltin{name: "operator<<", function: builtinShiftL, params: ast.Identifiers{"x", "y"}},
	ast.BopShiftR: &binaryBuiltin{name: "operator>>", function: builtinShiftR, params: ast.Identifiers{"x", "y"}},

	ast.BopGreater:   &binaryBuiltin{name: "operator>", function: builtinGreater, params: ast.Identifiers{"x", "y"}},
	ast.BopGreaterEq: &binaryBuiltin{name: "operator>=", function: builtinGreaterEq, params: ast.Identifiers{"x", "y"}},
	ast.BopLess:      &binaryBuiltin{name: "operator<,", function: builtinLess, params: ast.Identifiers{"x", "y"}},
	ast.BopLessEq:    &binaryBuiltin{name: "operator<=", function: builtinLessEq, params: ast.Identifiers{"x", "y"}},

	ast.BopManifestEqual:   &binaryBuiltin{name: "operator==", function: builtinEquals, params: ast.Identifiers{"x", "y"}},
	ast.BopManifestUnequal: &binaryBuiltin{name: "operator!=", function: builtinNotEquals, params: ast.Identifiers{"x", "y"}}, // Special case

	ast.BopBitwiseAnd: &binaryBuiltin{name: "operator&", function: builtinBitwiseAnd, params: ast.Identifiers{"x", "y"}},
	ast.BopBitwiseXor: &binaryBuiltin{name: "operator^", function: builtinBitwiseXor, params: ast.Identifiers{"x", "y"}},
	ast.BopBitwiseOr:  &binaryBuiltin{name: "operator|", function: builtinBitwiseOr, params: ast.Identifiers{"x", "y"}},
}

var uopBuiltins = []*unaryBuiltin{
	ast.UopNot:        &unaryBuiltin{name: "operator!", function: builtinNegation, params: ast.Identifiers{"x"}},
	ast.UopBitwiseNot: &unaryBuiltin{name: "operator~", function: builtinBitNeg, params: ast.Identifiers{"x"}},
	ast.UopPlus:       &unaryBuiltin{name: "operator+ (unary)", function: builtinUnaryPlus, params: ast.Identifiers{"x"}},
	ast.UopMinus:      &unaryBuiltin{name: "operator- (unary)", function: builtinUnaryMinus, params: ast.Identifiers{"x"}},
}

func buildBuiltinMap(builtins []builtin) map[string]evalCallable {
	result := make(map[string]evalCallable, len(builtins))
	for _, b := range builtins {
		result[string(b.Name())] = b
	}
	return result
}

func builtinParseInt(i *interpreter, x value) (value, error) {
	str, err := i.getString(x)
	if err != nil {
		return nil, err
	}
	res, err := strconv.ParseInt(str.getGoString(), 10, 64)
	if err != nil {
		return nil, i.Error(fmt.Sprintf("%s is not a base 10 integer", str.getGoString()))
	}
	return makeValueNumber(float64(res)), nil
}

var funcBuiltins = buildBuiltinMap([]builtin{
	builtinID,
	&unaryBuiltin{name: "extVar", function: builtinExtVar, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "length", function: builtinLength, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "toString", function: builtinToString, params: ast.Identifiers{"a"}},
	&binaryBuiltin{name: "trace", function: builtinTrace, params: ast.Identifiers{"str", "rest"}},
	&binaryBuiltin{name: "makeArray", function: builtinMakeArray, params: ast.Identifiers{"sz", "func"}},
	&binaryBuiltin{name: "flatMap", function: builtinFlatMap, params: ast.Identifiers{"func", "arr"}},
	&binaryBuiltin{name: "join", function: builtinJoin, params: ast.Identifiers{"sep", "arr"}},
	&unaryBuiltin{name: "reverse", function: builtinReverse, params: ast.Identifiers{"arr"}},
	&binaryBuiltin{name: "filter", function: builtinFilter, params: ast.Identifiers{"func", "arr"}},
	&ternaryBuiltin{name: "foldl", function: builtinFoldl, params: ast.Identifiers{"func", "arr", "init"}},
	&ternaryBuiltin{name: "foldr", function: builtinFoldr, params: ast.Identifiers{"func", "arr", "init"}},
	&binaryBuiltin{name: "member", function: builtinMember, params: ast.Identifiers{"arr", "x"}},
	&binaryBuiltin{name: "range", function: builtinRange, params: ast.Identifiers{"from", "to"}},
	&binaryBuiltin{name: "primitiveEquals", function: primitiveEquals, params: ast.Identifiers{"x", "y"}},
	&binaryBuiltin{name: "equals", function: builtinEquals, params: ast.Identifiers{"x", "y"}},
	&binaryBuiltin{name: "objectFieldsEx", function: builtinObjectFieldsEx, params: ast.Identifiers{"obj", "hidden"}},
	&ternaryBuiltin{name: "objectHasEx", function: builtinObjectHasEx, params: ast.Identifiers{"obj", "fname", "hidden"}},
	&unaryBuiltin{name: "type", function: builtinType, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "char", function: builtinChar, params: ast.Identifiers{"n"}},
	&unaryBuiltin{name: "codepoint", function: builtinCodepoint, params: ast.Identifiers{"str"}},
	&unaryBuiltin{name: "ceil", function: builtinCeil, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "floor", function: builtinFloor, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "sqrt", function: builtinSqrt, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "sin", function: builtinSin, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "cos", function: builtinCos, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "tan", function: builtinTan, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "asin", function: builtinAsin, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "acos", function: builtinAcos, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "atan", function: builtinAtan, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "log", function: builtinLog, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "exp", function: builtinExp, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "mantissa", function: builtinMantissa, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "exponent", function: builtinExponent, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "round", function: builtinRound, params: ast.Identifiers{"x"}},
	&binaryBuiltin{name: "pow", function: builtinPow, params: ast.Identifiers{"x", "n"}},
	&binaryBuiltin{name: "modulo", function: builtinModulo, params: ast.Identifiers{"x", "y"}},
	&unaryBuiltin{name: "md5", function: builtinMd5, params: ast.Identifiers{"s"}},
	&binaryBuiltin{name: "xnor", function: builtinXnor, params: ast.Identifiers{"x", "y"}},
	&binaryBuiltin{name: "xor", function: builtinXor, params: ast.Identifiers{"x", "y"}},
	&binaryBuiltin{name: "lstripChars", function: builtinLstripChars, params: ast.Identifiers{"str", "chars"}},
	&binaryBuiltin{name: "rstripChars", function: builtinRstripChars, params: ast.Identifiers{"str", "chars"}},
	&binaryBuiltin{name: "stripChars", function: builtinStripChars, params: ast.Identifiers{"str", "chars"}},
	&ternaryBuiltin{name: "substr", function: builtinSubstr, params: ast.Identifiers{"str", "from", "len"}},
	&ternaryBuiltin{name: "splitLimit", function: builtinSplitLimit, params: ast.Identifiers{"str", "c", "maxsplits"}},
	&ternaryBuiltin{name: "strReplace", function: builtinStrReplace, params: ast.Identifiers{"str", "from", "to"}},
	&unaryBuiltin{name: "isEmpty", function: builtinIsEmpty, params: ast.Identifiers{"str"}},
	&unaryBuiltin{name: "base64Decode", function: builtinBase64Decode, params: ast.Identifiers{"str"}},
	&unaryBuiltin{name: "base64DecodeBytes", function: builtinBase64DecodeBytes, params: ast.Identifiers{"str"}},
	&unaryBuiltin{name: "parseInt", function: builtinParseInt, params: ast.Identifiers{"str"}},
	&unaryBuiltin{name: "parseJson", function: builtinParseJSON, params: ast.Identifiers{"str"}},
	&unaryBuiltin{name: "parseYaml", function: builtinParseYAML, params: ast.Identifiers{"str"}},
	&generalBuiltin{name: "manifestJsonEx", function: builtinManifestJSONEx, params: []generalBuiltinParameter{{name: "value"}, {name: "indent"},
		{name: "newline", defaultValue: &valueFlatString{value: []rune("\n")}},
		{name: "key_val_sep", defaultValue: &valueFlatString{value: []rune(": ")}}}},
	&generalBuiltin{name: "manifestTomlEx", function: builtinManifestTomlEx, params: []generalBuiltinParameter{{name: "value"}, {name: "indent"}}},
	&unaryBuiltin{name: "base64", function: builtinBase64, params: ast.Identifiers{"input"}},
	&unaryBuiltin{name: "encodeUTF8", function: builtinEncodeUTF8, params: ast.Identifiers{"str"}},
	&unaryBuiltin{name: "decodeUTF8", function: builtinDecodeUTF8, params: ast.Identifiers{"arr"}},
	&generalBuiltin{name: "sort", function: builtinSort, params: []generalBuiltinParameter{{name: "arr"}, {name: "keyF", defaultValue: functionID}}},
	&unaryBuiltin{name: "native", function: builtinNative, params: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "sum", function: builtinSum, params: ast.Identifiers{"arr"}},

	// internal
	&unaryBuiltin{name: "$objectFlatMerge", function: builtinUglyObjectFlatMerge, params: ast.Identifiers{"x"}},
})
