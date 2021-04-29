// Copyright 2011 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package skiplist

import "reflect"

// Scorable is used by skip list using customized key comparing function.
// For built-in functions, there is no need to care of this interface.
//
// Every skip list element with customized key must have a score value
// to indicate its sequence.
// For any two elements with key "k1" and "k2":
// - If Compare(k1, k2) is true, k1.Score() >= k2.Score() must be true.
// - If Compare(k1, k2) is false and k1 doesn't equal to k2, k1.Score() < k2.Score() must be true.
type Scorable interface {
	Score() float64
}

// CalcScore calculates score of a key.
//
// The score is a hint to optimize comparable performance.
// A skip list keeps all elements sorted by score from smaller to largest.
// If there are keys with different scores, these keys must be different.
func CalcScore(key interface{}) (score float64) {
	if scorable, ok := key.(Scorable); ok {
		score = scorable.Score()
		return
	}

	val := reflect.ValueOf(key)
	score = calcScore(val)
	return
}
