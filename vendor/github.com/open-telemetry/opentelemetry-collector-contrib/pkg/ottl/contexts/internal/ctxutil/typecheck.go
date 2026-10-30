// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"

import (
	"errors"
	"fmt"
)

// ErrSetNil is returned by context setters for paths that cannot represent a nil
// value, such as scalar and struct-typed fields. Container paths represent nil as
// an empty value instead: pcommon.Value becomes an empty value, and maps and slices
// become empty.
var ErrSetNil = errors.New("setting a value to nil is not supported")

// ExpectType ensures val can be asserted to T, returning a descriptive error when it cannot.
func ExpectType[T any](val any) (T, error) {
	if val == nil {
		var zero T
		return zero, ErrSetNil
	}
	typed, ok := val.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("expects %T but got %T", zero, val)
	}
	return typed, nil
}
