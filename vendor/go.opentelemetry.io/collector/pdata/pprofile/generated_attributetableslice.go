// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pprofile

import (
	"go.opentelemetry.io/collector/pdata/internal"
	v1 "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

// AttributeTableSlice logically represents a slice of Attribute.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use NewAttributeTableSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type AttributeTableSlice struct {
	orig  *[]v1.KeyValue
	state *internal.State
}

func newAttributeTableSlice(orig *[]v1.KeyValue, state *internal.State) AttributeTableSlice {
	return AttributeTableSlice{orig: orig, state: state}
}

// NewAttributeTableSlice creates a AttributeTableSlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewAttributeTableSlice() AttributeTableSlice {
	orig := []v1.KeyValue(nil)
	state := internal.StateMutable
	return newAttributeTableSlice(&orig, &state)
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewAttributeTableSlice()".
func (es AttributeTableSlice) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//
//	for i := 0; i < es.Len(); i++ {
//	    e := es.At(i)
//	    ... // Do something with the element
//	}
func (es AttributeTableSlice) At(i int) Attribute {
	return newAttribute(&(*es.orig)[i], es.state)
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new AttributeTableSlice can be initialized:
//
//	es := NewAttributeTableSlice()
//	es.EnsureCapacity(4)
//	for i := 0; i < 4; i++ {
//	    e := es.AppendEmpty()
//	    // Here should set all the values for e.
//	}
func (es AttributeTableSlice) EnsureCapacity(newCap int) {
	es.state.AssertMutable()
	oldCap := cap(*es.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]v1.KeyValue, len(*es.orig), newCap)
	copy(newOrig, *es.orig)
	*es.orig = newOrig
}

// AppendEmpty will append to the end of the slice an empty Attribute.
// It returns the newly added Attribute.
func (es AttributeTableSlice) AppendEmpty() Attribute {
	es.state.AssertMutable()
	*es.orig = append(*es.orig, v1.KeyValue{})
	return es.At(es.Len() - 1)
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es AttributeTableSlice) MoveAndAppendTo(dest AttributeTableSlice) {
	es.state.AssertMutable()
	dest.state.AssertMutable()
	if *dest.orig == nil {
		// We can simply move the entire vector and avoid any allocations.
		*dest.orig = *es.orig
	} else {
		*dest.orig = append(*dest.orig, *es.orig...)
	}
	*es.orig = nil
}

// RemoveIf calls f sequentially for each element present in the slice.
// If f returns true, the element is removed from the slice.
func (es AttributeTableSlice) RemoveIf(f func(Attribute) bool) {
	es.state.AssertMutable()
	newLen := 0
	for i := 0; i < len(*es.orig); i++ {
		if f(es.At(i)) {
			continue
		}
		if newLen == i {
			// Nothing to move, element is at the right place.
			newLen++
			continue
		}
		(*es.orig)[newLen] = (*es.orig)[i]
		newLen++
	}
	*es.orig = (*es.orig)[:newLen]
}

// CopyTo copies all elements from the current slice overriding the destination.
func (es AttributeTableSlice) CopyTo(dest AttributeTableSlice) {
	dest.state.AssertMutable()
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]
	} else {
		(*dest.orig) = make([]v1.KeyValue, srcLen)
	}
	for i := range *es.orig {
		newAttribute(&(*es.orig)[i], es.state).CopyTo(newAttribute(&(*dest.orig)[i], dest.state))
	}
}