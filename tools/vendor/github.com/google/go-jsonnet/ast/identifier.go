/*
Copyright 2016 Google Inc. All rights reserved.

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

package ast

import (
	"sort"
)

// Identifier represents a variable / parameter / field name.
type Identifier string

// Identifiers represents an Identifier slice.
type Identifiers []Identifier

// IdentifierSet represents an Identifier set.
type IdentifierSet map[Identifier]struct{}

// NewIdentifierSet creates a new IdentifierSet.
func NewIdentifierSet(idents ...Identifier) IdentifierSet {
	set := make(IdentifierSet)
	for _, ident := range idents {
		set[ident] = struct{}{}
	}
	return set
}

// Add adds an Identifier to the set.
func (set IdentifierSet) Add(ident Identifier) bool {
	if _, ok := set[ident]; ok {
		return false
	}
	set[ident] = struct{}{}
	return true
}

// AddIdentifiers adds a slice of identifiers to the set.
func (set IdentifierSet) AddIdentifiers(idents Identifiers) {
	for _, ident := range idents {
		set.Add(ident)
	}
}

// Contains returns true if an Identifier is in the set.
func (set IdentifierSet) Contains(ident Identifier) bool {
	_, ok := set[ident]
	return ok
}

// Remove removes an Identifier from the set.
func (set IdentifierSet) Remove(ident Identifier) {
	delete(set, ident)
}

// ToSlice returns an Identifiers slice from the set.
func (set IdentifierSet) ToSlice() Identifiers {
	idents := make(Identifiers, len(set))
	i := 0
	for ident := range set {
		idents[i] = ident
		i++
	}
	return idents
}

// ToOrderedSlice returns the elements of the current set as an ordered slice.
func (set IdentifierSet) ToOrderedSlice() []Identifier {
	idents := set.ToSlice()
	sort.Sort(identifierSorter(idents))
	return idents
}

type identifierSorter []Identifier

func (s identifierSorter) Len() int           { return len(s) }
func (s identifierSorter) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s identifierSorter) Less(i, j int) bool { return s[i] < s[j] }

// Clone returns a clone of the set.
func (set IdentifierSet) Clone() IdentifierSet {
	newSet := make(IdentifierSet, len(set))
	for k, v := range set {
		newSet[k] = v
	}
	return newSet
}
