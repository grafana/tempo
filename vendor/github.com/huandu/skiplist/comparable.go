// Copyright 2011 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package skiplist

// Comparable defines a comparable func.
type Comparable interface {
	Compare(lhs, rhs interface{}) int
	CalcScore(key interface{}) float64
}

var (
	_ Comparable = GreaterThanFunc(nil)
	_ Comparable = LessThanFunc(nil)
)

// GreaterThanFunc returns true if lhs greater than rhs
type GreaterThanFunc func(lhs, rhs interface{}) int

// LessThanFunc returns true if lhs less than rhs
type LessThanFunc GreaterThanFunc

// Compare compares lhs and rhs using f.
func (f GreaterThanFunc) Compare(lhs, rhs interface{}) int {
	return f(lhs, rhs)
}

// CalcScore calculates score of a key through default CalcScore.
func (f GreaterThanFunc) CalcScore(key interface{}) float64 {
	return CalcScore(key)
}

// Compare compares lhs and rhs using f.
func (f LessThanFunc) Compare(lhs, rhs interface{}) int {
	return -f(lhs, rhs)
}

// CalcScore calculates score of a key through default CalcScore.
func (f LessThanFunc) CalcScore(key interface{}) float64 {
	return -CalcScore(key)
}

// Reverse creates a reversed comparable.
func Reverse(comparable Comparable) Comparable {
	return reversedComparable{
		comparable: comparable,
	}
}

type reversedComparable struct {
	comparable Comparable
}

var _ Comparable = reversedComparable{}

func (reversed reversedComparable) Compare(lhs, rhs interface{}) int {
	return -reversed.comparable.Compare(lhs, rhs)
}

func (reversed reversedComparable) CalcScore(key interface{}) float64 {
	return -reversed.comparable.CalcScore(key)
}
