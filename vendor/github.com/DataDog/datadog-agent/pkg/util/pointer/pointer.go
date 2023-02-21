// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package pointer

// Ptr returns a pointer from a value. It will allocate a new heap object for it.
func Ptr[T any](v T) *T {
	return &v
}

// UIntPtrToFloatPtr converts a uint64 value to float64 and returns a pointer.
func UIntPtrToFloatPtr(u *uint64) *float64 {
	if u == nil {
		return nil
	}

	f := float64(*u)
	return &f
}
