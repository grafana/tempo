// Copyright 2022 Twilio Inc.

// Package parquet is a library for working with parquet files. For an overview
// of Parquet's qualities as a storage format, see this blog post:
// https://blog.twitter.com/engineering/en_us/a/2013/dremel-made-simple-with-parquet
//
// Or see the Parquet documentation: https://parquet.apache.org/docs/
package parquet

import "reflect"

func atLeastOne(size int) int {
	return atLeast(size, 1)
}

func atLeast(size, least int) int {
	if size < least {
		return least
	}
	return size
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func typeNameOf(t reflect.Type) string {
	s1 := t.String()
	s2 := t.Kind().String()
	if s1 == s2 {
		return s1
	}
	return s1 + " (" + s2 + ")"
}

func isZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}
